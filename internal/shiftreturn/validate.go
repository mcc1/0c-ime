//go:build shiftreturneval

package shiftreturn

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/0c-biggo/0c-ime/core/layout"
	"github.com/0c-biggo/0c-ime/core/mixedparse"
)

func HashBytes(b []byte) string { h := sha256.Sum256(b); return hex.EncodeToString(h[:]) }

// Canonical encodes the integer/string/boolean identity objects used by v2.
// Model floating point canonicalization is owned exclusively by the Python writer.
func Canonical(v any) []byte {
	encoded, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	d := json.NewDecoder(bytes.NewReader(encoded))
	d.UseNumber()
	var ordered any
	if err := d.Decode(&ordered); err != nil {
		panic(err)
	}
	var b bytes.Buffer
	e := json.NewEncoder(&b)
	e.SetEscapeHTML(false)
	if err := e.Encode(ordered); err != nil {
		panic(err)
	}
	s := strings.TrimSuffix(b.String(), "\n")
	// Only unescape JSON Unicode separators at an unescaped backslash.
	// A literal backslash-u sequence must keep its escaped backslash intact.
	var unicode bytes.Buffer
	for i := 0; i < len(s); {
		if s[i] == '\\' && i+1 < len(s) && s[i+1] == '\\' {
			unicode.WriteString(s[i : i+2])
			i += 2
			continue
		}
		if strings.HasPrefix(s[i:], `\u2028`) {
			unicode.WriteRune('\u2028')
			i += 6
			continue
		}
		if strings.HasPrefix(s[i:], `\u2029`) {
			unicode.WriteRune('\u2029')
			i += 6
			continue
		}
		unicode.WriteByte(s[i])
		i++
	}
	return unicode.Bytes()
}
func Validate(f Fixture) error {
	if err := Decode(Canonical(f), new(Fixture)); err != nil {
		return err
	}
	if f.ID == "" || f.PrototypeID == "" || f.SourceRef == "" {
		return fmt.Errorf("empty fixture identity/source")
	}
	for _, es := range [][]Event{f.PreludeEvents, f.Events} {
		for _, e := range es {
			if err := checkEvent(e); err != nil {
				return err
			}
		}
	}
	var raw strings.Builder
	for _, e := range f.Events {
		if e.Type == "key" {
			raw.WriteString(e.Key)
		}
	}
	if raw.String() != f.ExpectedRaw {
		return fmt.Errorf("%s: event/raw mismatch", f.ID)
	}
	if f.Intent != "compat" {
		if len(f.Events) < 3 || f.Events[0].Type != "shift" || f.Events[len(f.Events)-1].Type != "enter" {
			return fmt.Errorf("%s: formal episode must be Shift/key+/Enter", f.ID)
		}
		for _, e := range f.Events[1 : len(f.Events)-1] {
			if e.Type != "key" {
				return fmt.Errorf("%s: non-key formal event", f.ID)
			}
		}
		for _, e := range f.PreludeEvents {
			if e.Type != "key" {
				return fmt.Errorf("%s: non-key prelude", f.ID)
			}
		}
	}
	for _, g := range f.Gold {
		at, n, phase := 0, 0, 0
		for _, s := range g.Segments {
			if s.Begin != at || s.End <= s.Begin || s.End > len(f.ExpectedRaw) {
				return fmt.Errorf("%s: span gap/overlap/range", f.ID)
			}
			at = s.End
			switch s.Kind {
			case "literal":
				if phase > 0 {
					return fmt.Errorf("literal after Chinese/pending")
				}
				if s.Reading != "" {
					return fmt.Errorf("literal reading")
				}
			case "chinese":
				if phase == 2 {
					return fmt.Errorf("Chinese after pending")
				}
				phase = 1
				n++
				keys := f.ExpectedRaw[s.Begin:s.End]
				tone, ok := layout.Standard.Symbol(rune(keys[len(keys)-1]))
				reading, spelled := mixedparse.SpellExact(keys[:len(keys)-1], tone)
				if !ok || !spelled || string(reading) != s.Reading {
					return fmt.Errorf("%s: raw/reading label mismatch at %d", f.ID, s.Begin)
				}
			case "pending":
				phase = 2
				if s.Reading != "" || f.Intent != "compat" {
					return fmt.Errorf("pending formal label")
				}
			}
		}
		if at != len(f.ExpectedRaw) {
			return fmt.Errorf("%s: incomplete gold ownership", f.ID)
		}
		if f.Intent == "return" && n == 0 {
			return fmt.Errorf("return without Chinese")
		}
		if f.Intent == "literal" && n != 0 {
			return fmt.Errorf("literal with Chinese")
		}
		if f.Intent == "literal" {
			for _, surface := range g.AcceptedSurfaces {
				if surface != f.ExpectedRaw {
					return fmt.Errorf("literal surface differs from raw")
				}
			}
		}
	}
	return nil
}
func checkEvent(e Event) error {
	b := Canonical(e)
	var v any
	if err := json.Unmarshal(b, &v); err != nil {
		return err
	}
	return checkSchema(exchangeSchema["$defs"].(map[string]any)["event"].(map[string]any), v, "event")
}
func ValidateFixtures(fs []Fixture) error {
	seen := map[string]bool{}
	for _, f := range fs {
		if seen[f.ID] {
			return fmt.Errorf("duplicate fixture %q", f.ID)
		}
		seen[f.ID] = true
		if e := Validate(f); e != nil {
			return e
		}
	}
	return nil
}
