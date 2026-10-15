// Command migrate-fotos zieht unverschlüsselte Schülerfotos aus uploads/fotos in die Datenbank:
// Es verschlüsselt jedes Bild, speichert es in schueler_fotos, liest es zur Gegenprobe zurück
// und löscht erst dann die Datei (docs/SCRIPTS.md, Abschnitt 2).
package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"bibliothek/db"
	"bibliothek/internal/crypto"
	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	// Strukturiertes Log als JSON.
	logAusgabe := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logAusgabe)

	datenbankURL := os.Getenv("DATABASE_URL")
	if datenbankURL == "" {
		slog.Error("DATABASE_URL environment variable is not set")
		os.Exit(1)
	}

	schluessel := os.Getenv("APP_ENCRYPTION_KEY")
	if schluessel == "" {
		slog.Error("APP_ENCRYPTION_KEY environment variable is not set")
		os.Exit(1)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, datenbankURL)
	if err != nil {
		slog.Error("Failed to connect to DB", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	// Das Werkzeug räumt hinter sich auf: Die Quelldateien sind unverschlüsselte
	// Schülerfotos, und ein Hinweis auf der Konsole sichert sie nicht.
	behalten := os.Getenv("FOTOS_BEHALTEN") == "1"
	if behalten {
		slog.Warn("FOTOS_BEHALTEN=1 — die unverschluesselten Quelldateien bleiben liegen")
	}

	fotoVerzeichnis := filepath.Join("uploads", "fotos")
	wurzel, err := os.OpenRoot(fotoVerzeichnis)
	if err != nil {
		if os.IsNotExist(err) {
			slog.Info("Kein uploads/fotos Verzeichnis gefunden. Nichts zu migrieren.")
			return
		}
		slog.Error("Fehler beim Öffnen des Foto-Verzeichnisses", "error", err)
		os.Exit(1)
	}
	defer func() {
		if schliessFehler := wurzel.Close(); schliessFehler != nil {
			slog.Error("Fehler beim Schließen des Root-Verzeichnisses", "error", schliessFehler)
		}
	}()

	ordner, err := wurzel.Open(".")
	if err != nil {
		slog.Error("Fehler beim Lesen des Foto-Verzeichnisses", "error", err)
		os.Exit(1)
	}
	eintraege, err := ordner.ReadDir(-1)
	if schliessFehler := ordner.Close(); schliessFehler != nil {
		slog.Error("Fehler beim Schließen des Foto-Verzeichnisses", "error", schliessFehler)
	}
	if err != nil {
		slog.Error("Fehler beim Lesen des Foto-Verzeichnisses", "error", err)
		os.Exit(1)
	}

	gefunden, uebernommen, geloescht := migriereAlleFotos(pool, wurzel, eintraege, behalten)

	fmt.Printf("Migration abgeschlossen. %d Fotos gefunden, %d erfolgreich migriert und verschlüsselt.\n", gefunden, uebernommen)
	fmt.Printf("%d Quelldateien nach bestandener Gegenprobe gelöscht.\n", geloescht)

	// Melden, was liegen bleibt: Jede verbliebene Datei ist ein unverschlüsseltes Schülerfoto.
	if uebrig := gefunden - geloescht; uebrig > 0 {
		fmt.Printf("ACHTUNG: %d unverschlüsselte Fotos liegen weiterhin in 'uploads/fotos'.\n", uebrig)
		fmt.Println("         Der Server liefert sie nicht aus, sie liegen aber lesbar auf der Platte.")
		fmt.Println("         Nach Prüfung entfernen:  shred -u uploads/fotos/*.jpg")
	} else if gefunden > 0 {
		fmt.Println("Das Verzeichnis 'uploads/fotos' enthält keine Fotos mehr.")
	}
}

// migriereAlleFotos verschlüsselt alle .jpg-Dateien im Verzeichnis und liefert, wie viele
// gefunden, migriert und gelöscht wurden.
//
// Die Kennungen der Schüler kommen in einer Abfrage für alle Barcodes; geschrieben wird je
// Foto einzeln und nicht als pgx.Batch. Ein Batch liefert je Ergebnis nur ErrNoRows und kann
// nicht sagen, welche Datei es traf und ob es ein Fehler war; die Liste ist das Ergebnis einer
// einmaligen Übernahme. Wer doch einen Batch nimmt, führt die Barcodes in dessen Reihenfolge
// mit.
func migriereAlleFotos(pool db.Pool, wurzel *os.Root, eintraege []os.DirEntry, behalten bool) (gefunden, uebernommen, geloescht int) {
	// ReadDir liefert die Dateien in der Reihenfolge des Dateisystems, auf ext4 beliebig. Sortiert
	// wird hier statt beim Aufrufer, damit Log und Tests dieselbe Reihenfolge sehen.
	sort.Slice(eintraege, func(i, j int) bool { return eintraege[i].Name() < eintraege[j].Name() })

	barcodes := barcodesDerFotos(eintraege)
	if len(barcodes) == 0 {
		return 0, 0, 0
	}
	schuelerIDs, ok := ladeSchuelerIDs(pool, barcodes)
	if !ok {
		return 0, 0, 0
	}

	for _, eintrag := range eintraege {
		barcodeID, istFoto := fotoBarcode(eintrag)
		if !istFoto {
			continue
		}
		gefunden++

		schuelerID, ok := schuelerIDs[barcodeID]
		if !ok {
			slog.Warn("Kein Schüler für Barcode gefunden (übersprungen)", "barcode", barcodeID)
			continue
		}

		if !migriereFoto(pool, wurzel, eintrag.Name(), barcodeID, schuelerID) {
			continue
		}
		uebernommen++
		if behalten {
			continue
		}
		if err := wurzel.Remove(eintrag.Name()); err != nil {
			slog.Error("Quelldatei konnte nicht gelöscht werden — sie bleibt unverschlüsselt liegen",
				"file", eintrag.Name(), "error", err)
			continue
		}
		geloescht++
	}
	return gefunden, uebernommen, geloescht
}

// fotoBarcode liefert den Barcode eines Fotos: der Name der .jpg-Datei ohne Endung. Abfrage
// und Schleife fragen beide hier, damit sie dieselben Dateien meinen.
func fotoBarcode(eintrag os.DirEntry) (string, bool) {
	if eintrag.IsDir() || !strings.HasSuffix(strings.ToLower(eintrag.Name()), ".jpg") {
		return "", false
	}
	return strings.TrimSuffix(eintrag.Name(), filepath.Ext(eintrag.Name())), true
}

// barcodesDerFotos sammelt die Barcodes aller Fotos des Verzeichnisses.
func barcodesDerFotos(eintraege []os.DirEntry) []string {
	var barcodes []string
	for _, eintrag := range eintraege {
		if barcodeID, istFoto := fotoBarcode(eintrag); istFoto {
			barcodes = append(barcodes, barcodeID)
		}
	}
	return barcodes
}

// ladeSchuelerIDs lädt die Schüler zu allen Barcodes in einer Abfrage. false heißt: Die Abfrage
// ist gescheitert, der Fehler steht im Log.
func ladeSchuelerIDs(pool db.Pool, barcodes []string) (map[string]string, bool) {
	schuelerIDs := make(map[string]string)
	abfrage := "SELECT barcode_id, id FROM schueler WHERE barcode_id = ANY($1)"
	zeilen, err := pool.Query(context.Background(), abfrage, barcodes)
	if err != nil {
		slog.Error("Fehler beim Laden der Schueler-IDs", "error", err)
		return nil, false
	}
	defer zeilen.Close()

	for zeilen.Next() {
		var barcode, schuelerID string
		if err := zeilen.Scan(&barcode, &schuelerID); err != nil {
			slog.Error("Fehler beim Scannen der Schueler-ID", "error", err)
			continue
		}
		schuelerIDs[barcode] = schuelerID
	}
	if zeilen.Err() != nil {
		slog.Error("Fehler nach dem Scannen der Schueler-IDs", "error", zeilen.Err())
		return nil, false
	}
	return schuelerIDs, true
}

// migriereFoto liest, verschlüsselt und speichert das Foto einer Datei (Dateiname
// "<barcode>.jpg") in schueler_fotos. Liefert true bei Erfolg; Fehler stehen im Log und enden
// mit false.
func migriereFoto(pool db.Pool, wurzel *os.Root, name string, barcodeID string, schuelerID string) bool {
	datei, err := wurzel.Open(name)
	if err != nil {
		slog.Error("Konnte Bild nicht öffnen", "file", name, "error", err)
		return false
	}
	defer func() {
		if schliessFehler := datei.Close(); schliessFehler != nil {
			slog.Error("Fehler beim Schließen der Datei", "file", name, "error", schliessFehler)
		}
	}()

	bild, err := io.ReadAll(datei)
	if err != nil {
		slog.Error("Konnte Bild nicht lesen", "file", name, "error", err)
		return false
	}

	verschluesselt, err := crypto.Verschluessele(bild)
	if err != nil {
		slog.Error("Konnte Bild nicht verschlüsseln", "file", name, "error", err)
		return false
	}

	abfrage := `
		INSERT INTO schueler_fotos (schueler_id, foto_encrypted)
		VALUES ($1, $2)
		ON CONFLICT (schueler_id) DO UPDATE SET
			foto_encrypted = EXCLUDED.foto_encrypted,
			aktualisiert_am = CURRENT_TIMESTAMP
	`
	_, err = pool.Exec(context.Background(), abfrage, schuelerID, verschluesselt)
	if err != nil {
		slog.Error("Fehler beim Einfügen in die Datenbank", "student_id", schuelerID, "error", err)
		return false
	}

	// Gegenprobe vor dem Löschen: Die Datei ist bis hierhin die einzige Kopie des Bildes. Ein
	// "INSERT ohne Fehler" heißt noch nicht, dass sich das Foto je wieder anzeigen lässt; ein falsch
	// abgeleiteter Schlüssel etwa fällt erst beim Entschlüsseln auf, und dann ist die Quelle längst
	// weg. Deshalb wird zurückgelesen, entschlüsselt und verglichen; erst danach meldet die Funktion
	// Erfolg.
	if !istGespeichertWieOriginal(pool, schuelerID, bild) {
		slog.Error("Gegenprobe fehlgeschlagen — das gespeicherte Foto ließ sich nicht "+
			"wieder herstellen; die Quelldatei bleibt liegen", "barcode", barcodeID)
		return false
	}

	slog.Info("Foto erfolgreich migriert und zurückgelesen", "barcode", barcodeID)
	return true
}

// istGespeichertWieOriginal holt das eben geschriebene Foto zurück, entschlüsselt es und
// vergleicht es mit dem Original.
func istGespeichertWieOriginal(pool db.Pool, schuelerID string, original []byte) bool {
	var gespeichert []byte
	err := pool.QueryRow(context.Background(),
		`SELECT foto_encrypted FROM schueler_fotos WHERE schueler_id = $1`, schuelerID).Scan(&gespeichert)
	if err != nil {
		slog.Error("Foto konnte nicht zurückgelesen werden", "student_id", schuelerID, "error", err)
		return false
	}

	klar, err := crypto.Entschluessele(gespeichert)
	if err != nil {
		slog.Error("Zurückgelesenes Foto ließ sich nicht entschlüsseln", "student_id", schuelerID, "error", err)
		return false
	}
	return bytes.Equal(klar, original)
}
