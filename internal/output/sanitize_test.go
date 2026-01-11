package output

import (
	"fmt"
	"strings"
	"testing"
	"unicode"
)

func FuzzAppendEscapedRune(f *testing.F) {
	f.Add(uint32(0x00))
	f.Add(uint32(0x1b))
	f.Add(uint32(0x7f))
	f.Add(uint32(0x80))
	f.Add(uint32(0xff))
	f.Add(uint32(0x100))
	f.Add(uint32(0x20ac))
	f.Add(uint32(0xffff))
	f.Add(uint32(0x10000))
	f.Add(uint32(0x10ffff))

	f.Fuzz(func(t *testing.T, raw uint32) {
		// keep this within the valid Unicode scalar range
		r := rune(raw % (unicode.MaxRune + 1))

		var b strings.Builder
		appendEscapedRune(&b, r)
		got := b.String()

		var want string
		switch {
		case r <= 0xFF:
			want = fmt.Sprintf(`\\x%02x`, r)
		case r <= 0xFFFF:
			want = fmt.Sprintf(`\\u%04x`, r)
		default:
			want = fmt.Sprintf(`\\U%08x`, r)
		}

		if got != want {
			t.Fatalf("appendEscapedRune(%#x) = %q, want %q", r, got, want)
		}

		// output must be visible ascii
		for i := 0; i < len(got); i++ {
			if got[i] >= 0x80 {
				t.Fatalf("appendEscapedRune(%#x) produced non-ASCII byte 0x%02x in %q", r, got[i], got)
			}
		}
	})
}
