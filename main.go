package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/template"
	"time"

	"github.com/wendao2000/resumejson2tex/types"
)

var funcMap = template.FuncMap{
	"join": strings.Join,
	"escapeTeX": func(s string) string {
		replacer := strings.NewReplacer(
			"&", "\\&",
			"%", "\\%",
			"$", "\\$",
			"#", "\\#",
			"_", "\\_",
			"{", "\\{",
			"}", "\\}",
			"~", "\\textasciitilde{}",
			"^", "\\textasciicircum{}",
			"\\", "\\textbackslash{}",
		)
		return replacer.Replace(s)
	},
	"formatDate": func(date string) string {
		if date == "" {
			return "Present"
		}
		t1, err := time.Parse("2006-01-02", date)
		if err == nil {
			return t1.Format("January 02, 2006")
		}
		t2, err := time.Parse("2006-01", date)
		if err == nil {
			return t2.Format("January 2006")
		}
		return date
	},
	"formatPhone": func(phone string) string {
		return strings.NewReplacer("+", "", " ", "", "-", "").Replace(phone)
	},
	"formatUrl": func(link string) string {
		url, err := url.Parse(link)
		if err != nil {
			return link
		}
		return url.Host + url.Path
	},
	"getProfileUrl": func(profiles []types.Profile, network string) string {
		for _, p := range profiles {
			if strings.EqualFold(p.Network, network) {
				return p.URL
			}
		}
		return ""
	},
}

var (
	resumeArgs   string
	templateArgs string
)

func validateFlag() error {
	if len(resumeArgs) == 0 || len(templateArgs) == 0 {
		flag.PrintDefaults()
		return errors.New("invalid flag variables")
	}
	return nil
}

func main() {
	flag.StringVar(&resumeArgs, "resume", "resume.json", "Resume file")
	flag.StringVar(&templateArgs, "template", "default.tex", "Template file")
	flag.Parse()

	if validateFlag() != nil {
		log.Fatal("Usage: json2tex [-resume=resume.json] [-template=template.tex]")
	}

	resumeFile, err := os.ReadFile(resumeArgs)
	if err != nil {
		log.Fatalf("Error reading resume file: %v", err)
	}

	templateFile, err := os.ReadFile(templateArgs)
	if err != nil {
		log.Fatalf("Error reading template file: %v", err)
	}

	var resume types.Resume
	if err := json.Unmarshal(resumeFile, &resume); err != nil {
		log.Fatalf("Error parsing resume json: %v", err)
	}

	tmpl, err := template.New("resume").Funcs(funcMap).Parse(string(templateFile))
	if err != nil {
		log.Fatalf("Error parsing template: %v", err)
	}

	outputDir := "output"
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		log.Fatalf("Error creating output directory: %v", err)
	}

	var latexBuffer bytes.Buffer
	if err := tmpl.Execute(&latexBuffer, resume); err != nil {
		log.Fatalf("Error executing template: %v", err)
	}

	latexFile := filepath.Join(outputDir, "resume.tex")
	if err := os.WriteFile(latexFile, latexBuffer.Bytes(), 0644); err != nil {
		log.Fatalf("Error writing LaTeX file: %v", err)
	}

	cmd := exec.Command("pdflatex", "-output-directory", outputDir, latexFile)
	var pdflatexOutput bytes.Buffer
	cmd.Stdout = &pdflatexOutput
	cmd.Stderr = &pdflatexOutput
	if err := cmd.Run(); err != nil {
		fmt.Printf("pdflatex output:\n%s\n", pdflatexOutput.String())
		log.Fatalf("Error running pdflatex: %v", err)
	}

	fmt.Printf("Resume generated successfully:\n")
	fmt.Printf("- LaTeX file: %s\n", latexFile)
	fmt.Printf("- PDF file: %s\n", filepath.Join(outputDir, "resume.pdf"))
}
