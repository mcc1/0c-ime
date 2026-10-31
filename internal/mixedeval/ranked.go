//go:build mixedeval

package mixedeval

import (
	"bufio"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/0c-biggo/0c-ime/core/engine"
	"github.com/0c-biggo/0c-ime/core/mixedparse"
	"github.com/0c-biggo/0c-ime/core/personal"
	"github.com/0c-biggo/0c-ime/core/session"
)

type LinearModel struct {
	Version        string    `json:"version"`
	FeatureVersion string    `json:"feature_version"`
	Coefficients   []float64 `json:"coefficients"`
	Margin         float64   `json:"margin"`
	FitSpecSHA256  string    `json:"fit_spec_sha256"`
}

func (m LinearModel) Validate() error {
	if m.Version != "mixed-linear-v1" || m.FeatureVersion != mixedparse.FeatureVersion || len(m.Coefficients) != 26 || m.Coefficients[12] != 0 || m.Margin < 0 || math.IsInf(m.Margin, 0) || math.IsNaN(m.Margin) {
		return fmt.Errorf("invalid frozen linear model")
	}
	for _, w := range m.Coefficients {
		if math.IsNaN(w) || math.IsInf(w, 0) {
			return fmt.Errorf("nonfinite coefficient")
		}
	}
	return nil
}
func (m LinearModel) Score(c mixedparse.Candidate) (float64, error) {
	v := 0.0
	for i, w := range m.Coefficients {
		if i == 12 && w == 0 {
			continue
		}
		if c.Features[i] == nil {
			return 0, fmt.Errorf("unavailable feature %d", i)
		}
		v += w * *c.Features[i]
	}
	if math.IsNaN(v) || math.IsInf(v, 0) {
		return 0, fmt.Errorf("nonfinite ranked score")
	}
	return v, nil
}
func compatible(c mixedparse.Candidate, w mixedparse.Window) bool {
	return w.FirstToneEnd > 0 && mixedparse.ProjectedIdentity(c.Segments, w.FirstToneEnd) == mixedparse.ProjectedIdentity(w.Shown, w.FirstToneEnd)
}

// Choose follows the preregistered strict float64 comparison and extension rule.
func (m LinearModel) Choose(w mixedparse.Window, cs []mixedparse.Candidate, legacyID string) (mixedparse.Candidate, error) {
	candidates := []mixedparse.Candidate{}
	scores := map[string]float64{}
	for _, c := range cs {
		if c.Pruned || c.Lane == "opaque" {
			continue
		}
		v, e := m.Score(c)
		if e != nil {
			return c, e
		}
		scores[c.ID] = v
		candidates = append(candidates, c)
	}
	if len(candidates) == 0 {
		return mixedparse.Candidate{}, fmt.Errorf("no protected literal")
	}
	less := func(a, b mixedparse.Candidate) bool {
		av, bv := scores[a.ID], scores[b.ID]
		if av != bv {
			return av > bv
		}
		ac, bc := compatible(a, w), compatible(b, w)
		if ac != bc {
			return ac
		}
		al, bl := a.ID == legacyID, b.ID == legacyID
		if al != bl {
			return al
		}
		aa, bb := mixedparse.FirstChinese(a.Segments), mixedparse.FirstChinese(b.Segments)
		if aa != bb {
			return aa < bb
		}
		return a.Identity < b.Identity
	}
	sort.SliceStable(candidates, func(i, j int) bool { return less(candidates[i], candidates[j]) })
	best := candidates[0]
	if w.FirstToneEnd == 0 {
		return best, nil
	}
	var incumbent *mixedparse.Candidate
	for i := range candidates {
		if compatible(candidates[i], w) {
			incumbent = &candidates[i]
			break
		}
	}
	if incumbent == nil {
		p := append([]mixedparse.Segment(nil), w.Shown...)
		p = append(p, mixedparse.Segment{Kind: "literal", Begin: w.FirstToneEnd, End: len(w.Raw), Text: w.Raw[w.FirstToneEnd:]})
		c := mixedparse.Candidate{Lane: "fixed_fallback", Segments: p, Text: w.Visible + w.Raw[w.FirstToneEnd:], ScoreStatus: "fixed_fallback", FeatureVersion: mixedparse.FeatureVersion, History: "causal-ranked-v1"}
		c.ID = mixedparse.CandidateID(p)
		c.Identity = mixedparse.Identity(p)
		return c, nil
	}
	if scores[best.ID]-scores[incumbent.ID] > m.Margin {
		return best, nil
	}
	return *incumbent, nil
}

