// Package latex renders resume data into LaTeX with context-aware escaping.
package latex

import (
	_ "embed"
	"fmt"
	"io"
	"strings"
	"text/template"

	model "github.com/wendao2000/resume2tex/internal/resume"
	"github.com/wendao2000/resume2tex/internal/utils"
)

// DefaultTemplate is embedded so the binary can run outside the source tree.
//
//go:embed templates/default.tex
var DefaultTemplate string

var funcMap = template.FuncMap{
	"join": strings.Join, "joinNonempty": utils.JoinNonempty,
	"escapeTeX": utils.EscapeTeX, "escapeTeXURL": utils.EscapeTeXURL, "link": utils.Link,
	"formatDate": utils.FormatDate, "dateRange": utils.DateRange,
	"monthYearRange": utils.MonthYearRange, "yearRange": utils.YearRange,
	"formatPhone": utils.FormatPhone, "formatUrl": utils.FormatURL,
	"hasText": utils.HasText, "nonempty": utils.Nonempty,
	"getProfileUrl": func(profiles []model.Profile, network string) string {
		for _, profile := range profiles {
			if strings.EqualFold(profile.Network, network) {
				return profile.URL
			}
		}
		return ""
	},
}

// Render executes source with the resume and its normalized display fields.
func Render(w io.Writer, source string, resume model.Resume) error {
	tmpl, err := template.New("resume").Funcs(funcMap).Parse(source)
	if err != nil {
		return fmt.Errorf("parse template: %w", err)
	}
	if err := tmpl.Execute(w, makeView(resume)); err != nil {
		return fmt.Errorf("render template: %w", err)
	}
	return nil
}
