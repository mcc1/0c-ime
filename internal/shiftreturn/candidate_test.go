//go:build shiftreturneval

package shiftreturn

import (
	"math"
	"reflect"
	"strings"
	"testing"

	"github.com/0c-biggo/0c-ime/core/layout"
	"github.com/0c-biggo/0c-ime/core/lexicon"
	"github.com/0c-biggo/0c-ime/core/mixedparse"
	"github.com/0c-biggo/0c-ime/core/zhuyin"
)

type toyScore map[string]float64

func (s toyScore) Prob(left, word string) float64 { return s[word] }

type toyEnglish map[string]bool

func (s toyEnglish) Has(word string) bool { return s[strings.ToLower(word)] }
func tinyProbe(single bool) *Probe {
	lex := lexicon.NewMem()
	if !single {
		lex.Add(lexicon.Entry{Surface: "窄小", Reading: zhuyin.ParseReading("ㄓㄞˇ ㄒㄧㄠˇ"), PRW: 1})
	}
	score := toyScore{"窄小": math.Exp(-2)}
	if single {
		lex.Add(lexicon.Entry{Surface: "窄", Reading: zhuyin.ParseReading("ㄓㄞˇ"), PRW: 1})
		score["窄"] = math.Exp(-2)
	}
	return &Probe{Lex: lex, Score: score, English: toyEnglish{"hello": true, "a": true, "i": true, "x": true}}
}
func query(raw string) Query {
	return Query{Version: Version, Kind: "query", Demo: true, Raw: raw, Eligible: true}
}
func cutCandidate(t *testing.T, r Result, cut int) Hypothesis {
	t.Helper()
	for _, h := range r.Candidates[1:] {
		if h.Cut == cut {
			return h
		}
	}
	t.Fatalf("missing cut %d: %s", cut, r.Status)
	return Hypothesis{}
}
func TestSpellAllStandardRoundtrip(t *testing.T) {
	// Enumerate every standard 1–3 body key and all five tone keys.
	keys := "1qaz2wsxedcrfv5tgbyhnujm8ik,9ol.0p;/-"
	valid := 0
	var walk func(string, int)
	walk = func(body string, left int) {
		if body != "" {
			for _, toneKey := range " 3467" {
				tone, _ := layout.Standard.Symbol(toneKey)
				s, ok := mixedparse.SpellExact(body, tone)
				if !ok {
					continue
				}
				valid++
				mapped := ""
				for _, k := range body {
					sym, _ := layout.Standard.Symbol(k)
					mapped += string(sym)
				}
				if toneKey != ' ' {
					mapped += string(tone)
				}
				if string(s) != mapped || !s.Valid() {
					t.Fatalf("%q -> %q differs %q", body+string(toneKey), s, mapped)
				}
			}
		}
		if left > 0 {
			for _, k := range keys {
				walk(body+string(k), left-1)
			}
		}
	}
	walk("", 3)
	if valid < 1000 {
		t.Fatal(valid)
	}
	for _, body := range []string{"", "uu", "u5", "59v", "vvvv"} {
		tone, _ := layout.Standard.Symbol('3')
		if _, ok := mixedparse.SpellExact(body, tone); ok {
			t.Fatalf("illegal body accepted %q", body)
		}
	}
}
func TestCandidateWorkedAndFeatures(t *testing.T) {
	cases := []struct {
		raw, display string
		single       bool
		k            int
		want         Vector
	}{
		{"x593", "x593", false, 0, Vector{0, 0, 0, 0, 0, -2, 0, 0, 0, 0, 0, 0, 1}},
		{"x593vul3", "x窄小", false, 2, Vector{0, 0, 0, 0, 0, -2, 0, 1, 0, 2, 1, -1, 0}},
		{"x593v", "x593v", false, 0, Vector{0, 0, 0, 0, 0, -2, 0, 0, 1, 0, 0, 0, 1}},
		{"x593v", "x窄v", true, 1, Vector{0, 0, 0, 0, 0, -2, 0, 0, 1, 1, 0, -2, 0}},
	}
	for _, c := range cases {
		p := tinyProbe(c.single)
		r, e := p.Generate(query(c.raw))
		if e != nil {
			t.Fatal(e)
		}
		h := cutCandidate(t, r, 1)
		if h.Display != c.display || h.K != c.k {
			t.Fatalf("%s -> %s k%d", c.raw, h.Display, h.K)
		}
		v := Features(query(c.raw), h)
		for j, w := range c.want {
			if math.Abs(v[j]-w) > 1e-12 {
				t.Fatalf("%s feature%d=%g want%g", c.raw, j, v[j], w)
			}
		}
		var decoded Hypothesis
		if e = Decode(Canonical(h), &decoded); e != nil {
			t.Fatal(e)
		}
	}
}
func TestFeatureSnapshotAndNoFuture(t *testing.T) {
	p := tinyProbe(false)
	var first Vector
	for _, raw := range []string{"x593", "x593v", "x593vul3"} {
		r, e := p.Generate(query(raw))
		if e != nil {
			t.Fatal(e)
		}
		v := Features(query(raw), cutCandidate(t, r, 1))
		if first == (Vector{}) {
			first = v
		}
		if !reflect.DeepEqual(v[:7], first[:7]) {
			t.Fatal("future changed B")
		}
	}
	p = tinyProbe(true)
	r, e := p.Generate(query("HELLO593"))
	if e != nil {
		t.Fatal(e)
	}
	if cutCandidate(t, r, 5).Features[1] != 1 {
		t.Fatal("English case source parity")
	}
}
func TestCandidateLimitsAndCustom(t *testing.T) {
	for _, n := range []int{63, 64, 65, 129, 511, 512, 513} {
		p := tinyProbe(true)
		r, e := p.Generate(query(strings.Repeat("A", n)))
		if e != nil {
			t.Fatal(e)
		}
		if r.Candidates[0].Display != strings.Repeat("A", n) {
			t.Fatal("lost literal")
		}
		if (r.Status == "capacity_failure") != (n > 512) {
			t.Fatal("wrong limit")
		}
		if n > 512 && p.GenerateCalls != 0 {
			t.Fatal("overflow generated")
		}
	}
	for _, raw := range []string{"i", "ip", "ipv6", "ipv6593"} {
		p := tinyProbe(true)
		q := query(raw)
		q.CustomTokens = []string{"ipv6", "ls"}
		r, e := p.Generate(q)
		if e != nil {
			t.Fatal(e)
		}
		for _, h := range r.Candidates[1:] {
			if h.Cut < 4 {
				t.Fatal("custom token crossed")
			}
		}
	}
	p := tinyProbe(true)
	p.ProbeLexQueries = MaxLexQueries
	r, e := p.Generate(query("x593"))
	if e != nil || r.Status != "resource_exhausted" || r.UncappedIDs != nil || p.ProbeLexQueries != MaxLexQueries {
		t.Fatal("budget not fail closed", r.Status, e)
	}
}
func TestRenderZeroProbabilitySourceSemantics(t *testing.T) {
	p := tinyProbe(true)
	p.Score = toyScore{}
	r, e := p.Generate(query("x593"))
	if e != nil {
		t.Fatal(e)
	}
	h := cutCandidate(t, r, 1)
	// The actual Decoder retains an exact path with score -Inf, so its text is
	// available. Missing scores do not imply missing geometry or invented text.
	if r.Status != "ok" || h.Display != "x窄" || h.Features[6] != 1 || h.Features[12] != 1 {
		t.Fatalf("zero-probability semantics: %+v", h)
	}
}
func TestCandidateSpaceAndCanonicalCap(t *testing.T) {
	lex := lexicon.NewMem()
	lex.Add(lexicon.Entry{Surface: "身", Reading: zhuyin.ParseReading("ㄕㄣ"), PRW: 1})
	p := &Probe{Lex: lex, Score: toyScore{"身": 1}}
	r, e := p.Generate(query("xgp "))
	if e != nil {
		t.Fatal(e)
	}
	h := cutCandidate(t, r, 1)
	if h.Display != "x身" || h.Segments[1].End != 4 {
		t.Fatal("first-tone space discarded")
	}
	// Three valid suffix bodies per four-key group give 384 hypotheses at
	// 512 bytes. The canonical first 256 survive, independently of any gold.
	lex = lexicon.NewMem()
	score := toyScore{}
	for i, reading := range []string{"ㄓㄨㄢˋ", "ㄨㄢˋ", "ㄢˋ"} {
		word := []string{"轉", "萬", "案"}[i]
		lex.Add(lexicon.Entry{Surface: word, Reading: zhuyin.ParseReading(reading), PRW: 1})
		score[word] = 1
	}
	p = &Probe{Lex: lex, Score: score}
	r, e = p.Generate(query(strings.Repeat("5j04", 128)))
	if e != nil {
		t.Fatal(e)
	}
	if r.Status != "ok" || r.UncappedCount == nil || *r.UncappedCount != 384 || r.KeptCount != 256 || r.CapPruned == nil || !*r.CapPruned {
		t.Fatalf("cap counts: status=%s uncapped=%v kept=%d", r.Status, r.UncappedCount, r.KeptCount)
	}
	for i, id := range r.KeptIDs {
		if id != r.UncappedIDs[i] {
			t.Fatal("noncanonical cap")
		}
	}
	for _, h := range r.Candidates[1:] {
		if h.ID != "h:"+HashBytes([]byte(h.Identity)) {
			t.Fatal("identity hash")
		}
	}
}
