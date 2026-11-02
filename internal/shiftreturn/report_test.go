//go:build shiftreturneval

package shiftreturn

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/0c-biggo/0c-ime/core/engine"
)

func toyEvidence(t *testing.T) Evidence {
	t.Helper()
	fs := demoFixtures(t)
	fs = append(append([]Fixture{}, fs[:3]...), fs[4])
	src := emptySources()
	assignment := Assign(fs, src)
	assignment.FixturesSHA256 = strings.Repeat("0", 64)
	lex, sys, english := baselineSources()
	loaded := engine.New(lex, sys)
	loaded.English = english
	e := Evidence{Fixtures: fs, Sources: src, Assignment: assignment, Models: map[string]Model{}, LockSHA256: strings.Repeat("0", 64), ModelSHA256: map[string]string{}, Split: assignment.Components[0].Split}
	byFixture := map[string]Component{}
	for _, c := range assignment.Components {
		for _, id := range c.RowIDs {
			byFixture[id] = c
		}
	}
	for _, name := range []string{"E0", "EL", "EW", "EC"} {
		m := modelForZero()
		m.Name = name
		m.Coefficients[0] = -1
		e.Models[name] = m
	}
	for _, condition := range Conditions {
		parts := strings.Split(condition, "-")
		for _, f := range fs {
			row, snaps, occ, gen, _, err := RunEpisode(f, byFixture[f.ID], condition, e.Models[parts[0]], parts[1], loaded, false)
			if err != nil {
				t.Fatal(err)
			}
			e.Rows = append(e.Rows, row)
			e.Snapshots = append(e.Snapshots, snaps...)
			e.Occurrences = append(e.Occurrences, occ...)
			e.Generators = append(e.Generators, gen...)
		}
	}
	return e
}
func TestMetricsIndependentReducerAllEightConditions(t *testing.T) {
	e := toyEvidence(t)
	report, err := Summarize(e)
	if err != nil {
		t.Fatal(err)
	}
	if report.Outcome != "INCONCLUSIVE" || len(report.Conditions) != 8 || len(report.Paired) != 10 || len(report.Derived) != 6 {
		t.Fatal("toy report masquerades as quality")
	}
	var decoded Report
	if err = Decode(Canonical(report), &decoded); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "evidence.json")
	data := map[string]any{"fixtures": e.Fixtures, "sources": e.Sources, "bank": []BankTask{}, "assignment": e.Assignment, "rows": e.Rows, "snapshots": e.Snapshots, "occurrences": e.Occurrences, "generators": e.Generators, "models": e.Models, "lock": e.Lock, "lock_hash": e.LockSHA256, "model_hashes": e.ModelSHA256, "split": e.Split}
	if err = os.WriteFile(path, Canonical(data), 0644); err != nil {
		t.Fatal(err)
	}
	command := exec.Command("python3", "-c", `import json,sys;from reduce import summarize; x=json.load(open(sys.argv[1]));print(json.dumps(summarize(**x),ensure_ascii=False,allow_nan=False))`, path)
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("independent reducer: %s %v", output, err)
	}
	var independent Report
	if err = json.Unmarshal(output, &independent); err != nil {
		t.Fatal(err)
	}
	// JSON number representation may differ, but values and exact integer joins
	// must agree. Numeric exchange tolerance is fixed at absolute1e-12.
	if err = compareJSON(Canonical(report), Canonical(independent)); err != nil {
		t.Fatal(err)
	}
	diagCommand := exec.Command("python3", "-c", `import json,sys;from reduce import diagnostics;x=json.load(open(sys.argv[1]));print(json.dumps(diagnostics(x['fixtures'],x['rows'],x['generators'],x['occurrences'],x['models'],x['snapshots']),ensure_ascii=False,allow_nan=False))`, path)
	diagOutput, err := diagCommand.CombinedOutput()
	if err != nil {
		t.Fatalf("independent diagnostics: %s %v", diagOutput, err)
	}
	if err = compareJSON(Canonical(diagnostics(e.Rows, e.Assignment, e.Fixtures, e.Generators, e.Occurrences, e.Models, e.Snapshots)), diagOutput); err != nil {
		t.Fatal(err)
	}
}
func compareJSON(a, b []byte) error {
	var x, y any
	if e := json.Unmarshal(a, &x); e != nil {
		return e
	}
	if e := json.Unmarshal(b, &y); e != nil {
		return e
	}
	return compareValue(x, y, "$")
}
func TestMetricsRejectTamperedEvidence(t *testing.T) {
	for _, mutate := range []func(*Evidence){
		func(e *Evidence) { e.Occurrences[0].QueryID = "q:wrong" },
		func(e *Evidence) { e.Generators = e.Generators[1:] },
		func(e *Evidence) { e.Rows[0].PrimaryMetadata = nil },
		func(e *Evidence) {
			for i := range e.Rows {
				if e.Rows[i].FinalBoundaryError != nil {
					v := !*e.Rows[i].FinalBoundaryError
					e.Rows[i].FinalBoundaryError = &v
					break
				}
			}
		},
	} {
		e := toyEvidence(t)
		mutate(&e)
		if _, err := Summarize(e); err == nil {
			t.Fatal("tampered raw evidence accepted")
		}
	}
}
func TestMetricsUnsupportedRetainsDenominator(t *testing.T) {
	e := toyEvidence(t)
	var targetRow int
	var fixture Fixture
	var component Component
	found := false
	for _, c := range e.Assignment.Components {
		if c.PrimaryReturn != nil {
			for _, f := range e.Fixtures {
				if f.ID == *c.PrimaryReturn {
					fixture = f
					component = c
					found = true
					break
				}
			}
			break
		}
	}
	if !found {
		t.Fatal("missing return fixture")
	}
	e.Split = component.Split
	for i, r := range e.Rows {
		if r.FixtureID == fixture.ID && r.Condition == "EC-revise" {
			targetRow = i
			break
		}
	}
	var q string
	for _, s := range e.Snapshots {
		if s.EpisodeID == fixture.ID && s.Condition == "EC-revise" && s.QueryID != nil {
			q = *s.QueryID
		}
	}
	lit := literal(Query{Raw: fixture.ExpectedRaw})
	for i := range e.Generators {
		g := &e.Generators[i]
		if g.QueryID == q {
			g.Status = "resource_exhausted"
			g.Reason = "fixed lex budget exhausted"
			g.LimitHit = "probe_lex_queries"
			g.ProbeLexQueries = ptr(MaxLexQueries)
			g.UncappedIDs = nil
			g.UncappedCount = nil
			g.CapPruned = nil
			g.KeptIDs = []string{}
			g.KeptCount = 0
		}
	}
	kept := []CandidateOccurrence{}
	for _, o := range e.Occurrences {
		if o.QueryID != q {
			kept = append(kept, o)
		} else if o.Candidate.ID == lit.ID {
			o.Score = nil
			kept = append(kept, o)
		}
	}
	e.Occurrences = kept
	episode := []Snapshot{}
	for i := range e.Snapshots {
		s := &e.Snapshots[i]
		if s.EpisodeID == fixture.ID && s.Condition == "EC-revise" {
			if s.QueryID != nil && *s.QueryID == q {
				s.SelectedID = &lit.ID
				s.Display = fixture.ExpectedRaw
				s.View = s.App + s.Prelude + s.Display
			}
			if s.Closed {
				s.App = fixture.ExpectedPrelude + fixture.ExpectedRaw
				s.Commit = s.App
				s.View = s.App
			}
			episode = append(episode, *s)
		}
	}
	row := &e.Rows[targetRow]
	row.Support = "resource_exhausted"
	row.FinalBoundaryError = nil
	row.FinalSurfaceError = nil
	row.FinalEnglishError = nil
	row.GoldSurvived = nil
	row.FirstCorrect, row.LastStable, row.TemporaryErrorLengths = temporal(fixture, episode, e.Occurrences, e.Generators)
	row.RevisionCount, row.ResolutionCount = revisionCounts(fixture, episode, e.Occurrences)
	report, err := Summarize(e)
	if err != nil {
		t.Fatal(err)
	}
	want := 0
	for _, c := range e.Assignment.Components {
		if c.Split == e.Split && c.PrimaryReturn != nil {
			want++
		}
	}
	m := report.Conditions["EC-revise"]["return_boundary"]
	if m.N != want || m.Value != nil || m.Wilson95 != nil || m.Status != "unavailable" || m.UnavailableComponents != 1 || m.Errors != m.ObservedErrors+1 {
		t.Fatalf("unsupported primary shrank/filled denominator: %+v wantn%d", m, want)
	}

	path := filepath.Join(t.TempDir(), "unavailable-evidence.json")
	data := map[string]any{"fixtures": e.Fixtures, "sources": e.Sources, "bank": []BankTask{}, "assignment": e.Assignment, "rows": e.Rows, "snapshots": e.Snapshots, "occurrences": e.Occurrences, "generators": e.Generators, "models": e.Models, "lock": e.Lock, "lock_hash": e.LockSHA256, "model_hashes": e.ModelSHA256, "split": e.Split}
	if err = os.WriteFile(path, Canonical(data), 0644); err != nil {
		t.Fatal(err)
	}
	for _, check := range []struct {
		code     string
		expected any
	}{
		{`from reduce import summarize;print(json.dumps(summarize(**x),ensure_ascii=False,allow_nan=False))`, report},
		{`from reduce import diagnostics;print(json.dumps(diagnostics(x['fixtures'],x['rows'],x['generators'],x['occurrences'],x['models'],x['snapshots']),ensure_ascii=False,allow_nan=False))`, diagnostics(e.Rows, e.Assignment, e.Fixtures, e.Generators, e.Occurrences, e.Models, e.Snapshots)},
	} {
		output, cmdErr := exec.Command("python3", "-c", `import json,sys;x=json.load(open(sys.argv[1]));`+check.code, path).CombinedOutput()
		if cmdErr != nil {
			t.Fatalf("independent unavailable evidence: %s %v", output, cmdErr)
		}
		if err = compareJSON(Canonical(check.expected), output); err != nil {
			t.Fatal(err)
		}
	}
	if report.Gates.G0.Status != "unavailable" || report.Outcome != "INCONCLUSIVE" {
		t.Fatal("unsupported evidence passed")
	}
	p := report.Paired[8]
	if p.AbsoluteErrorReductionCi95 != nil || p.ValidBootstrap != 0 {
		t.Fatal("unsupported evidence bootstrapped surviving subset")
	}
}
func TestFinalizeEpisodeUsesActualPrelude(t *testing.T) {
	lex, sys, english := baselineSources()
	loaded := engine.New(lex, sys)
	loaded.English = english
	f := demoFixtures(t)[0]
	f.PreludeEvents = []Event{}
	for _, key := range "5j/ " {
		f.PreludeEvents = append(f.PreludeEvents, Event{Type: "key", Key: string(key)})
	}
	f.ExpectedPrelude = "中"
	a := Assign([]Fixture{f}, emptySources())
	model := modelForZero()
	model.Coefficients[0] = -1
	row, snapshots, _, _, _, err := RunEpisode(f, a.Components[0], "EC-revise", model, "revise", loaded, false)
	if err != nil {
		t.Fatal(err)
	}
	if snapshots[0].Prelude != "中" || snapshots[len(snapshots)-1].View != "中microsoft窄小" || row.CommitFailures == nil || *row.CommitFailures != 0 {
		t.Fatal("actual prelude/commit not preserved")
	}
	f.ExpectedPrelude = "fabricated"
	if _, _, _, _, _, err = RunEpisode(f, a.Components[0], "EC-revise", model, "revise", loaded, false); err == nil {
		t.Fatal("gold prelude injected instead of actual Session composition")
	}
}
func TestMetricsNoEvidenceSafetyUnknown(t *testing.T) {
	source := emptySources()
	assignment := Assign(nil, source)
	assignment.FixturesSHA256 = strings.Repeat("0", 64)
	r, err := Summarize(Evidence{Sources: source, Assignment: assignment, LockSHA256: strings.Repeat("0", 64)})
	if err != nil {
		t.Fatal(err)
	}
	if r.SafetyStatus != "not_run" || r.Safety.KeyFailures != nil || r.Safety.HypotheticalLearningUpdates != nil || r.Gates.G5.Status != "unavailable" {
		t.Fatal("unrun safety fabricated as zero")
	}
	var decoded Report
	if err = Decode(Canonical(r), &decoded); err != nil {
		t.Fatal(err)
	}
}
func TestWeightsEnumerationNeverScoresOrGetsGoldInQuery(t *testing.T) {
	lex, sys, english := baselineSources()
	loaded := engine.New(lex, sys)
	loaded.English = english
	f := demoFixtures(t)[0]
	a := Assign([]Fixture{f}, emptySources())
	_, snapshots, occurrences, _, observations, err := RunEpisode(f, a.Components[0], "shared", modelForZero(), "revise", loaded, true)
	if err != nil {
		t.Fatal(err)
	}
	for _, s := range snapshots {
		if s.ScoreCalls == nil || *s.ScoreCalls != 0 {
			t.Fatal("enumeration executed linear scores")
		}
	}
	for _, o := range occurrences {
		if o.Score != nil {
			t.Fatal("enumeration fabricated score")
		}
	}
	if len(observations) == 0 {
		t.Fatal("external checkpoint labels missing")
	}
}
func TestWeightsCheckpointRequiresCompleteSpelling(t *testing.T) {
	lex, sys, english := baselineSources()
	loaded := engine.New(lex, sys)
	loaded.English = english
	f := demoFixtures(t)[4]
	f.ExpectedRaw = "3x"
	f.Events = []Event{{Type: "shift"}, {Type: "key", Key: "3"}, {Type: "key", Key: "x"}, {Type: "enter"}}
	f.Gold = []Gold{{Segments: []Segment{{Kind: "literal", Begin: 0, End: 2}}, AcceptedSurfaces: []string{"3x"}}}
	a := Assign([]Fixture{f}, emptySources())
	_, _, _, _, observations, err := RunEpisode(f, a.Components[0], "shared", modelForZero(), "revise", loaded, true)
	if err != nil {
		t.Fatal(err)
	}
	if len(observations) != 1 || observations[0].EventIndex != 2 {
		t.Fatal("bare tone incorrectly became a complete-syllable checkpoint")
	}
}
func TestWeightsGoldProjectionIndependent(t *testing.T) {
	values := []map[string]any{}
	for _, f := range demoFixtures(t) {
		for offset := 1; offset <= len(f.ExpectedRaw); offset++ {
			values = append(values, map[string]any{"fixture": f, "offset": offset, "gold_ids": projectedGoldIDs(f, offset)})
		}
	}
	path := filepath.Join(t.TempDir(), "gold.json")
	if err := os.WriteFile(path, Canonical(values), 0644); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("python3", "-c", `import json,sys;from reduce import projected_gold_ids; x=json.load(open(sys.argv[1]));assert all(sorted(projected_gold_ids(v['fixture'],v['offset']))==v['gold_ids'] for v in x)`, path)
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("independent prefix gold projection: %s %v", output, err)
	}
}

