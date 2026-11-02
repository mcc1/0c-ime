//go:build shiftreturneval

package shiftreturn

import (
	"encoding/json"
	"github.com/0c-biggo/0c-ime/core/lexicon"
	"github.com/0c-biggo/0c-ime/core/zhuyin"
	"os"
	"strings"
	"testing"
)

func tinyPolicy(single bool, flow string) Policy {
	m := Model{Coefficients: make([]float64, 13), Means: make([]float64, 13), Stds: make([]float64, 13)}
	for i := range m.Stds {
		m.Stds[i] = 1
	}
	m.Coefficients[0] = -1
	return Policy{Probe: tinyProbe(single), Model: m, Flow: flow, Enabled: true, Standard: true, Condition: "EC-" + flow}
}
func typeRaw(t *testing.T, s State, p Policy, raw string) (State, Snapshot) {
	t.Helper()
	var snap Snapshot
	for _, key := range raw {
		var e error
		s, snap, e = Step(s, Event{Type: "key", Key: string(key)}, p)
		if e != nil {
			t.Fatal(e)
		}
	}
	return s, snap
}
func startFragment(t *testing.T, p Policy) State {
	t.Helper()
	s, _, e := Step(State{FragmentID: "tiny", Mode: "zhuyin"}, Event{Type: "shift"}, p)
	if e != nil {
		t.Fatal(e)
	}
	return s
}
func TestFlowWorkedRevisionAndFreeze(t *testing.T) {
	for _, flow := range []string{"revise", "fixed"} {
		p := tinyPolicy(false, flow)
		s := startFragment(t, p)
		s, first := typeRaw(t, s, p, "x593")
		if first.Display != "x593" {
			t.Fatal(first.Display)
		}
		s, final := typeRaw(t, s, p, "vul3")
		want := "x窄小"
		if flow == "fixed" {
			want = "x593vul3"
		}
		if final.Display != want {
			t.Fatalf("%s -> %q want%q", flow, final.Display, want)
		}
		calls := p.Probe.GenerateCalls
		s, committed, e := Step(s, Event{Type: "enter"}, p)
		if e != nil {
			t.Fatal(e)
		}
		if committed.App != want || committed.Commit != want || committed.View != want || !s.Closed || s.LastResult != nil || p.Probe.GenerateCalls != calls {
			t.Fatalf("commit mismatch: %+v", committed)
		}
		if flow == "revise" && s.ResolutionCount == 0 {
			t.Fatal("pending completion not resolution")
		}
	}
}
func TestFlowLifecycle(t *testing.T) {
	b, e := os.ReadFile("../../design/shift-return-v2/lifecycle.json")
	if e != nil {
		t.Fatal(e)
	}
	var doc struct {
		Cases []struct {
			Event      string
			Before     struct{ App, Prelude, Display string }
			AppendText string `json:"append_text"`
			Expected   struct{ App, Prelude, Display, View, Commit string }
		}
	}
	if e = json.Unmarshal(b, &doc); e != nil {
		t.Fatal(e)
	}
	for _, c := range doc.Cases {
		p := tinyPolicy(true, "revise")
		s := State{FragmentID: "life", App: c.Before.App, Prelude: c.Before.Prelude, Display: c.Before.Display, Raw: "x593", Eligible: true, Selection: &Hypothesis{K: 1}}
		ev := Event{Type: c.Event}
		switch c.Event {
		case "paste":
			ev.Text = &c.AppendText
		case "cursor":
			ev.To = ptr(4)
		case "mode", "caps", "protect", "pause":
			ev.On = ptr(true)
		}
		next, snap, e := Step(s, ev, p)
		if e != nil {
			t.Fatal(e)
		}
		if snap.App != c.Expected.App || snap.Prelude != c.Expected.Prelude || snap.Display != c.Expected.Display || snap.View != c.Expected.View || snap.Commit != c.Expected.Commit || !next.Closed || next.Eligible || next.LastResult != nil || p.Probe.GenerateCalls != 0 {
			t.Fatalf("%s: %+v", c.Event, snap)
		}
		var decoded Snapshot
		if e = Decode(Canonical(snap), &decoded); e != nil {
			t.Fatalf("%s schema: %v", c.Event, e)
		}
	}
}
func TestFlowHardGatesAndExplicitLocks(t *testing.T) {
	for _, field := range []string{"disabled", "nonstandard", "toneless", "blocked", "protected", "paused", "caps", "english"} {
		p := tinyPolicy(true, "revise")
		s := State{Mode: "zhuyin"}
		switch field {
		case "disabled":
			p.Enabled = false
		case "nonstandard":
			p.Standard = false
		case "toneless":
			p.Toneless = true
		case "blocked":
			p.Blocked = true
		case "protected":
			s.Protected = true
		case "paused":
			s.Paused = true
		case "caps":
			s.Caps = true
		case "english":
			s.Mode = "english"
		}
		s, _, e := Step(s, Event{Type: "shift"}, p)
		if e != nil {
			t.Fatal(e)
		}
		s, _ = typeRaw(t, s, p, "x593")
		if s.Eligible || p.Probe.GenerateCalls != 0 || s.ScoreCalls != 0 {
			t.Fatalf("gate %s generated", field)
		}
	}
	p := tinyPolicy(true, "revise")
	s := startFragment(t, p)
	s, _ = typeRaw(t, s, p, "x593")
	s, _, e := Step(s, Event{Type: "select_literal"}, p)
	if e != nil {
		t.Fatal(e)
	}
	s, snap := typeRaw(t, s, p, "vul3")
	if snap.Display != "x593vul3" {
		t.Fatal("literal lock crossed")
	}
	p = tinyPolicy(true, "revise")
	s = startFragment(t, p)
	s, _ = typeRaw(t, s, p, "x593")
	id := s.Selection.ID
	s, _, e = Step(s, Event{Type: "select_candidate", CandidateID: id}, p)
	if e != nil {
		t.Fatal(e)
	}
	if len(s.Freeze) == 0 {
		t.Fatal("candidate selection failed to freeze")
	}
	if _, _, e = Step(s, Event{Type: "select_candidate", CandidateID: "absent"}, p); e == nil {
		t.Fatal("absent candidate accepted")
	}
}
func TestFlowTieOverflowBackspace(t *testing.T) {
	p := tinyPolicy(true, "revise")
	p.Model.Coefficients[0] = 0
	s := startFragment(t, p)
	s, snap := typeRaw(t, s, p, "x593")
	if snap.Display != "x593" {
		t.Fatal("literal tie lost")
	}
	p = tinyPolicy(true, "revise")
	s = startFragment(t, p)
	s, snap = typeRaw(t, s, p, strings.Repeat("A", 513))
	if snap.Display != strings.Repeat("A", 513) || s.LastResult.Status != "capacity_failure" {
		t.Fatal("overflow lost raw")
	}
	s, snap, e := Step(s, Event{Type: "backspace"}, p)
	if e != nil {
		t.Fatal(e)
	}
	if len(s.Raw) != 512 || len(snap.Display) != 512 || s.LastResult.Status != "ok" {
		t.Fatal("backspace did not recover bounded query")
	}
}
func TestFlowExplicitSurfaceLockAndFixedHomophones(t *testing.T) {
	lex := lexicon.NewMem()
	lex.Add(lexicon.Entry{Surface: "窄", Reading: zhuyin.ParseReading("ㄓㄞˇ"), PRW: 1})
	lex.Add(lexicon.Entry{Surface: "寨小", Reading: zhuyin.ParseReading("ㄓㄞˇ ㄒㄧㄠˇ"), PRW: 1})
	for _, explicit := range []bool{false, true} {
		p := tinyPolicy(true, "fixed")
		p.Probe = &Probe{Lex: lex, Score: toyScore{"窄": .1, "寨小": .9}}
		s := startFragment(t, p)
		s, snap := typeRaw(t, s, p, "x593")
		if snap.Display != "x窄" {
			t.Fatal(snap.Display)
		}
		if explicit {
			var e error
			s, _, e = Step(s, Event{Type: "select_candidate", CandidateID: s.Selection.ID}, p)
			if e != nil {
				t.Fatal(e)
			}
		}
		_, snap = typeRaw(t, s, p, "vul3")
		want := "x寨小"
		if explicit {
			want = "x窄vul3"
		}
		if snap.Display != want {
			t.Fatalf("explicit=%v got%q want%q", explicit, snap.Display, want)
		}
	}
}
func TestFlowChineseTieCanonicalCut(t *testing.T) {
	lex := lexicon.NewMem()
	scores := toyScore{}
	for i, reading := range []string{"ㄓㄨㄢˋ", "ㄨㄢˋ", "ㄢˋ"} {
		word := []string{"轉", "萬", "案"}[i]
		lex.Add(lexicon.Entry{Surface: word, Reading: zhuyin.ParseReading(reading), PRW: 1})
		scores[word] = 1
	}
	p := tinyPolicy(false, "revise")
	p.Probe = &Probe{Lex: lex, Score: scores}
	s := startFragment(t, p)
	s, snap := typeRaw(t, s, p, "5j04")
	if s.Selection == nil || s.Selection.Cut != 0 || snap.Display != "轉" {
		t.Fatal("Chinese tie did not use canonical earliest cut")
	}
}

