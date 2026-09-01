package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"

	"github.com/RJuho/jokateko/internal/version"
)

func cmdVersion(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("version", flag.ContinueOnError)
	fs.SetOutput(stderr)
	jsonFlag := fs.Bool("json", false, "Output version information as JSON")

	if err := fs.Parse(args); err != nil {
		return 1
	}

	info := version.Get()
	if *jsonFlag {
		enc := json.NewEncoder(stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(info); err != nil {
			fmt.Fprintf(stderr, "failed to encode version info: %v\n", err)
			return 1
		}
		return 0
	}

	fmt.Fprintln(stdout, info.String())
	return 0
}
