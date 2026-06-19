package crypto

import (
	"bytes"
	"fmt"
	"io"
	"log/slog"
	"mime/multipart"
	"net/http"
	"strconv"
)

// EncryptUpload liest einen hochgeladenen File-Stream komplett in den RAM
// und verschlüsselt die Daten mit AES-256-GCM. Das Ergebnis kann als BYTEA 
// in einer PostgreSQL-Datenbank abgelegt werden.
func EncryptUpload(file multipart.File) ([]byte, error) {
	if file == nil {
		return nil, fmt.Errorf("leere Datei übergeben")
	}
	defer func() {
		if err := file.Close(); err != nil {
			slog.Warn("Fehler beim Schließen der Upload-Datei", "error", err)
		}
	}()

	plaintext, err := io.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("fehler beim Lesen der Upload-Datei: %w", err)
	}

	return Encrypt(plaintext)
}

// DecryptAndServe nimmt verschlüsselte BYTEA-Daten aus der Datenbank, 
// entschlüsselt sie im Arbeitsspeicher und streamt sie mit dem passenden 
// Content-Type sicher zum Client. Dies fungiert als Output-Middleware/Handler.
func DecryptAndServe(w http.ResponseWriter, ciphertext []byte, contentType string) error {
	if len(ciphertext) == 0 {
		http.Error(w, "Keine Daten gefunden", http.StatusNotFound)
		return fmt.Errorf("leerer ciphertext")
	}

	plaintext, err := Decrypt(ciphertext)
	if err != nil {
		// Generischer Error an den Client, um keine kryptografischen Fehler preiszugeben
		http.Error(w, "Interner Serverfehler beim Lesen der Datei", http.StatusInternalServerError)
		return err
	}

	w.Header().Set("Content-Type", contentType)
	// Wir setzen private Caching-Header, da die Bilder sensibel sind
	w.Header().Set("Cache-Control", "private, no-cache, no-store, must-revalidate")
	w.Header().Set("Pragma", "no-cache")
	w.Header().Set("Expires", "0")
	w.Header().Set("Content-Length", strconv.Itoa(len(plaintext)))

	_, err = io.Copy(w, bytes.NewReader(plaintext))
	return err
}
