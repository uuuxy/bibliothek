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
// @Param        art     query     string  false  "Leer = nur Schüler; 'alle' = die Leserdatei (Schüler und Kollegium)"
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
		//
		// art=alle: die Leserdatei. Muss AUSDRÜCKLICH angefragt werden, denn an dieser
		// Tür hängen drei Ansichten mit verschiedenen Absichten — die Leserdatei, der
		// Reiter „Ehemalige" und die Schülersuche des Vormerkungs-Reiters. Für die
		// beiden letzten wäre ein Kollege in der Liste falsch: Eine Vormerkung schickt
		// eine Nachricht an die Elternadresse, und die hat er nicht. Deshalb bleibt die
		// Vorgabe „nur Schüler", und wer alle meint, sagt es.
		var students []repository.StudentListStat
		var err error
		switch {
		case r.URL.Query().Get("status") == "ehemalige":
			students, err = studentRepo.ListEhemaligeWithStats(r.Context(), suche)
		case r.URL.Query().Get("art") == "alle":
			students, err = studentRepo.ListLeserMitStats(r.Context(), klasse, suche)
		default:
			students, err = studentRepo.ListStudentsWithStats(r.Context(), klasse, suche)
		}
		if err != nil {
			return apierrors.Internal("Fehler beim Abrufen der Leserliste", err)
		}

		RespondJSON(w, http.StatusOK, students)
		return nil
	})
}
