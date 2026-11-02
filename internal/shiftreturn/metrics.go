//go:build shiftreturneval

package shiftreturn

import (
	"fmt"
	"math"
	"sort"
)

func Wilson(errors, n int) []float64 {
	if n == 0 {
		return nil
	}
	z := 1.959963984540054
	p := float64(errors) / float64(n)
	den := 1 + z*z/float64(n)
	center := (p + z*z/(2*float64(n))) / den
	half := z * math.Sqrt(p*(1-p)/float64(n)+z*z/(4*float64(n)*float64(n))) / den
	return []float64{max(0, center-half), min(1, center+half)}
}
func metric(errors, n int) Metric {
	m := Metric{Status: "unavailable", ObservedErrors: errors, Errors: errors, N: n, IndependentComponents: true, Wilson95: Wilson(errors, n)}
	if n > 0 {
		m.Status = "complete"
		m.Value = ptr(float64(errors) / float64(n))
	}
	return m
}

// pythonRandom reproduces CPython random.Random(integer), including randrange
// rejection sampling; it is not Go's different math/rand generator.
type pythonRandom struct {
	state [624]uint32
	index int
}

func newPythonRandom(seed uint32) *pythonRandom {
	r := &pythonRandom{}
	r.state[0] = 19650218
	for i := 1; i < 624; i++ {
		r.state[i] = 1812433253*(r.state[i-1]^(r.state[i-1]>>30)) + uint32(i)
	}
	i := 1
	for k := 0; k < 624; k++ {
		r.state[i] = (r.state[i] ^ ((r.state[i-1] ^ (r.state[i-1] >> 30)) * 1664525)) + seed
		i++
		if i >= 624 {
			r.state[0] = r.state[623]
			i = 1
		}
	}
	for k := 0; k < 623; k++ {
		r.state[i] = (r.state[i] ^ ((r.state[i-1] ^ (r.state[i-1] >> 30)) * 1566083941)) - uint32(i)
		i++
		if i >= 624 {
			r.state[0] = r.state[623]
			i = 1
		}
	}
	r.state[0] = 0x80000000
	r.index = 624
	return r
}
func (r *pythonRandom) uint32() uint32 {
	if r.index >= 624 {
		for i := 0; i < 624; i++ {
			y := (r.state[i] & 0x80000000) | (r.state[(i+1)%624] & 0x7fffffff)
			r.state[i] = r.state[(i+397)%624] ^ (y >> 1)
			if y&1 != 0 {
				r.state[i] ^= 0x9908b0df
			}
		}
		r.index = 0
	}
	y := r.state[r.index]
	r.index++
	y ^= y >> 11
	y ^= (y << 7) & 0x9d2c5680
	y ^= (y << 15) & 0xefc60000
	y ^= y >> 18
	return y
}
func (r *pythonRandom) randrange(n int) int {
	bits := 0
	for x := n; x > 0; x >>= 1 {
		bits++
	}
	for {
		v := int(r.uint32() >> uint(32-bits))
		if v < n {
			return v
		}
	}
}
func quantile(a []float64, p float64) float64 {
	x := float64(len(a)-1) * p
	lo, hi := int(math.Floor(x)), int(math.Ceil(x))
	return a[lo] + (a[hi]-a[lo])*(x-float64(lo))
}

// Bootstrap synchronously samples the sorted union for every supplied statistic.
// A nil statistic denotes an empty subgroup in that replicate, never zero.
func Bootstrap(ids []string, statistics []func([]string) *float64) ([][]float64, []int) {
	ids = append([]string{}, ids...)
	sort.Strings(ids)
	values := make([][]float64, len(statistics))
	valid := make([]int, len(statistics))
	if len(ids) == 0 {
		return values, valid
	}
	rng := newPythonRandom(271828)
	sample := make([]string, len(ids))
	for replicate := 0; replicate < 10000; replicate++ {
		for i := range sample {
			sample[i] = ids[rng.randrange(len(ids))]
		}
		for j, f := range statistics {
			v := f(sample)
			if v != nil {
				values[j] = append(values[j], *v)
				valid[j]++
			}
		}
	}
	cis := make([][]float64, len(statistics))
	for j, v := range values {
		if len(v) >= 9500 {
			sort.Float64s(v)
			cis[j] = []float64{quantile(v, .025), quantile(v, .975)}
		}
	}
	return cis, valid
}
func Paired(baseline, challenger, cohort string, base, new map[string]bool, union []string) (PairedComparison, error) {
	result := PairedComparison{Baseline: baseline, Challenger: challenger, Cohort: cohort}
	errors := 0
	for id, b := range base {
		n, ok := new[id]
		if !ok {
			return result, fmt.Errorf("missing paired component %s", id)
		}
		result.N++
		if b {
			errors++
		}
		if b && !n {
			result.Repaired++
		}
		if !b && n {
			result.Worsened++
		}
	}
	if len(base) != len(new) {
		return result, fmt.Errorf("paired cohort differs")
	}
	result.Net = result.Repaired - result.Worsened
	result.DiscordantComponents = result.Repaired + result.Worsened
	if errors > 0 {
		result.RelativeErrorReduction = ptr(float64(result.Net) / float64(errors))
	}
	if result.N == 0 {
		return result, nil
	}
	ci, valid := Bootstrap(union, []func([]string) *float64{func(sample []string) *float64 {
		n, delta := 0, 0
		for _, id := range sample {
			b, ok := base[id]
			if !ok {
				continue
			}
			n++
			c := new[id]
			if b && !c {
				delta++
			}
			if !b && c {
				delta--
			}
		}
		if n == 0 {
			return nil
		}
		return ptr(float64(delta) / float64(n))
	}})
	result.AbsoluteErrorReductionCi95 = ci[0]
	result.ValidBootstrap = valid[0]
	return result, nil
}

func validateMetric(m Metric) error {
	if m.Errors != m.ObservedErrors+m.UnavailableComponents || m.UnavailableComponents > m.N || m.ObservedErrors > m.N-m.UnavailableComponents {
		return fmt.Errorf("metric charged/observed/unavailable counts inconsistent")
	}
	if m.Status == "complete" && (m.N == 0 || m.UnavailableComponents != 0 || m.Value == nil || len(m.Wilson95) != 2) {
		return fmt.Errorf("complete metric lacks observed rate")
	}
	if m.Status == "unavailable" && (m.Value != nil || m.Wilson95 != nil) {
		return fmt.Errorf("unavailable metric fabricates rate/interval")
	}
	return nil
}
