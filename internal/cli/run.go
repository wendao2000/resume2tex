// Package cli coordinates argument parsing, resume rendering, and output generation.
package cli

import (
	"bytes"
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/wendao2000/resume2tex/internal/latex"
	"github.com/wendao2000/resume2tex/internal/pdf"
	"github.com/wendao2000/resume2tex/internal/resume"
	"github.com/wendao2000/resume2tex/internal/utils"
)

// Run executes one CLI invocation without terminating the process.
func Run(ctx context.Context, args []string, stdout, stderr io.Writer) error {
	opts, err := parseOptions(args, stderr)
	if errors.Is(err, flag.ErrHelp) {
		return nil
	}
	if err != nil {
		return err
	}
	data, err := os.ReadFile(opts.ResumePath)
	if err != nil {
		return fmt.Errorf("read resume %q: %w", opts.ResumePath, err)
	}
	document, diagnostics, err := resume.Decode(data)
	for _, diagnostic := range diagnostics {
		fmt.Fprintln(stderr, "warning:", diagnostic)
	}
	if err != nil {
		return fmt.Errorf("validate resume %q: %w", opts.ResumePath, err)
	}
	if opts.ValidateOnly {
		fmt.Fprintln(stdout, "Resume validation passed.")
		return nil
	}
	if opts.OutputPath == "" {
		opts.OutputPath = filepath.Join("output", "resume."+opts.Format)
	}
	if strings.TrimSpace(opts.OutputPath) == "" {
		return errors.New("-output must specify a destination file")
	}
	destinations := []string{opts.OutputPath}
	if opts.Format == "pdf" {
		texPath := utils.CompanionTeXPath(opts.OutputPath)
		if utils.SamePath(opts.OutputPath, texPath) {
			return fmt.Errorf("PDF destination %q conflicts with its companion .tex file; use a different extension", opts.OutputPath)
		}
		destinations = append(destinations, texPath)
	}
	for _, destination := range destinations {
		for _, input := range []string{opts.ResumePath, opts.TemplatePath} {
			if input != "" && utils.SamePath(destination, input) {
				return fmt.Errorf("output %q would overwrite input %q; choose another -output", destination, input)
			}
		}
	}
	source := latex.DefaultTemplate
	if opts.TemplatePath != "" {
		templateData, err := os.ReadFile(opts.TemplatePath)
		if err != nil {
			return fmt.Errorf("read template %q: %w", opts.TemplatePath, err)
		}
		source = string(templateData)
	} else {
		for _, diagnostic := range latex.DefaultWarnings(document) {
			fmt.Fprintln(stderr, "warning:", diagnostic)
		}
	}
	var compiler string
	if opts.Format == "pdf" {
		compiler, err = exec.LookPath(opts.Engine)
		if err != nil {
			return fmt.Errorf("PDF generation requires %s, but it is unavailable: %w\nInstall a TeX distribution or Tectonic (brew install tectonic), select another installed engine with -engine (pdflatex, xelatex, lualatex, tectonic), or use -format tex -output output/resume.tex to generate LaTeX without a PDF", opts.Engine, err)
		}
		// The compiler runs in an isolated build directory, so resolve a relative PATH entry now.
		compiler, err = filepath.Abs(compiler)
		if err != nil {
			return fmt.Errorf("resolve %s path: %w", opts.Engine, err)
		}
	}
	var rendered bytes.Buffer
	if err := latex.Render(&rendered, source, document); err != nil {
		return fmt.Errorf("render resume: %w", err)
	}
	if opts.Format == "tex" {
		if err := utils.AtomicWrite(opts.OutputPath, rendered.Bytes()); err != nil {
			return fmt.Errorf("write LaTeX %q: %w", opts.OutputPath, err)
		}
		fmt.Fprintf(stdout, "LaTeX generated: %s\n", opts.OutputPath)
		return nil
	}
	var templateDirectories []string
	if opts.TemplatePath != "" {
		templateDirectories = append(templateDirectories, filepath.Dir(opts.TemplatePath))
	}
	texPath, err := pdf.Compile(ctx, compiler, rendered.Bytes(), opts.OutputPath, opts.Timeout, templateDirectories...)
	if err != nil {
		return err
	}
	fmt.Fprintf(stdout, "Resume generated:\n- LaTeX: %s\n- PDF: %s\n", texPath, opts.OutputPath)
	return nil
}
