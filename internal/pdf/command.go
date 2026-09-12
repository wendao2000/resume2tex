package pdf

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func isTectonic(compiler string) bool {
	name := strings.TrimSuffix(strings.ToLower(filepath.Base(compiler)), ".exe")
	return name == "tectonic"
}

func compilerCommand(ctx context.Context, compiler, buildDir string, searchDirectories []string) *exec.Cmd {
	var args, env []string
	if isTectonic(compiler) {
		args = []string{"-X", "compile", "--keep-logs", "--outfmt", "pdf"}
		// Tectonic ignores TEXINPUTS; --untrusted would also disable these paths.
		// Shell escape is disabled by default.
		for _, directory := range searchDirectories {
			args = append(args, "-Z", "search-path="+directory)
		}
		args = append(args, "resume.tex")
	} else {
		args = []string{"-no-shell-escape", "-halt-on-error", "-file-line-error", "-interaction=nonstopmode", "resume.tex"}
		// An empty trailing TEXINPUTS component preserves distribution defaults.
		separator := string(os.PathListSeparator)
		texInputs := buildDir + separator + strings.Join(searchDirectories, separator) + separator + os.Getenv("TEXINPUTS")
		env = append(os.Environ(), "TEXINPUTS="+texInputs)
	}
	cmd := exec.CommandContext(ctx, compiler, args...)
	cmd.Dir = buildDir
	cmd.Env = env
	return cmd
}
