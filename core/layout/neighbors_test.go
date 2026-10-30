package layout

import (
	"slices"
	"testing"
)

func TestPhysicalNeighborsFollowStaggeredRows(t *testing.T) {
	for _, k := range "fhtyvb" {
		if !slices.Contains(AdjacentKeys('g'), k) || !slices.Contains(AdjacentKeys(k), 'g') {
			t.Fatalf("missing symmetric neighbor g/%c", k)
		}
	}
	for _, k := range "gapsG \n" {
		if slices.Contains(AdjacentKeys('g'), k) {
			t.Fatalf("not an immediate neighbor: %c", k)
		}
	}
	for _, k := range "G \n" {
		if len(AdjacentKeys(k)) != 0 {
			t.Fatalf("nonphysical/shifted key has neighbors: %q", k)
		}
	}
}
