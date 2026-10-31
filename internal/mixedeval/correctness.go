package mixedeval

import (
	"encoding/json"
	"fmt"
	"github.com/0c-biggo/0c-ime/core/layout"
	"github.com/0c-biggo/0c-ime/core/mixedparse"
	"github.com/0c-biggo/0c-ime/core/zhuyin"
)

type Correctness struct {
	Strings           int    `json:"strings"`
	KeyEvents         int    `json:"key_events"`
	TonePrefixes      int    `json:"tone_prefixes"`
	Candidates        int    `json:"candidates"`
	SpellingCases     int    `json:"spelling_cases"`
	InvariantFailures int    `json:"invariant_failures"`
	Alphabet          string `json:"alphabet"`
	MaxLength         int    `json:"max_length"`
}

// Exhaustive visits all contract strings including empty and every physical
// prefix. Each new tone is decoded twice, verifying deterministic full evidence.
func Exhaustive(e mixedparse.Evaluator) (Correctness, error) {
	out := Correctness{Alphabet: "asuj053467A_.", MaxLength: 5} // a,s,u,j,0,5,3,4,6,7,A,_,.
	var visit func(string, int) error
	visit = func(raw string, n int) error {
		out.Strings++
		for at := range raw {
			out.KeyEvents++
			if !mixedparse.Tone(raw[at]) {
				continue
			}
			out.TonePrefixes++
			r, err := e.Generate(mixedparse.Context{}, raw[:at+1], 0, nil)
			if err != nil {
				return err
			}
			r2, err := e.Generate(mixedparse.Context{}, raw[:at+1], 0, nil)
			if err != nil {
				return err
			}
			a, _ := json.Marshal(r)
			b, _ := json.Marshal(r2)
			if string(a) != string(b) {
				return fmt.Errorf("determinism at %q", raw[:at+1])
			}
			strict := 0
			lit := 0
			for _, c := range r.Candidates {
				out.Candidates++
				end := 0
				for _, s := range c.Segments {
					if s.Begin != end || s.End <= end {
						return fmt.Errorf("ownership at %q", raw)
					}
					end = s.End
				}
				if end != at+1 {
					return fmt.Errorf("coverage")
				}
				if c.Lane == "literal" {
					lit++
				}
				if c.Lane == "strict" && !c.Pruned {
					strict++
				}
			}
			if strict > 8 || lit != 1 {
				return fmt.Errorf("cap/literal")
			}
		}
		if n > 0 {
			for _, k := range out.Alphabet {
				if err := visit(raw+string(k), n-1); err != nil {
					return err
				}
			}
		}
		return nil
	}
	if err := visit("", 5); err != nil {
		out.InvariantFailures++
		return out, err
	}
	bodies := []rune{}
	for k := rune(33); k <= 126; k++ {
		s, ok := layout.Standard.Symbol(k)
		if ok && zhuyin.ClassOf(s) != zhuyin.Tone {
			bodies = append(bodies, k)
		}
	}
	var bodyVisit func(string, int) error
	bodyVisit = func(s string, n int) error {
		if s != "" {
			for _, key := range "3467" {
				tone, _ := layout.Standard.Symbol(key)
				got, ok := mixedparse.SpellExact(s, tone)
				out.SpellingCases++
				if ok && (layout.Standard.Keys(got) != s+string(key) || !got.Valid()) {
					return fmt.Errorf("strict roundtrip %q %q", s, got)
				}
			}
		}
		if n > 0 {
			for _, k := range bodies {
				if err := bodyVisit(s+string(k), n-1); err != nil {
					return err
				}
			}
		}
		return nil
	}
	if err := bodyVisit("", 3); err != nil {
		out.InvariantFailures++
		return out, err
	}
	return out, nil
}
