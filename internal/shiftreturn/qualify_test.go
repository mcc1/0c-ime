//go:build shiftreturneval

package shiftreturn

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func bankFixture(task BankTask) Fixture {
	f := Fixture{Version: Version, Kind: "fixture", Demo: task.Demo, ID: "task:" + task.ID, PrototypeID: task.PrototypeID, Intent: task.Intent, Category: task.Category, Stratum: task.Stratum, ContextClass: task.ContextClass, SourceRef: task.SourceRef, SourcePriority: int(task.SourcePriority), Families: task.Families, PreludeEvents: task.PreludeEvents, ExpectedPrelude: task.ExpectedPrelude, ExpectedRaw: task.Raw, Gold: task.Gold, Events: []Event{{Type: "shift"}}}
	for _, key := range task.Raw {
		f.Events = append(f.Events, Event{Type: "key", Key: string(key)})
	}
	f.Events = append(f.Events, Event{Type: "enter"})
	return f
}
func TestSchemaBankAnnotationSourceChain(t *testing.T) {
	b, err := os.ReadFile("../../design/shift-return-v2/bank-example.json")
	if err != nil {
		t.Fatal(err)
	}
	var task BankTask
	if err = Decode(b, &task); err != nil {
		t.Fatal(err)
	}
	// Design examples are exchange demonstrations, not qualified category labels.
	// The production predicate makes initial5 a digit_initial task.
	task.Category = "digit_initial"
	task.ContextClass = "none"
	f := bankFixture(task)
	cache := filepath.Join(t.TempDir(), "bank.jsonl")
	bytes := append(Canonical(task), '\n')
	if err = os.WriteFile(cache, bytes, 0644); err != nil {
		t.Fatal(err)
	}
	value := map[string]any{"version": Version, "kind": "sources", "demo": true, "entries": []any{map[string]any{"id": task.SourceRef, "kind": "synthetic-intent", "reference": "hypothetical unit fixture only", "version": "demo", "sha256": HashBytes(bytes), "license": "hypothetical unit fixture only", "annotation_passes": 2, "adjudication_log": "demo-only", "annotator_ids": []string{"demo-annotator-1", "demo-annotator-2"}, "cached_path": cache}}, "seen_prefixes": []string{}, "seen_contents": []string{}, "seen_raw": []string{}}
	var sources Sources
	if err = Decode(Canonical(value), &sources); err != nil {
		t.Fatal(err)
	}
	p := tinyProbe(false)
	english := toyEnglish{"microsoft": true}
	if err = ValidateBank([]Fixture{f}, sources, []BankTask{task}, p.Lex, english, false); err != nil {
		t.Fatal(err)
	}
	for _, mutate := range []func(*BankTask){func(t *BankTask) { t.Annotations[1].AnnotatorID = t.Annotations[0].AnnotatorID }, func(t *BankTask) { t.Annotations[1].LabelsSHA256 = HashBytes([]byte("different")) }, func(t *BankTask) { t.Annotations[1].Accepted = false }, func(t *BankTask) { t.ContextClass = "contrast" }} {
		var copy BankTask
		if err = json.Unmarshal(Canonical(task), &copy); err != nil {
			t.Fatal(err)
		}
		mutate(&copy)
		if err = ValidateBank([]Fixture{bankFixture(copy)}, sources, []BankTask{copy}, p.Lex, english, false); err == nil {
			t.Fatal("invalid annotation/predicate accepted")
		}
	}
	if err = ValidateBank([]Fixture{f}, sources, []BankTask{task}, p.Lex, english, true); err == nil {
		t.Fatal("demo became formal data")
	}
	if ok, _ := sampleQualification([]Fixture{f}, []BankTask{task}, Assign([]Fixture{f}, sources)); ok {
		t.Fatal("single demo qualified")
	}
}
