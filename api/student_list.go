package api

import (
	"net/http"
	"strings"

	"bibliothek/apierrors"
	"bibliothek/repository"
)

// ListStudentsHandler returns all students, optionally filtered by klasse.
// @Summary      List students
// @Description  Retrieves students, optionally filtered by a specific school class or a search term, along with loan counts.
// @Tags         students
// @Accept       json
// @Produce      json
// @Param        klasse  query     string  false  "School class to filter by"
// @Param        q       query     string  false  "Search term (name or barcode); searches server-side over all students"
// @Param        status  query     string  false  "Leer = aktive Schüler; 'ehemalige' = wer die Schule verlassen hat (ist_abgaenger)"
// @Success      200     {array}   repository.StudentListStat
// @Failure      500     {object}  map[string]string
// @Router       /schueler [get]
func (s *Server) ListStudentsHandler(studentRepo repository.StudentRepository) http.HandlerFunc {
	return apierrors.Wrap(func(w http.ResponseWriter, r *http.Request) error {
		klasse := r.URL.Query().Get("klasse")
		// q sucht auf dem Server. Ohne das filterte die Schülerdatei im Browser über die
		// gelieferten (gekappten) Zeilen und fand niemanden dahinter.
		suche := strings.TrimSpace(r.URL.Query().Get("q"))

		// status=ehemalige: der Reiter „Ehemalige / Archiv" — dieselbe Liste mit
		// umgekehrtem Vorzeichen (ist_abgaenger), kein eigener Endpunkt.
		var students []repository.StudentListStat
		var err error
		if r.URL.Query().Get("status") == "ehemalige" {
			students, err = studentRepo.ListEhemaligeWithStats(r.Context(), suche)
		} else {
			students, err = studentRepo.ListStudentsWithStats(r.Context(), klasse, suche)
		}
		if err != nil {
			return apierrors.Internal("Fehler beim Abrufen der Schülerliste", err)
		}

		RespondJSON(w, http.StatusOK, students)
		return nil
	})
}
