//go:build shiftreturneval

package shiftreturn

import (
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/0c-biggo/0c-ime/core/engine"
	"github.com/0c-biggo/0c-ime/internal/datadir"
)

type RunOptions struct {
	Command, Fixtures, Sources, Bank, Assignment, Lock, Out, Data, Model, Flow, Split string
	Args                                                                              []string
}
type budgetFile struct {
	file   *os.File
	budget *EvidenceBudget
}

func (w budgetFile) Write(b []byte) (int, error) {
	if err := w.budget.Add(int64(len(b))); err != nil {
		return 0, err
	}
	return w.file.Write(b)
}

type artifactWriter struct {
	file   *os.File
	zip    *gzip.Writer
	writer io.Writer
}

func newArtifact(path string, budget *EvidenceBudget, compressed bool) (*artifactWriter, error) {
	f, e := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0644)
	if e != nil {
		return nil, e
	}
	counted := budgetFile{f, budget}
	w := &artifactWriter{file: f, writer: counted}
	if compressed {
		w.zip = gzip.NewWriter(counted)
		w.writer = w.zip
	}
	return w, nil
}
func (w *artifactWriter) Write(v any) error {
	b := append(Canonical(v), '\n')
	_, e := w.writer.Write(b)
	return e
}
func (w *artifactWriter) Close() error {
	if w.zip != nil {
		if e := w.zip.Close(); e != nil {
			w.file.Close()
			return e
		}
	}
	return w.file.Close()
}
func projectedGoldIDs(f Fixture, offset int) []string {
	ids := []string{}
	for _, gold := range f.Gold {
		_, firstEnd := firstChinese(gold)
		segs := []Segment{}
		if f.Intent == "literal" || firstEnd < 0 || offset < firstEnd {
			if offset > 0 {
				segs = append(segs, Segment{"literal", 0, offset, ""})
			}
		} else {
			for _, seg := range gold.Segments {
				if seg.Begin >= offset {
					break
				}
				if seg.End <= offset {
					segs = append(segs, seg)
				} else {
					kind := seg.Kind
					reading := seg.Reading
					if kind == "chinese" {
						kind = "pending"
						reading = ""
					}
					segs = append(segs, Segment{kind, seg.Begin, offset, reading})
					break
				}
			}
		}
		ids = append(ids, "h:"+HashBytes([]byte(identity(segs))))
	}
	return sortedUnique(ids)
}
func diagnostic(f Fixture, c Component, condition string, event int, s Snapshot, r Result) GeneratorDiagnostic {
	return GeneratorDiagnostic{Version: Version, Kind: "generator_diagnostic", Demo: f.Demo, FixtureID: f.ID, ComponentID: c.ID, Condition: condition, EventIndex: event, KeyOffset: s.KeyOffset, Raw: "", QueryID: *s.QueryID, GenerationPerformed: r.GenerationPerformed, UncappedIDs: r.UncappedIDs, KeptIDs: r.KeptIDs, UncappedCount: r.UncappedCount, KeptCount: r.KeptCount, LiteralID: r.LiteralID, CapPruned: r.CapPruned, RawOverflow: r.RawOverflow, Status: r.Status, Reason: r.Reason, LimitHit: r.LimitHit, GenerateCalls: ptr(r.GenerateCalls), ProbeLexQueries: ptr(r.ProbeLexQueries), BestCalls: ptr(r.BestCalls)}
}
func modelForZero() Model {
	m := Model{Coefficients: make([]float64, 13), Means: make([]float64, 13), Stds: make([]float64, 13)}
	for i := range m.Stds {
		m.Stds[i] = 1
	}
	return m
}
func ValidateModel(m Model) error {
	if e := Decode(Canonical(m), new(Model)); e != nil {
		return e
	}
	mask := map[string][]int{"E0": {0, 1, 2, 3, 4, 5, 6}, "EL": {0, 1, 2, 3, 4, 5, 6, 7, 8}, "EW": {0, 1, 2, 3, 4, 5, 6, 9, 10, 11, 12}, "EC": {0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12}}
	if string(Canonical(m.Active)) != string(Canonical(mask[m.Name])) {
		return fmt.Errorf("model active mask differs")
	}
	active := map[int]bool{}
	for _, j := range m.Active {
		active[j] = true
	}
	for j, w := range m.Coefficients {
		if j != 5 && j != 7 && j != 8 && j != 9 && j != 10 && j != 11 {
			if m.Means[j] != 0 || m.Stds[j] != 1 || m.Counts[j] != 0 {
				return fmt.Errorf("flag normalization differs")
			}
		}
		if !active[j] && w != 0 {
			return fmt.Errorf("inactive coefficient nonzero")
		}
		if m.Stds[j] < 0 {
			return fmt.Errorf("negative population std")
		}
	}
	return nil
}

