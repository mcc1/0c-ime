//go:build mixedeval

package session

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"sync"

	"github.com/0c-biggo/0c-ime/core/english"
	"github.com/0c-biggo/0c-ime/core/lexicon"
	"github.com/0c-biggo/0c-ime/core/lm"
	"github.com/0c-biggo/0c-ime/core/mixedparse"
	"github.com/0c-biggo/0c-ime/core/personal"
	"github.com/0c-biggo/0c-ime/core/proto"
	"github.com/0c-biggo/0c-ime/core/vocab"
	"github.com/0c-biggo/0c-ime/core/zhuyin"
)

var evalObservers sync.Map // *Session -> *MixedEvalBaseline; never owns public stores
func init() {
	mixedEvalObserveInsert = func(s *Session, syl zhuyin.Syllable, keys string) func() {
		v, ok := evalObservers.Load(s)
		if !ok {
			return func() {}
		}
		b := v.(*MixedEvalBaseline)
		pos := s.cursor
		old, had := s.prefixLiteralBefore()
		prior := ""
		if had {
			prior = b.retained[old.start]
		}
		return func() {
			for _, p := range s.pendingLiterals {
				if p.start == pos && p.prefix && p.syllable() == syl {
					b.retained[pos] = keys
				}
			}
			if had && s.cursor == old.start+2 && !s.literals[old.start] && prior != "" {
				b.joined[old.start] = prior
			}
			b.InsertObservations++
		}
	}
}

// MixedEvalBaseline is an isolated actual legacy replay, never used to score a
// hypothetical candidate. Close removes the observer registration.
type MixedEvalBaseline struct {
	s                  *Session
	raw                string
	retained, joined   map[int]string
	unsupported        bool
	customTokens       []string
	InsertObservations int
}
type MixedEvalObservation struct {
	Snapshot           proto.Snapshot       `json:"snapshot"`
	Raw                string               `json:"raw"`
	Visible            string               `json:"visible"`
	Pending            string               `json:"pending"`
	Partition          []mixedparse.Segment `json:"partition"`
	OwnershipStatus    string               `json:"ownership_status"`
	OwnershipReason    string               `json:"ownership_reason,omitempty"`
	Context            mixedparse.Context   `json:"context"`
	StoreDigest        string               `json:"store_digest"`
	StoreTotal         float64              `json:"store_total"`
	InsertObservations int                  `json:"insert_observations"`
}

