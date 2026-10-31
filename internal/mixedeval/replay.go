//go:build mixedeval

package mixedeval

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/0c-biggo/0c-ime/core/engine"
	"github.com/0c-biggo/0c-ime/core/mixedparse"
	"github.com/0c-biggo/0c-ime/core/personal"
	"github.com/0c-biggo/0c-ime/core/proto"
	"github.com/0c-biggo/0c-ime/core/session"
)

type Count struct {
	Numerator   int         `json:"numerator"`
	Denominator int         `json:"denominator"`
	Value       *float64    `json:"value"`
	Wilson95    *[2]float64 `json:"wilson95"`
}

func Rate(n, d int) Count {
	r := Count{Numerator: n, Denominator: d}
	if d == 0 {
		return r
	}
	p := float64(n) / float64(d)
	r.Value = &p
	z := 1.959963984540054
	zz := z * z
	center := (p + zz/(2*float64(d))) / (1 + zz/float64(d))
	half := z * math.Sqrt(p*(1-p)/float64(d)+zz/(4*float64(d*d))) / (1 + zz/float64(d))
	ci := [2]float64{math.Max(0, center-half), math.Min(1, center+half)}
	r.Wilson95 = &ci
	return r
}

type Gate struct {
	Status string `json:"status"`
	Pass   *bool  `json:"pass"`
	Reason string `json:"reason"`
}

func gate(status, reason string) Gate {
	g := Gate{Status: status, Reason: reason}
	if status == "pass" || status == "fail" {
		p := status == "pass"
		g.Pass = &p
	}
	return g
}

type RowResult struct {
	WordOutputError     map[string]*bool          `json:"word_output_error"`
	EndToEndOutputError map[string]*bool          `json:"end_to_end_output_error"`
	ID                  string                    `json:"fixture_id"`
	Intent              string                    `json:"intent"`
	Split               string                    `json:"split"`
	GroupID             string                    `json:"group_id"`
	Primary             bool                      `json:"primary"`
	Categories          []string                  `json:"categories"`
	Mandatory           bool                      `json:"mandatory"`
	Eligible            bool                      `json:"eligible"`
	CappedRecall        *bool                     `json:"capped_recall"`
	UncappedRecall      *bool                     `json:"uncapped_recall"`
	AmbiguousSurvival   *bool                     `json:"ambiguous_survival"`
	BoundaryError       map[string]*bool          `json:"first_boundary_error"`
	PartitionError      map[string]*bool          `json:"partition_error"`
	LiteralConversion   map[string]*bool          `json:"literal_conversion"`
	Ownership           map[string]map[string]int `json:"ownership"`
	Failures            []string                  `json:"failures"`
}

