package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"text/tabwriter"

	"github.com/RJuho/jokateko/internal/version"
)

func cmdLicenses(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("licenses", flag.ContinueOnError)
	fs.SetOutput(stderr)
	jsonFlag := fs.Bool("json", false, "Output all licenses as structured JSON")
	fullFlag := fs.Bool("full", false, "Display full license texts for all dependencies")

	if err := fs.Parse(args); err != nil {
		return 1
	}

	report := version.GetLicenses()

	if *jsonFlag {
		enc := json.NewEncoder(stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(report); err != nil {
			fmt.Fprintf(stderr, "failed to encode licenses JSON: %v\n", err)
			return 1
		}
		return 0
	}

	fmt.Fprintln(stdout, "Jokateko Open Source Licenses")
	fmt.Fprintf(stdout, "Project License: %s (%s)\n", report.Project.License, report.Project.URL)
	fmt.Fprintln(stdout, "For project information and details, run: jokateko about")
	fmt.Fprintln(stdout, "")

	var goPkgs []version.PackageLicense
	var npmPkgs []version.PackageLicense

	for _, p := range report.Packages {
		if p.Ecosystem == "go" {
			goPkgs = append(goPkgs, p)
		} else {
			npmPkgs = append(npmPkgs, p)
		}
	}

	if *fullFlag {
		fmt.Fprintln(stdout, "================================================================================")
		fmt.Fprintf(stdout, "JOKATEKO (%s)\n", report.Project.License)
		fmt.Fprintln(stdout, "================================================================================")
		fmt.Fprintln(stdout, report.Project.Text)

		for _, p := range report.Packages {
			fmt.Fprintln(stdout, "================================================================================")
			fmt.Fprintf(stdout, "%s %s (%s) [%s]\n", p.Name, p.Version, p.License, p.Ecosystem)
			if p.URL != "" {
				fmt.Fprintf(stdout, "URL: %s\n", p.URL)
			}
			fmt.Fprintln(stdout, "================================================================================")
			if p.Text != "" {
				fmt.Fprintln(stdout, p.Text)
			} else {
				fmt.Fprintf(stdout, "Standard %s License. See package repository for full details.\n\n", p.License)
			}
		}
		return 0
	}

	// Tabular summary output
	printTable := func(title string, pkgs []version.PackageLicense) {
		fmt.Fprintf(stdout, "%s (%d):\n", title, len(pkgs))
		w := tabwriter.NewWriter(stdout, 0, 0, 3, ' ', 0)
		fmt.Fprintln(w, "  PACKAGE\tVERSION\tLICENSE\tURL")
		for _, p := range pkgs {
			fmt.Fprintf(w, "  %s\t%s\t%s\t%s\n", p.Name, p.Version, p.License, p.URL)
		}
		_ = w.Flush()
		fmt.Fprintln(stdout, "")
	}

	printTable("Go Backend Dependencies", goPkgs)
	printTable("Web Frontend Dependencies", npmPkgs)

	fmt.Fprintln(stdout, "Flags:")
	fmt.Fprintln(stdout, "  --full    Display full license texts for all dependencies")
	fmt.Fprintln(stdout, "  --json    Output complete license dataset as JSON")

	return 0
}
