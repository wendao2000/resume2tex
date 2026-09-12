package utils

import (
	"fmt"
	"net/url"
	"strings"
)

var phoneReplacer = strings.NewReplacer(" ", "", "-", "", "(", "", ")", "", ".", "")

// EscapeTeXURL encodes a destination for a hyperref argument. It encodes
// characters outside URI syntax before quoting hyperref's argument characters.
func EscapeTeXURL(target string) string {
	if parsed, err := url.Parse(target); err == nil {
		target = parsed.String()
	}
	const uriCharacters = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789-._~:/?#[]@!$&'()*+,;="
	var out strings.Builder
	for i := 0; i < len(target); i++ {
		c := target[i]
		if c == '%' && i+2 < len(target) && isHex(target[i+1]) && isHex(target[i+2]) {
			out.WriteString(`\%`)
			out.WriteString(target[i+1 : i+3])
			i += 2
			continue
		}
		if !strings.ContainsRune(uriCharacters, rune(c)) {
			fmt.Fprintf(&out, `\%%%02X`, c)
			continue
		}
		switch c {
		case '#', '&', '_', '~':
			out.WriteByte('\\')
		}
		out.WriteByte(c)
	}
	return out.String()
}

func isHex(c byte) bool {
	return c >= '0' && c <= '9' || c >= 'a' && c <= 'f' || c >= 'A' && c <= 'F'
}

// Link creates a LaTeX hyperlink, escaping its destination and label separately.
// A missing destination produces plain text; a missing label uses the URL.
func Link(target, label string) string {
	if !HasText(label) {
		label = FormatURL(target)
	}
	if strings.TrimSpace(target) == "" {
		return EscapeTeX(label)
	}
	return `\href{` + EscapeTeXURL(target) + `}{` + EscapeTeX(label) + `}`
}

// EmailURL encodes an email address as a mailto URI.
func EmailURL(address string) string {
	i := strings.LastIndexByte(address, '@')
	if i < 0 {
		return ""
	}
	component := func(s string) string { return strings.ReplaceAll(url.QueryEscape(s), "+", "%20") }
	return "mailto:" + component(address[:i]) + "@" + component(address[i+1:])
}

// FormatURL shortens an absolute URL to its host and escaped path for display.
func FormatURL(value string) string {
	u, err := url.Parse(value)
	if err != nil || u.Host == "" {
		return value
	}
	return u.Host + u.EscapedPath()
}

// FormatPhone removes common visual separators while retaining a leading plus.
func FormatPhone(value string) string {
	return phoneReplacer.Replace(value)
}

// PhoneURL returns a tel URI for a global number, or an empty string for a local
// or unrecognized number that should remain visible without a guessed context.
func PhoneURL(value string) string {
	number := FormatPhone(value)
	if len(number) < 2 || number[0] != '+' {
		return ""
	}
	for _, c := range number[1:] {
		if c < '0' || c > '9' {
			return ""
		}
	}
	return "tel:" + number
}
