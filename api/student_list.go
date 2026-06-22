package api

import (
	"net/http"

	"bibliothek/apierrors"
	"bibliothek/repository"
)

// ListStudentsHandler returns all students, optionally filtered by klasse.
// @Summary      List students
// @Description  Retrieves students, optionally filtered by a specific school class, along with loan counts.
// @Tags         students
// @Accept       json
// @Produce      json
// @Param        klasse  query     string  false  "School class to filter by"
// @Success      200     {array}   repository.StudentListStat
// @Failure      500     {object}  map[string]string
// @Router       /schueler [get]
func (s *Server) ListStudentsHandler(studentRepo repository.StudentRepository) http.HandlerFunc {
	return apierrors.Wrap(func(w http.ResponseWriter, r *http.Request) error {
		klasse := r.URL.Query().Get("klasse")

		students, err := studentRepo.ListStudentsWithStats(r.Context(), klasse)
		if err != nil {
			return apierrors.Internal("Fehler beim Abrufen der Schülerliste", err)
		}

		RespondJSON(w, http.StatusOK, students)
		return nil
	})
}
