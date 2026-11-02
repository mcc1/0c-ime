//go:build shiftreturneval

package shiftreturn

import (
	"fmt"
	"math"
	"reflect"
	"sort"
	"strings"
)

var Conditions = []string{"E0-fixed", "E0-revise", "EL-fixed", "EL-revise", "EW-fixed", "EW-revise", "EC-fixed", "EC-revise"}

type Evidence struct {
	Fixtures    []Fixture
	Sources     Sources
	Bank        []BankTask
	Assignment  Assignment
	Rows        []Row
	Snapshots   []Snapshot
	Occurrences []CandidateOccurrence
	Generators  []GeneratorDiagnostic
	Models      map[string]Model
	Lock        Lock
	// Verified raw-file provenance is necessary: decoded values cannot retain
	// whitespace/LF bytes used by the required file hashes.
	LockSHA256  string
	ModelSHA256 map[string]string
	Split       string
}
type graded struct {
	boundary, surface, english bool
	survived                   bool
	support                    string
}

func rowKey(fixture, condition string) string { return fixture + "\x00" + condition }
func sameSegments(a, b []Segment) bool        { return reflect.DeepEqual(a, b) }
func grade(f Fixture, h Hypothesis) (boundary, surface, english bool) {
	boundary = true
	surface = true
	english = false
	for _, gold := range f.Gold {
		if sameSegments(gold.Segments, h.DisplaySegments) {
			boundary = false
		}
		for _, accepted := range gold.AcceptedSurfaces {
			if accepted == h.Display {
				surface = false
			}
		}
	}
	if f.Intent == "return" {
		begin, _ := firstChinese(f.Gold[0])
		for _, seg := range h.DisplaySegments {
			if seg.Kind == "chinese" && seg.Begin < begin {
				english = true
			}
		}
		if h.Cut > begin { // A later cut preserves the prefix too; only raw rewriting is harm.
			english = english || !strings.HasPrefix(h.Display, f.ExpectedRaw[:begin])
		}
	} else {
		english = surface
	}
	return
}
func validateDiagnostic(g GeneratorDiagnostic) error {
	if g.QueryID != queryID(g.FixtureID, g.Condition, g.EventIndex, g.Raw) {
		return fmt.Errorf("generator query hash differs")
	}
	if g.KeptCount != len(g.KeptIDs) || g.KeptCount > MaxKeptChinese {
		return fmt.Errorf("generator kept count differs")
	}
	if g.ProbeLexQueries == nil || g.BestCalls == nil || g.GenerateCalls == nil {
		return &InsufficientError{"generator counters unavailable"}
	}
	if *g.ProbeLexQueries > MaxLexQueries || *g.BestCalls > MaxBestCalls {
		return fmt.Errorf("generator exceeded frozen limits")
	}
	if g.Status == "ok" {
		if g.UncappedCount == nil || *g.UncappedCount != len(g.UncappedIDs) || g.CapPruned == nil || *g.CapPruned != (len(g.UncappedIDs) > 256) || !reflect.DeepEqual(g.KeptIDs, g.UncappedIDs[:min(256, len(g.UncappedIDs))]) {
			return fmt.Errorf("generator cap evidence differs")
		}
	}
	return nil
}
func validateAndGrade(e Evidence) (map[string]graded, error) {
	fixtures := map[string]Fixture{}
	for _, f := range e.Fixtures {
		fixtures[f.ID] = f
	}
	generators := map[string]GeneratorDiagnostic{}
	generatorsByRow := map[string][]GeneratorDiagnostic{}
	occurrencesByRow := map[string][]CandidateOccurrence{}
	lastCounters := map[string][3]int{}
	for _, g := range e.Generators {
		if err := validateDiagnostic(g); err != nil {
			return nil, err
		}
		if _, ok := generators[g.QueryID]; ok {
			return nil, fmt.Errorf("duplicate generator query")
		}
		generators[g.QueryID] = g
		generatorsByRow[rowKey(g.FixtureID, g.Condition)] = append(generatorsByRow[rowKey(g.FixtureID, g.Condition)], g)
		key := rowKey(g.FixtureID, g.Condition)
		previous := lastCounters[key]
		current := [3]int{*g.GenerateCalls, *g.ProbeLexQueries, *g.BestCalls}
		for i := range current {
			if current[i] < previous[i] {
				return nil, fmt.Errorf("nonmonotone generator counters")
			}
		}
		lastCounters[key] = current
	}
	occurrences := map[string]map[string]CandidateOccurrence{}
	for _, o := range e.Occurrences {
		occurrencesByRow[rowKey(o.FixtureID, o.Condition)] = append(occurrencesByRow[rowKey(o.FixtureID, o.Condition)], o)
		g, ok := generators[o.QueryID]
		if !ok {
			return nil, fmt.Errorf("occurrence without generator")
		}
		if o.FixtureID != g.FixtureID || o.ComponentID != g.ComponentID || o.Condition != g.Condition || o.EventIndex != g.EventIndex || o.Raw != g.Raw || o.KeyOffset != g.KeyOffset {
			return nil, fmt.Errorf("occurrence/generator provenance differs")
		}
		if o.Candidate.ID != "h:"+HashBytes([]byte(o.Candidate.Identity)) || o.Candidate.Identity != identity(o.Candidate.Segments) || o.Candidate.DisplayIdentity != identity(o.Candidate.DisplaySegments) {
			return nil, fmt.Errorf("candidate identity differs")
		}
		if occurrences[o.QueryID] == nil {
			occurrences[o.QueryID] = map[string]CandidateOccurrence{}
		}
		if _, exists := occurrences[o.QueryID][o.Candidate.ID]; exists {
			return nil, fmt.Errorf("duplicate candidate occurrence")
		}
		occurrences[o.QueryID][o.Candidate.ID] = o
		if !o.Allowed && o.ExclusionReason != "freeze" || o.Allowed && o.ExclusionReason != "none" {
			return nil, fmt.Errorf("candidate compatibility label differs")
		}
		if o.Score != nil {
			model, ok := e.Models[strings.Split(o.Condition, "-")[0]]
			if !ok {
				return nil, fmt.Errorf("scored occurrence missing model")
			}
			value, err := score(model, o.Candidate)
			if err != nil || value != *o.Score {
				return nil, fmt.Errorf("candidate score differs from locked model")
			}
		}
	}
	for id, g := range generators {
		set := occurrences[id]
		if _, ok := set[g.LiteralID]; !ok {
			return nil, &InsufficientError{"literal occurrence missing"}
		}
		for _, kept := range g.KeptIDs {
			if _, ok := set[kept]; !ok {
				return nil, &InsufficientError{"kept occurrence missing"}
			}
		}
		if len(set) != len(g.KeptIDs)+1 {
			return nil, fmt.Errorf("extra occurrence outside capped generator")
		}
	}
	snapshots := map[string][]Snapshot{}
	querySnapshots := map[string]bool{}
	for _, s := range e.Snapshots {
		key := rowKey(s.EpisodeID, s.Condition)
		f, ok := fixtures[s.EpisodeID]
		if !ok {
			return nil, fmt.Errorf("snapshot fixture missing")
		}
		if s.EventIndex >= len(f.Events) {
			return nil, fmt.Errorf("snapshot event outside episode")
		}
		if s.View != s.App+s.Prelude+s.Display {
			return nil, fmt.Errorf("snapshot view ledger differs")
		}
		prior := snapshots[key]
		if len(prior) > 0 {
			last := prior[len(prior)-1]
			if s.EventIndex <= last.EventIndex || !strings.HasPrefix(s.App, last.App) {
				return nil, fmt.Errorf("event order/app rewrite")
			}
		}
		if s.QueryID != nil {
			g, ok := generators[*s.QueryID]
			if !ok {
				return nil, &InsufficientError{"snapshot generator missing"}
			}
			if querySnapshots[*s.QueryID] {
				return nil, fmt.Errorf("duplicate query snapshot")
			}
			querySnapshots[*s.QueryID] = true
			if g.FixtureID != s.EpisodeID || g.Condition != s.Condition || g.EventIndex != s.EventIndex || g.KeyOffset != s.KeyOffset {
				return nil, fmt.Errorf("snapshot generator provenance differs")
			}
			if g.Status == "ok" && s.Condition != "shared" && f.Intent != "compat" {
				model, ok := e.Models[strings.Split(s.Condition, "-")[0]]
				if !ok {
					return nil, &InsufficientError{"selection model unavailable"}
				}
				literalOccurrence := occurrences[*s.QueryID][g.LiteralID]
				literalScore, err := score(model, literalOccurrence.Candidate)
				if err != nil {
					return nil, err
				}
				var expected *string
				if literalOccurrence.Allowed {
					expected = ptr(g.LiteralID)
				}
				ordered := []CandidateOccurrence{}
				for _, candidate := range occurrences[*s.QueryID] {
					if candidate.Candidate.ID != g.LiteralID && candidate.Allowed {
						ordered = append(ordered, candidate)
					}
				}
				sort.Slice(ordered, func(i, j int) bool {
					a, b := ordered[i].Candidate, ordered[j].Candidate
					if a.Cut != b.Cut {
						return a.Cut < b.Cut
					}
					return a.Identity < b.Identity
				})
				best := math.Inf(-1)
				for _, candidate := range ordered {
					if candidate.Score == nil {
						return nil, &InsufficientError{"executed candidate score missing"}
					}
					if *candidate.Score > best {
						best = *candidate.Score
						if best-literalScore > model.Margin {
							expected = ptr(candidate.Candidate.ID)
						}
					}
				}
				if !reflect.DeepEqual(expected, s.SelectedID) {
					return nil, fmt.Errorf("selection differs from frozen score/margin/tie rule")
				}
			}
			if s.SelectedID != nil {
				selected, ok := occurrences[*s.QueryID][*s.SelectedID]
				if !ok || !selected.Allowed || selected.Candidate.Display != s.Display {
					return nil, fmt.Errorf("selected occurrence/display differs")
				}
			}
		} else if s.SelectedID != nil {
			return nil, fmt.Errorf("queryless selected ID")
		}
		snapshots[key] = append(snapshots[key], s)
	}
	for query := range generators {
		if !querySnapshots[query] {
			return nil, &InsufficientError{"generator lacks runtime snapshot"}
		}
	}
	grades := map[string]graded{}
	for _, r := range e.Rows {
		key := rowKey(r.FixtureID, r.Condition)
		if _, ok := grades[key]; ok {
			return nil, fmt.Errorf("duplicate row")
		}
		f, ok := fixtures[r.FixtureID]
		if !ok {
			return nil, fmt.Errorf("row fixture missing")
		}
		events := snapshots[key]
		if len(events) != len(f.Events) {
			return nil, &InsufficientError{"episode runtime evidence incomplete"}
		}
		var final Snapshot
		for _, s := range events {
			if f.Events[s.EventIndex].Type == "key" {
				final = s
			}
		}
		if f.Intent == "compat" {
			grades[key] = graded{support: r.Support}
			continue
		}
		if final.QueryID == nil {
			return nil, &InsufficientError{"final key query missing"}
		}
		g := generators[*final.QueryID]
		if g.ComponentID != r.ComponentID {
			return nil, fmt.Errorf("row component differs")
		}
		actual := graded{support: g.Status}
		if g.Status == "ok" {
			var h Hypothesis
			if final.SelectedID != nil {
				h = occurrences[*final.QueryID][*final.SelectedID].Candidate
			} else { // Fixed fallback projects its persisted freeze and literal extension.
				h = literal(Query{Raw: g.Raw})
				h.Display = final.Display
				h.DisplaySegments = append([]Segment{}, final.Freeze...)
				at := prevEnd(h.DisplaySegments)
				if at < len(g.Raw) {
					h.DisplaySegments = append(h.DisplaySegments, Segment{"literal", at, len(g.Raw), ""})
				}
			}
			actual.boundary, actual.surface, actual.english = grade(f, h)
			goldID := map[string]bool{}
			for _, gold := range f.Gold {
				goldID["h:"+HashBytes([]byte(identity(gold.Segments)))] = true
			}
			for _, id := range g.KeptIDs {
				actual.survived = actual.survived || goldID[id]
			}
			if f.Intent == "literal" {
				actual.survived = goldID[g.LiteralID]
			}
			if r.FinalBoundaryError == nil || r.FinalSurfaceError == nil || r.FinalEnglishError == nil || r.GoldSurvived == nil {
				return nil, &InsufficientError{"row grade unavailable"}
			}
			if *r.FinalBoundaryError != actual.boundary || *r.FinalSurfaceError != actual.surface || *r.FinalEnglishError != actual.english || *r.GoldSurvived != actual.survived {
				return nil, fmt.Errorf("row disagrees with independent final snapshot/occurrence grade")
			}
		} else if r.Support != g.Status {
			return nil, fmt.Errorf("row support differs")
		}
		last := events[len(events)-1]
		if f.Events[len(f.Events)-1].Type == "enter" && last.View != final.View {
			return nil, fmt.Errorf("Enter changed final visible output")
		}
		revisions, resolutions := revisionCounts(f, events, occurrencesByRow[key])
		if r.RevisionCount != revisions || r.ResolutionCount != resolutions {
			return nil, fmt.Errorf("row revision/resolution counts differ from ownership evidence")
		}
		first, stablePtr, lengths := temporal(f, events, occurrencesByRow[key], generatorsByRow[key])
		if !reflect.DeepEqual(first, r.FirstCorrect) || !reflect.DeepEqual(stablePtr, r.LastStable) || !reflect.DeepEqual(lengths, r.TemporaryErrorLengths) {
			return nil, fmt.Errorf("row temporal fields differ from runtime evidence")
		}
		grades[key] = actual
	}
	return grades, nil
}
func Summarize(e Evidence) (Report, error) {
	if err := ValidateFixtures(e.Fixtures); err != nil {
		return Report{}, err
	}
	if err := ValidateAssignment(e.Fixtures, e.Sources, e.Assignment, e.Assignment.FixturesSHA256); err != nil {
		return Report{}, err
	}
	grades, err := validateAndGrade(e)
	if err != nil {
		return Report{}, err
	}
	sourceComplete, sourceErr := qualifyEvidenceSources(e)
	if sourceErr != nil {
		return Report{}, sourceErr
	}
	report := Report{Version: Version, Kind: "report", Demo: e.Assignment.Demo, Outcome: "INCONCLUSIVE", PrimaryCondition: "EC-revise", Conditions: map[string]map[string]Metric{}, Paired: []PairedComparison{}, Derived: []Derived{}, SourceLockSHA256: e.LockSHA256, ModelsSHA256: e.ModelSHA256, DiagnosticsPath: "diagnostics.json", SafetyStatus: "complete"}
	if report.ModelsSHA256 == nil {
		report.ModelsSHA256 = map[string]string{}
	}
	report.Qualification.CategoryComponents = map[string]int{}
	report.Qualification.SourceComplete = sourceComplete
	report.Qualification.SurfaceGoldComplete = true
	fixtureByID := map[string]Fixture{}
	for _, f := range e.Fixtures {
		fixtureByID[f.ID] = f
	}
	components := map[string]Component{}
	for _, c := range e.Assignment.Components {
		components[c.ID] = c
		if c.Split == "test" {
			if c.PrimaryReturn != nil {
				report.Qualification.TestReturnComponents++
				f := fixtureByID[*c.PrimaryReturn]
				begin, _ := firstChinese(f.Gold[0])
				if begin > 0 {
					report.Qualification.TestEnglishComponents++
				}
				report.Qualification.CategoryComponents[f.Category]++
				if c.PrimaryReturnMetadata.ContrastEligible {
					report.Qualification.ContrastComponents++
				}
			}
			if c.PrimaryLiteral != nil {
				report.Qualification.TestLiteralComponents++
			}
		}
	}
	split := e.Split
	if split == "" {
		split = "test"
	}
	union := []string{}
	for _, c := range e.Assignment.Components {
		if c.Split == split && (c.PrimaryReturn != nil || c.PrimaryLiteral != nil) {
			union = append(union, c.ID)
		}
	}
	sort.Strings(union)
	rows := map[string]Row{}
	complete := true
	missing := false
	missingRows := false
	relevantRows := 0
	unsafe := false
	counts := make([]*int, 7)
	for i := range counts {
		counts[i] = ptr(0)
	}
	errors := map[string]map[string]map[string]bool{}
	for _, condition := range Conditions {
		errors[condition] = map[string]map[string]bool{}
		for _, m := range []string{"return_boundary", "return_surface", "literal_only", "return_english"} {
			errors[condition][m] = map[string]bool{}
		}
	}
	for _, r := range e.Rows {
		rows[rowKey(r.FixtureID, r.Condition)] = r
		c, ok := components[r.ComponentID]
		if !ok {
			return report, fmt.Errorf("row assignment component missing")
		}
		f := fixtureByID[r.FixtureID]
		primaryID := c.PrimaryReturn
		metadata := c.PrimaryReturnMetadata
		if f.Intent == "compat" {
			primaryID = nil
			metadata = nil
		}
		if f.Intent == "literal" {
			primaryID = c.PrimaryLiteral
			metadata = c.PrimaryLiteralMetadata
		}
		isPrimary := primaryID != nil && *primaryID == r.FixtureID
		if r.Intent != f.Intent || r.Primary != isPrimary || !reflect.DeepEqual(metadata, r.PrimaryMetadata) {
			return report, fmt.Errorf("row selected-primary metadata differs")
		}
		if metadata != nil && (r.PrimaryCategory != metadata.Category || r.ContextClass != metadata.ContextClass || r.ContrastEligible != metadata.ContrastEligible) {
			return report, fmt.Errorf("row metadata copy differs")
		}
		if c.Split != split {
			continue
		}
		relevantRows++
		if r.Condition == "shared" {
			continue
		}
		if _, ok := errors[r.Condition]; !ok {
			return report, fmt.Errorf("unknown condition")
		}
		g := grades[rowKey(r.FixtureID, r.Condition)]
		if g.support != "ok" {
			missing = true
		}
		counters := []*int{r.KeyFailures, r.CommitFailures, r.AppRewrites, r.LockViolations, r.HypotheticalLearningUpdates, r.PersistentWrites, r.ProtectedCalls}
		for i, n := range counters {
			if n == nil {
				counts[i] = nil
				complete = false
			} else {
				if *n != 0 {
					unsafe = true
				}
				if counts[i] != nil {
					*counts[i] += *n
				}
			}
		}
		if r.SafetyStatus != "complete" {
			complete = false
		}
		if isPrimary && g.support == "ok" {
			if f.Intent == "return" {
				errors[r.Condition]["return_boundary"][c.ID] = g.boundary
				errors[r.Condition]["return_surface"][c.ID] = g.surface
				begin, _ := firstChinese(f.Gold[0])
				if begin > 0 {
					errors[r.Condition]["return_english"][c.ID] = g.english
				}
			} else if f.Intent == "literal" {
				errors[r.Condition]["literal_only"][c.ID] = g.surface
			}
		}
	}
	for _, c := range e.Assignment.Components {
		if c.Split != split {
			continue
		}
		for _, id := range []*string{c.PrimaryReturn, c.PrimaryLiteral} {
			if id == nil {
				continue
			}
			for _, condition := range Conditions {
				if _, ok := rows[rowKey(*id, condition)]; !ok {
					missing = true
					missingRows = true
				}
			}
		}
	}
	for _, condition := range Conditions {
		report.Conditions[condition] = map[string]Metric{}
		for name, values := range errors[condition] {
			n := 0
			for _, v := range values {
				if v {
					n++
				}
			}
			expectedN := 0
			for _, component := range e.Assignment.Components {
				if component.Split != split {
					continue
				}
				switch name {
				case "return_boundary", "return_surface":
					if component.PrimaryReturn != nil {
						expectedN++
					}
				case "literal_only":
					if component.PrimaryLiteral != nil {
						expectedN++
					}
				case "return_english":
					if component.PrimaryReturn != nil {
						f := fixtureByID[*component.PrimaryReturn]
						begin, _ := firstChinese(f.Gold[0])
						if begin > 0 {
							expectedN++
						}
					}
				}
			}
			unknown := expectedN - len(values)
			m := metric(n+unknown, expectedN)
			m.ObservedErrors = n
			m.UnavailableComponents = unknown
			if unknown > 0 {
				m.Status = "unavailable"
				m.Value = nil
				m.Wilson95 = nil
				missing = true
			}
			report.Conditions[condition][name] = m
		}
	}
	comparisons := [][3]string{{"E0-revise", "EL-revise", "all_return"}, {"E0-revise", "EW-revise", "all_return"}, {"EL-revise", "EC-revise", "all_return"}, {"EW-revise", "EC-revise", "all_return"}, {"E0-fixed", "E0-revise", "all_return"}, {"EL-fixed", "EL-revise", "all_return"}, {"EW-fixed", "EW-revise", "all_return"}, {"EC-fixed", "EC-revise", "all_return"}, {"E0-revise", "EC-revise", "context_contrast"}, {"EL-revise", "EC-revise", "context_contrast"}}
	for _, comparison := range comparisons {
		base, new := map[string]bool{}, map[string]bool{}
		for id, v := range errors[comparison[0]]["return_boundary"] {
			if comparison[2] == "context_contrast" && !components[id].PrimaryReturnMetadata.ContrastEligible {
				continue
			}
			if n, ok := errors[comparison[1]]["return_boundary"][id]; ok {
				base[id] = v
				new[id] = n
			} else {
				missing = true
			}
		}
		p, err := Paired(comparison[0], comparison[1], comparison[2], base, new, union)
		if err != nil {
			return report, err
		}
		expectedN := 0
		for _, component := range e.Assignment.Components {
			if component.Split == split && component.PrimaryReturn != nil && (comparison[2] != "context_contrast" || component.PrimaryReturnMetadata.ContrastEligible) {
				expectedN++
			}
		}
		if p.N != expectedN {
			p.N = expectedN
			p.RelativeErrorReduction = nil
			p.AbsoluteErrorReductionCi95 = nil
			p.ValidBootstrap = 0
			missing = true
		}
		report.Paired = append(report.Paired, p)
	}
	report.Derived = deriveMetrics(errors, union)
	if missing {
		for i := range report.Derived {
			report.Derived[i].Value = nil
			report.Derived[i].Bootstrap95 = nil
			report.Derived[i].ValidBootstrap = 0
		}
	}
	if relevantRows == 0 || missingRows {
		for i := range counts {
			counts[i] = nil
		}
		complete = false
	}
	report.Safety.KeyFailures = counts[0]
	report.Safety.CommitFailures = counts[1]
	report.Safety.AppRewrites = counts[2]
	report.Safety.LockViolations = counts[3]
	report.Safety.HypotheticalLearningUpdates = counts[4]
	report.Safety.PersistentWrites = counts[5]
	report.Safety.ProtectedCalls = counts[6]
	if relevantRows == 0 {
		report.SafetyStatus = "not_run"
		report.SafetyReason = "no relevant replay rows supplied; safety counters unknown"
	} else if !complete {
		report.SafetyStatus = "incomplete"
		report.SafetyReason = "one or more required safety counters unavailable"
	}
	applyGates(&report, e, missing, unsafe, complete, sourceComplete)
	return report, nil
}
func applyGates(r *Report, e Evidence, missing, unsafe, complete, sourceComplete bool) {
	unavailable := func(reason string) Gate { return Gate{Status: "unavailable", Reason: reason} }
	test := func(pass bool, reason string) Gate {
		status := "fail"
		if pass {
			status = "pass"
		}
		return Gate{Status: status, Reason: reason}
	}
	r.Gates.G0 = unavailable("formal tool/data/cohort qualification or complete eight-condition evidence missing")
	q := r.Qualification
	samplesQualified, _ := sampleQualification(e.Fixtures, e.Bank, e.Assignment)
	qualified := sourceComplete && samplesQualified && len(e.Models) == 4 && len(e.ModelSHA256) == 4 && !r.Demo && !missing && q.TestReturnComponents >= 400 && q.TestLiteralComponents >= 400 && q.TestEnglishComponents >= 400 && q.ContrastComponents >= 100
	if qualified {
		r.Gates.G0 = test(true, "frozen qualification and complete evidence verified")
	}
	ec := r.Conditions["EC-revise"]
	ret := ec["return_boundary"]
	r.Gates.G1 = unavailable("candidate recall evidence unavailable")
	survived, total := 0, 0
	for _, row := range e.Rows {
		if (row.Condition == "EC-revise" || row.Condition == "shared") && row.Intent == "return" && row.Primary {
			for _, c := range e.Assignment.Components {
				if c.ID == row.ComponentID && c.Split == "test" {
					total++
					if row.GoldSurvived != nil && *row.GoldSurvived {
						survived++
					}
				}
			}
		}
	}
	if total > 0 {
		r.Gates.G1 = test(float64(survived)/float64(total) >= .99, "capped full-gold recall, including unavailable episodes as misses")
	}
	r.Gates.G2 = unavailable("return denominator unavailable")
	if ret.Value != nil {
		r.Gates.G2 = test(*ret.Value <= .1, "final boundary/reading errors; surface separately reported")
	}
	r.Gates.G3 = unavailable("context improvement evidence insufficient (baseline zero, cohort/discordance, or bootstrap unavailable)")
	p0, p1 := r.Paired[8], r.Paired[9]
	if p0.RelativeErrorReduction != nil && p1.RelativeErrorReduction != nil && p0.DiscordantComponents >= 20 && p1.DiscordantComponents >= 20 && len(p0.AbsoluteErrorReductionCi95) == 2 && len(p1.AbsoluteErrorReductionCi95) == 2 {
		r.Gates.G3 = test(*p0.RelativeErrorReduction >= .3 && p0.AbsoluteErrorReductionCi95[0] > 0 && p1.AbsoluteErrorReductionCi95[0] > 0, "predeclared contrast EC against both E0 and EL")
	}
	r.Gates.G4 = unavailable("literal risk denominators unavailable")
	literalOK := true
	for _, name := range []string{"literal_only", "return_english"} {
		m := ec[name]
		if m.Value == nil || len(m.Wilson95) != 2 {
			literalOK = false
			continue
		}
		literalOK = literalOK && *m.Value <= .01 && m.Wilson95[1] <= .01
		for _, base := range []string{"E0-revise", "EL-revise"} {
			literalOK = literalOK && m.Errors <= r.Conditions[base][name].Errors
		}
	}
	if ec["literal_only"].Value != nil && ec["return_english"].Value != nil {
		r.Gates.G4 = test(literalOK, "two component risk denominators and Wilson upper limits; author mandatory cases additionally required")
	}
	mandatoryComplete, mandatoryPassed := mandatoryLiteralStatus(e)
	if !mandatoryPassed {
		r.Gates.G4 = test(false, "specified author literal preservation case failed")
	} else if !mandatoryComplete && r.Gates.G4.Status == "pass" {
		r.Gates.G4 = unavailable("specified author preservation evidence missing")
	}
	r.Gates.G5 = unavailable("required safety evidence incomplete")
	if complete && !missing {
		r.Gates.G5 = test(!unsafe, "observed safety counters; source parity/regression additionally required")
	}
	// Until tool/mandatory/regression qualification is represented in the verified
	// frozen bundle, source qualification must remain false. No metric substitutes it.
	if r.Gates.G0.Status == "pass" && r.Gates.G1.Status == "pass" && r.Gates.G2.Status == "pass" && r.Gates.G3.Status == "pass" && r.Gates.G4.Status == "pass" && r.Gates.G5.Status == "pass" {
		r.Outcome = "GO_OUTPUT_EXPERIMENT"
	} else if r.Gates.G0.Status == "pass" && r.Gates.G1.Status != "unavailable" && r.Gates.G2.Status != "unavailable" && r.Gates.G3.Status != "unavailable" && r.Gates.G4.Status != "unavailable" && r.Gates.G5.Status != "unavailable" {
		for _, g := range []Gate{r.Gates.G1, r.Gates.G2, r.Gates.G3, r.Gates.G4, r.Gates.G5} {
			if g.Status == "fail" {
				r.Outcome = "NO_GO_METHOD"
			}
		}
	}
}
func deriveMetrics(errors map[string]map[string]map[string]bool, union []string) []Derived {
	names := []string{"interaction_I", "flow_gain", "flow_gain", "flow_gain", "flow_gain", "context_extra_revision_gain"}
	models := []*string{nil, ptr("E0"), ptr("EL"), ptr("EW"), ptr("EC"), nil}
	functions := []func([]string) *float64{}
	for index := range names {
		index := index
		functions = append(functions, func(sample []string) *float64 {
			sums := map[string]int{}
			n := 0
			for _, id := range sample {
				available := true
				for _, condition := range Conditions {
					if _, ok := errors[condition]["return_boundary"][id]; !ok {
						available = false
						break
					}
				}
				if !available {
					continue
				}
				n++
				for _, condition := range Conditions {
					if errors[condition]["return_boundary"][id] {
						sums[condition]++
					}
				}
			}
			if n == 0 {
				return nil
			}
			delta := 0
			switch index {
			case 0:
				delta = sums["EC-revise"] - sums["EL-revise"] - sums["EW-revise"] + sums["E0-revise"]
			case 5:
				delta = (sums["EC-fixed"] - sums["EC-revise"]) - (sums["E0-fixed"] - sums["E0-revise"])
			default:
				model := *models[index]
				delta = sums[model+"-fixed"] - sums[model+"-revise"]
			}
			return ptr(float64(delta) / float64(n))
		})
	}
	cis, valid := Bootstrap(union, functions)
	out := []Derived{}
	for i, name := range names {
		out = append(out, Derived{Name: name, Model: models[i], Cohort: "all_return", Value: functions[i](union), Bootstrap95: cis[i], ValidBootstrap: valid[i]})
	}
	return out
}

var MandatoryLiteralRaw = []string{"kevin87", "nina87", "lisa0304", "gpt4", "h264", "gpt4o", "hello4", "zoom3", "hello2.4"}

func mandatoryLiteralStatus(e Evidence) (complete, passed bool) {
	fixtures := map[string]Fixture{}
	for _, f := range e.Fixtures {
		fixtures[f.ID] = f
	}
	seen := map[string]bool{}
	passed = true
	for _, row := range e.Rows {
		if row.Condition != "EC-revise" {
			continue
		}
		f := fixtures[row.FixtureID]
		if f.Intent != "literal" {
			continue
		}
		for _, raw := range MandatoryLiteralRaw {
			if f.ExpectedRaw == raw {
				seen[raw] = true
				if row.Support != "ok" || row.FinalSurfaceError == nil || *row.FinalSurfaceError {
					passed = false
				}
			}
		}
	}
	complete = len(seen) == len(MandatoryLiteralRaw)
	return
}
