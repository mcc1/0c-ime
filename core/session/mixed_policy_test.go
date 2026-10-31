package session

import (
	"strings"
	"testing"

	"github.com/0c-biggo/0c-ime/core/lexicon"
	"github.com/0c-biggo/0c-ime/core/lm"
	"github.com/0c-biggo/0c-ime/core/proto"
	"github.com/0c-biggo/0c-ime/core/zhuyin"
)

func TestMixedPoliciesDistinguishChineseFromAccounts(t *testing.T) {
	for _, tc := range []struct{ keys, conservative, aggressive string }{
		{"microsoft593", "microsoft593", "microsoft窄"},
		{"microsoft593fmp6", "microsoft593群", "microsoft窄群"},
		{"microsoft5993fmp6su3cl3", "microsoft5993群你好", "microsoft5993群你好"},
		{"microsoft599fmp6su3cl3", "microsoft599fmp6你好", "microsoft599群你好"},
		{"kevin87", "kevin87", "kevi挱"},
		{"nina87", "nina87", "nin嗎"},
		{"lisa0304", "lisa0304", "lis滿04"},
		{"gpt4", "gpt4", "gp斥"},
		{"microsoftsu3cl3", "microsoft你好", "microsoft你好"},
	} {
		for _, aggressive := range []bool{false, true} {
			name := "conservative"
			want := tc.conservative
			if aggressive {
				name = "aggressive"
				want = tc.aggressive
			}
			t.Run(name+"/"+tc.keys, func(t *testing.T) {
				s := englishBoundarySession()
				s.MixedInputAggressive = aggressive
				for i := 0; i < 100; i++ {
					mixedType(s, "ji3")
					key(s, proto.KeyEnter)
				}
				if got := mixedType(s, tc.keys).Text(); got != want {
					t.Fatalf("preedit: %q, want %q", got, want)
				}
				if got := key(s, proto.KeyEnter).Commit; got != want {
					t.Fatalf("commit: %q, want %q", got, want)
				}
			})
		}
	}
}

func TestMixedPoliciesBothKeepDigitReadingsAndOriginalKeys(t *testing.T) {
	for _, aggressive := range []bool{false, true} {
		s := numberWritingSession()
		s.MixedInputAggressive = aggressive
		for i := 0; i < 100; i++ {
			mixedType(s, "ji3")
			key(s, proto.KeyEnter)
		}
		if got := mixedType(s, "593").Text(); got != "窄" {
			t.Fatalf("policy=%t: %q", aggressive, got)
		}
		items := downItems(key(s, proto.KeyDown))
		if len(items) < 2 || items[0] != "窄" || items[1] != "593" {
			t.Fatalf("policy=%t original keys: %q", aggressive, items)
		}
		s.Handle(proto.Request{Type: proto.Select, Index: 1})
		if got := key(s, proto.KeyEnter).Commit; got != "593" {
			t.Fatalf("literal selection: %q", got)
		}
	}
}

func TestMixedPoliciesKeepEnglishModeLiteral(t *testing.T) {
	for _, aggressive := range []bool{false, true} {
		for _, raw := range []string{"593", "kevin87", "lisa0304", "gpt4", "microsoft593fmp6"} {
			s := englishBoundarySession()
			s.MixedInputAggressive = aggressive
			s.Handle(proto.Request{Type: proto.Mode, On: true})
			var output strings.Builder
			for _, k := range raw {
				snap := key(s, string(k))
				output.WriteString(snap.Commit)
				if !snap.Handled {
					output.WriteRune(k)
				}
			}
			output.WriteString(key(s, proto.KeyEnter).Commit)
			if got := output.String(); got != raw {
				t.Fatalf("policy=%t English %q -> %q", aggressive, raw, got)
			}
		}
	}
}

func TestMixedPoliciesKeepSensitiveInputPrivate(t *testing.T) {
	for _, aggressive := range []bool{false, true} {
		for _, private := range []bool{false, true} {
			calls := 0
			s := neuralSession(&calls)
			s.MixedInput = true
			s.MixedInputAggressive = aggressive
			for _, k := range "ji3ru/3" {
				s.Handle(proto.Request{Type: proto.Key, Key: string(k), Private: private})
			}
			s.Handle(proto.Request{Type: proto.Commit, Private: private})
			if private {
				if calls != 0 || s.model.Store.Total != 0 {
					t.Fatalf("policy=%t private input scored/learned: %d %g", aggressive, calls, s.model.Store.Total)
				}
			} else if calls == 0 {
				t.Fatalf("policy=%t non-private control did not exercise neural scoring", aggressive)
			}
		}
	}
}

// A high-frequency shorter syllable must not push a digit-key initial into
// English in aggressive mode. Keep conservative account heuristics unchanged.
func TestMixedAggressiveKeepsDigitInitialAfterEnglish(t *testing.T) {
	for _, prefix := range []string{"hello", "WQ", "wq", "WD5"} {
		for _, neighbor := range []bool{false, true} {
			for _, aggressive := range []bool{false, true} {
				s := englishBoundarySession()
				s.MixedInputAggressive, s.NeighborKeyRecovery = aggressive, neighbor
				lex, sys := s.model.Lex.(*lexicon.Mem), s.model.LM.(*lm.Mem)
				for _, entry := range []struct {
					w, r string
					p    float64
				}{
					{"賺", "ㄓㄨㄢˋ", 1e-8}, {"萬", "ㄨㄢˋ", 1e-2},
					{"錢", "ㄑㄧㄢˊ", 1e-3}, {"賺錢", "ㄓㄨㄢˋ ㄑㄧㄢˊ", 1e-3},
				} {
					lex.Add(lexicon.Entry{Surface: entry.w, Reading: zhuyin.ParseReading(entry.r), PRW: 1})
					sys.SetUni(entry.w, entry.p)
				}
				keys := prefix + "5j04fu06"
				var got string
				if aggressive {
					first := mixedType(s, prefix+"5j04").Text()
					if first != prefix+"賺" {
						t.Fatalf("first syllable %q", first)
					}
					items := downItems(key(s, proto.KeyDown))
					if len(items) < 2 || items[1] != "5j04" {
						t.Fatalf("original keys: %v", items)
					}
					key(s, proto.KeyEscape)
					got = mixedType(s, "fu06").Text()
				} else {
					got = mixedType(s, keys).Text()
				}
				if aggressive {
					if want := prefix + "賺錢"; got != want {
						t.Fatalf("%s neighbor=%v: %q want %q", keys, neighbor, got, want)
					}
				} else {
					want := map[string]string{"hello": "hello5萬錢", "WQ": "WQ5j04fu06", "wq": "wq5j04錢", "WD5": "WD55j04fu06"}[prefix]
					if got != want {
						t.Fatalf("conservative %s: %q want %q", keys, got, want)
					}
				}
				if commit := key(s, proto.KeyEnter).Commit; commit != got {
					t.Fatalf("commit %q != preedit %q", commit, got)
				}
			}
		}
	}
}
