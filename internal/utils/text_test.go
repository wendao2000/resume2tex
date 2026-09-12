package utils

import "testing"

func TestEscapeTeXPlainText(t *testing.T) {
	tests := []struct{ input, want string }{
		{"plain text + punctuation", "plain text + punctuation"},
		{`$5 & 20% #1 foo_bar {item} \input{file}`, `\$5 \& 20\% \#1 foo\_bar \{item\} \textbackslash{}input\{file\}`},
		{`~ ^ ≈`, `\textasciitilde{} \textasciicircum{} \ensuremath{\approx}`},
		{`\{}$&%#_~^≈`, `\textbackslash{}\{\}\$\&\%\#\_\textasciitilde{}\textasciicircum{}\ensuremath{\approx}`},
	}
	for _, tt := range tests {
		if got := EscapeTeX(tt.input); got != tt.want {
			t.Errorf("EscapeTeX(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}
