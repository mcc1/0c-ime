package mixedparse

import (
	"encoding/json"
	"github.com/0c-biggo/0c-ime/core/english"
	"github.com/0c-biggo/0c-ime/core/lexicon"
	"github.com/0c-biggo/0c-ime/core/zhuyin"
	"math"
	"reflect"
	"strings"
	"testing"
)

type fixedScore struct{}

func (fixedScore) Prob(string, string) float64 { return math.Exp(-2) }
func (fixedScore) Prior(string) float64        { return math.Exp(-2) }
func tiny(words ...string) Evaluator {
	lex := lexicon.NewMem()
	for _, w := range words {
		r := map[string]string{"賺": "ㄓㄨㄢˋ", "錢": "ㄑㄧㄢˊ", "賺錢": "ㄓㄨㄢˋ ㄑㄧㄢˊ", "解": "ㄐㄧㄝˇ", "解賺": "ㄐㄧㄝˇ ㄓㄨㄢˋ"}[w]
		lex.Add(lexicon.Entry{Surface: w, Reading: zhuyin.ParseReading(r), PRW: 1})
	}
	return Evaluator{Lex: lex, Score: fixedScore{}, System: fixedScore{}, English: english.New([]english.Word{{Text: "hello", Weight: 1}})}
}
func get(t *testing.T, e Evaluator, raw string, offset int) Candidate {
	t.Helper()
	r, err := e.Generate(Context{CustomTokens: []string{"WQ"}}, raw, 0, nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range r.Candidates {
		if c.Lane == "strict" && FirstChinese(c.Segments) == offset {
			return c
		}
	}
	t.Fatalf("missing %s@%d", raw, offset)
	return Candidate{}
}
func TestWorkedVectors(t *testing.T) {
	cases := []struct {
		words  []string
		raw    string
		prefix int
		want   map[int]float64
	}{
		{[]string{"賺", "錢", "賺錢"}, "WQ5j04", 2, map[int]float64{1: 4.0 / 6, 2: .5, 3: 1, 4: 1, 7: -2, 10: 1, 24: 1}},
		{[]string{"賺", "錢", "賺錢"}, "WQ5j04fu06", 2, map[int]float64{1: .8, 2: 1, 3: 1, 4: 1, 7: -1, 10: 1, 24: 1}},
		{[]string{"賺", "錢"}, "WQ5j04fu06", 2, map[int]float64{1: .8, 2: 1, 4: 1, 7: -2, 10: 1, 24: 1}},
		{[]string{"賺錢"}, "WQ5j04", 2, map[int]float64{1: 4.0 / 6, 2: .5, 5: 1, 8: 1, 10: 1, 24: 1}},
		{[]string{"賺"}, "HELLO5j04", 5, map[int]float64{1: 4.0 / 9, 2: .5, 3: 1, 4: 1, 7: -2, 9: 1, 21: 1, 24: 1}},
	}
	for _, tc := range cases {
		c := get(t, tiny(tc.words...), tc.raw, tc.prefix)
		for i, v := range c.Features {
			if v == nil || math.Abs(*v-tc.want[i]) > 1e-12 {
				t.Errorf("%s words=%v feature%d got %v want %g", tc.raw, tc.words, i, v, tc.want[i])
			}
		}
	}
	r, _ := tiny("賺").Generate(Context{}, "WQ5j04", 0, nil)
	l := r.Candidates[len(r.Candidates)-1]
	for i, v := range l.Features {
		want := 0.0
		if i == 0 || i == 8 {
			want = 1
		}
		if *v != want {
			t.Fatal("literal vector", i, *v)
		}
	}
}
func TestStrictSpellingAndContext(t *testing.T) {
	for _, body := range []string{"uu", "u5", "WQ", "5j00", "3j", ""} {
		if _, ok := SpellExact(body, 'ˋ'); ok {
			t.Fatal(body)
		}
	}
	e := tiny("解賺")
	ctx := Context{Units: []Unit{{Reading: "ㄐㄧㄝˇ"}}, LastWord: "解"}
	if !e.reachable(ctx, "ㄓㄨㄢˋ") || e.reachable(Project(ctx, "X"), "ㄓㄨㄢˋ") {
		t.Fatal("literal reset")
	}
	ctx = Context{RetainedPrefix: &Retained{Reading: "ㄐㄧㄝˇ"}}
	if !e.reachable(ctx, "ㄓㄨㄢˋ") {
		t.Fatal("retained")
	}
	before := ctx.Clone()
	_, _ = e.Generate(ctx, "5j04", 0, nil)
	if !reflect.DeepEqual(ctx, before) {
		t.Fatal("mutated")
	}
}
func TestDeterminismOwnershipAndExhaustiveShort(t *testing.T) {
	e := tiny("賺", "錢", "賺錢")
	count := 0
	alphabet := "5ju0W34"
	var walk func(string, int)
	walk = func(s string, n int) {
		if len(s) > 0 && Tone(s[len(s)-1]) {
			a, err := e.Generate(Context{}, s, 0, nil)
			if err != nil {
				t.Fatal(err)
			}
			b, _ := e.Generate(Context{}, s, 0, nil)
			aa, _ := json.Marshal(a)
			bb, _ := json.Marshal(b)
			if string(aa) != string(bb) {
				t.Fatal("nondeterministic")
			}
			strict := 0
			for _, c := range a.Candidates {
				end := 0
				for _, z := range c.Segments {
					if z.Begin != end || z.End <= end {
						t.Fatal("ownership")
					}
					end = z.End
				}
				if end != len(s) {
					t.Fatal("coverage")
				}
				if c.Lane == "strict" && !c.Pruned {
					strict++
				}
			}
			if strict > 8 {
				t.Fatal("cap")
			}
			count++
		}
		if n == 0 {
			return
		}
		for _, k := range alphabet {
			walk(s+string(k), n-1)
		}
	}
	walk("", 4)
	t.Logf("exhaustive explicit-tone short strings=%d; max physical bytes=4", count)
}
func TestProtectedAndOverflow(t *testing.T) {
	e := tiny("賺")
	for _, c := range []Context{{Private: true}, {Paused: true}, {English: true}, {LocalBlocked: true}, {Toneless: true}, {Layout: "hsu"}} {
		r, err := e.Generate(c, "5j04", 0, nil)
		if err != nil || r.GenerateCalls != 0 || r.ScoreCalls != 0 {
			t.Fatal("protected", r, err)
		}
	}
	if _, err := e.Generate(Context{}, strings.Repeat("A", 65)+"4", 0, nil); err == nil {
		t.Fatal("overflow accepted")
	}
}
