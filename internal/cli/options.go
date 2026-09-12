package cli

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"strings"
	"time"
)

type options struct {
	ResumePath   string
	TemplatePath string
	Format       string
	Engine       string
	OutputPath   string
	Timeout      time.Duration
	ValidateOnly bool
}

func parseOptions(args []string, stderr io.Writer) (options, error) {
	var opts options
	flags := flag.NewFlagSet("resume2tex", flag.ContinueOnError)
	flags.SetOutput(stderr)
	flags.StringVar(&opts.ResumePath, "resume", "resume.json", "Resume JSON `file` to validate and render")
	flags.StringVar(&opts.TemplatePath, "template", "", "Custom Go/LaTeX template `file`; defaults to the embedded template")
	flags.StringVar(&opts.Format, "format", "pdf", "Output `format`: pdf (requires the selected TeX engine) or tex (no TeX installation needed)")
	flags.StringVar(&opts.Engine, "engine", "pdflatex", "Installed PDF `engine` on PATH: pdflatex, xelatex, lualatex, or tectonic; unused with -format tex or -validate")
	flags.StringVar(&opts.OutputPath, "output", "", "Destination `file`; defaults to output/resume.<format>; PDF also writes a companion .tex")
	flags.DurationVar(&opts.Timeout, "timeout", 30*time.Second, "Positive PDF compiler `duration`, including Tectonic downloads, such as 30s or 10m")
	flags.BoolVar(&opts.ValidateOnly, "validate", false, "Check resume data and exit; skip template loading, output writing, and PDF compilation")
	flags.Usage = func() {
		fmt.Fprintln(stderr, `Usage: resume2tex [flags]

Generate LaTeX or PDF from a JSON resume.
PDF output (the default) requires an installed TeX engine: pdflatex by default.
Use -engine xelatex, -engine lualatex, or -engine tectonic to select another installed engine.
Tectonic caches downloaded TeX resources; first use may need several minutes and a longer -timeout.
Without TeX installed, use -format tex to generate only LaTeX, or -validate to check input.

Flags:`)
		flags.PrintDefaults()
		fmt.Fprintln(stderr, "  -h, -help\n    \tShow this help and exit.")
		fmt.Fprintln(stderr, `
Examples:
  resume2tex -resume resume.json -validate
  resume2tex -format tex -output output/resume.tex
  resume2tex -engine xelatex -output output/resume.pdf
  resume2tex -engine tectonic -output output/resume.pdf -timeout 10m
  resume2tex -output output/resume.pdf -timeout 60s`)
	}
	if err := flags.Parse(args); err != nil {
		return opts, err
	}
	if flags.NArg() != 0 {
		return opts, fmt.Errorf("unexpected positional arguments: %s", strings.Join(flags.Args(), " "))
	}
	if strings.TrimSpace(opts.ResumePath) == "" {
		return opts, errors.New("-resume must specify a JSON file")
	}
	if opts.Format != "pdf" && opts.Format != "tex" {
		return opts, fmt.Errorf("unsupported -format %q: use pdf or tex", opts.Format)
	}
	switch opts.Engine {
	case "pdflatex", "xelatex", "lualatex", "tectonic":
	default:
		return opts, fmt.Errorf("unsupported -engine %q: use pdflatex, xelatex, lualatex, or tectonic", opts.Engine)
	}
	if opts.Timeout <= 0 {
		return opts, errors.New("-timeout must be greater than zero")
	}
	return opts, nil
}
