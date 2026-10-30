package layout

// AdjacentKeys returns immediate neighbors on an ANSI QWERTY keyboard. Physical
// geometry is independent of the selected Zhuyin mapping. Space, modifiers and
// shifted characters are deliberately absent. Coordinates use quarter-key units.
func AdjacentKeys(key rune) []rune {
	rows := []struct {
		keys string
		x    int
	}{{"1234567890-=", 0}, {"qwertyuiop[]", 2}, {"asdfghjkl;'", 3}, {"zxcvbnm,./", 5}}
	row, x := -1, 0
	for y, r := range rows {
		for i, k := range r.keys {
			if k == key {
				row, x = y, r.x+4*i
			}
		}
	}
	if row < 0 {
		return nil
	}
	var out []rune
	for y, r := range rows {
		if y < row-1 || y > row+1 {
			continue
		}
		for i, k := range r.keys {
			dx := r.x + 4*i - x
			if k != key && (y == row && (dx == -4 || dx == 4) || y != row && dx >= -3 && dx <= 3) {
				out = append(out, k)
			}
		}
	}
	return out
}
