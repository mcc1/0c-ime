//go:build shiftreturneval

package shiftreturn

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
)

func demoFixtures(t *testing.T) []Fixture {
	t.Helper()
	fs, _, e := ReadFixtures("../../design/shift-return-v2/examples.jsonl")
	if e != nil {
		t.Fatal(e)
	}
	return fs
}
func TestSchemaReviewedCopy(t *testing.T) {
	b, e := os.ReadFile("../../design/shift-return-v2/schema.json")
	if e != nil {
		t.Fatal(e)
	}
	if string(b) != schemaJSON {
		t.Fatal("embedded schema differs from reviewed schema")
	}
}
func TestSchemaExamples(t *testing.T) {
	fs := demoFixtures(t)
	if len(fs) != 5 {
		t.Fatal(len(fs))
	}
	for _, name := range []string{"bank-example.json", "canonical-model-sample.json", "worked-report.json"} {
		b, e := os.ReadFile("../../design/shift-return-v2/" + name)
		if e != nil {
			t.Fatal(e)
		}
		var v any
		if e = Decode(b, &v); e != nil {
			t.Fatalf("%s: %v", name, e)
		}
	}
	for _, name := range []string{"runtime-snapshots.jsonl", "occurrences.jsonl", "metadata-fixtures.jsonl", "metadata-rows.jsonl"} {
		b, e := os.ReadFile("../../design/shift-return-v2/" + name)
		if e != nil {
			t.Fatal(e)
		}
		for _, line := range strings.Split(strings.TrimSuffix(string(b), "\n"), "\n") {
			var v any
			if e = Decode([]byte(line), &v); e != nil {
				t.Fatalf("%s: %v", name, e)
			}
		}
	}
	b, e := os.ReadFile("../../design/shift-return-v2/generator-examples.json")
	if e != nil {
		t.Fatal(e)
	}
	var cases struct {
		Cases []struct {
			Diagnostic json.RawMessage `json:"diagnostic"`
		} `json:"cases"`
	}
	if e = json.Unmarshal(b, &cases); e != nil {
		t.Fatal(e)
	}
	for _, c := range cases.Cases {
		var v GeneratorDiagnostic
		if e = Decode(c.Diagnostic, &v); e != nil {
			t.Fatal(e)
		}
	}
}
func TestSchemaRejectMalformed(t *testing.T) {
	f := demoFixtures(t)[0]
	base := string(Canonical(f))
	tests := []string{
		strings.Replace(base, `"key":"m"`, `"missing":"m"`, 1),
		strings.Replace(base, `"key":"m"`, `"key":"mm"`, 1),
		strings.Replace(base, `"type":"shift"`, `"type":"shift","on":false`, 1),
		strings.Replace(base, `"kind":"fixture"`, `"kind":"fixture","secret":true`, 1),
		strings.Replace(base, `"demo":true`, `"demo":true,"demo":false`, 1),
		base + " {}",
	}
	for i, b := range tests {
		if e := Decode([]byte(b), new(Fixture)); e == nil {
			t.Fatalf("malformed case %d accepted", i)
		}
	}
	for _, mutate := range []func(*Fixture){
		func(f *Fixture) { f.Events = f.Events[1:] },
		func(f *Fixture) { f.ExpectedRaw += "x" },
		func(f *Fixture) { f.Gold[0].Segments[1].Begin++ },
		func(f *Fixture) { f.Gold[0].Segments[1].Reading = "ㄨㄢˋ" },
		func(f *Fixture) { f.Gold[0].AcceptedSurfaces = nil },
		func(f *Fixture) { f.Intent = "literal" },
	} {
		var copy Fixture
		if e := json.Unmarshal([]byte(base), &copy); e != nil {
			t.Fatal(e)
		}
		mutate(&copy)
		if e := Validate(copy); e == nil {
			t.Fatal("malformed semantic fixture accepted")
		}
	}
	if e := ValidateFixtures([]Fixture{f, f}); e == nil {
		t.Fatal("duplicate id accepted")
	}
}
func TestSchemaWrongTypedKind(t *testing.T) {
	f := demoFixtures(t)[0]
	for _, out := range []any{new(Sources), new(Model)} {
		if err := Decode(Canonical(f), out); err == nil {
			t.Fatalf("fixture accepted into %T", out)
		}
	}
	var generic any
	if err := Decode(Canonical(f), &generic); err != nil {
		t.Fatal(err)
	}
}
func TestSchemaCanonicalSeparators(t *testing.T) {
	cases := []struct{ input, want string }{
		{`\u2028`, `"\\u2028"`}, {`\u2029`, `"\\u2029"`},
		{"\u2028", "\"\u2028\""}, {"\u2029", "\"\u2029\""},
		{"\\\u2028", "\"\\\\\u2028\""},
	}
	for _, c := range cases {
		if got := string(Canonical(c.input)); got != c.want {
			t.Fatalf("input %q: got %q want Python %q", c.input, got, c.want)
		}
	}
}
