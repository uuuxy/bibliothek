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
		f.Close()
		return nil, err
	}

	if s.IsDir() {
		index := strings.TrimSuffix(path, "/") + "/index.html"
		if idx, err := nfs.fs.Open(index); err != nil {
			f.Close()
			return nil, err
		} else {
			idx.Close()
		}
	}

	return f, nil
}

type APIHandlerConfig struct {
	Repo             *BookRepository
	Metadaten        *MetadatenClient
	RequireAuth      func(http.Handler) http.Handler
	RequireAdminAuth func(http.Handler) http.Handler
}

type APIHandler struct {
	repo      *BookRepository
	metadaten *MetadatenClient
	backup    *BackupManager
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

	// Protected GETs (view_books / view_inventur)
	handler.mux.Handle("GET /api/books", config.RequireAuth(http.HandlerFunc(handler.BearbeiteBuecherListe)))
	handler.mux.Handle("GET /api/books/{id}", config.RequireAuth(http.HandlerFunc(handler.BearbeiteBuchLesen)))
	handler.mux.Handle("GET /api/class-books", config.RequireAuth(http.HandlerFunc(handler.handleClassBooks)))
	handler.mux.Handle("GET /api/lookup/", config.RequireAuth(http.HandlerFunc(handler.handleLookup)))
	handler.mux.Handle("GET /api/subjects", config.RequireAuth(http.HandlerFunc(handler.handleGetSubjects)))

	// Admin Protected (edit_books)
	adminH := config.RequireAdminAuth(http.HandlerFunc(handler.handleAdminBooks))
	
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
