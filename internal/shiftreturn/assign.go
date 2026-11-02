//go:build shiftreturneval

package shiftreturn

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math/big"
	"os/exec"
	"reflect"
	"sort"
	"strings"
)

func sortedUnique(a []string) []string {
	r := append([]string{}, a...)
	sort.Strings(r)
	out := []string{}
	for _, s := range r {
		if len(out) == 0 || out[len(out)-1] != s {
			out = append(out, s)
		}
	}
	return out
}
func SeenHash(s Sources) string {
	return HashBytes(Canonical(map[string]any{"seen_prefixes": sortedUnique(s.SeenPrefixes), "seen_contents": sortedUnique(s.SeenContents), "seen_raw": sortedUnique(s.SeenRaw)}))
}

type seenMatcher struct {
	prefix, content, reading, raw map[string]bool
	normalized                    map[string]string
}

func newSeenMatcher(fs []Fixture, s Sources) seenMatcher {
	values := append([]string{}, s.SeenContents...)
	for _, f := range fs {
		for _, gold := range f.Gold {
			values = append(values, gold.AcceptedSurfaces...)
		}
		for _, family := range f.Families {
			if strings.HasPrefix(family, "word:") {
				values = append(values, strings.TrimPrefix(family, "word:"))
			}
			if strings.HasPrefix(family, "content:") {
				values = append(values, strings.TrimPrefix(family, "content:"))
			}
		}
	}
	normalized := []string{}
	if len(values) > 0 {
		cmd := exec.Command("python3", "-c", `import sys,json,unicodedata;print(json.dumps([unicodedata.normalize("NFC",x) for x in json.load(sys.stdin)],ensure_ascii=False))`)
		cmd.Stdin = bytes.NewReader(Canonical(values))
		output, err := cmd.Output()
		if err != nil {
			panic(fmt.Errorf("pinned Python NFC normalization unavailable: %w", err))
		}
		if err = json.Unmarshal(output, &normalized); err != nil {
			panic(err)
		}
	}
	m := seenMatcher{map[string]bool{}, map[string]bool{}, map[string]bool{}, map[string]bool{}, map[string]string{}}
	for i, v := range values {
		m.normalized[v] = normalized[i]
	}
	for _, v := range s.SeenPrefixes {
		m.prefix[strings.ToLower(v)] = true
	}
	for _, v := range s.SeenContents {
		m.content[m.normalized[v]] = true
		m.reading[strings.Join(strings.Fields(v), " ")] = true
	}
	for _, v := range s.SeenRaw {
		m.raw[v] = true
	}
	return m
}
func (m seenMatcher) matches(f Fixture) bool {
	if m.raw[f.ExpectedRaw] {
		return true
	}
	for _, family := range f.Families {
		if strings.HasPrefix(family, "prefix:") && m.prefix[strings.ToLower(strings.TrimPrefix(family, "prefix:"))] {
			return true
		}
		for _, tag := range []string{"word:", "content:"} {
			if strings.HasPrefix(family, tag) && m.content[m.normalized[strings.TrimPrefix(family, tag)]] {
				return true
			}
		}
		if strings.HasPrefix(family, "reading:") && m.reading[strings.Join(strings.Fields(strings.TrimPrefix(family, "reading:")), " ")] {
			return true
		}
		for _, tag := range []string{"literal:", "raw:"} {
			if strings.HasPrefix(family, tag) && m.raw[strings.TrimPrefix(family, tag)] {
				return true
			}
		}
	}

	for _, gold := range f.Gold {
		prefix := ""
		readings := []string{}
		for _, seg := range gold.Segments {
			if seg.Kind == "literal" && len(readings) == 0 {
				prefix += f.ExpectedRaw[seg.Begin:seg.End]
			}
			if seg.Kind == "chinese" {
				readings = append(readings, seg.Reading)
			}
		}
		if m.prefix[strings.ToLower(prefix)] {
			return true
		}
		if len(readings) > 0 {
			if m.reading[strings.Join(strings.Fields(strings.Join(readings, " ")), " ")] {
				return true
			}
			for _, surface := range gold.AcceptedSurfaces {
				normalized := m.normalized[surface]
				if strings.HasPrefix(normalized, prefix) && m.content[strings.TrimPrefix(normalized, prefix)] {
					return true
				}
			}
		}
	}
	return false
}

