//go:build mixedeval

package session

import (
	"encoding/json"
	"github.com/0c-biggo/0c-ime/core/english"
	"github.com/0c-biggo/0c-ime/core/layout"
	"github.com/0c-biggo/0c-ime/core/lexicon"
	"github.com/0c-biggo/0c-ime/core/lm"
	"github.com/0c-biggo/0c-ime/core/mixedparse"
	"github.com/0c-biggo/0c-ime/core/personal"
	"github.com/0c-biggo/0c-ime/core/proto"
	"github.com/0c-biggo/0c-ime/core/zhuyin"
	"reflect"
	"strings"
	"testing"
)

func TestMixedEvalObserverParity(t *testing.T) {
	events := 0
	for _, aggressive := range []bool{false, true} {
		for _, raw := range []string{"WQ5j04fu06", "hello5j04fu06", "hellosu3cl3", "microsoft593fmp6", "593", "kevin87", "nina87", "lisa0304", "gpt4", "h264", "gpt4o", "testsu.3", "appsul4", "useru/4", "api2k7", strings.Repeat("A", 65)} {
			base := englishBoundarySession()
			base.Preview02 = true
			base.MixedInputAggressive = aggressive
			base.Opt.DocumentContextWeight = 0
			b, err := NewMixedEvalBaseline(base.model.Lex, base.model.LM, base.English, aggressive, nil)
			if err != nil {
				t.Fatal(err)
			}
			for _, k := range raw {
				req := proto.Request{Type: proto.Key, Key: string(k)}
				got := b.Handle(req)
				want := base.Handle(req)
				if !reflect.DeepEqual(got.Snapshot, want) {
					t.Fatalf("%s %c snapshot parity\ngot%+v\nwant%+v", raw, k, got.Snapshot, want)
				}
				a, _ := json.Marshal(base.model.Store)
				z, _ := json.Marshal(b.s.model.Store)
				if string(a) != string(z) {
					t.Fatal("store parity")
				}
				events++
			}
			for _, req := range []proto.Request{{Type: proto.Key, Key: proto.KeyDown}, {Type: proto.Cancel}} {
				if got, want := b.Handle(req), base.Handle(req); !reflect.DeepEqual(got.Snapshot, want) {
					t.Fatal("finalizer parity")
				}
				events++
			}
			b.Close()
		}
	}
	t.Logf("observer-on/off snapshot+store parity events=%d", events)
}
func TestMixedEvalGuardProjectionParity(t *testing.T) {
	base := englishBoundarySession()
	e := mixedparse.Evaluator{Lex: base.model.Source(), Score: base.model, System: base.model, English: base.English}
	contexts := []mixedparse.Context{{}, {Units: []mixedparse.Unit{{Literal: true, Text: "3"}}}, {Units: []mixedparse.Unit{{Reading: "ㄋㄧˇ"}}}, {Units: []mixedparse.Unit{{Literal: true, Text: " "}}}, {Units: []mixedparse.Unit{{Literal: true, Text: "A"}}}, {Units: []mixedparse.Unit{{Literal: true, Text: "ㄋ"}, {Literal: true, Text: "ㄧ"}, {Literal: true, Text: "ˇ"}}, RetainedPrefix: &mixedparse.Retained{Reading: "ㄋㄧˇ", OriginalKeys: "su3"}}}
	count := 0
	for _, ctx := range contexts {
		for _, prefix := range []string{"", "hello", "WQ", "3", "hello2"} {
			for _, body := range []string{"5j0", "593", "su", "u.", "ul", "u/", "2k", "87", "1/", ".4", "5.", "81", "2/", ".", ",", "5", "k", "hello2.", "", "uu", "u5"} {
				for _, key := range []string{"3", "4", "6", "7", " "} {
					c := mixedparse.Project(ctx, prefix)
					s := englishBoundarySession()
					s.Preview02 = true
					s.English = base.English
					s.syls = nil
					s.literals = map[int]bool{}
					for i, u := range c.Units {
						if u.Literal {
							s.syls = append(s.syls, zhuyin.Syllable(u.Text))
							s.literals[i] = true
						} else {
							s.syls = append(s.syls, zhuyin.Syllable(u.Reading))
						}
					}
					s.cursor = len(s.syls)
					if c.RetainedPrefix != nil {
						var comp zhuyin.Composer
						for _, r := range string(zhuyin.Syllable(c.RetainedPrefix.Reading).Base()) {
							comp.Add(r)
						}
						s.pendingLiterals = []pendingLiteral{{start: 0, composer: comp, tone: 3, prefix: true}}
					}
					tone, _ := layout.Standard.Symbol(rune(key[0]))
					raw := prefix + body + key
					got := e.Guards(ctx, raw, prefix, body, key, tone)
					want := [13]bool{numberLike(raw), numberLike(prefix), s.englishThenNumber(body, key), s.digitToneNumber(body, tone), s.numberAfterLiteral(body, key), s.rareDigitReading(body, tone), s.literalDigitBefore(), s.englishThenPunct(body), s.mixedLiteralPrefix(prefix, body, key), s.digitWord(body, tone), s.punctKeyWord(body), s.digitsReadAsChinese(body, tone), s.mixedSpaceNumber(body, tone)}
					if got != want {
						t.Fatalf("context%+v p%q body%q key%q got%v want%v", ctx, prefix, body, key, got, want)
					}
					count++
				}
			}
		}
	}
	t.Logf("guard vectors=%d (13 guards each)", count)
}
func TestMixedEvalRowsOwnDisposableStore(t *testing.T) {
	s := englishBoundarySession()
	a, _ := NewMixedEvalBaseline(s.model.Lex, s.model.LM, s.English, true, nil)
	b, _ := NewMixedEvalBaseline(s.model.Lex, s.model.LM, s.English, true, nil)
	defer a.Close()
	defer b.Close()
	for _, k := range "su3" {
		a.Handle(proto.Request{Type: proto.Key, Key: string(k)})
	}
	a.Handle(proto.Request{Type: proto.Key, Key: proto.KeyEnter})
	if b.s.model.Store.Total != 0 || a.s.model.Store == b.s.model.Store {
		t.Fatal("cross-row state")
	}
	if a.s.model.Now != nil || b.s.model.P != personal.DefaultParams() {
		t.Fatal("unfixed model")
	}
}

