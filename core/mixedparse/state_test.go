package mixedparse

import (
	"strings"
	"testing"
)

func TestStateOverflowAndFreeze(t *testing.T) {
	e := tiny("賺", "錢", "賺錢")
	for _, tc := range []struct {
		raw    string
		window int
		state  string
		old    int
		reason string
	}{{strings.Repeat("A", 63), 0, "Collecting", 0, ""}, {strings.Repeat("A", 64), 0, "Collecting", 0, ""}, {strings.Repeat("A", 65), 1, "Collecting", 64, "overflow"}, {strings.Repeat("A", 60) + "su3AA", 1, "Collecting", 64, "overflow"}, {strings.Repeat("A", 61) + "su3f", 1, "Collecting", 64, "overflow"}, {strings.Repeat("A", 60) + "su3f6", 1, "Open1", 64, "overflow"}, {strings.Repeat("A", 58) + "su3cl3", 0, "Closed", 0, "second_tone"}} {
		m := NewMachine(Context{})
		for _, k := range tc.raw {
			if _, err := m.Step(Event{Type: "key", Key: string(k)}, e, nil, nil); err != nil {
				t.Fatal(err)
			}
		}
		if m.Current.Index != tc.window || m.Current.State != tc.state || m.NextOffset != tc.old {
			t.Fatalf("length%d current%+v", len(tc.raw), m.Current)
		}
		if len(m.Windows) > 0 && m.Windows[0].ClosureReason != tc.reason {
			t.Fatal("overflow closure")
		}
		if len(m.Windows) == 0 && m.Current.ClosureReason != tc.reason {
			t.Fatal("second tone")
		}
		got := ""
		for _, w := range m.Windows {
			got += w.Raw
		}
		got += m.Current.Raw
		if got != tc.raw {
			t.Fatal("lost/duplicated bytes")
		}
	}
}
func TestScriptedRevisionPendingCommit(t *testing.T) {
	e := tiny("賺", "錢", "賺錢")
	m := NewMachine(Context{})
	choose := func(w Window, c []Candidate) (Candidate, error) {
		for _, x := range c {
			if x.Lane == "strict" && FirstChinese(x.Segments) == 2 {
				return x, nil
			}
		}
		return literal(w.Raw), nil
	}
	for i, k := range "WQ5j04fu06" {
		tr, err := m.Step(Event{Type: "key", Key: string(k)}, e, nil, choose)
		if err != nil {
			t.Fatal(err)
		}
		if i == 5 && m.Current.Visible != "WQ賺" {
			t.Fatal(m.Current.Visible)
		}
		if i > 5 && i < 9 && m.Current.Visible != "WQ賺" {
			t.Fatal("incomplete reranked")
		}
		if i == 9 && (m.Current.State != "Closed" || m.Current.RevisionCount != 0 || tr.GenerateCalls != 1) {
			t.Fatal("normal extension revision")
		}
	}
	last := m.Visible()
	tr, err := m.Step(Event{Type: "finalize", Reason: "enter"}, e, nil, nil)
	if err != nil || tr.Commit == nil || *tr.Commit != last || tr.GenerateCalls != 0 || tr.ScoreCalls != 0 || tr.DelegateCount != 1 {
		t.Fatal("enter", tr, err)
	}
	if m.Current != nil {
		t.Fatal("reset")
	}
}
func TestEnvironmentSafetyAndNextContext(t *testing.T) {
	e := tiny("賺", "錢", "賺錢")
	m := NewMachine(Context{})
	choose := func(w Window, c []Candidate) (Candidate, error) { return Longest(c), nil }
	for _, k := range "5j04fu06A" {
		_, err := m.Step(Event{Type: "key", Key: string(k)}, e, nil, choose)
		if err != nil {
			t.Fatal(err)
		}
	}
	if m.Current.Context.LastWord != "賺錢" {
		t.Fatalf("lost frozen LM context: %+v", m.Current.Context)
	}
	old := m.Windows[0]
	tr, _ := m.Step(Event{Type: "environment", Private: true, Layout: "standard"}, e, nil, nil)
	if tr.GenerateCalls != 0 || tr.ScoreCalls != 0 {
		t.Fatal("privacy freeze inference")
	}
	for _, k := range "5j04" {
		tr, _ = m.Step(Event{Type: "key", Key: string(k)}, e, nil, nil)
		if tr.GenerateCalls != 0 || tr.ScoreCalls != 0 {
			t.Fatal("protected inference")
		}
	}
	if old.Raw != m.Windows[0].Raw || Identity(old.Shown) != Identity(m.Windows[0].Shown) {
		t.Fatal("closed mutation")
	}
	m.Step(Event{Type: "finalize", Reason: "reset"}, e, nil, nil)
	if !m.Bypass || !m.Context.Private {
		t.Fatal("field privacy cleared by composition reset")
	}
}

