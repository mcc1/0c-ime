package mixedeval

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
)

func HashBytes(b []byte) string            { h := sha256.Sum256(b); return hex.EncodeToString(h[:]) }
func HashFile(path string) (string, error) { b, e := os.ReadFile(path); return HashBytes(b), e }

type RunLock struct {
	Version              string            `json:"version"`
	Baseline             string            `json:"baseline"`
	SourceCommit         string            `json:"source_commit"`
	DirtyDiffSHA256      string            `json:"dirty_diff_sha256"`
	Files                map[string]string `json:"files"`
	Toolchain            map[string]string `json:"toolchain"`
	Command              string            `json:"command"`
	BaselineRuntimeTrees map[string]string `json:"baseline_runtime_trees"`
}

func FreezeLock(data, fixtures, aliases, assignment, lock string) error {
	l, err := ComputeLock(data, fixtures, aliases, assignment)
	if err != nil {
		return err
	}
	b, _ := json.MarshalIndent(l, "", "  ")
	f, err := os.OpenFile(lock, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0644)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.Write(append(b, '\n'))
	return err
}
func ComputeLock(data, fixtures, aliases, assignment string) (RunLock, error) {
	l := RunLock{Version: "mixed-run-lock-v1", Baseline: "e754019571573051d8c29e06526802c31d5172e2", Files: map[string]string{}, Toolchain: map[string]string{"go": runtime.Version(), "os": runtime.GOOS, "arch": runtime.GOARCH}, BaselineRuntimeTrees: map[string]string{}, Command: "go run -tags=mixedeval ./cmd/0c-mixed-eval -data assets/data -fixtures testdata/mixed-boundary/fixtures.jsonl -aliases testdata/mixed-boundary/aliases.json -assignment testdata/mixed-boundary/assignment.json -phase enumerate -split all -out <isolated-directory>"}
	run := func(args ...string) (string, error) {
		b, e := exec.Command("git", args...).Output()
		return strings.TrimSpace(string(b)), e
	}
	var err error
	l.SourceCommit, err = run("rev-parse", "HEAD")
	if err != nil {
		return l, err
	}
	diff, err := exec.Command("git", "diff", "--binary", "HEAD").Output()
	if err != nil {
		return l, err
	}
	l.DirtyDiffSHA256 = HashBytes(diff)
	for _, tree := range []string{"core", "internal", "cmd"} {
		oid, e := run("rev-parse", l.Baseline+":"+tree)
		if e != nil {
			return l, e
		}
		l.BaselineRuntimeTrees[tree] = oid
	}
	paths := []string{fixtures, aliases, assignment, "testdata/mixed-boundary/protocol.json", "testdata/mixed-boundary/metadata.json", "design/mixed-input-experiment-acceptance.md", "design/mixed-input-experiment-plan.md", "design/mixed-input-evaluator-contract.md", "design/mixed-input-experiment/fixture.schema.json"}
	for _, root := range []string{"core/mixedparse", "internal/mixedeval", "cmd/0c-mixed-eval", "testdata/mixed-boundary"} {
		err = filepath.WalkDir(root, func(p string, d os.DirEntry, e error) error {
			if e != nil {
				return e
			}
			if d.IsDir() {
				if d.Name() == "__pycache__" {
					return filepath.SkipDir
				}
				return nil
			}
			if filepath.Base(p) == "run-lock.json" {
				return nil
			}
			paths = append(paths, p)
			return nil
		})
		if err != nil {
			return l, err
		}
	}
	paths = append(paths, "core/session/mixed.go", "core/session/mixed_eval_observer.go", "core/session/mixed_eval_adapter.go", "core/session/mixed_eval_adapter_test.go")
	for _, root := range []string{data} {
		err = filepath.WalkDir(root, func(p string, d os.DirEntry, e error) error {
			if e != nil {
				return e
			}
			if !d.IsDir() {
				paths = append(paths, p)
			}
			return nil
		})
		if err != nil {
			return l, err
		}
	}
	sort.Strings(paths)
	for _, p := range paths {
		h, e := HashFile(p)
		if e != nil {
			return l, e
		}
		l.Files[filepath.ToSlash(p)] = h
	}
	for _, m := range []string{filepath.Join(data, "manifest.json"), filepath.Join(data, "english/manifest.json")} {
		b, e := os.ReadFile(m)
		if e != nil {
			return l, e
		}
		var manifest any
		if json.Unmarshal(b, &manifest) != nil {
			return l, fmt.Errorf("manifest invalid")
		}
		l.Toolchain[filepath.ToSlash(m)+"_payload_manifest"] = string(b)
	}
	return l, nil
}
func VerifyLock(path string) (RunLock, error) {
	var l RunLock
	b, e := os.ReadFile(path)
	if e != nil {
		return l, e
	}
	if e = json.Unmarshal(b, &l); e != nil {
		return l, e
	}
	for p, want := range l.Files {
		got, e := HashFile(p)
		if e != nil || got != want {
			return l, fmt.Errorf("frozen file changed: %s", p)
		}
	}
	if runtime.Version() != l.Toolchain["go"] || runtime.GOOS != l.Toolchain["os"] || runtime.GOARCH != l.Toolchain["arch"] {
		return l, fmt.Errorf("toolchain changed")
	}
	return l, nil
}