func NewMixedEvalBaseline(lex lexicon.Source, sys lm.System, eng *english.Dictionary, aggressive bool, tokens []string) (*MixedEvalBaseline, error) {
	m := personal.NewModel(lex, sys)
	m.Now = nil
	m.Store = personal.NewStore()
	if len(tokens) > 0 {
		m.Vocab = vocab.New()
		tokens = append([]string(nil), tokens...)
		sort.Strings(tokens)
		entries := []vocab.Entry{}
		for _, t := range tokens {
			entries = append(entries, vocab.Entry{Text: t, Readings: []string{}})
		}
		if _, err := m.Vocab.Add(entries, vocab.SourceManual, 0); err != nil {
			return nil, err
		}
	}
	s := New(m)
	s.Opt.DocumentContextWeight = 0
	s.MixedInput = true
	s.MixedInputAggressive = aggressive
	s.Preview02 = true
	s.English = eng
	s.EnglishAutocomplete = false
	b := &MixedEvalBaseline{s: s, retained: map[int]string{}, joined: map[int]string{}, customTokens: append([]string(nil), tokens...)}
	evalObservers.Store(s, b)
	return b, nil
}
func (b *MixedEvalBaseline) Close() { evalObservers.Delete(b.s) }
func (b *MixedEvalBaseline) Handle(r proto.Request) MixedEvalObservation {
	if r.Type == proto.Key && len(r.Key) == 1 && r.Key[0] >= 33 && r.Key[0] <= 126 {
		b.raw += r.Key
	}
	if r.Type == proto.Cursor || r.Type == proto.Backspace || r.Type == proto.Key && (r.Key == proto.KeyDelete || r.Key == " " || len(r.Key) == 1 && strings.ContainsRune("!@#$%^&*()_+{}[]:<>?\"|\\", rune(r.Key[0]))) {
		b.unsupported = true
	}
	previous := b.observe(b.s.View())
	snap := b.s.Handle(r)
	o := b.observe(snap)
	if snap.Commit != "" && previous.OwnershipStatus == "mapped" && (r.Type == proto.Commit || r.Type == proto.Key && r.Key == proto.KeyEnter) {
		o.Partition = previous.Partition
		o.Context = previous.Context
		o.Pending = ""
		o.OwnershipStatus = "mapped"
		o.OwnershipReason = ""
	}
	if r.Type == proto.Cancel || r.Type == proto.Commit || snap.Commit != "" {
		b.raw = ""
		b.retained = map[int]string{}
		b.joined = map[int]string{}
		b.unsupported = false
	}
	return o
}
func (b *MixedEvalBaseline) Observe() MixedEvalObservation { return b.observe(b.s.View()) }
func (b *MixedEvalBaseline) observe(snap proto.Snapshot) MixedEvalObservation {
	s := b.s
	o := MixedEvalObservation{Snapshot: snap, Raw: b.raw, Visible: snap.Text(), Pending: s.mixed.raw, OwnershipStatus: "mapped", InsertObservations: b.InsertObservations}
	j, _ := json.Marshal(s.model.Store)
	h := sha256.Sum256(j)
	o.StoreDigest = hex.EncodeToString(h[:])
	o.StoreTotal = s.model.Store.Total
	c := mixedparse.Context{Layout: s.layout.Name(), Private: s.private, Paused: s.paused, LocalBlocked: s.localBlocked, English: s.english, Toneless: s.toneless(), CustomTokens: append([]string(nil), b.customTokens...)}
	at := 0
	fail := func(reason string) { o.OwnershipStatus = "unsupported"; o.OwnershipReason = reason; o.Partition = nil }
	consume := func(keys string) bool {
		if keys == "" || at+len(keys) > len(b.raw) || b.raw[at:at+len(keys)] != keys {
			return false
		}
		at += len(keys)
		return true
	}
	for pos := 0; pos < len(s.syls); {
		if p, ok := s.syllableLiteralAt(pos); ok && p.start == pos && p.prefix {
			keys := b.retained[pos]
			start := at
			if !consume(keys) {
				fail("pending_literal_provenance")
				break
			}
			o.Partition = append(o.Partition, mixedparse.Segment{Kind: "chinese", Begin: start, End: at, Reading: string(p.syllable())})
			for k := pos; k < p.end()+p.mark(); k++ {
				c.Units = append(c.Units, mixedparse.Unit{Text: string(s.syls[k]), Literal: true})
			}
			if p.end()+p.mark() == s.cursor {
				c.RetainedPrefix = &mixedparse.Retained{Reading: string(p.syllable()), OriginalKeys: keys}
			}
			pos = p.end() + p.mark()
			continue
		}
		start := at
		u := mixedparse.Unit{Literal: s.literals[pos]}
		if u.Literal {
			u.Text = string(s.syls[pos])
			u.OriginalKeys = u.Text
			if x := s.literalKeys[pos]; x != "" {
				u.OriginalKeys = x
			}
		} else {
			u.Reading = string(s.syls[pos])
			u.OriginalKeys = s.mixedKeys[pos]
			if u.OriginalKeys == "" {
				u.OriginalKeys = b.joined[pos]
			}
			for _, p := range s.path {
				if p.Start <= pos && pos < p.End {
					u.Text = p.Word
					break
				}
			}
		}
		if !consume(u.OriginalKeys) {
			fail("legacy_rewrite_or_lost_keys")
			break
		}
		seg := mixedparse.Segment{Kind: "literal", Begin: start, End: at, Text: u.OriginalKeys}
		if !u.Literal {
			seg.Kind = "chinese"
			seg.Text = ""
			seg.Reading = u.Reading
		}
		o.Partition = append(o.Partition, seg)
		c.Units = append(c.Units, u)
		pos++
	}
	if o.OwnershipStatus == "mapped" && s.mixed.raw != "" {
		start := at
		if !consume(s.mixed.raw) {
			fail("assisted_pending_rewrite")
		} else {
			o.Partition = append(o.Partition, mixedparse.Segment{Kind: "literal", Begin: start, End: at, Text: s.mixed.raw})
		}
	}
	if b.unsupported {
		fail("editing_or_literal_punctuation")
		o.OwnershipStatus = "unsupported"
	}
	if snap.Commit != "" || len(s.syls) == 0 && s.mixed.raw == "" && b.raw != "" {
		fail("closed_snapshot_use_previous_ledger")
		o.OwnershipStatus = "unsupported"
	}
	if o.OwnershipStatus == "mapped" && at != len(b.raw) {
		fail("legacy_rewrite_or_lost_keys")
	}
	if len(s.path) > 0 {
		p := s.path[len(s.path)-1]
		if p.End == s.cursor && !p.Partial {
			literal := false
			for i := p.Start; i < p.End; i++ {
				literal = literal || s.literals[i]
			}
			if !literal {
				c.LastWord = p.Word
			}
		}
	}
	if c.RetainedPrefix != nil {
		c.LastWord = ""
	}
	o.Context = c.Clone()
	o.Partition = mergeEvalLiterals(o.Partition)
	return o
}
func mergeEvalLiterals(p []mixedparse.Segment) []mixedparse.Segment {
	out := []mixedparse.Segment{}
	for _, s := range p {
		if len(out) > 0 && s.Kind == "literal" && out[len(out)-1].Kind == "literal" {
			out[len(out)-1].End = s.End
			out[len(out)-1].Text += s.Text
		} else {
			out = append(out, s)
		}
	}
	return out
}

// ResolveSelection matches a typed canonical menu reference, never a UI rank.
func (b *MixedEvalBaseline) ResolveSelection(kind string, span [2]int, readings []string, text string) (int, error) {
	s := b.s
	if s.cand == nil {
		return 0, fmt.Errorf("no menu")
	}
	hits := []int{}
	for i, x := range s.cand.items {
		start, n := s.cand.span(i)
		if start < 0 || start+n > len(s.syls) {
			continue
		}
		r := []string{}
		keys := ""
		begin, end := -1, -1
		o := b.Observe()
		for _, z := range o.Partition {
			if z.Kind == "chinese" {
				r = append(r, z.Reading)
			}
		}
		_ = r
		at := 0
		for j := 0; j < start; j++ {
			k := s.mixedKeys[j]
			if s.literals[j] {
				k = string(s.syls[j])
			}
			at += len(k)
		}
		begin = at
		for j := start; j < start+n; j++ {
			keys += s.mixedKeys[j]
		}
		end = begin + len(keys)
		if begin != span[0] || end != span[1] {
			continue
		}
		if kind == "original" && s.cand.keyLiterals[i] == text && text == keys {
			hits = append(hits, i)
		}
		if kind == "word" && x.Entry.Surface == text {
			rr := []string{}
			for _, v := range x.Entry.Reading {
				rr = append(rr, string(v))
			}
			a, _ := json.Marshal(rr)
			z, _ := json.Marshal(readings)
			if string(a) == string(z) {
				hits = append(hits, i)
			}
		}
	}
	if len(hits) != 1 {
		return 0, fmt.Errorf("selection matches=%d", len(hits))
	}
	return hits[0], nil
}