func TestProtectedEnvironmentSurvivesFinalizers(t *testing.T) {
	e := tiny("賺")
	for _, reason := range []string{"cancel", "reset", "enter", "commit"} {
		m := NewMachine(Context{})
		m.Step(Event{Type: "environment", Private: true, Paused: true, Layout: "standard"}, e, nil, nil)
		m.Step(Event{Type: "finalize", Reason: reason}, e, nil, nil)
		for _, k := range "5j04" {
			tr, err := m.Step(Event{Type: "key", Key: string(k)}, e, nil, nil)
			if err != nil || tr.GenerateCalls != 0 || tr.ScoreCalls != 0 || !m.Context.Private || !m.Context.Paused {
				t.Fatal(reason, tr, err)
			}
		}
	}
}
func TestSelectionDictionaryAndStaleMenu(t *testing.T) {
	e := tiny("賺")
	makeMenu := func() *Machine {
		m := NewMachine(Context{})
		for _, k := range "5j04" {
			m.Step(Event{Type: "key", Key: string(k)}, e, nil, func(w Window, c []Candidate) (Candidate, error) { return Longest(c), nil })
		}
		m.Step(Event{Type: "finalize", Reason: "open_candidates"}, e, nil, nil)
		return m
	}
	m := makeMenu()
	_, err := m.Step(Event{Type: "select", Ref: &Selection{MenuEvent: 4, Kind: "word", Span: [2]int{0, 4}, Reading: []string{"ㄓㄨㄢˋ"}, Text: "憑空杜撰"}}, e, nil, nil)
	if err == nil {
		t.Fatal("fabricated dictionary selection")
	}
	m = makeMenu()
	m.Step(Event{Type: "key", Key: "A"}, e, nil, nil)
	_, err = m.Step(Event{Type: "select", Ref: &Selection{MenuEvent: 4, Kind: "word", Span: [2]int{0, 4}, Reading: []string{"ㄓㄨㄢˋ"}, Text: "賺"}}, e, nil, nil)
	if err == nil {
		t.Fatal("stale menu")
	}
}

func TestAllFinalizersFreezeDelegateOnce(t *testing.T) {
	e := tiny("賺")
	for _, reason := range []string{"enter", "commit", "space", "punctuation", "cursor", "backspace", "delete", "cancel", "reset", "focus_end", "open_candidates"} {
		m := NewMachine(Context{})
		for _, k := range "WQ5j04A" {
			m.Step(Event{Type: "key", Key: string(k)}, e, nil, nil)
		}
		target := 0
		yes := true
		ev := Event{Type: "finalize", Reason: reason}
		if reason == "cursor" {
			ev.Target = &target
		}
		if reason == "punctuation" {
			ev.Text = "!"
		}
		if reason == "focus_end" {
			ev.CommitVisible = &yes
		}
		raw := m.Current.Raw
		tr, err := m.Step(ev, e, nil, nil)
		if err != nil || tr.GenerateCalls != 0 || tr.ScoreCalls != 0 || tr.DelegateCount != 1 || len(tr.ClosedWindows) != 1 || tr.ClosedWindows[0].Raw != raw {
			t.Fatal(reason, tr, err)
		}
	}
}
