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
		return best, bestSyl, true
	}
	if at, done, ok := s.mixedAfterTypo(raw, tone); ok {
		return at, done, true
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
	return best, bestSyl, best > 0
}
