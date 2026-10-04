package mcp

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/RJuho/jokateko/internal/model"
	"github.com/RJuho/jokateko/internal/service"
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

type CreateGlossaryTermInput struct {
	Title   string   `json:"title" jsonschema:"required,Term title or name"`
	Summary string   `json:"summary" jsonschema:"required,Short definition of the term"`
	Tags    []string `json:"tags,omitempty" jsonschema:"Categorization tags"`
	Body    string   `json:"body,omitempty" jsonschema:"Extended markdown description, examples, or notes"`
}

type UpdateGlossaryTermInput struct {
	ID      string   `json:"id" jsonschema:"required,Glossary term slug (e.g. tasks-as-code)"`
	Title   string   `json:"title,omitempty" jsonschema:"Updated term title"`
	Summary string   `json:"summary,omitempty" jsonschema:"Updated short definition"`
	Tags    []string `json:"tags,omitempty" jsonschema:"Updated categorization tags"`
	Body    string   `json:"body,omitempty" jsonschema:"Updated extended markdown description"`
}

type DeleteGlossaryTermInput struct {
	ID string `json:"id" jsonschema:"required,Glossary term slug (e.g. tasks-as-code)"`
}

func (s *Server) registerGlossaryTools() {
	// 1. lookup_glossary
	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name:        "lookup_glossary",
		Description: "Looks up standardized project terms, definitions, and domain concepts.",
	}, s.toolLookupGlossary)

	// 2. create_glossary_term
	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name:        "create_glossary_term",
		Description: "Creates a new project glossary term markdown file inside .jokateko/glossary/.",
	}, s.toolCreateGlossaryTerm)

	// 3. update_glossary_term
	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name:        "update_glossary_term",
		Description: "Updates an existing glossary term's definition, summary, tags, or extended body.",
	}, s.toolUpdateGlossaryTerm)

	// 4. delete_glossary_term
	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name:        "delete_glossary_term",
		Description: "Deletes a glossary term file and removes it from the store.",
	}, s.toolDeleteGlossaryTerm)
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
			results = append(results, toGlossaryEntry(t))
		}
	}

	if len(results) == 0 && search != "" {
		// Fallback to FTS glossary search if exact match not found
		ftsResults, err := s.store.SearchGlossary(ctx, in.Term, "", 10)
		if err == nil && len(ftsResults) > 0 {
			for _, r := range ftsResults {
				term, err := s.store.GetGlossaryTerm(ctx, r.ID)
				if err == nil {
					results = append(results, toGlossaryEntry(term))
				}
			}
		}
	}

	return nil, results, nil
}

func toGlossaryEntry(t model.GlossaryTerm) GlossaryEntry {
	return GlossaryEntry{
		ID:      t.ID,
		Title:   t.Title,
		Tags:    t.Tags,
		Summary: t.Summary,
		Body:    t.Body,
	}
}

func (s *Server) toolCreateGlossaryTerm(ctx context.Context, _ *mcp.CallToolRequest, in CreateGlossaryTermInput) (*mcp.CallToolResult, *GlossaryEntry, error) {
	if err := s.checkMutations(); err != nil {
		return nil, nil, err
	}
	if strings.TrimSpace(in.Summary) == "" {
		return nil, nil, errors.New("summary is required")
	}
	if err := s.svc.CheckTags(in.Tags); err != nil {
		return nil, nil, err
	}

	term, err := s.svc.CreateGlossaryTerm(ctx, service.NewGlossaryTerm{
		Title:   in.Title,
		Tags:    in.Tags,
		Summary: in.Summary,
		Body:    in.Body,
	})
	if err != nil {
		return nil, nil, err
	}
	entry := toGlossaryEntry(term)
	return nil, &entry, nil
}

func (s *Server) toolUpdateGlossaryTerm(ctx context.Context, _ *mcp.CallToolRequest, in UpdateGlossaryTermInput) (*mcp.CallToolResult, *GlossaryEntry, error) {
	if err := s.checkMutations(); err != nil {
		return nil, nil, err
	}
	id, err := requireID(in.ID, "term")
	if err != nil {
		return nil, nil, err
	}
	if in.Tags != nil {
		if err := s.svc.CheckTags(in.Tags); err != nil {
			return nil, nil, err
		}
	}

	term, err := s.svc.UpdateGlossaryTerm(ctx, id, func(t *model.GlossaryTerm) error {
		if v := strings.TrimSpace(in.Title); v != "" {
			t.Title = v
		}
		if v := strings.TrimSpace(in.Summary); v != "" {
			t.Summary = v
		}
		if in.Tags != nil {
			t.Tags = in.Tags
		}
		if in.Body != "" {
			t.Body = in.Body
		}
		return nil
	})
	if err != nil {
		return nil, nil, err
	}
	entry := toGlossaryEntry(term)
	return nil, &entry, nil
}

func (s *Server) toolDeleteGlossaryTerm(ctx context.Context, _ *mcp.CallToolRequest, in DeleteGlossaryTermInput) (*mcp.CallToolResult, *DeleteEntityOutput, error) {
	if err := s.checkMutations(); err != nil {
		return nil, nil, err
	}
	id, err := requireID(in.ID, "glossary term")
	if err != nil {
		return nil, nil, err
	}
	if err := s.svc.DeleteGlossaryTerm(ctx, id); err != nil {
		return nil, nil, err
	}
	return nil, &DeleteEntityOutput{
		Success: true,
		ID:      id,
		Message: fmt.Sprintf("Glossary term %q deleted successfully", id),
	}, nil
}
