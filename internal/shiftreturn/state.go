//go:build shiftreturneval

package shiftreturn

import (
	"fmt"
	"math"
)

type State struct {
	FrozenDisplay                                                     string
	FragmentID, Raw, App, Prelude, Display, Mode                      string
	Eligible, Closed, Protected, Paused, Caps, LiteralLock            bool
	Selection                                                         *Hypothesis
	Freeze                                                            []Segment
	Snapshots                                                         []Snapshot
	LastResult                                                        *Result
	EventIndex, KeyOffset, ScoreCalls, ResolutionCount, RevisionCount int
}
type Policy struct {
	Probe             *Probe
	Model             Model
	Flow              string
	Enabled, Standard bool
	Toneless, Blocked bool
	CustomTokens      []string
	Condition         string
}

func projection(segs []Segment, n int, pendingLiteral bool) []Segment {
	out := make([]Segment, n)
	for _, s := range segs {
		for i := s.Begin; i < min(s.End, n); i++ {
			if i < 0 {
				continue
			}
			v := s
			if pendingLiteral && v.Kind == "pending" {
				v.Kind = "literal"
				v.Reading = ""
			}
			if v.Kind != "chinese" {
				v.Begin = 0
				v.End = 0
			}
			out[i] = v
		}
	}
	return out
}
func compatible(freeze, segs []Segment) bool {
	if len(freeze) == 0 {
		return true
	}
	n := freeze[len(freeze)-1].End
	a, b := projection(freeze, n, true), projection(segs, n, true)
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
func freezeProjection(segs []Segment) []Segment {
	out := append([]Segment{}, segs...)
	for i := range out {
		if out[i].Kind == "pending" {
			out[i].Kind = "literal"
			out[i].Reading = ""
		}
	}
	return out
}
func score(m Model, h Hypothesis) (float64, error) {
	if len(m.Coefficients) != 13 || len(m.Means) != 13 || len(m.Stds) != 13 || len(h.Features) != 13 {
		return 0, fmt.Errorf("expected 13 model/feature dimensions")
	}
	total := 0.0
	for j := 0; j < 13; j++ {
		x := h.Features[j]
		if j == 5 || j == 7 || j == 8 || j == 9 || j == 10 || j == 11 {
			missing := j == 5 && h.Features[6] == 1 || j == 11 && h.Features[12] == 1
			if missing || m.Stds[j] == 0 {
				x = 0
			} else {
				x = (x - m.Means[j]) / m.Stds[j]
			}
		}
		total += m.Coefficients[j] * x
	}
	if math.IsNaN(total) || math.IsInf(total, 0) {
		return 0, fmt.Errorf("nonfinite model score")
	}
	return total, nil
}
func queryID(fixture, condition string, event int, raw string) string {
	return "q:" + HashBytes(Canonical(map[string]any{"fixture_id": fixture, "condition": condition, "event_index": event, "raw": raw}))
}
func Step(s State, e Event, p Policy) (State, Snapshot, error) {
	if err := checkEvent(e); err != nil {
		return s, Snapshot{}, err
	}
	if p.Flow != "fixed" && p.Flow != "revise" {
		return s, Snapshot{}, fmt.Errorf("flow must be fixed/revise")
	}
	lastResult := s.LastResult
	s.LastResult = nil
	snap := Snapshot{Version: Version, Kind: "snapshot", EpisodeID: s.FragmentID, Condition: p.Condition, EventIndex: s.EventIndex, Handled: true, SafetyStatus: "complete", LearningUpdates: ptr(0), PersistentWrites: ptr(0)}
	s.EventIndex++
	closeEpisode := func(appendText, action, target string) {
		snap.Commit = s.Prelude + s.Display + appendText
		s.App += snap.Commit
		s.Prelude = ""
		s.Display = ""
		s.Raw = ""
		s.Freeze = nil
		s.FrozenDisplay = ""
		s.Eligible = false
		s.Closed = true
		s.Selection = nil
		snap.Handoff = &Handoff{Event: e, Action: action, TargetMode: target, Prelude: ""}
	}
	if s.Closed {
		snap.Handled = false
		snap.Handoff = &Handoff{Event: e, Action: "delegate", TargetMode: "retain", Prelude: ""}
		snap = snapshotOf(s, snap, p)
		return finish(s, snap, p), snap, nil
	}
	recompute := false
	switch e.Type {
	case "shift":
		if s.Eligible {
			closeEpisode("", "delegate", "zhuyin")
			s.Mode = "zhuyin"
		} else if (s.Mode == "" || s.Mode == "zhuyin") && p.Enabled && p.Standard && !p.Toneless && !p.Blocked && !s.Protected && !s.Paused && !s.Caps {
			s.Eligible = true
			s.Raw = ""
			s.Display = ""
			s.Mode = "english"
		} else {
			s.Mode = "english"
			snap.Handled = false
		}
	case "key":
		if s.Eligible && !s.Protected && !s.Paused && !p.Blocked && !s.Caps {
			s.Raw += e.Key
			s.KeyOffset++
			recompute = true
		} else {
			snap.Handled = false
		}
	case "backspace":
		if s.Eligible && len(s.Raw) > 0 {
			s.Raw = s.Raw[:len(s.Raw)-1]
			s.KeyOffset = len(s.Raw)
			if len(s.Freeze) > 0 && s.Freeze[len(s.Freeze)-1].End > len(s.Raw) {
				s.Freeze = nil
				s.LiteralLock = false
			}
			recompute = true
		} else {
			snap.Handled = false
		}
	case "select_literal":
		if !s.Eligible {
			return s, snap, fmt.Errorf("literal selection outside fragment")
		}
		s.LiteralLock = true
		s.Display = s.Raw
		h := literal(Query{Raw: s.Raw})
		s.Selection = &h
		s.Freeze = freezeProjection(h.DisplaySegments)
		s.FrozenDisplay = h.Display
	case "select_candidate":
		if !s.Eligible || lastResult == nil {
			return s, snap, fmt.Errorf("candidate selection outside query")
		}
		found := false
		for _, h := range lastResult.Candidates {
			if h.ID == e.CandidateID && compatible(s.Freeze, h.DisplaySegments) {
				selected := h
				s.Selection = &selected
				s.Display = h.Display
				s.Freeze = freezeProjection(h.DisplaySegments)
				s.FrozenDisplay = h.Display
				found = true
				break
			}
		}
		if !found {
			return s, snap, fmt.Errorf("candidate not in compatible current list")
		}

	case "cancel":
		oldPrelude := s.Prelude
		s.Raw = ""
		s.Display = ""
		s.Eligible = false
		s.Closed = true
		s.Selection = nil
		s.Freeze = nil
		s.Mode = "zhuyin"
		snap.Handoff = &Handoff{Event: e, Action: "restore_prelude", TargetMode: "zhuyin", Prelude: oldPrelude}
	case "enter", "commit":
		target := "english"
		if s.Selection != nil && s.Selection.K > 0 {
			target = "zhuyin"
		}
		if s.Eligible {
			closeEpisode("", "commit_only", target)
		} else {
			snap.Handled = false
		}
	case "paste":
		closeEpisode(*e.Text, "commit_only", "retain")
	case "focus", "cursor", "mode", "caps", "protect", "pause":
		if e.Type == "protect" {
			s.Protected = *e.On
		}
		if e.Type == "pause" {
			s.Paused = *e.On
		}
		if e.Type == "caps" {
			s.Caps = *e.On
		}
		target := "retain"
		if e.Type == "mode" {
			target = "zhuyin"
			if *e.On {
				target = "english"
			}
			s.Mode = target
		}
		if (e.Type == "protect" || e.Type == "pause") && !*e.On {
			break
		}
		closeEpisode("", "delegate", target)
	}
	if recompute {
		q := Query{Version: Version, Kind: "query", Raw: s.Raw, Eligible: true, CustomTokens: p.CustomTokens}
		if p.Probe == nil {
			return s, snap, fmt.Errorf("missing probe")
		}
		r, err := p.Probe.Generate(q)
		if err != nil {
			return s, snap, err
		}
		s.LastResult = &r
		snap.QueryID = ptr(queryID(s.FragmentID, p.Condition, snap.EventIndex, s.Raw))
		prev := s.Selection
		if r.Status != "ok" || s.LiteralLock {
			h := r.Candidates[0]
			s.Selection = &h
			s.Display = h.Display
			snap.SelectedID = &h.ID
		} else {
			selected := r.Candidates[0]
			literalScore := 0.0
			var err error
			if len(r.Candidates) > 1 {
				literalScore, err = score(p.Model, selected)
				if err != nil {
					return s, snap, err
				}
				s.ScoreCalls++
			}
			bestScore := math.Inf(-1)
			found := compatible(s.Freeze, selected.DisplaySegments)
			for _, h := range r.Candidates[1:] {
				if !compatible(s.Freeze, h.DisplaySegments) {
					continue
				}
				value, err := score(p.Model, h)
				if err != nil {
					return s, snap, err
				}
				s.ScoreCalls++
				if value > bestScore {
					bestScore = value
					if value-literalScore > p.Model.Margin {
						selected = h
						found = true
					}
				}
			}
			if found {
				s.Selection = &selected
				s.Display = selected.Display
				snap.SelectedID = &selected.ID
			} else {
				oldEnd := 0
				if len(s.Freeze) > 0 {
					oldEnd = s.Freeze[len(s.Freeze)-1].End
				}
				s.Display = s.FrozenDisplay + s.Raw[min(oldEnd, len(s.Raw)):]
				s.Selection = nil
			}
			if p.Flow == "fixed" && len(s.Freeze) == 0 && len(r.Candidates) > 1 && s.Selection != nil {
				s.Freeze = freezeProjection(s.Selection.DisplaySegments)
				s.FrozenDisplay = s.Selection.Display
			}
		}
		if prev != nil && s.Selection != nil {
			n := min(len(s.Raw), prevEnd(prev.DisplaySegments))
			a, b := projection(prev.DisplaySegments, n, false), projection(s.Selection.DisplaySegments, n, false)
			revision, resolution := false, false
			for i := range a {
				if a[i].Kind == "pending" && b[i].Kind == "chinese" {
					resolution = true
				} else if a[i] != b[i] {
					revision = true
				}
			}
			if resolution {
				s.ResolutionCount++
			}
			if revision {
				s.RevisionCount++
			}
		}
	}
	snap = snapshotOf(s, snap, p)
	s = finish(s, snap, p)
	return s, snap, nil
}
func prevEnd(segs []Segment) int {
	if len(segs) == 0 {
		return 0
	}
	return segs[len(segs)-1].End
}
func snapshotOf(s State, snap Snapshot, p Policy) Snapshot {
	snap.App = s.App
	snap.Prelude = s.Prelude
	snap.Display = s.Display
	snap.View = s.App + s.Prelude + s.Display
	snap.Eligible = s.Eligible
	snap.Closed = s.Closed
	snap.Freeze = s.Freeze
	snap.KeyOffset = s.KeyOffset
	snap.Protected = s.Protected || s.Paused || p.Blocked || s.Caps
	snap.ScoreCalls = ptr(s.ScoreCalls)
	calls := 0
	if p.Probe != nil {
		calls = p.Probe.GenerateCalls
	}
	snap.GenerateCalls = ptr(calls)
	snap.ResolutionCount = s.ResolutionCount
	snap.RevisionCount = s.RevisionCount
	return snap
}
func finish(s State, snap Snapshot, p Policy) State {
	s.Snapshots = append(append([]Snapshot{}, s.Snapshots...), snap)
	return s
}
