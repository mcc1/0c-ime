package mixedeval

import (
	"encoding/json"
	"github.com/0c-biggo/0c-ime/core/mixedparse"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFormatExamplesSemantic(t *testing.T) {
	rows, err := ReadFixtures("../../design/mixed-input-experiment/examples.jsonl")
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 5 {
		t.Fatal(len(rows))
	}
	t.Logf("semantic format examples=%d (not data adequacy)", len(rows))
}
func TestPartitionNegatives(t *testing.T) {
	bad := [][]mixedparse.Segment{{{Kind: "literal", Begin: 0, End: 6, Text: "wrong"}}, {{Kind: "chinese", Begin: 0, End: 6, Reading: "ㄓㄨㄢˋ"}}, {{Kind: "literal", Begin: 0, End: 2, Text: "WQ"}, {Kind: "chinese", Begin: 2, End: 6, Reading: "ㄨㄢˋ"}}, {{Kind: "literal", Begin: 0, End: 2, Text: "WQ"}, {Kind: "chinese", Begin: 2, End: 6, Reading: "ㄓㄨㄢ"}}, {{Kind: "literal", Begin: 0, End: 1, Text: "W"}, {Kind: "literal", Begin: 2, End: 6, Text: "5j04"}}}
	for _, p := range bad {
		if validatePartition("WQ5j04", p) == nil {
			t.Fatal("accepted invalid", p)
		}
	}
}
func TestAssignmentDeterminismAndOverlap(t *testing.T) {
	dir := t.TempDir()
	src, err := os.ReadFile("../../design/mixed-input-experiment/examples.jsonl")
	if err != nil {
		t.Fatal(err)
	}
	fixture := filepath.Join(dir, "f.jsonl")
	alias := filepath.Join(dir, "a.json")
	assignment := filepath.Join(dir, "s.json")
	os.WriteFile(fixture, src, 0600)
	os.WriteFile(alias, []byte("{\"version\":\"mixed-family-v1\",\"aliases\":{}}\n"), 0600)
	a, err := Structure("assign", fixture, alias, assignment)
	if err != nil {
		t.Fatal(err)
	}
	if a.PilotAdequate || a.ReleaseAdequate {
		t.Fatal("format examples passed adequacy")
	}
	b, err := Structure("validate", fixture, alias, assignment)
	if err != nil {
		t.Fatal(err)
	}
	aa, _ := json.Marshal(a)
	bb, _ := json.Marshal(b)
	if string(aa) != string(bb) {
		t.Fatal("assignment unstable")
	}
	if _, err := Structure("assign", fixture, alias, assignment); err == nil {
		t.Fatal("overwrote assignment")
	}
	raw, _ := os.ReadFile(assignment)
	var as Assignment
	json.Unmarshal(raw, &as)
	if as.Rows["chinese-hello"].GroupID != as.Rows["ambiguous-wq"].GroupID || as.Rows["selection-original"].GroupID != as.Rows["ambiguous-wq"].GroupID {
		t.Fatal("transitive prefix/word edges")
	}
	raw = append(raw, ' ')
	os.WriteFile(assignment, raw, 0600)
	if _, err := Structure("validate", fixture, alias, assignment); err == nil {
		t.Fatal("noncanonical assignment")
	}
}

func TestFinalizerAndNewCompositionSemantic(t *testing.T) {
	rows, err := ReadFixtures("../../design/mixed-input-experiment/examples.jsonl")
	if err != nil {
		t.Fatal(err)
	}
	r := rows[1]
	r.Events = append(r.Events, mixedparse.Event{Type: "key", Key: "A"})
	if ValidateSemantic(r) == nil {
		t.Fatal("unfinished trailing composition accepted")
	}
	r.Events = append(r.Events, mixedparse.Event{Type: "finalize", Reason: "enter"})
	r.Assertions = append(r.Assertions, Assertion{At: len(r.Events) - 2, Window: 0, Policy: "generator", Stage: "final", Accepted: [][]mixedparse.Segment{{{Kind: "literal", Begin: 0, End: 1, Text: "A"}}}})
	if err := ValidateSemantic(r); err != nil {
		t.Fatal("new composition not reset", err)
	}
}

func TestStrictStructureNegatives(t *testing.T) {
	src, err := os.ReadFile("../../design/mixed-input-experiment/examples.jsonl")
	if err != nil {
		t.Fatal(err)
	}
	line := strings.Split(string(src), "\n")[0]
	for name, bad := range map[string]string{"duplicate_key": strings.Replace(line, "{", "{\"id\":\"duplicate\",", 1), "unknown": strings.Replace(line, "{", "{\"unknown\":1,", 1), "nonfinite": strings.Replace(line, "{", "{\"unknown\":NaN,", 1), "backtick": strings.Replace(line, "\"key\":\"h\"", "\"key\":\"`\"", 1), "multibyte_event": strings.Replace(line, "\"key\":\"h\"", "\"key\":\"hello\"", 1), "missing_intent": strings.Replace(line, "\"intent\":\"chinese\",", "", 1)} {
		t.Run(name, func(t *testing.T) {
			dir := t.TempDir()
			f := filepath.Join(dir, "f.jsonl")
			a := filepath.Join(dir, "a.json")
			assignment := filepath.Join(dir, "s.json")
			os.WriteFile(f, []byte(bad+"\n"), 0600)
			os.WriteFile(a, []byte("{\"version\":\"mixed-family-v1\",\"aliases\":{}}\n"), 0600)
			if _, err := Structure("assign", f, a, assignment); err == nil {
				t.Fatal("accepted invalid JSON/schema")
			}
		})
	}
}
