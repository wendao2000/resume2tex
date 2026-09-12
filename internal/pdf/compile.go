// Package pdf compiles LaTeX and publishes PDF output.
package pdf

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/wendao2000/resume2tex/internal/utils"
)

// Compile runs the LaTeX compiler in an isolated directory, then publishes the
// PDF and companion TeX file. Failed builds retain their source and diagnostics.
// A cleanup failure returns the published companion path and an error identifying
// the build directory that could not be fully removed.
func Compile(ctx context.Context, compiler string, source []byte, destination string, timeout time.Duration, templateDirectories ...string) (string, error) {
	engine := filepath.Base(compiler)
	workingDirectory, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("resolve TeX working directory: %w", err)
	}
	searchDirectories := []string{workingDirectory}
	for _, directory := range templateDirectories {
		absoluteDirectory, err := filepath.Abs(directory)
		if err != nil {
			return "", fmt.Errorf("resolve template directory: %w", err)
		}
		searchDirectories = append(searchDirectories, absoluteDirectory)
	}
	buildDir, err := os.MkdirTemp("", "resume2tex-build-")
	if err != nil {
		return "", fmt.Errorf("create PDF build directory: %w", err)
	}
	texPath := filepath.Join(buildDir, "resume.tex")
	logPath := filepath.Join(buildDir, engine+"-output.log")
	failure := func(err error) (string, error) {
		return "", fmt.Errorf("%w; build files retained in %q (LaTeX: %s; compiler output: %s)", err, buildDir, texPath, logPath)
	}
	if err := os.WriteFile(texPath, source, 0600); err != nil {
		if cleanupErr := os.RemoveAll(buildDir); cleanupErr != nil {
			return "", fmt.Errorf("write temporary LaTeX: %w; remove PDF build directory %q: %w", err, buildDir, cleanupErr)
		}
		return "", fmt.Errorf("write temporary LaTeX: %w", err)
	}
	logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0600)
	if err != nil {
		return failure(fmt.Errorf("create compiler output log: %w", err))
	}
	compileCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	cmd := compilerCommand(compileCtx, compiler, buildDir, searchDirectories)
	cmd.Stdout = logFile
	cmd.Stderr = logFile
	runErr := cmd.Run()
	closeErr := logFile.Close()
	contextErr := compileCtx.Err()
	if contextErr == context.DeadlineExceeded {
		timeoutErr := fmt.Errorf("%s timed out after %s: %w", engine, timeout, contextErr)
		if isTectonic(compiler) {
			timeoutErr = fmt.Errorf("%w; Tectonic's initial downloads count toward -timeout; if the compiler log shows downloads, retry with a longer -timeout", timeoutErr)
		}
		return failure(timeoutErr)
	}
	if contextErr != nil {
		return failure(fmt.Errorf("%s interrupted: %w", engine, contextErr))
	}
	if runErr != nil {
		return failure(fmt.Errorf("%s failed: %w", engine, runErr))
	}
	if closeErr != nil {
		return failure(fmt.Errorf("close compiler output log: %w", closeErr))
	}
	pdf, err := os.ReadFile(filepath.Join(buildDir, "resume.pdf"))
	if err != nil {
		return failure(fmt.Errorf("%s did not produce a readable PDF: %w", engine, err))
	}
	if !bytes.HasPrefix(pdf, []byte("%PDF-")) {
		return failure(fmt.Errorf("%s did not produce a PDF with a valid header", engine))
	}
	// Stage both files before replacing either destination. Publish the PDF last,
	// so a failed compilation or staging operation always preserves the prior PDF.
	pdfStage, err := utils.Stage(destination, pdf)
	if err != nil {
		return failure(fmt.Errorf("stage PDF %q: %w", destination, err))
	}
	defer os.Remove(pdfStage)
	companion := utils.CompanionTeXPath(destination)
	texStage, err := utils.Stage(companion, source)
	if err != nil {
		return failure(fmt.Errorf("stage LaTeX %q: %w", companion, err))
	}
	defer os.Remove(texStage)
	if err := os.Rename(texStage, companion); err != nil {
		return failure(fmt.Errorf("publish LaTeX %q: %w", companion, err))
	}
	if err := os.Rename(pdfStage, destination); err != nil {
		return failure(fmt.Errorf("publish PDF %q (updated LaTeX is at %q): %w", destination, companion, err))
	}
	if err := os.RemoveAll(buildDir); err != nil {
		return companion, fmt.Errorf("published PDF %q and LaTeX %q, but could not fully remove PDF build directory %q: %w", destination, companion, buildDir, err)
	}
	return companion, nil
}
