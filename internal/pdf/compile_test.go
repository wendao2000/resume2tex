package pdf

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/wendao2000/resume2tex/internal/utils"
	"github.com/wendao2000/resume2tex/internal/utils/testutil"
)

func TestMain(m *testing.M) {
	testutil.Main(m)
}

func TestCompilePDFPublishesAndCleansBuild(t *testing.T) {
	for _, engine := range []string{"pdflatex", "tectonic", "tectonic.exe", "TECTONIC.EXE"} {
		t.Run(engine, func(t *testing.T) {
			compiler, invocationPath := testutil.InstallNamedCompiler(t, engine, "success")
			destination := filepath.Join(t.TempDir(), "nested", "custom.pdf")
			source := []byte("complete source")
			companion, err := Compile(context.Background(), compiler, source, destination, 5*time.Second)
			if err != nil {
				t.Fatal(err)
			}
			if companion != filepath.Join(filepath.Dir(destination), "custom.tex") {
				t.Fatalf("unexpected companion path %q", companion)
			}
			if got, err := os.ReadFile(companion); err != nil || !bytes.Equal(got, source) {
				t.Fatalf("companion = %q, error = %v", got, err)
			}
			if got, err := os.ReadFile(destination); err != nil || !bytes.HasPrefix(got, []byte("%PDF-")) {
				t.Fatalf("PDF = %q, error = %v", got, err)
			}
			invocation := testutil.ReadInvocation(t, invocationPath)
			wantArgs := []string{"-no-shell-escape", "-halt-on-error", "-file-line-error", "-interaction=nonstopmode", "resume.tex"}
			if engine != "pdflatex" {
				workingDirectory, err := os.Getwd()
				if err != nil {
					t.Fatal(err)
				}
				wantArgs = []string{"-X", "compile", "--keep-logs", "--outfmt", "pdf", "-Z", "search-path=" + workingDirectory, "resume.tex"}
			}
			if !slices.Equal(invocation.Args, wantArgs) {
				t.Fatalf("compiler arguments = %q, want %q", invocation.Args, wantArgs)
			}
			if _, err := os.Stat(invocation.Directory); !os.IsNotExist(err) {
				t.Fatalf("successful build directory remains: %v", err)
			}
		})
	}
}

func TestCompilePDFReportsCleanupFailureAfterPublishing(t *testing.T) {
	if runtime.GOOS == "windows" || os.Geteuid() == 0 {
		t.Skip("requires a user subject to Unix directory write permissions")
	}
	compiler, invocationPath := testutil.InstallCompiler(t, "cleanup-fail")
	destination := filepath.Join(t.TempDir(), "resume.pdf")
	source := []byte("complete source")
	companion, err := Compile(context.Background(), compiler, source, destination, 5*time.Second)
	invocation := testutil.ReadInvocation(t, invocationPath)
	t.Cleanup(func() {
		if err := os.Chmod(filepath.Join(invocation.Directory, "protected"), 0700); err != nil {
			t.Error(err)
		}
	})
	if !errors.Is(err, os.ErrPermission) {
		t.Fatalf("expected cleanup permission error, got %v", err)
	}
	for _, detail := range []string{"published PDF", "could not fully remove PDF build directory", destination, utils.CompanionTeXPath(destination), invocation.Directory} {
		if !strings.Contains(err.Error(), detail) {
			t.Errorf("cleanup error does not include %q: %v", detail, err)
		}
	}
	if companion != utils.CompanionTeXPath(destination) {
		t.Fatalf("published companion path = %q", companion)
	}
	if got, err := os.ReadFile(companion); err != nil || !bytes.Equal(got, source) {
		t.Fatalf("published companion = %q, error = %v", got, err)
	}
	if got, err := os.ReadFile(destination); err != nil || !bytes.HasPrefix(got, []byte("%PDF-")) {
		t.Fatalf("published PDF = %q, error = %v", got, err)
	}
}

func TestCompilePDFFailuresPreserveOutputsAndDiagnostics(t *testing.T) {
	for _, engine := range []string{"pdflatex", "tectonic"} {
		for _, mode := range []string{"fail", "timeout", "no-pdf", "bad-pdf"} {
			t.Run(engine+"/"+mode, func(t *testing.T) {
				compiler, invocationPath := testutil.InstallNamedCompiler(t, engine, mode)
				destination := filepath.Join(t.TempDir(), "resume.pdf")
				companion := utils.CompanionTeXPath(destination)
				for _, path := range []string{destination, companion} {
					if err := os.WriteFile(path, []byte("previous successful output"), 0600); err != nil {
						t.Fatal(err)
					}
				}
				timeout := 5 * time.Second
				if mode == "timeout" {
					timeout = 250 * time.Millisecond
				}
				started := time.Now()
				_, err := Compile(context.Background(), compiler, []byte("new source"), destination, timeout)
				if err == nil {
					t.Fatal("expected compilation to fail")
				}
				if mode == "timeout" && (!errors.Is(err, context.DeadlineExceeded) || time.Since(started) > 5*time.Second) {
					t.Fatalf("compiler did not respect its timeout: %v", err)
				}
				wantDownloadHint := engine == "tectonic" && mode == "timeout"
				if strings.Contains(err.Error(), "initial downloads") != wantDownloadHint || strings.Contains(err.Error(), "longer -timeout") != wantDownloadHint {
					t.Fatalf("download guidance does not match failure: %v", err)
				}
				invocation := testutil.ReadInvocation(t, invocationPath)
				if !strings.Contains(err.Error(), invocation.Directory) {
					t.Fatalf("error does not locate diagnostics: %v", err)
				}
				for _, name := range []string{"resume.tex", "resume.log", engine + "-output.log"} {
					data, readErr := os.ReadFile(filepath.Join(invocation.Directory, name))
					if readErr != nil || len(data) == 0 {
						t.Errorf("missing retained %s: %v", name, readErr)
					}
				}
				for _, path := range []string{destination, companion} {
					data, err := os.ReadFile(path)
					if err != nil || string(data) != "previous successful output" {
						t.Errorf("previous output %s changed to %q: %v", path, data, err)
					}
				}
			})
		}
	}
}

