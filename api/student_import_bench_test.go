package api

import (
	"bytes"
	"context"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"bibliothek/db"
)

func BenchmarkImportStudents(b *testing.B) {
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, "postgres://postgres:postgrespassword@localhost:5432/bibliothek?sslmode=disable")
	if err != nil {
		b.Skip("DB not available:", err)
	}
	defer pool.Close()

	s := &Server{
		DB: &db.Database{Pool: pool},
	}

	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)
	part, _ := writer.CreateFormFile("file", "students.csv")

	csvData := "Vorname;Nachname;Klasse;lusd_id\n"
	for i := 0; i < 1000; i++ {
		csvData += fmt.Sprintf("Vor%d;Nach%d;5A;LUSD%d\n", i, i, i)
	}
	part.Write([]byte(csvData))
	writer.Close()

	handler := s.ImportStudentsLUSDHandler()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		_, _ = pool.Exec(ctx, "TRUNCATE TABLE schueler CASCADE")
		b.StartTimer()

		req := httptest.NewRequest("POST", "/api/students/import", bytes.NewReader(buf.Bytes()))
		req.Header.Set("Content-Type", writer.FormDataContentType())
		rr := httptest.NewRecorder()

		handler.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			b.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
		}
	}
}