func boolp(v bool) *bool { return &v }
func request(e mixedparse.Event, private bool, b *session.MixedEvalBaseline) ([]proto.Request, error) {
	r := proto.Request{Private: private}
	switch e.Type {
	case "key":
		r.Type = proto.Key
		r.Key = e.Key
	case "finalize":
		switch e.Reason {
		case "enter":
			r.Type = proto.Key
			r.Key = proto.KeyEnter
		case "commit":
			r.Type = proto.Commit
		case "space":
			r.Type = proto.Key
			r.Key = " "
		case "punctuation":
			r.Type = proto.Key
			r.Key = e.Text
		case "cursor":
			r.Type = proto.Cursor
			if e.Target == nil {
				return nil, fmt.Errorf("cursor target")
			}
			r.To = *e.Target
		case "backspace":
			r.Type = proto.Backspace
		case "delete":
			r.Type = proto.Key
			r.Key = proto.KeyDelete
		case "open_candidates":
			r.Type = proto.Key
			r.Key = proto.KeyDown
		case "cancel", "reset":
			r.Type = proto.Cancel
		case "focus_end":
			if e.CommitVisible == nil {
				return nil, fmt.Errorf("focus policy")
			}
			r.Type = proto.Cancel
			if *e.CommitVisible {
				r.Type = proto.Commit
			}
		default:
			return nil, fmt.Errorf("finalizer")
		}
	case "select":
		if e.Ref == nil {
			return nil, fmt.Errorf("selection ref")
		}
		id, err := b.ResolveSelection(e.Ref.Kind, e.Ref.Span, e.Ref.Reading, e.Ref.Text)
		if err != nil {
			return nil, err
		}
		r.Type = proto.Select
		r.Index = id
	case "environment":
		return []proto.Request{{Type: proto.Pause, On: e.Paused, Private: e.Private}, {Type: proto.Mode, On: e.English, Private: e.Private}, {Type: proto.Display, Private: e.Private}}, nil
	default:
		return nil, fmt.Errorf("event")
	}
	return []proto.Request{r}, nil
}
func slicePartition(p []mixedparse.Segment, begin, end int) ([]mixedparse.Segment, bool) {
	out := []mixedparse.Segment{}
	at := begin
	for _, s := range p {
		if s.End <= begin || s.Begin >= end {
			continue
		}
		if s.Kind == "chinese" && (s.Begin < begin || s.End > end) {
			return nil, false
		}
		old := s.Begin
		if s.Begin < begin {
			s.Begin = begin
		}
		if s.End > end {
			s.End = end
		}
		if s.Begin != at {
			return nil, false
		}
		if s.Kind == "literal" {
			s.Text = s.Text[s.Begin-old : s.End-old]
		}
		at = s.End
		s.Begin -= begin
		s.End -= begin
		out = append(out, s)
	}
	return out, at == end
}
func match(c []mixedparse.Candidate, gold []mixedparse.Segment, capped bool) bool {
	id := mixedparse.CandidateID(gold)
	for _, x := range c {
		if x.Lane != "opaque" && (!capped || !x.Pruned) && x.ID == id {
			return true
		}
	}
	return false
}
func hasChinese(p []mixedparse.Segment) bool { return mixedparse.FirstChinese(p) != math.MaxInt }
func isGold(p []mixedparse.Segment, accepted [][]mixedparse.Segment) bool {
	id := mixedparse.Identity(p)
	for _, g := range accepted {
		if id == mixedparse.Identity(g) {
			return true
		}
	}
	return false
}
func primarySet(a Adequacy) map[string]bool {
	m := map[string]bool{}
	for _, id := range a.Primary {
		m[id] = true
	}
	return m
}
func Enumerate(data, fixtures, aliases, assignment, lock, split, out string, eng *engine.Engine) error {
	adequacy, err := Structure("validate", fixtures, aliases, assignment)
	if err != nil {
		return err
	}
	rows, err := ReadFixtures(fixtures)
	if err != nil {
		return err
	}
	var as Assignment
	ab, err := os.ReadFile(assignment)
	if err != nil {
		return err
	}
	if err = json.Unmarshal(ab, &as); err != nil {
		return err
	}
	var meta struct {
		Rows map[string]struct {
			Categories      []string `json:"categories"`
			Mandatory       bool     `json:"mandatory"`
			QualityEligible bool     `json:"quality_eligible"`
		} `json:"rows"`
	}
	mb, err := os.ReadFile("testdata/mixed-boundary/metadata.json")
	if err != nil {
		return err
	}
	if err = json.Unmarshal(mb, &meta); err != nil {
		return err
	}
	files := map[string]*os.File{}
	enc := map[string]*json.Encoder{}
	for _, name := range []string{"candidates.jsonl", "events.jsonl", "rows.jsonl"} {
		f, e := os.OpenFile(filepath.Join(out, name), os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0644)
		if e != nil {
			return e
		}
		files[name] = f
		enc[name] = json.NewEncoder(f)
		defer f.Close()
	}
	lockHash, err := HashFile(lock)
	if err != nil {
		return err
	}
	primary := primarySet(adequacy)
	all := []RowResult{}
	invalidEvents, protectedCalls, generateCalls, scoreCalls, totalEvents := 0, 0, 0, 0, 0
	for _, f := range rows {
		assign := as.Rows[f.ID]
		if split != "all" && assign.Split != split {
			continue
		}
		md := meta.Rows[f.ID]
		rr := RowResult{ID: f.ID, Intent: f.Intent, Split: assign.Split, GroupID: assign.GroupID, Primary: primary[f.ID], Categories: md.Categories, Mandatory: md.Mandatory, Eligible: md.QualityEligible, WordOutputError: map[string]*bool{}, EndToEndOutputError: map[string]*bool{}, BoundaryError: map[string]*bool{}, PartitionError: map[string]*bool{}, LiteralConversion: map[string]*bool{}, Ownership: map[string]map[string]int{"aggressive": {}, "conservative": {}}}
		a, e := session.NewMixedEvalBaseline(eng.Model.Lex, eng.Model.LM, eng.English, true, f.CustomTokens)
		if e != nil {
			return e
		}
		c, e := session.NewMixedEvalBaseline(eng.Model.Lex, eng.Model.LM, eng.English, false, f.CustomTokens)
		if e != nil {
			a.Close()
			return e
		}
		preludeBytes := 0
		for _, ev := range f.Prelude {
			requests, e := request(ev, false, a)
			if e != nil {
				return e
			}
			for _, q := range requests {
				a.Handle(q)
				c.Handle(q)
			}
			if ev.Type == "key" {
				preludeBytes += len(ev.Key)
			}
		}
		ctx := a.Observe().Context
		ctx.CustomTokens = append([]string(nil), f.CustomTokens...)
		machine := mixedparse.NewMachine(ctx)
		model := personal.NewModel(eng.Model.Lex, eng.Model.LM)
		model.Now = nil
		model.Store = personal.NewStore()
		evl := mixedparse.Evaluator{Lex: model.Source(), Score: model, System: model, English: eng.English}
		beforeStore, _ := json.Marshal(model.Store)
		previous := []mixedparse.Segment(nil)
		private := false
		assertions := map[int][]Assertion{}
		for _, g := range f.Assertions {
			assertions[g.At] = append(assertions[g.At], g)
		}
		recall, uncapped, ambig := true, true, true
		chineseAssertions, ambigAssertions := 0, 0
		boundary := map[string]bool{"aggressive": false, "conservative": false, "longest": false}
		partition := map[string]bool{"aggressive": false, "conservative": false, "longest": false}
		conversion := map[string]bool{"aggressive": false, "conservative": false, "longest": false}
		available := map[string]bool{"aggressive": true, "conservative": true, "longest": true}
		lastCandidates := map[int][]mixedparse.Candidate{}
		for at, ev := range f.Events {
			totalEvents++
			if ev.Type == "environment" {
				private = ev.Private
			}
			observed := map[string]session.MixedEvalObservation{}
			controlCount := 0
			for name, b := range map[string]*session.MixedEvalBaseline{"aggressive": a, "conservative": c} {
				requests, e := request(ev, private, b)
				if e != nil {
					rr.Failures = append(rr.Failures, name+": "+e.Error())
					available[name] = false
					observed[name] = b.Observe()
					continue
				}
				var o session.MixedEvalObservation
				for _, q := range requests {
					o = b.Handle(q)
				}
				if ev.Type == "environment" {
					controlCount = len(requests)
				}
				observed[name] = o
				rr.Ownership[name][o.OwnershipStatus]++
				if o.OwnershipStatus == "invalid" {
					invalidEvents++
					rr.Failures = append(rr.Failures, fmt.Sprintf("%s invalid ownership event %d: %s", name, at, o.OwnershipReason))
				}
				if e := enc["events.jsonl"].Encode(map[string]any{"fixture_id": f.ID, "event": at, "policy": name, "state": nil, "raw": o.Raw, "visible": o.Visible, "pending": o.Pending, "partition": o.Partition, "ownership_status": o.OwnershipStatus, "ownership_reason": o.OwnershipReason, "revision_count": nil, "closure_reason": nil, "generate_calls": 0, "score_calls": 0, "delegate_count": len(requests), "control_request_count": controlCount, "commit": o.Snapshot.Commit, "locks": nil, "snapshot": o.Snapshot, "store_digest": o.StoreDigest, "store_total": o.StoreTotal}); e != nil {
					return e
				}
			}
			start := time.Now()
			choose := func(w mixedparse.Window, candidates []mixedparse.Candidate) (mixedparse.Candidate, error) {
				o := observed["aggressive"]
				p, ok := slicePartition(o.Partition, preludeBytes+w.Offset, preludeBytes+w.Offset+len(w.Raw))
				if o.OwnershipStatus != "mapped" || !ok {
					return mixedparse.Literal(w.Raw), nil
				}
				for _, x := range candidates {
					if x.ID == mixedparse.CandidateID(p) {
						return x, nil
					}
				}
				x := mixedparse.Candidate{ID: mixedparse.CandidateID(p), Lane: "opaque", Segments: p, Text: w.Raw, Identity: mixedparse.Identity(p), FeatureVersion: mixedparse.FeatureVersion, History: "legacy-lagged-v1"}
				return x, nil
			}
			tr, e := machine.Step(ev, evl, previous, choose)
			if e != nil {
				a.Close()
				c.Close()
				return fmt.Errorf("%s event%d: %w", f.ID, at, e)
			}
			duration := time.Since(start).Nanoseconds()
			generateCalls += tr.GenerateCalls
			scoreCalls += tr.ScoreCalls
			if (ctx.Protected() || ev.Type == "environment" && (ev.Private || ev.Paused || ev.Blocked || ev.English || ev.Toneless || ev.Layout != "standard")) && (tr.GenerateCalls > 0 || tr.ScoreCalls > 0) {
				protectedCalls += tr.GenerateCalls + tr.ScoreCalls
			}
			if len(tr.Candidates) > 0 && tr.After != nil {
				wi := tr.After.Index
				lastCandidates[wi] = tr.Candidates
				o := observed["aggressive"]
				p, ok := slicePartition(o.Partition, preludeBytes+tr.After.Offset, preludeBytes+tr.After.Offset+len(tr.After.Raw))
				if o.OwnershipStatus == "mapped" && ok {
					previous = p
				} else {
					previous = nil
				}
				legacyID := ""
				if ok {
					legacyID = mixedparse.CandidateID(p)
				}
				equiv := false
				for _, candidate := range tr.Candidates {
					if candidate.ID == legacyID {
						equiv = true
					}
					record := map[string]any{"fixture_id": f.ID, "event": at, "window": wi, "candidate_id": candidate.ID, "lane": candidate.Lane, "segments": candidate.Segments, "readings": candidate.Readings, "support": candidate.Support, "raw_score": candidate.RawScore, "score_status": candidate.ScoreStatus, "features": candidate.Features, "feature_version": candidate.FeatureVersion, "history": candidate.History, "pruned": candidate.Pruned, "text": candidate.Text, "legacy_exact_equivalent": candidate.ID == legacyID}
					if e := enc["candidates.jsonl"].Encode(record); e != nil {
						return e
					}
				}
				if !equiv {
					if e := enc["candidates.jsonl"].Encode(map[string]any{"fixture_id": f.ID, "event": at, "window": wi, "candidate_id": "legacy:" + f.ID + fmt.Sprint(at), "lane": "opaque", "segments": nil, "readings": nil, "support": nil, "raw_score": nil, "score_status": "opaque", "features": nil, "feature_version": mixedparse.FeatureVersion, "history": "legacy-lagged-v1", "pruned": false, "text": o.Visible, "ownership_status": o.OwnershipStatus}); e != nil {
						return e
					}
				}
			}
			if e := enc["events.jsonl"].Encode(map[string]any{"fixture_id": f.ID, "event": at, "policy": "enumerate", "state": tr.After, "raw": func() string {
				if tr.After != nil {
					return tr.After.Raw
				}
				return ""
			}(), "visible": nil, "proposed_display": nil, "pending": nil, "partition": nil, "revision_count": nil, "closure_reason": nil, "generate_calls": tr.GenerateCalls, "score_calls": tr.ScoreCalls, "delegate_count": tr.DelegateCount, "control_request_count": controlCount, "commit": nil, "locks": tr.Locks, "transition": tr, "duration_ns": duration, "history": "legacy-lagged-v1"}); e != nil {
				return e
			}
			for _, g := range assertions[at] {
				if g.Policy != "generator" || g.Stage != "final" {
					continue
				}
				var w *mixedparse.Window
				if tr.After != nil && tr.After.Index == g.Window {
					w = tr.After
				}
				for _, closed := range tr.ClosedWindows {
					if closed.Index == g.Window {
						x := closed
						w = &x
					}
				}
				if w == nil && g.Window < len(machine.Windows) {
					x := machine.Windows[g.Window]
					w = &x
				}
				if w == nil {
					return fmt.Errorf("%s assertion unavailable window", f.ID)
				}
				candidates := append([]mixedparse.Candidate(nil), lastCandidates[g.Window]...)
				candidates = append(candidates, mixedparse.Literal(w.Raw))
				longest := mixedparse.Longest(candidates)
				hasChineseGold := false
				thisRecall, thisUncapped := false, false
				for _, gold := range g.Accepted {
					if hasChinese(gold) {
						hasChineseGold = true
						thisRecall = thisRecall || match(candidates, gold, true)
						thisUncapped = thisUncapped || match(candidates, gold, false)
					}
				}
				if f.Intent == "chinese" && hasChineseGold {
					chineseAssertions++
					recall = recall && thisRecall
					uncapped = uncapped && thisUncapped
					if !thisRecall {
						rr.Failures = append(rr.Failures, fmt.Sprintf("capped intended path absent at event%d window%d", at, g.Window))
					}
				}
				if f.Intent == "ambiguous" {
					ambigAssertions++
					for _, gold := range g.Accepted {
						ambig = ambig && match(candidates, gold, true)
					}
				}
				for _, policy := range []string{"aggressive", "conservative", "longest"} {
					var p []mixedparse.Segment
					ok := true
					if policy == "longest" {
						p = longest.Segments
					} else {
						o := observed[policy]
						p, ok = slicePartition(o.Partition, preludeBytes+w.Offset, preludeBytes+w.Offset+len(w.Raw))
						ok = ok && o.OwnershipStatus == "mapped"
					}
					if !ok {
						available[policy] = false
						continue
					}
					if f.Intent == "chinese" {
						if len(g.AcceptedText) > 0 {
							actual := longest.Text
							if policy != "longest" {
								actual = observed[policy].Visible
							}
							textOK := false
							for _, goldText := range g.AcceptedText {
								textOK = textOK || actual == goldText
							}
							rr.EndToEndOutputError[policy] = boolp(!textOK)
							if isGold(p, g.Accepted) {
								rr.WordOutputError[policy] = boolp(!textOK)
							}
						}
						rightFirst := false
						for _, gold := range g.Accepted {
							rightFirst = rightFirst || mixedparse.FirstChinese(p) == mixedparse.FirstChinese(gold)
						}
						boundary[policy] = boundary[policy] || !rightFirst
						partition[policy] = partition[policy] || !isGold(p, g.Accepted)
					}
					if f.Intent == "literal" {
						conversion[policy] = conversion[policy] || hasChinese(p) || !isGold(p, g.Accepted)
					}
				}
			}
		}
		afterStore, _ := json.Marshal(model.Store)
		if string(beforeStore) != string(afterStore) {
			return fmt.Errorf("hypothetical model mutated")
		}
		a.Close()
		c.Close()
		if chineseAssertions > 0 {
			rr.CappedRecall = boolp(recall)
			rr.UncappedRecall = boolp(uncapped)
		}
		if ambigAssertions > 0 {
			rr.AmbiguousSurvival = boolp(ambig)
		}
		for _, p := range []string{"aggressive", "conservative", "longest"} {
			if available[p] && rr.Eligible {
				if f.Intent == "chinese" {
					rr.BoundaryError[p] = boolp(boundary[p])
					rr.PartitionError[p] = boolp(partition[p])
				}
				if f.Intent == "literal" {
					rr.LiteralConversion[p] = boolp(conversion[p])
				}
			} else {
				rr.BoundaryError[p] = nil
				rr.PartitionError[p] = nil
				rr.LiteralConversion[p] = nil
			}
		}
		all = append(all, rr)
		if e := enc["rows.jsonl"].Encode(rr); e != nil {
			return e
		}
	}
	for _, f := range files {
		if err := f.Sync(); err != nil {
			return err
		}
	}
	report := Summarize(all, adequacy, invalidEvents, protectedCalls)
	report["provenance"] = map[string]any{"protocol": mixedparse.Protocol, "feature_version": mixedparse.FeatureVersion, "run_lock_sha256": lockHash, "run_lock": lock, "data": data, "split": split, "phase": "enumerate", "history": "legacy-lagged-v1", "baseline": "e754019571573051d8c29e06526802c31d5172e2"}
	report["calls"] = map[string]int{"events": totalEvents, "generate": generateCalls, "decode_best_full": scoreCalls, "protected_experimental": protectedCalls, "hypothetical_learning": 0, "hypothetical_persistence": 0}
	b, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(out, "report.json"), append(b, '\n'), 0644)
}
func Summarize(rows []RowResult, a Adequacy, invalid, protected int) map[string]any {
	chineseN, chineseHit, uncappedHit, ambN, ambHit := 0, 0, 0, 0, 0
	mandatoryFailure := []string{}
	for _, r := range rows {
		if r.CappedRecall != nil {
			chineseN++
			if *r.CappedRecall {
				chineseHit++
			} else if r.Mandatory {
				mandatoryFailure = append(mandatoryFailure, r.ID)
			}
		}
		if r.UncappedRecall != nil && *r.UncappedRecall {
			uncappedHit++
		}
		if r.AmbiguousSurvival != nil {
			ambN++
			if *r.AmbiguousSurvival {
				ambHit++
			} else if r.Mandatory {
				mandatoryFailure = append(mandatoryFailure, r.ID)
			}
		}
	}
	recall := Rate(chineseHit, chineseN)
	g3 := "fail"
	if recall.Value != nil && *recall.Value >= .99 && len(mandatoryFailure) == 0 && ambN > 0 && ambN == ambHit {
		g3 = "pass"
	}
	gates := map[string]Gate{"G1": gate("unavailable", "external correctness/parity evidence plus data qualification required"), "G2": gate("pass", "value-only hypotheses; store digest unchanged for every row; no persistence API"), "G3": gate(g3, "all labeled Chinese rows, all windows; capped strict candidates; mandatory and eligible ambiguous alternatives"), "G4": gate("unavailable", "no fitted proposed ranker"), "G5": gate("unavailable", "no fitted proposed ranker"), "G6": gate("unavailable", "Phase1 legacy-lagged feature audit is not proposed display stability"), "G7": gate("unavailable", "state/conservative parity tests separate; live materialization outside offline scope"), "G8": gate("unavailable", "required paired proposed metrics unavailable; source category/group qualification required")}
	outcome := "INCONCLUSIVE"
	if !a.PilotAdequate || !a.ReleaseAdequate {
		gates["G1"] = gate("fail", "frozen dataset does not meet prescribed independent-group/split counts")
	}
	if invalid > 0 {
		gates["G1"] = gate("fail", fmt.Sprintf("%d invalid baseline ownership events", invalid))
	}
	if protected > 0 {
		gates["G2"] = gate("fail", "experimental calls under protected flags")
	}
	if g3 == "fail" {
		outcome = "NO_GO"
	}
	if invalid > 0 {
		outcome = "INCONCLUSIVE"
	}
	diagnostics := map[string]any{}
	categories := map[string][]RowResult{}
	for _, r := range rows {
		for _, cat := range r.Categories {
			categories[cat] = append(categories[cat], r)
		}
	}
	for _, policy := range []string{"aggressive", "conservative", "longest"} {
		diagnostics[policy] = policySummary(rows, policy)
	}
	categorySummary := map[string]any{}
	for cat, rs := range categories {
		n, hit := 0, 0
		testPrimaryGroups := map[string]bool{}
		for _, r := range rs {
			if r.CappedRecall != nil {
				n++
				if *r.CappedRecall {
					hit++
				}
			}
			if r.Split == "test" && r.Primary {
				testPrimaryGroups[r.GroupID] = true
			}
		}
		categorySummary[cat] = map[string]any{"rows": len(rs), "capped_recall": Rate(hit, n), "test_primary_groups": len(testPrimaryGroups), "policies": map[string]any{"aggressive": policySummary(rs, "aggressive"), "conservative": policySummary(rs, "conservative"), "longest": policySummary(rs, "longest")}}
	}
	return map[string]any{"outcome": outcome, "gates": gates, "adequacy": a, "rows": len(rows), "capped_intended_recall": recall, "uncapped_intended_recall": Rate(uncappedHit, chineseN), "pruning_loss": Rate(uncappedHit-chineseHit, chineseN), "ambiguous_both_survival": Rate(ambHit, ambN), "mandatory_failures": mandatoryFailure, "baseline_diagnostics": diagnostics, "categories": categorySummary, "proposed": nil, "fitting": "not started; require valid G1/G2/G7 and G3 before fitting", "ownership_invalid_events": invalid, "limitations": []string{"synthetic labels are experimenter-declared intentions, not user-typing performance", "fixture declares run origin and context", "public hardware tokens establish lexical literal stress only", "Linux timing is exploratory; no native Windows evidence", "scripted tests do not establish learned stability"}}
}
func policySummary(rows []RowResult, policy string) map[string]any {
	boundaryN, boundaryErr, partitionN, partitionErr, literalN, converted, missingB, missingL := 0, 0, 0, 0, 0, 0, 0, 0
	owned := map[string]int{}
	testLN, testConverted := 0, 0
	groups := map[string]bool{}
	for _, r := range rows {
		for k, v := range r.Ownership[policy] {
			owned[k] += v
		}
		if !r.Eligible {
			continue
		}
		if r.Intent == "chinese" {
			if v := r.BoundaryError[policy]; v != nil {
				boundaryN++
				if *v {
					boundaryErr++
				}
			} else {
				missingB++
			}
			if v := r.PartitionError[policy]; v != nil {
				partitionN++
				if *v {
					partitionErr++
				}
			}
		}
		if r.Intent == "literal" {
			if v := r.LiteralConversion[policy]; v != nil {
				literalN++
				if *v {
					converted++
					groups[r.GroupID] = true
				}
				if r.Split == "test" && r.Primary {
					testLN++
					if *v {
						testConverted++
					}
				}
			} else {
				missingL++
			}
		}
	}
	var pairedB, pairedL any
	if missingB == 0 {
		pairedB = Rate(boundaryErr, boundaryN)
	}
	if missingL == 0 {
		pairedL = Rate(converted, literalN)
	}
	return map[string]any{"first_boundary_error": pairedB, "literal_conversion": pairedL, "mapped_only_first_boundary_error": Rate(boundaryErr, boundaryN), "mapped_only_complete_partition_error": Rate(partitionErr, partitionN), "mapped_only_literal_conversion": Rate(converted, literalN), "mapped_only_test_primary_literal_conversion": Rate(testConverted, testLN), "missing_chinese_rows": missingB, "missing_literal_rows": missingL, "ownership_events": owned, "test_primary": primaryPolicySummary(rows, policy), "literal_worst_case_groups_with_conversion": len(groups), "note": strings.Join([]string{"mapped-only values are diagnostics", "any unsupported row makes the paired metric unavailable"}, "; ")}
}

