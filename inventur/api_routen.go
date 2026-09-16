package inventur

import (
	"net/http"
	"strings"
)

// neuteredFileSystem prevents directory listing by wrapping an http.FileSystem
// and returning an error if a requested path is a directory without an index.html.
type neuteredFileSystem struct {
	fs http.FileSystem
}

func (nfs neuteredFileSystem) Open(path string) (http.File, error) {
	f, err := nfs.fs.Open(path)
	if err != nil {
		return nil, err
	}

	s, err := f.Stat()
	if err != nil {
		_ = f.Close() //nolint:errcheck
		return nil, err
	}

	if s.IsDir() {
		index := strings.TrimSuffix(path, "/") + "/index.html"
		if idx, err := nfs.fs.Open(index); err != nil {
			_ = f.Close() //nolint:errcheck
			return nil, err
		} else {
			_ = idx.Close() //nolint:errcheck
		}
	}

	return f, nil
}

// APIHandlerConfig ist der Bauplan für den APIHandler. Die beiden Middleware-Felder
// werden von api/router.go hereingereicht: Das Inventur-Modul kennt die RBAC-Prüfung
// nicht selbst, bekommt sie aber verpflichtend gestellt.
type APIHandlerConfig struct {
	Repo             *BookRepository
	Metadaten        *MetadatenClient
	RequireViewBooks func(http.Handler) http.Handler
	RequireEditBooks func(http.Handler) http.Handler
	// RequireDeleteBooks ist das Recht, Bestand zu VERNICHTEN — getrennt von
	// edit_books, weil Ändern und Löschen verschiedene Fehler sind. Bis zum
	// 17.09.2026 hing der Massenlöschweg (DELETE /api/books) an edit_books,
	// während das Löschen eines einzelnen Titels delete_books verlangte: Die
	// größere Handlung stand unter dem kleineren Recht.
	RequireDeleteBooks func(http.Handler) http.Handler
	// RequireAuthenticated verlangt eine gültige Sitzung, aber kein Fachrecht —
	// für die Nur-Lese-Sichten des Lehrerportals (siehe Portal-Routen unten).
	RequireAuthenticated func(http.Handler) http.Handler
}

// APIHandler bündelt die Endpunkte des Inventur-Moduls.
//
// Hier stand bis zum 30.07.2026 ein Feld backup *BackupManager, das NewAPIHandler nie
// gesetzt hat: ein vollständiges zweites Backup-System (Dump, Datei-Rotation,
// Benachrichtigungs-Mail), das nichts erreichen konnte. Gesichert wird über
// jobs.BackupJob — verschlüsselt, täglich, mit Statusanzeige in der Oberfläche.
// Die .env.example lud allerdings dazu ein, BACKUP_EMAIL_TO zu setzen und auf
// Benachrichtigungen zu warten, die nie kommen konnten. Ersatzlos entfernt.
type APIHandler struct {
	repo      *BookRepository
	metadaten *MetadatenClient
	mux       *http.ServeMux
}

// NewAPIHandler baut den Endpunkt-Satz des Inventur-Moduls samt eigenem ServeMux.
func NewAPIHandler(config APIHandlerConfig) *APIHandler {
	handler := &APIHandler{
		repo:      config.Repo,
		metadaten: config.Metadaten,
		mux:       http.NewServeMux(),
	}

	// Unprotected Uploads (oder durch parent geschützt)
	handler.mux.Handle("GET /uploads/", http.StripPrefix("/uploads/", http.FileServer(neuteredFileSystem{http.Dir("uploads")})))

	// Lesend: RBAC-Permission view_books (injiziert aus api/router.go)
	handler.mux.Handle("GET /api/books", config.RequireViewBooks(http.HandlerFunc(handler.BearbeiteBuecherListe)))
	handler.mux.Handle("GET /api/books/{id}", config.RequireViewBooks(http.HandlerFunc(handler.BearbeiteBuchLesen)))
	handler.mux.Handle("GET /api/class-books", config.RequireViewBooks(http.HandlerFunc(handler.handleClassBooks)))
	handler.mux.Handle("GET /api/lookup/{isbn}", config.RequireViewBooks(http.HandlerFunc(handler.handleLookup)))

	// Lehrerportal (Betreiber-Entscheidung 24.08.2026): Das Kollegium sieht Bestand
	// und Mengen der Lernmittel sowie die Klassensatz-Zuordnung — hinter der
	// Anmeldung, aber OHNE view_books: Das Recht würde der Rolle auch den ganzen
	// Medienkatalog im Menü öffnen. Beide Antworten enthalten ausschließlich
	// Buch- und Zähldaten, keine Ausleih- oder Personendaten (dieselben Handler
	// wie /api/books und /api/class-books, nur eine andere Tür).
	handler.mux.Handle("GET /api/portal/klassensaetze", config.RequireAuthenticated(http.HandlerFunc(handler.handleClassBooks)))
	// Schulbücher je Fach für die Fachsprecher (03.09.2026): Zahlen, Titel, PDF-Export —
	// nur Lernmittel (ist_lernmittel), nur Buch- und Zähldaten.
	handler.mux.Handle("GET /api/portal/lernmittel", config.RequireAuthenticated(http.HandlerFunc(handler.handlePortalLernmittel)))
	handler.mux.Handle("GET /api/portal/lernmittel/export", config.RequireAuthenticated(http.HandlerFunc(handler.handlePortalLernmittelExport)))

	// Schreibend: RBAC-Permission edit_books (injiziert aus api/router.go)
	adminH := config.RequireEditBooks(http.HandlerFunc(handler.handleAdminBooks))

	handler.mux.Handle("GET /api/admin/", adminH)
	handler.mux.Handle("POST /api/admin/", adminH)
	handler.mux.Handle("PUT /api/admin/", adminH)
	handler.mux.Handle("DELETE /api/admin/", adminH)

	handler.mux.Handle("POST /api/books/import", adminH)
	handler.mux.Handle("POST /api/books", adminH)
	handler.mux.Handle("DELETE /api/books", config.RequireDeleteBooks(http.HandlerFunc(handler.handleAdminBooks)))

	// Die Buch-Kennung als Platzhalter statt Sammelroute POST/PUT /api/books/ mit selbst
	// zerlegtem Pfad: Nur einen Platzhalter sieht ValidateUUIDParamsMiddleware (sie steckt
	// in RequirePermission, die api/router.go hier einsetzt), und das API-Inventar gleicht
	// die Routen einzeln ab. Bis zum 13.09.2026 kam PUT /api/books/x als 500 zurück.
	handler.mux.Handle("PUT /api/books/{id}", config.RequireEditBooks(http.HandlerFunc(handler.BearbeiteBuchAktualisieren)))
	handler.mux.Handle("PUT /api/books/{id}/cover", config.RequireEditBooks(http.HandlerFunc(handler.handleUpdateCover)))
	handler.mux.Handle("POST /api/books/{id}/refresh-cover", config.RequireEditBooks(http.HandlerFunc(handler.handleRefreshCover)))
	handler.mux.Handle("POST /api/books/{id}/cover-upload", config.RequireEditBooks(http.HandlerFunc(handler.handleUploadCover)))

	return handler
}

func (handler *APIHandler) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	handler.mux.ServeHTTP(writer, request)
}
