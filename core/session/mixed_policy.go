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
		// 把字面前段最後一個數字聲母鍵讓給後段（學過的 wq5 接 j04 → wq＋5j04 賺）時，較短的前段不必自己是英文詞：
		// 積極模式中文優先，要整段照字面請先按 Shift 切英文。
		extend := best == at+1 && s.mixedDigitInitialExtension(raw, at, best)
		if !extend && !s.mixedLiteralPrefix(raw[:at], raw[at:], key) || !s.keysInSyllableOrder(raw[at:]) {
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
		if best == 0 || s.mixedDigitInitialExtension(raw, at, best) || score > 0 && score >= ratio*bestScore {
			best, bestScore, bestSyl = at, score, done
		}
	}
	if best > 0 {
		return best, bestSyl, true
	}
	if at, done, ok := s.mixedAfterTypo(raw, tone); ok {
		return at, done, true
	}
	// 字典不認得的前段：大寫字母不是注音鍵，先照上游 #241 從大寫後面切（Vxk7 是 V了、Gq06 是 G盤，不是 Vx呃、Gq玵）；
	// 沒有大寫邊界時才在下面比較各種切法。
	// 大寫邊界的切點前一鍵是數字聲母時，一樣讓給後段（WD55j04 是 WD5賺，不是 WD55萬）。
	if at, done, ok := s.mixedAfterToken(raw, key, tone); ok {
		return s.mixedDigitInitialYield(raw, at, done, tone)
	}
	if at, done, ok := s.mixedAfterLetter(raw, key, tone); ok {
		return s.mixedDigitInitialYield(raw, at, done, tone)
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
		if score := s.mixedSyllableScore(done); best == 0 || s.mixedDigitInitialExtension(raw, at, best) || score >= mixedUnknownRatio*bestScore {
			best, bestScore, bestSyl = at, score, done
		}
	}
	return best, bestSyl, best > 0
}

// In aggressive mode, keep a digit-key initial with a completed syllable rather
// than attaching that key to English solely because the shorter reading is
// more frequent: WQ5j04 is WQ + ㄓㄨㄢˋ, not WQ5 + ㄨㄢˋ. Both proposed
// readings must already have passed exact spelling and dictionary checks.
func (s *Session) mixedDigitInitialExtension(raw string, at, best int) bool {
	return best == at+1 && unicode.IsDigit(rune(raw[at])) && s.keyIsInitial(rune(raw[at]))
}

// mixedDigitInitialYield moves a cut one key left when that key is a digit-key
// initial and the longer rest still spells a reachable reading in key order.
func (s *Session) mixedDigitInitialYield(raw string, at int, done zhuyin.Syllable, tone rune) (int, zhuyin.Syllable, bool) {
	if at >= 2 && s.mixedDigitInitialExtension(raw, at-1, at) && s.keysInSyllableOrder(raw[at-1:]) {
		if longer, ok := s.mixedExactReading(raw[at-1:], tone); ok {
			return at - 1, longer, true
		}
	}
	return at, done, true
}