type legacyReference struct {
	Raw             string               `json:"raw"`
	Partition       []mixedparse.Segment `json:"partition"`
	OwnershipStatus string               `json:"ownership_status"`
}
type RankedRow struct {
	RowResult
	UsefulRevision      bool `json:"useful_revision"`
	HarmfulRevision     bool `json:"harmful_revision"`
	UnnecessaryRevision bool `json:"unnecessary_revision"`
	BenignRevision      bool `json:"benign_revision"`
	Revisions           int  `json:"revisions"`
	Fallbacks           int  `json:"fixed_fallbacks"`
	OutsideChanges      int  `json:"outside_changes"`
	KeyFailures         int  `json:"key_failures"`
	CommitFailures      int  `json:"commit_failures"`
	KeyEvents           int  `json:"key_events"`
}

func loadLegacyReferences(path string, wanted map[string]bool) (map[string]map[int]legacyReference, error) {
	f, e := os.Open(path)
	if e != nil {
		return nil, e
	}
	defer f.Close()
	scan := bufio.NewScanner(f)
	scan.Buffer(make([]byte, 65536), 8<<20)
	out := map[string]map[int]legacyReference{}
	for scan.Scan() {
		var header struct {
			FixtureID string `json:"fixture_id"`
			Event     int    `json:"event"`
			Policy    string `json:"policy"`
		}
		if e := json.Unmarshal(scan.Bytes(), &header); e != nil {
			return nil, e
		}
		if header.Policy != "aggressive" || !wanted[header.FixtureID] {
			continue
		}
		var r legacyReference
		if e := json.Unmarshal(scan.Bytes(), &r); e != nil {
			return nil, e
		}
		if out[header.FixtureID] == nil {
			out[header.FixtureID] = map[int]legacyReference{}
		}
		out[header.FixtureID][header.Event] = r
	}
	return out, scan.Err()
}
func Ranked(eng *engine.Engine, fixtures, aliases, assignment, modelPath, phase1, split, out string, marginOverride *float64) error {
	var model LinearModel
	mb, e := os.ReadFile(modelPath)
	if e != nil {
		return e
	}
	if e = json.Unmarshal(mb, &model); e != nil {
		return e
	}
	if marginOverride != nil {
		model.Margin = *marginOverride
	}
	if e = model.Validate(); e != nil {
		return e
	}
	if split == "test" && marginOverride != nil {
		return fmt.Errorf("test uses frozen selected margin only")
	}
	if split == "dev" {
		allowed := false
		for _, v := range []float64{0, .1, .25, .5, 1, 2, 4, 8} {
			allowed = allowed || model.Margin == v
		}
		if !allowed {
			return fmt.Errorf("undeclared development margin")
		}
	}
	var spec struct {
		Phase1RowsSHA256 string `json:"phase1_rows_sha256"`
	}
	specBytes, e := os.ReadFile("testdata/mixed-boundary/fitting-v1.json")
	if e != nil {
		return e
	}
	if e = json.Unmarshal(specBytes, &spec); e != nil {
		return e
	}
	rh, e := HashFile(filepath.Join(phase1, "rows.jsonl"))
	if e != nil || rh != spec.Phase1RowsSHA256 {
		return fmt.Errorf("Phase1 frozen rows changed")
	}

	specHash, e := HashFile("testdata/mixed-boundary/fitting-v1.json")
	if e != nil || specHash != model.FitSpecSHA256 {
		return fmt.Errorf("fitting spec changed")
	}
	adequacy, e := Structure("validate", fixtures, aliases, assignment)
	if e != nil {
		return e
	}
	fs, e := ReadFixtures(fixtures)
	if e != nil {
		return e
	}
	var as Assignment
	ab, e := os.ReadFile(assignment)
	if e != nil {
		return e
	}
	if e = json.Unmarshal(ab, &as); e != nil {
		return e
	}
	phaseRows := map[string]RowResult{}
	rf, e := os.Open(filepath.Join(phase1, "rows.jsonl"))
	if e != nil {
		return e
	}
	scan := bufio.NewScanner(rf)
	scan.Buffer(make([]byte, 65536), 8<<20)
	for scan.Scan() {
		var r RowResult
		if e = json.Unmarshal(scan.Bytes(), &r); e != nil {
			return e
		}
		phaseRows[r.ID] = r
	}
	rf.Close()
	if e = scan.Err(); e != nil {
		return e
	}
	wanted := map[string]bool{}
	for _, f := range fs {
		if as.Rows[f.ID].Split == split && phaseRows[f.ID].Eligible {
			wanted[f.ID] = true
		}
	}
	refs, e := loadLegacyReferences(filepath.Join(phase1, "events.jsonl"), wanted)
	if e != nil {
		return e
	}
	eventFile, e := os.OpenFile(filepath.Join(out, "events.jsonl"), os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0644)
	if e != nil {
		return e
	}
	defer eventFile.Close()
	candidateFile, e := os.OpenFile(filepath.Join(out, "candidates.jsonl"), os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0644)
	if e != nil {
		return e
	}
	defer candidateFile.Close()
	rowFile, e := os.OpenFile(filepath.Join(out, "rows.jsonl"), os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0644)
	if e != nil {
		return e
	}
	defer rowFile.Close()
	eventEnc, candidateEnc, rowEnc := json.NewEncoder(eventFile), json.NewEncoder(candidateFile), json.NewEncoder(rowFile)
	results := []RankedRow{}
	for _, f := range fs {
		if !wanted[f.ID] {
			continue
		}
		r := RankedRow{RowResult: phaseRows[f.ID]}
		r.WordOutputError = map[string]*bool{}
		r.EndToEndOutputError = map[string]*bool{}
		r.BoundaryError["proposed"] = nil
		r.PartitionError["proposed"] = nil
		r.LiteralConversion["proposed"] = nil
		ctx := mixedparse.Context{Layout: "standard", CustomTokens: append([]string(nil), f.CustomTokens...)}
		preludeBytes := 0
		if len(f.Prelude) > 0 {
			b, e := session.NewMixedEvalBaseline(eng.Model.Lex, eng.Model.LM, eng.English, true, f.CustomTokens)
			if e != nil {
				return e
			}
			for _, ev := range f.Prelude {
				reqs, e := request(ev, false, b)
				if e != nil {
					return e
				}
				for _, q := range reqs {
					b.Handle(q)
				}
				if ev.Type == "key" {
					preludeBytes += len(ev.Key)
				}
			}
			o := b.Observe()
			b.Close()
			if o.OwnershipStatus != "mapped" {
				return fmt.Errorf("unmapped frozen prelude %s", f.ID)
			}
			ctx = o.Context
			ctx.CustomTokens = append([]string(nil), f.CustomTokens...)
		}
		machine := mixedparse.NewMachine(ctx)
		m := personal.NewModel(eng.Model.Lex, eng.Model.LM)
		m.Now = nil
		evl := mixedparse.Evaluator{Lex: m.Source(), Score: m, System: m, English: eng.English}
		storeBefore, _ := json.Marshal(m.Store)
		gold := map[int][][]mixedparse.Segment{}
		assertions := map[int][]Assertion{}
		for _, a := range f.Assertions {
			if a.Policy == "generator" && a.Stage == "final" {
				gold[a.Window] = a.Accepted
				assertions[a.At] = append(assertions[a.At], a)
			}
		}
		firstErr, partErr, converted := false, false, false
		rawAll := ""
		closedIdentity := map[int]string{}
		for at, ev := range f.Events {
			beforeVisible := machine.Visible()
			var prev []mixedparse.Segment
			if machine.Current != nil {
				prev = append([]mixedparse.Segment(nil), machine.Current.Shown...)
			}
			selectedLane := ""
			selectedScore := (*float64)(nil)
			choose := func(w mixedparse.Window, cs []mixedparse.Candidate) (mixedparse.Candidate, error) {
				legacyID := ""
				ref := refs[f.ID][at]
				if ref.OwnershipStatus == "mapped" {
					p, ok := slicePartition(ref.Partition, preludeBytes+w.Offset, preludeBytes+w.Offset+len(w.Raw))
					if ok {
						legacyID = mixedparse.CandidateID(p)
					}
				}
				for i := range cs {
					cs[i].History = "causal-ranked-v1"
				}
				selected, e := model.Choose(w, cs, legacyID)
				if e != nil {
					return selected, e
				}
				selectedLane = selected.Lane
				if selected.Lane != "fixed_fallback" {
					v, e := model.Score(selected)
					if e != nil {
						return selected, e
					}
					selectedScore = &v
				} else {
					r.Fallbacks++
				}
				return selected, nil
			}
			start := time.Now()
			tr, e := machine.Step(ev, evl, prev, choose)
			if e != nil {
				return fmt.Errorf("ranked %s event%d: %w", f.ID, at, e)
			}
			duration := time.Since(start).Nanoseconds()
			for i := range tr.Candidates {
				tr.Candidates[i].History = "causal-ranked-v1"
			}
			if ev.Type == "key" {
				r.KeyEvents++
				rawAll += ev.Key
				actual := ""
				for _, w := range machine.Windows {
					actual += w.Raw
				}
				if machine.Current != nil {
					actual += machine.Current.Raw
				}
				if !machine.Bypass && actual != rawAll {
					r.KeyFailures++
				}
			}
			for _, w := range machine.Windows {
				id := mixedparse.Identity(w.Shown)
				if prev, ok := closedIdentity[w.Index]; ok && prev != id {
					r.OutsideChanges++
				}
				closedIdentity[w.Index] = id
			}
			if tr.Commit != nil && *tr.Commit != beforeVisible {
				r.CommitFailures++
			}
			if tr.Before != nil && tr.After != nil && tr.Before.FirstToneEnd > 0 && tr.After.RevisionCount > tr.Before.RevisionCount {
				r.Revisions++
				end := tr.Before.FirstToneEnd
				oldGood, newGood := false, false
				for _, g := range gold[tr.After.Index] {
					gp := mixedparse.ProjectedIdentity(g, end)
					oldGood = oldGood || mixedparse.ProjectedIdentity(tr.Before.Shown, end) == gp
					newGood = newGood || mixedparse.ProjectedIdentity(tr.After.Shown, end) == gp
				}
				switch {
				case !oldGood && newGood:
					r.UsefulRevision = true
				case oldGood && !newGood:
					r.HarmfulRevision = true
				case !oldGood && !newGood:
					r.UnnecessaryRevision = true
				case oldGood && newGood:
					r.BenignRevision = true
				}
			}
			if f.Intent == "literal" && tr.After != nil && hasChinese(tr.After.Shown) {
				converted = true
			}
			for _, c := range tr.Candidates {
				c.History = "causal-ranked-v1"
				rankScore, e := model.Score(c)
				if e != nil {
					return e
				}
				if e = candidateEnc.Encode(map[string]any{"fixture_id": f.ID, "event": at, "window": tr.After.Index, "candidate": c, "rank_score": rankScore}); e != nil {
					return e
				}
			}
			if e = eventEnc.Encode(map[string]any{"fixture_id": f.ID, "event": at, "policy": "ranked", "transition": tr, "visible": machine.Visible(), "selected_lane": selectedLane, "selected_score": selectedScore, "duration_ns": duration, "history": "causal-ranked-v1"}); e != nil {
				return e
			}
			for _, a := range assertions[at] {
				var w *mixedparse.Window
				if tr.After != nil && tr.After.Index == a.Window {
					w = tr.After
				}
				for _, x := range tr.ClosedWindows {
					if x.Index == a.Window {
						z := x
						w = &z
					}
				}
				if w == nil && a.Window < len(machine.Windows) {
					x := machine.Windows[a.Window]
					w = &x
				}
				if w == nil {
					return fmt.Errorf("ranked assertion window")
				}
				p := append([]mixedparse.Segment(nil), w.Shown...)
				if w.Pending != "" {
					begin := len(w.Raw) - len(w.Pending)
					p = append(p, mixedparse.Segment{Kind: "literal", Begin: begin, End: len(w.Raw), Text: w.Pending})
				}
				p = mergeRankedLiterals(p)
				if f.Intent == "chinese" {
					firstOK := false
					for _, g := range a.Accepted {
						firstOK = firstOK || mixedparse.FirstChinese(p) == mixedparse.FirstChinese(g)
					}
					firstErr = firstErr || !firstOK
					partErr = partErr || !isGold(p, a.Accepted)
					if len(a.AcceptedText) > 0 {
						textOK := false
						for _, text := range a.AcceptedText {
							textOK = textOK || w.Visible+w.Pending == text
						}
						r.EndToEndOutputError["proposed"] = boolp(!textOK)
						if isGold(p, a.Accepted) {
							r.WordOutputError["proposed"] = boolp(!textOK)
						}
					}
				}
			}
		}
		storeAfter, _ := json.Marshal(m.Store)
		if string(storeBefore) != string(storeAfter) {
			return fmt.Errorf("ranked hypothesis learned")
		}
		if f.Intent == "chinese" {
			r.BoundaryError["proposed"] = boolp(firstErr)
			r.PartitionError["proposed"] = boolp(partErr)
		}
		if f.Intent == "literal" {
			r.LiteralConversion["proposed"] = boolp(converted)
		}
		results = append(results, r)
		if e = rowEnc.Encode(r); e != nil {
			return e
		}
	}
	report := rankedSummary(results, split, model.Margin, adequacy)
	report["model_sha256"] = HashBytes(mb)
	report["fit_spec_sha256"] = specHash
	report["history"] = "causal-ranked-v1"
	report["coefficient12"] = 0
	report["phase1_rows_sha256"], _ = HashFile(filepath.Join(phase1, "rows.jsonl"))
	b, e := json.MarshalIndent(report, "", "  ")
	if e != nil {
		return e
	}
	return os.WriteFile(filepath.Join(out, "report.json"), append(b, '\n'), 0644)
}
func mergeRankedLiterals(p []mixedparse.Segment) []mixedparse.Segment {
	out := []mixedparse.Segment{}
	for _, s := range p {
		if len(out) > 0 && s.Kind == "literal" && out[len(out)-1].Kind == "literal" && out[len(out)-1].End == s.Begin {
			out[len(out)-1].End = s.End
			out[len(out)-1].Text += s.Text
		} else {
			out = append(out, s)
		}
	}
	return out
}
func rankedSummary(rows []RankedRow, split string, margin float64, a Adequacy) map[string]any {
	cn, ce, ln, le, n, useful, harmful, unnecessary, benign, revisions, outside, keyfail, commitfail, fallbacks, baselineN, baselineErr := 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0
	all := []RowResult{}
	for _, r := range rows {
		all = append(all, r.RowResult)
		if !r.Primary || r.Intent == "ambiguous" {
			continue
		}
		n++
		if r.Intent == "chinese" {
			cn++
			if *r.BoundaryError["proposed"] {
				ce++
			}
			if b := r.BoundaryError["aggressive"]; b != nil {
				baselineN++
				if *b {
					baselineErr++
				}
			}
		}
		if r.Intent == "literal" {
			ln++
			if *r.LiteralConversion["proposed"] {
				le++
			}
		}
		if r.UsefulRevision {
			useful++
		}
		if r.HarmfulRevision {
			harmful++
		}
		if r.UnnecessaryRevision {
			unnecessary++
		}
		if r.BenignRevision {
			benign++
		}
		revisions += r.Revisions
		outside += r.OutsideChanges
		keyfail += r.KeyFailures
		commitfail += r.CommitFailures
		fallbacks += r.Fallbacks
	}
	var improvement *float64
	if baselineN == cn {
		v := 0.0
		if baselineErr > 0 {
			v = float64(baselineErr-ce) / float64(baselineErr)
		}
		improvement = &v
	}
	ci := Rate(le, ln)
	g4 := false
	if improvement != nil {
		g4 = baselineErr == 0 && ce == 0 || baselineErr > 0 && *improvement >= .3
	}
	g5 := ci.Value != nil && *ci.Value <= .01 && ci.Wilson95[1] <= .01
	g6 := harmful == 0 && n > 0 && float64(unnecessary)/float64(n) <= .01 && outside == 0
	categories := map[string][]RowResult{}
	for _, r := range all {
		for _, cat := range r.Categories {
			categories[cat] = append(categories[cat], r)
		}
	}
	categoryReports := map[string]any{}
	for cat, rs := range categories {
		categoryReports[cat] = policySummary(rs, "proposed")
	}
	return map[string]any{"phase": "ranked", "split": split, "margin": margin, "exploratory_pilot": true, "rows": len(rows), "primary_unambiguous_rows": n, "chinese_first_boundary_error": Rate(ce, cn), "literal_any_event_conversion": ci, "baseline_aggressive_first_boundary_error_mapped": Rate(baselineErr, baselineN), "relative_error_reduction": improvement, "zero_baseline_note": func() any {
		if baselineErr == 0 {
			return "no measurable improvement"
		}
		return nil
	}(), "useful_revision": Rate(useful, n), "harmful_revision": Rate(harmful, n), "unnecessary_revision": Rate(unnecessary, n), "benign_revision": Rate(benign, n), "ownership_revisions": revisions, "outside_changes": outside, "key_failures": keyfail, "commit_failures": commitfail, "fixed_fallbacks": fallbacks, "diagnostic_gate_points": map[string]bool{"G4": g4, "G5": g5, "G6": g6, "G7_offline": outside == 0 && keyfail == 0 && commitfail == 0}, "qualification": a, "formal_GO": false, "policies": map[string]any{"proposed": policySummary(all, "proposed"), "aggressive": policySummary(all, "aggressive"), "conservative": policySummary(all, "conservative")}, "categories": categoryReports, "notes": strings.Join([]string{"formal literal>=400 qualification remains mandatory", "Phase1 unsupported paired rows remain unavailable", "literal conversion counts any automatic Chinese ownership event", "model scores are ranking values, not probabilities"}, "; ")}
}
