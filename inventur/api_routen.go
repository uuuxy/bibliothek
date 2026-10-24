// Package inventur ist der Medienkatalog mit eigenen Türen und eigener Datenbankschicht: Titel
// lesen, anlegen, ändern und löschen, der Import aus Excel, die Abfrage über die ISBN, die
// Cover und die hochgeladenen Dateien (docs/ARCHITEKTUR.md 5.2.3).
package inventur

import (
	"net/http"
	"os"
	"strings"

	"bibliothek/db"
	"bibliothek/internal/katalogdienste"
)

// neuteredFileSystem liefert nur Dateien direkt im Verzeichnis aus: /uploads/ ist ohne
// Anmeldung lesbar, und nur dort legen Upload und Cover-Abruf ihre Dateien ab. Ein
// Verzeichnis öffnet es nie, auch keines mit index.html — http.FileServer zeigte dessen
// Dateinamen sonst als Liste.
type neuteredFileSystem struct {
	fs http.FileSystem
}

func (dateisystem neuteredFileSystem) Open(pfad string) (http.File, error) {
	if strings.Contains(strings.Trim(pfad, "/"), "/") {
		return nil, os.ErrNotExist
	}

	f, err := dateisystem.fs.Open(pfad)
	if err != nil {
		return nil, err
	}

	s, err := f.Stat()
	if err != nil {
		_ = f.Close() //nolint:errcheck
		return nil, err
	}

	if s.IsDir() {
		_ = f.Close() //nolint:errcheck
		return nil, os.ErrNotExist
	}

	return f, nil
}

// TuerenKonfig ist der Bauplan für die Türen. Die Prüfungen des Zugriffs reicht api/router.go
// herein: Das Inventur-Modul kennt die Prüfung der Rechte nicht selbst, bekommt sie aber
// verpflichtend gestellt.
type TuerenKonfig struct {
	Abfragen *TitelAbfragen
	// Pool trägt die Schlüssel des Listenimports (repository.ReserviereImportSchluessel).
	Pool            db.Pool
	Metadaten       *katalogdienste.MetadatenClient
	VerlangeLesen   func(http.Handler) http.Handler
	VerlangeAendern func(http.Handler) http.Handler
	// VerlangeLoeschen ist das Recht, Bestand zu vernichten, getrennt von edit_books, weil Ändern
	// und Löschen verschiedene Fehler sind. Hinge das Löschen vieler Titel (DELETE /api/books) an
	// edit_books, während das Löschen eines einzelnen Titels delete_books verlangt, stünde die
	// größere Handlung unter dem kleineren Recht.
	VerlangeLoeschen func(http.Handler) http.Handler
	// VerlangeAnmeldung verlangt eine gültige Sitzung, aber kein Fachrecht —
	// für die Nur-Lese-Sichten des Lehrerportals (siehe Portal-Routen unten).
	VerlangeAnmeldung func(http.Handler) http.Handler
}

// Tueren bündelt die Türen des Inventur-Moduls. Gesichert wird nicht hier, sondern über
// jobs.BackupLauf: verschlüsselt, täglich, mit Anzeige des Stands in der Oberfläche.
type Tueren struct {
	abfragen  *TitelAbfragen
	metadaten *katalogdienste.MetadatenClient
	pool      db.Pool
	mux       *http.ServeMux
}

// NewTueren baut den Endpunkt-Satz des Inventur-Moduls samt eigenem ServeMux.
func NewTueren(konfig TuerenKonfig) *Tueren {
	tueren := &Tueren{
		abfragen:  konfig.Abfragen,
		pool:      konfig.Pool,
		metadaten: konfig.Metadaten,
		mux:       http.NewServeMux(),
	}

	// Ohne Anmeldung lesbar: Katalog und Monitor zeigen Cover vor jeder Anmeldung.
	tueren.mux.Handle("GET /uploads/", http.StripPrefix("/uploads/", http.FileServer(neuteredFileSystem{http.Dir("uploads")})))

	// Lesend: RBAC-Permission view_books (injiziert aus api/router.go)
	tueren.mux.Handle("GET /api/books", konfig.VerlangeLesen(http.HandlerFunc(tueren.TitelListe)))
	tueren.mux.Handle("GET /api/books/{id}", konfig.VerlangeLesen(http.HandlerFunc(tueren.Titel)))
	tueren.mux.Handle("GET /api/books/vorhanden", konfig.VerlangeLesen(http.HandlerFunc(tueren.TitelVorhanden)))
	tueren.mux.Handle("GET /api/class-books", konfig.VerlangeLesen(http.HandlerFunc(tueren.Klassensaetze)))
	tueren.mux.Handle("GET /api/lookup/{isbn}", konfig.VerlangeLesen(http.HandlerFunc(tueren.ISBNSuche)))

	// Lehrerportal: Das Kollegium sieht Bestand
	// und Mengen der Lernmittel sowie die Klassensatz-Zuordnung — hinter der
	// Anmeldung, aber ohne view_books: Das Recht würde der Rolle auch den ganzen
	// Medienkatalog im Menü öffnen. Beide Antworten enthalten ausschließlich
	// Buch- und Zähldaten, keine Ausleih- oder Personendaten (dieselben Türen wie
	// /api/books und /api/class-books, nur eine andere Route).
	tueren.mux.Handle("GET /api/portal/klassensaetze", konfig.VerlangeAnmeldung(http.HandlerFunc(tueren.Klassensaetze)))
	// Schulbücher je Fach für die Fachsprecher: Zahlen, Titel, Export als PDF, nur Lernmittel
	// (ist_lernmittel), nur Buch- und Zähldaten.
	tueren.mux.Handle("GET /api/portal/lernmittel", konfig.VerlangeAnmeldung(http.HandlerFunc(tueren.PortalLernmittel)))
	tueren.mux.Handle("GET /api/portal/lernmittel/export", konfig.VerlangeAnmeldung(http.HandlerFunc(tueren.PortalLernmittelExport)))

	// Schreibend: RBAC-Permission edit_books (injiziert aus api/router.go)
	verwaltung := konfig.VerlangeAendern(http.HandlerFunc(tueren.TitelVerwaltung))

	tueren.mux.Handle("GET /api/admin/", verwaltung)
	tueren.mux.Handle("POST /api/admin/", verwaltung)
	tueren.mux.Handle("DELETE /api/admin/", verwaltung)

	tueren.mux.Handle("POST /api/books/import", verwaltung)
	tueren.mux.Handle("POST /api/books", verwaltung)
	tueren.mux.Handle("DELETE /api/books", konfig.VerlangeLoeschen(http.HandlerFunc(tueren.TitelVerwaltung)))

	// Die Kennung des Titels als Platzhalter statt einer Sammelroute POST/PUT /api/books/ mit
	// selbst zerlegtem Pfad: Nur einen Platzhalter sieht uuidPfadMiddleware (sie steckt in
	// verlangeRecht, die api/router.go hier einsetzt), und das API-Inventar gleicht die Routen
	// einzeln ab. Eine falsche Kennung wird so zu 400 statt 500.
	tueren.mux.Handle("PUT /api/books/{id}", konfig.VerlangeAendern(http.HandlerFunc(tueren.TitelAendern)))
	tueren.mux.Handle("POST /api/books/{id}/refresh-cover", konfig.VerlangeAendern(http.HandlerFunc(tueren.CoverAktualisieren)))
	tueren.mux.Handle("POST /api/books/{id}/cover-upload", konfig.VerlangeAendern(http.HandlerFunc(tueren.CoverHochladen)))

	return tueren
}

func (t *Tueren) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	t.mux.ServeHTTP(w, r)
}
