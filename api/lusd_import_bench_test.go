package api

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"bibliothek/db"
)

func createBenchCSVBytes(numRows int) []byte {
	var b bytes.Buffer
	b.WriteString("lusd_id,vorname,nachname,klasse\n")
	for i := 0; i < numRows; i++ {
		b.WriteString(fmt.Sprintf("ID-%05d,Vorname%d,Nachname%d,10A\n", i, i, i))
	}
	return b.Bytes()
}

func BenchmarkImportLUSDHandler(b *testing.B) {
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		databaseURL = "postgres://postgres:postgres@localhost:5432/bibliothek_test?sslmode=disable"
	}

	database, err := db.Connect(context.Background(), databaseURL)
	if err != nil {
		b.Fatalf("Failed to connect to db: %v", err)
	}

	server := &Server{DB: database}
	handler := server.ImportLUSDHandler()

	csvData := createBenchCSVBytes(1000)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		b.StopTimer()

		// Clean table before each run
		_, err := database.Pool.Exec(context.Background(), "TRUNCATE TABLE schueler CASCADE;")
		if err != nil {
			b.Fatalf("Failed to truncate table: %v", err)
		}

		body := new(bytes.Buffer)
		writer := multipart.NewWriter(body)
		part, err := writer.CreateFormFile("file", "test.csv")
		if err != nil {
			b.Fatalf("CreateFormFile failed: %v", err)
		}

		_, err = io.Copy(part, bytes.NewReader(csvData))
		if err != nil {
			b.Fatalf("Copy failed: %v", err)
		}

		err = writer.Close()
		if err != nil {
			b.Fatalf("Close failed: %v", err)
		}

		req := httptest.NewRequest("POST", "/import", body)
		req.Header.Set("Content-Type", writer.FormDataContentType())
		rr := httptest.NewRecorder()

		b.StartTimer()
		handler.ServeHTTP(rr, req)
		b.StopTimer()

		if rr.Code != http.StatusOK {
			b.Fatalf("Handler returned wrong status code: got %v want %v", rr.Code, http.StatusOK)
		}
	}
}
