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
	"github.com/pashagolub/pgxmock/v4"
)

// createDummyImage returns the bytes of a generic dummy image in the specified format
func createDummyImage(format string, width, height int) []byte {
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			img.Set(x, y, color.RGBA{255, 0, 0, 255})
		}
	}

	var buf bytes.Buffer
	switch format {
	case "png":
		_ = png.Encode(&buf, img) //nolint:errcheck
	case "jpeg", "jpg":
		_ = jpeg.Encode(&buf, img, nil) //nolint:errcheck
	}
	return buf.Bytes()
}

// createMultipartRequest creates a multipart request with an image file
func createMultipartRequest(t *testing.T, fieldName, fileName, format string, data []byte) *http.Request {
	t.Helper()
	var b bytes.Buffer
	w := multipart.NewWriter(&b)

	fw, err := w.CreateFormFile(fieldName, fileName)
	if err != nil {
		t.Fatalf("failed to create form file: %v", err)
	}
	if _, err := fw.Write(data); err != nil {
		t.Fatalf("failed to write data: %v", err)
	}
	_ = w.Close() //nolint:errcheck

	req := httptest.NewRequest(http.MethodPost, "/api/books/123/cover-upload", &b)
	req.Header.Set("Content-Type", w.FormDataContentType())
	return req
}

func TestProcessUploadedImage(t *testing.T) {
	t.Run("Valid JPEG", func(t *testing.T) {
		imgData := createDummyImage("jpeg", 100, 150)
		result, ext, err := processUploadedImage(imgData, "test-id")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if ext != ".jpg" {
			t.Errorf("expected .jpg extension, got %s", ext)
		}
		if len(result) == 0 {
			t.Error("expected non-empty image data")
		}
	})

	t.Run("Valid PNG", func(t *testing.T) {
		imgData := createDummyImage("png", 100, 150)
		result, ext, err := processUploadedImage(imgData, "test-id")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if ext != ".png" {
			t.Errorf("expected .png extension, got %s", ext)
		}
		if len(result) == 0 {
			t.Error("expected non-empty image data")
		}
	})

	t.Run("Invalid Format", func(t *testing.T) {
		imgData := []byte("not an image")
		_, _, err := processUploadedImage(imgData, "test-id")
		if err == nil {
			t.Fatal("expected error for invalid image data, got nil")
		}
	})

	t.Run("Resize Large Image", func(t *testing.T) {
		imgData := createDummyImage("jpeg", 1200, 1800)
		result, ext, err := processUploadedImage(imgData, "test-id")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if ext != ".jpg" {
			t.Errorf("expected .jpg extension, got %s", ext)
		}

		img, format, err := image.Decode(bytes.NewReader(result))
		if err != nil {
			t.Fatalf("failed to decode result image: %v", err)
		}
		if format != "jpeg" {
			t.Errorf("expected jpeg format, got %s", format)
		}
		bounds := img.Bounds()
		if bounds.Dx() > 600 || bounds.Dy() > 900 {
			t.Errorf("image was not properly resized. got %dx%d", bounds.Dx(), bounds.Dy())
		}
	})
}

func TestValidateCoverRoute(t *testing.T) {
	tests := []struct {
		name       string
		path       string
		wantID     string
		wantOk     bool
		wantStatus int
	}{
		{
			name:       "Valid Route",
			path:       "/api/books/123/cover-upload",
			wantID:     "123",
			wantOk:     true,
			wantStatus: http.StatusOK,
		},
		{
			name:       "Invalid Route Parts",
			path:       "/api/books/123/other",
			wantID:     "",
			wantOk:     false,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "Empty ID",
			path:       "/api/books//cover-upload",
			wantID:     "",
			wantOk:     false,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "Missing Parts",
			path:       "/api/books/123",
			wantID:     "",
			wantOk:     false,
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, tt.path, nil)
			w := httptest.NewRecorder()

			id, ok := validateCoverRoute(w, req)

			if id != tt.wantID {
				t.Errorf("validateCoverRoute() id = %v, want %v", id, tt.wantID)
			}
			if ok != tt.wantOk {
				t.Errorf("validateCoverRoute() ok = %v, want %v", ok, tt.wantOk)
			}
			if !ok {
				if w.Code != tt.wantStatus {
					t.Errorf("validateCoverRoute() status = %v, want %v", w.Code, tt.wantStatus)
				}
				if !strings.Contains(w.Body.String(), "error") {
					t.Errorf("Expected error response, got %s", w.Body.String())
				}
			}
		})
	}
}

