//go:build shiftreturneval

package shiftreturn

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
)

const MaxArtifactBytes int64 = 8589934592
const MinFreeBytes uint64 = 10737418240

func VerifyLock(path string) error {
	var lock Lock
	if e := ReadRecord(path, &lock); e != nil {
		return e
	}
	if lock.BaselineCommit != "e754019571573051d8c29e06526802c31d5172e2" {
		return fmt.Errorf("baseline commit differs")
	}
	for path, want := range lock.Files {
		b, e := os.ReadFile(path)
		if e != nil {
			return e
		}
		if HashBytes(b) != want {
			return fmt.Errorf("locked bytes differ: %s", path)
		}
	}
	goVersion, e := exec.Command("go", "version").Output()
	if e != nil {
		return e
	}
	python, e := exec.Command("python3", "-c", "import json,platform,numpy;print(json.dumps([platform.python_version(),numpy.__version__]))").Output()
	if e != nil {
		return e
	}
	var versions []string
	if e = json.Unmarshal(python, &versions); e != nil {
		return e
	}
	if len(versions) != 2 || lock.Versions.Go != strings.TrimSpace(string(goVersion)) || lock.Versions.Python != versions[0] || lock.Versions.Numpy != versions[1] || lock.Versions.Threads != 1 {
		return fmt.Errorf("tool versions differ from lock")
	}
	head, e := exec.Command("git", "rev-parse", "HEAD").Output()
	if e != nil {
		return e
	}
	diff, e := exec.Command("git", "diff", "--binary", "HEAD").Output()
	if e != nil {
		return e
	}
	if err := exec.Command("git", "merge-base", "--is-ancestor", lock.SourceCommit, strings.TrimSpace(string(head))).Run(); err != nil {
		return fmt.Errorf("locked source commit is not an ancestor of checkout: %w", err)
	}
	if HashBytes(diff) != lock.DirtyDiffSHA256 {
		return fmt.Errorf("source checkout differs from lock")
	}
	// A lock must cover every tracked implementation/protocol file, not only the
	// convenient inputs a caller happened to list.
	untracked, e := exec.Command("git", "ls-files", "--others", "--exclude-standard", "-z", "core", "internal/shiftreturn", "cmd/0c-shift-eval", "internal/datadir", "internal/englishpack", "design/shift-return-protocol-v2.md", "design/shift-return-source-map-v2.md", "design/shift-return-v2/schema.json").Output()
	if e != nil {
		return e
	}
	if len(untracked) > 0 {
		return fmt.Errorf("implementation source must be committed before a formal lock")
	}
	files, e := exec.Command("git", "ls-files", "-z", "core", "internal/shiftreturn", "cmd/0c-shift-eval", "internal/datadir", "internal/englishpack", "design/shift-return-protocol-v2.md", "design/shift-return-source-map-v2.md", "design/shift-return-v2/schema.json", "go.mod", "go.sum").Output()
	if e != nil {
		return e
	}
	absolute := map[string]bool{}
	for path := range lock.Files {
		p, e := filepath.Abs(path)
		if e != nil {
			return e
		}
		absolute[p] = true
	}
	for _, file := range bytes.Split(files, []byte{0}) {
		if len(file) == 0 {
			continue
		}
		p, e := filepath.Abs(string(file))
		if e != nil {
			return e
		}
		if !absolute[p] {
			return fmt.Errorf("source missing from lock: %s", file)
		}
	}
	return nil
}
func RequireLocked(path string, inputs ...string) error {
	var lock Lock
	if e := ReadRecord(path, &lock); e != nil {
		return e
	}
	covered := map[string]bool{}
	for p := range lock.Files {
		abs, e := filepath.Abs(p)
		if e != nil {
			return e
		}
		covered[abs] = true
	}
	for _, p := range inputs {
		abs, e := filepath.Abs(p)
		if e != nil {
			return e
		}
		if !covered[abs] {
			return fmt.Errorf("required input missing from lock: %s", p)
		}
	}
	return nil
}
func PreflightSpace(path string) error {
	var stat syscall.Statfs_t
	if e := syscall.Statfs(path, &stat); e != nil {
		return e
	}
	available := stat.Bavail * uint64(stat.Bsize)
	if available < MinFreeBytes {
		return fmt.Errorf("insufficient disk: %d free bytes; fixed minimum %d", available, MinFreeBytes)
	}
	return nil
}

// EvidenceBudget checks bytes actually written, never removes older evidence.
type EvidenceBudget struct{ Bytes int64 }

func (b *EvidenceBudget) Add(n int64) error {
	if n < 0 || n > MaxArtifactBytes-b.Bytes {
		return fmt.Errorf("raw evidence exceeds fixed 8GiB cap")
	}
	b.Bytes += n
	return nil
}

const ExperimentRoot = "/workspace/scratch/shift-return-v2"

func ExistingEvidenceBytes(root string) (int64, error) {
	var total int64
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if os.IsNotExist(err) {
			return nil
		}
		if err != nil {
			return err
		}
		if entry.Type()&os.ModeSymlink != 0 {
			return fmt.Errorf("evidence root contains symlink: %s", path)
		}
		if entry.IsDir() {
			return nil
		}
		if strings.HasSuffix(path, ".jsonl") || strings.HasSuffix(path, ".jsonl.gz") {
			info, err := entry.Info()
			if err != nil {
				return err
			}
			total += info.Size()
		}
		return nil
	})
	return total, err
}
func ValidateOutputRoot(path string) error {
	absolute, e := filepath.Abs(path)
	if e != nil {
		return e
	}
	relative, e := filepath.Rel(ExperimentRoot, absolute)
	if e != nil || relative == "." || relative == ".." || strings.HasPrefix(relative, ".."+string(os.PathSeparator)) {
		return fmt.Errorf("formal output must be a new phase below %s", ExperimentRoot)
	}
	return nil
}
