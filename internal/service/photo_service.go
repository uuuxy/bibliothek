package service

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"

	"bibliothek/db"
	"bibliothek/internal/crypto"
	"bibliothek/pkg/imageutil"
	"bibliothek/repository"

	"github.com/jackc/pgx/v5"
)

// Die zwei Abweisungen des Foto-Uploads tragen einen Namen, damit die Route sie von einer
// Störung des Servers unterscheidet.
var (
	// ErrFotoLeserUnbekannt meldet: Unter der Kennung steht kein Leser.
	ErrFotoLeserUnbekannt = errors.New("schüler nicht gefunden")
	// ErrFotoUnlesbar meldet: Die Daten sind kein Bild, das sich lesen und umwandeln lässt.
	ErrFotoUnlesbar = errors.New("das bild ist nicht lesbar")
)

// UploadStudentPhoto verarbeitet den Base64-String eines Fotos, konvertiert ihn zu WebP,
// verschlüsselt ihn per AES und speichert ihn in der Datenbank ab.
func UploadStudentPhoto(ctx context.Context, dbPool db.PgxPoolIface, studentID string, base64DataStr string) (string, error) {
	// 1. Die Ausweisnummer des LESERS — Tabelle `leser`, nicht die Sicht `schueler`: Die
	// Akte bietet den Kamera-Knopf jedem Leser an, und die Auslieferung (api/photo_serve.go)
	// liest sein Bild über dieselbe Tabelle. Über die Sicht bekam ein Kollege hier 404
	// (TestUploadStudentPhoto_AuchFuerKollegen). COALESCE, weil ein Kollege aus der
	// Selbstanmeldung noch keine Nummer hat (OFFEN.md 5.16 C) — das Bild wird gespeichert,
	// eine Bild-URL gibt es wie in resolveFotoURL erst mit der Nummer.
	barcodeID, err := repository.AusweisnummerDesLesers(ctx, dbPool, studentID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", ErrFotoLeserUnbekannt
		}
		return "", err
	}

	// 2. Decode base64 image data
	base64Data := strings.TrimPrefix(base64DataStr, "data:image/jpeg;base64,")
	base64Data = strings.TrimPrefix(base64Data, "data:image/png;base64,")

	imgBytes, err := base64.StdEncoding.DecodeString(base64Data)
	if err != nil {
		return "", fmt.Errorf("%w: ungültiges base64-format: %w", ErrFotoUnlesbar, err)
	}

	// 3. Konvertierung nach WebP. Die Umwandlung rechnet nur auf den übergebenen Bytes;
	// scheitert sie, liegt es am Bild (kein JPEG oder PNG, zu groß), nicht am Server.
	webpBytes, err := imageutil.ConvertToWebP(imgBytes, 80)
	if err != nil {
		return "", fmt.Errorf("%w: %w", ErrFotoUnlesbar, err)
	}

	// 4. Verschlüsseln der WebP-Bytes
	encryptedData, err := crypto.Encrypt(webpBytes)
	if err != nil {
		return "", fmt.Errorf("fehler bei der fotostrukturierung: %w", err)
	}

	// 5. In der Datenbank abspeichern (Upsert in schueler_fotos)
	if err := repository.SpeichereFoto(ctx, dbPool, studentID, encryptedData); err != nil {
		return "", fmt.Errorf("fehler beim speichern des fotos in der db: %w", err)
	}

	if barcodeID == "" {
		return "", nil
	}
	photoURL := fmt.Sprintf("/api/schueler/%s/photo", barcodeID)
	return photoURL, nil
}
