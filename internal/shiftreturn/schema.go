//go:build shiftreturneval

package shiftreturn

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"reflect"
	"regexp"
	"strings"
)

var exchangeSchema = func() map[string]any {
	var s map[string]any
	if err := json.Unmarshal([]byte(schemaJSON), &s); err != nil {
		panic(err)
	}
	return s
}()

// Decode verifies the strict exchange shape before populating a typed record.
// Duplicate JSON keys are rejected, including within nested objects.
func Decode(data []byte, out any) error {
	d := json.NewDecoder(bytes.NewReader(data))
	v, err := jsonValue(d)
	if err != nil {
		return err
	}
	if _, err = d.Token(); err != io.EOF {
		return fmt.Errorf("trailing JSON: %v", err)
	}
	if err = checkSchema(exchangeSchema, v, "$"); err != nil {
		return err
	}
	kinds := map[reflect.Type]string{reflect.TypeOf((*SelectedModelManifest)(nil)): "selected_model_manifest",
		reflect.TypeOf((*Fixture)(nil)): "fixture", reflect.TypeOf((*Sources)(nil)): "sources", reflect.TypeOf((*Query)(nil)): "query", reflect.TypeOf((*Hypothesis)(nil)): "candidate", reflect.TypeOf((*Snapshot)(nil)): "snapshot", reflect.TypeOf((*Model)(nil)): "model", reflect.TypeOf((*Assignment)(nil)): "assignment", reflect.TypeOf((*Lock)(nil)): "lock", reflect.TypeOf((*Row)(nil)): "row", reflect.TypeOf((*Report)(nil)): "report", reflect.TypeOf((*BankTask)(nil)): "bank_task", reflect.TypeOf((*TrainingObservation)(nil)): "training_observation", reflect.TypeOf((*CandidateOccurrence)(nil)): "candidate_occurrence", reflect.TypeOf((*GeneratorDiagnostic)(nil)): "generator_diagnostic",
	}
	if expected, ok := kinds[reflect.TypeOf(out)]; ok {
		if v.(map[string]any)["kind"] != expected {
			return fmt.Errorf("record kind must be %s for %T", expected, out)
		}
	} else if _, ok := out.(*any); !ok {
		return fmt.Errorf("unsupported exchange target %T", out)
	}
	if err := json.Unmarshal(data, out); err != nil {
		return err
	}
	if report, ok := out.(*Report); ok {
		for condition, metrics := range report.Conditions {
			for name, m := range metrics {
				if err := validateMetric(m); err != nil {
					return fmt.Errorf("%s/%s: %w", condition, name, err)
				}
			}
		}
	}
	return nil
}
func jsonValue(d *json.Decoder) (any, error) {
	t, e := d.Token()
	if e != nil {
		return nil, e
	}
	switch x := t.(type) {
	case json.Delim:
		if x == '{' {
			m := map[string]any{}
			for d.More() {
				k, e := d.Token()
				if e != nil {
					return nil, e
				}
				s := k.(string)
				if _, ok := m[s]; ok {
					return nil, fmt.Errorf("duplicate key %q", s)
				}
				v, e := jsonValue(d)
				if e != nil {
					return nil, e
				}
				m[s] = v
			}
			_, e = d.Token()
			return m, e
		}
		if x == '[' {
			a := []any{}
			for d.More() {
				v, e := jsonValue(d)
				if e != nil {
					return nil, e
				}
				a = append(a, v)
			}
			_, e = d.Token()
			return a, e
		}
		return nil, fmt.Errorf("unexpected delimiter %q", x)
	default:
		return t, nil
	}
}
func checkSchema(s map[string]any, v any, path string) error {
	fail := func(reason string) error { return fmt.Errorf("%s: %s", path, reason) }
	if r, ok := s["$ref"].(string); ok {
		return checkSchema(exchangeSchema["$defs"].(map[string]any)[strings.TrimPrefix(r, "#/$defs/")].(map[string]any), v, path)
	}
	if c, ok := s["const"]; ok && !reflect.DeepEqual(c, v) {
		return fail("constant differs")
	}
	if es, ok := s["enum"].([]any); ok {
		found := false
		for _, x := range es {
			found = found || reflect.DeepEqual(x, v)
		}
		if !found {
			return fail("invalid enum")
		}
	}
	if choices, ok := s["oneOf"].([]any); ok {
		n := 0
		for _, c := range choices {
			if checkSchema(c.(map[string]any), v, path) == nil {
				n++
			}
		}
		if n != 1 {
			return fail(fmt.Sprintf("matched %d oneOf alternatives", n))
		}
	}
	if ts, ok := s["type"]; ok {
		types, ok := ts.([]any)
		if !ok {
			types = []any{ts}
		}
		valid := false
		for _, t := range types {
			switch t {
			case "null":
				valid = valid || v == nil
			case "object":
				_, ok := v.(map[string]any)
				valid = valid || ok
			case "array":
				_, ok := v.([]any)
				valid = valid || ok
			case "string":
				_, ok := v.(string)
				valid = valid || ok
			case "boolean":
				_, ok := v.(bool)
				valid = valid || ok
			case "number", "integer":
				x, ok := v.(float64)
				valid = valid || ok && !math.IsNaN(x) && !math.IsInf(x, 0) && (t == "number" || math.Trunc(x) == x)
			}
		}
		if !valid {
			return fail("wrong type")
		}
	}
	if x, ok := v.(float64); ok {
		if n, ok := s["minimum"].(float64); ok && x < n {
			return fail("below minimum")
		}
		if n, ok := s["maximum"].(float64); ok && x > n {
			return fail("above maximum")
		}
	}
	if x, ok := v.(string); ok {
		if p, ok := s["pattern"].(string); ok && !regexp.MustCompile(p).MatchString(x) {
			return fail("pattern mismatch")
		}
	}
	if a, ok := v.([]any); ok {
		if n, ok := s["minItems"].(float64); ok && len(a) < int(n) {
			return fail("too few items")
		}
		if n, ok := s["maxItems"].(float64); ok && len(a) > int(n) {
			return fail("too many items")
		}
		if s["uniqueItems"] == true {
			for i := range a {
				for j := 0; j < i; j++ {
					if reflect.DeepEqual(a[i], a[j]) {
						return fail("duplicate item")
					}
				}
			}
		}
		if item, ok := s["items"].(map[string]any); ok {
			for i, x := range a {
				if e := checkSchema(item, x, fmt.Sprintf("%s[%d]", path, i)); e != nil {
					return e
				}
			}
		}
	}
	if m, ok := v.(map[string]any); ok {
		if rs, ok := s["required"].([]any); ok {
			for _, k := range rs {
				if _, ok := m[k.(string)]; !ok {
					return fail("missing " + k.(string))
				}
			}
		}
		props, _ := s["properties"].(map[string]any)
		for k, x := range m {
			if p, ok := props[k].(map[string]any); ok {
				if e := checkSchema(p, x, path+"."+k); e != nil {
					return e
				}
			} else if s["additionalProperties"] == false {
				return fail("unknown " + k)
			} else if p, ok := s["additionalProperties"].(map[string]any); ok {
				if e := checkSchema(p, x, path+"."+k); e != nil {
					return e
				}
			}
		}
	}
	if cs, ok := s["allOf"].([]any); ok {
		for _, c := range cs {
			if e := checkSchema(c.(map[string]any), v, path); e != nil {
				return e
			}
		}
	}
	if c, ok := s["if"].(map[string]any); ok && checkSchema(c, v, path) == nil {
		if c, ok := s["then"].(map[string]any); ok {
			return checkSchema(c, v, path)
		}
	}
	return nil
}
