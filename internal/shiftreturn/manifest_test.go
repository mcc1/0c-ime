//go:build shiftreturneval

package shiftreturn

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestSchemaSelectedManifestExplicitTransport(t *testing.T) {
	dir := t.TempDir()
	sample, err := os.ReadFile("../../design/shift-return-v2/canonical-model-sample.json")
	if err != nil {
		t.Fatal(err)
	}
	manifest := SelectedModelManifest{Version: Version, Kind: "selected_model_manifest", Demo: true, SourceLockSHA256: strings.Repeat("0", 64), FitLockSHA256: strings.Repeat("0", 64), Models: map[string]SelectedModelEntry{}}
	lock := Lock{Demo: true, Files: map[string]string{}}
	masks := map[string][]int{"E0": {0, 1, 2, 3, 4, 5, 6}, "EL": {0, 1, 2, 3, 4, 5, 6, 7, 8}, "EW": {0, 1, 2, 3, 4, 5, 6, 9, 10, 11, 12}, "EC": {0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12}}
	for _, name := range []string{"E0", "EL", "EW", "EC"} {
		var model Model
		if err = json.Unmarshal(sample, &model); err != nil {
			t.Fatal(err)
		}
		model.Name = name
		model.Active = masks[name]
		path := filepath.Join(dir, name+"-chosen.json")
		bytes := append(Canonical(model), '\n')
		if err = os.WriteFile(path, bytes, 0644); err != nil {
			t.Fatal(err)
		}
		hash := HashBytes(bytes)
		manifest.Models[name] = SelectedModelEntry{Path: path, SHA256: hash}
		lock.Files[path] = hash
	}
	path := filepath.Join(dir, "selected-model-manifest.json")
	bytes := append(Canonical(manifest), '\n')
	if err = os.WriteFile(path, bytes, 0644); err != nil {
		t.Fatal(err)
	}
	lock.Files[path] = HashBytes(bytes)
	models, hashes, err := evidenceLockModels(lock)
	if err != nil || len(models) != 4 || len(hashes) != 4 {
		t.Fatal(models, err)
	}
	lockPath := filepath.Join(dir, "lock-input.json")
	if err = os.WriteFile(lockPath, Canonical(lock), 0644); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("python3", "-c", `import json,sys;from reduce import selected_models;m,h=selected_models(json.load(open(sys.argv[1])));assert len(m)==len(h)==4`, lockPath)
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("independent manifest transport %s %v", output, err)
	}
	delete(lock.Files, path)
	if _, _, err = evidenceLockModels(lock); err == nil {
		t.Fatal("arbitrary model files discovered without manifest")
	}
	lock.Files[path] = HashBytes(bytes)
	duplicate := filepath.Join(dir, "another", "selected-model-manifest.json")
	lock.Files[duplicate] = HashBytes(bytes)
	if _, _, err = evidenceLockModels(lock); err == nil {
		t.Fatal("multiple manifest accepted")
	}
	delete(lock.Files, duplicate)
	entry := manifest.Models["EC"]
	entry.SHA256 = strings.Repeat("f", 64)
	manifest.Models["EC"] = entry
	bytes = append(Canonical(manifest), '\n')
	if err = os.WriteFile(path, bytes, 0644); err != nil {
		t.Fatal(err)
	}
	lock.Files[path] = HashBytes(bytes)
	if _, _, err = evidenceLockModels(lock); err == nil {
		t.Fatal("model hash mismatch accepted")
	}
}
func TestMetricsExplicitUnavailableSemantics(t *testing.T) {
	m := metric(2, 5)
	m.ObservedErrors = 1
	m.UnavailableComponents = 1
	m.Status = "unavailable"
	m.Value = nil
	m.Wilson95 = nil
	if err := validateMetric(m); err != nil {
		t.Fatal(err)
	}
	m.Errors = 1
	if err := validateMetric(m); err == nil {
		t.Fatal("charged count inconsistent")
	}
	m.Errors = 2
	m.Value = ptr(.4)
	if err := validateMetric(m); err == nil {
		t.Fatal("charged count presented as observed rate")
	}
}
