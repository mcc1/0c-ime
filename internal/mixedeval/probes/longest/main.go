//go:build mixedeval

// Independent diagnostic replay of the preregistered, unfitted longest suffix.
package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"github.com/0c-biggo/0c-ime/core/mixedparse"
	"github.com/0c-biggo/0c-ime/core/personal"
	"github.com/0c-biggo/0c-ime/core/proto"
	"github.com/0c-biggo/0c-ime/core/session"
	"github.com/0c-biggo/0c-ime/internal/datadir"
	"github.com/0c-biggo/0c-ime/internal/mixedeval"
	"os"
	"path/filepath"
)

func first(p []mixedparse.Segment) int { return mixedparse.FirstChinese(p) }
func main() {
	if len(os.Args) != 3 {
		panic("phase1 directory and new output directory required")
	}
	phase1 := os.Args[1]
	out := os.Args[2]
	if err := os.Mkdir(out, 0755); err != nil {
		panic(err)
	}
	if _, err := mixedeval.VerifyLock("testdata/mixed-boundary/fitting-run-lock.json"); err != nil {
		panic(err)
	}
	eng, err := datadir.Open("assets/data")
	if err != nil {
		panic(err)
	}
	fs, err := mixedeval.ReadFixtures("testdata/mixed-boundary/fixtures.jsonl")
	if err != nil {
		panic(err)
	}
	phase := map[string]mixedeval.RowResult{}
	f, err := os.Open(filepath.Join(phase1, "rows.jsonl"))
	if err != nil {
		panic(err)
	}
	scan := bufio.NewScanner(f)
	scan.Buffer(make([]byte, 65536), 8<<20)
	for scan.Scan() {
		var r mixedeval.RowResult
		if err := json.Unmarshal(scan.Bytes(), &r); err != nil {
			panic(err)
		}
		phase[r.ID] = r
	}
	f.Close()
	rf, err := os.Create(filepath.Join(out, "rows.jsonl"))
	if err != nil {
		panic(err)
	}
	defer rf.Close()
	enc := json.NewEncoder(rf)
	cn, ce, ln, le, n := 0, 0, 0, 0, 0
	for _, fx := range fs {
		base := phase[fx.ID]
		if base.Split != "test" || !base.Eligible {
			continue
		}
		ctx := mixedparse.Context{Layout: "standard", CustomTokens: fx.CustomTokens}
		if len(fx.Prelude) > 0 {
			b, e := session.NewMixedEvalBaseline(eng.Model.Lex, eng.Model.LM, eng.English, true, fx.CustomTokens)
			if e != nil {
				panic(e)
			}
			for _, ev := range fx.Prelude {
				if ev.Type != "key" {
					panic("key-only prelude required by diagnostic")
				}
				b.Handle(proto.Request{Type: proto.Key, Key: ev.Key})
			}
			o := b.Observe()
			b.Close()
			if o.OwnershipStatus != "mapped" {
				panic("prelude unmapped")
			}
			ctx = o.Context
			ctx.CustomTokens = fx.CustomTokens
		}
		model := personal.NewModel(eng.Model.Lex, eng.Model.LM)
		evaluator := mixedparse.Evaluator{Lex: model.Source(), Score: model, System: model, English: eng.English}
		machine := mixedparse.NewMachine(ctx)
		bad, partBad, converted := false, false, false
		assertions := map[int][]mixedeval.Assertion{}
		for _, a := range fx.Assertions {
			if a.Policy == "generator" && a.Stage == "final" {
				assertions[a.At] = append(assertions[a.At], a)
			}
		}
		for at, ev := range fx.Events {
			var previous []mixedparse.Segment
			if machine.Current != nil {
				previous = machine.Current.Shown
			}
			tr, e := machine.Step(ev, evaluator, previous, func(w mixedparse.Window, c []mixedparse.Candidate) (mixedparse.Candidate, error) {
				return mixedparse.Longest(c), nil
			})
			if e != nil {
				panic(e)
			}
			if fx.Intent == "literal" && tr.After != nil {
				for _, s := range tr.After.Shown {
					converted = converted || s.Kind == "chinese"
				}
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
					panic("assertion window")
				}
				if fx.Intent == "chinese" {
					firstOK, partOK := false, false
					for _, gold := range a.Accepted {
						firstOK = firstOK || first(w.Shown) == first(gold)
						partOK = partOK || mixedparse.Identity(w.Shown) == mixedparse.Identity(gold)
					}
					bad = bad || !firstOK
					partBad = partBad || !partOK
				}
			}
		}
		if base.Primary && fx.Intent != "ambiguous" {
			n++
			if fx.Intent == "chinese" {
				cn++
				if bad {
					ce++
				}
			}
			if fx.Intent == "literal" {
				ln++
				if converted {
					le++
				}
			}
		}
		if err := enc.Encode(map[string]any{"fixture_id": fx.ID, "intent": fx.Intent, "primary": base.Primary, "categories": base.Categories, "first_boundary_error": bad, "partition_error": partBad, "literal_any_event_conversion": converted}); err != nil {
			panic(err)
		}
	}
	r := map[string]any{"policy": "longest-valid-suffix", "history": "own causal longest display; unchanged preregistered generator/caps; unfitted diagnostic", "split": "test", "chinese_first_boundary_error": mixedeval.Rate(ce, cn), "literal_any_event_conversion": mixedeval.Rate(le, ln), "primary_unambiguous": n}
	b, _ := json.MarshalIndent(r, "", "  ")
	if err := os.WriteFile(filepath.Join(out, "report.json"), append(b, '\n'), 0644); err != nil {
		panic(err)
	}
	fmt.Println(string(b))
}
