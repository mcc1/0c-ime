//go:build shiftreturneval

// 0c-shift-eval is an opt-in offline tool; it is excluded from product builds.
package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"

	sr "github.com/0c-biggo/0c-ime/internal/shiftreturn"
)

func main() { os.Exit(run(os.Args[1:], os.Stderr)) }
func run(args []string, stderr io.Writer) int {
	fail := func(e error) int { fmt.Fprintln(stderr, e); return 2 }
	if len(args) == 0 {
		return fail(fmt.Errorf("expected validate/assign/verify/enumerate/replay/summarize"))
	}
	command := args[0]
	flags := flag.NewFlagSet(command, flag.ContinueOnError)
	flags.SetOutput(stderr)
	fixtures := flags.String("fixtures", "", "fixture JSONL")
	sources := flags.String("sources", "", "source manifest")
	bank := flags.String("bank", "", "frozen task bank")
	assignment := flags.String("assignment", "", "assignment")
	lock := flags.String("lock", "", "run lock")
	out := flags.String("out", "", "new output directory")
	data := flags.String("data", "assets/data", "dictionary directory")
	for _, k := range []string{"model", "flow", "split", "rows", "events", "candidates", "generators"} {
		flags.String(k, "", k)
	}
	if e := flags.Parse(args[1:]); e != nil {
		return 2
	}
	if flags.NArg() != 0 {
		return fail(fmt.Errorf("unexpected positional arguments"))
	}
	_, _, _ = bank, lock, data
	switch command {
	case "assign", "validate", "verify", "enumerate", "replay", "summarize":
	default:
		return fail(fmt.Errorf("unknown command %q", command))
	}
	if *fixtures == "" || *sources == "" || *out == "" {
		return fail(fmt.Errorf("--fixtures --sources --out are required"))
	}
	fs, hash, e := sr.ReadFixtures(*fixtures)
	if e != nil {
		return fail(e)
	}
	var src sr.Sources
	if e = sr.ReadRecord(*sources, &src); e != nil {
		return fail(e)
	}
	if command == "assign" {
		a := sr.Assign(fs, src)
		a.FixturesSHA256 = hash
		if e = sr.NewOutput(*out); e != nil {
			return fail(e)
		}
		if e = sr.WriteRecord(filepath.Join(*out, "assignment.json"), a); e != nil {
			return fail(e)
		}
		return 0
	}
	if *assignment == "" {
		return fail(fmt.Errorf("--assignment is required"))
	}
	var a sr.Assignment
	if e = sr.ReadRecord(*assignment, &a); e != nil {
		return fail(e)
	}
	if e = sr.ValidateAssignment(fs, src, a, hash); e != nil {
		return fail(e)
	}
	// P0 provides the exchange/assignment commands. Later stages must supply all
	// source qualification and locked execution before these paths can succeed.
	if command != "validate" {
		return fail(fmt.Errorf("%s execution is not implemented at P0", command))
	}
	if e = sr.NewOutput(*out); e != nil {
		return fail(e)
	}
	if e = sr.WriteRecord(filepath.Join(*out, "validation.json"), map[string]any{"version": sr.Version, "fixtures": len(fs), "assignment_valid": true, "qualification": "not_run", "reason": "P0 exchange validation; bank/dictionary qualification not executed"}); e != nil {
		return fail(e)
	}
	return 0
}
