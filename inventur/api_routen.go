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

type APIHandlerConfig struct {
	Repo             *BookRepository
	Metadaten        *MetadatenClient
	RequireViewBooks func(http.Handler) http.Handler
	RequireEditBooks func(http.Handler) http.Handler
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
	handler.mux.Handle("GET /api/lookup/", config.RequireViewBooks(http.HandlerFunc(handler.handleLookup)))
	handler.mux.Handle("GET /api/subjects", config.RequireViewBooks(http.HandlerFunc(handler.handleGetSubjects)))

	// Schreibend: RBAC-Permission edit_books (injiziert aus api/router.go)
	adminH := config.RequireEditBooks(http.HandlerFunc(handler.handleAdminBooks))

	handler.mux.Handle("GET /api/admin/", adminH)
	handler.mux.Handle("POST /api/admin/", adminH)
	handler.mux.Handle("PUT /api/admin/", adminH)
	handler.mux.Handle("DELETE /api/admin/", adminH)

	handler.mux.Handle("POST /api/books/import", adminH)
	handler.mux.Handle("POST /api/books", adminH)
	handler.mux.Handle("POST /api/books/", adminH)
	handler.mux.Handle("PUT /api/books/", adminH)
	handler.mux.Handle("DELETE /api/books", adminH)

	return handler
}

func (handler *APIHandler) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	handler.mux.ServeHTTP(writer, request)
}
