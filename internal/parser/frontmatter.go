// Package parser handles TOML frontmatter splitting and decoding,
// Markdown AST parsing, and acceptance criteria extraction for Jokateko.
package parser

import (
	"bytes"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/RJuho/jokateko/internal/model"
	"github.com/pelletier/go-toml/v2"
)

var (
	// ErrNoFrontmatter is returned when content does not start with the required +++ frontmatter delimiter.
	ErrNoFrontmatter = errors.New("file does not start with frontmatter delimiter (+++)")

	// ErrUnclosedFrontmatter is returned when the opening delimiter has no matching closing delimiter.
	ErrUnclosedFrontmatter = errors.New("unclosed frontmatter delimiter (+++)")
)

const (
	// DelimTOML is the required TOML frontmatter delimiter.
	DelimTOML = "+++"
)

// Split extracts the raw frontmatter bytes, the remaining markdown body,
// and the 1-based line number where the body begins.
// Frontmatter MUST begin and end with +++.
func Split(content []byte) (frontmatter []byte, body []byte, delim string, bodyLine int, err error) {
	trimmed := bytes.TrimPrefix(content, []byte("\xef\xbb\xbf")) // Strip UTF-8 BOM if present

	if !bytes.HasPrefix(trimmed, []byte("+++\n")) && !bytes.HasPrefix(trimmed, []byte("+++\r\n")) {
		return nil, nil, "", 0, ErrNoFrontmatter
	}
	delim = DelimTOML
	opening := []byte("+++")

	// Skip opening delimiter and its newline
	rest := trimmed[len(opening):]
	if len(rest) > 0 && rest[0] == '\r' {
		rest = rest[1:]
	}
	if len(rest) > 0 && rest[0] == '\n' {
		rest = rest[1:]
	}

	// Look for closing delimiter on its own line: \n+++ or \n---
	closingPatterns := [][]byte{
		[]byte("\n" + delim + "\n"),
		[]byte("\n" + delim + "\r\n"),
		[]byte("\r\n" + delim + "\r\n"),
		[]byte("\r\n" + delim + "\n"),
	}

	var closeIdx = -1
	var matchLen = 0

	for _, pattern := range closingPatterns {
		if idx := bytes.Index(rest, pattern); idx != -1 {
			if closeIdx == -1 || idx < closeIdx {
				closeIdx = idx
				matchLen = len(pattern)
			}
		}
	}

	// Also handle delimiter at the very end of file without trailing newline
	if closeIdx == -1 {
		endPatternLF := []byte("\n" + delim)
		endPatternCRLF := []byte("\r\n" + delim)
		if bytes.HasSuffix(rest, endPatternLF) {
			closeIdx = len(rest) - len(endPatternLF)
			matchLen = len(endPatternLF)
		} else if bytes.HasSuffix(rest, endPatternCRLF) {
			closeIdx = len(rest) - len(endPatternCRLF)
			matchLen = len(endPatternCRLF)
		}
	}

	if closeIdx == -1 {
		return nil, nil, "", 0, ErrUnclosedFrontmatter
	}

	frontmatter = rest[:closeIdx]
	rawBody := rest[closeIdx+matchLen:]

	// Count lines to determine 1-based start line of body
	// Line 1: opening delimiter
	// Lines 2..(closeLine): frontmatter lines
	// Body starts after closing delimiter
	fmLines := bytes.Count(rest[:closeIdx+matchLen], []byte("\n"))
	bodyLine = 1 + fmLines + 1

	return frontmatter, rawBody, delim, bodyLine, nil
}

// ParseFrontmatter generic helper unmarshals TOML frontmatter into target struct
// and returns the markdown body bytes and delimiter.
func ParseFrontmatter[T any](content []byte, target *T) (body []byte, delim string, bodyLine int, err error) {
	fmBytes, bodyBytes, d, line, err := Split(content)
	if err != nil {
		return nil, "", 0, err
	}

	if err := toml.Unmarshal(fmBytes, target); err != nil {
		return nil, "", 0, fmt.Errorf("failed to parse TOML frontmatter: %w", err)
	}

	return bodyBytes, d, line, nil
}

