package inventur

import (
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// Lehrerportal → Schulbücher je Fach (Peter, 03.09.2026). Beide Türen liegen hinter der
// Anmeldung, ohne view_books (wie /api/portal/klassensaetze): Sie liefern nur Buch- und
// Zähldaten, keine Ausleih- oder Personendaten.

// handlePortalLernmittel bedient GET /api/portal/lernmittel[?fach=…]: immer die
// Fach-Zahlen und die Titel — ohne ?fach alle Schulbücher (Reiter-Startansicht seit
// 03.09.2026 abends: Liste statt Kachelwand), mit ?fach= die des Fachs ("" = ohne Fach).
func (handler *APIHandler) handlePortalLernmittel(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	filter := lernmittelFilterParam(r)
	faecher, err := handler.repo.GetLernmittelFaecher(ctx, filter)
	if err != nil {
		log.Printf("Portal Lernmittel: Fächer: %v", err)
		writeError(w, http.StatusInternalServerError, "schulbücher konnten nicht geladen werden")
		return
	}
	fach, gewaehlt := r.URL.Query()["fach"]
	fachName := ""
	if gewaehlt {
		fachName = strings.TrimSpace(fach[0])
	}
	titel, err := handler.repo.GetLernmittelTitel(ctx, fachName, !gewaehlt, filter)
	if err != nil {
		log.Printf("Portal Lernmittel: Titel: %v", err)
		writeError(w, http.StatusInternalServerError, "schulbücher konnten nicht geladen werden")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"faecher": faecher, "titel": titel})
}

// handlePortalLernmittelExport bedient GET /api/portal/lernmittel/export[?fach=…] und
// liefert eine Excel-Datei (ein Blatt, eine Zeile je Titel). Ohne ?fach alle Schulbücher.
func (handler *APIHandler) handlePortalLernmittelExport(w http.ResponseWriter, r *http.Request) {
	fach, gewaehlt := r.URL.Query()["fach"]
	fachName := ""
	if gewaehlt {
		fachName = strings.TrimSpace(fach[0])
	}
	titel, err := handler.repo.GetLernmittelTitel(r.Context(), fachName, !gewaehlt, lernmittelFilterParam(r))
	if err != nil {
		log.Printf("Portal Lernmittel-Export: %v", err)
		writeError(w, http.StatusInternalServerError, "export konnte nicht erstellt werden")
		return
	}
	datei, err := SchulbuecherAlsExcel(titel, coverBasis(r))
	if err != nil {
		log.Printf("Portal Lernmittel-Export: Excel: %v", err)
		writeError(w, http.StatusInternalServerError, "export konnte nicht erstellt werden")
		return
	}
	defer func() {
		if err := datei.Close(); err != nil {
			log.Printf("Portal Lernmittel-Export: Datei schließen: %v", err)
		}
	}()
	name := "schulbuecher"
	if gewaehlt {
		name += "_" + dateinamenTeil(fachName)
	}
	w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s_%s.xlsx"`, name, time.Now().Format("2006-01-02")))
	if err := datei.Write(w); err != nil {
		log.Printf("Portal Lernmittel-Export: schreiben: %v", err)
	}
}

// lernmittelFilterParam liest ?jahrgang= (5–13), ?zweig= und ?q=; alles andere heißt
// „alle". Der Suchtext wird gekappt: Er geht als ILIKE-Muster in die Abfrage.
func lernmittelFilterParam(r *http.Request) LernmittelFilter {
	q := r.URL.Query()
	f := LernmittelFilter{Zweig: strings.TrimSpace(q.Get("zweig")), Suche: strings.TrimSpace(q.Get("q"))}
	if len(f.Suche) > 100 {
		f.Suche = f.Suche[:100]
	}
	if j, err := strconv.Atoi(q.Get("jahrgang")); err == nil && j >= 5 && j <= 13 {
		f.Jahrgang = j
	}
	return f
}
