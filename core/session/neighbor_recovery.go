package session

import (
	"cmp"
	"slices"
	"strings"
	"unicode"

	"github.com/0c-biggo/0c-ime/core/decode"
	"github.com/0c-biggo/0c-ime/core/input"
	"github.com/0c-biggo/0c-ime/core/layout"
	"github.com/0c-biggo/0c-ime/core/zhuyin"
)

type neighborChoice struct {
	at   int // unchanged English prefix, in ASCII bytes
	syl  zhuyin.Syllable
	keys string // corrected physical keys, excluding the unchanged tone
}

type neighborSelection struct {
	choice neighborChoice
	word   string
}

// First release: opt-in, explicit tones, standard-layout aggressive mixed input.
// This is a fallback after the existing whole-reading rules fail, not a
// spell checker for already valid Chinese. It never searches an entire token.
func (s *Session) neighborChoices(raw string, tone rune) []neighborChoice {
	if !s.NeighborKeyRecovery || !s.MixedInputAggressive || !s.mixedInput() ||
		s.english || s.private || s.paused || s.localBlocked || s.layout.Name() != layout.NameStandard ||
		tone == zhuyin.Tone1 || s.mixed.choice != 0 || s.mixed.accepted != "" ||
		s.literalNumberBefore() != "" || s.mixedEnglishWord(raw) || s.vocabEnglishPrefix(raw) {
		return nil
	}
	var out []neighborChoice
	// At most two possible suffix lengths; a longer prefix must be an existing
	// English word. Unknown account/identifier prefixes are not repair anchors.
	for _, n := range []int{3, 2} {
		at := len(raw) - n
		if at < 0 || at > 0 && !s.mixedEnglishWord(raw[:at]) {
			continue
		}
		body := raw[at:]
		if !strings.ContainsFunc(body, unicode.IsLetter) || strings.ContainsFunc(body, func(k rune) bool {
			sym, ok := s.layout.Symbol(k)
			return k > unicode.MaxASCII || !ok || zhuyin.ClassOf(sym) == zhuyin.Tone
		}) {
			continue
		}
		if _, ok := s.mixedReading(body, tone); ok {
			continue // a valid original reading is never rewritten
		}
		for i, k := range body {
			for _, replacement := range layout.AdjacentKeys(k) {
				sym, ok := s.layout.Symbol(replacement)
				if !ok || zhuyin.ClassOf(sym) == zhuyin.Tone {
					continue
				}
				keys := body[:i] + string(replacement) + body[i+1:]
				syl, ok := s.mixedExactReading(keys, tone)
				if !ok || len(s.model.Source().Exact([]zhuyin.Syllable{syl})) == 0 {
					continue // incomplete dictionary prefixes cannot auto-repair
				}
				if !slices.ContainsFunc(out, func(c neighborChoice) bool { return c.at == at && c.syl == syl }) {
					out = append(out, neighborChoice{at, syl, keys})
				}
			}
		}
	}
	return out
}

func (s *Session) acceptNeighbor(c neighborChoice, raw, word string) {
	s.clearMixed()
	if c.at > 0 {
		s.insertMixedLiteral(raw[:c.at], false, false)
	}
	at := s.cursor
	s.insertMixedReading(c.syl, raw[c.at:]) // original physical keys, not the proposal
	if word != "" {
		s.locks[at] = decode.Lock{Word: word, Len: 1} // explicit selection may learn
	} else {
		if s.neighborKeys == nil {
			s.neighborKeys = map[int]string{}
		}
		s.neighborKeys[at] = raw[c.at:] // unconfirmed repairs must not self-train
	}
	s.refresh()
}

func (s *Session) neighborTouches(start, end int) bool {
	for at := range s.neighborKeys {
		if at >= start && at < end {
			return true
		}
	}
	return false
}

// Ambiguous readings are offered only on Down. Enter keeps the displayed raw
// input. Bounded enumeration and the existing statistical candidate scorer are
// used; the neural model is not asked to compare different phonetic readings.
func (s *Session) addNeighborCandidates(c *candState, seen map[string]bool) {
	raw := s.mixed.raw
	if len(raw) < 3 {
		return
	}
	tone, ok := s.layout.Symbol(rune(raw[len(raw)-1]))
	if !ok || zhuyin.ClassOf(tone) != zhuyin.Tone {
		return
	}
	choices := s.neighborChoices(raw[:len(raw)-1], tone)
	dec := &decode.Decoder{Lex: s.model.Source(), Score: s.model}
	type proposal struct {
		choice neighborChoice
		item   decode.Candidate
	}
	var proposals []proposal
	for _, choice := range choices {
		left := dec.InitialLeft()
		if choice.at == 0 {
			for _, g := range s.path {
				if g.End == s.cursor && !g.Partial && !s.literals[g.Start] {
					left = g.Word
				}
			}
		}
		items := dec.Candidates(input.Exact{}.Expand([]zhuyin.Syllable{choice.syl}), 0, 1, left, nil)
		for _, item := range items[:min(3, len(items))] {
			proposals = append(proposals, proposal{choice, item})
		}
	}
	slices.SortStableFunc(proposals, func(a, b proposal) int { return cmp.Compare(b.item.Score, a.item.Score) })
	for _, p := range proposals {
		text := raw[:p.choice.at] + p.item.Entry.Surface
		if seen[text] || len(c.neighbors) >= 8 {
			continue
		}
		seen[text] = true
		if c.neighbors == nil {
			c.neighbors = map[int]neighborSelection{}
		}
		c.neighbors[len(c.items)] = neighborSelection{p.choice, p.item.Entry.Surface}
		c.items = append(c.items, rawCandidate(text))
	}
}
