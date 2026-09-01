package validator

import (
	"fmt"
	"strings"
)

// Severity indicates the severity level of a validation diagnostic.
type Severity string

const (
	SeverityError   Severity = "ERROR"
	SeverityWarning Severity = "WARNING"
)

// Diagnostic represents an issue found during project parsing and validation.
type Diagnostic struct {
	RuleID   string
	Severity Severity
	File     string
	Line     int
	Message  string
	Fix      string
	Context  []string
}

// String formats the diagnostic compiler-style.
func (d Diagnostic) String() string {
	var sb strings.Builder
	loc := d.File
	if d.Line > 0 {
		loc = fmt.Sprintf("%s:%d", d.File, d.Line)
	}
	fmt.Fprintf(&sb, "%s [%s] %s\n  %s", d.Severity, d.RuleID, loc, d.Message)
	for _, c := range d.Context {
		fmt.Fprintf(&sb, "\n  %s", c)
	}
	if d.Fix != "" {
		fmt.Fprintf(&sb, "\n  Fix: %s", d.Fix)
	}
	return sb.String()
}
