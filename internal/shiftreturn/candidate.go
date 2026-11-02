//go:build shiftreturneval

package shiftreturn

import (
	"fmt"
	"math"
	"sort"
	"strings"

	"github.com/0c-biggo/0c-ime/core/decode"
	"github.com/0c-biggo/0c-ime/core/input"
	"github.com/0c-biggo/0c-ime/core/layout"
	"github.com/0c-biggo/0c-ime/core/lexicon"
	"github.com/0c-biggo/0c-ime/core/mixedparse"
	"github.com/0c-biggo/0c-ime/core/zhuyin"
)

const (
	MaxRawBytes    = 512
	MaxKeptChinese = 256
	MaxLexQueries  = 250000
	MaxBestCalls   = 100000
)

type EnglishLookup interface{ Has(string) bool }
type Probe struct {
	Lex                                       lexicon.Source
	Score                                     decode.Scorer
	English                                   EnglishLookup
	GenerateCalls, ProbeLexQueries, BestCalls int
	bSnapshots                                map[string]Vector
	exhausted                                 string
}
type Result struct {
	Candidates                                []Hypothesis
	GenerationPerformed                       bool
	UncappedIDs                               []string
	KeptIDs                                   []string
	UncappedCount                             *int
	KeptCount                                 int
	LiteralID                                 string
	CapPruned                                 *bool
	RawOverflow                               bool
	Status, Reason, LimitHit                  string
	GenerateCalls, ProbeLexQueries, BestCalls int
}
type Display struct {
	Text     string
	Segments []Segment
}

