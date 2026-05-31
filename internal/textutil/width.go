// Package textutil provides text display utilities.
package textutil

func DisplayWidth(s string) int {
	width := 0
	for _, r := range s {
		switch {
		case r == 0:
		case r < 0x20 || (r >= 0x7f && r < 0xa0):
		case isWide(r):
			width += 2
		default:
			width++
		}
	}
	return width
}

func PadRight(s string, width int) string {
	pad := width - DisplayWidth(s)
	for pad > 0 {
		s += " "
		pad--
	}
	return s
}

func Truncate(s string, width int) string {
	if width <= 0 {
		return ""
	}
	if DisplayWidth(s) <= width {
		return s
	}
	out := ""
	for _, r := range s {
		next := out + string(r)
		if DisplayWidth(next+"…") > width {
			break
		}
		out = next
	}
	return out + "…"
}

func isWide(r rune) bool {
	return (r >= 0x1100 && r <= 0x115f) ||
		(r >= 0x2e80 && r <= 0xa4cf) ||
		(r >= 0xac00 && r <= 0xd7a3) ||
		(r >= 0xf900 && r <= 0xfaff) ||
		(r >= 0xfe10 && r <= 0xfe19) ||
		(r >= 0xfe30 && r <= 0xfe6f) ||
		(r >= 0xff00 && r <= 0xff60) ||
		(r >= 0xffe0 && r <= 0xffe6)
}