func TestTemporalIndicesUseResearchKeyOrdinal(t *testing.T) {
	p := tinyPolicy(true, "revise")
	s := startFragment(t, p)
	f := Fixture{Intent: "return", Gold: []Gold{{Segments: []Segment{{Kind: "literal", Begin: 0, End: 1}, {Kind: "chinese", Begin: 1, End: 4, Reading: "ㄓㄞˇ"}}}}}
	var snapshots []Snapshot
	var occurrences []CandidateOccurrence
	var generators []GeneratorDiagnostic
	f.Events = append(f.Events, Event{Type: "shift"})
	for _, key := range "x593" {
		event := Event{Type: "key", Key: string(key)}
		f.Events = append(f.Events, event)
		var snap Snapshot
		var err error
		s, snap, err = Step(s, event, p)
		if err != nil {
			t.Fatal(err)
		}
		snapshots = append(snapshots, snap)
		r := s.LastResult
		generators = append(generators, GeneratorDiagnostic{QueryID: *snap.QueryID, Raw: s.Raw, Status: r.Status})
		for _, h := range r.Candidates {
			occurrences = append(occurrences, CandidateOccurrence{QueryID: *snap.QueryID, Candidate: h})
		}
	}
	first, last, _ := temporal(f, snapshots, occurrences, generators)
	if first == nil || last == nil || *first != 3 || *last != 3 || snapshots[3].EventIndex != 4 || snapshots[3].KeyOffset != 4 {
		t.Fatalf("first=%v last=%v snapshot=%+v", first, last, snapshots[3])
	}
}
