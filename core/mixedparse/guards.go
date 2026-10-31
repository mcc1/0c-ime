package mixedparse

// These predicates project the pinned mixed.go bodies. Their arguments preserve
// source case; dictionary evidence and personal eligibility remain separate.
import (
	"github.com/0c-biggo/0c-ime/core/decode"
	"github.com/0c-biggo/0c-ime/core/input"
	"github.com/0c-biggo/0c-ime/core/zhuyin"
	"math"
	"strings"
	"unicode"
	"unicode/utf8"
)

func NumberLike(raw string) bool {
	digit := false
	for i, r := range raw {
		switch {
		case r >= '0' && r <= '9':
			digit = true
		case strings.ContainsRune(".,:/-", r) && digit:
			last := i == len(raw)-1
			if last && r != '.' && r != ',' || !last && (raw[i+1] < '0' || raw[i+1] > '9') {
				return false
			}
			digit = false
		default:
			return false
		}
	}
	return raw != ""
}
func (e Evaluator) englishWord(c Context, s string) bool {
	return (len(s) > 1 || strings.EqualFold(s, "a") || strings.EqualFold(s, "i")) && (e.English != nil && e.English.Has(s) || !c.Private && !c.Paused && !c.LocalBlocked && e.PersonalEnglish != nil && e.PersonalEnglish(s))
}
func (e Evaluator) literalPrefix(c Context, p, b, k string) bool {
	word := strings.TrimRightFunc(p, func(r rune) bool { return !unicode.IsLetter(r) })
	if word == p {
		return e.englishWord(c, p)
	}
	if k == " " || !strings.ContainsFunc(b, unicode.IsLetter) {
		return false
	}
	return word == "" && !strings.ContainsFunc(p, unicode.IsLetter) || e.englishWord(c, word)
}
func englishThenNumber(raw, key string) bool {
	word := strings.TrimRightFunc(raw, func(r rune) bool { return !unicode.IsLetter(r) })
	tail := raw[len(word):] + key
	return word != "" && tail != key && strings.ContainsAny(tail, ".,:") && !strings.ContainsAny(tail, "/") && NumberLike(tail) && !strings.ContainsFunc(word, func(r rune) bool { return r > unicode.MaxASCII })
}
func (e Evaluator) digitToneNumber(c Context, b string, t rune) bool {
	if b == "" || strings.ContainsFunc(b, unicode.IsLetter) || !strings.ContainsFunc(b, unicode.IsDigit) {
		return false
	}
	digits := strings.Trim(b, "0123456789") == ""
	if digits && (literalNumberBefore(c) != "" || !keysInSyllableOrder(b) && !afterChinese(c)) {
		return true
	}
	v, ok := composer(b)
	if !ok {
		return false
	}
	s, ok := v.Add(t)
	return ok && s.ToneOf() == 5 && !e.reachable(c, s)
}
func punctKeyWord(c Context, b string) bool {
	if b != "." && b != "," {
		return false
	}
	if len(c.Units) == 0 || !c.Units[len(c.Units)-1].Literal || c.Units[len(c.Units)-1].Text == "" {
		return true
	}
	r, _ := utf8.DecodeLastRuneInString(c.Units[len(c.Units)-1].Text)
	return r > unicode.MaxASCII || unicode.IsSpace(r)
}
func (e Evaluator) digitWord(c Context, b string, t rune) bool {
	if t != zhuyin.Tone1 || b == "" || strings.Trim(b, "0123456789") != "" && !punctKeyWord(c, b) || !keysInSyllableOrder(b) || literalNumberBefore(c) != "" {
		return false
	}
	s, ok := e.LegacyReading(c, b, t)
	return ok && s.ToneOf() == 1 && len(e.Lex.Exact([]zhuyin.Syllable{s})) > 0
}
func (e Evaluator) digitsReadAsChinese(c Context, b string, t rune) bool {
	if !strings.ContainsFunc(b, unicode.IsDigit) {
		return false
	}
	if _, ok := e.LegacyReading(c, b, t); !ok {
		return false
	}
	full := literalNumberBefore(c) + b
	return !NumberLike(full) || strings.HasSuffix(full, ".")
}
func (e Evaluator) isolated(s zhuyin.Syllable, system bool) float64 {
	score := e.Score
	if system && e.System != nil {
		score = e.System
	}
	d := decode.Decoder{Lex: e.Lex, Score: score}
	_, v := d.Best(input.Exact{}.Expand([]zhuyin.Syllable{s}), nil, true)
	return math.Exp(v)
}
func (e Evaluator) Guards(c Context, raw, p, b, k string, t rune) [13]bool {
	c = Project(c, p)
	rare := false
	if t != zhuyin.Tone1 && b != "" && !strings.ContainsFunc(b, unicode.IsLetter) {
		if s, ok := e.LegacyReading(c, b, t); ok && e.isolated(s, false) < 1.5e-6 && !e.continues(c, s) {
			rare = e.isolated(s, true) < 1.5e-6
		}
	}
	litDigit := false
	if len(c.Units) > 0 {
		u := c.Units[len(c.Units)-1]
		litDigit = u.Literal && len(u.Text) == 1 && u.Text[0] >= '0' && u.Text[0] <= '9'
	}
	word := strings.TrimRightFunc(b, func(r rune) bool { return !unicode.IsLetter(r) && !unicode.IsDigit(r) })
	digitsChinese := e.digitsReadAsChinese(c, b, t)
	return [13]bool{NumberLike(raw), NumberLike(p), englishThenNumber(b, k), e.digitToneNumber(c, b, t), literalNumberBefore(c) != "" && !strings.ContainsFunc(b, unicode.IsLetter) && NumberLike(literalNumberBefore(c)+b+k), rare, litDigit, word != b && word != "" && e.englishWord(c, word), e.literalPrefix(c, p, b, k), e.digitWord(c, b, t), punctKeyWord(c, b), digitsChinese, !strings.ContainsFunc(b, unicode.IsLetter) && !afterChinese(c) && !digitsChinese && !e.digitWord(c, b, t)}
}
