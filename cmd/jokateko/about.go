package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"

	"github.com/RJuho/jokateko/internal/version"
)

type AboutInfo struct {
	Name        string       `json:"name"`
	Description string       `json:"description"`
	Repository  string       `json:"repository"`
	License     string       `json:"license"`
	LicenseText string       `json:"license_text"`
	Build       version.Info `json:"build"`
}

func cmdAbout(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("about", flag.ContinueOnError)
	fs.SetOutput(stderr)
	jsonFlag := fs.Bool("json", false, "Output about and license details as JSON")

	if err := fs.Parse(args); err != nil {
		return 1
	}

	info := version.Get()
	report := version.GetLicenses()

	about := AboutInfo{
		Name:        "Jokateko",
		Description: "Local, Markdown-driven Kanban and task management for developers and AI agents",
		Repository:  "https://github.com/RJuho/jokateko",
		License:     report.Project.License,
		LicenseText: report.Project.Text,
		Build:       info,
	}

	if *jsonFlag {
		enc := json.NewEncoder(stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(about); err != nil {
			fmt.Fprintf(stderr, "failed to encode about info: %v\n", err)
			return 1
		}
		return 0
	}

	fmt.Fprintln(stdout, "Jokateko")
	fmt.Fprintln(stdout, "Local, Markdown-driven Kanban and task management for developers and AI agents")
	fmt.Fprintln(stdout, "")
	fmt.Fprintf(stdout, "  Version:     %s\n", info.Version)
	fmt.Fprintf(stdout, "  Commit:      %s\n", info.Commit)
	fmt.Fprintf(stdout, "  Build Date:  %s\n", info.Date)
	fmt.Fprintf(stdout, "  Go Version:  %s\n", info.GoVersion)
	fmt.Fprintf(stdout, "  Platform:    %s\n", info.Platform)
	fmt.Fprintf(stdout, "  Repository:  https://github.com/RJuho/jokateko\n")
	fmt.Fprintf(stdout, "  License:     %s\n", about.License)
	fmt.Fprintln(stdout, "")
	fmt.Fprintln(stdout, "--- Project License ---")
	fmt.Fprint(stdout, about.LicenseText)
	if len(about.LicenseText) == 0 || about.LicenseText[len(about.LicenseText)-1] != '\n' {
		fmt.Fprintln(stdout)
	}
	fmt.Fprintln(stdout, "-----------------------")
	fmt.Fprintln(stdout, "")
	fmt.Fprintln(stdout, "To inspect bundled third-party open-source libraries and licenses, run:")
	fmt.Fprintln(stdout, "  jokateko licenses")

	return 0
}
