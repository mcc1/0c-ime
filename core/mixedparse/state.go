package mixedparse

import (
	"fmt"
	"github.com/0c-biggo/0c-ime/core/zhuyin"
	"strings"
)

// Event is an offline harness value, not a desktop protocol extension.
type Event struct {
	Type          string     `json:"type"`
	Key           string     `json:"key,omitempty"`
	Reason        string     `json:"reason,omitempty"`
	Text          string     `json:"text,omitempty"`
	Target        *int       `json:"target,omitempty"`
	CommitVisible *bool      `json:"commit_visible,omitempty"`
	Private       bool       `json:"private,omitempty"`
	Paused        bool       `json:"paused,omitempty"`
	Blocked       bool       `json:"blocked,omitempty"`
	English       bool       `json:"english,omitempty"`
	Toneless      bool       `json:"toneless,omitempty"`
	Layout        string     `json:"layout,omitempty"`
	Generation    string     `json:"generation,omitempty"`
	Ref           *Selection `json:"ref,omitempty"`
}
type Selection struct {
	MenuEvent int      `json:"menu_event"`
	Kind      string   `json:"kind"`
	Span      [2]int   `json:"span"`
	Reading   []string `json:"reading"`
	Text      string   `json:"text"`
}
type Window struct {
	LastWord      string    `json:"last_word"`
	Index         int       `json:"index"`
	Offset        int       `json:"offset"`
	Raw           string    `json:"raw"`
	State         string    `json:"state"`
	FirstToneEnd  int       `json:"first_tone_end"`
	Opportunities int       `json:"opportunities"`
	Shown         []Segment `json:"partition"`
	Visible       string    `json:"visible"`
	Pending       string    `json:"pending"`
	RevisionCount int       `json:"revision_count"`
	ClosureReason string    `json:"closure_reason"`
	Context       Context   `json:"context"`
}
type Transition struct {
	Before              *Window     `json:"before"`
	After               *Window     `json:"after"`
	ClosedWindows       []Window    `json:"closed_windows"`
	Candidates          []Candidate `json:"candidates"`
	Commit              *string     `json:"commit"`
	GenerateCalls       int         `json:"generate_calls"`
	ScoreCalls          int         `json:"score_calls"`
	DelegateCount       int         `json:"delegate_count"`
	Locks               []Selection `json:"locks"`
	DelegatedUnverified bool        `json:"delegated_unverified"`
}
type Machine struct {
	Context       Context
	Current       *Window
	Windows       []Window
	FrozenVisible string
	Bypass        bool
	Locks         []Selection
	MenuEvent     int
	EventIndex    int
	NextOffset    int
}

func NewMachine(c Context) *Machine { return &Machine{Context: c.Clone(), MenuEvent: -1} }
func copyWindow(w *Window) *Window {
	if w == nil {
		return nil
	}
	x := *w
	x.Shown = append([]Segment(nil), w.Shown...)
	x.Context = w.Context.Clone()
	return &x
}
func (m *Machine) close(reason string, t *Transition) {
	if m.Current == nil {
		return
	}
	w := m.Current
	if w.State != "Closed" {
		w.State = "Closed"
		w.ClosureReason = reason
	}
	t.ClosedWindows = append(t.ClosedWindows, *copyWindow(w))
}
func (m *Machine) advance() {
	if m.Current == nil {
		return
	}
	w := *copyWindow(m.Current)
	m.Windows = append(m.Windows, w)
	m.FrozenVisible += w.Visible + w.Pending
	m.NextOffset += len(w.Raw)
	c := m.Context.Clone()
	for _, s := range w.Shown {
		if s.Kind == "literal" {
			c = Project(c, s.Text)
		} else {
			c.Units = append(c.Units, Unit{Reading: s.Reading, OriginalKeys: w.Raw[s.Begin:s.End]})
		}
	}
	if w.Pending != "" {
		c = Project(c, w.Pending)
	}
	if w.Pending == "" {
		c.LastWord = w.LastWord
	}
	m.Context = c
	m.Current = nil
}
func (m *Machine) Visible() string {
	s := m.FrozenVisible
	if m.Current != nil {
		s += m.Current.Visible + m.Current.Pending
	}
	return s
}

