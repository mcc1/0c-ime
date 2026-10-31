package mixedparse

import (
	"github.com/0c-biggo/0c-ime/core/decode"
	"github.com/0c-biggo/0c-ime/core/layout"
	"github.com/0c-biggo/0c-ime/core/zhuyin"
	"strings"
)

func SpellExact(body string, tone rune) (zhuyin.Syllable, bool) {
	if len(body) < 1 || len(body) > 3 {
		return "", false
	}
	c, ok := composer(body)
	if !ok || !keysInSyllableOrder(body) {
		return "", false
	}
	s, ok := c.Add(tone)
	return s, ok && s.Valid() && !s.Untoned()
}
func composer(body string) (zhuyin.Composer, bool) {
	var c zhuyin.Composer
	seen := map[zhuyin.Class]bool{}
	for _, k := range body {
		sym, ok := layout.Standard.Symbol(k)
		cl := zhuyin.ClassOf(sym)
		if !ok || cl == zhuyin.None || cl == zhuyin.Tone || seen[cl] {
			return zhuyin.Composer{}, false
		}
		seen[cl] = true
		c.Add(sym)
	}
	return c, body != ""
}
func keysInSyllableOrder(keys string) bool {
	last := zhuyin.None
	for _, k := range keys {
		sym, _ := layout.Standard.Symbol(k)
		c := zhuyin.ClassOf(sym)
		if c <= last {
			return false
		}
		last = c
	}
	return true
}
func (e Evaluator) reachable(c Context, s zhuyin.Syllable) bool {
	r := []zhuyin.Syllable{s}
	return len(e.Lex.Exact(r)) > 0 || e.Lex.HasPrefix(r) || e.continues(c, s)
}
func (e Evaluator) continues(c Context, s zhuyin.Syllable) bool {
	if c.RetainedPrefix != nil {
		r := []zhuyin.Syllable{zhuyin.Syllable(c.RetainedPrefix.Reading), s}
		if len(e.Lex.Exact(r)) > 0 || e.Lex.HasPrefix(r) {
			return true
		}
	}
	for at := len(c.Units) - 1; at >= 0 && len(c.Units)-at < decode.MaxWordLen; at-- {
		if c.Units[at].Literal {
			break
		}
		r := []zhuyin.Syllable{}
		for _, u := range c.Units[at:] {
			r = append(r, zhuyin.Syllable(u.Reading))
		}
		r = append(r, s)
		if len(e.Lex.Exact(r)) > 0 || e.Lex.HasPrefix(r) {
			return true
		}
	}
	return false
}
func (e Evaluator) LegacyReading(c Context, body string, tone rune) (zhuyin.Syllable, bool) {
	v, ok := composer(body)
	if !ok {
		return "", false
	}
	s, ok := v.Add(tone)
	if !ok {
		return "", false
	}
	if s.ToneOf() == 5 && !e.reachable(c, s) && e.reachable(c, s.WithTone(1)) {
		s = s.WithTone(1)
	}
	return s, e.reachable(c, s)
}
func afterChinese(c Context) bool {
	return c.RetainedPrefix != nil || len(c.Units) > 0 && !c.Units[len(c.Units)-1].Literal
}
func literalNumberBefore(c Context) string {
	at := len(c.Units)
	for at > 0 {
		u := c.Units[at-1]
		if !u.Literal || len(u.Text) != 1 || !strings.Contains("0123456789.,:/-", u.Text) {
			break
		}
		at--
	}
	s := ""
	for _, u := range c.Units[at:] {
		s += u.Text
	}
	return s
}
