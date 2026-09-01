package server

import (
	"net/http"

	"github.com/RJuho/jokateko/internal/model"
)

func (s *Server) handleGetTags(w http.ResponseWriter, r *http.Request) {
	tagCounts, err := s.store.GetTagCounts(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to get tag counts: "+err.Error())
		return
	}

	resp := model.TagList{
		Enforced: s.cfg.Tags.EnforceAllowed,
		Tags:     tagCounts,
	}

	writeJSON(w, http.StatusOK, resp)
}
