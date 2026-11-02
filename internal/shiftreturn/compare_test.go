//go:build shiftreturneval

package shiftreturn

import (
	"fmt"
	"math"
)

func compareValue(a, b any, path string) error {
	switch x := a.(type) {
	case float64:
		y, ok := b.(float64)
		if !ok || math.Abs(x-y) > 1e-12 {
			return fmt.Errorf("%s numeric mismatch %v/%v", path, a, b)
		}
	case map[string]any:
		y, ok := b.(map[string]any)
		if !ok || len(x) != len(y) {
			return fmt.Errorf("%s object shape mismatch", path)
		}
		for k, v := range x {
			if err := compareValue(v, y[k], path+"."+k); err != nil {
				return err
			}
		}
	case []any:
		y, ok := b.([]any)
		if !ok || len(x) != len(y) {
			return fmt.Errorf("%s array shape mismatch", path)
		}
		for i, v := range x {
			if err := compareValue(v, y[i], fmt.Sprintf("%s[%d]", path, i)); err != nil {
				return err
			}
		}
	default:
		if a != b {
			return fmt.Errorf("%s mismatch %v/%v", path, a, b)
		}
	}
	return nil
}
