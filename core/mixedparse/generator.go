package mixedparse

import (
	"fmt"
	"github.com/0c-biggo/0c-ime/core/decode"
	"github.com/0c-biggo/0c-ime/core/input"
	"github.com/0c-biggo/0c-ime/core/layout"
	"github.com/0c-biggo/0c-ime/core/zhuyin"
	"math"
	"sort"
	"strings"
)

func Tone(k byte) bool          { return strings.ContainsRune("3467", rune(k)) }
func floatp(v float64) *float64 { return &v }
func literal(raw string) Candidate {
	c := Candidate{Lane: "literal", Segments: []Segment{{Kind: "literal", Begin: 0, End: len(raw), Text: raw}}, ScoreStatus: "not_scored", FeatureVersion: FeatureVersion, History: "legacy-lagged-v1", Text: raw}
	for i := range c.Features {
		c.Features[i] = floatp(0)
	}
	c.Features[0] = floatp(1)
	c.Features[8] = floatp(1)
	c.Identity = Identity(c.Segments)
	c.ID = CandidateID(c.Segments)
	return c
}
func parse(raw string, a int) ([]Segment, []zhuyin.Syllable, bool) {
	segs := []Segment{}
	r := []zhuyin.Syllable{}
	if a > 0 {
		segs = append(segs, Segment{Kind: "literal", End: a, Text: raw[:a]})
	}
	at := a
	for at < len(raw) {
		end := at
		for end < len(raw) && !Tone(raw[end]) {
			end++
		}
		if end == len(raw) || len(r) == 2 {
			return nil, nil, false
		}
		t, _ := layout.Standard.Symbol(rune(raw[end]))
		s, ok := SpellExact(raw[at:end], t)
		if !ok {
			return nil, nil, false
		}
		segs = append(segs, Segment{Kind: "chinese", Begin: at, End: end + 1, Reading: string(s)})
		r = append(r, s)
		at = end + 1
	}
	return segs, r, len(r) > 0
}
func (e Evaluator) support(r []zhuyin.Syllable) Support {
	s := Support{WholeWord: len(e.Lex.Exact(r)) > 0}
	// Dynamic coverage needs one canonical witness; homophones are decoded by
	// the existing decoder and never form different boundary candidates.
	reach := make([]bool, len(r)+1)
	w := make([][]string, len(r)+1)
	reach[0] = true
	for at := 0; at < len(r); at++ {
		if !reach[at] {
			continue
		}
		for end := at + 1; end <= len(r); end++ {
			entries := e.Lex.Exact(r[at:end])
			if len(entries) > 0 && !reach[end] {
				reach[end] = true
				w[end] = append(append([]string(nil), w[at]...), entries[0].Surface)
			}
		}
	}
	s.CompleteCover = reach[len(r)]
	if s.CompleteCover {
		s.Witnesses = [][]string{w[len(r)]}
	} else {
		for at := 0; at < len(r); at++ {
			if reach[at] && e.Lex.HasPrefix(r[at:]) {
				s.PrefixOnly = true
				break
			}
		}
	}
	s.ContextOnly = !s.WholeWord && !s.CompleteCover && !s.PrefixOnly
	return s
}

