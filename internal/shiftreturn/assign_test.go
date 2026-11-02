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
func TestAssignmentDerivedDependenciesAndNormalizedSeen(t *testing.T) {
	makeEvents := func(raw string) []Event {
		events := []Event{{Type: "shift"}}
		for _, key := range raw {
			events = append(events, Event{Type: "key", Key: string(key)})
		}
		return append(events, Event{Type: "enter"})
	}
	first := demoFixtures(t)[0]
	first.Families = []string{"unrelated-a"}
	second := first
	second.ID = "another"
	second.PrototypeID = "another-prototype"
	second.Families = []string{"unrelated-b"}
	second.ExpectedRaw = "abcdefghi593vul3"
	second.Events = makeEvents(second.ExpectedRaw)
	second.Gold = []Gold{{Segments: append([]Segment{}, first.Gold[0].Segments...), AcceptedSurfaces: []string{"abcdefghi窄筱"}}}
	if e := ValidateFixtures([]Fixture{first, second}); e != nil {
		t.Fatal(e)
	}
	if len(Assign([]Fixture{first, second}, emptySources()).Components) != 1 {
		t.Fatal("same target readings escaped family closure")
	}
	second.ExpectedRaw = "MICROSOFTgp "
	second.Events = makeEvents(second.ExpectedRaw)
	second.Gold = []Gold{{Segments: []Segment{{Kind: "literal", Begin: 0, End: 9}, {Kind: "chinese", Begin: 9, End: 12, Reading: "ㄕㄣ"}}, AcceptedSurfaces: []string{"MICROSOFT身"}}}
	if e := Validate(second); e != nil {
		t.Fatal(e)
	}
	if len(Assign([]Fixture{first, second}, emptySources()).Components) != 1 {
		t.Fatal("case-normalized exact prefix escaped closure")
	}
	for _, s := range []Sources{func() Sources { s := emptySources(); s.SeenPrefixes = []string{"MICROSOFT"}; return s }(), func() Sources { s := emptySources(); s.SeenContents = []string{" ㄓㄞˇ  ㄒㄧㄠˇ "}; return s }()} {
		if Assign([]Fixture{first}, s).Components[0].Split != "regression" {
			t.Fatal("normalized prefix/reading seen match failed")
		}
	}
	first.Gold = []Gold{{Segments: first.Gold[0].Segments, AcceptedSurfaces: []string{"microsoftÅ"}}}
	s := emptySources()
	s.SeenContents = []string{"A\u030a"}
	if Assign([]Fixture{first}, s).Components[0].Split != "regression" {
		t.Fatal("NFC surface seen match failed")
	}
}
func TestAssignmentSeenDeclaredContentAndLiteralPrefix(t *testing.T) {
	f := demoFixtures(t)[4]
	f.Families = []string{"prefix:KEVIN", "word:窄"}
	s := emptySources()
	s.SeenPrefixes = []string{"kevin"}
	if Assign([]Fixture{f}, s).Components[0].Split != "regression" {
		t.Fatal("literal task reused a declared seen English prefix")
	}
	s = emptySources()
	s.SeenContents = []string{"窄"}
	if Assign([]Fixture{f}, s).Components[0].Split != "regression" {
		t.Fatal("declared target content prototype escaped seen exclusion")
	}
}