func primaryPolicySummary(rows []RowResult, policy string) map[string]any {
	bn, be, pn, pe, ln, le, bmissing, lmissing, wn, we, on, oe := 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0
	for _, r := range rows {
		if !r.Primary || r.Split != "test" || !r.Eligible {
			continue
		}
		if r.Intent == "chinese" {
			if v := r.BoundaryError[policy]; v != nil {
				bn++
				if *v {
					be++
				}
			} else {
				bmissing++
			}
			if v := r.PartitionError[policy]; v != nil {
				pn++
				if *v {
					pe++
				}
			}
		}
		if r.Intent == "literal" {
			if v := r.LiteralConversion[policy]; v != nil {
				ln++
				if *v {
					le++
				}
			} else {
				lmissing++
			}
		}
		if v := r.WordOutputError[policy]; v != nil {
			wn++
			if *v {
				we++
			}
		}
		if v := r.EndToEndOutputError[policy]; v != nil {
			on++
			if *v {
				oe++
			}
		}
	}
	var boundary, literal, partition any
	if bmissing == 0 {
		boundary = Rate(be, bn)
		partition = Rate(pe, pn)
	}
	if lmissing == 0 {
		literal = Rate(le, ln)
	}
	return map[string]any{"first_boundary_error": boundary, "complete_partition_exact_reading_error": partition, "literal_conversion": literal, "missing_chinese_rows": bmissing, "missing_literal_rows": lmissing, "mapped_only_first_boundary_error": Rate(be, bn), "mapped_only_literal_conversion": Rate(le, ln), "correct_reading_chinese_surface_error": Rate(we, wn), "end_to_end_chinese_surface_error": Rate(oe, on)}
}
