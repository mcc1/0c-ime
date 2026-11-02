//go:build shiftreturneval

package shiftreturn

import (
	"encoding/json"
	"fmt"
	"sort"

	"github.com/0c-biggo/0c-ime/core/english"
	"github.com/0c-biggo/0c-ime/core/lexicon"
	"github.com/0c-biggo/0c-ime/core/lm"
	"github.com/0c-biggo/0c-ime/core/personal"
	"github.com/0c-biggo/0c-ime/core/proto"
	"github.com/0c-biggo/0c-ime/core/session"
	"github.com/0c-biggo/0c-ime/core/vocab"
)

// ApplicationLedger mirrors the host: committed text precedes unhandled keys.
// It never infers ownership from the surface or drops first-tone ASCII space.
type ApplicationLedger struct{ App string }

func (l *ApplicationLedger) Apply(request proto.Request, snap proto.Snapshot) string {
	l.App += snap.Commit
	if request.Type == proto.Key && !snap.Handled && len(request.Key) == 1 && request.Key[0] >= 32 && request.Key[0] <= 126 {
		l.App += request.Key
	}
	return l.App + snap.Text()
}

type BaselineObservation struct {
	Request     proto.Request  `json:"request"`
	Snapshot    proto.Snapshot `json:"snapshot"`
	App         string         `json:"app"`
	View        string         `json:"view"`
	StoreBefore string         `json:"store_before"`
	StoreAfter  string         `json:"store_after"`
}
type Baseline struct {
	Session      *session.Session
	Model        *personal.Model
	Ledger       ApplicationLedger
	Observations []BaselineObservation
	Observe      bool
}

func NewBaseline(lex lexicon.Source, sys lm.System, eng *english.Dictionary, condition string, tokens []string, observe bool) (*Baseline, error) {
	if condition != "M" && condition != "R" && condition != "C" && condition != "A" {
		return nil, fmt.Errorf("baseline must be M/R/C/A")
	}
	m := personal.NewModel(lex, sys)
	m.Now = nil
	m.Store = personal.NewStore()
	if len(tokens) > 0 {
		m.Vocab = vocab.New()
		tokens = append([]string{}, tokens...)
		sort.Strings(tokens)
		entries := []vocab.Entry{}
		for _, token := range tokens {
			entries = append(entries, vocab.Entry{Text: token, Readings: []string{}})
		}
		if _, e := m.Vocab.Add(entries, vocab.SourceManual, 0); e != nil {
			return nil, e
		}
	}
	s := session.New(m)
	s.MixedInput = true
	s.MixedInputAggressive = condition == "A"
	s.Preview02 = true
	s.English = eng
	s.EnglishAutocomplete = false
	s.NeighborKeyRecovery = false
	s.Opt.DocumentContextWeight = 0
	return &Baseline{Session: s, Model: m, Observe: observe, Observations: []BaselineObservation{}}, nil
}
func StoreDigest(m *personal.Model) string {
	b, e := json.Marshal(m.Store)
	if e != nil {
		panic(e)
	}
	return HashBytes(b)
}
func (b *Baseline) Handle(r proto.Request) BaselineObservation {
	before := ""
	if b.Observe {
		before = StoreDigest(b.Model)
	}
	snap := b.Session.Handle(r)
	view := b.Ledger.Apply(r, snap)
	o := BaselineObservation{Request: r, Snapshot: snap, App: b.Ledger.App, View: view}
	if b.Observe {
		o.StoreBefore = before
		o.StoreAfter = StoreDigest(b.Model)
		b.Observations = append(b.Observations, o)
	}
	return o
}
func baselineRequest(e Event) (proto.Request, error) {
	switch e.Type {
	case "key":
		return proto.Request{Type: proto.Key, Key: e.Key}, nil
	case "shift":
		return proto.Request{Type: proto.Toggle}, nil
	case "enter":
		return proto.Request{Type: proto.Key, Key: proto.KeyEnter}, nil
	case "commit":
		return proto.Request{Type: proto.Commit}, nil
	}
	return proto.Request{}, fmt.Errorf("baseline formal event %q unsupported", e.Type)
}

// ReplayBaseline is the only gold-aware control adapter. R alone receives the
// known first Chinese boundary; C/A omit only the initial research Toggle.
func ReplayBaseline(b *Baseline, f Fixture, condition string) ([]BaselineObservation, error) {
	out := []BaselineObservation{}
	for _, e := range f.PreludeEvents {
		r, err := baselineRequest(e)
		if err != nil {
			return nil, err
		}
		out = append(out, b.Handle(r))
	}
	boundary := -1
	if condition == "R" && f.Intent == "return" {
		for _, s := range f.Gold[0].Segments {
			if s.Kind == "chinese" {
				boundary = s.Begin
				break
			}
		}
	}
	offset := 0
	inserted := false
	for i, e := range f.Events {
		if (condition == "C" || condition == "A") && i == 0 && e.Type == "shift" {
			continue
		}
		if e.Type == "key" && offset == boundary && !inserted {
			out = append(out, b.Handle(proto.Request{Type: proto.Toggle}))
			inserted = true
		}
		r, err := baselineRequest(e)
		if err != nil {
			return nil, err
		}
		out = append(out, b.Handle(r))
		if e.Type == "key" {
			offset += len(e.Key)
		}
	}
	return out, nil
}

// NewScoringProbe owns a different empty disposable store than every baseline.
func NewScoringProbe(lex lexicon.Source, sys lm.System, eng *english.Dictionary) (*Probe, *personal.Model) {
	m := personal.NewModel(lex, sys)
	m.Now = nil
	m.Store = personal.NewStore()
	return &Probe{Lex: m.Source(), Score: m, English: eng}, m
}
