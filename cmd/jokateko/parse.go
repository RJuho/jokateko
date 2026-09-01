package main

import (
	"flag"
	"fmt"
	"io"
	"path/filepath"

	"github.com/RJuho/jokateko/internal/validator"
)

func cmdParse(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("parse", flag.ContinueOnError)
	fs.SetOutput(stderr)
	dirFlag := fs.String("dir", ".", "Project root directory to validate")

	if err := fs.Parse(args); err != nil {
		return 1
	}

	targetDir, err := filepath.Abs(*dirFlag)
	if err != nil {
		fmt.Fprintf(stderr, "failed to resolve directory %q: %v\n", *dirFlag, err)
		return 1
	}

	res, err := validator.ValidateWorkspace(targetDir)
	if err != nil {
		fmt.Fprintf(stderr, "error validating workspace %q: %v\n", targetDir, err)
		return 1
	}

	report := res.FormatReport()
	if res.HasErrors() {
		fmt.Fprintln(stderr, report)
		return 1
	}

	fmt.Fprintln(stdout, report)
	return 0
}
