package pkg

import (
	"fmt"
	"os"
)

func sanitizeIdent(s string) string {
	// keep letters/digits/underscore, cannot start with digit
	var out []rune
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' {
			out = append(out, r)
		}
	}
	if len(out) == 0 {
		return ""
	}
	if out[0] >= '0' && out[0] <= '9' {
		out = append([]rune{'p', '_'}, out...)
	}
	return string(out)
}

func must(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
