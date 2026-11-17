package honorscope

import "strings"

// validUsername matches GitHub's login rules: 1–39 ASCII letters, digits, or
// single hyphens, not starting or ending with a hyphen.
func validUsername(name string) bool {
	if name == "" || len(name) > 39 {
		return false
	}
	if strings.Contains(name, "--") || name[0] == '-' || name[len(name)-1] == '-' {
		return false
	}
	for i := 0; i < len(name); i++ {
		c := name[i]
		switch {
		case c >= 'a' && c <= 'z', c >= 'A' && c <= 'Z', c >= '0' && c <= '9', c == '-':
		default:
			return false
		}
	}
	return true
}
