package inventur

import (
	"bytes"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/xuri/excelize/v2"
)

func createTestRequest(t *testing.T, filename, content string, isExcel bool, missingFile bool) (*http.Request, *httptest.ResponseRecorder) {
	body := new(bytes.Buffer)
	writer := multipart.NewWriter(body)

	if !missingFile {
		part, err := writer.CreateFormFile("file", filename)
		if err != nil {
			t.Fatal(err)
		}

		if isExcel {
			f := excelize.NewFile()
			if content != "" {
				lines := strings.Split(content, "\n")
				for r, line := range lines {
					if line == "" {
						continue
					}
					cells := strings.Split(line, ",")
					for c, cell := range cells {
						cellName, _ := excelize.CoordinatesToCellName(c+1, r+1)
						f.SetCellValue("Sheet1", cellName, cell)
					}
				}
			} else {
				// empty excel file content
			}
			if err := f.Write(part); err != nil {
				t.Fatal(err)
			}
		} else {
			part.Write([]byte(content))
		}
	} else if filename != "" {
		// Just a text field
		writer.WriteField("not_a_file", "foo")
	}

	writer.Close()

	req := httptest.NewRequest("POST", "/upload", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	w := httptest.NewRecorder()

	return req, w
}

func TestExtractImportRows(t *testing.T) {
	tests := []struct {
		name        string
		filename    string
		content     string
		isExcel     bool
		missingFile bool
		wantErr     bool
		errMsg      string
		wantRows    int
	}{
		{
			name:     "Valid CSV",
			filename: "test.csv",
			content:  "isbn,titel\n123,test1\n456,test2",
			isExcel:  false,
			wantRows: 3,
		},
		{
			name:     "CSV with Semicolons",
			filename: "test2.csv",
			content:  "isbn;titel\n123;test1\n456;test2",
			isExcel:  false,
			wantRows: 3,
		},
		{
			name:    "Invalid CSV format",
			filename: "test.csv",
			content: "a,b\nc,d,e",
			isExcel: false,
			wantErr: true,
			errMsg:  "ungültige csv-datei",
		},
		{
			name:     "Empty CSV",
			filename: "empty.csv",
			content:  "",
			isExcel:  false,
			wantErr:  true,
			errMsg:   "datei ist leer",
		},
		{
			name:     "Valid Excel",
			filename: "test.xlsx",
			content:  "isbn,titel\n123,test1\n456,test2",
			isExcel:  true,
			wantRows: 3,
		},
		{
			name:     "Empty Excel content",
			filename: "empty.xlsx",
			content:  "",
			isExcel:  true,
			wantErr:  true,
			errMsg:   "keine daten gefunden",
		},
		{
			name:        "Missing file",
			filename:    "",
			missingFile: true,
			wantErr:     true,
			errMsg:      "keine datei gefunden",
		},
		{
			name:     "Invalid Excel file format",
			filename: "test.xlsx",
			content:  "this is not a valid excel file format just string",
			isExcel:  false, // Send string without excelize, causing parsing failure
			wantErr:  true,
			errMsg:   "ungültige excel-datei",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, w := createTestRequest(t, tt.filename, tt.content, tt.isExcel, tt.missingFile)

			rows, err := extractImportRows(w, req)

			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error containing %q, got nil", tt.errMsg)
				} else if err.Error() != tt.errMsg {
					t.Errorf("expected error %q, got %q", tt.errMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if len(rows) != tt.wantRows {
					t.Errorf("expected %d rows, got %d", tt.wantRows, len(rows))
				}
			}
		})
	}
}

func TestExtractImportRows_TooLargeOrInvalidForm(t *testing.T) {
	// Let's create a request with no Content-Type to see if ParseMultipartForm fails
	req := httptest.NewRequest("POST", "/upload", nil)
	w := httptest.NewRecorder()

	_, err := extractImportRows(w, req)
	if err == nil {
		t.Error("expected error for invalid form, got nil")
	} else if err.Error() != "datei zu groß oder ungültig" {
		t.Errorf("expected error %q, got %q", "datei zu groß oder ungültig", err.Error())
	}
}
