//go:build shiftreturneval

package shiftreturn

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
)

func ReadRecord(path string, out any) error {
	b, e := os.ReadFile(path)
	if e != nil {
		return e
	}
	return Decode(b, out)
}
func ReadFixtures(path string) ([]Fixture, string, error) {
	b, e := os.ReadFile(path)
	if e != nil {
		return nil, "", e
	}
	fs := []Fixture{}
	s := bufio.NewScanner(bytes.NewReader(b))
	s.Buffer(make([]byte, 4096), 16<<20)
	line := 0
	for s.Scan() {
		line++
		if len(s.Bytes()) == 0 {
			return nil, "", fmt.Errorf("blank JSONL line %d", line)
		}
		var f Fixture
		if e := Decode(s.Bytes(), &f); e != nil {
			return nil, "", fmt.Errorf("line %d: %w", line, e)
		}
		fs = append(fs, f)
	}
	if e := s.Err(); e != nil {
		return nil, "", e
	}
	if e := ValidateFixtures(fs); e != nil {
		return nil, "", e
	}
	return fs, HashBytes(b), nil
}

// NewOutput never overwrites an earlier run, including an existing empty directory.
func NewOutput(path string) error {
	if path == "" {
		return fmt.Errorf("--out is required")
	}
	return os.Mkdir(path, 0755)
}
func WriteRecord(path string, v any) error {
	b := append(Canonical(v), '\n')
	f, e := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0644)
	if e != nil {
		return e
	}
	_, e = f.Write(b)
	closeErr := f.Close()
	if e != nil {
		return e
	}
	return closeErr
}