// Assign computes the transitive family closure, including observable collisions.
// FixturesSHA256 is populated by the file reader from original JSONL bytes.
func Assign(fs []Fixture, s Sources) Assignment {
	matcher := newSeenMatcher(fs, s)
	a := Assignment{Version: Version, Kind: "assignment", Demo: s.Demo, Components: []Component{}, SeenSHA256: SeenHash(s)}
	parent := make([]int, len(fs))
	for i := range parent {
		parent[i] = i
	}
	var find func(int) int
	find = func(i int) int {
		if parent[i] != i {
			parent[i] = find(parent[i])
		}
		return parent[i]
	}
	owners := map[string]int{}
	for i, f := range fs {
		a.Demo = a.Demo || f.Demo
		keys := []string{}
		for _, family := range f.Families {
			key := family
			switch {
			case strings.HasPrefix(family, "prefix:"):
				key = "prefix:" + strings.ToLower(strings.TrimPrefix(family, "prefix:"))
			case strings.HasPrefix(family, "word:"):
				key = "word:" + matcher.normalized[strings.TrimPrefix(family, "word:")]
			case strings.HasPrefix(family, "content:"):
				key = "word:" + matcher.normalized[strings.TrimPrefix(family, "content:")]
			case strings.HasPrefix(family, "reading:"):
				key = "reading:" + strings.Join(strings.Fields(strings.TrimPrefix(family, "reading:")), " ")
			}
			keys = append(keys, key)
		}
		keys = append(keys, "raw:"+f.ExpectedRaw)
		if f.Intent == "literal" {
			keys = append(keys, "literal:"+f.ExpectedRaw)
		}
		for _, gold := range f.Gold {
			prefixEnd := -1
			readings := []string{}
			for _, seg := range gold.Segments {
				if seg.Kind == "chinese" {
					if prefixEnd < 0 {
						prefixEnd = seg.Begin
					}
					readings = append(readings, seg.Reading)
				}
			}
			if prefixEnd >= 0 {
				prefix := f.ExpectedRaw[:prefixEnd]
				if prefix != "" {
					keys = append(keys, "prefix:"+strings.ToLower(prefix))
				}
				keys = append(keys, "reading:"+strings.Join(readings, " "))
				for _, surface := range gold.AcceptedSurfaces {
					normalized := matcher.normalized[surface]
					if strings.HasPrefix(normalized, prefix) {
						keys = append(keys, "word:"+strings.TrimPrefix(normalized, prefix))
					}
				}
			}
		}

		// Same observable prelude/raw must never cross splits, even when labels differ.
		keys = append(keys, "observable:"+string(Canonical([]any{f.PreludeEvents, f.ExpectedPrelude, f.ExpectedRaw})))
		keys = append(keys, "prototype:"+f.PrototypeID)
		for _, k := range keys {
			if j, ok := owners[k]; ok {
				parent[find(i)] = find(j)
			} else {
				owners[k] = i
			}
		}
	}
	groups := map[int][]Fixture{}
	for i, f := range fs {
		groups[find(i)] = append(groups[find(i)], f)
	}
	for _, members := range groups {
		sort.Slice(members, func(i, j int) bool { return members[i].ID < members[j].ID })
		c := Component{RowIDs: []string{}, Categories: []string{}}
		regression := false
		for _, f := range members {
			c.RowIDs = append(c.RowIDs, f.ID)
			c.Categories = append(c.Categories, f.Category)
			regression = regression || matcher.matches(f)
		}
		c.Categories = sortedUnique(c.Categories)
		c.ID = HashBytes(Canonical(c.RowIDs))
		h := HashBytes([]byte("shift-return-v2:" + c.ID))
		z := new(big.Int)
		z.SetString(h, 16)
		bucket := new(big.Int).Mod(z, big.NewInt(10)).Int64()
		c.Split = "train"
		if bucket >= 6 {
			c.Split = "dev"
		}
		if bucket >= 8 {
			c.Split = "test"
		}
		if regression {
			c.Split = "regression"
		}
		sort.Slice(members, func(i, j int) bool {
			if members[i].SourcePriority != members[j].SourcePriority {
				return members[i].SourcePriority < members[j].SourcePriority
			}
			return members[i].ID < members[j].ID
		})
		for _, f := range members {
			f := f
			switch f.Intent {
			case "return":
				if c.PrimaryReturn == nil {
					c.PrimaryReturn = &f.ID
					c.PrimaryReturnMetadata = &struct {
						ID               string `json:"id"`
						Category         string `json:"category"`
						Stratum          string `json:"stratum"`
						ContextClass     string `json:"context_class"`
						ContrastEligible bool   `json:"contrast_eligible"`
					}{f.ID, f.Category, f.Stratum, f.ContextClass, f.ContextClass == "contrast"}
				}
			case "literal":
				if c.PrimaryLiteral == nil {
					c.PrimaryLiteral = &f.ID
					c.PrimaryLiteralMetadata = &struct {
						ID               string `json:"id"`
						Category         string `json:"category"`
						Stratum          string `json:"stratum"`
						ContextClass     string `json:"context_class"`
						ContrastEligible bool   `json:"contrast_eligible"`
					}{f.ID, f.Category, f.Stratum, f.ContextClass, false}
				}
			}
		}
		a.Components = append(a.Components, c)
	}
	sort.Slice(a.Components, func(i, j int) bool { return a.Components[i].ID < a.Components[j].ID })
	return a
}
func ValidateAssignment(fs []Fixture, s Sources, a Assignment, fixturesHash string) error {
	if err := ValidateFixtures(fs); err != nil {
		return err
	}
	expected := Assign(fs, s)
	expected.FixturesSHA256 = fixturesHash
	if !reflect.DeepEqual(expected, a) {
		return fmt.Errorf("assignment differs from deterministic family/seen/primary allocation")
	}
	return nil
}
