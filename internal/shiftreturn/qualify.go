//go:build shiftreturneval

package shiftreturn

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"github.com/0c-biggo/0c-ime/internal/datadir"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"

	"github.com/0c-biggo/0c-ime/core/lexicon"
	"github.com/0c-biggo/0c-ime/core/zhuyin"
)

type InsufficientError struct{ Reason string }

func (e *InsufficientError) Error() string { return e.Reason }
func ReadBank(path string) ([]BankTask, error) {
	b, e := os.ReadFile(path)
	if e != nil {
		return nil, e
	}
	out := []BankTask{}
	scanner := bufio.NewScanner(bytes.NewReader(b))
	scanner.Buffer(make([]byte, 4096), 16<<20)
	for scanner.Scan() {
		var task BankTask
		if e = Decode(scanner.Bytes(), &task); e != nil {
			return nil, e
		}
		out = append(out, task)
	}
	return out, scanner.Err()
}
func ValidateBank(fs []Fixture, src Sources, bank []BankTask, lex lexicon.Source, english EnglishLookup, formal bool) error {
	if formal && src.Demo {
		return fmt.Errorf("formal sources cannot be demo")
	}
	entries := map[string]int{}
	for i, s := range src.Entries {
		if _, ok := entries[s.ID]; ok {
			return fmt.Errorf("duplicate source %s", s.ID)
		}
		entries[s.ID] = i
		if s.License == "" || s.Reference == "" || s.Version == "" || len(s.AnnotatorIDs) != 2 || s.AnnotatorIDs[0] == s.AnnotatorIDs[1] {
			return &InsufficientError{"source provenance/independent annotation missing"}
		}
		b, e := os.ReadFile(s.CachedPath)
		if e != nil {
			return e
		}
		if HashBytes(b) != s.SHA256 {
			return fmt.Errorf("source cache hash mismatch: %s", s.CachedPath)
		}
		if s.Kind == "synthetic-intent" {
			cached, err := ReadBank(s.CachedPath)
			if err != nil {
				return err
			}
			if !reflect.DeepEqual(cached, bank) {
				return fmt.Errorf("synthetic source cache is not the frozen bank")
			}
		}
		if formal && s.AdjudicationLog != "" {
			if _, err := os.ReadFile(s.AdjudicationLog); err != nil {
				return fmt.Errorf("frozen adjudication log missing: %w", err)
			}
		}
	}
	tasks := map[string]BankTask{}
	prompts := []string{}
	for _, task := range bank {
		if _, ok := tasks[task.ID]; ok {
			return fmt.Errorf("duplicate bank task %s", task.ID)
		}
		tasks[task.ID] = task
		prompts = append(prompts, task.Prompt)
		if formal && task.Demo {
			return fmt.Errorf("formal bank cannot be demo")
		}
	}
	// Python's pinned Unicode database supplies NFC; no heuristic character table.
	cmd := exec.Command("python3", "-c", "import sys,json,unicodedata; a=json.load(sys.stdin);sys.exit(0 if all(x==unicodedata.normalize('NFC',x.replace('\\r\\n','\\n')) for x in a) else 2)")
	cmd.Stdin = bytes.NewReader(Canonical(prompts))
	if e := cmd.Run(); e != nil {
		return fmt.Errorf("bank prompts not canonical NFC/LF: %w", e)
	}
	used := map[string]bool{}
	for _, f := range fs {
		if formal && f.Demo {
			return fmt.Errorf("formal fixture cannot be demo")
		}
		if f.Intent == "compat" {
			continue
		}
		task, ok := tasks[strings.TrimPrefix(f.ID, "task:")]
		if !ok || f.ID != "task:"+task.ID {
			return &InsufficientError{"fixture lacks frozen bank task: " + f.ID}
		}
		used[task.ID] = true
		sourceIndex, ok := entries[task.SourceRef]
		if !ok {
			return &InsufficientError{"bank source missing: " + task.SourceRef}
		}
		source := src.Entries[sourceIndex]
		if (source.Kind == "public" && task.SourcePriority != 0) || (source.Kind == "synthetic-intent" && task.SourcePriority != 1) {
			return fmt.Errorf("source priority/kind differs")
		}
		if len(task.Annotations) != 2 || task.Annotations[0].AnnotatorID == task.Annotations[1].AnnotatorID {
			return &InsufficientError{"independent annotation missing"}
		}
		labels := HashBytes(Canonical(map[string]any{"intent": task.Intent, "raw": task.Raw, "gold": task.Gold}))
		for _, a := range task.Annotations {
			if formal && a.AdjudicationRef != "" && source.AdjudicationLog == "" {
				return &InsufficientError{"adjudication reference has no frozen source log"}
			}
			if !a.Accepted || !a.ReadingVerified || a.Intent != task.Intent || a.LabelsSHA256 != labels {
				return &InsufficientError{"annotation labels not independently accepted: " + task.ID}
			}
			found := false
			for _, id := range source.AnnotatorIDs {
				found = found || id == a.AnnotatorID
			}
			if !found {
				return fmt.Errorf("annotation/source identity differs")
			}
		}
		if task.VariantIndex == 0 {
			prototype := HashBytes(Canonical(map[string]any{"source_ref": task.SourceRef, "intent": task.Intent, "prompt": task.Prompt, "raw": task.Raw, "gold": task.Gold}))[:24]
			if task.ID != prototype || task.PrototypeID != prototype {
				return fmt.Errorf("prototype hash differs: %s", task.ID)
			}
		} else {
			base, ok := tasks[task.PrototypeID]
			if !ok || base.VariantIndex != 0 || base.PrototypeID != task.PrototypeID || base.SourceRef != task.SourceRef || base.Intent != task.Intent {
				return fmt.Errorf("variant prototype join differs")
			}
		}
		expected := Fixture{Version: Version, Kind: "fixture", Demo: task.Demo, ID: "task:" + task.ID, PrototypeID: task.PrototypeID, Intent: task.Intent, Category: task.Category, Stratum: task.Stratum, ContextClass: task.ContextClass, SourceRef: task.SourceRef, SourcePriority: int(task.SourcePriority), Families: task.Families, PreludeEvents: task.PreludeEvents, ExpectedPrelude: task.ExpectedPrelude, ExpectedRaw: task.Raw, Gold: task.Gold, Events: []Event{{Type: "shift"}}}
		for _, key := range task.Raw {
			expected.Events = append(expected.Events, Event{Type: "key", Key: string(key)})
		}
		expected.Events = append(expected.Events, Event{Type: "enter"})
		if !reflect.DeepEqual(expected, f) {
			return fmt.Errorf("bank/fixture label copy differs: %s", f.ID)
		}
		if task.Category == "typo" && len(task.DesignatedTypoSpans) == 0 {
			return fmt.Errorf("typo task without designated span")
		}
		if task.Category != "typo" && len(task.DesignatedTypoSpans) != 0 {
			return fmt.Errorf("non-typo has typo spans")
		}
		for _, span := range task.DesignatedTypoSpans {
			if span.Begin < 0 || span.End <= span.Begin || span.End > len(task.Raw) {
				return fmt.Errorf("invalid typo span")
			}
		}
		if source.Kind == "synthetic-intent" && len(task.SourceEdits) != 0 {
			return fmt.Errorf("synthetic task has public source edits")
		}
		for _, edit := range task.SourceEdits {
			if edit.Begin < 0 || edit.End < edit.Begin {
				return fmt.Errorf("invalid source edit")
			}
		}
		if lex == nil || english == nil {
			return &InsufficientError{"bank dictionary predicate unavailable"}
		}
		if e := qualifyLabels(task, bank, lex, english); e != nil {
			return e
		}
	}
	// Every accepted task must be materialized; rejected evidence stays in bank.
	for _, task := range bank {
		accepted := len(task.Annotations) == 2 && task.Annotations[0].Accepted && task.Annotations[1].Accepted
		if accepted && !used[task.ID] {
			return fmt.Errorf("accepted bank task omitted from fixtures: %s", task.ID)
		}
	}
	return nil
}
func firstChinese(g Gold) (int, int) {
	for _, s := range g.Segments {
		if s.Kind == "chinese" {
			return s.Begin, s.End
		}
	}
	return -1, -1
}
func qualifyLabels(t BankTask, bank []BankTask, lex lexicon.Source, english EnglishLookup) error {
	if len(t.Gold) == 0 {
		return fmt.Errorf("bank task has no gold")
	}
	gold := t.Gold[0]
	begin, end := firstChinese(gold)
	if t.Intent == "return" {
		n := 0
		readings := []zhuyin.Syllable{}
		for _, s := range gold.Segments {
			if s.Kind == "chinese" {
				n++
				readings = append(readings, zhuyin.Syllable(s.Reading))
			}
		}
		stratum := "1"
		if n >= 2 {
			stratum = "2-4"
		}
		if n >= 5 {
			stratum = "5-16"
		}
		if n >= 17 {
			stratum = "17-64"
		}
		if n > 64 || len(t.Raw) > 512 {
			return fmt.Errorf("main return outside acquired length budget")
		}
		if t.Stratum != stratum {
			return fmt.Errorf("return stratum differs")
		}
		prefix := t.Raw[:begin]
		category := "known"
		if !english.Has(prefix) {
			category = "unknown"
		}
		if strings.IndexFunc(prefix, func(r rune) bool { return r >= 'A' && r <= 'Z' }) >= 0 {
			category = "uppercase"
		}
		if begin < len(t.Raw) && strings.ContainsRune("012589", rune(t.Raw[begin])) {
			category = "digit_initial"
		}
		if strings.IndexFunc(prefix, func(r rune) bool { return !(r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z') }) >= 0 {
			category = "identifier"
		}
		if len(t.DesignatedTypoSpans) > 0 {
			category = "typo"
			for _, span := range t.DesignatedTypoSpans {
				if span.End > begin {
					return fmt.Errorf("typo span crosses Chinese start")
				}
			}
		}
		if t.Category != category {
			return fmt.Errorf("return category differs: %s got%s want%s", t.ID, t.Category, category)
		}
		contrast := false
		collision := false
		for _, other := range bank {
			if other.ID == t.ID || len(other.Gold) == 0 {
				continue
			}
			ob, _ := firstChinese(other.Gold[0])
			different := other.Intent == "literal" || ob != begin
			if different && other.Raw == t.Raw {
				collision = true
			}
			if different && other.Raw != t.Raw && end <= len(other.Raw) && other.Raw[:end] == t.Raw[:end] && n >= 2 {
				for length := 2; length <= min(8, n); length++ {
					if len(lex.Exact(readings[:length])) > 0 {
						contrast = true
					}
				}
			}
		}
		// Identical collisions cannot support distinguishing context evidence.
		expected := "none"
		if contrast {
			expected = "contrast"
		}
		if collision {
			expected = "identical_collision"
		}
		if t.ContextClass != expected {
			return fmt.Errorf("contrast/collision predicate differs: %s got%s want%s", t.ID, t.ContextClass, expected)
		}
	} else {
		knownCategory := false
		for _, category := range []string{"ordinary_english", "account", "phonetic_account", "model_version", "numeric_date_time", "url_path", "identifier", "random_ascii"} {
			knownCategory = knownCategory || t.Category == category
		}
		if !knownCategory {
			return fmt.Errorf("unknown literal category")
		}
		n := len(t.Raw)
		stratum := "1-8"
		if n >= 9 {
			stratum = "9-32"
		}
		if n >= 33 {
			stratum = "33-128"
		}
		if n >= 129 {
			stratum = "129-512"
		}
		if n > 512 || t.Stratum != stratum {
			return fmt.Errorf("literal stratum differs")
		}
	}
	return nil
}

func qualifyEvidenceSources(e Evidence) (bool, error) {
	if e.Assignment.Demo || e.Sources.Demo {
		return false, nil
	}
	dataPath := ""
	for path := range e.Lock.Files {
		if filepath.Base(path) == "manifest.json" {
			directory := filepath.Dir(path)
			if _, err := os.Stat(filepath.Join(directory, "english", "manifest.json")); err == nil {
				dataPath = directory
				break
			}
		}
	}
	if dataPath == "" {
		return false, nil
	}
	loaded, err := datadir.Open(dataPath)
	if err != nil {
		return false, err
	}
	if loaded.EnglishPack == "" {
		return false, fmt.Errorf("prototype English fallback forbidden")
	}
	if err = ValidateBank(e.Fixtures, e.Sources, e.Bank, loaded.Model.Lex, loaded.English, true); err != nil {
		var insufficient *InsufficientError
		if errors.As(err, &insufficient) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}
func sampleQualification(fs []Fixture, bank []BankTask, a Assignment) (bool, string) {
	returnCategories := []string{"typo", "identifier", "digit_initial", "uppercase", "unknown", "known"}
	literalCategories := []string{"ordinary_english", "account", "phonetic_account", "model_version", "numeric_date_time", "url_path", "identifier", "random_ascii"}
	totals := map[string]int{}
	counts := map[string]map[string]int{}
	variants := map[string]map[int]bool{}
	for _, task := range bank {
		if variants[task.PrototypeID] == nil {
			variants[task.PrototypeID] = map[int]bool{}
		}
		if variants[task.PrototypeID][task.VariantIndex] {
			return false, "duplicate prototype variant index"
		}
		variants[task.PrototypeID][task.VariantIndex] = true
		if task.VariantIndex != 0 {
			continue
		}
		totals[task.Intent]++
		if totals[task.Intent] > 3000 {
			return false, "acquisition prototype budget exceeded"
		}
		key := task.Intent + ":" + task.Category
		if counts[key] == nil {
			counts[key] = map[string]int{}
		}
		counts[key][task.Stratum]++
	}
	for _, category := range returnCategories {
		for _, stratum := range []string{"1", "2-4", "5-16", "17-64"} {
			if counts["return:"+category][stratum] != 125 {
				return false, "fixed return acquisition quota incomplete/exceeded"
			}
		}
	}
	for _, category := range literalCategories {
		for i, stratum := range []string{"1-8", "9-32", "33-128", "129-512"} {
			want := 94
			if i == 3 {
				want = 93
			}
			if counts["literal:"+category][stratum] != want {
				return false, "fixed literal acquisition quota incomplete/exceeded"
			}
		}
	}
	compatPrototypes := map[string]bool{}
	for _, f := range fs {
		if f.Intent == "compat" {
			compatPrototypes[f.PrototypeID] = true
		}
	}
	if len(compatPrototypes) > 500 {
		return false, "compatibility prototype budget exceeded"
	}
	byID := map[string]Fixture{}
	for _, f := range fs {
		byID[f.ID] = f
	}
	splitCounts := map[string]map[string]int{"train": {}, "dev": {}, "test": {}, "regression": {}}
	categoryCounts := map[string]int{}
	space, returns := 0, 0
	for _, c := range a.Components {
		for _, intent := range []string{"return", "literal"} {
			id := c.PrimaryReturn
			if intent == "literal" {
				id = c.PrimaryLiteral
			}
			if id == nil {
				continue
			}
			splitCounts[c.Split][intent]++
			if c.Split == "test" {
				f := byID[*id]
				categoryCounts[intent+":"+f.Category]++
				if intent == "return" {
					returns++
					hasSpace := false
					for _, s := range f.Gold[0].Segments {
						if s.Kind == "chinese" && f.ExpectedRaw[s.End-1] == ' ' {
							hasSpace = true
						}
					}
					if hasSpace {
						space++
					}
				}
			}
		}
	}
	for _, split := range []string{"train", "dev"} {
		for _, intent := range []string{"return", "literal"} {
			if splitCounts[split][intent] < 100 {
				return false, "train/dev independent primary components insufficient"
			}
		}
	}
	for _, category := range returnCategories {
		if categoryCounts["return:"+category] < 20 {
			return false, "return category test component quota insufficient"
		}
	}
	for _, category := range literalCategories {
		if categoryCounts["literal:"+category] < 20 {
			return false, "literal category test component quota insufficient"
		}
	}
	if returns == 0 || float64(space)/float64(returns) < .2 {
		return false, "first-tone-space test primary share insufficient"
	}
	return true, "fixed acquisition and independent sample qualification verified"
}