func TestMixedEvalCustomContextAndCommitLedger(t *testing.T) {
	s := englishBoundarySession()
	b, err := NewMixedEvalBaseline(s.model.Lex, s.model.LM, s.English, true, []string{"WQ"})
	if err != nil {
		t.Fatal(err)
	}
	defer b.Close()
	if !reflect.DeepEqual(b.Observe().Context.CustomTokens, []string{"WQ"}) {
		t.Fatal("custom tokens lost")
	}
	for _, k := range "su3" {
		b.Handle(proto.Request{Type: proto.Key, Key: string(k)})
	}
	before := b.Observe()
	after := b.Handle(proto.Request{Type: proto.Key, Key: proto.KeyEnter})
	if before.OwnershipStatus != "mapped" || after.OwnershipStatus != "mapped" || !reflect.DeepEqual(before.Partition, after.Partition) {
		t.Fatal("commit ledger", before, after)
	}
}

func TestMixedEvalPrefixOnlyObserverJoin(t *testing.T) {
	lex := lexicon.NewMem()
	lex.Add(lexicon.Entry{Surface: "詞頭", Reading: zhuyin.ParseReading("ㄟˋ ㄌㄨㄛˊ"), PRW: 1})
	sys := lm.NewMem()
	sys.SetUni("詞頭", .1)
	b, err := NewMixedEvalBaseline(lex, sys, english.Prototype(), true, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer b.Close()
	var first MixedEvalObservation
	for _, k := range "o4" {
		first = b.Handle(proto.Request{Type: proto.Key, Key: string(k)})
	}
	if first.OwnershipStatus != "mapped" || first.Context.RetainedPrefix == nil || len(first.Partition) != 1 || first.Partition[0].Reading != "ㄟˋ" {
		t.Fatalf("prefix-only provenance %+v", first)
	}
	for _, k := range "xji6" {
		b.Handle(proto.Request{Type: proto.Key, Key: string(k)})
	}
	last := b.Observe()
	if last.OwnershipStatus != "mapped" || len(last.Partition) != 2 || last.Partition[0].Begin != 0 || last.Partition[0].End != 2 || last.Partition[1].End != 6 {
		t.Fatalf("joined provenance %+v", last)
	}
}
