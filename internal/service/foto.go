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

// SchreibeLeserFoto wandelt das Foto eines Lesers (Base64) in WebP, verschlüsselt es und
// speichert es. Zurück kommt die Adresse des Bildes, leer, solange der Leser keine
// Ausweisnummer hat.
func SchreibeLeserFoto(ctx context.Context, pool db.PgxPoolIface, leserID string, base64Text string) (string, error) {
	// Die Ausweisnummer kommt aus der Tabelle leser, nicht aus der Sicht schueler: Die Akte
	// bietet den Kamera-Knopf jedem Leser an, und die Auslieferung (api/leser/foto.go)
	// liest das Bild über dieselbe Tabelle (TestSchreibeLeserFoto_AuchFuerKollegen). Ein
	// Kollege aus der Selbstanmeldung hat noch keine Nummer: Das Bild wird gespeichert, eine
	// Adresse gibt es erst mit der Nummer.
	barcodeID, err := repository.LadeAusweisnummerDesLesers(ctx, pool, leserID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", ErrFotoLeserUnbekannt
		}
		return "", err
	}

	// Die Oberfläche schickt eine Daten-Adresse; ihre Vorsilbe fällt weg.
	base64Daten := strings.TrimPrefix(base64Text, "data:image/jpeg;base64,")
	base64Daten = strings.TrimPrefix(base64Daten, "data:image/png;base64,")

	bild, err := base64.StdEncoding.DecodeString(base64Daten)
	if err != nil {
		return "", fmt.Errorf("%w: ungültiges base64-format: %w", ErrFotoUnlesbar, err)
	}

	// Die Umwandlung in WebP rechnet nur auf den übergebenen Bytes; scheitert sie, liegt es am
	// Bild (kein JPEG oder PNG, zu groß), nicht am Server.
	webp, err := imageutil.AlsWebP(bild, 80)
	if err != nil {
		return "", fmt.Errorf("%w: %w", ErrFotoUnlesbar, err)
	}

	verschluesselt, err := crypto.Verschluessele(webp)
	if err != nil {
		return "", fmt.Errorf("fehler bei der fotostrukturierung: %w", err)
	}

	if err := repository.SchreibeFoto(ctx, pool, leserID, verschluesselt); err != nil {
		return "", fmt.Errorf("fehler beim speichern des fotos in der db: %w", err)
	}

	if barcodeID == "" {
		return "", nil
	}
	fotoURL := fmt.Sprintf("/api/schueler/%s/photo", barcodeID)
	return fotoURL, nil
}
