// Package mixedparse implements the read-only mixed-eval-v1 experiment.
// It never accepts a Session, mutates a model, or persists personal data.
package mixedparse

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"github.com/0c-biggo/0c-ime/core/decode"
	"github.com/0c-biggo/0c-ime/core/lexicon"
)

const Protocol = "mixed-eval-v1"
const FeatureVersion = "mixed-features-v1"

type Context struct {
	Generation                                       string
	Units                                            []Unit
	RetainedPrefix                                   *Retained
	LastWord                                         string
	Private, Paused, LocalBlocked, English, Toneless bool
	Layout                                           string
	CustomTokens                                     []string
}
type Unit struct {
	Text, Reading string
	Literal       bool
	OriginalKeys  string
}
type Retained struct{ Reading, OriginalKeys string }

func (c Context) Clone() Context {
	c.Units = append([]Unit(nil), c.Units...)
	c.CustomTokens = append([]string(nil), c.CustomTokens...)
	if c.RetainedPrefix != nil {
		r := *c.RetainedPrefix
		c.RetainedPrefix = &r
	}
	return c
}
func (c Context) Protected() bool {
	return c.Private || c.Paused || c.LocalBlocked || c.English || c.Toneless || (c.Layout != "" && c.Layout != "standard")
}

type Segment struct {
	Kind    string `json:"kind"`
	Begin   int    `json:"begin"`
	End     int    `json:"end"`
	Reading string `json:"reading,omitempty"`
	Text    string `json:"text,omitempty"`
}
type Candidate struct {
	ID             string       `json:"candidate_id"`
	Lane           string       `json:"lane"`
	Segments       []Segment    `json:"segments"`
	Readings       []string     `json:"readings"`
	Support        Support      `json:"support"`
	RawScore       *float64     `json:"raw_score"`
	ScoreStatus    string       `json:"score_status"`
	Features       [26]*float64 `json:"features"`
	FeatureVersion string       `json:"feature_version"`
	History        string       `json:"history"`
	Pruned         bool         `json:"pruned"`
	Text           string       `json:"text"`
	LastWord       string       `json:"last_word"`
	Identity       string       `json:"identity"`
}
type Support struct {
	WholeWord     bool       `json:"whole_word"`
	CompleteCover bool       `json:"complete_cover"`
	PrefixOnly    bool       `json:"prefix_only"`
	ContextOnly   bool       `json:"context_only"`
	Witnesses     [][]string `json:"witnesses"`
}
type English interface{ Has(string) bool }
type Evaluator struct {
	Lex             lexicon.Source
	Score           decode.Scorer
	System          decode.Scorer
	English         English
	PersonalEnglish func(string) bool
}
type Result struct {
	Candidates    []Candidate `json:"candidates"`
	GenerateCalls int         `json:"generate_calls"`
	ScoreCalls    int         `json:"score_calls"`
}

func Identity(s []Segment) string {
	a := make([][]any, 0, len(s))
	for _, x := range s {
		a = append(a, []any{x.Kind, x.Begin, x.End, x.Reading})
	}
	b, _ := json.Marshal(a)
	return string(b)
}
func CandidateID(s []Segment) string {
	h := sha256.Sum256([]byte(Identity(s)))
	return "c:" + hex.EncodeToString(h[:])
}
func Project(c Context, p string) Context {
	c = c.Clone()
	if p != "" {
		c.RetainedPrefix = nil
		c.LastWord = ""
	}
	for _, r := range p {
		c.Units = append(c.Units, Unit{Text: string(r), Literal: true, OriginalKeys: string(r)})
	}
	return c
}
