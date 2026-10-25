package katalogdienste

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"path/filepath"

	"bibliothek/pkg/coverablage"
	"bibliothek/pkg/coverquelle"
)

// speichereCoverDatei schreibt das aufbereitete Cover in uploads/ und liefert den
// öffentlichen Pfad; "" bei Fehler. Der Name trägt die ISBN und eine Prüfsumme des Inhalts:
// Dieselbe Abfrage legt dieselbe Datei ab statt bei jedem Mal eine weitere, und ein anderes
// Bild bekommt einen anderen Namen, sodass kein Browser das alte weiter zeigt.
func speichereCoverDatei(fertig []byte, isbn, endung string) string {
	summe := sha256.Sum256(fertig)
	dateiname := fmt.Sprintf("cover_auto_%s_%x%s", filepath.Base(isbn), summe[:6], endung)

	if coverablage.IstVorhanden(dateiname) {
		return "/uploads/" + dateiname
	}
	if err := coverablage.Schreibe(dateiname, fertig); err != nil {
		slog.Error("Cover-Download: Datei nicht gespeichert", "fehler", err)
		return "" // kein externer Fallback: lieber leer lassen und später erneut versuchen
	}
	return "/uploads/" + dateiname
}

// speichereCoverLokal holt das Cover von einer Adresse im Netz, legt es in uploads/ ab und
// liefert den lokalen Pfad; "" heißt, die nächste Quelle ist dran. Die Regeln des Abrufs stehen
// in coverquelle.Hole und gelten ebenso für die Anzeige im Katalog.
func speichereCoverLokal(ctx context.Context, client *http.Client, coverURL string, isbn string) string {
	fertig, err := coverquelle.Hole(ctx, client, coverURL, coverquelle.IstErsatzbild)
	if err != nil {
		if !errors.Is(err, coverquelle.ErrKeinCover) {
			slog.Info("Cover-Download: kein Cover übernommen", "url", coverURL, "fehler", err)
		}
		return ""
	}
	return speichereCoverDatei(fertig, isbn, ".webp")
}
