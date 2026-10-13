package crypto

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"strconv"
)

// SchreibeEntschluesselt entschlüsselt Daten aus der Datenbank im Speicher und schreibt sie
// mit dem genannten Content-Type in die Antwort.
func SchreibeEntschluesselt(w http.ResponseWriter, geheimtext []byte, inhaltstyp string) error {
	if len(geheimtext) == 0 {
		http.Error(w, "Keine Daten gefunden", http.StatusNotFound)
		return fmt.Errorf("leerer ciphertext")
	}

	klartext, err := Entschluessele(geheimtext)
	if err != nil {
		// Die Antwort nennt den Fehler der Entschlüsselung nicht.
		http.Error(w, "Interner Serverfehler beim Lesen der Datei", http.StatusInternalServerError)
		return err
	}

	w.Header().Set("Content-Type", inhaltstyp)
	// Die Bilder sind Personendaten; weder Browser noch Zwischenspeicher legen sie ab.
	w.Header().Set("Cache-Control", "private, no-cache, no-store, must-revalidate")
	w.Header().Set("Pragma", "no-cache")
	w.Header().Set("Expires", "0")
	w.Header().Set("Content-Length", strconv.Itoa(len(klartext)))

	_, err = io.Copy(w, bytes.NewReader(klartext))
	return err
}