func TestReadCoverUpload(t *testing.T) {
	imgData := createDummyImage("jpeg", 100, 100)

	t.Run("Valid Upload", func(t *testing.T) {
		req := createMultipartRequest(t, "cover", "test.jpg", "jpeg", imgData)
		w := httptest.NewRecorder()

		data, ok := readCoverUpload(w, req, "123")
		if !ok {
			t.Fatal("readCoverUpload() failed unexpectedly")
		}
		if !bytes.Equal(data, imgData) {
			t.Error("read data does not match uploaded data")
		}
	})

	t.Run("Invalid Field Name", func(t *testing.T) {
		req := createMultipartRequest(t, "wrong_field", "test.jpg", "jpeg", imgData)
		w := httptest.NewRecorder()

		_, ok := readCoverUpload(w, req, "123")
		if ok {
			t.Error("expected readCoverUpload() to fail with wrong field name")
		}
		if w.Code != http.StatusBadRequest {
			t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
		}
	})

	t.Run("Invalid Extension", func(t *testing.T) {
		req := createMultipartRequest(t, "cover", "test.txt", "txt", imgData)
		w := httptest.NewRecorder()

		_, ok := readCoverUpload(w, req, "123")
		if ok {
			t.Error("expected readCoverUpload() to fail with invalid extension")
		}
		if w.Code != http.StatusBadRequest {
			t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
		}
	})

	t.Run("Empty File", func(t *testing.T) {
		req := createMultipartRequest(t, "cover", "test.jpg", "jpeg", []byte{})
		w := httptest.NewRecorder()

		_, ok := readCoverUpload(w, req, "123")
		if ok {
			t.Error("expected readCoverUpload() to fail with empty file")
		}
		if w.Code != http.StatusBadRequest {
			t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
		}
	})
}

func TestSaveCoverFile(t *testing.T) {
	// Change working directory to temp dir for this test
	// Avoid changing global directory state, which breaks parallel execution.
	// Actually, running `os.Chdir` is not needed since the function creates `uploads/`
	// in the current directory (which for `go test` is the `inventur` package directory).
	// We'll clean up the created file after the test.

	t.Run("Valid Save", func(t *testing.T) {
		w := httptest.NewRecorder()
		imgData := createDummyImage("jpeg", 100, 100)
		id := "test-id"

		url, ok := saveCoverFile(w, id, imgData, ".jpg")

		if !ok {
			t.Fatalf("saveCoverFile() failed unexpectedly. Status: %d, Body: %s", w.Code, w.Body.String())
		}

		if !strings.HasPrefix(url, "/uploads/") {
			t.Errorf("expected URL to start with /uploads/, got %s", url)
		}

		if !strings.HasSuffix(url, ".jpg") {
			t.Errorf("expected URL to end with .jpg, got %s", url)
		}

		// Verify file exists
		filename := filepath.Base(url)
		filePath := filepath.Join("uploads", filename)
		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			t.Errorf("expected file %s to be created, but it was not", filePath)
		}

		// Clean up the created file
		defer func() { _ = os.Remove(filePath) }() //nolint:errcheck
	})

	t.Run("Path Traversal Protection", func(t *testing.T) {
		w := httptest.NewRecorder()
		imgData := createDummyImage("jpeg", 100, 100)

		// This should be stripped to "evil" or similar by filepath.Base in the handler
		id := "../../../etc/passwd"

		url, ok := saveCoverFile(w, id, imgData, ".jpg")

		if !ok {
			t.Fatalf("saveCoverFile() failed unexpectedly")
		}

		// Check that the URL does not contain path traversal elements
		if strings.Contains(url, "..") {
			t.Errorf("expected URL to not contain path traversal elements, got %s", url)
		}

		filename := filepath.Base(url)
		if strings.Contains(filename, "passwd") { // since filepath.Base("../../../etc/passwd") is "passwd"
			// it should use "passwd", which is safe within the uploads directory.
			filePath := filepath.Join("uploads", filename)
			if !strings.HasPrefix(filepath.Clean(filePath), filepath.Clean("uploads")) {
				t.Errorf("file path escapes uploads directory: %s", filePath)
			}

			// Clean up the created file
			defer func() { _ = os.Remove(filePath) }() //nolint:errcheck
		}
	})
}