// NormalizeTimestamp parses an RFC3339, RFC3339Nano, or YYYY-MM-DD date string and formats it as RFC3339 UTC ("YYYY-MM-DDTHH:MM:SSZ").
func NormalizeTimestamp(s string) (string, error) {
	trimmed := strings.TrimSpace(s)
	if trimmed == "" {
		return "", nil
	}
	if t, err := time.Parse(time.RFC3339, trimmed); err == nil {
		return t.UTC().Format(time.RFC3339), nil
	}
	if t, err := time.Parse(time.RFC3339Nano, trimmed); err == nil {
		return t.UTC().Format(time.RFC3339), nil
	}
	if t, err := time.Parse(time.DateOnly, trimmed); err == nil {
		return t.UTC().Format(time.RFC3339), nil
	}
	if t, err := time.Parse("2006-01-02 15:04:05", trimmed); err == nil {
		return t.UTC().Format(time.RFC3339), nil
	}
	return "", fmt.Errorf("invalid timestamp format: %q (expected RFC3339 or YYYY-MM-DD)", trimmed)
}

// DeriveFallbackCreatedAt derives a backward-compatible created_at timestamp
// from the task ID's YYMMDD prefix, falling back to mtime or now if unavailable.
func DeriveFallbackCreatedAt(id string, modTime time.Time) string {
	if len(id) >= 6 {
		if t, err := time.Parse("060102", id[:6]); err == nil {
			return t.UTC().Format(time.RFC3339)
		}
	}
	if !modTime.IsZero() {
		return modTime.UTC().Format(time.RFC3339)
	}
	return time.Now().UTC().Format(time.RFC3339)
}

// DeriveFallbackChangedAt derives a backward-compatible changed_at timestamp
// from filesystem mtime, falling back to created_at or now.
func DeriveFallbackChangedAt(modTime time.Time, createdAt string) string {
	if !modTime.IsZero() {
		return modTime.UTC().Format(time.RFC3339)
	}
	if createdAt != "" {
		return createdAt
	}
	return time.Now().UTC().Format(time.RFC3339)
}

// ParseTask parses a complete task markdown document into a model.Task entity.
func ParseTask(content []byte, id string) (*model.Task, error) {
	var fm model.TaskFrontmatter
	body, _, _, err := ParseFrontmatter(content, &fm)
	if err != nil {
		return nil, err
	}

	if strings.TrimSpace(fm.Title) == "" {
		return nil, errors.New("task frontmatter missing required field: title")
	}
	if strings.TrimSpace(fm.Summary) == "" {
		return nil, errors.New("task frontmatter missing required field: summary")
	}

	priority := fm.Priority
	if priority == "" {
		priority = model.PriorityMedium
	}

	tags := fm.Tags
	if tags == nil {
		tags = []string{}
	}

	deps := fm.Dependencies
	if deps == nil {
		deps = []string{}
	}

	bodyStr := strings.TrimSpace(string(body))
	bodyHTML, _ := RenderHTML([]byte(bodyStr))

	createdAt := strings.TrimSpace(fm.CreatedAt)
	if createdAt != "" {
		if normalized, err := NormalizeTimestamp(createdAt); err == nil {
			createdAt = normalized
		}
	} else {
		createdAt = DeriveFallbackCreatedAt(id, time.Time{})
	}

	changedAt := strings.TrimSpace(fm.ChangedAt)
	if changedAt != "" {
		if normalized, err := NormalizeTimestamp(changedAt); err == nil {
			changedAt = normalized
		}
	} else {
		changedAt = createdAt
	}

	targetAt := strings.TrimSpace(fm.TargetAt)
	if targetAt != "" {
		if normalized, err := NormalizeTimestamp(targetAt); err == nil {
			targetAt = normalized
		}
	}

	return &model.Task{
		ID:           id,
		Title:        strings.TrimSpace(fm.Title),
		Status:       strings.TrimSpace(fm.Status),
		Priority:     priority,
		Milestone:    strings.TrimSpace(fm.Milestone),
		Tags:         tags,
		Summary:      strings.TrimSpace(fm.Summary),
		Dependencies: deps,
		Body:         bodyStr,
		BodyHTML:     bodyHTML,
		CreatedAt:    createdAt,
		ChangedAt:    changedAt,
		TargetAt:     targetAt,
	}, nil
}

