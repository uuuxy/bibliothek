package katalog

import (
	"net/http"
	"os"
	"strings"

	"bibliothek/pkg/coverablage"
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

// Uploads liefert die Dateien aus uploads/ aus, ohne Anmeldung: Katalog und Monitor zeigen Cover
// vor jeder Anmeldung.
// GET /uploads/
func (t *Tueren) Uploads() http.Handler {
	return http.StripPrefix("/uploads/", http.FileServer(neuteredFileSystem{http.Dir(coverablage.Wurzel)}))
}
