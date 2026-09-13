package inventur

import (
	"bibliothek/pkg/imageutil"
	"bibliothek/pkg/logger"
	"bytes"
	"errors"
	"fmt"
	"image"
	"image/jpeg"
	_ "image/png"
	"io"
	"log"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"golang.org/x/image/draw"
	_ "golang.org/x/image/webp"
)

const (
	maxCoverUploadBytes = 10 << 20 // 10 MB
	maxCoverWidth       = 600
	maxCoverHeight      = 900
	coverJPEGQuality    = 82
)

func processUploadedImage(fileBytes []byte, id string) ([]byte, string, error) {
	// Decompression-Bomb-Schutz: Dimensionen anhand des Headers prüfen, bevor die
	// vollständige Pixelmatrix alloziert wird (image.Decode würde sonst sofort
	// width×height×4 Byte reservieren — bei manipulierten Bildern Gigabytes).
	if err := imageutil.GuardImageDimensions(fileBytes); err != nil {
		return nil, "", err
	}
	img, format, err := image.Decode(bytes.NewReader(fileBytes))
	if err != nil {
		return nil, "", errors.New("ungültiges bildformat: muss jpg, png oder webp sein")
	}
	if format != "jpeg" && format != "png" && format != "webp" {
		return nil, "", errors.New("ungültiges bildformat: muss jpg, png oder webp sein")
	}

	bounds := img.Bounds()
	width, height := bounds.Dx(), bounds.Dy()

	finalBytes := fileBytes
	saveExt := ""
	switch format {
	case "jpeg":
		saveExt = ".jpg"
	case "png":
		saveExt = ".png"
	case "webp":
		saveExt = ".webp"
	default:
		saveExt = ".jpg"
	}

	// Bild verkleinern, falls es die Maximalmaße überschreitet
	if width > maxCoverWidth || height > maxCoverHeight {
		ratio := float64(width) / float64(height)
		newWidth, newHeight := width, height

		if newWidth > maxCoverWidth {
			newWidth = maxCoverWidth
			newHeight = int(float64(newWidth) / ratio)
		}
		if newHeight > maxCoverHeight {
			newHeight = maxCoverHeight
			newWidth = int(float64(newHeight) * ratio)
		}

		dst := image.NewRGBA(image.Rect(0, 0, newWidth, newHeight))
		draw.ApproxBiLinear.Scale(dst, dst.Bounds(), img, bounds, draw.Over, nil)

		var buf bytes.Buffer
		err = jpeg.Encode(&buf, dst, &jpeg.Options{Quality: coverJPEGQuality})
		if err != nil {
			log.Printf("cover-upload: jpeg encode failed for book %s: %v", logger.SanitizeLog(id), err)
			return nil, "", fmt.Errorf("fehler bei der bildverarbeitung: %w", err)
		}
		finalBytes = buf.Bytes()
		saveExt = ".jpg" // Wenn wir serverseitig verkleinern, speichern wir mangels WebP-Encoder als JPG
	}

	return finalBytes, saveExt, nil
}

// loescheCoverDatei entfernt ein Cover aus uploads/, wenn coverURL dorthin zeigt. Fehler
// werden nur protokolliert: Aufräumen darf die Antwort an den Aufrufer nicht ändern.
func loescheCoverDatei(coverURL string) {
	if !strings.HasPrefix(coverURL, "/uploads/") {
		return
	}
	filename := filepath.Base(coverURL)
	if filename == "" || filename == "/" || filename == "." {
		return
	}
	if err := loescheUploadDatei(filename); err != nil {
		log.Printf("cover-upload: Cover %s konnte nicht entfernt werden: %v", logger.SanitizeLog(filename), err)
	}
}

