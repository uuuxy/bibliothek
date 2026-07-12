package inventur

import (
	"bibliothek/pkg/imageutil"
	"bibliothek/pkg/logger"
	"bytes"
	"context"
	"errors"
	"fmt"
	"image"
	"image/jpeg"
	_ "image/png"
	"io"
	"log"
	"net/http"
	"os"
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

func deleteOldCoverFile(ctx context.Context, handler *APIHandler, id string) {
	altesBook, abfrageErr := handler.repo.GetBookByID(ctx, id)
	if abfrageErr == nil && altesBook != nil && strings.HasPrefix(altesBook.CoverURL, "/uploads/") {
		filename := filepath.Base(altesBook.CoverURL)
		if filename != "" && filename != "/" && filename != "." {
			cleanDir := filepath.Clean("uploads")
			alterPfad := filepath.Clean(filepath.Join(cleanDir, filename))
			if strings.HasPrefix(alterPfad, cleanDir+string(filepath.Separator)) {
				_ = os.Remove(alterPfad) // Fehler ignorieren (Datei existiert ggf. nicht mehr)
			}
		}
	}
}

func (handler *APIHandler) handleUploadCover(writer http.ResponseWriter, request *http.Request) {
	id, ok := validateCoverRoute(writer, request)
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

	coverURL, ok := saveCoverFile(writer, id, finalBytes, saveExt)
	if !ok {
		return
	}

	deleteOldCoverFile(request.Context(), handler, id)

	err = handler.repo.UpdateBookMetadata(request.Context(), id, "", "", coverURL)
	if err != nil {
		if errors.Is(err, ErrBookNotFound) {
			writeError(writer, http.StatusNotFound, "buch nicht gefunden")
			return
		}
		log.Printf("cover-upload: metadata update failed for book %s: %v", logger.SanitizeLog(id), err)
		writeError(writer, http.StatusInternalServerError, "metadaten konnten nicht gespeichert werden")
		return
	}

	writeJSON(writer, http.StatusOK, map[string]any{
		"message": "bild hochgeladen",
		"data": map[string]any{
			"id":       id,
			"coverUrl": coverURL,
		},
	})
}

// validateCoverRoute validiert die Upload-Route (/api/books/{id}/cover-upload) und
// extrahiert die Buch-ID. ok=false: die Fehlerantwort wurde bereits geschrieben.
func validateCoverRoute(writer http.ResponseWriter, request *http.Request) (string, bool) {
	parts := strings.Split(strings.Trim(request.URL.Path, "/"), "/")
	if len(parts) != 4 || parts[0] != "api" || parts[1] != "books" || parts[3] != "cover-upload" {
		writeError(writer, http.StatusBadRequest, "ungültige route")
		return "", false
	}

	id := filepath.Base(parts[2])
	if id == "" || id == "." || id == "/" {
		writeError(writer, http.StatusBadRequest, "id darf nicht leer sein")
		return "", false
	}
	return id, true
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
	defer func() { _ = file.Close() }()

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
	if err := os.MkdirAll("uploads", 0750); err != nil {
		log.Printf("cover-upload: mkdir uploads failed for book %s: %v", logger.SanitizeLog(id), err)
		writeError(writer, http.StatusInternalServerError, "uploads-verzeichnis konnte nicht erstellt werden")
		return "", false
	}

	cleanDir := filepath.Clean("uploads")
	filename := fmt.Sprintf("cover_%s_%d%s", filepath.Base(id), time.Now().Unix(), saveExt)
	savePath := filepath.Clean(filepath.Join(cleanDir, filename))

	if !strings.HasPrefix(savePath, cleanDir+string(filepath.Separator)) {
		writeError(writer, http.StatusBadRequest, "invalid file path")
		return "", false
	}

	// #nosec G304 - filename is safely generated on the server side
	if err := os.WriteFile(savePath, finalBytes, 0600); err != nil {
		log.Printf("cover-upload: write file failed for book %s (%s): %v", logger.SanitizeLog(id), logger.SanitizeLog(savePath), err)
		writeError(writer, http.StatusInternalServerError, "fehler beim speichern")
		return "", false
	}

	return fmt.Sprintf("/uploads/%s", filename), true
}
