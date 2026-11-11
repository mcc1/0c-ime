package session

import (
	"strings"
	"testing"

	"github.com/0c-biggo/0c-ime/core/lexicon"
	"github.com/0c-biggo/0c-ime/core/lm"
	"github.com/0c-biggo/0c-ime/core/personal"
	"github.com/0c-biggo/0c-ime/core/proto"
	"github.com/0c-biggo/0c-ime/core/zhuyin"
)

// Conservative expectations follow upstream 0.2.10 account protection. Digit
// typo recovery beyond that protection belongs to the aggressive policy.
func TestMixedPoliciesDistinguishChineseFromAccounts(t *testing.T) {
	for _, tc := range []struct{ keys, conservative, aggressive string }{
		{"microsoft593", "microsoft593", "microsoft窄"},
		{"microsoft593fmp6", "microsoft593fmp6", "microsoft窄群"},
		{"microsoft5993fmp6su3cl3", "microsoft5993fmp6你好", "microsoft5993群你好"},
		{"microsoft599fmp6su3cl3", "microsoft599fmp6你好", "microsoft599群你好"},
		{"kevin87", "kevin87", "kevi挱"},
		{"nina87", "nina87", "nin嗎"},
		{"lisa0304", "lisa0304", "lis滿04"},
		{"gpt4", "gpt4", "gpt4"},
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
// English in aggressive mode. Conservative follows upstream rules: since 0.2.8
// (#241) an uppercase or two-letter prefix is itself a boundary (WQ賺錢, wq賺錢).
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
					want := map[string]string{"hello": "hello5萬錢", "WQ": "WQ賺錢", "wq": "wq賺錢", "WD5": "WD5賺錢"}[prefix]
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

// 積極模式自己的切法切不開時，照舊走保守的 #241 大寫邊界（V了、G盤），積極不會比保守少切；
// #249 照字面的英數詞（ps4、x86）在兩種模式都先於積極切法。
func TestMixedAggressiveKeepsConservativeTokenBoundaries(t *testing.T) {
	for _, tc := range []struct{ keys, want string }{
		{"Vxk7", "V了"}, {"Gq06", "G盤"}, {"GCP187", "GCP吧"}, {"OKa87", "OK嗎"}, {"oka87", "ok嗎"},
		{"OKg4", "OK是"}, {"Q2h96", "Q2財"}, {"CoWoSq06", "CoWoS盤"}, {"GCP187ji3", "GCP吧我"},
	} {
		s := tokenBoundarySession()
		s.MixedInputAggressive = true
		// 真實字典裡有的罕用切法（Vx＋呃、Gq＋玵）也要輸給大寫邊界。
		lex, sys := s.model.Lex.(*lexicon.Mem), s.model.LM.(*lm.Mem)
		for _, e := range []struct{ w, r string }{{"呃", "ㄜ˙"}, {"玵", "ㄢˊ"}} {
			lex.Add(lexicon.Entry{Surface: e.w, Reading: zhuyin.ParseReading(e.r), PRW: 1})
			sys.SetUni(e.w, 1e-4)
		}
		mixedType(s, tc.keys)
		if got := key(s, proto.KeyEnter).Commit; got != tc.want {
			t.Errorf("aggressive %q: %q, want %q", tc.keys, got, tc.want)
		}
	}
	for _, aggressive := range []bool{false, true} {
		for _, raw := range []string{"ps4", "x86"} {
			s := englishBoundarySession()
			s.MixedInputAggressive = aggressive
			if got := mixedType(s, raw).Text(); got != raw {
				t.Errorf("policy=%t %q: %q", aggressive, raw, got)
			}
		}
	}
}

// 學過的英數詞（英文模式打過的 wq5）以數字聲母鍵結尾時，積極模式把那個鍵讓給後面完整的讀音（wq＋賺），
// 保守模式照上游 #249 留字面前段。要照字面請先按 Shift 切英文。
func TestMixedAggressiveLearnedTokenYieldsDigitInitial(t *testing.T) {
	for _, aggressive := range []bool{false, true} {
		s := englishBoundarySession()
		s.MixedInputAggressive = aggressive
		lex, sys := s.model.Lex.(*lexicon.Mem), s.model.LM.(*lm.Mem)
		for _, entry := range []struct {
			w, r string
			p    float64
		}{{"賺", "ㄓㄨㄢˋ", 1e-5}, {"萬", "ㄨㄢˋ", 1e-2}} {
			lex.Add(lexicon.Entry{Surface: entry.w, Reading: zhuyin.ParseReading(entry.r), PRW: 1})
			sys.SetUni(entry.w, entry.p)
		}
		s.model.Store.English = map[string]personal.EnglishEvidence{"wq5": {Text: "wq5", Confirmed: 5, Days: 3, Usage: 5}}
		if !s.literalToken("wq5") {
			t.Fatal("fixture: wq5 not learned")
		}
		want := "wq5萬"
		if aggressive {
			want = "wq賺"
		}
		if got := mixedType(s, "wq5j04").Text(); got != want {
			t.Fatalf("policy=%t: %q want %q", aggressive, got, want)
		}
	}
}

// 積極模式只把數字、標點鍵併進後段的音節（WQ5j04 → WQ賺）；字母鍵不併，英文字尾的聲母鍵不被吃掉：
// page 的 e 是 ㄍ，pageji3 是 page我，不是 pag果（真實字典的 iPhone我、mac我、python我 同理）。
func TestMixedAggressiveAbsorbsOnlyNonLetterKeys(t *testing.T) {
	for _, tc := range []struct{ keys, want string }{
		{"pageji3", "page我"}, {"lineji3", "line我"}, {"WQ5j04", "WQ賺"}, {"hello5j04", "hello賺"},
	} {
		s := englishBoundarySession()
		s.MixedInputAggressive = true
		lex, sys := s.model.Lex.(*lexicon.Mem), s.model.LM.(*lm.Mem)
		for _, e := range []struct {
			w, r string
			p    float64
		}{{"果", "ㄍㄨㄛˇ", 2e-3}, {"賺", "ㄓㄨㄢˋ", 1e-6}, {"萬", "ㄨㄢˋ", 1e-2}} {
			lex.Add(lexicon.Entry{Surface: e.w, Reading: zhuyin.ParseReading(e.r), PRW: 1})
			sys.SetUni(e.w, e.p)
		}
		if got := mixedType(s, tc.keys).Text(); got != tc.want {
			t.Errorf("%q: %q want %q", tc.keys, got, tc.want)
		}
	}
}
