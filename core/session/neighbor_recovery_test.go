package session

import (
	"slices"
	"testing"

	"github.com/0c-biggo/0c-ime/core/layout"
	"github.com/0c-biggo/0c-ime/core/lexicon"
	"github.com/0c-biggo/0c-ime/core/proto"
	"github.com/0c-biggo/0c-ime/core/zhuyin"
)

func neighborSession() *Session {
	s := englishBoundarySession()
	s.NeighborKeyRecovery, s.MixedInputAggressive = true, true
	return s
}

func TestNeighborUniqueRepairRetainsRawKeysAndDoesNotSelfTrain(t *testing.T) {
	s := neighborSession()
	if got := mixedType(s, "ki3").Text(); got != "我" {
		t.Fatalf("k next to j should recover ji3: %q", got)
	}
	if got := downItems(key(s, proto.KeyDown)); len(got) < 2 || got[1] != "ki3" {
		t.Fatalf("original keys must be on first page: %v", got)
	}
	key(s, proto.KeyEscape)
	if got := key(s, proto.KeyEnter).Commit; got != "我" || s.model.Store.Total != 0 {
		t.Fatalf("automatic repair learned: %q total=%g", got, s.model.Store.Total)
	}
	mixedType(s, "ki3")
	key(s, proto.KeyDown)
	s.Handle(proto.Request{Type: proto.Select, Index: 1})
	if got := key(s, proto.KeyEnter).Commit; got != "ki3" || s.model.Store.Total != 0 {
		t.Fatalf("literal restoration: %q total=%g", got, s.model.Store.Total)
	}
	mixedType(s, "ki3")
	key(s, proto.KeyDown)
	s.Handle(proto.Request{Type: proto.Select, Index: 0})
	key(s, proto.KeyEnter)
	if s.model.Store.Total == 0 {
		t.Fatal("explicit confirmation did not learn the corrected reading")
	}
}

func TestNeighborAmbiguityRequiresSelectionAndKeepsTone(t *testing.T) {
	s := neighborSession()
	lex := s.model.Lex.(*lexicon.Mem)
	for _, entry := range []struct{ w, r string }{{"躲", "ㄉㄨㄛˇ"}, {"妥", "ㄊㄨㄛˇ"}} {
		lex.Add(lexicon.Entry{Surface: entry.w, Reading: zhuyin.ParseReading(entry.r), PRW: 1})
	}
	if got := mixedType(s, "qji3").Text(); got != "qji3" {
		t.Fatalf("ambiguous input was guessed: %q", got)
	}
	items := downItems(key(s, proto.KeyDown))
	for _, word := range []string{"qji3", "躲", "妥"} {
		if !slices.Contains(items, word) {
			t.Fatalf("missing %q: %v", word, items)
		}
	}
	s.Handle(proto.Request{Type: proto.Select, Index: slices.Index(items, "妥")})
	if got := key(s, proto.KeyEnter).Commit; got != "妥" {
		t.Fatal(got)
	}
	mixedType(s, "qji3")
	if got := key(s, proto.KeyEnter).Commit; got != "qji3" {
		t.Fatalf("Enter accepted an unselected correction: %q", got)
	}
}

func TestNeighborFeatureGatesAndEnglishMode(t *testing.T) {
	for _, gate := range []string{"off", "conservative", "private", "paused", "toneless", "eten", "hsu", "english"} {
		t.Run(gate, func(t *testing.T) {
			s := neighborSession()
			switch gate {
			case "off":
				s.NeighborKeyRecovery = false
			case "conservative":
				s.MixedInputAggressive = false
			case "private":
				s.private = true
			case "paused":
				s.paused = true
			case "toneless":
				s.Toneless = true
			case "eten":
				s.layout = layout.ETen
			case "hsu":
				s.layout = layout.Hsu
			case "english":
				s.english = true
			}
			if cs := s.neighborChoices("ki", 'ˇ'); len(cs) != 0 {
				t.Fatalf("gate %s: %+v", gate, cs)
			}
		})
	}
}

func TestNeighborPrivateRequestsDoNotRepairOrLearn(t *testing.T) {
	s := neighborSession()
	for _, k := range "ki3" {
		s.Handle(proto.Request{Type: proto.Key, Key: string(k), Private: true})
	}
	if len(s.neighborKeys) != 0 {
		t.Fatal("private input was repaired")
	}
	s.Handle(proto.Request{Type: proto.Key, Key: proto.KeyDown, Private: true})
	if len(s.cand.neighbors) != 0 {
		t.Fatal("private input produced repair candidates")
	}
	s.Handle(proto.Request{Type: proto.Close, Private: true})
	out := s.Handle(proto.Request{Type: proto.Commit, Private: true})
	if out.Commit != "ki3" || s.model.Store.Total != 0 {
		t.Fatalf("private output/learning: %+v total=%g", out, s.model.Store.Total)
	}
	if got := mixedType(s, "ki3").Text(); got != "我" {
		t.Fatalf("non-private control did not exercise recovery: %q", got)
	}
}

func TestNeighborDoesNotBroadenValidReadingsAccountsOrNumbers(t *testing.T) {
	for _, raw := range []string{"593", "microsoft593fmp6", "kevin87", "lisa0304", "gpt4", "3.4", "10:30", "hello3", "su3cl3", "microsoft5993fmp6su3cl3"} {
		a, b := neighborSession(), neighborSession()
		b.NeighborKeyRecovery = false
		if got, want := mixedType(a, raw).Text(), mixedType(b, raw).Text(); got != want {
			t.Fatalf("existing input changed: %q: %q want %q", raw, got, want)
		}
	}
	for _, raw := range []string{"59", "k", "kkkk", "KI", "https://ki", "user_ki", "1111"} {
		if cs := neighborSession().neighborChoices(raw, 'ˇ'); len(cs) != 0 {
			t.Fatalf("overbroad correction: %q: %+v", raw, cs)
		}
	}
}

func TestNeighborKnownEnglishPrefixAndFollowingChinese(t *testing.T) {
	s := neighborSession()
	if got := mixedType(s, "helloki3cl3").Text(); got != "hello我好" {
		t.Fatalf("English prefix / following Chinese: %q", got)
	}
	// Inserting before an automatic repair must shift its provenance with it.
	s = neighborSession()
	mixedType(s, "ki3")
	s.Handle(proto.Request{Type: proto.Cursor, To: 0})
	mixedType(s, "su3")
	if !s.neighborTouches(1, 2) || s.neighborTouches(0, 1) {
		t.Fatalf("provenance did not shift: %v", s.neighborKeys)
	}
	s.Handle(proto.Request{Type: proto.Cursor, To: 2})
	key(s, proto.KeyBackspace)
	if len(s.neighborKeys) != 0 {
		t.Fatalf("deleted repair left provenance: %v", s.neighborKeys)
	}
}
