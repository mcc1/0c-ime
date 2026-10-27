package session

import (
	"strings"
	"testing"

	"github.com/0c-biggo/0c-ime/core/english"
	"github.com/0c-biggo/0c-ime/core/lexicon"
	"github.com/0c-biggo/0c-ime/core/lm"
	"github.com/0c-biggo/0c-ime/core/personal"
	"github.com/0c-biggo/0c-ime/core/proto"
	"github.com/0c-biggo/0c-ime/core/zhuyin"
)

// englishBoundarySession 照真實資料包的量級收字：ㄋㄧˇ（你）與 ㄧˇ（以）單字機率相當，
// ㄋㄧㄡˇ（紐）、ㄋㄧㄠˋ（尿）、ㄙㄚ˙（挱）遠低於 ㄧㄡˇ（有）、ㄧㄠˋ（要）。英文字典有 hello／hellos 這類
// 「英文字＋s」也是單字的組合；ok、gmail、iphone 不在字典裡。
func englishBoundarySession() *Session {
	lex, sys := lexicon.NewMem(), lm.NewMem()
	for _, e := range []struct {
		w, r string
		p    float64
	}{
		{"你", "ㄋㄧˇ", 1e-3}, {"以", "ㄧˇ", 1e-3}, {"好", "ㄏㄠˇ", 2.4e-4}, {"你好", "ㄋㄧˇ ㄏㄠˇ", 1e-4},
		{"我", "ㄨㄛˇ", 2e-3}, {"有", "ㄧㄡˇ", 1.5e-3}, {"紐", "ㄋㄧㄡˇ", 4e-6}, {"要", "ㄧㄠˋ", 9e-4}, {"尿", "ㄋㄧㄠˋ", 4e-6},
		{"的", "ㄉㄜ˙", 1.6e-2}, {"挱", "ㄙㄚ˙", 3e-8}, {"狗", "ㄍㄡˇ", 3.5e-5}, {"斥", "ㄔˋ", 2e-5}, {"是", "ㄕˋ", 5e-3},
		{"欸", "ㄟˋ", 1e-4}, {"與", "ㄩˇ", 5e-4}, {"艮", "ㄍㄣˇ", 1e-6}, {"社", "ㄕㄜˋ", 2e-4},
		{"會", "ㄏㄨㄟˋ", 1.1e-3}, {"為", "ㄨㄟˋ", 1.2e-3}, {"柳", "ㄌㄧㄡˇ", 2e-5},
		{"嗎", "ㄇㄚ˙", 3e-4}, {"滿", "ㄇㄢˇ", 2e-4}, {"們", "ㄇㄣ˙", 5e-5},
		{"應", "ㄧㄥˋ", 1.7e-4}, {"競", "ㄐㄧㄥˋ", 1.2e-4}, {"呃", "ㄜ˙", 2e-6},
		{"爾", "ㄦˇ", 1e-4}, {"二", "ㄦˋ", 3e-4}, {"偶", "ㄡˇ", 5e-5}, {"天", "ㄊㄧㄢ", 1e-3}, {"換", "ㄏㄨㄢˋ", 2e-4},
		{"一", "ㄧ", 2e-3}, {"誒", "ㄟ", 1e-5}, {"資", "ㄗ", 3e-4},
		{"窄", "ㄓㄞˇ", 2e-5}, {"群", "ㄑㄩㄣˊ", 2e-5},
	} {
		lex.Add(lexicon.Entry{Surface: e.w, Reading: zhuyin.ParseReading(e.r), PRW: 1})
		sys.SetUni(e.w, e.p)
	}
	s := New(personal.NewModel(lex, sys))
	s.MixedInput = true
	var words []english.Word
	for _, w := range []string{"hello", "hellos", "test", "tests", "app", "apps", "line", "zoom", "world", "step", "page", "user", "name", "win", "kevin", "is", "hell", "zoo", "st", "use", "api"} {
		words = append(words, english.Word{Text: w, Weight: 65})
	}
	s.English = english.New(words)
	return s
}

// #119：英文後面不打空白直接接注音。「英文字＋s」也是英文字時，s（ㄋ）不能被英文吃掉；
// 不在字典裡的英文（ok、gmail、iphone）後面接明確的聲調也要切開。
func TestMixedEnglishThenZhuyinWithoutSpace(t *testing.T) {
	for _, tc := range []struct{ keys, want string }{
		{"hellosu3cl3", "hello你好"}, // 不是 hellos以好
		{"testsu3", "test你"},       // 不是 tests以
		{"okji3", "ok我"},
		{"gmailsu3cl3", "gmail你好"},
		{"iphoneji3", "iphone我"},
		{"iPhoneji3", "iPhone我"},
		{"OKji3", "OK我"},
		{"ok2k7", "ok的"},
		{"okcjo4", "ok會"},          // 不是 okc為：會、為差不多常用，取較長的音節
		{"netflixu.3", "netflix有"}, // 不是 netfli柳
		// 原本就對的不變
		{"helloji3", "hello我"}, {"LINEji3", "LINE我"}, {"zoomji3", "zoom我"}, {"hello su3cl3", "hello 你好"},
		{"hellosu3", "hello你"},
		// 短的音節常用得多時，英文照舊保留結尾的字母：tests有、apps要，不是 test紐、app尿
		{"testsu.3", "tests有"}, {"appsul4", "apps要"},
		// 其他字母要中文明顯常用得多才讓：user應 不是 use競（r 是 ㄐ）
		{"useru/4", "user應"},
		// 英文字＋數字鍵：2 是 ㄉ，的 遠比 呃 常用（原本是 api2呃）
		{"api2k7", "api的"},
	} {
		s := englishBoundarySession()
		mixedType(s, tc.keys)
		if got := key(s, proto.KeyEnter).Commit; got != tc.want {
			t.Errorf("%q：%q，要 %q", tc.keys, got, tc.want)
		}
	}
}

