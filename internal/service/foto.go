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

// SchreibeLeserFoto verarbeitet den Base64-String eines Fotos, konvertiert ihn zu WebP,
// verschlüsselt ihn per AES und speichert ihn in der Datenbank ab.
func SchreibeLeserFoto(ctx context.Context, pool db.PgxPoolIface, leserID string, base64Text string) (string, error) {
	// 1. Die Ausweisnummer des LESERS — Tabelle `leser`, nicht die Sicht `schueler`: Die
	// Akte bietet den Kamera-Knopf jedem Leser an, und die Auslieferung (api/leser/photo_serve.go)
	// liest sein Bild über dieselbe Tabelle. Über die Sicht bekam ein Kollege hier 404
	// (TestSchreibeLeserFoto_AuchFuerKollegen). COALESCE, weil ein Kollege aus der
	// Selbstanmeldung noch keine Nummer hat (OFFEN.md 5.16 C) — das Bild wird gespeichert,
	// eine Bild-URL gibt es wie in resolveFotoURL erst mit der Nummer.
	barcodeID, err := repository.LadeAusweisnummerDesLesers(ctx, pool, leserID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", ErrFotoLeserUnbekannt
		}
		return "", err
	}

	// 2. Decode base64 image data
	base64Daten := strings.TrimPrefix(base64Text, "data:image/jpeg;base64,")
	base64Daten = strings.TrimPrefix(base64Daten, "data:image/png;base64,")

	bild, err := base64.StdEncoding.DecodeString(base64Daten)
	if err != nil {
		return "", fmt.Errorf("%w: ungültiges base64-format: %w", ErrFotoUnlesbar, err)
	}

	// 3. Konvertierung nach WebP. Die Umwandlung rechnet nur auf den übergebenen Bytes;
	// scheitert sie, liegt es am Bild (kein JPEG oder PNG, zu groß), nicht am Server.
	webp, err := imageutil.AlsWebP(bild, 80)
	if err != nil {
		return "", fmt.Errorf("%w: %w", ErrFotoUnlesbar, err)
	}

	// 4. Verschlüsseln der WebP-Bytes
	verschluesselt, err := crypto.Verschluessele(webp)
	if err != nil {
		return "", fmt.Errorf("fehler bei der fotostrukturierung: %w", err)
	}

	// 5. In der Datenbank abspeichern (Upsert in schueler_fotos)
	if err := repository.SchreibeFoto(ctx, pool, leserID, verschluesselt); err != nil {
		return "", fmt.Errorf("fehler beim speichern des fotos in der db: %w", err)
	}

	if barcodeID == "" {
		return "", nil
	}
	fotoURL := fmt.Sprintf("/api/schueler/%s/photo", barcodeID)
	return fotoURL, nil
}
