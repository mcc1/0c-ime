//go:build shiftreturneval

package shiftreturn

import (
	"fmt"
	"math"
	"sort"
)

// FirstWordSensitivity holds EC coefficients fixed and replaces W only with
// its first-complete-syllable snapshot. It never trains or creates a main arm.
func FirstWordSensitivity(fs []Fixture, occurrences []CandidateOccurrence, model Model) ([]map[string]any, error) {
	byFixture := map[string]Fixture{}
	for _, f := range fs {
		byFixture[f.ID] = f
	}
	ordered := append([]CandidateOccurrence{}, occurrences...)
	sort.SliceStable(ordered, func(i, j int) bool {
		a, b := ordered[i], ordered[j]
		if a.FixtureID != b.FixtureID {
			return a.FixtureID < b.FixtureID
		}
		if a.EventIndex != b.EventIndex {
			return a.EventIndex < b.EventIndex
		}
		if a.Candidate.Cut != b.Candidate.Cut {
			return a.Candidate.Cut < b.Candidate.Cut
		}
		return a.Candidate.Identity < b.Candidate.Identity
	})
	first := map[string][4]float64{}
	last := map[string]int{}
	queries := map[string][]CandidateOccurrence{}
	for _, o := range ordered {
		if o.Condition != "EC-revise" {
			continue
		}
		if o.EventIndex > last[o.FixtureID] {
			last[o.FixtureID] = o.EventIndex
		}
		queries[o.QueryID] = append(queries[o.QueryID], o)
		if len(o.Candidate.Readings) > 0 {
			key := fmt.Sprintf("%s:%d:%s", o.FixtureID, o.Candidate.Cut, o.Candidate.FirstReading)
			if _, ok := first[key]; !ok {
				var w [4]float64
				copy(w[:], o.Candidate.Features[9:13])
				first[key] = w
			}
		}
	}
	results := []map[string]any{}
	for _, query := range queries {
		if len(query) == 0 {
			continue
		}
		firstOccurrence := query[0]
		if firstOccurrence.EventIndex != last[firstOccurrence.FixtureID] {
			continue
		}
		var literalCandidate *Hypothesis
		for _, o := range query {
			if len(o.Candidate.Readings) == 0 {
				h := o.Candidate
				literalCandidate = &h
				break
			}
		}
		if literalCandidate == nil {
			return nil, fmt.Errorf("sensitivity query lacks literal")
		}
		literalScore, err := score(model, *literalCandidate)
		if err != nil {
			return nil, err
		}
		chosen := *literalCandidate
		best := math.Inf(-1)
		sort.SliceStable(query, func(i, j int) bool {
			if query[i].Candidate.Cut != query[j].Candidate.Cut {
				return query[i].Candidate.Cut < query[j].Candidate.Cut
			}
			return query[i].Candidate.Identity < query[j].Candidate.Identity
		})
		for _, o := range query {
			if !o.Allowed || len(o.Candidate.Readings) == 0 {
				continue
			}
			h := o.Candidate
			h.Features = append([]float64{}, h.Features...)
			w := first[fmt.Sprintf("%s:%d:%s", o.FixtureID, h.Cut, h.FirstReading)]
			copy(h.Features[9:13], w[:])
			value, err := score(model, h)
			if err != nil {
				return nil, err
			}
			if value > best {
				best = value
				if value-literalScore > model.Margin {
					chosen = h
				}
			}
		}
		f := byFixture[firstOccurrence.FixtureID]
		boundary, surface, english := grade(f, chosen)
		results = append(results, map[string]any{"fixture_id": f.ID, "component_id": firstOccurrence.ComponentID, "selected_id": chosen.ID, "final_boundary_error": boundary, "final_surface_error": surface, "final_english_error": english})
	}
	sort.Slice(results, func(i, j int) bool { return results[i]["fixture_id"].(string) < results[j]["fixture_id"].(string) })
	return results, nil
}
