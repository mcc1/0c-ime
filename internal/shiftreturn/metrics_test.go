//go:build shiftreturneval

package shiftreturn

import (
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestMetricsWilsonIndependentComponents(t *testing.T) {
	m := metric(0, 400)
	if math.Abs(m.Wilson95[1]-.009512294334296508) > 1e-12 {
		t.Fatal(m.Wilson95)
	}
	if metric(0, 0).Value != nil || Wilson(0, 0) != nil {
		t.Fatal("empty denominator fabricated")
	}
}
func TestBootstrapSparseAndEmpty(t *testing.T) {
	b := map[string]bool{"a": true, "b": false, "c": false, "d": false}
	n := map[string]bool{"a": false, "b": false, "c": false, "d": false}
	p, e := Paired("base", "new", "sparse", b, n, []string{"d", "c", "b", "a"})
	if e != nil {
		t.Fatal(e)
	}
	if p.ValidBootstrap != 10000 || p.AbsoluteErrorReductionCi95[0] != 0 || p.AbsoluteErrorReductionCi95[1] != .75 {
		t.Fatalf("sparse bootstrap mismatch: %+v", p)
	}
	rng := newPythonRandom(271828)
	zero := 0
	for rep := 0; rep < 10000; rep++ {
		errors := 0
		for i := 0; i < 4; i++ {
			if rng.randrange(4) == 0 {
				errors++
			}
		}
		if errors == 0 {
			zero++
		}
	}
	if zero != 3165 {
		t.Fatalf("CPython RNG mismatch: baseline-zero replicates%d", zero)
	}
	p, e = Paired("base", "new", "empty", map[string]bool{}, map[string]bool{}, []string{"a"})
	if e != nil || p.ValidBootstrap != 0 || p.RelativeErrorReduction != nil || p.AbsoluteErrorReductionCi95 != nil {
		t.Fatal("empty subgroup fabricated")
	}
}
func TestLimitsEvidenceAndNoOverwrite(t *testing.T) {
	b := EvidenceBudget{}
	if e := b.Add(MaxArtifactBytes); e != nil {
		t.Fatal(e)
	}
	if e := b.Add(1); e == nil || b.Bytes != MaxArtifactBytes {
		t.Fatal("artifact cap bypass")
	}
	if e := b.Add(-1); e == nil {
		t.Fatal("negative artifact accounting")
	}
	dir := filepath.Join(t.TempDir(), "new")
	if e := NewOutput(dir); e != nil {
		t.Fatal(e)
	}
	if e := NewOutput(dir); e == nil {
		t.Fatal("output overwrite")
	}
}
func TestWeightsNumericalPython(t *testing.T) {
	cmd := exec.Command("python3", "fit.py", "verify-numerical", "../../design/shift-return-v2/numerical.json")
	if out, e := cmd.CombinedOutput(); e != nil {
		t.Fatalf("numerical: %s %v", out, e)
	}
	_ = os.Getenv("GOCACHE")
}
func TestLimitsOldEvaluatorSourceUnchanged(t *testing.T) {
	command := exec.Command("git", "diff", "--exit-code", "c512578157607615ed114feba20c8a443f260a19", "--", "core/mixedparse", "internal/mixedeval", "cmd/0c-mixed-eval")
	command.Dir = "../.."
	if out, e := command.CombinedOutput(); e != nil {
		t.Fatalf("old evaluator source changed: %s %v", out, e)
	}
}
func TestLimitsGlobalRetainedEvidenceAndPhysicalCap(t *testing.T) {
	root := t.TempDir()
	if e := os.Mkdir(filepath.Join(root, "phase"), 0755); e != nil {
		t.Fatal(e)
	}
	if e := os.WriteFile(filepath.Join(root, "phase", "rows.jsonl"), []byte("abc"), 0644); e != nil {
		t.Fatal(e)
	}
	if e := os.WriteFile(filepath.Join(root, "events.jsonl.gz"), []byte("12345"), 0644); e != nil {
		t.Fatal(e)
	}
	if e := os.WriteFile(filepath.Join(root, "report.json"), []byte("metadata"), 0644); e != nil {
		t.Fatal(e)
	}
	n, e := ExistingEvidenceBytes(root)
	if e != nil || n != 8 {
		t.Fatal("retained phase evidence not aggregated", n, e)
	}
	path := filepath.Join(root, "limited.jsonl.gz")
	budget := &EvidenceBudget{Bytes: MaxArtifactBytes - 64}
	writer, e := newArtifact(path, budget, true)
	if e != nil {
		t.Fatal(e)
	}
	payload := make([]byte, 10000)
	for i := range payload {
		payload[i] = byte(32 + i%95)
	}
	writeErr := writer.Write(map[string]any{"payload": string(payload)})
	closeErr := writer.Close()
	if writeErr == nil && closeErr == nil {
		t.Fatal("compressed evidence cap bypassed")
	}
	info, e := os.Stat(path)
	if e != nil {
		t.Fatal(e)
	}
	if info.Size() > 64 || budget.Bytes > MaxArtifactBytes {
		t.Fatal("bytes written beyond aggregate cap")
	}
}
