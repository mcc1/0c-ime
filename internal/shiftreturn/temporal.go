//go:build shiftreturneval

package shiftreturn

// Temporal indices are zero-based research-key ordinals. Lifecycle events do
// not count as keys; event_index and raw key_offset remain in the evidence.
func temporal(f Fixture, snapshots []Snapshot, occurrences []CandidateOccurrence, generators []GeneratorDiagnostic) (first, last *int, lengths []int) {
	byQuery := map[string]map[string]Hypothesis{}
	for _, o := range occurrences {
		if byQuery[o.QueryID] == nil {
			byQuery[o.QueryID] = map[string]Hypothesis{}
		}
		byQuery[o.QueryID][o.Candidate.ID] = o.Candidate
	}
	gs := map[string]GeneratorDiagnostic{}
	for _, g := range generators {
		gs[g.QueryID] = g
	}
	correct := []bool{}
	eligible := []bool{}
	run := 0
	lengths = []int{}
	_, firstEnd := firstChinese(f.Gold[0])
	for _, s := range snapshots {
		if f.Events[s.EventIndex].Type != "key" {
			continue
		}
		right := false
		literalHarm := false
		ready := false
		if s.QueryID != nil {
			g := gs[*s.QueryID]
			raw := g.Raw
			ready = f.Intent == "return" && len(raw) >= firstEnd && firstEnd >= 0
			if g.Status == "ok" {
				h := literal(Query{Raw: raw})
				h.Display = s.Display
				if s.SelectedID != nil {
					h = byQuery[*s.QueryID][*s.SelectedID]
				} else {
					h.DisplaySegments = append([]Segment{}, s.Freeze...)
					at := prevEnd(h.DisplaySegments)
					if at < len(raw) {
						h.DisplaySegments = append(h.DisplaySegments, Segment{"literal", at, len(raw), ""})
					}
				}
				if f.Intent == "literal" {
					right = h.Display == raw
					literalHarm = !right
				} else {
					for _, id := range projectedGoldIDs(f, len(raw)) {
						right = right || "h:"+HashBytes([]byte(h.DisplayIdentity)) == id
					}
					begin, _ := firstChinese(f.Gold[0])
					for _, seg := range h.DisplaySegments {
						if seg.Kind == "chinese" && seg.Begin < min(begin, len(raw)) {
							literalHarm = true
						}
					}
				}
			}
		}
		index := len(correct)
		correct = append(correct, right)
		eligible = append(eligible, ready)
		if ready && right && first == nil {
			first = ptr(index)
		}
		if literalHarm {
			run++
		} else if run > 0 {
			lengths = append(lengths, run)
			run = 0
		}
	}
	stable := true
	for i := len(correct) - 1; i >= 0; i-- {
		stable = stable && correct[i]
		if stable && (eligible[i] || f.Intent == "literal") {
			last = ptr(i)
		}
	}
	return
}
func transitionCounts(previous, current []Segment, common int) (revision, resolution bool) {
	a, b := projection(previous, common, false), projection(current, common, false)
	for i := range a {
		if a[i].Kind != "chinese" && b[i].Kind != "chinese" {
			continue
		}
		// A syllable completed by the new key is an ordinary addition. Its tone
		// endpoint was not yet in the common processed prefix.
		if a[i].Kind == "chinese" && a[i].End > common || b[i].Kind == "chinese" && b[i].End > common {
			continue
		}
		if a[i].Kind == "pending" && b[i].Kind == "chinese" {
			resolution = true
		} else if a[i] != b[i] {
			revision = true
		}
	}
	return
}
func revisionCounts(f Fixture, snapshots []Snapshot, occurrences []CandidateOccurrence) (revisions, resolutions int) {
	candidates := map[string]map[string]Hypothesis{}
	for _, o := range occurrences {
		if candidates[o.QueryID] == nil {
			candidates[o.QueryID] = map[string]Hypothesis{}
		}
		candidates[o.QueryID][o.Candidate.ID] = o.Candidate
	}
	var previous *Hypothesis
	for _, s := range snapshots {
		if f.Events[s.EventIndex].Type != "key" || s.QueryID == nil {
			continue
		}
		if s.SelectedID == nil {
			previous = nil
			continue
		}
		h, ok := candidates[*s.QueryID][*s.SelectedID]
		if !ok {
			previous = nil
			continue
		}
		if previous != nil {
			common := min(prevEnd(previous.DisplaySegments), prevEnd(h.DisplaySegments))
			r, q := transitionCounts(previous.DisplaySegments, h.DisplaySegments, common)
			if r {
				revisions++
			}
			if q {
				resolutions++
			}
		}
		previous = &h
	}
	return
}
