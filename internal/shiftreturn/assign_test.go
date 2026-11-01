//go:build shiftreturneval

package shiftreturn

import (
	"encoding/json"
	"os"
	"reflect"
	"testing"
)

func emptySources() Sources {
	return Sources{Version: Version, Kind: "sources", Demo: true, SeenPrefixes: []string{}, SeenContents: []string{}, SeenRaw: []string{}}
}
func TestAssignmentClosureAndPrimary(t *testing.T) {
	fs := demoFixtures(t)
	s := emptySources()
	a := Assign(fs, s)
	if len(a.Components) != 3 {
		t.Fatalf("closure: got %d components", len(a.Components))
	}
	reversed := append([]Fixture{}, fs...)
	for i, j := 0, len(reversed)-1; i < j; i, j = i+1, j-1 {
		reversed[i], reversed[j] = reversed[j], reversed[i]
	}
	if !reflect.DeepEqual(a, Assign(reversed, s)) {
		t.Fatal("input order changed assignment")
	}
	a.FixturesSHA256 = HashBytes([]byte("original JSONL\n"))
	if e := ValidateAssignment(fs, s, a, a.FixturesSHA256); e != nil {
		t.Fatal(e)
	}
	a.Components[0].Split = "bogus"
	if e := ValidateAssignment(fs, s, a, a.FixturesSHA256); e == nil {
		t.Fatal("cross-family assignment accepted")
	}
	// Public priority precedes ID; metadata belongs to the selected primary,
	// even when a different variant declares a contrast or a harder stratum.
	fs = fs[:1]
	v := fs[0]
	v.ID = "A"
	v.SourcePriority = 1
	v.ContextClass = "contrast"
	v.Stratum = "17-64"
	fs[0].ID = "B"
	fs[0].SourcePriority = 0
	fs[0].ContextClass = "none"
	fs[0].Stratum = "1"
	a = Assign(append(fs, v), s)
	p := a.Components[0].PrimaryReturnMetadata
	if p == nil || p.ID != "B" || p.Stratum != "1" || p.ContrastEligible {
		t.Fatalf("variant metadata contaminated primary: %+v", p)
	}
}
func TestAssignmentSeenWholeComponent(t *testing.T) {
	fs := demoFixtures(t)[:2]
	s := emptySources()
	a := Assign(fs, s)
	if a.Components[0].Split == "regression" {
		t.Fatal("unexpected seen")
	}
	for _, manifest := range []Sources{
		func() Sources { x := s; x.SeenPrefixes = []string{"microsoft"}; return x }(),
		func() Sources { x := s; x.SeenContents = []string{"窄小"}; return x }(),
		func() Sources { x := s; x.SeenContents = []string{"ㄓㄞˇ ㄒㄧㄠˇ"}; return x }(),
		func() Sources { x := s; x.SeenRaw = []string{"microsoft593vul3"}; return x }(),
	} {
		b := Assign(fs, manifest)
		if len(b.Components) != 1 || b.Components[0].Split != "regression" || len(b.Components[0].RowIDs) != 2 {
			t.Fatal("seen failed to propagate")
		}
		if b.SeenSHA256 == a.SeenSHA256 {
			t.Fatal("seen hash unchanged")
		}
	}
	if SeenHash(s) != "fcfc2ca31ec93a6f0baa22f67a5fb8699feac24e7d9f15afc310c7b31f6f5822" {
		t.Fatal("Python canonical seen hash mismatch")
	}
	b, e := os.ReadFile("../../design/shift-return-v2/seen-example.json")
	if e != nil {
		t.Fatal(e)
	}
	var example struct {
		RowIDs      []string `json:"row_ids"`
		ComponentID string   `json:"component_id"`
	}
	if e = json.Unmarshal(b, &example); e != nil {
		t.Fatal(e)
	}
	if HashBytes(Canonical(example.RowIDs)) != example.ComponentID {
		t.Fatal("Python canonical component hash mismatch")
	}
}
func TestAssignmentImplicitCollision(t *testing.T) {
	fs := demoFixtures(t)[:2]
	fs[0].Families = []string{"a"}
	fs[1].Families = []string{"b"}
	a := Assign(fs, emptySources())
	if len(a.Components) != 1 {
		t.Fatal("same observable input split despite opposite intents")
	}
}