// RunEpisode keeps gold outside Step. It grades only after snapshots/queries are
// returned, and builds training labels in a distinct external pass.
func RunEpisode(f Fixture, c Component, condition string, model Model, flow string, loaded *engine.Engine, enumerate bool) (Row, []Snapshot, []CandidateOccurrence, []GeneratorDiagnostic, []TrainingObservation, error) {
	probe, scoring := NewScoringProbe(loaded.Model.Lex, loaded.Model.LM, loaded.English)
	before := StoreDigest(scoring)
	p := Policy{Enumerate: enumerate, Probe: probe, Model: model, Flow: flow, Enabled: true, Standard: true, Condition: condition}
	preludeSession, err := NewBaseline(loaded.Model.Lex, loaded.Model.LM, loaded.English, "M", nil, true)
	if err != nil {
		return Row{}, nil, nil, nil, nil, err
	}
	for _, event := range f.PreludeEvents {
		request, err := baselineRequest(event)
		if err != nil {
			return Row{}, nil, nil, nil, nil, err
		}
		preludeSession.Handle(request)
	}
	actualPrelude := preludeSession.Session.View().Text()
	if actualPrelude != f.ExpectedPrelude {
		return Row{}, nil, nil, nil, nil, fmt.Errorf("fixture prelude differs from actual original composition: %s", f.ID)
	}
	s := State{FragmentID: f.ID, App: preludeSession.Ledger.App, Prelude: actualPrelude, Mode: "zhuyin"}
	snapshots := []Snapshot{}
	occurrences := []CandidateOccurrence{}
	generators := []GeneratorDiagnostic{}
	observations := []TrainingObservation{}
	raw := ""
	support := "ok"
	var finalSnapshot Snapshot
	var finalCandidate *Hypothesis
	var finalResult *Result
	lastKey := -1
	for i, e := range f.Events {
		if e.Type == "key" {
			lastKey = i
		}
	}
	for i, event := range f.Events {
		if event.Type == "key" {
			raw += event.Key
		} else if event.Type == "backspace" && len(raw) > 0 {
			raw = raw[:len(raw)-1]
		}
		beforeFreeze := append([]Segment{}, s.Freeze...)
		next, snap, e := Step(s, event, p)
		if e != nil {
			return Row{}, snapshots, occurrences, generators, observations, e
		}
		s = next
		snap.Demo = f.Demo
		snapshots = append(snapshots, snap)
		if s.LastResult != nil {
			r := *s.LastResult
			g := diagnostic(f, c, condition, i, snap, r)
			g.Raw = raw
			generators = append(generators, g)
			if r.Status != "ok" {
				support = r.Status
			}
			for _, h := range r.Candidates {
				allowed := compatible(beforeFreeze, h.DisplaySegments)
				reason := "none"
				if !allowed {
					reason = "freeze"
				}
				var value *float64
				if !enumerate && r.Status == "ok" && (allowed || len(h.Readings) == 0) && !s.LiteralLock && len(r.Candidates) > 1 {
					score, e := score(model, h)
					if e != nil {
						return Row{}, snapshots, occurrences, generators, observations, e
					}
					value = &score
				}
				occurrences = append(occurrences, CandidateOccurrence{Version: Version, Kind: "candidate_occurrence", Demo: f.Demo, FixtureID: f.ID, ComponentID: c.ID, Condition: condition, EventIndex: i, KeyOffset: snap.KeyOffset, Raw: raw, QueryID: *snap.QueryID, Candidate: h, Score: value, Allowed: allowed, ExclusionReason: reason})
				if enumerate && event.Type == "key" && (completeSpellingAtEnd(raw) || i == lastKey) {
					observations = append(observations, TrainingObservation{Version: Version, Kind: "training_observation", Demo: f.Demo, FixtureID: f.ID, ComponentID: c.ID, EventIndex: i, KeyOffset: len(raw), Raw: raw, Candidate: h, GoldIDs: projectedGoldIDs(f, len(raw))})
				}
			}
		}
		if i == lastKey {
			finalSnapshot = snap
			finalResult = s.LastResult
			if s.Selection != nil {
				h := *s.Selection
				finalCandidate = &h
			}
		}
	}
	row := Row{Version: Version, Kind: "row", Demo: f.Demo, FixtureID: f.ID, ComponentID: c.ID, Condition: condition, Intent: f.Intent, Support: support, TemporaryErrorLengths: []int{}, RevisionCount: s.RevisionCount, ResolutionCount: s.ResolutionCount, SafetyStatus: "complete", SnapshotsPath: "events.jsonl.gz", CandidatesPath: "candidates.jsonl.gz", GeneratorsPath: "generators.jsonl.gz", KeyFailures: ptr(0), CommitFailures: ptr(0), AppRewrites: ptr(0), LockViolations: ptr(0), HypotheticalLearningUpdates: ptr(0), PersistentWrites: ptr(0), ProtectedCalls: ptr(0), ContextClass: "none"}
	if f.Intent == "return" {
		row.Primary = c.PrimaryReturn != nil && *c.PrimaryReturn == f.ID
		row.PrimaryMetadata = c.PrimaryReturnMetadata
	} else if f.Intent == "literal" {
		row.Primary = c.PrimaryLiteral != nil && *c.PrimaryLiteral == f.ID
		row.PrimaryMetadata = c.PrimaryLiteralMetadata
	}
	if row.PrimaryMetadata != nil {
		row.PrimaryCategory = row.PrimaryMetadata.Category
		row.ContextClass = row.PrimaryMetadata.ContextClass
		row.ContrastEligible = row.PrimaryMetadata.ContrastEligible
	}
	if StoreDigest(scoring) != before {
		*row.HypotheticalLearningUpdates = 1
	}
	if len(snapshots) > 0 && snapshots[len(snapshots)-1].View != finalSnapshot.View {
		*row.CommitFailures = 1
	}
	if support == "ok" && finalResult != nil {
		if finalCandidate == nil {
			h := literal(Query{Raw: f.ExpectedRaw})
			h.Display = finalSnapshot.Display
			h.DisplaySegments = append([]Segment{}, finalSnapshot.Freeze...)
			at := prevEnd(h.DisplaySegments)
			if at < len(f.ExpectedRaw) {
				h.DisplaySegments = append(h.DisplaySegments, Segment{"literal", at, len(f.ExpectedRaw), ""})
			}
			finalCandidate = &h
		}
		boundary, surface, english := grade(f, *finalCandidate)
		row.FinalBoundaryError = &boundary
		row.FinalSurfaceError = &surface
		row.FinalEnglishError = &english
		survived := false
		goldIDs := projectedGoldIDs(f, len(f.ExpectedRaw))
		for _, id := range append(append([]string{}, finalResult.KeptIDs...), finalResult.LiteralID) {
			for _, gold := range goldIDs {
				survived = survived || id == gold
			}
		}
		row.GoldSurvived = &survived
	}
	row.FirstCorrect, row.LastStable, row.TemporaryErrorLengths = temporal(f, snapshots, occurrences, generators)
	return row, snapshots, occurrences, generators, observations, nil
}
func ReadJSONL[T any](path string) ([]T, error) {
	var out []T
	var reader io.ReadCloser
	f, e := os.Open(path)
	if e != nil {
		return nil, e
	}
	defer f.Close()
	reader = f
	if strings.HasSuffix(path, ".gz") {
		z, e := gzip.NewReader(f)
		if e != nil {
			return nil, e
		}
		defer z.Close()
		reader = z
	}
	d := json.NewDecoder(reader)
	for {
		var raw json.RawMessage
		if e = d.Decode(&raw); e == io.EOF {
			break
		} else if e != nil {
			return nil, e
		}
		var value T
		if e = Decode(raw, &value); e != nil {
			return nil, e
		}
		out = append(out, value)
	}
	return out, nil
}

