package inventur

import (
	"net/http"
)

// TitelVerwaltung ist der zentrale Router für admin-geschützte Buch-Operationen.
// Routen mit Buch-Kennung im Pfad stehen nicht hier, sondern mit Platzhalter in
// api_routen.go.
func (t *Tueren) TitelVerwaltung(w http.ResponseWriter, r *http.Request) {
	pfad := r.URL.Path

	switch r.Method {
	case http.MethodGet:
		switch pfad {
		case routeKlassensaetze:
			t.Klassensaetze(w, r)
		case "/api/admin/books/external-covers":
			t.FremdeCoverListe(w, r)
		case "/api/admin/books/export":
			t.CSVExport(w, r)
		default:
			schreibeFehler(w, http.StatusNotFound, meldungUnbekannteRoute)
		}
	case http.MethodPost:
		switch pfad {
		case routeKlassensaetze:
			t.KlassensaetzeAendern(w, r)
		case "/api/admin/class-books/add":
			t.KlassensaetzeZuordnen(w, r)
		case "/api/books/import":
			t.ExcelImport(w, r)
		case "/api/admin/books/retry-covers":
			t.FremdeCoverNeuLaden(w, r)
		case "/api/books":
			t.TitelAnlegen(w, r)
		default:
			schreibeFehler(w, http.StatusNotFound, meldungUnbekannteRoute)
		}
	case http.MethodDelete:
		switch pfad {
		case routeKlassensaetze:
			t.KlassensatzGruppeLoeschen(w, r)
		case "/api/books":
			t.TitelLoeschen(w, r)
		default:
			schreibeFehler(w, http.StatusNotFound, meldungUnbekannteRoute)
		}
	default:
		schreibeFehler(w, http.StatusNotFound, meldungUnbekannteRoute)
	}
}
