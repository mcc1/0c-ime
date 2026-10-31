package mixedeval

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/0c-biggo/0c-ime/core/layout"
	"github.com/0c-biggo/0c-ime/core/mixedparse"
	"github.com/0c-biggo/0c-ime/core/zhuyin"
)

type Fixture struct {
	Version      string                                    `json:"version"`
	ID           string                                    `json:"id"`
	Intent       string                                    `json:"intent"`
	Source       struct{ Kind, Reference, License string } `json:"source"`
	Families     []string                                  `json:"families"`
	Prelude      []mixedparse.Event                        `json:"prelude"`
	CustomTokens []string                                  `json:"custom_tokens"`
	Events       []mixedparse.Event                        `json:"events"`
	Assertions   []Assertion                               `json:"assertions"`
}
type Assertion struct {
	At            int                    `json:"at"`
	Window        int                    `json:"window"`
	Policy        string                 `json:"policy"`
	Stage         string                 `json:"stage"`
	Accepted      [][]mixedparse.Segment `json:"accepted"`
	Absent        [][]mixedparse.Segment `json:"absent"`
	AcceptedText  []string               `json:"accepted_text"`
	State         string                 `json:"state"`
	ClosedReason  string                 `json:"closed_reason"`
	Commit        *string                `json:"commit"`
	DelegateCount *int                   `json:"delegate_count"`
}
type Assignment struct {
	Version, Seed  string
	FixturesSHA256 string `json:"fixtures_sha256"`
	AliasesSHA256  string `json:"aliases_sha256"`
	Rows           map[string]struct {
		GroupID string `json:"group_id"`
		Split   string `json:"split"`
	} `json:"rows"`
}
type Adequacy struct {
	Counts          map[string]map[string]struct{ Rows, Groups int } `json:"counts"`
	Components      int                                              `json:"components"`
	PilotAdequate   bool                                             `json:"pilot_adequate"`
	ReleaseAdequate bool                                             `json:"release_adequate"`
	Primary         map[string]string                                `json:"primary"`
}

