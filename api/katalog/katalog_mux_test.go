package katalog

import "net/http"

// katalogMux stellt die Türen des Medienkatalogs unter ihren Adressen auf, ohne die Prüfung der
// Rechte: Die Tests hier prüfen die Türen samt Platzhalter im Pfad. Adresse und Recht im Betrieb
// stehen in api/routen_katalog.go, und api/ prüft sie über alle Routen.
func katalogMux(t *Tueren) *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/books", t.TitelListe())
	mux.HandleFunc("GET /api/books/{id}", t.Titel())
	mux.HandleFunc("GET /api/books/vorhanden", t.TitelVorhanden())
	mux.HandleFunc("POST /api/books", t.TitelAnlegen())
	mux.HandleFunc("PUT /api/books/{id}", t.TitelAendern())
	mux.HandleFunc("DELETE /api/books", t.MarkierteTitelLoeschen())
	mux.HandleFunc("POST /api/books/import", t.ExcelImport())
	mux.HandleFunc("POST /api/books/{id}/refresh-cover", t.CoverAktualisieren())
	mux.HandleFunc("POST /api/books/{id}/cover-upload", t.CoverHochladen())
	mux.Handle("GET /uploads/", t.Uploads())
	return mux
}
