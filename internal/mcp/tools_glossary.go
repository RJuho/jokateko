package mcp

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/RJuho/jokateko/internal/model"
	"github.com/RJuho/jokateko/internal/parser"
	"github.com/RJuho/jokateko/internal/store"
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

func (s *Server) toolCreateGlossaryTerm(ctx context.Context, _ *mcp.CallToolRequest, in CreateGlossaryTermInput) (*mcp.CallToolResult, *GlossaryEntry, error) {
	title := strings.TrimSpace(in.Title)
	if title == "" {
		return nil, nil, errors.New("title is required")
	}

	summary := strings.TrimSpace(in.Summary)
	if summary == "" {
		return nil, nil, errors.New("summary is required")
	}

	if s.cfg.Tags.EnforceAllowed && len(in.Tags) > 0 {
		for _, tag := range in.Tags {
			if !slices.Contains(s.cfg.Tags.Allowed, tag) {
				return nil, nil, fmt.Errorf("tag %q is not permitted. Allowed tags: %v", tag, s.cfg.Tags.Allowed)
			}
		}
	}

	tags := in.Tags
	if tags == nil {
		tags = []string{}
	}

	id := slugify(title)
	fm := model.GlossaryFrontmatter{
		Title:   title,
		Summary: summary,
		Tags:    tags,
	}

	fileBytes, err := parser.Format(fm, in.Body)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to format glossary markdown: %w", err)
	}

	filePath := filepath.Join(s.GlossaryDir(), fmt.Sprintf("%s.md", id))
	if err := s.writer.WriteFile(filePath, fileBytes, 0644); err != nil {
		return nil, nil, fmt.Errorf("failed to save glossary file: %w", err)
	}

	term := model.GlossaryTerm{
		ID:       id,
		Title:    title,
		Summary:  summary,
		Tags:     tags,
		Body:     in.Body,
		FilePath: filePath,
		ModTime:  time.Now(),
	}

	_ = s.store.UpsertGlossaryTerm(ctx, term)

	return nil, &GlossaryEntry{
		ID:      term.ID,
		Title:   term.Title,
		Tags:    term.Tags,
		Summary: term.Summary,
		Body:    term.Body,
	}, nil
}

func (s *Server) toolUpdateGlossaryTerm(ctx context.Context, _ *mcp.CallToolRequest, in UpdateGlossaryTermInput) (*mcp.CallToolResult, *GlossaryEntry, error) {
	id := strings.TrimSpace(in.ID)
	if id == "" {
		return nil, nil, errors.New("term id is required")
	}

	existing, err := s.store.GetGlossaryTerm(ctx, id)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return nil, nil, fmt.Errorf("glossary term %q not found", id)
		}
		return nil, nil, fmt.Errorf("failed to get glossary term %q: %w", id, err)
	}

	title := existing.Title
	if strings.TrimSpace(in.Title) != "" {
		title = strings.TrimSpace(in.Title)
	}

	summary := existing.Summary
	if strings.TrimSpace(in.Summary) != "" {
		summary = strings.TrimSpace(in.Summary)
	}

	tags := existing.Tags
	if in.Tags != nil {
		if s.cfg.Tags.EnforceAllowed && len(in.Tags) > 0 {
			for _, tag := range in.Tags {
				if !slices.Contains(s.cfg.Tags.Allowed, tag) {
					return nil, nil, fmt.Errorf("tag %q is not permitted. Allowed tags: %v", tag, s.cfg.Tags.Allowed)
				}
			}
		}
		tags = in.Tags
	}

	body := existing.Body
	if in.Body != "" {
		body = in.Body
	}

	fm := model.GlossaryFrontmatter{
		Title:   title,
		Summary: summary,
		Tags:    tags,
	}

	fileBytes, err := parser.Format(fm, body)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to format glossary markdown: %w", err)
	}

	filePath := existing.FilePath
	if filePath == "" {
		filePath = filepath.Join(s.GlossaryDir(), fmt.Sprintf("%s.md", id))
	}

	if err := s.writer.WriteFile(filePath, fileBytes, 0644); err != nil {
		return nil, nil, fmt.Errorf("failed to save glossary file: %w", err)
	}

	updated := model.GlossaryTerm{
		ID:       id,
		Title:    title,
		Summary:  summary,
		Tags:     tags,
		Body:     body,
		FilePath: filePath,
		ModTime:  time.Now(),
	}

	_ = s.store.UpsertGlossaryTerm(ctx, updated)

	return nil, &GlossaryEntry{
		ID:      updated.ID,
		Title:   updated.Title,
		Tags:    updated.Tags,
		Summary: updated.Summary,
		Body:    updated.Body,
	}, nil
}

func (s *Server) toolDeleteGlossaryTerm(ctx context.Context, _ *mcp.CallToolRequest, in DeleteGlossaryTermInput) (*mcp.CallToolResult, *DeleteEntityOutput, error) {
	if !s.cfg.MCP.AllowMutations {
		return nil, nil, errors.New("mutations are disabled in configuration")
	}

	id := strings.TrimSpace(in.ID)
	if id == "" {
		return nil, nil, errors.New("glossary term id is required")
	}

	term, err := s.store.GetGlossaryTerm(ctx, id)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return nil, nil, fmt.Errorf("glossary term %q not found", id)
		}
		return nil, nil, fmt.Errorf("failed to get glossary term %q: %w", id, err)
	}

	if term.FilePath != "" {
		if err := s.writer.RemoveFile(term.FilePath); err != nil {
			return nil, nil, fmt.Errorf("failed to remove glossary term file: %w", err)
		}
	}

	if err := s.store.DeleteGlossaryTerm(ctx, id); err != nil {
		return nil, nil, fmt.Errorf("failed to delete glossary term from store: %w", err)
	}

	return nil, &DeleteEntityOutput{
		Success: true,
		ID:      id,
		Message: fmt.Sprintf("Glossary term %q deleted successfully", id),
	}, nil
}

