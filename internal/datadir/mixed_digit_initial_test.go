package datadir

import (
	"github.com/0c-biggo/0c-ime/core/personal"
	"github.com/0c-biggo/0c-ime/core/proto"
	"os"
	"testing"
)

func TestMixedDigitInitialWithPublishedDictionary(t *testing.T) {
	dir := os.Getenv("ZC_TEST_PUBLIC_DICTIONARY")
	if dir == "" {
		t.Skip("set ZC_TEST_PUBLIC_DICTIONARY to a verified dictionary directory")
	}
	e, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	e.MixedInput, e.MixedInputAggressive = true, true
	for _, tc := range []struct{ keys, want string }{
		{"WQ5j04fu06", "WQ賺錢"}, {"wq5j04fu06", "wq賺錢"},
		{"ru,3WQ5j04fu06", "解WQ賺錢"}, {"hello5j04fu06", "hello賺錢"},
		{"WD55j04fu06", "WD5賺錢"},
	} {
		e.Model.Store = personal.NewStore()
		id := e.OpenSession()
		for _, k := range tc.keys {
			if _, err := e.Do(id, proto.Request{Type: proto.Key, Key: string(k)}); err != nil {
				t.Fatal(err)
			}
		}
		out, err := e.Do(id, proto.Request{Type: proto.Commit})
		if err != nil {
			t.Fatal(err)
		}
		e.CloseSession(id)
		if out.Commit != tc.want {
			t.Fatalf("%s: %q want %q", tc.keys, out.Commit, tc.want)
		}
		t.Logf("%s -> %s", tc.keys, out.Commit)
	}
}
