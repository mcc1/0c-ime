//go:build shiftreturneval

package shiftreturn

import (
	"math"
	"strconv"
	"strings"

	"github.com/0c-biggo/0c-ime/core/mixedparse"
	"github.com/0c-biggo/0c-ime/core/zhuyin"
)

func (p *Probe) compute(q Query, h Hypothesis) (v Vector, k int, text string, err error) {
	if p.bSnapshots == nil {
		p.bSnapshots = map[string]Vector{}
	}
	key := strconv.Itoa(h.Cut) + ":" + q.Raw[:h.Cut] + ":" + h.FirstReading
	if b, ok := p.bSnapshots[key]; ok {
		v = b
	} else {
		prefix := q.Raw[:h.Cut]
		if p.English != nil && prefix != "" && (len(prefix) > 1 || strings.EqualFold(prefix, "a") || strings.EqualFold(prefix, "i")) && p.English.Has(prefix) {
			v[1] = 1
		}
		for _, c := range []byte(prefix) {
			if c >= '0' && c <= '9' {
				v[2] = 1
			}
			if !(c >= '0' && c <= '9' || c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z') {
				v[3] = 1
			}
		}
		if mixedparse.NumberLike(prefix) {
			v[4] = 1
		}
		_, score, e := p.best([]zhuyin.Syllable{zhuyin.Syllable(h.FirstReading)}, true)
		if e != nil {
			err = e
			return
		}
		if math.IsInf(score, -1) {
			v[6] = 1
		} else {
			v[5] = score
		}
		p.bSnapshots[key] = v
	}
	n := len(h.Readings)
	v[7] = float64(n - 1)
	v[8] = float64(len(q.Raw) - h.ParsedEnd)
	readings := make([]zhuyin.Syllable, n)
	for i, s := range h.Readings {
		readings[i] = zhuyin.Syllable(s)
	}
	exact := make([][]bool, n)
	for start := 0; start < n; start++ {
		exact[start] = make([]bool, min(8, n-start)+1)
		for length := 1; length < len(exact[start]); length++ {
			entries, e := p.exact(readings[start : start+length])
			if e != nil {
				err = e
				return
			}
			exact[start][length] = len(entries) > 0
			if start == 0 && len(entries) > 0 {
				v[9] = float64(length)
			}
		}
	}
	reachable := make([]bool, n+1)
	reachable[0] = true
	coverage := make([]int, n+1)
	for start := 0; start < n; start++ {
		coverage[start+1] = max(coverage[start+1], coverage[start])
		for length := 1; length < len(exact[start]); length++ {
			if !exact[start][length] {
				continue
			}
			if reachable[start] {
				reachable[start+length] = true
				k = max(k, start+length)
			}
			if length >= 2 {
				coverage[start+length] = max(coverage[start+length], coverage[start]+length)
			}
		}
	}
	v[10] = float64(coverage[n]) / float64(n)
	if k == 0 {
		v[12] = 1
		return
	}
	var score float64
	text, score, err = p.best(readings[:k], false)
	if err != nil {
		return
	}
	if math.IsInf(score, -1) {
		v[12] = 1
	} else {
		v[11] = score / float64(k)
	}
	return
}

func Features(q Query, h Hypothesis) Vector { var v Vector; copy(v[:], h.Features); return v }
