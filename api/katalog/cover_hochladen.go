package katalog

import (
	"bytes"
	"errors"
	"fmt"
	"image"
	"image/jpeg"
	"io"
	"log/slog"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"bibliothek/api/tuer"
	"bibliothek/apierrors"
	"bibliothek/pkg/coverablage"
	"bibliothek/pkg/imageutil"
	"bibliothek/pkg/logger"
	"bibliothek/repository"
)

const (
	maxCoverUploadBytes = 10 << 20 // 10 MB
	coverJPEGQualitaet  = 82
)

// errBildverarbeitung heißt: Das Bild ließ sich lesen, aber nicht verkleinert speichern. Eine
// Störung am Server (500), kein unbrauchbares Bild (400).
var errBildverarbeitung = errors.New("fehler bei der bildverarbeitung")

// meldungBildUngueltig antwortet auf ein Bild, das sich nicht lesen lässt, zu groß ist oder ein
// anderes Format hat.
const meldungBildUngueltig = "Das Bild ließ sich nicht lesen oder ist zu groß. Möglich sind JPG, PNG oder WebP bis 50 Megapixel."

func verarbeiteHochgeladenesBild(inhalt []byte, id string) ([]byte, string, error) {
	// Decompression-Bomb-Schutz: Dimensionen anhand des Headers prüfen, bevor die
	// vollständige Pixelmatrix alloziert wird (image.Decode würde sonst sofort
	// width×height×4 Byte reservieren — bei manipulierten Bildern Gigabytes).
	if err := imageutil.PruefeBildgroesse(inhalt); err != nil {
		return nil, "", apierrors.BadRequest(meldungBildUngueltig, err)
	}
	bild, format, err := image.Decode(bytes.NewReader(inhalt))
	if err != nil {
		return nil, "", apierrors.BadRequest(meldungBildUngueltig, err)
	}
	if format != "jpeg" && format != "png" && format != "webp" {
		return nil, "", apierrors.BadRequest(meldungBildUngueltig, nil)
	}

	rahmen := bild.Bounds()
	breite, hoehe := rahmen.Dx(), rahmen.Dy()

	fertig := inhalt
	endung := ""
	switch format {
	case "jpeg":
		endung = ".jpg"
	case "png":
		endung = ".png"
	case "webp":
		endung = ".webp"
	default:
		endung = ".jpg"
	}

	// Bild verkleinern, falls es die Maximalmaße überschreitet
	if breite > coverablage.MaxBreite || hoehe > coverablage.MaxHoehe {
		ziel := imageutil.Verkleinere(bild, coverablage.MaxBreite, coverablage.MaxHoehe)

		var puffer bytes.Buffer
		err = jpeg.Encode(&puffer, ziel, &jpeg.Options{Quality: coverJPEGQualitaet})
		if err != nil {
			slog.Error("cover-upload: jpeg encode failed", "titel", logger.OhneUmbruch(id), "fehler", err)
			return nil, "", fmt.Errorf("%w: %w", errBildverarbeitung, err)
		}
		fertig = puffer.Bytes()
		endung = ".jpg" // Wenn wir serverseitig verkleinern, speichern wir mangels WebP-Encoder als JPG
	}

	return fertig, endung, nil
}

// loescheCoverDatei entfernt ein Cover aus uploads/, wenn coverURL dorthin zeigt
// (coverablage.Loesche). Fehler werden nur protokolliert: Aufräumen darf die Antwort an den
// Aufrufer nicht ändern.
func loescheCoverDatei(coverURL string) {
	if err := coverablage.Loesche(coverURL); err != nil {
		slog.Warn("cover-upload: Cover konnte nicht entfernt werden", "datei", logger.OhneUmbruch(coverURL), "fehler", err)
	}
}

// CoverHochladen ist die Tür POST /api/books/{id}/cover-upload: Sie nimmt ein Bild als Cover
// eines Titels an.
func (t *Tueren) CoverHochladen() http.HandlerFunc {
	return t.schreibeHochgeladenesCover
}

