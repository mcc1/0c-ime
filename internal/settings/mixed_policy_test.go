package settings

import (
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"
)

func TestMixedPolicyDefaultsPersistsAndKeepsFutureValues(t *testing.T) {
	path := filepath.Join(t.TempDir(), FileName)
	s, err := Load(path)
	if err != nil || s.MixedInputPolicy != MixedConservative {
		t.Fatalf("default: %+v %v", s, err)
	}
	for _, policy := range []string{MixedAggressive, MixedConservative, MixedAggressive} {
		if err = s.Set("mixed-input-policy", policy); err != nil {
			t.Fatal(err)
		}
		if err = Save(path, s); err != nil {
			t.Fatal(err)
		}
		s, err = Load(path)
		if err != nil || s.MixedInputPolicy != policy {
			t.Fatalf("reload: %+v %v", s, err)
		}
	}
	before := s
	if err = s.Set("mixed-input-policy", "invalid"); err == nil || s != before {
		t.Fatal("invalid policy accepted or mutated settings")
	}
	if err = json.Unmarshal([]byte(`{"mixed_input_policy":"future","mixed_input":true}`), &s); err != nil {
		t.Fatal(err)
	}
	if s.MixedInputPolicy != MixedConservative || !s.MixedInput {
		t.Fatalf("future fallback: %+v", s)
	}
	b, _ := json.Marshal(s)
	if !strings.Contains(string(b), `"mixed_input_policy":"future"`) {
		t.Fatalf("future value lost: %s", b)
	}
	s.Set("mixed-input-policy", MixedAggressive)
	b, _ = json.Marshal(s)
	if !strings.Contains(string(b), `"mixed_input_policy":"aggressive"`) {
		t.Fatalf("explicit change lost: %s", b)
	}
}

func TestMixedPolicyAvailabilityAndLabels(t *testing.T) {
	s := Default()
	policy := func() Item {
		for _, it := range s.Items() {
			if it.Key == "mixed-input-policy" {
				return it
			}
		}
		t.Fatal("policy missing")
		return Item{}
	}
	it := policy()
	if !it.Disabled || it.DisabledBy != "mixed-input" || it.Value != MixedConservative {
		t.Fatalf("off: %+v", it)
	}
	s.MixedInput = true
	s.MixedInputPolicy = MixedAggressive
	it = policy()
	if it.Disabled || it.Value != MixedAggressive || it.ValueLabels[MixedAggressive] != "積極" || it.ValueLabels[MixedConservative] != "保守" {
		t.Fatalf("on: %+v", it)
	}
	for _, layout := range []string{LayoutETen, LayoutHsu} {
		s.KeyboardLayout = layout
		it = policy()
		if !it.Disabled || it.DisabledBy != "keyboard-layout" || it.Value != MixedAggressive {
			t.Fatalf("layout: %+v", it)
		}
	}
	s.KeyboardLayout = LayoutStandard
	s.Toneless = true
	if it = policy(); !it.Disabled || it.DisabledBy != "toneless" {
		t.Fatalf("toneless: %+v", it)
	}
}
