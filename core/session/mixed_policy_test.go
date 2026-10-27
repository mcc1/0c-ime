package session

import (
	"github.com/0c-biggo/0c-ime/core/proto"
	"strings"
	"testing"
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
