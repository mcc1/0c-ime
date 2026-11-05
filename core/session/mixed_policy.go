package session

import (
	"strings"
	"unicode"

	"github.com/0c-biggo/0c-ime/core/zhuyin"
)

// Aggressive mixed input is opt-in. Conservative account and number rules remain
// in mixedBoundary; this path lets completed Chinese readings compete without
// an absolute frequency gate. Both modes preserve original-key candidates.
func (s *Session) mixedExactReading(raw string, tone rune) (zhuyin.Syllable, bool) {
	if !s.keysInSyllableOrder(raw) {
		return "", false
	}
	c, ok := s.mixedComposer(raw)
	if !ok {
		return "", false
	}
	done, ok := c.Add(tone)
	return done, ok && s.readingReachable(done)
}

func (s *Session) mixedAggressiveBoundary(raw, key string, tone rune) (int, zhuyin.Syllable, bool) {
	// The caller only uses this path for explicit tone keys, never spaces.
	best, bestScore, bestSyl := 0, 0.0, zhuyin.Syllable("")
	for at := len(raw) - 1; at >= 1; at-- {
		// 後段的按鍵要照聲母、介音、韻母的順序（#176：hello13 的 o1 是 ㄟㄅ，不是 hell北）。真的在打注音時照順序打，
		// 把英文字尾借去組字時常顛倒；跟 #175 英文後面的反序數字留數字一樣。
		if !s.mixedLiteralPrefix(raw[:at], raw[at:], key) || !s.keysInSyllableOrder(raw[at:]) {
			continue
		}
		done, ok := s.mixedExactReading(raw[at:], tone)
		if !ok {
			continue
		}
		score := s.mixedSyllableScore(done)
		ratio := float64(mixedMoveRatio)
		if best > 0 && raw[at:best] == "s" && s.mixedEnglishWord(raw[:at]) {
			ratio = mixedPluralRatio
		}
		if best == 0 || score > 0 && score >= ratio*bestScore {
			best, bestScore, bestSyl = at, score, done
		}
	}
	if best > 0 {
		return s.mixedAbsorbNonLetters(raw, best, bestSyl, tone)
	}
	if at, done, ok := s.mixedAfterTypo(raw, tone); ok {
		return at, done, true
	}
	// 字典不認得的前段：大寫字母不是注音鍵，先照上游 #241 從大寫後面切（Vxk7 是 V了、Gq06 是 G盤，不是 Vx呃、Gq玵）；
	// 沒有大寫邊界時才在下面比較各種切法。
	if at, done, ok := s.mixedAfterToken(raw, key, tone); ok {
		return s.mixedAbsorbNonLetters(raw, at, done, tone)
	}
	if at, done, ok := s.mixedAfterLetter(raw, key, tone); ok {
		return s.mixedAbsorbNonLetters(raw, at, done, tone)
	}
	for at := len(raw) - 1; at >= 2; at-- {
		prefix, rest := raw[:at], raw[at:]
		if !strings.ContainsFunc(prefix, unicode.IsLetter) ||
			strings.ContainsFunc(prefix, func(r rune) bool { return r > unicode.MaxASCII || !unicode.IsLetter(r) && !unicode.IsDigit(r) }) ||
			strings.ContainsFunc(prefix, unicode.IsDigit) && !strings.ContainsFunc(rest, unicode.IsLetter) ||
			!s.keysInSyllableOrder(raw[at:]) {
			continue
		}
		done, ok := s.mixedExactReading(raw[at:], tone)
		if !ok {
			continue
		}
		if score := s.mixedSyllableScore(done); best == 0 || score >= mixedUnknownRatio*bestScore {
			best, bestScore, bestSyl = at, score, done
		}
	}
	if best == 0 {
		return 0, "", false
	}
	return s.mixedAbsorbNonLetters(raw, best, bestSyl, tone)
}

// mixedAbsorbNonLetters：切點決定後，前一鍵是數字或標點鍵、併進後段仍組得成字時就併進去，一直併到組不成或碰到字母。
// 數字與標點幾乎不會是英文字的一部分，所以中文優先（WQ5j04 是 WQ賺，不是 WQ5萬；學過的 wq5 接 j04 也是 wq賺）。
// 字母鍵不併：英文字常以聲母鍵結尾（iPhone 的 e 是 ㄍ、mac 的 c 是 ㄏ），併進去會變成 iPhon果、ma火。
func (s *Session) mixedAbsorbNonLetters(raw string, at int, syl zhuyin.Syllable, tone rune) (int, zhuyin.Syllable, bool) {
	for at >= 2 && !unicode.IsLetter(rune(raw[at-1])) {
		done, ok := s.mixedExactReading(raw[at-1:], tone)
		if !ok {
			break
		}
		at, syl = at-1, done
	}
	return at, syl, true
}
