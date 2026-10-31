//go:build mixedeval

package mixedeval

import (
	"github.com/0c-biggo/0c-ime/core/mixedparse"
	"math"
	"testing"
)

func rankCandidate(p []mixedparse.Segment, score float64) mixedparse.Candidate {
	c := mixedparse.Candidate{Segments: p, Lane: "strict"}
	c.ID = mixedparse.CandidateID(p)
	c.Identity = mixedparse.Identity(p)
	for i := range c.Features {
		v := 0.0
		c.Features[i] = &v
	}
	c.Features[1] = &score
	return c
}
func TestRankedMarginIncumbentAndFallback(t *testing.T) {
	m := LinearModel{Version: "mixed-linear-v1", FeatureVersion: mixedparse.FeatureVersion, Coefficients: make([]float64, 26), Margin: .5}
	m.Coefficients[1] = 1
	old := []mixedparse.Segment{{Kind: "literal", Begin: 0, End: 3, Text: "WQ5"}, {Kind: "chinese", Begin: 3, End: 6, Reading: "ㄨㄢˋ"}}
	inc := rankCandidate(append(append([]mixedparse.Segment(nil), old...), mixedparse.Segment{Kind: "chinese", Begin: 6, End: 10, Reading: "ㄑㄧㄢˊ"}), 1)
	newc := rankCandidate([]mixedparse.Segment{{Kind: "literal", Begin: 0, End: 2, Text: "WQ"}, {Kind: "chinese", Begin: 2, End: 6, Reading: "ㄓㄨㄢˋ"}, {Kind: "chinese", Begin: 6, End: 10, Reading: "ㄑㄧㄢˊ"}}, 1.5)
	w := mixedparse.Window{Raw: "WQ5j04fu06", Shown: old, Visible: "WQ5萬", FirstToneEnd: 6}
	got, e := m.Choose(w, []mixedparse.Candidate{newc, inc}, "")
	if e != nil || got.ID != inc.ID {
		t.Fatal("strict >margin", got, e)
	}
	m.Margin = .49
	got, e = m.Choose(w, []mixedparse.Candidate{newc, inc}, "")
	if e != nil || got.ID != newc.ID {
		t.Fatal("challenger", got, e)
	}
	got, e = m.Choose(w, []mixedparse.Candidate{newc}, "")
	if e != nil || got.Lane != "fixed_fallback" || got.Text != "WQ5萬fu06" || mixedparse.ProjectedIdentity(got.Segments, 6) != mixedparse.Identity(old) {
		t.Fatal("no compatible extension", got, e)
	}
}
func TestRankedExactTiesAndFiniteScores(t *testing.T) {
	m := LinearModel{Version: "mixed-linear-v1", FeatureVersion: mixedparse.FeatureVersion, Coefficients: make([]float64, 26)}
	a := rankCandidate([]mixedparse.Segment{{Kind: "literal", Begin: 0, End: 2, Text: "WQ"}, {Kind: "chinese", Begin: 2, End: 6, Reading: "ㄓㄨㄢˋ"}}, 0)
	b := rankCandidate([]mixedparse.Segment{{Kind: "literal", Begin: 0, End: 3, Text: "WQ5"}, {Kind: "chinese", Begin: 3, End: 6, Reading: "ㄨㄢˋ"}}, 0)
	got, e := m.Choose(mixedparse.Window{}, []mixedparse.Candidate{b, a}, "")
	if e != nil || got.ID != a.ID {
		t.Fatal("offset tie", got, e)
	}
	got, e = m.Choose(mixedparse.Window{}, []mixedparse.Candidate{a, b}, b.ID)
	if e != nil || got.ID != b.ID {
		t.Fatal("legacy exact tie", got, e)
	}
	m.Coefficients[1] = math.Inf(1)
	if m.Validate() == nil {
		t.Fatal("nonfinite model accepted")
	}
}
