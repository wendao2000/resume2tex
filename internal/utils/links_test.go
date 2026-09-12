package utils

import "testing"

func TestLinkSeparatesURLTargetFromVisibleLabel(t *testing.T) {
	tests := []struct {
		name, target, label, want string
	}{
		{
			"fragment, query, and encoded slash",
			"https://example.org/a_b%2Fc?q=x_y&lang=en#top#part", "",
			`\href{https://example.org/a\_b\%2Fc?q=x\_y\&lang=en\#top\%23part}{example.org/a\_b\%2Fc}`,
		},
		{
			"label parsed before text escaping",
			"https://example.org/me#top", "",
			`\href{https://example.org/me\#top}{example.org/me}`,
		},
		{
			"tilde in target and label",
			"https://example.org/~alex", "",
			`\href{https://example.org/\~alex}{example.org/\textasciitilde{}alex}`,
		},
		{
			"TeX syntax encoded in target and escaped in label",
			`https://example.org/\input{evil}^name`, `$label_{x}\evil`,
			`\href{https://example.org/\%5Cinput\%7Bevil\%7D\%5Ename}{\$label\_\{x\}\textbackslash{}evil}`,
		},
		{
			"literal percent in query",
			"https://example.org/search?q=100%", "Search & results",
			`\href{https://example.org/search?q=100\%25}{Search \& results}`,
		},
		{
			"label without destination",
			"", "GitHub: alex_dev",
			`GitHub: alex\_dev`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Link(tt.target, tt.label); got != tt.want {
				t.Errorf("Link() = %q, want %q", got, tt.want)
			}
		})
	}
}
