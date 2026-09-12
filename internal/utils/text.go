// Package utils contains shared date, file, text, and link helpers.
package utils

import "strings"

var texReplacer = strings.NewReplacer(
	`\`, `\textbackslash{}`, "{", `\{`, "}", `\}`, "$", `\$`,
	"&", `\&`, "%", `\%`, "#", `\#`, "_", `\_`,
	"~", `\textasciitilde{}`, "^", `\textasciicircum{}`, "≈", `\ensuremath{\approx}`,
)

// EscapeTeX encodes plain text for display inside a LaTeX document.
func EscapeTeX(s string) string { return texReplacer.Replace(s) }

// HasText reports whether any value contains non-whitespace text.
func HasText(values ...string) bool {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return true
		}
	}
	return false
}

// Nonempty excludes blank strings while preserving the remaining values and order.
func Nonempty(values []string) []string {
	return SelectItems(values, func(s string) bool { return HasText(s) })
}

// JoinNonempty joins values after excluding blank strings.
func JoinNonempty(values []string, separator string) string {
	return strings.Join(Nonempty(values), separator)
}

// SelectItems returns the matching values in their original order.
func SelectItems[T any](values []T, keep func(T) bool) []T {
	var selected []T
	for _, value := range values {
		if keep(value) {
			selected = append(selected, value)
		}
	}
	return selected
}
