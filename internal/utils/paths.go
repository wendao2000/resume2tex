package utils

import (
	"os"
	"path/filepath"
	"strings"
)

// CompanionTeXPath replaces the PDF path extension with .tex.
func CompanionTeXPath(pdfPath string) string {
	return strings.TrimSuffix(pdfPath, filepath.Ext(pdfPath)) + ".tex"
}

// SamePath reports whether two paths have the same absolute spelling or refer
// to the same existing file, including symbolic links and hard links.
func SamePath(first, second string) bool {
	firstAbs, firstErr := filepath.Abs(first)
	secondAbs, secondErr := filepath.Abs(second)
	if firstErr == nil && secondErr == nil && firstAbs == secondAbs {
		return true
	}
	firstInfo, firstErr := os.Stat(first)
	secondInfo, secondErr := os.Stat(second)
	return firstErr == nil && secondErr == nil && os.SameFile(firstInfo, secondInfo)
}
