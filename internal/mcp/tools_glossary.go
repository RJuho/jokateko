package mcp

import (
	"context"
	"fmt"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type LookupGlossaryInput struct {
	Term string `json:"term,omitempty" jsonschema:"Specific term or slug to find. If omitted, returns entire glossary"`
}

type GlossaryEntry struct {
	ID      string   `json:"id"`
	Title   string   `json:"title"`
	Tags    []string `json:"tags"`
	Summary string   `json:"summary"`
	Body    string   `json:"body,omitempty"`
}

func (s *Server) registerGlossaryTools() {
	// 1. lookup_glossary
	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name:        "lookup_glossary",
		Description: "Looks up standardized project terms, definitions, and domain concepts.",
	}, s.toolLookupGlossary)
}

func (s *Server) toolLookupGlossary(ctx context.Context, _ *mcp.CallToolRequest, in LookupGlossaryInput) (*mcp.CallToolResult, []GlossaryEntry, error) {
	allTerms, err := s.store.ListGlossaryTerms(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to list glossary terms: %w", err)
	}

	search := strings.ToLower(strings.TrimSpace(in.Term))
	var results []GlossaryEntry

	for _, t := range allTerms {
		if search == "" || strings.ToLower(t.ID) == search || strings.Contains(strings.ToLower(t.Title), search) {
			results = append(results, GlossaryEntry{
				ID:      t.ID,
				Title:   t.Title,
				Tags:    t.Tags,
				Summary: t.Summary,
				Body:    t.Body,
			})
		}
	}

	if len(results) == 0 && search != "" {
		// Fallback to FTS glossary search if exact match not found
		ftsResults, err := s.store.SearchGlossary(ctx, in.Term, "", 10)
		if err == nil && len(ftsResults) > 0 {
			for _, r := range ftsResults {
				term, err := s.store.GetGlossaryTerm(ctx, r.ID)
				if err == nil {
					results = append(results, GlossaryEntry{
						ID:      term.ID,
						Title:   term.Title,
						Tags:    term.Tags,
						Summary: term.Summary,
						Body:    term.Body,
					})
				}
			}
		}
	}

	return nil, results, nil
}
