package inventur

import (
	"bytes"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/pashagolub/pgxmock/v5"
)

// createDummyImage liefert die Bytes eines Bildes im genannten Format.
func createDummyImage(format string, breite, hoehe int) []byte {
	bild := image.NewRGBA(image.Rect(0, 0, breite, hoehe))
	for y := 0; y < hoehe; y++ {
		for x := 0; x < breite; x++ {
			bild.Set(x, y, color.RGBA{255, 0, 0, 255})
		}
	}

	var puffer bytes.Buffer
	switch format {
	case "png":
		_ = png.Encode(&puffer, bild) //nolint:errcheck
	case "jpeg", "jpg":
		_ = jpeg.Encode(&puffer, bild, nil) //nolint:errcheck
	}
	return puffer.Bytes()
}

// createMultipartRequest baut eine Anfrage multipart mit einer Bilddatei.
func createMultipartRequest(t *testing.T, feldname, dateiname, format string, daten []byte) *http.Request {
	t.Helper()
	var b bytes.Buffer
	w := multipart.NewWriter(&b)

	fw, err := w.CreateFormFile(feldname, dateiname)
	if err != nil {
		t.Fatalf("failed to create form file: %v", err)
	}
	if _, err := fw.Write(daten); err != nil {
		t.Fatalf("failed to write data: %v", err)
	}
	_ = w.Close() //nolint:errcheck

	anfrage := httptest.NewRequest(http.MethodPost, "/api/books/123/cover-upload", &b)
	anfrage.Header.Set("Content-Type", w.FormDataContentType())
	return anfrage
}

func TestVerarbeiteHochgeladenesBild(t *testing.T) {
	t.Run("Valid JPEG", func(t *testing.T) {
		bildDaten := createDummyImage("jpeg", 100, 150)
		ergebnis, ext, err := verarbeiteHochgeladenesBild(bildDaten, "test-id")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if ext != ".jpg" {
			t.Errorf("expected .jpg extension, got %s", ext)
		}
		if len(ergebnis) == 0 {
			t.Error("expected non-empty image data")
		}
	})

	t.Run("Valid PNG", func(t *testing.T) {
		bildDaten := createDummyImage("png", 100, 150)
		ergebnis, ext, err := verarbeiteHochgeladenesBild(bildDaten, "test-id")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if ext != ".png" {
			t.Errorf("expected .png extension, got %s", ext)
		}
		if len(ergebnis) == 0 {
			t.Error("expected non-empty image data")
		}
	})

	t.Run("Invalid Format", func(t *testing.T) {
		bildDaten := []byte("not an image")
		_, _, err := verarbeiteHochgeladenesBild(bildDaten, "test-id")
		if err == nil {
			t.Fatal("expected error for invalid image data, got nil")
		}
	})

	t.Run("Resize Large Image", func(t *testing.T) {
		bildDaten := createDummyImage("jpeg", 1200, 1800)
		ergebnis, ext, err := verarbeiteHochgeladenesBild(bildDaten, "test-id")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if ext != ".jpg" {
			t.Errorf("expected .jpg extension, got %s", ext)
		}

		bild, format, err := image.Decode(bytes.NewReader(ergebnis))
		if err != nil {
			t.Fatalf("failed to decode result image: %v", err)
		}
		if format != "jpeg" {
			t.Errorf("expected jpeg format, got %s", format)
		}
		bounds := bild.Bounds()
		if bounds.Dx() > 600 || bounds.Dy() > 900 {
			t.Errorf("image was not properly resized. got %dx%d", bounds.Dx(), bounds.Dy())
		}
	})
}

func TestLeseCoverUpload(t *testing.T) {
	bildDaten := createDummyImage("jpeg", 100, 100)

	t.Run("Valid Upload", func(t *testing.T) {
		anfrage := createMultipartRequest(t, "cover", "test.jpg", "jpeg", bildDaten)
		w := httptest.NewRecorder()

		daten, ok := leseCoverUpload(w, anfrage, "123")
		if !ok {
			t.Fatal("leseCoverUpload() failed unexpectedly")
		}
		if !bytes.Equal(daten, bildDaten) {
			t.Error("read data does not match uploaded data")
		}
	})

	t.Run("Invalid Field Name", func(t *testing.T) {
		anfrage := createMultipartRequest(t, "wrong_field", "test.jpg", "jpeg", bildDaten)
		w := httptest.NewRecorder()

		_, ok := leseCoverUpload(w, anfrage, "123")
		if ok {
			t.Error("expected leseCoverUpload() to fail with wrong field name")
		}
		if w.Code != http.StatusBadRequest {
			t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
		}
	})

	t.Run("Invalid Extension", func(t *testing.T) {
		anfrage := createMultipartRequest(t, "cover", "test.txt", "txt", bildDaten)
		w := httptest.NewRecorder()

		_, ok := leseCoverUpload(w, anfrage, "123")
		if ok {
			t.Error("expected leseCoverUpload() to fail with invalid extension")
		}
		if w.Code != http.StatusBadRequest {
			t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
		}
	})

	t.Run("Empty File", func(t *testing.T) {
		anfrage := createMultipartRequest(t, "cover", "test.jpg", "jpeg", []byte{})
		w := httptest.NewRecorder()

		_, ok := leseCoverUpload(w, anfrage, "123")
		if ok {
			t.Error("expected leseCoverUpload() to fail with empty file")
		}
		if w.Code != http.StatusBadRequest {
			t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
		}
	})
}