// 未完成中文音節的英文、網址、識別字與型號照舊原樣；未知英文接空白也不猜切點。
func TestMixedUnknownEnglishStaysLiteral(t *testing.T) {
	for _, keys := range []string{
		"hello world ", "user_name ", "someNewName ", "okji ", "gmailsu ",
		"hello4", "zoom3", "step3", "page3", // 完整英文字接數字：不是 hell欸、zoo與、st艮
		"test4", "win7", "mp3", "ps4", "h264", "abc3", "user4",
		"example.com/ji3", "https://example.com/su3",
	} {
		s := englishBoundarySession()
		mixedType(s, keys)
		if got := key(s, proto.KeyEnter).Commit; got != keys {
			t.Errorf("%q 不能切成中文：%q", keys, got)
		}
	}
}

func TestMixedCompletedZhuyinAfterEnglishPrefersChinese(t *testing.T) {
	for _, tc := range []struct{ keys, want string }{
		{"microsoft593", "microsoft窄"},
		{"microsoft593fmp6", "microsoft窄群"},
		{"microsoftsu3cl3", "microsoft你好"},
		// A valid reading is Chinese evidence even if its frequency is low.
		{"kevin87", "kevi挱"}, {"nina87", "nin嗎"}, {"lisa0304", "lis滿04"},
		{"facebookg4", "facebook是"}, {"gpt4", "gp斥"}, {"gpt4o ", "gp斥誒"},
	} {
		t.Run(tc.keys, func(t *testing.T) {
			s := englishBoundarySession()
			// Unrelated learning must not block an unknown-English boundary either.
			for i := 0; i < 100; i++ {
				mixedType(s, "ji3")
				key(s, proto.KeyEnter)
			}
			if got := mixedType(s, tc.keys).Text(); got != tc.want {
				t.Fatalf("preedit: %q, want %q", got, tc.want)
			}
			if got := key(s, proto.KeyEnter).Commit; got != tc.want {
				t.Fatalf("commit: %q, want %q", got, tc.want)
			}
		})
	}
}

func TestMixedTypoWithDigitsDoesNotSwallowFollowingZhuyin(t *testing.T) {
	for _, tc := range []struct{ keys, want string }{
		{"microsoft5993fmp6su3cl3", "microsoft5993群你好"},
		{"microsoft599fmp6su3cl3", "microsoft599群你好"},
		{"qji3593fmp6", "qji3窄群"},
		{"qji3fmp6su3cl3", "qji3群你好"},
	} {
		t.Run(tc.keys, func(t *testing.T) {
			s := englishBoundarySession()
			if got := mixedType(s, tc.keys).Text(); got != tc.want {
				t.Fatalf("preedit: %q, want %q", got, tc.want)
			}
			if got := key(s, proto.KeyEnter).Commit; got != tc.want {
				t.Fatalf("commit: %q, want %q", got, tc.want)
			}
		})
	}
}

func TestMixedChinesePriorityDoesNotApplyInEnglishMode(t *testing.T) {
	for _, raw := range []string{"593", "1/3", "86", "kevin87", "lisa0304", "gpt4", "microsoft593fmp6"} {
		t.Run(raw, func(t *testing.T) {
			s := englishBoundarySession()
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
				t.Fatalf("English mode: %q, want %q", got, raw)
			}
		})
	}
}

// 打錯一鍵的注音（qji3 是 ㄆㄨㄛˇ、ju3 是 ㄨㄧˇ，組不成音節）照字面留下，後面的注音照常組成中文；
// 不再一路併成同一串，直到空白才整串變成英文字母（qji3su3cl3 不是整串照字面）。
func TestMixedTypoDoesNotSwallowFollowingZhuyin(t *testing.T) {
	for _, tc := range []struct{ keys, want string }{
		{"qji3su3cl3", "qji3你好"},
		{"ju3su3cl3", "ju3你好"},
		{"qji3su3", "qji3你"},
		// 句首機率低、但不是罕用字的音節也切開（個人用字可能讓們、門的句首機率掉到 5e-5）
		{"qji3ap7su3", "qji3們你"},
		// 整串已知英文接數字仍照字面，沒有可獨立辨識的新音節。
		{"hello4", "hello4"},
		// 只有數字與標點的前段是數字寫法，不切（-3 爾、.3 偶）
		{"1-4-3", "1-4-3"}, {"2025.3.31", "2025.3.31"}, {"2.7-3.1", "2.7-3.1"}, {"1-2-3-4-5-6", "1-2-3-4-5-6"},
		// 打錯那段後面的音節以數字鍵結尾（ㄊㄧㄢ 的 0、ㄏㄨㄢ 的 0）也切開；後段要有字母
		{"qji3wu0 ", "qji3天"}, {"qji3cj04", "qji3換"}, {"qji3cj04su3cl3", "qji3換你好"},
		// 型號、版本後面的單一字母接空白照字面（0.2.4 的行為）；單一聲母照樣切（ㄗ 資）
		{"qji3y ", "qji3資"},
		{"ps4u ", "ps4u "}, {"mp3u ", "mp3u "}, {"win7u ", "win7u "}, {"a4u ", "a4u "},
	} {
		s := englishBoundarySession()
		mixedType(s, tc.keys)
		if got := key(s, proto.KeyEnter).Commit; got != tc.want {
			t.Errorf("%q：%q，要 %q", tc.keys, got, tc.want)
		}
	}
}
