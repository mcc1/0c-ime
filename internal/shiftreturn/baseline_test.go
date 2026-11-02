//go:build shiftreturneval

package shiftreturn

import (
	"reflect"
	"strings"
	"testing"

	"github.com/0c-biggo/0c-ime/core/english"
	"github.com/0c-biggo/0c-ime/core/lexicon"
	"github.com/0c-biggo/0c-ime/core/lm"
	"github.com/0c-biggo/0c-ime/core/proto"
	"github.com/0c-biggo/0c-ime/core/zhuyin"
)

func baselineSources() (*lexicon.Mem, *lm.Mem, *english.Dictionary) {
	lex := lexicon.NewMem()
	sys := lm.NewMem()
	for word, reading := range map[string]string{"中": "ㄓㄨㄥ", "窄": "ㄓㄞˇ", "小": "ㄒㄧㄠˇ", "窄小": "ㄓㄞˇ ㄒㄧㄠˇ", "身": "ㄕㄣ"} {
		lex.Add(lexicon.Entry{Surface: word, Reading: zhuyin.ParseReading(reading), PRW: 1})
		sys.SetUni(word, .01)
	}
	return lex, sys, english.New([]english.Word{{Text: "hello", Weight: 1}, {Text: "kevin", Weight: 1}, {Text: "nina", Weight: 1}, {Text: "lisa", Weight: 1}, {Text: "zoom", Weight: 1}})
}
func TestBaselineObserverParity(t *testing.T) {
	lex, sys, eng := baselineSources()
	for _, condition := range []string{"M", "R", "C", "A"} {
		on, e := NewBaseline(lex, sys, eng, condition, nil, true)
		if e != nil {
			t.Fatal(e)
		}
		off, e := NewBaseline(lex, sys, eng, condition, nil, false)
		if e != nil {
			t.Fatal(e)
		}
		requests := []proto.Request{{Type: proto.Key, Key: "5"}, {Type: proto.Key, Key: "j"}, {Type: proto.Key, Key: "/"}, {Type: proto.Key, Key: " "}, {Type: proto.Toggle}}
		for _, k := range "hello593gp " {
			requests = append(requests, proto.Request{Type: proto.Key, Key: string(k)})
		}
		requests = append(requests, proto.Request{Type: proto.Key, Key: proto.KeyEnter}, proto.Request{Type: proto.Mode, On: true}, proto.Request{Type: proto.Key, Key: "!"}, proto.Request{Type: proto.Mode, On: false}, proto.Request{Type: proto.Key, Key: "5"}, proto.Request{Type: proto.Key, Key: "9"}, proto.Request{Type: proto.Key, Key: "3"}, proto.Request{Type: proto.Candidates}, proto.Request{Type: proto.Select, Index: 0}, proto.Request{Type: proto.Commit}, proto.Request{Type: proto.Cancel})
		for _, r := range requests {
			a, b := on.Handle(r), off.Handle(r)
			if !reflect.DeepEqual(a.Snapshot, b.Snapshot) || a.App != b.App || a.View != b.View || StoreDigest(on.Model) != StoreDigest(off.Model) {
				t.Fatalf("%s observer changed Session at %+v", condition, r)
			}
		}
		if on.Model.Store == off.Model.Store {
			t.Fatal("shared store")
		}
	}
}
func TestLedgerPassThroughSpaceEnterAndCommit(t *testing.T) {
	l := ApplicationLedger{}
	if got := l.Apply(proto.Request{Type: proto.Key, Key: " "}, proto.Snapshot{Handled: false, Commit: "中"}); got != "中 " {
		t.Fatal(got)
	}
	if got := l.Apply(proto.Request{Type: proto.Key, Key: proto.KeyEnter}, proto.Snapshot{Handled: false}); got != "中 " {
		t.Fatal("named Enter leaked", got)
	}
	lex, sys, eng := baselineSources()
	b, e := NewBaseline(lex, sys, eng, "M", nil, true)
	if e != nil {
		t.Fatal(e)
	}
	b.Handle(proto.Request{Type: proto.Mode, On: true})
	for _, k := range "h264 hello2.4 " {
		b.Handle(proto.Request{Type: proto.Key, Key: string(k)})
	}
	last := b.Handle(proto.Request{Type: proto.Key, Key: proto.KeyEnter})
	if last.View != "h264 hello2.4 " {
		t.Fatal("pass-through lost bytes", last.View)
	}
}
func TestBaselineManualAndReferenceControls(t *testing.T) {
	lex, sys, eng := baselineSources()
	f := demoFixtures(t)[2]
	for _, condition := range []string{"M", "R", "C", "A"} {
		b, e := NewBaseline(lex, sys, eng, condition, nil, true)
		if e != nil {
			t.Fatal(e)
		}
		observed, e := ReplayBaseline(b, f, condition)
		if e != nil {
			t.Fatal(e)
		}
		toggles := 0
		for _, o := range observed {
			if o.Request.Type == proto.Toggle {
				toggles++
			}
		}
		want := 1
		if condition == "R" {
			want = 2
		}
		if condition == "C" || condition == "A" {
			want = 0
		}
		if toggles != want {
			t.Fatalf("%s toggles=%d want%d", condition, toggles, want)
		}
	}
}
func TestPrivacyScoringStoreAndGates(t *testing.T) {
	lex, sys, eng := baselineSources()
	p, m := NewScoringProbe(lex, sys, eng)
	before := StoreDigest(m)
	for _, raw := range []string{"x593", "x593vul3", "hello593gp "} {
		r, e := p.Generate(query(raw))
		if e != nil || r.Status != "ok" {
			t.Fatal(r.Status, e)
		}
	}
	if StoreDigest(m) != before {
		t.Fatal("hypothetical generation learned")
	}
	policy := tinyPolicy(true, "revise")
	s := startFragment(t, policy)
	s, _ = typeRaw(t, s, policy, "x593")
	s, _, e := Step(s, Event{Type: "protect", On: ptr(true)}, policy)
	if e != nil {
		t.Fatal(e)
	}
	calls, scores := policy.Probe.GenerateCalls, s.ScoreCalls
	s, _ = typeRaw(t, s, policy, "vul3")
	if policy.Probe.GenerateCalls != calls || s.ScoreCalls != scores {
		t.Fatal("protected experiment calls")
	}
	b, e := NewBaseline(lex, sys, eng, "M", nil, true)
	if e != nil {
		t.Fatal(e)
	}
	digest := StoreDigest(b.Model)
	for _, k := range "593" {
		b.Handle(proto.Request{Type: proto.Key, Key: string(k), Private: true})
	}
	b.Handle(proto.Request{Type: proto.Commit, Private: true})
	if StoreDigest(b.Model) != digest {
		t.Fatal("private baseline learned")
	}
}
func TestFinalizePreludeEnterAndAuthorLiteralCases(t *testing.T) {
	lex, sys, eng := baselineSources()
	for _, raw := range []string{"kevin87", "nina87", "lisa0304", "gpt4", "h264", "gpt4o", "hello4", "zoom3", "hello2.4"} {
		b, e := NewBaseline(lex, sys, eng, "M", nil, true)
		if e != nil {
			t.Fatal(e)
		}
		for _, key := range "5j/ " {
			b.Handle(proto.Request{Type: proto.Key, Key: string(key)})
		}
		b.Handle(proto.Request{Type: proto.Toggle})
		for _, key := range raw {
			b.Handle(proto.Request{Type: proto.Key, Key: string(key)})
		}
		last := b.Handle(proto.Request{Type: proto.Key, Key: proto.KeyEnter})
		if last.View != "中"+raw {
			t.Fatalf("prelude/literal lost %s: %q", raw, last.View)
		}
		if strings.Contains(last.View, "Enter") {
			t.Fatal("named key emitted")
		}
	}
}
func TestBaselineCustomTokenFactoryAndNeutralSource(t *testing.T) {
	lex, sys, english := baselineSources()
	for _, token := range []string{"ipv6", "ls"} {
		b, err := NewBaseline(lex, sys, english, "C", []string{token}, true)
		if err != nil {
			t.Fatal(err)
		}
		word, ok := b.Model.Vocab.Lookup(token)
		if !ok || len(word.Readings) != 0 || word.Source != "manual" || word.Added != 0 {
			t.Fatal("custom token factory changed source/readings/timestamp")
		}
		for _, key := range token {
			b.Handle(proto.Request{Type: proto.Key, Key: string(key)})
		}
		if !strings.HasPrefix(b.Ledger.App+b.Session.View().Text(), token) {
			t.Fatal("original Session rewrote explicit token")
		}
	}
	neutralLex := lexicon.NewMem()
	neutralLex.Add(lexicon.Entry{Surface: "喔", Reading: zhuyin.ParseReading("ㄛ"), PRW: 1})
	neutralLM := lm.NewMem()
	neutralLM.SetUni("喔", .1)
	probe, model := NewScoringProbe(neutralLex, neutralLM, english)
	before := StoreDigest(model)
	result, err := probe.Generate(query("xi7"))
	if err != nil {
		t.Fatal(err)
	}
	h := cutCandidate(t, result, 1)
	if h.Display != "x喔" || h.FirstReading != "ㄛ˙" {
		t.Fatal("empty-store Source neutral-particle view lost")
	}
	if StoreDigest(model) != before {
		t.Fatal("neutral fallback learned")
	}
}