func TestSpeichereHochgeladenesCover(t *testing.T) {
	// Das Arbeitsverzeichnis bleibt, wie es ist (das bräche das parallele Laufen): Die Funktion legt
	// `uploads/` im aktuellen Verzeichnis an, für `go test` das Verzeichnis des Pakets; die erzeugte
	// Datei wird nach dem Test gelöscht.

	t.Run("Valid Save", func(t *testing.T) {
		w := httptest.NewRecorder()
		bildDaten := createDummyImage("jpeg", 100, 100)
		id := "test-id"

		adresse, ok := speichereHochgeladenesCover(w, id, bildDaten, ".jpg")

		if !ok {
			t.Fatalf("speichereHochgeladenesCover() failed unexpectedly. Status: %d, Body: %s", w.Code, w.Body.String())
		}

		if !strings.HasPrefix(adresse, "/uploads/") {
			t.Errorf("expected URL to start with /uploads/, got %s", adresse)
		}

		if !strings.HasSuffix(adresse, ".jpg") {
			t.Errorf("expected URL to end with .jpg, got %s", adresse)
		}

		// Die Datei muss bestehen.
		dateiname := filepath.Base(adresse)
		dateipfad := filepath.Join("uploads", dateiname)
		if _, err := os.Stat(dateipfad); os.IsNotExist(err) {
			t.Errorf("expected file %s to be created, but it was not", dateipfad)
		}

		// Die erzeugte Datei löschen.
		defer func() { _ = os.Remove(dateipfad) }() //nolint:errcheck
	})

	t.Run("Path Traversal Protection", func(t *testing.T) {
		w := httptest.NewRecorder()
		bildDaten := createDummyImage("jpeg", 100, 100)

		// filepath.Base in der Tür macht daraus einen harmlosen Namen.
		id := "../../../etc/passwd"

		adresse, ok := speichereHochgeladenesCover(w, id, bildDaten, ".jpg")

		if !ok {
			t.Fatalf("speichereHochgeladenesCover() failed unexpectedly")
		}

		// Die Adresse darf keinen Weg aus dem Verzeichnis heraus enthalten.
		if strings.Contains(adresse, "..") {
			t.Errorf("expected URL to not contain path traversal elements, got %s", adresse)
		}

		dateiname := filepath.Base(adresse)
		if strings.Contains(dateiname, "passwd") { // since filepath.Base("../../../etc/passwd") is "passwd"
			// Übrig bleibt "passwd", sicher innerhalb von uploads.
			dateipfad := filepath.Join("uploads", dateiname)
			if !strings.HasPrefix(filepath.Clean(dateipfad), filepath.Clean("uploads")) {
				t.Errorf("file path escapes uploads directory: %s", dateipfad)
			}

			// Die erzeugte Datei löschen.
			defer func() { _ = os.Remove(dateipfad) }() //nolint:errcheck
		}
	})
}

// uploadTestBuchID: eine UUID, weil buchIDAusPfad alles andere mit 400 abweist. Die
// Upload-Dateien heißen danach (cover_<id>_…), das Aufräumen unten sucht sie so.
const uploadTestBuchID = "6ba7b810-9dad-41d1-80b4-00c04fd430c8"