// schreibeHochgeladenesCover ist der Rumpf von CoverHochladen. Er steht auf der obersten Ebene,
// weil eine Closure für die Komplexitätsmessung als eigene Ebene zählt.
func (t *Tueren) schreibeHochgeladenesCover(w http.ResponseWriter, r *http.Request) {
	id, ok := titelKennungAusPfad(w, r)
	if !ok {
		return
	}

	inhalt, ok := leseCoverUpload(w, r, id)
	if !ok {
		return
	}

	fertig, endung, err := verarbeiteHochgeladenesBild(inhalt, id)
	if err != nil {
		if errors.Is(err, errBildverarbeitung) {
			apierrors.AntworteFehlerMitMeldung(w, http.StatusInternalServerError, "Das Bild ließ sich nicht verarbeiten", err)
		} else {
			apierrors.AntworteFehler(w, http.StatusBadRequest, err)
		}
		return
	}

	// Reihenfolge: erst der Titel, dann die Datei, dann das UPDATE, zuletzt das alte Cover. Sonst
	// läge die neue Datei schon auf der Platte, wenn sich herausstellt, dass es den Titel nicht
	// gibt, und das alte Cover wäre schon gelöscht, wenn das UPDATE scheitert; die Datenbank zeigte
	// dann auf eine Datei, die fehlt (cover_upload_reihenfolge_test.go).
	buch, err := t.titelAbfragen().LadeTitel(r.Context(), id)
	if errors.Is(err, repository.ErrTitelNichtGefunden) {
		apierrors.AntworteFehlerMitMeldung(w, http.StatusNotFound, "Titel nicht gefunden", nil)
		return
	}
	if err != nil {
		apierrors.AntworteFehlerMitMeldung(w, http.StatusInternalServerError, "buch konnte nicht geladen werden", fmt.Errorf("cover hochladen, titel laden: %w", err))
		return
	}

	coverURL, ok := speichereHochgeladenesCover(w, id, fertig, endung)
	if !ok {
		return
	}

	// Scheitert das UPDATE, geht die neue Datei wieder weg — sonst läge sie verwaist in
	// uploads/, während die Datenbank weiter auf das alte Cover zeigt.
	err = t.titelAbfragen().AendereTitelMetadaten(r.Context(), id, "", "", coverURL)
	if errors.Is(err, repository.ErrTitelNichtGefunden) {
		loescheCoverDatei(coverURL)
		apierrors.AntworteFehlerMitMeldung(w, http.StatusNotFound, "Titel nicht gefunden", nil)
		return
	}
	if err != nil {
		loescheCoverDatei(coverURL)
		apierrors.AntworteFehlerMitMeldung(w, http.StatusInternalServerError, "metadaten konnten nicht gespeichert werden", fmt.Errorf("cover hochladen, angaben speichern: %w", err))
		return
	}

	// Zwei Uploads in derselben Sekunde tragen denselben Dateinamen (cover_<id>_<Sekunde>):
	// Dann ist das alte Cover das neue und darf nicht gelöscht werden.
	if buch.CoverURL != coverURL {
		loescheCoverDatei(buch.CoverURL)
	}

	tuer.AntworteJSON(w, http.StatusOK, map[string]string{"id": id, "coverUrl": coverURL})

}

// leseCoverUpload liest und validiert das hochgeladene Bild (Größe, Nicht-Leer,
// zulässige Endung). ok=false: die Fehlerantwort wurde bereits geschrieben.
func leseCoverUpload(w http.ResponseWriter, r *http.Request, id string) ([]byte, bool) {
	r.Body = http.MaxBytesReader(w, r.Body, maxCoverUploadBytes)
	if err := r.ParseMultipartForm(maxCoverUploadBytes); err != nil {
		slog.Warn("cover-upload: multipart parse failed", "titel", logger.OhneUmbruch(id), "fehler", err)
		apierrors.AntworteFehlerMitMeldung(w, http.StatusBadRequest, "Die Datei ist zu groß oder ungültig, höchstens 10 MB", nil)
		return nil, false
	}

	datei, kopf, err := r.FormFile("cover")
	if err != nil {
		apierrors.AntworteFehlerMitMeldung(w, http.StatusBadRequest, "Kein Bild gefunden", nil)
		return nil, false
	}
	defer func() { _ = datei.Close() }() //nolint:errcheck

	inhalt, err := io.ReadAll(datei)
	if err != nil {
		slog.Warn("cover-upload: read failed", "titel", logger.OhneUmbruch(id), "fehler", err)
		apierrors.AntworteFehlerMitMeldung(w, http.StatusInternalServerError, "fehler beim lesen der datei", nil)
		return nil, false
	}
	if len(inhalt) == 0 {
		apierrors.AntworteFehlerMitMeldung(w, http.StatusBadRequest, "Die Datei ist leer", nil)
		return nil, false
	}

	endung := strings.ToLower(filepath.Ext(kopf.Filename))
	if endung != ".jpg" && endung != ".jpeg" && endung != ".png" && endung != ".webp" {
		apierrors.AntworteFehlerMitMeldung(w, http.StatusBadRequest, "Nur JPG, PNG oder WebP", nil)
		return nil, false
	}
	return inhalt, true
}

// speichereHochgeladenesCover legt das uploads-Verzeichnis an, schreibt die Bilddatei unter einem
// serverseitig generierten (traversal-geschützten) Pfad und liefert die öffentliche
// Cover-URL. ok=false: die Fehlerantwort wurde bereits geschrieben.
func speichereHochgeladenesCover(w http.ResponseWriter, id string, fertig []byte, endung string) (string, bool) {
	dateiname := fmt.Sprintf("cover_%s_%d%s", filepath.Base(id), time.Now().Unix(), endung)

	if err := coverablage.Schreibe(dateiname, fertig); err != nil {
		apierrors.AntworteFehlerMitMeldung(w, http.StatusInternalServerError, "fehler beim speichern", fmt.Errorf("cover hochladen, datei speichern: %w", err))
		return "", false
	}

	return fmt.Sprintf("/uploads/%s", dateiname), true
}