func TestLockedBudgetFailurePreservesShownSurface(t *testing.T) {
	for _, flow := range []string{"fixed", "revise"} {
		p := tinyPolicy(true, flow)
		s := startFragment(t, p)
		s, _ = typeRaw(t, s, p, "x593")
		if s.Selection == nil || s.Selection.K != 1 {
			t.Fatal("missing real Chinese selection")
		}
		if flow == "revise" {
			var err error
			s, _, err = Step(s, Event{Type: "select_candidate", CandidateID: s.Selection.ID}, p)
			if err != nil {
				t.Fatal(err)
			}
		}
		shown, scores := s.Display, s.ScoreCalls
		p.Probe.ProbeLexQueries = MaxLexQueries
		s, snap := typeRaw(t, s, p, "v")
		if s.Raw != "x593v" || s.Display != shown+"v" || s.ScoreCalls != scores || s.LastResult.Status != "resource_exhausted" || snap.SelectedID != nil {
			t.Fatalf("%s failed preservation: %+v", flow, s)
		}
		_, closed, err := Step(s, Event{Type: "enter"}, p)
		if err != nil || closed.Commit != shown+"v" || closed.Handoff.TargetMode != "zhuyin" {
			t.Fatalf("handoff %+v err%v", closed, err)
		}
	}
}
