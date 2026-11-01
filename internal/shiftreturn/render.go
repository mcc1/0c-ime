//go:build shiftreturneval

package shiftreturn

func Render(q Query, h Hypothesis) Display {
	return Display{h.Display, append([]Segment{}, h.DisplaySegments...)}
}