func TestCoverHochladen(t *testing.T) {
	faelle := []struct {
		name           string
		method         string
		id             string
		bodyData       []byte
		dateiname      string
		contentType    string
		setupMock      func(pgxmock.PgxPoolIface)
		expectedStatus int
	}{
		{
			name:           "Kennung ist keine UUID",
			method:         http.MethodPost,
			id:             "123",
			bodyData:       createDummyImage("jpeg", 100, 100),
			dateiname:      "test.jpg",
			contentType:    "multipart/form-data",
			setupMock:      func(m pgxmock.PgxPoolIface) {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "Empty ID",
			method:         http.MethodPost,
			id:             "",
			bodyData:       nil,
			dateiname:      "",
			contentType:    "",
			setupMock:      func(m pgxmock.PgxPoolIface) {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "Missing File Data",
			method:         http.MethodPost,
			id:             uploadTestBuchID,
			bodyData:       nil,
			dateiname:      "",
			contentType:    "multipart/form-data",
			setupMock:      func(m pgxmock.PgxPoolIface) {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:        "Success Upload",
			method:      http.MethodPost,
			id:          uploadTestBuchID,
			bodyData:    createDummyImage("jpeg", 100, 100),
			dateiname:   "test.jpg",
			contentType: "multipart/form-data",
			setupMock: func(m pgxmock.PgxPoolIface) {
				m.ExpectQuery("(?s)SELECT id, COALESCE.*").
					WithArgs(uploadTestBuchID).
					WillReturnRows(pgxmock.NewRows([]string{
						"id", "isbn", "title", "author", "signatur", "cover_url", "subject", "track", "stock", "last_counted", "sort_order", "medientyp", "jahrgang_von", "jahrgang_bis", "erweiterte_eigenschaften", "auflage",
					}).AddRow(
						uploadTestBuchID, "9781234567890", "Test Title", "Test Author", "", "/uploads/old_cover.jpg", "", "", 1, nil, 1, "Buch", 5, 10, nil, "",
					))

				m.ExpectExec("(?s)UPDATE buecher_titel.*").
					WithArgs("", "", pgxmock.AnyArg(), uploadTestBuchID).
					WillReturnResult(pgxmock.NewResult("UPDATE", 1))
			},
			expectedStatus: http.StatusOK,
		},
		{
			// Das Buch fehlt: 404, bevor eine Datei geschrieben oder ein UPDATE geschickt
			// wird (cover_upload_reihenfolge_test.go).
			name:        "Buch unbekannt",
			method:      http.MethodPost,
			id:          uploadTestBuchID,
			bodyData:    createDummyImage("jpeg", 100, 100),
			dateiname:   "test.jpg",
			contentType: "multipart/form-data",
			setupMock: func(m pgxmock.PgxPoolIface) {
				m.ExpectQuery("(?s)SELECT id, COALESCE.*").
					WithArgs(uploadTestBuchID).
					WillReturnError(pgx.ErrNoRows)
			},
			expectedStatus: http.StatusNotFound,
		},
	}

	for _, fall := range faelle {
		t.Run(fall.name, func(t *testing.T) {
			mock, err := pgxmock.NewPool()
			if err != nil {
				t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
			}
			defer mock.Close()

			abfragen := NewTitelAbfragen(mock)
			tueren := &Tueren{abfragen: abfragen}

			fall.setupMock(mock)

			pfad := "/api/books/" + fall.id + "/cover-upload"
			var anfrage *http.Request
			if fall.contentType == "multipart/form-data" && len(fall.bodyData) > 0 {
				var b bytes.Buffer
				w := multipart.NewWriter(&b)
				fw, _ := w.CreateFormFile("cover", fall.dateiname) //nolint:errcheck
				_, _ = fw.Write(fall.bodyData)                     //nolint:errcheck
				_ = w.Close()                                      //nolint:errcheck

				anfrage = httptest.NewRequest(fall.method, pfad, &b)
				anfrage.Header.Set("Content-Type", w.FormDataContentType())
			} else if fall.contentType == "multipart/form-data" && len(fall.bodyData) == 0 {
				var b bytes.Buffer
				w := multipart.NewWriter(&b)
				_ = w.Close() //nolint:errcheck
				anfrage = httptest.NewRequest(fall.method, pfad, &b)
				anfrage.Header.Set("Content-Type", w.FormDataContentType())
			} else {
				anfrage = httptest.NewRequest(fall.method, pfad, nil)
			}
			// Den Platzhalter {id} füllt im Betrieb der Mux (api_routen.go).
			anfrage.SetPathValue("id", fall.id)

			w := httptest.NewRecorder()

			tueren.CoverHochladen(w, anfrage)

			if w.Code != fall.expectedStatus {
				t.Errorf("expected status %d, got %d. Body: %s", fall.expectedStatus, w.Code, w.Body.String())
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("there were unfulfilled expectations: %s", err)
			}
		})
	}

	// Clean up created upload files during tests
	dateipfade, _ := filepath.Glob("uploads/cover_" + uploadTestBuchID + "_*.jpg") //nolint:errcheck
	for _, f := range dateipfade {
		_ = os.Remove(f) //nolint:errcheck
	}
}
