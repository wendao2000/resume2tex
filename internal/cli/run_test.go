package cli

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/wendao2000/resume2tex/internal/utils"
	"github.com/wendao2000/resume2tex/internal/utils/testutil"
)

func writeCLIResume(t *testing.T, directory string) string {
	t.Helper()
	path := filepath.Join(directory, "resume.json")
	if err := os.WriteFile(path, []byte(`{"basics":{"name":"Example Person"}}`), 0600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestMain(m *testing.M) { testutil.Main(m) }

func TestRunEmbeddedTemplateAndDefaultPathsOutsideRepository(t *testing.T) {
	for _, format := range []string{"tex", "pdf"} {
		t.Run(format, func(t *testing.T) {
			directory := t.TempDir()
			writeCLIResume(t, directory)
			if format == "pdf" {
				testutil.InstallCompiler(t, "success")
			} else {
				t.Setenv("PATH", t.TempDir())
			}
			t.Chdir(directory)
			var stdout, stderr bytes.Buffer
			args := []string(nil)
			if format == "tex" {
				args = []string{"-format", "tex"}
			}
			if err := Run(context.Background(), args, &stdout, &stderr); err != nil {
				t.Fatalf("run: %v\n%s", err, stderr.String())
			}
			data, err := os.ReadFile(filepath.Join(directory, "output", "resume.tex"))
			if err != nil || !strings.Contains(string(data), "Example Person") {
				t.Fatalf("embedded template output = %q, error = %v", data, err)
			}
			if !strings.Contains(stdout.String(), "output"+string(filepath.Separator)+"resume."+format) {
				t.Fatalf("success does not identify output: %s", stdout.String())
			}
		})
	}
}

func TestRunCustomTemplateAndOutput(t *testing.T) {
	directory := t.TempDir()
	resume := writeCLIResume(t, directory)
	template := filepath.Join(directory, "custom.tex")
	if err := os.WriteFile(template, []byte("Custom: {{.Basics.Name}}"), 0600); err != nil {
		t.Fatal(err)
	}
	destination := filepath.Join(directory, "nested", "custom-output.tex")
	var stdout, stderr bytes.Buffer
	args := []string{"-resume", resume, "-template", template, "-format", "tex", "-output", destination}
	if err := Run(context.Background(), args, &stdout, &stderr); err != nil {
		t.Fatal(err)
	}
	if data, err := os.ReadFile(destination); err != nil || string(data) != "Custom: Example Person" {
		t.Fatalf("custom output = %q, error = %v", data, err)
	}
}

func TestRunUsesSelectedEngine(t *testing.T) {
	for _, engine := range []string{"pdflatex", "xelatex", "lualatex", "tectonic"} {
		t.Run(engine, func(t *testing.T) {
			_, invocationPath := testutil.InstallNamedCompiler(t, engine, "success")
			directory := t.TempDir()
			resume := writeCLIResume(t, directory)
			destination := filepath.Join(directory, "result.pdf")
			var stdout, stderr bytes.Buffer
			err := Run(context.Background(), []string{"-resume", resume, "-engine", engine, "-output", destination}, &stdout, &stderr)
			if err != nil {
				t.Fatal(err)
			}
			if data, err := os.ReadFile(destination); err != nil || !bytes.HasPrefix(data, []byte("%PDF-")) {
				t.Fatalf("selected engine did not publish a PDF: %q, %v", data, err)
			}
			invocation := testutil.ReadInvocation(t, invocationPath)
			if engine == "tectonic" {
				if got := strings.Join(invocation.Args, " "); !strings.HasPrefix(got, "-X compile ") || strings.Contains(got, "-no-shell-escape") || strings.Contains(got, "-interaction=") {
					t.Fatalf("unexpected Tectonic arguments: %q", got)
				}
				return
			}
			if got := strings.Join(invocation.Args, " "); got != "-no-shell-escape -halt-on-error -file-line-error -interaction=nonstopmode resume.tex" {
				t.Fatalf("unexpected engine arguments: %q", got)
			}
		})
	}
}

func TestRunDoesNotReplaceUnavailableSelectedEngine(t *testing.T) {
	for _, engine := range []string{"xelatex", "tectonic"} {
		t.Run(engine, func(t *testing.T) {
			_, invocationPath := testutil.InstallCompiler(t, "success")
			resume := writeCLIResume(t, t.TempDir())
			var stdout, stderr bytes.Buffer
			err := Run(context.Background(), []string{"-resume", resume, "-engine", engine}, &stdout, &stderr)
			if err == nil || !strings.Contains(err.Error(), "requires "+engine) || !strings.Contains(err.Error(), "-format tex") || !strings.Contains(err.Error(), "-engine") {
				t.Fatalf("missing selected-engine prerequisite or alternatives: %v", err)
			}
			if engine == "tectonic" && !strings.Contains(err.Error(), "brew install tectonic") {
				t.Fatalf("missing Tectonic installation guidance: %v", err)
			}
			if _, err := os.Stat(invocationPath); !os.IsNotExist(err) {
				t.Fatalf("a different engine was invoked: %v", err)
			}
		})
	}
}

func TestRunWithoutTeXInstallation(t *testing.T) {
	for _, test := range []struct{ mode, engine string }{
		{"tex", "lualatex"},
		{"validate", "lualatex"},
		{"tex", "tectonic"},
		{"validate", "tectonic"},
	} {
		t.Run(test.mode+"/"+test.engine, func(t *testing.T) {
			t.Setenv("PATH", t.TempDir())
			directory := t.TempDir()
			resume := writeCLIResume(t, directory)
			destination := filepath.Join(directory, "result.tex")
			args := []string{"-resume", resume, "-engine", test.engine, "-output", destination}
			if test.mode == "tex" {
				args = append(args, "-format", "tex")
			} else {
				args = append(args, "-validate")
			}
			var stdout, stderr bytes.Buffer
			if err := Run(context.Background(), args, &stdout, &stderr); err != nil {
				t.Fatalf("%s should work without a TeX engine: %v", test.mode, err)
			}
			if test.mode == "tex" {
				data, err := os.ReadFile(destination)
				if err != nil || !strings.Contains(string(data), `\end{document}`) {
					t.Fatalf("missing LaTeX output: %v", err)
				}
			} else if _, err := os.Stat(destination); !os.IsNotExist(err) {
				t.Fatalf("validation wrote output: %v", err)
			}
		})
	}
}

func TestRunMissingCompilerIsActionableAndPreservesOutput(t *testing.T) {
	directory := t.TempDir()
	resume := writeCLIResume(t, directory)
	destination := filepath.Join(directory, "resume.pdf")
	if err := os.WriteFile(destination, []byte("previous PDF"), 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", t.TempDir())
	var stdout, stderr bytes.Buffer
	err := Run(context.Background(), []string{"-resume", resume, "-output", destination}, &stdout, &stderr)
	if err == nil || !strings.Contains(err.Error(), "pdflatex") || !strings.Contains(err.Error(), "-format tex") {
		t.Fatalf("unhelpful missing compiler error: %v", err)
	}
	if stdout.Len() != 0 {
		t.Fatalf("reported success: %s", stdout.String())
	}
	if data, err := os.ReadFile(destination); err != nil || string(data) != "previous PDF" {
		t.Fatalf("old PDF changed to %q: %v", data, err)
	}
	if _, err := os.Stat(utils.CompanionTeXPath(destination)); !os.IsNotExist(err) {
		t.Fatalf("preflight created TeX output: %v", err)
	}
}

func TestRunValidationRequiresNeitherTemplateNorCompiler(t *testing.T) {
	directory := t.TempDir()
	resume := writeCLIResume(t, directory)
	t.Setenv("PATH", t.TempDir())
	var stdout, stderr bytes.Buffer
	err := Run(context.Background(), []string{"-resume", resume, "-template", "does-not-exist.tex", "-validate"}, &stdout, &stderr)
	if err != nil || !strings.Contains(stdout.String(), "validation passed") {
		t.Fatalf("validation: %v, output: %s", err, stdout.String())
	}
	if err := os.WriteFile(resume, []byte(`{"basics":{"nmae":"Typo"}}`), 0600); err != nil {
		t.Fatal(err)
	}
	stdout.Reset()
	stderr.Reset()
	err = Run(context.Background(), []string{"-resume", resume, "-validate"}, &stdout, &stderr)
	if err == nil || !strings.Contains(err.Error(), "basics.name") || !strings.Contains(stderr.String(), "basics.nmae") {
		t.Fatalf("missing validation diagnostics: error = %v, stderr = %s", err, stderr.String())
	}
}

func TestRunRejectsInvalidFlags(t *testing.T) {
	for _, args := range [][]string{{"-format", "html"}, {"-engine", "unknown"}, {"-engine", ""}, {"-timeout", "0"}, {"-timeout", "-1s"}, {"-resume", ""}, {"-unknown"}, {"extra.json"}} {
		t.Run(strings.Join(args, " "), func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			if err := Run(context.Background(), args, &stdout, &stderr); err == nil {
				t.Fatal("expected argument error")
			}
		})
	}
	var stdout, stderr bytes.Buffer
	if err := Run(context.Background(), []string{"-help"}, &stdout, &stderr); err != nil || !strings.Contains(stderr.String(), "-format") || !strings.Contains(stderr.String(), "-engine tectonic") {
		t.Fatalf("help: %v, stderr = %s", err, stderr.String())
	}
}

func TestRunPreventsInputOverwrite(t *testing.T) {
	for _, kind := range []string{"same", "symlink", "hardlink", "template", "companion-template"} {
		t.Run(kind, func(t *testing.T) {
			directory := t.TempDir()
			resume := writeCLIResume(t, directory)
			destination := resume
			args := []string{"-resume", resume, "-format", "tex"}
			switch kind {
			case "symlink", "hardlink":
				destination = filepath.Join(directory, "linked.tex")
				link := os.Symlink
				if kind == "hardlink" {
					link = os.Link
				}
				if err := link(resume, destination); err != nil {
					t.Skipf("filesystem does not support %s: %v", kind, err)
				}
			case "template", "companion-template":
				template := filepath.Join(directory, "template.tex")
				if err := os.WriteFile(template, []byte("original template"), 0600); err != nil {
					t.Fatal(err)
				}
				destination = template
				args = append(args, "-template", template)
				if kind == "companion-template" {
					destination = filepath.Join(directory, "template.pdf")
					args = append(args, "-format", "pdf")
				}
			}
			args = append(args, "-output", destination)
			var stdout, stderr bytes.Buffer
			if err := Run(context.Background(), args, &stdout, &stderr); err == nil || !strings.Contains(err.Error(), "overwrite input") {
				t.Fatalf("expected overwrite protection, got %v", err)
			}
		})
	}
}

func TestRunRenderFailurePreservesTeX(t *testing.T) {
	directory := t.TempDir()
	resume := writeCLIResume(t, directory)
	template := filepath.Join(directory, "invalid.tex")
	destination := filepath.Join(directory, "result.tex")
	if err := os.WriteFile(template, []byte("{{"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(destination, []byte("previous TeX"), 0600); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	err := Run(context.Background(), []string{"-resume", resume, "-template", template, "-format", "tex", "-output", destination}, &stdout, &stderr)
	if err == nil {
		t.Fatal("expected invalid template error")
	}
	if data, err := os.ReadFile(destination); err != nil || string(data) != "previous TeX" {
		t.Fatalf("old TeX changed to %q: %v", data, err)
	}
}