// Generate evaluates only immutable values. Nil previous at a nonzero tone end
// means ownership is unavailable, rather than a fabricated stable reference.
func (e Evaluator) Generate(ctx Context, raw string, lastToneEnd int, previous []Segment) (Result, error) {
	out := Result{}
	if ctx.Protected() {
		return out, nil
	}
	if raw == "" || len(raw) > 64 {
		return out, fmt.Errorf("raw length outside 1..64")
	}
	for _, b := range []byte(raw) {
		if b < 33 || b > 126 || b == '`' {
			return out, fmt.Errorf("non-fixture byte")
		}
	}
	if !Tone(raw[len(raw)-1]) {
		return out, fmt.Errorf("search requires explicit tone")
	}
	out.GenerateCalls = 1
	for a := 0; a < len(raw)-1; a++ {
		segs, r, ok := parse(raw, a)
		if !ok {
			continue
		}
		cctx := Project(ctx, raw[:a])
		valid := true
		for _, s := range r {
			if !e.reachable(cctx, s) {
				valid = false
				break
			}
			cctx.Units = append(cctx.Units, Unit{Reading: string(s)})
		}
		if !valid {
			continue
		}
		c := Candidate{Lane: "strict", Segments: segs, Support: e.support(r), FeatureVersion: FeatureVersion, History: "legacy-lagged-v1"}
		c.ID = CandidateID(segs)
		c.Identity = Identity(segs)
		for _, s := range r {
			c.Readings = append(c.Readings, string(s))
		}
		d := decode.Decoder{Lex: e.Lex, Score: e.Score, Left: Project(ctx, raw[:a]).LastWord, Beam: 0, TopM: 64, TopB: 4, Continue: .5, SeparateContinuation: true, SuggestPrior: 5}
		path, v := d.Best(input.Exact{}.Expand(r), nil, false)
		out.ScoreCalls++
		c.ScoreStatus = "finite"
		switch {
		case math.IsNaN(v):
			c.ScoreStatus = "nan"
		case math.IsInf(v, 1):
			c.ScoreStatus = "positive_inf"
		case math.IsInf(v, -1):
			c.ScoreStatus = "unreachable"
		default:
			c.RawScore = floatp(v)
		}
		c.Text = raw[:a]
		if c.RawScore != nil {
			for _, s := range path {
				c.Text += s.Word
				c.LastWord = s.Word
			}
		} else {
			for _, s := range r {
				c.Text += s.Text()
			}
		}
		for i := range c.Features {
			c.Features[i] = floatp(0)
		}
		set := func(i int, b bool) {
			if b {
				c.Features[i] = floatp(1)
			}
		}
		c.Features[1] = floatp(float64(len(raw)-a) / float64(len(raw)))
		c.Features[2] = floatp(float64(len(r)) / 2)
		set(3, c.Support.WholeWord)
		set(4, c.Support.CompleteCover)
		set(5, c.Support.PrefixOnly)
		set(6, c.Support.ContextOnly)
		if c.RawScore != nil {
			c.Features[7] = floatp(v / float64(len(r)))
		} else {
			set(8, true)
		}
		p := raw[:a]
		set(9, p != "" && e.English != nil && e.English.Has(p))
		for _, x := range ctx.CustomTokens {
			set(10, *c.Features[10] == 1 || x == p && p != "")
		}
		set(11, p != "" && !ctx.Private && !ctx.Paused && !ctx.LocalBlocked && e.PersonalEnglish != nil && e.PersonalEnglish(p))
		if lastToneEnd > 0 {
			if previous == nil {
				c.Features[12] = nil
			} else {
				set(12, ProjectedIdentity(previous, lastToneEnd) != ProjectedIdentity(segs, lastToneEnd))
			}
		}
		first := segs[len(segs)-len(r)]
		t, _ := layout.Standard.Symbol(rune(raw[first.End-1]))
		g := e.Guards(ctx, raw, p, raw[first.Begin:first.End-1], raw[first.End-1:first.End], t)
		for i, b := range g {
			set(13+i, b)
		}
		out.Candidates = append(out.Candidates, c)
	}
	sort.Slice(out.Candidates, func(i, j int) bool {
		a, b := out.Candidates[i], out.Candidates[j]
		if a.Support.CompleteCover != b.Support.CompleteCover {
			return a.Support.CompleteCover
		}
		ac, bc := ChineseBytes(a), ChineseBytes(b)
		if ac != bc {
			return ac > bc
		}
		aa, bb := FirstChinese(a.Segments), FirstChinese(b.Segments)
		if aa != bb {
			return aa < bb
		}
		ar, br := strings.Join(a.Readings, "\x00"), strings.Join(b.Readings, "\x00")
		if ar != br {
			return ar < br
		}
		return a.Identity < b.Identity
	})
	for i := range out.Candidates {
		out.Candidates[i].Pruned = i >= 8
	}
	out.Candidates = append(out.Candidates, literal(raw))
	return out, nil
}
func FirstChinese(s []Segment) int {
	for _, x := range s {
		if x.Kind == "chinese" {
			return x.Begin
		}
	}
	return math.MaxInt
}
func ChineseBytes(c Candidate) int {
	n := 0
	for _, s := range c.Segments {
		if s.Kind == "chinese" {
			n += s.End - s.Begin
		}
	}
	return n
}
func ProjectedIdentity(s []Segment, end int) string {
	p := []Segment{}
	for _, x := range s {
		if x.Begin >= end {
			break
		}
		if x.End > end {
			x.End = end
			if x.Kind == "literal" {
				x.Text = ""
			}
		}
		p = append(p, x)
	}
	return Identity(p)
}
func Longest(c []Candidate) Candidate {
	var best Candidate
	for _, x := range c {
		if x.Lane == "literal" && best.ID == "" {
			best = x
		}
		if x.Lane != "strict" || x.Pruned {
			continue
		}
		if best.Lane != "strict" || ChineseBytes(x) > ChineseBytes(best) || ChineseBytes(x) == ChineseBytes(best) && (x.Support.CompleteCover && !best.Support.CompleteCover || x.Support.CompleteCover == best.Support.CompleteCover && x.Identity < best.Identity) {
			best = x
		}
	}
	return best
}

// Literal is the protected raw interpretation, including incomplete keys.
func Literal(raw string) Candidate { return literal(raw) }
