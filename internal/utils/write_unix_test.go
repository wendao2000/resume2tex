//go:build unix

package utils

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"testing"
)

func TestAtomicWritePermissions(t *testing.T) {
	// Umask is process-wide, so exercise it in a separate test process.
	const childEnv = "RESUME2TEX_TEST_WRITE_PERMISSIONS"
	if os.Getenv(childEnv) != "1" {
		executable, err := os.Executable()
		if err != nil {
			t.Fatal(err)
		}
		cmd := exec.Command(executable, "-test.run=^TestAtomicWritePermissions$")
		cmd.Env = append(os.Environ(), childEnv+"=1")
		if output, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("permission checks: %v\n%s", err, output)
		}
		return
	}

	checkMode := func(t *testing.T, path string, want os.FileMode) {
		t.Helper()
		info, err := os.Stat(path)
		if err != nil {
			t.Fatal(err)
		}
		if got := info.Mode().Perm(); got != want {
			t.Errorf("%s permissions = %04o, want %04o", path, got, want)
		}
	}
	for _, mask := range []int{0022, 0077} {
		t.Run(fmt.Sprintf("umask_%03o", mask), func(t *testing.T) {
			previous := syscall.Umask(mask)
			defer syscall.Umask(previous)

			t.Run("new_outputs_are_private", func(t *testing.T) {
				parent := filepath.Join(t.TempDir(), "output")
				directory := filepath.Join(parent, "nested")
				destination := filepath.Join(directory, "resume.tex")
				if err := AtomicWrite(destination, []byte("new resume")); err != nil {
					t.Fatal(err)
				}
				checkMode(t, parent, 0700)
				checkMode(t, directory, 0700)
				checkMode(t, destination, 0600)
			})

			for _, mode := range []os.FileMode{0600, 0640} {
				t.Run(fmt.Sprintf("preserve_%04o", mode), func(t *testing.T) {
					directory := t.TempDir()
					if err := os.Chmod(directory, 0750); err != nil {
						t.Fatal(err)
					}
					destination := filepath.Join(directory, "resume.tex")
					if err := os.WriteFile(destination, []byte("previous resume"), 0600); err != nil {
						t.Fatal(err)
					}
					if err := os.Chmod(destination, mode); err != nil {
						t.Fatal(err)
					}
					if err := AtomicWrite(destination, []byte("updated resume")); err != nil {
						t.Fatal(err)
					}
					checkMode(t, directory, 0750)
					checkMode(t, destination, mode)
					data, err := os.ReadFile(destination)
					if err != nil || string(data) != "updated resume" {
						t.Fatalf("replacement content = %q, error = %v", data, err)
					}
				})
			}
		})
	}
}