func ptr[T any](v T) *T { return &v }
func identity(segs []Segment) string {
	a := []any{}
	for _, s := range segs {
		a = append(a, []any{s.Kind, s.Begin, s.End, s.Reading})
	}
	return string(Canonical(a))
}
func literal(q Query) Hypothesis {
	segs := []Segment{}
	if len(q.Raw) > 0 {
		segs = append(segs, Segment{"literal", 0, len(q.Raw), ""})
	}
	id := identity(segs)
	return Hypothesis{Version: Version, Kind: "candidate", Demo: q.Demo, ID: "h:" + HashBytes([]byte(id)), Identity: id, DisplayIdentity: id, Cut: len(q.Raw), Readings: []string{}, ParsedEnd: len(q.Raw), TailStart: len(q.Raw), Segments: segs, DisplaySegments: segs, Display: q.Raw, Features: []float64{1, 0, 0, 0, 0, 0, 1, 0, 0, 0, 0, 0, 1}, Kept: true}
}
func (p *Probe) exact(r []zhuyin.Syllable) ([]lexicon.Entry, error) {
	if p.ProbeLexQueries >= MaxLexQueries {
		p.exhausted = "probe_lex_queries"
		return nil, fmt.Errorf("lex query limit")
	}
	p.ProbeLexQueries++
	return p.Lex.Exact(r), nil
}
func (p *Probe) prefix(r []zhuyin.Syllable) (bool, error) {
	if p.ProbeLexQueries >= MaxLexQueries {
		p.exhausted = "probe_lex_queries"
		return false, fmt.Errorf("lex query limit")
	}
	p.ProbeLexQueries++
	return p.Lex.HasPrefix(r), nil
}
func (p *Probe) best(r []zhuyin.Syllable, prefix bool) (string, float64, error) {
	if p.BestCalls >= MaxBestCalls {
		p.exhausted = "best_calls"
		return "", 0, fmt.Errorf("decoder limit")
	}
	p.BestCalls++
	d := decode.Decoder{Lex: p.Lex, Score: p.Score}
	segs, score := d.Best((input.Exact{}).Expand(r), nil, prefix)
	if math.IsNaN(score) || math.IsInf(score, 1) {
		return "", 0, fmt.Errorf("nonfinite decoder score")
	}
	var text strings.Builder
	for _, s := range segs {
		text.WriteString(s.Word)
	}
	return text.String(), score, nil
}
func (p *Probe) Generate(q Query) (result Result, err error) {
	lit := literal(q)
	result = Result{Candidates: []Hypothesis{lit}, KeptIDs: []string{}, LiteralID: lit.ID, Status: "ok", LimitHit: "none"}
	defer func() {
		result.GenerateCalls = p.GenerateCalls
		result.ProbeLexQueries = p.ProbeLexQueries
		result.BestCalls = p.BestCalls
	}()
	if !q.Eligible {
		result.UncappedIDs = []string{}
		result.UncappedCount = ptr(0)
		result.CapPruned = ptr(false)
		return
	}
	if len(q.Raw) > MaxRawBytes {
		result.Status = "capacity_failure"
		result.Reason = "raw exceeds 512 bytes"
		result.RawOverflow = true
		result.LimitHit = "raw_bytes"
		return
	}
	for _, b := range []byte(q.Raw) {
		if b < 32 || b > 126 {
			return result, fmt.Errorf("non-ASCII raw")
		}
	}
	if len(q.ContextUnits) != 0 {
		return result, fmt.Errorf("v2 context_units must be empty")
	}
	if p.Lex == nil || p.Score == nil {
		return result, fmt.Errorf("missing frozen source/scorer")
	}
	if p.exhausted != "" {
		result.Status = "resource_exhausted"
		result.Reason = "episode budget exhausted"
		result.LimitHit = p.exhausted
		return
	}
	p.GenerateCalls++
	result.GenerationPerformed = true
	fail := func(e error) (Result, error) {
		if p.exhausted != "" {
			result.Status = "resource_exhausted"
			result.Reason = e.Error()
			result.LimitHit = p.exhausted
			result.UncappedIDs = nil
			result.UncappedCount = nil
			result.CapPruned = nil
			result.Candidates = []Hypothesis{lit}
			result.KeptIDs = []string{}
			result.KeptCount = 0
			return result, nil
		}
		return result, e
	}
	minCut := 0
	for _, token := range q.CustomTokens {
		if token != "" && strings.HasPrefix(token, q.Raw) && len(q.Raw) < len(token) {
			result.UncappedIDs = []string{}
			result.UncappedCount = ptr(0)
			result.CapPruned = ptr(false)
			return
		}
		if token != "" && strings.HasPrefix(q.Raw, token) {
			minCut = max(minCut, len(token))
		}
	}
	candidates := []Hypothesis{}
	for cut := minCut; cut < len(q.Raw); cut++ {
		h := Hypothesis{Version: Version, Kind: "candidate", Demo: q.Demo, Cut: cut, Readings: []string{}, Segments: []Segment{}}
		if cut > 0 {
			h.Segments = append(h.Segments, Segment{"literal", 0, cut, ""})
		}
		begin := cut
		for at := cut; at < len(q.Raw); at++ {
			if !strings.ContainsRune(" 3467", rune(q.Raw[at])) {
				continue
			}
			tone, _ := layout.Standard.Symbol(rune(q.Raw[at]))
			s, ok := mixedparse.SpellExact(q.Raw[begin:at], tone)
			if !ok {
				break
			}
			h.Readings = append(h.Readings, string(s))
			h.Segments = append(h.Segments, Segment{"chinese", begin, at + 1, string(s)})
			begin = at + 1
		}
		if len(h.Readings) == 0 {
			continue
		}
		h.FirstReading = h.Readings[0]
		h.ParsedEnd = begin
		rs := []zhuyin.Syllable{zhuyin.Syllable(h.FirstReading)}
		exact, e := p.exact(rs)
		if e != nil {
			return fail(e)
		}
		supported := len(exact) > 0
		if !supported {
			supported, e = p.prefix(rs)
			if e != nil {
				return fail(e)
			}
		}
		if !supported {
			continue
		}
		if begin < len(q.Raw) {
			h.Segments = append(h.Segments, Segment{"pending", begin, len(q.Raw), ""})
		}
		h.Identity = identity(h.Segments)
		h.ID = "h:" + HashBytes([]byte(h.Identity))
		candidates = append(candidates, h)
	}
	sort.Slice(candidates, func(i, j int) bool {
		if candidates[i].Cut != candidates[j].Cut {
			return candidates[i].Cut < candidates[j].Cut
		}
		return candidates[i].Identity < candidates[j].Identity
	})
	result.UncappedIDs = []string{}
	for _, h := range candidates {
		result.UncappedIDs = append(result.UncappedIDs, h.ID)
	}
	result.UncappedCount = ptr(len(candidates))
	result.CapPruned = ptr(len(candidates) > MaxKeptChinese)
	if len(candidates) > MaxKeptChinese {
		candidates = candidates[:MaxKeptChinese]
	}
	for _, h := range candidates {
		v, k, text, e := p.compute(q, h)
		if e != nil {
			return fail(e)
		}
		if k > 0 && text == "" {
			result.Status = "unsupported"
			result.Reason = "exact lexical coverage has no complete decoder surface/path"
			result.Candidates = []Hypothesis{lit}
			result.KeptIDs = []string{}
			result.KeptCount = 0
			return result, nil
		}
		h.Features = append([]float64{}, v[:]...)
		h.K = k
		h.Kept = true
		h.DisplaySegments = []Segment{}
		h.TailStart = h.Cut
		if h.Cut > 0 {
			h.DisplaySegments = append(h.DisplaySegments, Segment{"literal", 0, h.Cut, ""})
		}
		n := 0
		for _, s := range h.Segments {
			if s.Kind == "chinese" && n < k {
				h.DisplaySegments = append(h.DisplaySegments, s)
				h.TailStart = s.End
				n++
			}
		}
		if h.TailStart < len(q.Raw) {
			h.DisplaySegments = append(h.DisplaySegments, Segment{"pending", h.TailStart, len(q.Raw), ""})
		}
		h.Display = q.Raw[:h.Cut] + text + q.Raw[h.TailStart:]
		h.DisplayIdentity = identity(h.DisplaySegments)
		result.Candidates = append(result.Candidates, h)
		result.KeptIDs = append(result.KeptIDs, h.ID)
	}
	result.KeptCount = len(result.KeptIDs)
	return
}
func completeSpellingAtEnd(raw string) bool {
	if len(raw) < 2 || !strings.ContainsRune(" 3467", rune(raw[len(raw)-1])) {
		return false
	}
	tone, _ := layout.Standard.Symbol(rune(raw[len(raw)-1]))
	for n := 1; n <= 3 && n < len(raw); n++ {
		if _, ok := mixedparse.SpellExact(raw[len(raw)-1-n:len(raw)-1], tone); ok {
			return true
		}
	}
	return false
}
