package server

import (
	"net/http"

	"github.com/RJuho/jokateko/internal/model"
)

func (s *Server) handleGetBoard(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	filter := model.FilterCriteria{
		Status:      q.Get("status"),
		Milestone:   q.Get("milestone"),
		Tag:         q.Get("tag"),
		Priority:    model.Priority(q.Get("priority")),
		SearchQuery: q.Get("q"),
	}

	board, err := s.store.GetBoardState(r.Context(), s.cfg.Project.Name, s.columns, filter)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to get board state: "+err.Error())
		return
	}

	writeJSON(w, http.StatusOK, board)
}