func TestHandleUploadCover(t *testing.T) {
	tests := []struct {
		name           string
		method         string
		path           string
		bodyData       []byte
		fileName       string
		contentType    string
		setupMock      func(pgxmock.PgxPoolIface)
		expectedStatus int
	}{
		{
			name:           "Invalid Route Structure",
			method:         http.MethodPost,
			path:           "/api/books/123/wrong",
			bodyData:       nil,
			fileName:       "",
			contentType:    "",
			setupMock:      func(m pgxmock.PgxPoolIface) {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "Empty ID",
			method:         http.MethodPost,
			path:           "/api/books//cover-upload",
			bodyData:       nil,
			fileName:       "",
			contentType:    "",
			setupMock:      func(m pgxmock.PgxPoolIface) {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "Missing File Data",
			method:         http.MethodPost,
			path:           "/api/books/123/cover-upload",
			bodyData:       nil,
			fileName:       "",
			contentType:    "multipart/form-data",
			setupMock:      func(m pgxmock.PgxPoolIface) {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:        "Success Upload",
			method:      http.MethodPost,
			path:        "/api/books/123/cover-upload",
			bodyData:    createDummyImage("jpeg", 100, 100),
			fileName:    "test.jpg",
			contentType: "multipart/form-data",
			setupMock: func(m pgxmock.PgxPoolIface) {
				m.ExpectQuery("(?s)SELECT id, COALESCE.*").
					WithArgs("123").
					WillReturnRows(pgxmock.NewRows([]string{
						"id", "isbn", "title", "author", "signatur", "cover_url", "subject", "grade_level", "track", "stock", "last_counted", "sort_order", "medientyp", "jahrgang_von", "jahrgang_bis", "erweiterte_eigenschaften",
					}).AddRow(
						"123", "9781234567890", "Test Title", "Test Author", "", "/uploads/old_cover.jpg", "", int16(0), "", 1, nil, 1, "Buch", 5, 10, nil,
					))

				m.ExpectExec("(?s)UPDATE buecher_titel.*").
					WithArgs("", "", pgxmock.AnyArg(), "123").
					WillReturnResult(pgxmock.NewResult("UPDATE", 1))
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:        "Database Update Error",
			method:      http.MethodPost,
			path:        "/api/books/123/cover-upload",
			bodyData:    createDummyImage("jpeg", 100, 100),
			fileName:    "test.jpg",
			contentType: "multipart/form-data",
			setupMock: func(m pgxmock.PgxPoolIface) {
				m.ExpectQuery("(?s)SELECT id, COALESCE.*").
					WithArgs("123").
					WillReturnError(pgx.ErrNoRows) // Not found for old cover delete, handled silently

				m.ExpectExec("(?s)UPDATE buecher_titel.*").
					WithArgs("", "", pgxmock.AnyArg(), "123").
					WillReturnError(ErrBookNotFound)
			},
			expectedStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock, err := pgxmock.NewPool()
			if err != nil {
				t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
			}
			defer mock.Close()

			repo := NewBookRepository(mock)
			handler := &APIHandler{repo: repo}

			tt.setupMock(mock)

			var req *http.Request
			if tt.contentType == "multipart/form-data" && len(tt.bodyData) > 0 {
				var b bytes.Buffer
				w := multipart.NewWriter(&b)
				fw, _ := w.CreateFormFile("cover", tt.fileName) //nolint:errcheck
				_, _ = fw.Write(tt.bodyData) //nolint:errcheck
				_ = w.Close() //nolint:errcheck

				req = httptest.NewRequest(tt.method, tt.path, &b)
				req.Header.Set("Content-Type", w.FormDataContentType())
			} else if tt.contentType == "multipart/form-data" && len(tt.bodyData) == 0 {
				var b bytes.Buffer
				w := multipart.NewWriter(&b)
				_ = w.Close() //nolint:errcheck
				req = httptest.NewRequest(tt.method, tt.path, &b)
				req.Header.Set("Content-Type", w.FormDataContentType())
			} else {
				req = httptest.NewRequest(tt.method, tt.path, nil)
			}

			w := httptest.NewRecorder()

			handler.handleUploadCover(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d. Body: %s", tt.expectedStatus, w.Code, w.Body.String())
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("there were unfulfilled expectations: %s", err)
			}
		})
	}

	// Clean up created upload files during tests
	filepaths, _ := filepath.Glob("uploads/cover_123_*.jpg") //nolint:errcheck
	for _, f := range filepaths {
		_ = os.Remove(f) //nolint:errcheck
	}
}
