// Package testutil contains shared test support for CLI and compiler tests.
package testutil

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// Invocation captures how the application started the fake compiler.
type Invocation struct {
	Directory string
	Args      []string
	TeXInputs string
}

// Main runs a package test suite or handles a fake compiler subprocess.
// Packages using InstallCompiler must call Main from their TestMain.
func Main(m *testing.M) {
	if mode := os.Getenv("RESUME2TEX_TEST_COMPILER"); mode != "" {
		compilerHelper(mode)
		return
	}
	os.Exit(m.Run())
}

func compilerHelper(mode string) {
	directory, err := os.Getwd()
	if err != nil {
		os.Exit(2)
	}
	invocation, _ := json.Marshal(Invocation{Directory: directory, Args: os.Args[1:], TeXInputs: os.Getenv("TEXINPUTS")})
	if err := os.WriteFile(os.Getenv("RESUME2TEX_TEST_INVOCATION"), invocation, 0600); err != nil {
		os.Exit(2)
	}
	fmt.Fprintln(os.Stdout, "test compiler diagnostic")
	if err := os.WriteFile("resume.log", []byte("test engine log"), 0600); err != nil {
		os.Exit(2)
	}
	switch mode {
	case "fail":
		os.WriteFile("resume.pdf", []byte("%PDF-partial output"), 0600)
		os.Exit(1)
	case "timeout":
		time.Sleep(10 * time.Minute)
	case "no-pdf":
		return
	case "bad-pdf":
		os.WriteFile("resume.pdf", []byte("not a PDF"), 0600)
	default:
		if err := os.WriteFile("resume.pdf", []byte("%PDF-1.4\ntest PDF\n"), 0600); err != nil {
			os.Exit(2)
		}
		if mode == "cleanup-fail" {
			if err := os.Mkdir("protected", 0700); err != nil {
				os.Exit(2)
			}
			if err := os.WriteFile(filepath.Join("protected", "artifact"), []byte("compiler artifact"), 0600); err != nil {
				os.Exit(2)
			}
			if err := os.Chmod("protected", 0500); err != nil {
				os.Exit(2)
			}
		}
	}
}

// InstallCompiler exposes the test binary as pdflatex and selects its behavior.
// It also isolates PATH and TMPDIR for the calling test.
func InstallCompiler(t *testing.T, mode string) (string, string) {
	t.Helper()
	return InstallNamedCompiler(t, "pdflatex", mode)
}

// InstallNamedCompiler exposes the test binary under the requested engine name.
func InstallNamedCompiler(t *testing.T, name, mode string) (string, string) {
	t.Helper()
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	directory := t.TempDir()
	compiler := filepath.Join(directory, name)
	if err := os.Link(executable, compiler); err != nil {
		data, err := os.ReadFile(executable)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(compiler, data, 0700); err != nil {
			t.Fatal(err)
		}
	}
	invocationPath := filepath.Join(t.TempDir(), "invocation.json")
	t.Setenv("RESUME2TEX_TEST_COMPILER", mode)
	t.Setenv("RESUME2TEX_TEST_INVOCATION", invocationPath)
	t.Setenv("PATH", directory)
	t.Setenv("TMPDIR", t.TempDir())
	return compiler, invocationPath
}

// ReadInvocation reads the invocation recorded by the fake compiler.
func ReadInvocation(t *testing.T, path string) Invocation {
	t.Helper()
	var invocation Invocation
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(data, &invocation); err != nil {
		t.Fatal(err)
	}
	return invocation
}