// ParseMilestone parses a milestone document into a model.Milestone entity.
func ParseMilestone(content []byte, id string) (*model.Milestone, error) {
	var fm model.MilestoneFrontmatter
	body, _, _, err := ParseFrontmatter(content, &fm)
	if err != nil {
		return nil, err
	}

	if strings.TrimSpace(fm.Title) == "" {
		return nil, errors.New("milestone frontmatter missing required field: title")
	}
	if strings.TrimSpace(fm.Summary) == "" {
		return nil, errors.New("milestone frontmatter missing required field: summary")
	}

	status := fm.Status
	if status == "" {
		status = model.MilestoneStatusOpen
	}

	tags := fm.Tags
	if tags == nil {
		tags = []string{}
	}

	bodyStr := strings.TrimSpace(string(body))
	bodyHTML, _ := RenderHTML([]byte(bodyStr))

	return &model.Milestone{
		ID:         id,
		Title:      strings.TrimSpace(fm.Title),
		Status:     status,
		TargetDate: strings.TrimSpace(fm.TargetDate),
		Tags:       tags,
		Summary:    strings.TrimSpace(fm.Summary),
		Body:       bodyStr,
		BodyHTML:   bodyHTML,
	}, nil
}

// ParseStrategy parses an architectural strategy document into a model.Strategy entity.
func ParseStrategy(content []byte, id string) (*model.Strategy, error) {
	var fm model.StrategyFrontmatter
	body, _, _, err := ParseFrontmatter(content, &fm)
	if err != nil {
		return nil, err
	}

	if strings.TrimSpace(fm.Title) == "" {
		return nil, errors.New("strategy frontmatter missing required field: title")
	}
	if strings.TrimSpace(fm.Summary) == "" {
		return nil, errors.New("strategy frontmatter missing required field: summary")
	}
	if !fm.Tier.IsValid() {
		return nil, fmt.Errorf("strategy frontmatter has invalid tier %d; expected 1, 2, or 3", fm.Tier)
	}

	tags := fm.Tags
	if tags == nil {
		tags = []string{}
	}

	bodyStr := strings.TrimSpace(string(body))
	bodyHTML, _ := RenderHTML([]byte(bodyStr))

	return &model.Strategy{
		ID:      id,
		Title:   strings.TrimSpace(fm.Title),
		Tier:    fm.Tier,
		Tags:    tags,
		Summary: strings.TrimSpace(fm.Summary),
		Body:    bodyStr,
		BodyHTML: bodyHTML,
	}, nil
}

// ParseGlossaryTerm parses a glossary term document into a model.GlossaryTerm entity.
func ParseGlossaryTerm(content []byte, id string) (*model.GlossaryTerm, error) {
	var fm model.GlossaryFrontmatter
	body, _, _, err := ParseFrontmatter(content, &fm)
	if err != nil {
		return nil, err
	}

	if strings.TrimSpace(fm.Title) == "" {
		return nil, errors.New("glossary frontmatter missing required field: title")
	}
	if strings.TrimSpace(fm.Summary) == "" {
		return nil, errors.New("glossary frontmatter missing required field: summary")
	}

	tags := fm.Tags
	if tags == nil {
		tags = []string{}
	}

	bodyStr := strings.TrimSpace(string(body))
	bodyHTML, _ := RenderHTML([]byte(bodyStr))

	return &model.GlossaryTerm{
		ID:      id,
		Title:   strings.TrimSpace(fm.Title),
		Tags:    tags,
		Summary: strings.TrimSpace(fm.Summary),
		Body:    bodyStr,
		BodyHTML: bodyHTML,
	}, nil
}

// Format serializes any frontmatter struct to TOML and rejoins it with the markdown body
// using the standard +++ delimiter.
func Format[T any](frontmatter T, body string) ([]byte, error) {
	delim := DelimTOML

	fmBytes, err := toml.Marshal(frontmatter)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal frontmatter to TOML: %w", err)
	}

	var buf bytes.Buffer
	buf.WriteString(delim)
	buf.WriteByte('\n')
	buf.Write(fmBytes)
	buf.WriteString(delim)
	buf.WriteByte('\n')

	trimmedBody := strings.TrimSpace(body)
	if trimmedBody != "" {
		buf.WriteByte('\n')
		buf.WriteString(trimmedBody)
		buf.WriteByte('\n')
	}

	return buf.Bytes(), nil
}
