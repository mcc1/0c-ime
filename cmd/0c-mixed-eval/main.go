//go:build mixedeval

// 0c-mixed-eval is an offline experiment. It never installs an input method.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/0c-biggo/0c-ime/core/lexicon"
	"github.com/0c-biggo/0c-ime/core/lm"
	"github.com/0c-biggo/0c-ime/core/mixedparse"
	"github.com/0c-biggo/0c-ime/core/personal"
	"github.com/0c-biggo/0c-ime/core/zhuyin"
	"github.com/0c-biggo/0c-ime/internal/datadir"
	"github.com/0c-biggo/0c-ime/internal/mixedeval"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
func run() error {
	data := flag.String("data", "assets/data", "verified sealed public package")
	fixtures := flag.String("fixtures", "testdata/mixed-boundary/fixtures.jsonl", "frozen fixtures")
	aliases := flag.String("aliases", "testdata/mixed-boundary/aliases.json", "exact family aliases")
	assignment := flag.String("assignment", "testdata/mixed-boundary/assignment.json", "frozen family assignment")
	lock := flag.String("lock", "testdata/mixed-boundary/run-lock.json", "frozen run lock")
	phase := flag.String("phase", "validate", "validate|assign|lock|correctness|enumerate")
	split := flag.String("split", "all", "all|train|dev|test")
	out := flag.String("out", "", "new isolated output directory")
	flag.Parse()
	if *split != "all" && *split != "train" && *split != "dev" && *split != "test" {
		return fmt.Errorf("invalid split")
	}
	if *phase == "assign" {
		a, e := mixedeval.Structure("assign", *fixtures, *aliases, *assignment)
		if e != nil {
			return e
		}
		return json.NewEncoder(os.Stdout).Encode(a)
	}
	if *phase == "validate" {
		a, e := mixedeval.Structure("validate", *fixtures, *aliases, *assignment)
		if e != nil {
			return e
		}
		rows, e := mixedeval.ReadFixtures(*fixtures)
		if e != nil {
			return e
		}
		return json.NewEncoder(os.Stdout).Encode(map[string]any{"schema_semantic_split_valid": true, "rows": len(rows), "adequacy": a})
	}
	if *phase == "lock" {
		return mixedeval.FreezeLock(*data, *fixtures, *aliases, *assignment, *lock)
	}
	if *phase != "correctness" && *phase != "enumerate" {
		return fmt.Errorf("unfrozen ranking is forbidden; invalid phase %s", *phase)
	}
	for _, p := range []string{filepath.Join(*data, "manifest.json"), filepath.Join(*data, "english/manifest.json")} {
		b, e := os.ReadFile(p)
		if e != nil || len(b) == 0 {
			return fmt.Errorf("nonempty public manifests required")
		}
	}
	engine, e := datadir.Open(*data)
	if e != nil {
		return e
	}
	if engine.EnglishPack == "" {
		return fmt.Errorf("English prototype fallback forbidden")
	}
	if *out == "" {
		return fmt.Errorf("-out required")
	}
	if e := os.Mkdir(*out, 0755); e != nil {
		return e
	}
	if *phase == "correctness" {
		model := personal.NewModel(engine.Model.Lex, engine.Model.LM)
		tiny := lexicon.NewMem()
		sys := lm.NewMem()
		for _, entry := range []struct{ w, r string }{{"賺", "ㄓㄨㄢˋ"}, {"錢", "ㄑㄧㄢˊ"}, {"賺錢", "ㄓㄨㄢˋ ㄑㄧㄢˊ"}, {"你", "ㄋㄧˇ"}, {"你好", "ㄋㄧˇ ㄏㄠˇ"}} {
			tiny.Add(lexicon.Entry{Surface: entry.w, Reading: zhuyin.ParseReading(entry.r), PRW: 1})
			sys.SetUni(entry.w, .01)
		}
		tinyModel := personal.NewModel(tiny, sys)
		tinyResult, e := mixedeval.Exhaustive(mixedparse.Evaluator{Lex: tiny, Score: tinyModel, System: tinyModel, English: engine.English})
		if e != nil {
			return e
		}
		r, e := mixedeval.Exhaustive(mixedparse.Evaluator{Lex: model.Source(), Score: model, System: model, English: engine.English})
		if e != nil {
			return e
		}
		b, _ := json.MarshalIndent(map[string]any{"tiny": tinyResult, "public": r}, "", "  ")
		return os.WriteFile(filepath.Join(*out, "correctness.json"), append(b, '\n'), 0644)
	}
	_, e = mixedeval.VerifyLock(*lock)
	if e != nil {
		return e
	}
	return mixedeval.Enumerate(*data, *fixtures, *aliases, *assignment, *lock, *split, *out, engine)
}