func TestCompileCancellationDoesNotSuggestLongerTimeout(t *testing.T) {
	compiler, _ := testutil.InstallNamedCompiler(t, "tectonic", "success")
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := Compile(ctx, compiler, []byte("source"), filepath.Join(t.TempDir(), "resume.pdf"), 5*time.Second)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected canceled compilation, got %v", err)
	}
	if strings.Contains(err.Error(), "timeout") || strings.Contains(err.Error(), "initial downloads") {
		t.Fatalf("cancellation should not be reported as a timeout: %v", err)
	}
}

func TestCompilePDFCompanionFailurePreservesPDF(t *testing.T) {
	compiler, _ := testutil.InstallCompiler(t, "success")
	destination := filepath.Join(t.TempDir(), "resume.pdf")
	if err := os.WriteFile(destination, []byte("previous PDF"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(utils.CompanionTeXPath(destination), 0700); err != nil {
		t.Fatal(err)
	}
	if _, err := Compile(context.Background(), compiler, []byte("source"), destination, 5*time.Second); err == nil {
		t.Fatal("expected companion publication to fail")
	}
	if data, err := os.ReadFile(destination); err != nil || string(data) != "previous PDF" {
		t.Fatalf("previous PDF changed to %q: %v", data, err)
	}
}

func TestCompileNamesSelectedEngineInFailureDiagnostics(t *testing.T) {
	for _, engine := range []string{"xelatex", "lualatex", "tectonic"} {
		t.Run(engine, func(t *testing.T) {
			compiler, invocationPath := testutil.InstallNamedCompiler(t, engine, "fail")
			destination := filepath.Join(t.TempDir(), "resume.pdf")
			_, err := Compile(context.Background(), compiler, []byte("source"), destination, 5*time.Second)
			if err == nil || !strings.Contains(err.Error(), engine+" failed") {
				t.Fatalf("error did not identify selected engine: %v", err)
			}
			invocation := testutil.ReadInvocation(t, invocationPath)
			logPath := filepath.Join(invocation.Directory, engine+"-output.log")
			if data, err := os.ReadFile(logPath); err != nil || !bytes.Contains(data, []byte("test compiler diagnostic")) {
				t.Fatalf("missing selected-engine log: %q, %v", data, err)
			}
		})
	}
}

func TestCompileTectonicPreservesTemplateInputSearch(t *testing.T) {
	compiler, invocationPath := testutil.InstallNamedCompiler(t, "tectonic", "success")
	workingDirectory := t.TempDir()
	templateDirectory := filepath.Join(workingDirectory, "custom templates")
	if err := os.Mkdir(templateDirectory, 0700); err != nil {
		t.Fatal(err)
	}
	t.Chdir(workingDirectory)
	destination := filepath.Join(t.TempDir(), "resume.pdf")
	if _, err := Compile(context.Background(), compiler, []byte("source"), destination, 5*time.Second, "custom templates"); err != nil {
		t.Fatal(err)
	}
	invocation := testutil.ReadInvocation(t, invocationPath)
	wantArgs := []string{
		"-X", "compile", "--keep-logs", "--outfmt", "pdf",
		"-Z", "search-path=" + workingDirectory,
		"-Z", "search-path=" + templateDirectory,
		"resume.tex",
	}
	if !slices.Equal(invocation.Args, wantArgs) {
		t.Fatalf("compiler arguments = %q, want %q", invocation.Args, wantArgs)
	}
}

func TestCompilePDFPreservesTeXInputSearch(t *testing.T) {
	compiler, invocationPath := testutil.InstallCompiler(t, "success")
	workingDirectory := t.TempDir()
	templateDirectory := filepath.Join(workingDirectory, "templates")
	if err := os.Mkdir(templateDirectory, 0700); err != nil {
		t.Fatal(err)
	}
	t.Chdir(workingDirectory)
	separator := string(os.PathListSeparator)
	existingInputs := filepath.Join(t.TempDir(), "tex-assets") + separator
	t.Setenv("TEXINPUTS", existingInputs)
	destination := filepath.Join(t.TempDir(), "resume.pdf")
	if _, err := Compile(context.Background(), compiler, []byte("source"), destination, 5*time.Second, "templates"); err != nil {
		t.Fatal(err)
	}
	invocation := testutil.ReadInvocation(t, invocationPath)
	want := invocation.Directory + separator + workingDirectory + separator + templateDirectory + separator + existingInputs
	if invocation.TeXInputs != want {
		t.Fatalf("TEXINPUTS = %q, want %q", invocation.TeXInputs, want)
	}
}
