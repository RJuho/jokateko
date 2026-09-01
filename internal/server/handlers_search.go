package server

import (
	"net/http"
	"strconv"

	"github.com/RJuho/jokateko/internal/model"
)

func (s *Server) handleSearch(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query().Get("q")
	tag := r.URL.Query().Get("tag")
	entityType := r.URL.Query().Get("type")
	limitStr := r.URL.Query().Get("limit")

	limit := 20
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}
	}

	var (
		results []model.SearchResult
		err     error
	)

	ctx := r.Context()
	switch entityType {
	case "task":
		results, err = s.store.SearchTasks(ctx, q, tag, limit)
	case "milestone":
		results, err = s.store.SearchMilestones(ctx, q, tag, limit)
	case "strategy":
		results, err = s.store.SearchStrategies(ctx, q, tag, limit)
	case "glossary":
		results, err = s.store.SearchGlossary(ctx, q, tag, limit)
	default:
		results, err = s.store.SearchAll(ctx, q, tag, limit)
	}

	if err != nil {
		writeError(w, http.StatusInternalServerError, "search failed: "+err.Error())
		return
	}

	if results == nil {
		results = []model.SearchResult{}
	}

	writeJSON(w, http.StatusOK, results)
}
