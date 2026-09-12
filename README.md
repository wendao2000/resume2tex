**resume2tex**

Keep your resume in version control and quickly generate LaTeX or PDF from JSON using Go. The current input format adapts JSON Resume with employer groups and nested roles. The project prioritizes readable, extractable text for online job applications. The built-in template is embedded in the executable; you can also provide a custom Go text template.

A native resume format with optional tags and profiles is being designed; a [minimal starter](examples/resume-minimal.json) shows the authoring format. Its schema and examples are review drafts; the commands below use the current `basics`/`work` format shown in [examples/resume.json](examples/resume.json).

**Build and run.** Use Go 1.26.0 or newer. Building the Go program, running `make check`, validating input, and generating TeX require no TeX installation. **PDF generation is the default, so plain `make run` requires `pdflatex` on `PATH`.** Use `-format tex` to generate only TeX, or select an installed alternative with `-engine tectonic`, `-engine xelatex`, or `-engine lualatex`.

For your first run, copy the fictional sample to the default personal input file, then replace its content with your own verified information:

```sh
cp -n examples/resume.json resume.json
```

The copy command preserves an existing `resume.json`. Tests read `examples/resume.json` directly and do not depend on your personal input.

For a standalone compiler, install [Tectonic with Homebrew](https://formulae.brew.sh/formula/tectonic) and select it explicitly:

```sh
brew install tectonic
make run ARGS='-engine tectonic -timeout 10m'
```

This produces `output/resume.pdf` and its companion `output/resume.tex`. Tectonic downloads required support files on demand and caches them for later runs. Its initial downloads and LaTeX format setup count toward `-timeout` and can take several minutes; the first build needs network access. The example allows ten minutes, but slower downloads may need more time. [Tectonic getting started](https://tectonic-typesetting.github.io/book/latest/getting-started/first-document.html)

If Tectonic times out, inspect the `tectonic-output.log` path printed in the error for recent downloads or compilation messages. Retry with a larger `-timeout` if setup is still progressing. Completed downloads remain cached and are reused on subsequent runs.

For `pdflatex`, `xelatex`, or `lualatex`, install a TeX distribution using the official instructions: [TeX Live](https://tug.org/texlive/quickinstall.html), [MacTeX for macOS](https://tug.org/mactex/mactex-download.html), or [MiKTeX](https://miktex.org/download). Ensure the selected engine's executable is on `PATH`. The default template needs these LaTeX packages: `inputenc`, `fontenc`, `geometry`, `hyperref`, `enumitem`, `xcolor`, `tabularx`, `needspace`, and `tcolorbox` with its `breakable` library and dependencies. The program uses your selected engine; it does not install a compiler or automatically switch engines.

```sh
go build -o bin/resume2tex ./cmd/app

# Validate input without rendering or invoking a compiler.
./bin/resume2tex -resume resume.json -validate

# Generate only the LaTeX file.
./bin/resume2tex -resume resume.json -format tex -output output/resume.tex

# Generate a PDF and its companion output/resume.tex.
./bin/resume2tex -resume resume.json -output output/resume.pdf

# Generate the PDF with an installed alternative engine.
./bin/resume2tex -resume resume.json -engine xelatex

# Supply a custom template and compiler timeout.
./bin/resume2tex -resume resume.json -template internal/latex/templates/default.tex -timeout 60s
```

`go run ./cmd/app` accepts the same flags. With no flags, the program reads `resume.json` from the current directory and generates `output/resume.pdf` plus `output/resume.tex`. The embedded template works when the binary runs outside the repository. To render the fictional sample directly, pass `-resume examples/resume.json`.

The Makefile provides the same workflow:

```sh
make                          # Build bin/resume2tex; no TeX required.
make run ARGS='-validate'      # Validate without TeX.
make run ARGS='-format tex'    # Generate output/resume.tex without TeX.
make run                      # Generate PDF; requires pdflatex.
make run ARGS='-engine xelatex'  # Requires xelatex instead.
make run ARGS='-engine lualatex' # Requires lualatex instead.
make run ARGS='-engine tectonic -timeout 10m' # Requires tectonic; allows extra time for initial downloads.
make check                    # Run tests and go vet; no TeX required.
make fmt                      # Format Go sources.
make clean                    # Remove bin/; retain generated resumes.
make help                     # List all targets and prerequisites.
```

`make test` and `make vet` run either check separately. Set `GO` to select a Go executable, for example `make build GO=/usr/local/go/bin/go`; `ARGS` supplies flags to `make run`. Make is optional.

**Command-line flags.** String and duration flags accept either `-flag value` or `-flag=value`; quote paths containing spaces. There are no positional arguments. Unknown flags, unsupported formats or engine names, and nonpositive timeouts fail with an error.

| Flag | Default | Behavior |
| --- | --- | --- |
| `-resume path` | `resume.json` | Read JSON from this file. A blank path is invalid. |
| `-template path` | Empty; embedded template | Use a custom Go text template. The built-in source is `internal/latex/templates/default.tex`. Ignored in validation mode. |
| `-format pdf` or `-format tex` | `pdf` | Generate a PDF with companion TeX, or only TeX. Only PDF generation requires a compiler. |
| `-engine name` | `pdflatex` | Select `pdflatex`, `xelatex`, `lualatex`, or `tectonic` for PDF generation. The selected executable must be installed on `PATH`; no fallback is attempted. TeX output and validation check the name but do not locate or invoke the engine. |
| `-output path` | `output/resume.pdf` or `output/resume.tex`, selected by format | Select the destination file and create missing parent directories. PDF's companion has the same basename and a `.tex` extension; the two paths must differ. An empty value selects the default. |
| `-timeout duration` | `30s` | Bound PDF compilation using a positive Go duration such as `500ms`, `60s`, or `2m`. Includes any Tectonic downloads; allow more time for its first run. No compiler runs for TeX or validation mode, but the value must still be positive. |
| `-validate` | `false` | Check the resume and report diagnostics without reading a template, creating outputs, or locating the compiler. Use `-validate=true` or `-validate=false` for an explicit value. |
| `-h`, `-help` | Off | Print usage and exit successfully without reading input. |

Relative file paths are resolved from the current working directory. Existing outputs are replaced on success; destinations that alias the input resume or custom template are rejected. Validation mode still checks flag syntax and values. Warnings go to standard error, successful results to standard output, and errors produce a nonzero exit status.

**Experience belongs to employers and roles.** Each `work` entry holds `name`, `url`, `location`, an optional employer `summary`, and `roles`. Put `position`, `startDate`, `endDate`, a role-specific `summary`, and `highlights` inside each role, even for a single position:

```json
{
  "name": "Example Company",
  "summary": "Builds developer tools.",
  "roles": [
    {
      "position": "Software Engineer",
      "startDate": "2022-01",
      "endDate": "2024-03",
      "summary": "Maintained the build platform.",
      "highlights": ["Reduced build time by 30%."]
    }
  ]
}
```

When migrating older files, remove company `description` and its display text. Preserve existing summaries, moving role-specific summaries and flat role fields into `roles`. Keep each role's dates instead of repeating an aggregate employer range. Projects use `summary` instead of `description` and `startDate`/`endDate` instead of `status`; preserve any useful project description text in the summary. The removed fields produce unsupported-field warnings and are not rendered. Custom templates must use the updated fields too.

**Input checks catch errors before rendering.** The program requires a nonblank `basics.name`, checks known field types, validates calendar dates and date ranges, and checks email and absolute HTTP/HTTPS links. Errors identify fields such as `work[0].roles[0].endDate`. Unknown fields produce warnings instead of silently disappearing; permitted extensions remain accepted but are unavailable to templates unless added to the Go model. Validation covers the fields understood by this application and does not claim full JSON Schema conformance or fetch `$schema` over the network.

Dates accept `YYYY`, `YYYY-MM`, and `YYYY-MM-DD`. The default template displays work, project, and volunteer ranges as month/year, education ranges as years, and individual award/publication dates at their supplied precision, including the day when given. Year-only input stays year-only, and the source data is unchanged. A work role, project, or volunteer position with a start date and an empty or omitted end date is treated as ongoing and ends in `Present`. Leave both fields empty or omitted when dates are unknown; no date label is shown. Education dates do not infer ongoing status. Partial dates are compared as intervals, so overlapping periods are allowed. For completed work or projects with an unknown end, omit both dates until the dates are known rather than supplying a start-only range that implies ongoing activity.

**The default template preserves the supported content and omits empty sections.** Page numbers are hidden, and Technical Skills uses the same line spacing as bullet lists. A section that fits on one page stays together, moving to the next page when needed. Longer sections split across pages and repeat their heading with `(cont.)`. This can leave unused space on the preceding page. The following table describes what is displayed:

| Data | Default behavior |
| --- | --- |
| Basics | Name, headline, city/region/country, populated contacts, all profiles with a URL or username, and summary. Global phone numbers retain `+`; local or unrecognized phone formats remain visible without a guessed dialing link. |
| Work | Employer, location, and employer summary once, followed by each role's title, dates, summary, and highlights in authored order. |
| Skills | Category names and keywords; subjective `level` labels are omitted. |
| Projects | Linked name, role names, right-aligned month/year dates, a separate keyword line with a bold `Technologies:` label, summary, and highlights. `entity` and `type` remain available to custom templates. |
| Education | Linked institution, location, degree/subject, dates, score, and courses. |
| Awards | Issuer, location, title, date, and summary. |
| Volunteer work | Linked organization, position, dates, summary, and highlights. |
| Publications | Linked title, publisher, date, and summary. |
| Languages | Language and supplied fluency. |
| Photo, interests, references | Omitted with a warning when populated; available to custom templates. |
| Address/postal code and metadata | Retained as input fields for custom templates; omitted from the default document. |
| Certificates and other unmodeled fields | Reported as unsupported; extending the data model is a separate change. |

**Resume text is plain text.** The built-in template escapes TeX special characters, and hyperlink destinations use separate URL escaping. Custom templates are trusted LaTeX code and must apply the same helpers to their own text fields: `escapeTeX`, `link target label`, `formatDate`, and `dateRange start end ongoing`. `link` handles the destination and label separately; do not escape a URL before passing it to that helper. Existing `join`, `formatUrl`, `formatPhone`, and `getProfileUrl` helpers remain available. The original `.Work` input is accessible to custom templates; `.Experience` filters empty employers and roles for display without changing the source.

`monthYearRange start end ongoing` and `yearRange start end ongoing` provide the default template's compact range styles. They validate the original date before shortening its display. `formatDate` and `dateRange` retain their existing source-precision behavior for custom templates.

**PDF failures retain useful diagnostics.** The compiler runs noninteractively with a timeout and shell escape disabled. Its input search path includes the original working directory and custom template directory. For `pdflatex`, `xelatex`, and `lualatex`, this is followed by any existing `TEXINPUTS` configuration (or the TeX distribution defaults when unset). Tectonic uses its standalone `-X compile` command with `--keep-logs` and explicit `-Z search-path` arguments; it does not read `TEXINPUTS`. An inherited `TECTONIC_UNTRUSTED_MODE` setting disables those extra Tectonic search paths. [Tectonic compiler options](https://tectonic-typesetting.github.io/book/latest/v2cli/compile.html)

A failed compilation preserves existing final outputs and reports a temporary directory containing the generated TeX and available compiler logs, including `<engine>-output.log`. Successful builds stage their files before replacing the requested destinations. Each replacement is atomic; publishing the PDF and its companion is not a single filesystem transaction. The program refuses destinations that would overwrite the input resume or custom template.

If removing the temporary build directory fails after publication, the error identifies the directory and confirms that the PDF and companion TeX were already saved.

On Unix, new output directories use `0700` permissions and new files use `0600`, subject to the process umask. Replacing a file preserves its existing permissions; existing directory permissions are retained.

**Run the regression checks.** These tests cover input diagnostics, rendering and content preservation, links, engine selection, and compiler failures using a controlled stand-in for the TeX executable:

```sh
go test ./...
go vet ./...
```

With Tectonic installed and its package cache populated by a successful PDF build, run the optional pagination regression checks:

```sh
RESUME2TEX_TEST_TECTONIC=1 go test ./internal/latex -run TestSectionPaginationWithTectonic
```

These checks use cached packages only and verify that short sections stay together, oversized paragraphs and lists split with continuation headings, and section bodies are typeset once.

An actual PDF build remains a separate check requiring the selected engine and TeX packages above. The sample has been compiled successfully with Tectonic 0.17.0, including a repeat build using its populated cache. Both pages of the sample PDF have been visually checked, with readable extracted text, link annotations, and no text extending beyond the page boundaries. Additional PDF fixtures verified oversized paragraphs/lists, repeated continuation headings, and short sections near page boundaries without losing or duplicating text. Other engines are covered with stand-ins. Inspect the resulting PDF for line wrapping, page breaks, links, and text extraction when changing the template or engine.

**Code layout.** The executable owns process startup and exit; the implementation stays in `internal/` packages, with focused helpers separated from orchestration.

| Path | Responsibility |
| --- | --- |
| `cmd/app/main.go` | Call the CLI and translate errors into process exit status. |
| `internal/cli/` | Parse flags into local options and coordinate validation, rendering, and output. |
| `internal/resume/` | Resume data types, decoding, and validation. |
| `internal/latex/` | Template execution, rendering data, and `templates/default.tex`. |
| `internal/pdf/` | Compiler execution, timeout, and build diagnostics. |
| `internal/utils/` | Date parsing/formatting, TeX text/link helpers, output staging, atomic replacement, and path comparisons. |
| `internal/utils/testutil/` | Shared compiler test helpers, kept separate from production utilities. |

**Go version policy.** `go 1.26.0` in `go.mod` declares the required minimum and the language semantics used by this module. A newer installed toolchain can build it; there is no separate `toolchain` directive selecting a preferred version. [Go module reference](https://go.dev/doc/modules/gomod-ref#go)

Go 1.26 is the project's baseline because it is the oldest supported release branch as of 12 September 2026. Go supports a release until two newer major releases exist. Use a current patch release of a supported branch for development. [Go release policy and history](https://go.dev/doc/devel/release)

**Why `StringVar` for flags.** `FlagSet.String` creates flag storage and returns a `*string`; `FlagSet.StringVar` receives a pointer to existing storage, such as `&opts.ResumePath`. Both use the same string parsing. A local options struct keeps the CLI's related values together without global state, so this project uses `StringVar`, `DurationVar`, and `BoolVar`. [Go flag documentation](https://pkg.go.dev/flag#FlagSet.StringVar)
