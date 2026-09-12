package latex

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"

	model "github.com/wendao2000/resume2tex/internal/resume"
)

// Run with RESUME2TEX_TEST_TECTONIC=1 after warming Tectonic's package cache.
func TestSectionPaginationWithTectonic(t *testing.T) {
	if os.Getenv("RESUME2TEX_TEST_TECTONIC") != "1" {
		t.Skip("set RESUME2TEX_TEST_TECTONIC=1 to run real TeX pagination checks")
	}
	compiler, err := exec.LookPath("tectonic")
	if err != nil {
		t.Fatal(err)
	}
	template := renderForTest(t, model.Resume{Basics: model.Basics{Name: "Pagination test"}})
	// Zero-size deferred writes report the page on which each title is shipped,
	// while preserving the real title formatting and content measurement hooks.
	for _, key := range []string{"title", "title after break"} {
		prefix := key + "={#1"
		if !strings.Contains(template, prefix) {
			t.Fatalf("cannot instrument %s", key)
		}
		marker := "heading"
		if key == "title after break" {
			marker = "continued"
		}
		template = strings.Replace(template, prefix, key+`={\PaginationMark{`+marker+`}#1`, 1)
	}
	preamble, _, ok := strings.Cut(template, `\begin{document}`)
	if !ok {
		t.Fatal("rendered template has no document body")
	}
	preamble += `
\newcount\PaginationEvaluations
\newcommand{\PaginationMark}[1]{\write16{RESUME-PAGE:#1:\thepage}}
\begin{document}
`
	for _, fixture := range []struct {
		name, space, body string
		startPage         int
		splits            bool
	}{
		{"fits", "0.2", "Short section content.", 1, false},
		{"moves_whole", "0.86", strings.Repeat("A short section line.\\\\\n", 6) + "Last line.", 2, false},
		{"long_paragraph", "0.2", strings.Repeat("Continuous paragraph content that must span pages. ", 450), 1, true},
		{"long_list", "0.2", "\\begin{itemize}\n" + strings.Repeat("\\item A list item that must remain in order.\n", 120) + "\\end{itemize}", 1, true},
		{"beyond_dimension_limit", "0.2", strings.Repeat("A paragraph in a very long section.\\par\n", 1300), 1, true},
	} {
		t.Run(fixture.name, func(t *testing.T) {
			directory := t.TempDir()
			source := preamble + fmt.Sprintf(`
Before the section.\PaginationMark{before}\par
\vspace*{%s\textheight}
\begin{resumesection}{Pagination check}
\global\advance\PaginationEvaluations by 1
\noindent\PaginationMark{start}%s\PaginationMark{end}
\end{resumesection}
\typeout{RESUME-EVALUATIONS:\the\PaginationEvaluations}
\end{document}
`, fixture.space, fixture.body)
			path := filepath.Join(directory, "resume.tex")
			if err := os.WriteFile(path, []byte(source), 0600); err != nil {
				t.Fatal(err)
			}
			ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
			defer cancel()
			command := exec.CommandContext(ctx, compiler, "--only-cached", "--keep-logs", "--outdir", directory, path)
			if output, err := command.CombinedOutput(); err != nil {
				t.Fatalf("Tectonic: %v\n%s", err, output)
			}
			log, err := os.ReadFile(filepath.Join(directory, "resume.log"))
			if err != nil {
				t.Fatal(err)
			}
			pages := map[string][]int{}
			for _, match := range regexp.MustCompile(`RESUME-PAGE:(\w+):(\d+)`).FindAllStringSubmatch(string(log), -1) {
				page, _ := strconv.Atoi(match[2])
				pages[match[1]] = append(pages[match[1]], page)
			}
			for marker, want := range map[string]int{"before": 1, "start": fixture.startPage, "heading": fixture.startPage} {
				if got := pages[marker]; len(got) != 1 || got[0] != want {
					t.Fatalf("%s pages = %v, want [%d]; markers: %v", marker, got, want, pages)
				}
			}
			if len(pages["end"]) != 1 {
				t.Fatalf("expected one section end, markers: %v", pages)
			}
			last := pages["end"][0]
			if (last > fixture.startPage) != fixture.splits || last < fixture.startPage {
				t.Fatalf("unexpected section split, markers: %v", pages)
			}
			if len(pages["continued"]) != last-fixture.startPage {
				t.Fatalf("expected a continuation heading on every later section page, markers: %v", pages)
			}
			for index, page := range pages["continued"] {
				if page != fixture.startPage+index+1 {
					t.Fatalf("heading on wrong page, markers: %v", pages)
				}
			}
			if !strings.Contains(string(log), "RESUME-EVALUATIONS:1\n") {
				t.Fatalf("section body was not evaluated exactly once:\n%s", log)
			}
		})
	}
}
