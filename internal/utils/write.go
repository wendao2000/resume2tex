package utils

import (
	"fmt"
	"os"
	"path/filepath"
)

// AtomicWrite stages data beside destination before replacing it with a rename.
func AtomicWrite(destination string, data []byte) error {
	staged, err := Stage(destination, data)
	if err != nil {
		return err
	}
	defer os.Remove(staged)
	return os.Rename(staged, destination)
}

// Stage writes data to a temporary file beside destination. The caller must
// rename or remove the returned file. An existing destination must be regular.
// New files and directories are private; replacements retain destination permissions.
func Stage(destination string, data []byte) (string, error) {
	info, err := os.Stat(destination)
	if err == nil && !info.Mode().IsRegular() {
		return "", fmt.Errorf("destination %q is not a regular file", destination)
	} else if err != nil && !os.IsNotExist(err) {
		return "", err
	}
	directory := filepath.Dir(destination)
	if err := os.MkdirAll(directory, 0700); err != nil {
		return "", err
	}
	file, err := os.CreateTemp(directory, ".resume2tex-*")
	if err != nil {
		return "", err
	}
	name := file.Name()
	if _, err := file.Write(data); err != nil {
		file.Close()
		os.Remove(name)
		return "", err
	}
	if info != nil {
		if err := file.Chmod(info.Mode().Perm()); err != nil {
			file.Close()
			os.Remove(name)
			return "", err
		}
	}
	if err := file.Close(); err != nil {
		os.Remove(name)
		return "", err
	}
	return name, nil
}
