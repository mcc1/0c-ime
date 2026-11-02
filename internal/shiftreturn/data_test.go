//go:build shiftreturneval

package shiftreturn

import (
	"encoding/json"
	"math"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/0c-biggo/0c-ime/core/decode"
	"github.com/0c-biggo/0c-ime/core/input"
	"github.com/0c-biggo/0c-ime/core/lexicon"
	"github.com/0c-biggo/0c-ime/core/proto"
	"github.com/0c-biggo/0c-ime/core/zhuyin"
	"github.com/0c-biggo/0c-ime/internal/datadir"
)

func TestFeatureLegacyScoreSourceParity(t *testing.T) {
	lex, sys, eng := baselineSources()
	p, m := NewScoringProbe(lex, sys, eng)
	if _, ok := p.Lex.(decode.Layered); !ok {
		t.Fatal("scoring source lost Layered interface")
	}
	if _, ok := p.Lex.(lexicon.Long); !ok {
		t.Fatal("scoring source lost Long interface")
	}
	// This is the exact source-map expression in Session.mixedSyllableScore;
	// it is unexported, and production source is deliberately unchanged.
	for _, s := range []string{"ㄓㄞˇ", "ㄒㄧㄠˇ", "ㄋㄩㄝˋ"} {
		r := []zhuyin.Syllable{zhuyin.Syllable(s)}
		d := decode.Decoder{Lex: m.Source(), Score: m}
		_, legacyLog := d.Best((input.Exact{}).Expand(r), nil, true)
		_, log, e := p.best(r, true)
		if e != nil {
			t.Fatal(e)
		}
		if math.Exp(log) != math.Exp(legacyLog) {
			t.Fatal("source helper expression differs")
		}
	}
	// Prefix-only source, complete source, and no source have distinct paths.
	tiny := tinyProbe(false)
	_, log, e := tiny.best(zhuyin.ParseReading("ㄓㄞˇ"), true)
	if e != nil || math.Abs(log+2) > 1e-12 {
		t.Fatal("prefix source parity", log, e)
	}
}

// This inventory reads public dictionaries and a seen manifest only. It does
// not call Generate, train, score tasks, or inspect prior quality outcomes.
func TestPublicDataInventory(t *testing.T) {
	if os.Getenv("SHIFTRETURN_DATA_INVENTORY") != "1" {
		t.Skip("explicit read-only public data inventory")
	}
	eng, e := datadir.Open("../../assets/data")
	if e != nil {
		t.Fatal(e)
	}
	if eng.EnglishPack == "" {
		t.Fatal("prototype English fallback")
	}
	b, e := os.ReadFile("/workspace/scratch/shift-return-v2-parent/seen-draft.json")
	if e != nil {
		t.Fatal(e)
	}
	var seen struct {
		Contents []string `json:"seen_contents"`
	}
	if e = json.Unmarshal(b, &seen); e != nil {
		t.Fatal(e)
	}
	excluded := map[string]bool{}
	for _, s := range seen.Contents {
		excluded[s] = true
	}
	all, ok := eng.Model.Lex.(interface{ All(func(lexicon.Entry)) })
	if !ok {
		t.Fatal("lexicon inventory unavailable")
	}
	byLength, unseen := map[int]int{}, map[int]int{}
	unique := map[string]bool{}
	unseenUnique := map[string]bool{}
	firstSpace := 0
	uniqueReadings := map[int]map[string]bool{}
	all.All(func(entry lexicon.Entry) {
		n := len(entry.Reading)
		byLength[n]++
		reading := zhuyin.Key(entry.Reading)
		unique[entry.Surface+"\t"+reading] = true
		if !excluded[entry.Surface] && !excluded[strings.Join(strings.Fields(reading), " ")] {
			unseen[n]++
			if uniqueReadings[n] == nil {
				uniqueReadings[n] = map[string]bool{}
			}
			uniqueReadings[n][reading] = true
			unseenUnique[entry.Surface+"\t"+reading] = true
			if n > 0 && entry.Reading[0].ToneOf() == 1 {
				firstSpace++
			}
		}
	})
	readingCounts := map[int]int{}
	for n, rs := range uniqueReadings {
		readingCounts[n] = len(rs)
	}
	t.Logf("draft-unseen unique reading prototypes by length=%v", readingCounts)
	t.Logf("public dictionary: English=%d Chinese unique surface-reading=%d draft-unseen unique=%d draft-unseen first-tone entries=%d lengths=%v draft-unseen-lengths=%v", eng.English.Len(), len(unique), len(unseenUnique), firstSpace, byLength, unseen)
}
func TestBaselinePublicParity(t *testing.T) {
	if os.Getenv("SHIFTRETURN_DATA_INVENTORY") != "1" {
		t.Skip("explicit fixed public parity")
	}
	loaded, e := datadir.Open("../../assets/data")
	if e != nil {
		t.Fatal(e)
	}
	for _, condition := range []string{"M", "R", "C", "A"} {
		for _, raw := range []string{"kevin87", "nina87", "lisa0304", "gpt4", "h264", "gpt4o", "hello4", "zoom3", "hello2.4", "x593gp "} {
			on, e := NewBaseline(loaded.Model.Lex, loaded.Model.LM, loaded.English, condition, nil, true)
			if e != nil {
				t.Fatal(e)
			}
			off, e := NewBaseline(loaded.Model.Lex, loaded.Model.LM, loaded.English, condition, nil, false)
			if e != nil {
				t.Fatal(e)
			}
			requests := []proto.Request{}
			for _, key := range "5j/ " {
				requests = append(requests, proto.Request{Type: proto.Key, Key: string(key)})
			}
			requests = append(requests, proto.Request{Type: proto.Toggle})
			for _, key := range raw {
				requests = append(requests, proto.Request{Type: proto.Key, Key: string(key)})
			}
			requests = append(requests, proto.Request{Type: proto.Key, Key: proto.KeyEnter})
			for _, request := range requests {
				a, b := on.Handle(request), off.Handle(request)
				if !reflect.DeepEqual(a.Snapshot, b.Snapshot) || a.View != b.View || a.App != b.App || StoreDigest(on.Model) != StoreDigest(off.Model) {
					t.Fatalf("public observer parity %s/%s", condition, raw)
				}
			}
		}
	}
}
func TestLimitsFixedCalibration(t *testing.T) {
	if os.Getenv("SHIFTRETURN_DATA_INVENTORY") != "1" {
		t.Skip("explicit20-case operational calibration")
	}
	loaded, e := datadir.Open("../../assets/data")
	if e != nil {
		t.Fatal(e)
	}
	// Calibrate's reviewed artifacts are resolved from repository root.
	old, e := os.Getwd()
	if e != nil {
		t.Fatal(e)
	}
	if e = os.Chdir("../.."); e != nil {
		t.Fatal(e)
	}
	defer os.Chdir(old)
	cases, e := Calibrate(loaded)
	if e != nil {
		t.Fatal(e)
	}
	if len(cases) != 20 {
		t.Fatal(len(cases))
	}
	for _, c := range cases {
		t.Logf("calibration: %s", Canonical(c))
	}
}
