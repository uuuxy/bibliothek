package inventur

import (
	"net/http"
)

type Subject struct {
	ID       int    `json:"id"`
	Name     string `json:"name"`
	IsActive bool   `json:"isActive"`
}

func (handler *APIHandler) handleGetSubjects(writer http.ResponseWriter, request *http.Request) {
	subjects, err := handler.repo.GetActiveSubjects(request.Context())
	if err != nil {
		writeError(writer, http.StatusInternalServerError, "Fächer konnten nicht geladen werden")
		return
	}

	writeJSON(writer, http.StatusOK, map[string]any{"data": subjects})
}
