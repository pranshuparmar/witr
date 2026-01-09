package output

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

const hexDigits = "0123456789abcdef"

// SanitizeTerminal makes a string safe to print to an interactive terminal
// by just replacing control characters with visible escape sequences (e.g. "\x1b")
// Examples:
//   - "hi\x1b[31mred" -> "hi\x1b[31mred" (ESC becomes visible)
//   - "nul:\x00"      -> "nul:\x00"
//   - "bad:\xff"      -> "bad:\xff" (invalid UTF-8 byte)
//   - "a\tb\nc"        -> "a\tb\nc" (tabs/newlines are untouched)
func SanitizeTerminal(s string) string {
	idx := 0
	// fast path: scan until we find a control rune / invalid UTF-8 byte
	for idx < len(s) {
		r, size := utf8.DecodeRuneInString(s[idx:])
		if r == utf8.RuneError && size == 1 {
			break
		}
		if r == '\n' || r == '\t' {
			idx += size
			continue
		}
		if unicode.IsControl(r) {
			break
		}
		idx += size
	}
	if idx == len(s) {
		return s
	}

	var b strings.Builder
	b.Grow(len(s) + 8)
	b.WriteString(s[:idx])

	// slow path: walk the remainder and rewrite any control bytes/runes
	for idx < len(s) {
		r, size := utf8.DecodeRuneInString(s[idx:])
		if r == utf8.RuneError && size == 1 {
			// preserve invalid bytes without letting them act as controls just in case
			appendEscapedByte(&b, s[idx])
			idx++
			continue
		}

		if r == '\n' || r == '\t' {
			b.WriteRune(r)
			idx += size
			continue
		}

		if unicode.IsControl(r) {
			appendEscapedRune(&b, r)
			idx += size
			continue
		}
		b.WriteString(s[idx : idx+size])
		idx += size
	}

	return b.String()
}

func appendEscapedByte(b *strings.Builder, bt byte) {
	b.WriteString(`\\x`)
	b.WriteByte(hexDigits[bt>>4])
	b.WriteByte(hexDigits[bt&0x0f])
}

// while this looks extremely bad, it just outputs this:
//   - r = 0x1b     -> "\x1b"
//   - r = 0x2028   -> "\u2028"
//   - r = 0x1f600  -> "\U0001f600"
func appendEscapedRune(b *strings.Builder, r rune) {
	// 0xFF: "\xHH" (simple byte escape)
	if r <= 0xFF {
		appendEscapedByte(b, byte(r))
		return
	}

	// <= 0xFFFF: "\uHHHH" (BMP escape)
	if r <= 0xFFFF {
		b.WriteString(`\\u`)
		b.WriteByte(hexDigits[(r>>12)&0x0f])
		b.WriteByte(hexDigits[(r>>8)&0x0f])
		b.WriteByte(hexDigits[(r>>4)&0x0f])
		b.WriteByte(hexDigits[r&0x0f])
		return
	}

	// otherwise: "\UHHHHHHHH" (full 32-bit escape)
	b.WriteString(`\\U`)
	b.WriteByte(hexDigits[(r>>28)&0x0f])
	b.WriteByte(hexDigits[(r>>24)&0x0f])
	b.WriteByte(hexDigits[(r>>20)&0x0f])
	b.WriteByte(hexDigits[(r>>16)&0x0f])
	b.WriteByte(hexDigits[(r>>12)&0x0f])
	b.WriteByte(hexDigits[(r>>8)&0x0f])
	b.WriteByte(hexDigits[(r>>4)&0x0f])
	b.WriteByte(hexDigits[r&0x0f])
}