// Step uses a supplied causal policy only at tone opportunities. A nil policy
// selects literal solely for scripted state testing; it is not ranking evidence.
func (m *Machine) Step(ev Event, e Evaluator, previous []Segment, choose func(Window, []Candidate) (Candidate, error)) (Transition, error) {
	t := Transition{Before: copyWindow(m.Current)}
	idx := m.EventIndex
	m.EventIndex++
	if ev.Type == "environment" {
		c := Context{Private: ev.Private, Paused: ev.Paused, LocalBlocked: ev.Blocked, English: ev.English, Toneless: ev.Toneless, Layout: ev.Layout, Generation: ev.Generation}
		reason := "environment"
		if ev.Generation != m.Context.Generation && !c.Protected() {
			reason = "generation"
		}
		m.close(reason, &t)
		m.Context.Private = c.Private
		m.Context.Paused = c.Paused
		m.Context.LocalBlocked = c.LocalBlocked
		m.Context.English = c.English
		m.Context.Toneless = c.Toneless
		m.Context.Layout = c.Layout
		m.Context.Generation = c.Generation
		if c.Protected() {
			m.Bypass = true
		}
		t.After = copyWindow(m.Current)
		return t, nil
	}
	if ev.Type == "key" {
		m.MenuEvent = -1
		if len(ev.Key) != 1 || ev.Key[0] < 33 || ev.Key[0] > 126 || ev.Key == "`" {
			return t, fmt.Errorf("invalid physical key")
		}
		if m.Bypass || m.Context.Protected() {
			t.DelegateCount = 1
			t.After = copyWindow(m.Current)
			return t, nil
		}
		if m.Current != nil && (m.Current.State == "Closed" || len(m.Current.Raw) == 64) {
			if m.Current.State != "Closed" {
				m.close("overflow", &t)
			}
			m.advance()
		}
		if m.Current == nil {
			m.Current = &Window{Index: len(m.Windows), Offset: m.NextOffset, State: "Collecting", Context: m.Context.Clone()}
		}
		w := m.Current
		w.Raw += ev.Key
		w.Pending += ev.Key
		if Tone(ev.Key[0]) {
			r, err := e.Generate(w.Context, w.Raw, w.FirstToneEnd, previous)
			if err != nil {
				return t, err
			}
			t.Candidates = r.Candidates
			t.GenerateCalls = r.GenerateCalls
			t.ScoreCalls = r.ScoreCalls
			c := literal(w.Raw)
			if choose != nil {
				c, err = choose(*copyWindow(w), r.Candidates)
				if err != nil {
					return t, err
				}
			}
			if c.ID == "" {
				return t, fmt.Errorf("policy missing candidate")
			}
			if w.FirstToneEnd > 0 && ProjectedIdentity(w.Shown, w.FirstToneEnd) != ProjectedIdentity(c.Segments, w.FirstToneEnd) {
				w.RevisionCount++
			}
			w.Shown = append([]Segment(nil), c.Segments...)
			w.Visible = c.Text
			w.LastWord = c.LastWord
			w.Pending = ""
			w.Opportunities++
			if w.Opportunities == 1 {
				w.State = "Open1"
				w.FirstToneEnd = len(w.Raw)
			} else {
				m.close("second_tone", &t)
			}
		}
		t.After = copyWindow(w)
		t.Locks = append([]Selection(nil), m.Locks...)
		return t, nil
	}
	if ev.Type == "select" {
		if ev.Ref == nil || m.MenuEvent != ev.Ref.MenuEvent || m.Current == nil {
			return t, fmt.Errorf("selection without current frozen menu")
		}
		r := *ev.Ref
		w := m.Current
		if r.Span[0] < 0 || r.Span[1] > len(w.Raw) || r.Span[0] >= r.Span[1] {
			return t, fmt.Errorf("selection span")
		}
		if r.Kind == "original" {
			if r.Text != w.Raw[r.Span[0]:r.Span[1]] {
				return t, fmt.Errorf("original key mismatch")
			}
			p := []Segment{}
			for _, s := range w.Shown {
				if s.End <= r.Span[0] || s.Begin >= r.Span[1] {
					p = append(p, s)
				} else if s.Begin < r.Span[0] || s.End > r.Span[1] {
					return t, fmt.Errorf("selection crosses ownership")
				}
			}
			p = append(p, Segment{Kind: "literal", Begin: r.Span[0], End: r.Span[1], Text: r.Text})
			sortSegments(p)
			w.Shown = mergeLiterals(p)
			w.Visible = ""
			for _, s := range w.Shown {
				if s.Kind == "literal" {
					w.Visible += s.Text
				} else {
					return t, fmt.Errorf("original selection with unrendered neighboring Chinese")
				}
			}
		} else if r.Kind == "word" {
			validWord := false
			sr := []zhuyin.Syllable{}
			for _, reading := range r.Reading {
				sr = append(sr, zhuyin.Syllable(reading))
			}
			for _, entry := range e.Lex.Exact(sr) {
				validWord = validWord || entry.Surface == r.Text
			}
			if !validWord {
				return t, fmt.Errorf("selection text not in exact dictionary tuple")
			}
			rr := []string{}
			for _, s := range w.Shown {
				if s.Begin >= r.Span[0] && s.End <= r.Span[1] && s.Kind == "chinese" {
					rr = append(rr, s.Reading)
				}
			}
			if strings.Join(rr, "\x00") != strings.Join(r.Reading, "\x00") {
				return t, fmt.Errorf("selection reading mismatch")
			}
			if r.Span[0] == FirstChinese(w.Shown) && r.Span[1] == len(w.Raw) {
				w.Visible = w.Raw[:r.Span[0]] + r.Text
			} else {
				t.DelegatedUnverified = true
			}
		} else {
			return t, fmt.Errorf("selection kind")
		}
		m.Locks = append(m.Locks, r)
		m.close("select", &t)
		w.ClosureReason = "select"
		t.After = copyWindow(w)
		t.Locks = append([]Selection(nil), m.Locks...)
		return t, nil
	}
	if ev.Type != "finalize" {
		return t, fmt.Errorf("event type")
	}
	reason := ev.Reason
	m.close(reason, &t)
	switch reason {
	case "enter", "commit":
		v := m.Visible()
		t.Commit = &v
		t.DelegateCount = 1
		m.reset()
	case "cancel", "reset":
		m.reset()
		t.DelegateCount = 1
	case "focus_end":
		if ev.CommitVisible == nil {
			return t, fmt.Errorf("focus policy missing")
		}
		if *ev.CommitVisible {
			v := m.Visible()
			t.Commit = &v
		}
		m.reset()
		t.DelegateCount = 1
	case "open_candidates":
		m.MenuEvent = idx
		t.DelegateCount = 1
	case "space", "punctuation", "cursor", "backspace", "delete":
		t.DelegateCount = 1
		t.DelegatedUnverified = true
		if reason == "cursor" || reason == "backspace" || reason == "delete" {
			m.Bypass = true
		}
	default:
		return t, fmt.Errorf("unknown finalizer %s", reason)
	}
	t.After = copyWindow(m.Current)
	t.Locks = append([]Selection(nil), m.Locks...)
	return t, nil
}
func (m *Machine) reset() {
	m.Current = nil
	m.Windows = nil
	m.FrozenVisible = ""
	m.NextOffset = 0
	m.Locks = nil
	m.MenuEvent = -1
	m.Bypass = false
	c := m.Context.Clone()
	c.Units = nil
	c.RetainedPrefix = nil
	c.LastWord = ""
	m.Context = c
	m.Bypass = c.Protected()
}
func sortSegments(s []Segment) {
	for i := 1; i < len(s); i++ {
		for j := i; j > 0 && s[j].Begin < s[j-1].Begin; j-- {
			s[j], s[j-1] = s[j-1], s[j]
		}
	}
}
func mergeLiterals(p []Segment) []Segment {
	out := []Segment{}
	for _, s := range p {
		if len(out) > 0 && s.Kind == "literal" && out[len(out)-1].Kind == "literal" {
			out[len(out)-1].End = s.End
			out[len(out)-1].Text += s.Text
		} else {
			out = append(out, s)
		}
	}
	return out
}
