//go:build shiftreturneval

package shiftreturn

import "fmt"

func surfaceChanges(occurrences []CandidateOccurrence, snapshots []Snapshot) map[string]any {
	lookup := map[string]map[string]CandidateOccurrence{}
	for _, o := range occurrences {
		if lookup[o.QueryID] == nil {
			lookup[o.QueryID] = map[string]CandidateOccurrence{}
		}
		lookup[o.QueryID][o.Candidate.ID] = o
	}
	previous := map[string]CandidateOccurrence{}
	events, spans := 0, 0
	glyphs := func(o CandidateOccurrence) (map[string]string, bool) {
		h := o.Candidate
		tail := len(o.Raw) - h.TailStart
		if h.Cut > len(h.Display)-tail || tail < 0 {
			return nil, false
		}
		text := []rune(h.Display[h.Cut : len(h.Display)-tail])
		if len(text) != h.K {
			return nil, false
		}
		out := map[string]string{}
		i := 0
		for _, s := range h.DisplaySegments {
			if s.Kind == "chinese" {
				out[fmt.Sprintf("%d:%d:%s", s.Begin, s.End, s.Reading)] = string(text[i])
				i++
			}
		}
		return out, true
	}
	for _, s := range snapshots {
		if s.QueryID == nil {
			continue
		}
		key := rowKey(s.EpisodeID, s.Condition)
		if s.SelectedID == nil {
			delete(previous, key)
			continue
		}
		current, ok := lookup[*s.QueryID][*s.SelectedID]
		if !ok {
			continue
		}
		if old, ok := previous[key]; ok {
			a, validA := glyphs(old)
			b, validB := glyphs(current)
			if !validA || !validB {
				return map[string]any{"status": "unavailable", "events": nil, "changed_spans": nil, "reason": "surface/readings cannot be aligned to Unicode codepoints"}
			}
			changed := 0
			for span, glyph := range a {
				if value, ok := b[span]; ok && value != glyph {
					changed++
				}
			}
			if changed > 0 {
				events++
				spans += changed
			}
		}
		previous[key] = current
	}
	return map[string]any{"status": "complete", "events": events, "changed_spans": spans, "reason": "same owned reading span changed its surface; ordinary appended spans excluded"}
}
