package main

import (
	"encoding/json"
	"fmt"
	"github.com/0c-biggo/0c-ime/core/lexicon"
	"github.com/0c-biggo/0c-ime/core/lm"
	"github.com/0c-biggo/0c-ime/core/mixedparse"
	"github.com/0c-biggo/0c-ime/core/personal"
	"github.com/0c-biggo/0c-ime/core/zhuyin"
	"github.com/0c-biggo/0c-ime/internal/datadir"
	"os"
)

type counts struct{ Strings, KeyEvents, ToneEvents, MachineSteps, Candidates, Failures int }

func check(e mixedparse.Evaluator) (counts, error) {
	var c counts
	var visit func(string, int) error
	visit = func(raw string, left int) error {
		c.Strings++
		a := mixedparse.NewMachine(mixedparse.Context{})
		b := mixedparse.NewMachine(mixedparse.Context{})
		old := map[int]string{}
		for at, k := range raw {
			c.KeyEvents++
			if mixedparse.Tone(byte(k)) {
				c.ToneEvents++
			}
			ev := mixedparse.Event{Type: "key", Key: string(k)}
			x, err := a.Step(ev, e, nil, nil)
			if err != nil {
				return err
			}
			y, err := b.Step(ev, e, nil, nil)
			if err != nil {
				return err
			}
			c.MachineSteps += 2
			xx, _ := json.Marshal(x)
			yy, _ := json.Marshal(y)
			if string(xx) != string(yy) {
				return fmt.Errorf("nondeterministic %q", raw[:at+1])
			}
			owned := ""
			for _, w := range a.Windows {
				owned += w.Raw
				id := mixedparse.Identity(w.Shown)
				if v, ok := old[w.Index]; ok && v != id {
					return fmt.Errorf("closed ownership changed")
				}
				old[w.Index] = id
			}
			if a.Current != nil {
				owned += a.Current.Raw
				if len(a.Current.Raw) > 64 {
					return fmt.Errorf("cap")
				}
			}
			if owned != raw[:at+1] {
				return fmt.Errorf("raw coverage %q %q", owned, raw[:at+1])
			}
			strict, lit := 0, 0
			for _, p := range x.Candidates {
				c.Candidates++
				end := 0
				for _, s := range p.Segments {
					if s.Begin != end || s.End <= end {
						return fmt.Errorf("candidate span")
					}
					end = s.End
				}
				if end != len(x.After.Raw) {
					return fmt.Errorf("candidate ownership")
				}
				if p.Lane == "strict" && !p.Pruned {
					strict++
				}
				if p.Lane == "literal" {
					lit++
				}
			}
			if strict > 8 || len(x.Candidates) > 0 && lit != 1 {
				return fmt.Errorf("protected lane")
			}
		}
		if left > 0 {
			for _, k := range "asuj053467A_." {
				if err := visit(raw+string(k), left-1); err != nil {
					return err
				}
			}
		}
		return nil
	}
	err := visit("", 5)
	if err != nil {
		c.Failures++
	}
	return c, err
}
func main() {
	eng, err := datadir.Open("assets/data")
	if err != nil {
		panic(err)
	}
	tiny := lexicon.NewMem()
	sys := lm.NewMem()
	for _, v := range []struct{ w, r string }{{"賺", "ㄓㄨㄢˋ"}, {"錢", "ㄑㄧㄢˊ"}, {"賺錢", "ㄓㄨㄢˋ ㄑㄧㄢˊ"}, {"你", "ㄋㄧˇ"}} {
		tiny.Add(lexicon.Entry{Surface: v.w, Reading: zhuyin.ParseReading(v.r), PRW: 1})
		sys.SetUni(v.w, .01)
	}
	out := map[string]any{}
	for _, kind := range []string{"tiny", "public"} {
		m := eng.Model
		if kind == "tiny" {
			m = personal.NewModel(tiny, sys)
		}
		c, err := check(mixedparse.Evaluator{Lex: m.Source(), Score: m, System: m, English: eng.English})
		out[kind] = c
		if err != nil {
			out["error"] = err.Error()
			b, _ := json.MarshalIndent(out, "", "  ")
			os.Stdout.Write(b)
			os.Exit(1)
		}
	}
	b, _ := json.MarshalIndent(out, "", "  ")
	os.Stdout.Write(append(b, '\n'))
}
