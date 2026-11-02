//go:build shiftreturneval

package shiftreturn

import (
	"encoding/json"
	"fmt"
	"github.com/0c-biggo/0c-ime/core/proto"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"time"

	"github.com/0c-biggo/0c-ime/core/engine"
)

// Calibrate uses exactly the frozen five examples, eleven lifecycle cases and
// four long stress inputs. These are operational measurements, never samples.
func Calibrate(loaded *engine.Engine) ([]map[string]any, error) {
	cases := []map[string]any{}
	fixtures, _, e := ReadFixtures("design/shift-return-v2/examples.jsonl")
	if e != nil {
		return nil, e
	}
	for _, f := range fixtures {
		probe, _ := NewScoringProbe(loaded.Model.Lex, loaded.Model.LM, loaded.English)
		p := Policy{Probe: probe, Model: modelForZero(), Flow: "revise", Enabled: true, Standard: true, Condition: "EC-revise"}
		state := State{FragmentID: f.ID, Prelude: f.ExpectedPrelude, Mode: "zhuyin"}
		start := time.Now()
		status := "ok"
		for _, event := range f.Events {
			var err error
			state, _, err = Step(state, event, p)
			if err != nil {
				return nil, err
			}
			if state.LastResult != nil && state.LastResult.Status != "ok" {
				status = state.LastResult.Status
			}
		}
		cases = append(cases, map[string]any{"id": f.ID, "raw_bytes": len(f.ExpectedRaw), "status": status, "generate_calls": probe.GenerateCalls, "probe_lex_queries": probe.ProbeLexQueries, "best_calls": probe.BestCalls, "elapsed_ns": time.Since(start).Nanoseconds()})
	}
	b, e := os.ReadFile("design/shift-return-v2/lifecycle.json")
	if e != nil {
		return nil, e
	}
	var lifecycle struct {
		Cases []struct {
			Event      string
			Before     struct{ App, Prelude, Display string }
			AppendText string `json:"append_text"`
			Expected   struct{ View, Commit string }
		}
	}
	if e = json.Unmarshal(b, &lifecycle); e != nil {
		return nil, e
	}
	for _, c := range lifecycle.Cases {
		probe, _ := NewScoringProbe(loaded.Model.Lex, loaded.Model.LM, loaded.English)
		p := Policy{Probe: probe, Model: modelForZero(), Flow: "revise", Enabled: true, Standard: true, Condition: "EC-revise"}
		state := State{FragmentID: "lifecycle-" + c.Event, App: c.Before.App, Prelude: c.Before.Prelude, Display: c.Before.Display, Raw: "x593", Eligible: true, Selection: &Hypothesis{K: 1}}
		event := Event{Type: c.Event}
		switch c.Event {
		case "paste":
			event.Text = &c.AppendText
		case "cursor":
			event.To = ptr(4)
		case "mode", "caps", "protect", "pause":
			event.On = ptr(true)
		}
		start := time.Now()
		_, snapshot, e := Step(state, event, p)
		if e != nil {
			return nil, e
		}
		if snapshot.View != c.Expected.View || snapshot.Commit != c.Expected.Commit || probe.GenerateCalls != 0 {
			return nil, fmt.Errorf("lifecycle calibration differs")
		}
		cases = append(cases, map[string]any{"id": "lifecycle-" + c.Event, "raw_bytes": 0, "status": "ok", "generate_calls": 0, "probe_lex_queries": 0, "best_calls": 0, "elapsed_ns": time.Since(start).Nanoseconds()})
	}
	for i, raw := range []string{strings.Repeat("A", 129), strings.Repeat("A", 512), "x" + strings.Repeat("593", 17), "x" + strings.Repeat("593", 64)} {
		probe, _ := NewScoringProbe(loaded.Model.Lex, loaded.Model.LM, loaded.English)
		start := time.Now()
		status := "ok"
		for offset := 1; offset <= len(raw); offset++ {
			r, e := probe.Generate(Query{Raw: raw[:offset], Eligible: true})
			if e != nil {
				return nil, e
			}
			if r.Status != "ok" {
				status = r.Status
			}
		}
		cases = append(cases, map[string]any{"id": fmt.Sprintf("stress-%d", i), "raw_bytes": len(raw), "status": status, "generate_calls": probe.GenerateCalls, "probe_lex_queries": probe.ProbeLexQueries, "best_calls": probe.BestCalls, "elapsed_ns": time.Since(start).Nanoseconds()})
	}
	if len(cases) != 20 {
		return nil, fmt.Errorf("calibration must have exactly20 cases")
	}
	return cases, nil
}
func verifyCalibration(o RunOptions, lock Lock, loaded *engine.Engine) (int, error) {
	if e := ValidateOutputRoot(o.Out); e != nil {
		return 2, e
	}
	if e := NewOutput(o.Out); e != nil {
		return 2, e
	}
	for _, check := range []struct {
		name string
		args []string
	}{{"verification-go.log", []string{"go", "test", "-count=1", "-tags=shiftreturneval", "./internal/shiftreturn", "./cmd/0c-shift-eval"}}, {"verification-python.log", []string{"python3", "-m", "unittest", "discover", "-s", "internal/shiftreturn", "-p", "test_*.py"}}} {
		command := exec.Command(check.args[0], check.args[1:]...)
		command.Env = append(os.Environ(), "SHIFTRETURN_DATA_INVENTORY=0")
		output, err := command.CombinedOutput()
		file, writeErr := os.OpenFile(filepath.Join(o.Out, check.name), os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0644)
		if writeErr != nil {
			return 2, writeErr
		}
		_, writeErr = file.Write(output)
		closeErr := file.Close()
		if writeErr != nil {
			return 2, writeErr
		}
		if closeErr != nil {
			return 2, closeErr
		}
		if err != nil {
			return 2, fmt.Errorf("required verification failed; see %s: %w", check.name, err)
		}
	}
	for _, condition := range []string{"M", "R", "C", "A"} {
		for _, raw := range append(append([]string{}, MandatoryLiteralRaw...), "x593gp ") {
			on, err := NewBaseline(loaded.Model.Lex, loaded.Model.LM, loaded.English, condition, nil, true)
			if err != nil {
				return 2, err
			}
			off, err := NewBaseline(loaded.Model.Lex, loaded.Model.LM, loaded.English, condition, nil, false)
			if err != nil {
				return 2, err
			}
			requests := []proto.Request{}
			for _, key := range "5j/ " {
				requests = append(requests, proto.Request{Type: proto.Key, Key: string(key)})
			}
			requests = append(requests, proto.Request{Type: proto.Toggle})
			for _, key := range raw {
				requests = append(requests, proto.Request{Type: proto.Key, Key: string(key)})
			}
			requests = append(requests, proto.Request{Type: proto.Key, Key: proto.KeyEnter})
			for _, request := range requests {
				a, b := on.Handle(request), off.Handle(request)
				if !reflect.DeepEqual(a.Snapshot, b.Snapshot) || a.App != b.App || a.View != b.View || StoreDigest(on.Model) != StoreDigest(off.Model) {
					return 2, fmt.Errorf("fixed public baseline observer parity differs")
				}
			}
		}
	}
	cases, e := Calibrate(loaded)
	if e != nil {
		return 2, e
	}
	if e = WriteRecord(filepath.Join(o.Out, "calibration.json"), map[string]any{"cases": cases, "quality": "not_run"}); e != nil {
		return 2, e
	}
	source := Sources{Version: Version, Kind: "sources", Demo: true}
	assignment := Assign(nil, source)
	assignment.FixturesSHA256 = HashBytes(nil)
	bytes, e := os.ReadFile(o.Lock)
	if e != nil {
		return 2, e
	}
	report, e := Summarize(Evidence{Sources: source, Assignment: assignment, Lock: lock, LockSHA256: HashBytes(bytes)})
	if e != nil {
		return 2, e
	}
	report.DiagnosticsPath = "calibration.json"
	if e = WriteRecord(filepath.Join(o.Out, "report.json"), report); e != nil {
		return 2, e
	}
	return 0, nil
}
