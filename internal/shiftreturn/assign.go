//go:build shiftreturneval

package shiftreturn

import (
	"fmt"
	"math/big"
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
func fixtureSeen(f Fixture, s Sources) bool {
	has := func(a []string, v string) bool {
		for _, x := range a {
			if x == v {
				return true
			}
		}
		return false
	}
	if has(s.SeenRaw, f.ExpectedRaw) {
		return true
	}
	for _, g := range f.Gold {
		prefix := ""
		readings := []string{}
		for _, seg := range g.Segments {
			if seg.Kind == "literal" && len(readings) == 0 {
				prefix += f.ExpectedRaw[seg.Begin:seg.End]
			}
			if seg.Kind == "chinese" {
				readings = append(readings, seg.Reading)
			}
		}
		if has(s.SeenPrefixes, strings.ToLower(prefix)) {
			return true
		}
		if len(readings) > 0 {
			if has(s.SeenContents, strings.Join(strings.Fields(strings.Join(readings, " ")), " ")) {
				return true
			}
			for _, surface := range g.AcceptedSurfaces {
				if strings.HasPrefix(surface, prefix) && has(s.SeenContents, strings.TrimPrefix(surface, prefix)) {
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
		keys := append([]string{}, f.Families...)
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
			regression = regression || fixtureSeen(f, s)
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