func (handler *APIHandler) handleUploadCover(writer http.ResponseWriter, request *http.Request) {
	id, ok := buchIDAusPfad(writer, request)
	if !ok {
		return
	}

	fileBytes, ok := readCoverUpload(writer, request, id)
	if !ok {
		return
	}

	finalBytes, saveExt, err := processUploadedImage(fileBytes, id)
	if err != nil {
		if strings.HasPrefix(err.Error(), "fehler bei der bildverarbeitung") {
			writeError(writer, http.StatusInternalServerError, "fehler bei der bildverarbeitung")
		} else {
			writeError(writer, http.StatusBadRequest, err.Error())
		}
		return
	}

	// Reihenfolge: erst das Buch, dann die Datei, dann das UPDATE, zuletzt das alte Cover.
	// Bis zum 13.09.2026 lag die neue Datei schon auf der Platte, wenn sich herausstellte,
	// dass es das Buch nicht gibt, und das alte Cover war schon gelöscht, wenn das UPDATE
	// scheiterte — die Datenbank zeigte dann auf eine Datei, die fehlte
	// (cover_upload_reihenfolge_test.go).
	buch, err := handler.repo.GetBookByID(request.Context(), id)
	if errors.Is(err, ErrBookNotFound) {
		writeError(writer, http.StatusNotFound, "buch nicht gefunden")
		return
	}
	if err != nil {
		log.Printf("cover-upload: buch %s laden: %v", logger.SanitizeLog(id), err)
		writeError(writer, http.StatusInternalServerError, "buch konnte nicht geladen werden")
		return
	}

	coverURL, ok := saveCoverFile(writer, id, finalBytes, saveExt)
	if !ok {
		return
	}

	// Scheitert das UPDATE, geht die neue Datei wieder weg — sonst läge sie verwaist in
	// uploads/, während die Datenbank weiter auf das alte Cover zeigt.
	err = handler.repo.UpdateBookMetadata(request.Context(), id, "", "", coverURL)
	if errors.Is(err, ErrBookNotFound) {
		loescheCoverDatei(coverURL)
		writeError(writer, http.StatusNotFound, "buch nicht gefunden")
		return
	}
	if err != nil {
		loescheCoverDatei(coverURL)
		log.Printf("cover-upload: metadata update failed for book %s: %v", logger.SanitizeLog(id), err)
		writeError(writer, http.StatusInternalServerError, "metadaten konnten nicht gespeichert werden")
		return
	}

	// Zwei Uploads in derselben Sekunde tragen denselben Dateinamen (cover_<id>_<Sekunde>):
	// Dann IST das alte Cover das neue und darf nicht gelöscht werden.
	if buch.CoverURL != coverURL {
		loescheCoverDatei(buch.CoverURL)
	}

	writeJSON(writer, http.StatusOK, map[string]any{
		"message": "bild hochgeladen",
		"data": map[string]any{
			"id":       id,
			"coverUrl": coverURL,
		},
	})
}

// readCoverUpload liest und validiert das hochgeladene Bild (Größe, Nicht-Leer,
// zulässige Endung). ok=false: die Fehlerantwort wurde bereits geschrieben.
func readCoverUpload(writer http.ResponseWriter, request *http.Request, id string) ([]byte, bool) {
	request.Body = http.MaxBytesReader(writer, request.Body, maxCoverUploadBytes)
	if err := request.ParseMultipartForm(maxCoverUploadBytes); err != nil {
		log.Printf("cover-upload: multipart parse failed for book %s: %v", logger.SanitizeLog(id), err)
		writeError(writer, http.StatusBadRequest, "datei zu groß oder ungültig (max. 10 MB)")
		return nil, false
	}

	file, header, err := request.FormFile("cover")
	if err != nil {
		writeError(writer, http.StatusBadRequest, "kein bild gefunden")
		return nil, false
	}
	defer func() { _ = file.Close() }() //nolint:errcheck

	fileBytes, err := io.ReadAll(file)
	if err != nil {
		log.Printf("cover-upload: read failed for book %s: %v", logger.SanitizeLog(id), err)
		writeError(writer, http.StatusInternalServerError, "fehler beim lesen der datei")
		return nil, false
	}
	if len(fileBytes) == 0 {
		writeError(writer, http.StatusBadRequest, "leere datei")
		return nil, false
	}

	ext := strings.ToLower(filepath.Ext(header.Filename))
	if ext != ".jpg" && ext != ".jpeg" && ext != ".png" && ext != ".webp" {
		writeError(writer, http.StatusBadRequest, "nur jpg, jpeg, png oder webp erlaubt")
		return nil, false
	}
	return fileBytes, true
}

// saveCoverFile legt das uploads-Verzeichnis an, schreibt die Bilddatei unter einem
// serverseitig generierten (traversal-geschützten) Pfad und liefert die öffentliche
// Cover-URL. ok=false: die Fehlerantwort wurde bereits geschrieben.
func saveCoverFile(writer http.ResponseWriter, id string, finalBytes []byte, saveExt string) (string, bool) {
	filename := fmt.Sprintf("cover_%s_%d%s", filepath.Base(id), time.Now().Unix(), saveExt)

	if err := schreibeUploadDatei(filename, finalBytes); err != nil {
		log.Printf("cover-upload: speichern fehlgeschlagen für Buch %s: %v", logger.SanitizeLog(id), err)
		writeError(writer, http.StatusInternalServerError, "fehler beim speichern")
		return "", false
	}

	return fmt.Sprintf("/uploads/%s", filename), true
}