func evidenceLockModels(lock Lock) (map[string]Model, map[string]string, error) {
	manifests := []string{}
	for path := range lock.Files {
		if filepath.Base(path) == "selected-model-manifest.json" {
			manifests = append(manifests, path)
		}
	}
	if len(manifests) != 1 {
		return nil, nil, fmt.Errorf("execution/evidence lock must contain exactly one selected-model-manifest.json")
	}
	path := manifests[0]
	bytes, err := os.ReadFile(path)
	if err != nil {
		return nil, nil, err
	}
	if HashBytes(bytes) != lock.Files[path] {
		return nil, nil, fmt.Errorf("selected manifest bytes changed")
	}
	var manifest SelectedModelManifest
	if err = Decode(bytes, &manifest); err != nil {
		return nil, nil, err
	}
	if manifest.Demo && !lock.Demo {
		return nil, nil, fmt.Errorf("formal selected manifest cannot be demo")
	}
	frozen := map[string]string{}
	for path, hash := range lock.Files {
		absolute, err := filepath.Abs(path)
		if err != nil {
			return nil, nil, err
		}
		frozen[absolute] = hash
	}
	models := map[string]Model{}
	hashes := map[string]string{}
	for _, name := range []string{"E0", "EL", "EW", "EC"} {
		entry := manifest.Models[name]
		if !filepath.IsAbs(entry.Path) || frozen[filepath.Clean(entry.Path)] != entry.SHA256 {
			return nil, nil, fmt.Errorf("selected model path must be absolute and frozen")
		}
		bytes, err := os.ReadFile(entry.Path)
		if err != nil {
			return nil, nil, err
		}
		if HashBytes(bytes) != entry.SHA256 {
			return nil, nil, fmt.Errorf("selected model bytes changed")
		}
		var m Model
		if err = Decode(bytes, &m); err != nil {
			return nil, nil, err
		}
		if err = ValidateModel(m); err != nil {
			return nil, nil, err
		}
		if m.Name != name || m.Demo != manifest.Demo || m.CodeLockSHA256 != manifest.SourceLockSHA256 || m.FitLockSHA256 != manifest.FitLockSHA256 {
			return nil, nil, fmt.Errorf("selected model manifest provenance differs")
		}
		models[name] = m
		hashes[name] = entry.SHA256
	}
	return models, hashes, nil
}
func modelProvenance(m Model, lock Lock) error {
	code, fit := false, false
	for path, hash := range lock.Files {
		if hash != m.CodeLockSHA256 && hash != m.FitLockSHA256 {
			continue
		}
		var ancestor Lock
		if err := ReadRecord(path, &ancestor); err != nil {
			return err
		}
		if err := VerifyLock(path); err != nil {
			return err
		}
		if hash == m.CodeLockSHA256 {
			code = true
		}
		if hash == m.FitLockSHA256 {
			fit = true
		}
	}
	if !code || !fit {
		return fmt.Errorf("model source/fit lock ancestry missing from execution lock")
	}
	return nil
}
func RequireDataLocked(lockPath, data string) error {
	inputs := []string{filepath.Join(data, "manifest.json"), filepath.Join(data, "english", "manifest.json")}
	for _, manifest := range append([]string{}, inputs...) {
		b, e := os.ReadFile(manifest)
		if e != nil {
			return e
		}
		var m struct {
			Files map[string]any `json:"files"`
		}
		if e = json.Unmarshal(b, &m); e != nil {
			return e
		}
		for name := range m.Files {
			inputs = append(inputs, filepath.Join(filepath.Dir(manifest), name))
		}
	}
	return RequireLocked(lockPath, inputs...)
}
func checkedInputs(o RunOptions) ([]Fixture, Sources, []BankTask, Assignment, Lock, *engine.Engine, error) {
	fs, hash, e := ReadFixtures(o.Fixtures)
	if e != nil {
		return nil, Sources{}, nil, Assignment{}, Lock{}, nil, e
	}
	var src Sources
	if e = ReadRecord(o.Sources, &src); e != nil {
		return nil, src, nil, Assignment{}, Lock{}, nil, e
	}
	bank, e := ReadBank(o.Bank)
	if e != nil {
		return nil, src, nil, Assignment{}, Lock{}, nil, e
	}
	var assignment Assignment
	if e = ReadRecord(o.Assignment, &assignment); e != nil {
		return nil, src, bank, assignment, Lock{}, nil, e
	}
	if e = ValidateAssignment(fs, src, assignment, hash); e != nil {
		return nil, src, bank, assignment, Lock{}, nil, e
	}
	var lock Lock
	if o.Command != "validate" {
		if e = VerifyLock(o.Lock); e != nil {
			return nil, src, bank, assignment, lock, nil, e
		}
		if e = RequireLocked(o.Lock, o.Fixtures, o.Sources, o.Bank, o.Assignment); e != nil {
			return nil, src, bank, assignment, lock, nil, e
		}
		sourcePaths := []string{}
		for _, entry := range src.Entries {
			sourcePaths = append(sourcePaths, entry.CachedPath)
			if entry.AdjudicationLog != "" {
				sourcePaths = append(sourcePaths, entry.AdjudicationLog)
			}
		}
		if e = RequireLocked(o.Lock, sourcePaths...); e != nil {
			return nil, src, bank, assignment, lock, nil, e
		}
		if e = RequireDataLocked(o.Lock, o.Data); e != nil {
			return nil, src, bank, assignment, lock, nil, e
		}
		if e = ReadRecord(o.Lock, &lock); e != nil {
			return nil, src, bank, assignment, lock, nil, e
		}
		if lock.Demo {
			return nil, src, bank, assignment, lock, nil, fmt.Errorf("formal execution rejects demo lock")
		}
	}
	loaded, e := datadir.Open(o.Data)
	if e != nil {
		return nil, src, bank, assignment, lock, nil, e
	}
	if loaded.EnglishPack == "" {
		return nil, src, bank, assignment, lock, nil, fmt.Errorf("prototype English fallback forbidden")
	}
	if e = ValidateBank(fs, src, bank, loaded.Model.Lex, loaded.English, true); e != nil {
		return nil, src, bank, assignment, lock, nil, e
	}
	if ok, reason := sampleQualification(fs, bank, assignment); !ok {
		return nil, src, bank, assignment, lock, nil, &InsufficientError{reason}
	}
	testReturn, testLiteral, testEnglish, contrast := 0, 0, 0, 0
	byID := map[string]Fixture{}
	for _, f := range fs {
		byID[f.ID] = f
	}
	for _, c := range assignment.Components {
		if c.Split != "test" {
			continue
		}
		if c.PrimaryReturn != nil {
			testReturn++
			f := byID[*c.PrimaryReturn]
			begin, _ := firstChinese(f.Gold[0])
			if begin > 0 {
				testEnglish++
			}
			if c.PrimaryReturnMetadata.ContrastEligible {
				contrast++
			}
		}
		if c.PrimaryLiteral != nil {
			testLiteral++
		}
	}
	if testReturn < 400 || testLiteral < 400 || testEnglish < 400 || contrast < 100 {
		return nil, src, bank, assignment, lock, nil, &InsufficientError{"test independent component/cohort qualification insufficient"}
	}
	return fs, src, bank, assignment, lock, loaded, nil
}

