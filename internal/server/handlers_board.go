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

	editable := s.cfg.Board.EditableStates
	if len(editable) == 0 {
		editable = []string{"backlog"}
	}
	board.EditableStates = editable

	creatable := s.cfg.Board.CreatableStates
	if len(creatable) == 0 {
		creatable = []string{"backlog"}
	}
	board.CreatableStates = creatable

	defaultCreate := s.cfg.Board.DefaultCreateState
	if defaultCreate == "" {
		defaultCreate = "backlog"
	}
	board.DefaultCreateState = defaultCreate
	board.MCPInstructions = s.cfg.MCP.Instructions
	board.Locale = s.cfg.Project.Locale
	board.Translations = s.cfg.Translations

	writeJSON(w, http.StatusOK, board)
}
