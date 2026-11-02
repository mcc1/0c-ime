//go:build shiftreturneval

package shiftreturn

import (
	"github.com/0c-biggo/0c-ime/core/engine"
	"github.com/0c-biggo/0c-ime/core/lm"
	"testing"
)

func TestFeatureFrozenCoefficientSensitivitySeparate(t *testing.T) {
	probe := tinyProbe(false)
	sys := lm.NewMem()
	sys.SetUni("窄小", .1)
	loaded := engine.New(probe.Lex, sys)
	f := demoFixtures(t)[0]
	a := Assign([]Fixture{f}, emptySources())
	m := modelForZero()
	m.Name = "EC"
	m.Coefficients[9] = 1
	row, _, occurrences, _, _, err := RunEpisode(f, a.Components[0], "EC-revise", m, "revise", loaded, false)
	if err != nil {
		t.Fatal(err)
	}
	if row.FinalBoundaryError == nil || *row.FinalBoundaryError {
		t.Fatal("current W did not recover tiny word")
	}
	rows, err := FirstWordSensitivity([]Fixture{f}, occurrences, m)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 || rows[0]["final_boundary_error"] != true {
		t.Fatal("first W replacement failed to remove future word evidence")
	}
	if m.Coefficients[9] != 1 {
		t.Fatal("sensitivity changed/refit coefficients")
	}
}