// Execute performs only the caller's frozen phase. Formal commands require an
// exact command entry in an immutable execution lock and create-only output.
func Execute(o RunOptions) (int, error) {
	fs, src, bank, assignment, lock, loaded, e := checkedInputs(o)
	if e != nil {
		return 2, e
	}
	if o.Command == "validate" {
		if e = NewOutput(o.Out); e != nil {
			return 2, e
		}
		e = WriteRecord(filepath.Join(o.Out, "validation.json"), map[string]any{"version": Version, "fixtures": len(fs), "assignment_valid": true, "source_complete": true, "quality": "not_run"})
		return 0, e
	}
	if o.Command == "verify" {
		return verifyCalibration(o, lock, loaded)
	}
	approved := false
	invocation := strings.Join(o.Args, " ")
	for _, command := range lock.Commands {
		if command == invocation || strings.HasSuffix(command, "./cmd/0c-shift-eval "+invocation) {
			approved = true
		}
	}
	if !approved {
		return 2, fmt.Errorf("exact phase invocation missing from frozen lock commands")
	}
	if e = ValidateOutputRoot(o.Out); e != nil {
		return 2, e
	}
	priorBytes, err := ExistingEvidenceBytes(ExperimentRoot)
	if err != nil {
		return 2, err
	}
	if priorBytes > MaxArtifactBytes {
		return 3, fmt.Errorf("retained experiment evidence already exceeds8GiB")
	}
	if e = PreflightSpace(filepath.Dir(o.Out)); e != nil {
		return 3, e
	}
	if e = NewOutput(o.Out); e != nil {
		return 2, e
	}
	model := modelForZero()
	condition := "shared"
	flow := "revise"
	models := map[string]Model{}
	hashes := map[string]string{}
	if o.Command == "replay" {
		if o.Split != "train" && o.Split != "dev" && o.Split != "test" {
			return 2, fmt.Errorf("split must be train/dev/test")
		}
		if o.Flow != "fixed" && o.Flow != "revise" {
			return 2, fmt.Errorf("flow must be fixed/revise")
		}
		if e = ReadRecord(o.Model, &model); e != nil {
			return 2, e
		}
		if e = ValidateModel(model); e != nil {
			return 2, e
		}
		if e = modelProvenance(model, lock); e != nil {
			return 2, e
		}
		if e = RequireLocked(o.Lock, o.Model); e != nil {
			return 2, e
		}
		flow = o.Flow
		condition = model.Name + "-" + flow
		models[model.Name] = model
		b, e := os.ReadFile(o.Model)
		if e != nil {
			return 2, e
		}
		hashes[model.Name] = HashBytes(b)
	}
	budget := &EvidenceBudget{Bytes: priorBytes}
	writers := map[string]*artifactWriter{}
	for _, name := range []string{"rows.jsonl", "events.jsonl.gz", "candidates.jsonl.gz", "generators.jsonl.gz"} {
		w, e := newArtifact(filepath.Join(o.Out, name), budget, strings.HasSuffix(name, ".gz"))
		if e != nil {
			return 2, e
		}
		writers[name] = w
	}
	defer func() {
		for _, w := range writers {
			_ = w.file.Close()
		}
	}()
	var baselineWriter *artifactWriter
	if o.Command == "enumerate" {
		var err error
		baselineWriter, err = newArtifact(filepath.Join(o.Out, "baselines.jsonl.gz"), budget, true)
		if err != nil {
			return 2, err
		}
		defer baselineWriter.file.Close()
	}
	training := map[string]*artifactWriter{}
	if o.Command == "enumerate" {
		for _, split := range []string{"train", "dev", "test"} {
			w, e := newArtifact(filepath.Join(o.Out, split+".jsonl"), budget, false)
			if e != nil {
				return 2, e
			}
			training[split] = w
			defer w.file.Close()
		}
	}
	byFixture := map[string]Component{}
	for _, c := range assignment.Components {
		for _, id := range c.RowIDs {
			byFixture[id] = c
		}
	}
	evidence := Evidence{Fixtures: fs, Sources: src, Bank: bank, Assignment: assignment, Models: models, Lock: lock, ModelSHA256: hashes, Split: o.Split}
	lockBytes, e := os.ReadFile(o.Lock)
	if e != nil {
		return 2, e
	}
	evidence.LockSHA256 = HashBytes(lockBytes)
	incomplete := false
	for _, f := range fs {
		c := byFixture[f.ID]
		if o.Command == "replay" && c.Split != o.Split && c.Split != "regression" {
			continue
		}
		if baselineWriter != nil && f.Intent != "compat" {
			for _, control := range []string{"M", "R", "C", "A"} {
				baseline, err := NewBaseline(loaded.Model.Lex, loaded.Model.LM, loaded.English, control, nil, true)
				if err != nil {
					return 2, err
				}
				trace, err := ReplayBaseline(baseline, f, control)
				if err != nil {
					return 2, err
				}
				correct := false
				toggles := 0
				for _, observation := range trace {
					if observation.Request.Type == "toggle" {
						toggles++
					}
				}
				if len(trace) > 0 {
					view := trace[len(trace)-1].View
					for _, gold := range f.Gold {
						for _, accepted := range gold.AcceptedSurfaces {
							correct = correct || view == f.ExpectedPrelude+accepted
						}
					}
				}
				if err = baselineWriter.Write(map[string]any{"fixture_id": f.ID, "component_id": c.ID, "condition": control, "observations": trace, "manual_toggles": toggles, "final_surface_error": !correct}); err != nil {
					return 3, err
				}
			}
		}
		row, snaps, occ, gen, obs, err := RunEpisode(f, c, condition, model, flow, loaded, o.Command == "enumerate")
		if err != nil {
			return 2, err
		}
		if row.Support != "ok" {
			incomplete = true
		}
		if e = writers["rows.jsonl"].Write(row); e != nil {
			return 3, e
		}
		for _, v := range snaps {
			if e = writers["events.jsonl.gz"].Write(v); e != nil {
				return 3, e
			}
		}
		for _, v := range occ {
			if e = writers["candidates.jsonl.gz"].Write(v); e != nil {
				return 3, e
			}
		}
		for _, v := range gen {
			if e = writers["generators.jsonl.gz"].Write(v); e != nil {
				return 3, e
			}
		}
		if w := training[c.Split]; w != nil {
			for _, v := range obs {
				if e = w.Write(v); e != nil {
					return 3, e
				}
			}
		}
		evidence.Rows = append(evidence.Rows, row)
		evidence.Snapshots = append(evidence.Snapshots, snaps...)
		evidence.Occurrences = append(evidence.Occurrences, occ...)
		evidence.Generators = append(evidence.Generators, gen...)
	}
	if baselineWriter != nil {
		if e = baselineWriter.Close(); e != nil {
			return 3, e
		}
	}
	for _, w := range writers {
		if e = w.Close(); e != nil {
			return 3, e
		}
	}
	for _, w := range training {
		if e = w.Close(); e != nil {
			return 3, e
		}
	}
	report, e := Summarize(evidence)
	if e != nil {
		return 2, e
	}
	if e = WriteRecord(filepath.Join(o.Out, "report.json"), report); e != nil {
		return 2, e
	}
	if e = WriteRecord(filepath.Join(o.Out, "diagnostics.json"), diagnostics(evidence.Rows, assignment, fs, evidence.Generators, evidence.Occurrences, models, evidence.Snapshots)); e != nil {
		return 2, e
	}
	assignmentInputBytes, err := os.ReadFile(o.Assignment)
	if err != nil {
		return 2, err
	}
	metadata := map[string]any{"split": o.Split, "flow": flow, "condition": condition, "command": o.Command, "lock_sha256": evidence.LockSHA256, "support_complete": !incomplete, "fixtures_sha256": assignment.FixturesSHA256, "assignment_sha256": HashBytes(assignmentInputBytes), "raw_evidence_bytes": budget.Bytes}
	if o.Command == "replay" {
		metadata["model_sha256"] = hashes[model.Name]
		metadata["margin"] = model.Margin
	}
	if e = WriteRecord(filepath.Join(o.Out, "run.json"), metadata); e != nil {
		return 2, e
	}
	if incomplete {
		return 3, nil
	}
	if o.Command == "enumerate" {
		if report.Gates.G1.Status == "pass" {
			return 0, nil
		}
		if report.Gates.G1.Status == "fail" {
			return 4, nil
		}
		return 3, nil
	}
	return 3, nil
}
func SummarizeFiles(o RunOptions, rowsPath, eventsPath, candidatesPath, generatorsPath string) (int, error) {
	fs, src, bank, assignment, lock, _, err := checkedInputs(o)
	if err != nil {
		return 2, err
	}
	if err = RequireLocked(o.Lock, rowsPath, eventsPath, candidatesPath, generatorsPath); err != nil {
		return 2, err
	}
	models, hashes, err := evidenceLockModels(lock)
	if err != nil {
		return 2, err
	}
	for _, m := range models {
		if err = modelProvenance(m, lock); err != nil {
			return 2, err
		}
	}
	rows, err := ReadJSONL[Row](rowsPath)
	if err != nil {
		return 2, err
	}
	snapshots, err := ReadJSONL[Snapshot](eventsPath)
	if err != nil {
		return 2, err
	}
	occurrences, err := ReadJSONL[CandidateOccurrence](candidatesPath)
	if err != nil {
		return 2, err
	}
	generators, err := ReadJSONL[GeneratorDiagnostic](generatorsPath)
	if err != nil {
		return 2, err
	}
	lockBytes, err := os.ReadFile(o.Lock)
	if err != nil {
		return 2, err
	}
	report, err := Summarize(Evidence{Fixtures: fs, Sources: src, Bank: bank, Assignment: assignment, Rows: rows, Snapshots: snapshots, Occurrences: occurrences, Generators: generators, Models: models, Lock: lock, LockSHA256: HashBytes(lockBytes), ModelSHA256: hashes})
	if err != nil {
		return 2, err
	}
	if err = NewOutput(o.Out); err != nil {
		return 2, err
	}
	if err = WriteRecord(filepath.Join(o.Out, "report.json"), report); err != nil {
		return 2, err
	}
	if err = WriteRecord(filepath.Join(o.Out, "diagnostics.json"), diagnostics(rows, assignment, fs, generators, occurrences, models, snapshots)); err != nil {
		return 2, err
	}
	if report.Outcome == "GO_OUTPUT_EXPERIMENT" {
		return 0, nil
	}
	if report.Outcome == "NO_GO_METHOD" {
		return 4, nil
	}
	return 3, nil
}
func diagnostics(rows []Row, a Assignment, fixtures []Fixture, generators []GeneratorDiagnostic, occurrences []CandidateOccurrence, models map[string]Model, snapshots []Snapshot) map[string]any {
	categories := map[string]map[string]map[string][]*bool{}
	variants := map[string]map[string]bool{}
	for _, r := range rows {
		if categories[r.Condition] == nil {
			categories[r.Condition] = map[string]map[string][]*bool{}
		}
		if categories[r.Condition][r.PrimaryCategory] == nil {
			categories[r.Condition][r.PrimaryCategory] = map[string][]*bool{}
		}
		if variants[r.Condition] == nil {
			variants[r.Condition] = map[string]bool{}
		}
		for _, v := range []struct {
			name  string
			value *bool
		}{{"boundary", r.FinalBoundaryError}, {"surface", r.FinalSurfaceError}, {"english", r.FinalEnglishError}} {
			metricName := ""
			if r.Intent == "return" {
				metricName = "return_" + v.name
			} else if r.Intent == "literal" && v.name == "surface" {
				metricName = "literal_only"
			}
			if metricName == "" {
				continue
			}
			if r.Primary {
				categories[r.Condition][r.PrimaryCategory][metricName] = append(categories[r.Condition][r.PrimaryCategory][metricName], v.value)
			}
			if v.value != nil {
				variants[r.Condition][r.ComponentID+":"+metricName] = variants[r.Condition][r.ComponentID+":"+metricName] || *v.value
			}

		}
	}
	metrics := map[string]map[string]map[string]Metric{}
	for condition, cats := range categories {
		metrics[condition] = map[string]map[string]Metric{}
		for category, names := range cats {
			metrics[condition][category] = map[string]Metric{}
			for name, values := range names {
				observed, unavailable := 0, 0
				for _, value := range values {
					if value == nil {
						unavailable++
					} else if *value {
						observed++
					}
				}
				m := metric(observed+unavailable, len(values))
				m.ObservedErrors = observed
				m.UnavailableComponents = unavailable
				if unavailable > 0 {
					m.Status = "unavailable"
					m.Value = nil
					m.Wilson95 = nil
				}
				metrics[condition][category][name] = m

			}
		}
	}
	queryRecall := map[string]string{}
	byID := map[string]Fixture{}
	for _, f := range fixtures {
		byID[f.ID] = f
	}
	for _, g := range generators {
		status := "unavailable"
		if g.Status == "ok" {
			gold := projectedGoldIDs(byID[g.FixtureID], len(g.Raw))
			kept, uncapped := false, false
			for _, id := range gold {
				if id == g.LiteralID {
					kept = true
					uncapped = true
				}
				for _, candidate := range g.KeptIDs {
					kept = kept || id == candidate
				}
				for _, candidate := range g.UncappedIDs {
					uncapped = uncapped || id == candidate
				}
			}
			status = "generator_miss"
			if uncapped {
				status = "cap_loss"
			}
			if kept {
				status = "survived"
			}
		}
		queryRecall[g.QueryID] = status
	}
	result := map[string]any{"categories": metrics, "component_any_variant": variants, "recall_by_query": queryRecall, "homophone_surface_changes": surfaceChanges(occurrences, snapshots), "unavailable_primary_metrics": "full component n retained; unavailable rows charged as failures in errors, value/CI null; paired/derived CI unavailable"}
	if model, ok := models["EC"]; ok {
		rows, err := FirstWordSensitivity(fixtures, occurrences, model)
		if err != nil {
			result["ec_first_W_sensitivity_error"] = err.Error()
		} else {
			result["ec_first_W_sensitivity"] = map[string]any{"coefficients": "same frozen EC", "retrained": false, "main_arm": false, "rows": rows}
		}
	}
	return result
}