func Structure(phase, fixtures, aliases, assignment string) (Adequacy, error) {
	_, file, _, _ := runtime.Caller(0)
	root := filepath.Join(filepath.Dir(file), "../..")
	cmd := exec.Command("python3", filepath.Join(filepath.Dir(file), "fixturecheck.py"), phase, fixtures, aliases, filepath.Join(root, "design/mixed-input-experiment/fixture.schema.json"), assignment)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return Adequacy{}, fmt.Errorf("strict structure/split validation: %w: %s", err, out)
	}
	var a Adequacy
	err = json.Unmarshal(out, &a)
	return a, err
}
func ReadFixtures(path string) ([]Fixture, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	scan := bufio.NewScanner(f)
	scan.Buffer(make([]byte, 65536), 4<<20)
	out := []Fixture{}
	for scan.Scan() {
		var r Fixture
		dec := json.NewDecoder(strings.NewReader(scan.Text()))
		dec.DisallowUnknownFields()
		if err := dec.Decode(&r); err != nil {
			return nil, err
		}
		if err := ValidateSemantic(r); err != nil {
			return nil, fmt.Errorf("%s: %w", r.ID, err)
		}
		out = append(out, r)
	}
	return out, scan.Err()
}
func ValidateSemantic(r Fixture) error {
	for _, e := range r.Prelude {
		if e.Type == "select" || e.Type == "environment" {
			return fmt.Errorf("unsupported prelude event")
		}
	}
	windows := map[[2]int]string{}
	current := map[int]string{}
	raw := ""
	window := 0
	tones := 0
	closed := false
	finalized := false
	resetNext := false
	for at, e := range r.Events {
		if resetNext {
			current = map[int]string{}
			raw = ""
			window = 0
			tones = 0
			closed = false
			resetNext = false
		}
		switch e.Type {
		case "key":
			finalized = false
			if closed || len(raw) == 64 {
				window++
				raw = ""
				tones = 0
				closed = false
			}
			raw += e.Key
			if mixedparse.Tone(e.Key[0]) {
				tones++
				closed = tones == 2
			}
			current[window] = raw
		case "finalize":
			if e.Reason != "punctuation" && e.Text != "" {
				return fmt.Errorf("text on non-punctuation")
			}
			if e.Reason == "punctuation" && (len(e.Text) != 1 || e.Text[0] < 33 || e.Text[0] > 126) {
				return fmt.Errorf("punctuation must be single printable ASCII")
			}
			if e.Reason != "cursor" && e.Target != nil {
				return fmt.Errorf("target on non-cursor")
			}
			if e.Reason != "focus_end" && e.CommitVisible != nil {
				return fmt.Errorf("focus policy on other finalizer")
			}
			if raw != "" {
				current[window] = raw
			}
			if e.Reason == "enter" || e.Reason == "commit" || e.Reason == "cancel" || e.Reason == "reset" || e.Reason == "focus_end" {
				finalized = true
				resetNext = true
			} else {
				closed = true
			}
		case "select":
			if e.Ref == nil || e.Ref.MenuEvent >= at || e.Ref.MenuEvent < 0 || r.Events[e.Ref.MenuEvent].Reason != "open_candidates" {
				return fmt.Errorf("invalid selection menu ref")
			}
			for j := e.Ref.MenuEvent + 1; j < at; j++ {
				if r.Events[j].Type == "key" || r.Events[j].Type == "finalize" || r.Events[j].Type == "environment" {
					return fmt.Errorf("stale menu reference")
				}
			}
			current[window] = raw
			closed = true
		case "environment":
			if e.Layout != "standard" && e.Layout != "eten" && e.Layout != "hsu" {
				return fmt.Errorf("invalid layout")
			}
			if raw != "" {
				current[window] = raw
			}
			closed = true
		default:
			return fmt.Errorf("unknown event")
		}
		for wi, v := range current {
			windows[[2]int{at, wi}] = v
		}
	}

	finals := 0
	for _, a := range r.Assertions {
		if a.At < 0 || a.At >= len(r.Events) {
			return fmt.Errorf("assert event index")
		}
		v, ok := windows[[2]int{a.At, a.Window}]
		if !ok || v == "" {
			return fmt.Errorf("assert window missing")
		}
		for _, parts := range append(append([][]mixedparse.Segment{}, a.Accepted...), a.Absent...) {
			if err := validatePartition(v, parts); err != nil {
				return err
			}
		}
		if a.Stage == "final" && a.Policy == "generator" {
			finals++
			if r.Intent == "ambiguous" {
				lit, chi := false, false
				for _, p := range a.Accepted {
					if mixedparse.FirstChinese(p) == int(^uint(0)>>1) {
						lit = true
					} else {
						chi = true
					}
				}
				if !lit || !chi {
					return fmt.Errorf("ambiguous missing alternatives")
				}
			}
		}
	}
	if !finalized {
		return fmt.Errorf("missing explicit terminal finalizer")
	}
	if finals == 0 {
		script := false
		for _, a := range r.Assertions {
			script = script || a.Policy == "scripted"
		}
		if !script {
			return fmt.Errorf("missing final generator gold")
		}
	}
	return nil
}
func validatePartition(raw string, p []mixedparse.Segment) error {
	at := 0
	for _, s := range p {
		if s.Begin != at || s.End <= s.Begin || s.End > len(raw) {
			return fmt.Errorf("invalid contiguous partition span")
		}
		if s.Kind == "literal" {
			if s.Text != raw[s.Begin:s.End] || s.Reading != "" {
				return fmt.Errorf("literal does not preserve keys")
			}
		} else if s.Kind == "chinese" {
			syl := zhuyin.Syllable(s.Reading)
			if !syl.Valid() || syl.Untoned() || string(syl.WithTone(syl.ToneOf())) != s.Reading {
				return fmt.Errorf("noncanonical reading")
			}
			key := raw[s.End-1]
			if !mixedparse.Tone(key) {
				return fmt.Errorf("missing strict tone byte")
			}
			tone, _ := layout.Standard.Symbol(rune(key))
			got, ok := mixedparse.SpellExact(raw[s.Begin:s.End-1], tone)
			if !ok || got != syl {
				return fmt.Errorf("strict spelling/reading mismatch")
			}
		} else {
			return fmt.Errorf("segment kind")
		}
		at = s.End
	}
	if at != len(raw) {
		return fmt.Errorf("partition incomplete")
	}
	return nil
}
