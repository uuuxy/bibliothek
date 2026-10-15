package inventur

import (
	"net/http"
	"os"
	"strings"
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

// TuerenKonfig ist der Bauplan für den APIHandler. Die beiden Middleware-Felder
// werden von api/router.go hereingereicht: Das Inventur-Modul kennt die RBAC-Prüfung
// nicht selbst, bekommt sie aber verpflichtend gestellt.
type TuerenKonfig struct {
	Abfragen        *TitelAbfragen
	Metadaten       *MetadatenClient
	VerlangeLesen   func(http.Handler) http.Handler
	VerlangeAendern func(http.Handler) http.Handler
	// VerlangeLoeschen ist das Recht, Bestand zu VERNICHTEN — getrennt von
	// edit_books, weil Ändern und Löschen verschiedene Fehler sind. Bis zum
	// 17.09.2026 hing der Massenlöschweg (DELETE /api/books) an edit_books,
	// während das Löschen eines einzelnen Titels delete_books verlangte: Die
	// größere Handlung stand unter dem kleineren Recht.
	VerlangeLoeschen func(http.Handler) http.Handler
	// VerlangeAnmeldung verlangt eine gültige Sitzung, aber kein Fachrecht —
	// für die Nur-Lese-Sichten des Lehrerportals (siehe Portal-Routen unten).
	VerlangeAnmeldung func(http.Handler) http.Handler
}

// Tueren bündelt die Endpunkte des Inventur-Moduls.
//
// Hier stand bis zum 30.07.2026 ein Feld backup *BackupManager, das NewTueren nie
// gesetzt hat: ein vollständiges zweites Backup-System (Dump, Datei-Rotation,
// Benachrichtigungs-Mail), das nichts erreichen konnte. Gesichert wird über
// jobs.BackupLauf — verschlüsselt, täglich, mit Statusanzeige in der Oberfläche.
// Die .env.example lud allerdings dazu ein, BACKUP_EMAIL_TO zu setzen und auf
// Benachrichtigungen zu warten, die nie kommen konnten. Ersatzlos entfernt.
type Tueren struct {
	abfragen  *TitelAbfragen
	metadaten *MetadatenClient
	mux       *http.ServeMux
}

// NewTueren baut den Endpunkt-Satz des Inventur-Moduls samt eigenem ServeMux.
func NewTueren(konfig TuerenKonfig) *Tueren {
	tueren := &Tueren{
		abfragen:  konfig.Abfragen,
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

	// Lehrerportal (Betreiber-Entscheidung 24.08.2026): Das Kollegium sieht Bestand
	// und Mengen der Lernmittel sowie die Klassensatz-Zuordnung — hinter der
	// Anmeldung, aber OHNE view_books: Das Recht würde der Rolle auch den ganzen
	// Medienkatalog im Menü öffnen. Beide Antworten enthalten ausschließlich
	// Buch- und Zähldaten, keine Ausleih- oder Personendaten (dieselben Handler
	// wie /api/books und /api/class-books, nur eine andere Tür).
	tueren.mux.Handle("GET /api/portal/klassensaetze", konfig.VerlangeAnmeldung(http.HandlerFunc(tueren.Klassensaetze)))
	// Schulbücher je Fach für die Fachsprecher (03.09.2026): Zahlen, Titel, PDF-Export —
	// nur Lernmittel (ist_lernmittel), nur Buch- und Zähldaten.
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

	// Die Buch-Kennung als Platzhalter statt Sammelroute POST/PUT /api/books/ mit selbst
	// zerlegtem Pfad: Nur einen Platzhalter sieht uuidPfadMiddleware (sie steckt
	// in verlangeRecht, die api/router.go hier einsetzt), und das API-Inventar gleicht
	// die Routen einzeln ab. Bis zum 13.09.2026 kam PUT /api/books/x als 500 zurück.
	tueren.mux.Handle("PUT /api/books/{id}", konfig.VerlangeAendern(http.HandlerFunc(tueren.TitelAendern)))
	tueren.mux.Handle("POST /api/books/{id}/refresh-cover", konfig.VerlangeAendern(http.HandlerFunc(tueren.CoverAktualisieren)))
	tueren.mux.Handle("POST /api/books/{id}/cover-upload", konfig.VerlangeAendern(http.HandlerFunc(tueren.CoverHochladen)))

	return tueren
}

func (t *Tueren) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	t.mux.ServeHTTP(w, r)
}
