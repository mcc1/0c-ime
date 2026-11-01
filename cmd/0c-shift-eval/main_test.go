//go:build shiftreturneval

package main

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	sr "github.com/0c-biggo/0c-ime/internal/shiftreturn"
)

func TestAssignmentCLI(t *testing.T) {
	root := t.TempDir()
	src := filepath.Join(root, "sources.json")
	if e := sr.WriteRecord(src, sr.Sources{Version: sr.Version, Kind: "sources", Demo: true, Entries: []struct {
		ID               string   `json:"id"`
		Kind             string   `json:"kind"`
		Reference        string   `json:"reference"`
		Version          string   `json:"version"`
		SHA256           string   `json:"sha256"`
		License          string   `json:"license"`
		AnnotationPasses float64  `json:"annotation_passes"`
		AdjudicationLog  string   `json:"adjudication_log"`
		AnnotatorIDs     []string `json:"annotator_ids"`
		CachedPath       string   `json:"cached_path"`
	}{}, SeenPrefixes: []string{}, SeenContents: []string{}, SeenRaw: []string{}}); e != nil {
		t.Fatal(e)
	}
	fixtures := "../../design/shift-return-v2/examples.jsonl"
	out := filepath.Join(root, "assign")
	args := []string{"assign", "--fixtures", fixtures, "--sources", src, "--out", out}
	var stderr bytes.Buffer
	if code := run(args, &stderr); code != 0 {
		t.Fatalf("exit %d: %s", code, stderr.String())
	}
	original, e := os.ReadFile(filepath.Join(out, "assignment.json"))
	if e != nil {
		t.Fatal(e)
	}
	if code := run(args, &stderr); code != 2 {
		t.Fatalf("existing out exit %d", code)
	}
	after, e := os.ReadFile(filepath.Join(out, "assignment.json"))
	if e != nil || !bytes.Equal(original, after) {
		t.Fatal("overwrote earlier output")
	}
	args = []string{"validate", "--fixtures", fixtures, "--sources", src, "--assignment", filepath.Join(out, "assignment.json"), "--out", filepath.Join(root, "validate")}
	if code := run(args, &stderr); code != 0 {
		t.Fatalf("validate exit %d: %s", code, stderr.String())
	}
	if code := run([]string{"enumerate", "--fixtures", fixtures, "--sources", src, "--assignment", filepath.Join(out, "assignment.json"), "--out", filepath.Join(root, "quality")}, &stderr); code != 2 {
		t.Fatalf("unimplemented quality returned %d", code)
	}
	if _, e = os.Stat(filepath.Join(root, "quality")); !os.IsNotExist(e) {
		t.Fatal("quality skeleton created evidence")
	}
}
