package api

import (
	"errors"
	"net/http"
	"strconv"
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
// @Param        klasse    query   string  false  "School class to filter by"
// @Param        jahrgang  query   string  false  "Jahrgang (1–13); filtert über alle Klassen dieses Jahrgangs, Oberstufe eingeschlossen"
// @Param        q       query     string  false  "Search term (name or barcode); searches server-side over all students"
// @Param        status  query     string  false  "Leer = aktive Schüler; 'ehemalige' = wer die Schule verlassen hat (ist_abgaenger)"
// @Param        art     query     string  false  "Leer = nur Schüler; 'alle' = die Leserdatei (Schüler und Kollegium)"
// @Param        sortierung query  string  false  "Spalte: name | klasse | ausgeliehen; leer = Reihenfolge der Kartei"
// @Param        richtung   query  string  false  "auf | ab (Vorgabe: auf)"
// @Success      200     {array}   repository.StudentListStat
// @Failure      500     {object}  map[string]string
// @Router       /schueler [get]
// lmfRepo liefert die Klassen für den Jahrgangsfilter. Als PARAMETER, nicht über s.DB:
// Die Handler dieses Pakets bekommen ihre Repositories von außen, und ein Griff nach
// s.DB.Pool sprengt jeden Test, der den Server ohne Datenbank baut — genau daran ist
// TestListStudentsLeereListeIstArray am 17.09.2026 mit einem nil-Zeiger abgestürzt.
func (s *Server) ListStudentsHandler(studentRepo repository.StudentRepository, lmfRepo *repository.LmfTerminRepository) http.HandlerFunc {
	return apierrors.Wrap(func(w http.ResponseWriter, r *http.Request) error {
		// Klasse ODER Jahrgang — die Klasse gewinnt, wenn beides kommt (sie ist die
		// genauere Angabe). Die Übersetzung steht in schueler_jahrgang_filter.go.
		klassen, ohneTreffer, filterErr := klassenFilter(r.Context(), lmfRepo,
			r.URL.Query().Get("klasse"), r.URL.Query().Get("jahrgang"))
		if filterErr != nil {
			return apierrors.Internal("Klassen des Jahrgangs konnten nicht gelesen werden", filterErr)
		}
		if ohneTreffer {
			// Zu diesem Jahrgang gibt es keine Klasse. Eine leere Liste ist die richtige
			// Antwort; ohne diesen Zweig stünde hier die ganze Kartei.
			RespondJSON(w, http.StatusOK, []repository.StudentListStat{})
			return nil
		}
		// q sucht auf dem Server. Ohne das filterte die Schülerdatei im Browser über die
		// gelieferten (gekappten) Zeilen und fand niemanden dahinter.
		suche := strings.TrimSpace(r.URL.Query().Get("q"))

		sortierung, sortErr := leserSortierung(r)
		if sortErr != nil {
			return sortErr
		}

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
			students, err = studentRepo.ListEhemaligeWithStats(r.Context(), suche, sortierung)
		case r.URL.Query().Get("art") == "alle":
			students, err = studentRepo.ListLeserMitStats(r.Context(), klassen, suche, sortierung)
		default:
			students, err = studentRepo.ListStudentsWithStats(r.Context(), klassen, suche, sortierung)
		}
		if err != nil {
			return apierrors.Internal("Fehler beim Abrufen der Leserliste", err)
		}

		RespondJSON(w, http.StatusOK, students)
		return nil
	})
}

// leserSortierung liest Spalte und Richtung aus der Anfrage.
//
// Ein unbekannter Spaltenname ist ein 400 und KEINE stille Vorgabe. Der stille
// Ersatzwert ist in diesem Projekt schon zweimal teuer geworden (zuletzt bei den
// Einstellungen, Rasterdurchgang 16.09.2026): Die Liste käme sortiert nach irgendetwas
// zurück, die Oberfläche zeigte ihren Pfeil an der geklickten Spalte, und niemand würde
// je erfahren, dass beides nicht zusammenpasst. Die Meldung nennt die erlaubten Werte.
func leserSortierung(r *http.Request) (repository.SchuelerSortierung, *apierrors.APIError) {
	spalte := repository.SchuelerSortierspalte(strings.TrimSpace(r.URL.Query().Get("sortierung")))
	richtung := strings.TrimSpace(r.URL.Query().Get("richtung"))

	s := repository.SchuelerSortierung{Spalte: spalte, Absteigend: richtung == "ab"}
	if !s.Erlaubt() {
		erlaubt := make([]string, 0, len(repository.SchuelerSortierspalten))
		for _, e := range repository.SchuelerSortierspalten {
			erlaubt = append(erlaubt, string(e))
		}
		return repository.SchuelerSortierung{}, apierrors.BadRequest(
			"Unbekannte Sortierung "+strconv.Quote(string(spalte))+" — erlaubt sind: "+
				strings.Join(erlaubt, ", ")+" (oder leer für die Reihenfolge der Kartei).",
			errors.New("unbekannte sortierspalte"))
	}
	// „ab" oder leer/„auf" — ein dritter Wert ist ein Tippfehler des Aufrufers und
	// bekommt dieselbe Antwort wie eine unbekannte Spalte.
	if richtung != "" && richtung != "auf" && richtung != "ab" {
		return repository.SchuelerSortierung{}, apierrors.BadRequest(
			"Unbekannte Richtung "+strconv.Quote(richtung)+" — erlaubt sind: auf, ab.",
			errors.New("unbekannte sortierrichtung"))
	}
	return s, nil
}
