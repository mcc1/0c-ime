package datadir

import (
	"os"
	"testing"

	"github.com/0c-biggo/0c-ime/core/personal"
	"github.com/0c-biggo/0c-ime/core/proto"
)

// Opt-in integration check against a maintainer's verified public dictionary;
// no download, no user store, and no desktop installation are involved.
func TestNeighborRecoveryWithPublishedDictionary(t *testing.T) {
	dir := os.Getenv("ZC_TEST_PUBLIC_DICTIONARY")
	if dir == "" {
		t.Skip("set ZC_TEST_PUBLIC_DICTIONARY to a verified dictionary directory")
	}
	e, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	e.MixedInput, e.MixedInputAggressive = true, true
	for _, keys := range []string{"ki3", "593", "microsoft593fmp6", "kevin87", "gpt4", "3.4", "10:30", "helloki3cl3", "qji3cl3"} {
		var results []string
		for _, enabled := range []bool{false, true} {
			e.NeighborKeyRecovery = enabled
			e.Model.Store = personal.NewStore()
			id := e.OpenSession()
			for _, k := range keys {
				if _, err := e.Do(id, proto.Request{Type: proto.Key, Key: string(k)}); err != nil {
					t.Fatal(err)
				}
			}
			out, err := e.Do(id, proto.Request{Type: proto.Commit})
			if err != nil {
				t.Fatal(err)
			}
			e.CloseSession(id)
			results = append(results, out.Commit)
		}
		t.Logf("%s: off=%q on=%q", keys, results[0], results[1])
		if want := map[string]string{"ki3": "我", "helloki3cl3": "hello我好", "qji3cl3": "qji3好"}[keys]; want != "" {
			if results[1] != want || results[0] == results[1] {
				t.Fatalf("public dictionary did not recover unique neighbor: %v", results)
			}
		} else if results[0] != results[1] {
			t.Fatalf("existing valid input changed: %s: %v", keys, results)
		}
	}
}
