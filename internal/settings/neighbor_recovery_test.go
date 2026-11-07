package settings

import (
	"encoding/json"
	"path/filepath"
	"testing"
)

func TestNeighborRecoveryIsIndependentOptInAndPersists(t *testing.T) {
	s := Default()
	if s.NeighborKeyRecovery {
		t.Fatal("recovery must default off")
	}
	if err := s.Set("neighbor-key-recovery", "on"); err != nil {
		t.Fatal(err)
	}
	if s.MixedInput || s.MixedInputPolicy != MixedConservative {
		t.Fatal("independent setting changed input mode or policy")
	}
	p := filepath.Join(t.TempDir(), FileName)
	if err := Save(p, s); err != nil {
		t.Fatal(err)
	}
	s, err := Load(p)
	if err != nil || !s.NeighborKeyRecovery {
		t.Fatalf("reload: %+v %v", s, err)
	}
	for _, tc := range []struct {
		mixed  bool
		policy string
		by     string
	}{
		{false, MixedConservative, "mixed-input"}, {true, MixedConservative, "mixed-input-policy"}, {true, MixedAggressive, ""},
	} {
		s.MixedInput, s.MixedInputPolicy = tc.mixed, tc.policy
		found := false
		for _, it := range s.Items() {
			if it.Key == "neighbor-key-recovery" {
				found = true
				if it.DisabledBy != tc.by || it.Disabled != (tc.by != "") || it.Value != "on" {
					t.Fatalf("availability: %+v", it)
				}
			}
		}
		if !found {
			t.Fatal("missing user-facing switch")
		}
	}
	before := s
	if err := s.Set("neighbor-key-recovery", "invalid"); err == nil || s != before {
		t.Fatal("invalid value accepted or mutated settings")
	}
	if err := json.Unmarshal([]byte(`{"neighbor_key_recovery":"future"}`), &s); err != nil || s.NeighborKeyRecovery {
		t.Fatalf("future fallback: %+v %v", s, err)
	}
	b, _ := json.Marshal(s)
	var raw map[string]any
	json.Unmarshal(b, &raw)
	if raw["neighbor_key_recovery"] != "future" {
		t.Fatalf("future setting lost: %s", b)
	}
}
