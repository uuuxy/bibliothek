package service

import (
	"context"
	"errors"
	"testing"

	"bibliothek/internal/crypto"

	"github.com/jackc/pgx/v5"
	"github.com/pashagolub/pgxmock/v5"
)

func TestSchreibeLeserFoto(t *testing.T) {
	// Der Upload verschlüsselt das Foto mit diesem Schlüssel.
	t.Setenv(crypto.SchluesselVariable, "12345678901234567890123456789012")

	// 10x10 red pixel png image base64
	validBase64 := "data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAAAoAAAAKCAIAAAACUFjqAAAAF0lEQVR4nGL5z4APMOGVHbHSgAAAAP//RM4BFjLZ0j4AAAAASUVORK5CYII="

	t.Run("successful upload", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		if err != nil {
			t.Fatalf("pgxmock.NewPool() = %v", err)
		}
		defer mock.Close()

		mock.ExpectQuery("SELECT COALESCE\\(barcode_id, ''\\) FROM leser WHERE id = \\$1").
			WithArgs("student-123").
			WillReturnRows(pgxmock.NewRows([]string{"barcode_id"}).AddRow("barcode-123"))

		mock.ExpectExec("INSERT INTO schueler_fotos").
			WithArgs("student-123", pgxmock.AnyArg()).
			WillReturnResult(pgxmock.NewResult("INSERT", 1))

		photoURL, err := SchreibeLeserFoto(context.Background(), mock, "student-123", validBase64)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		expectedURL := "/api/schueler/barcode-123/photo"
		if photoURL != expectedURL {
			t.Errorf("SchreibeLeserFoto(…) = %s, erwartet url %s", photoURL, expectedURL)
		}

		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("there were unfulfilled expectations: %s", err)
		}
	})

	t.Run("missing student", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		if err != nil {
			t.Fatalf("pgxmock.NewPool() = %v", err)
		}
		defer mock.Close()

		mock.ExpectQuery("SELECT COALESCE\\(barcode_id, ''\\) FROM leser WHERE id = \\$1").
			WithArgs("student-not-found").
			WillReturnError(pgx.ErrNoRows)

		photoURL, err := SchreibeLeserFoto(context.Background(), mock, "student-not-found", validBase64)
		if err == nil {
			t.Error("SchreibeLeserFoto(unbekannter Schüler) = nil, erwartet einen Fehler")
		}
		if photoURL != "" {
			t.Errorf("SchreibeLeserFoto(…) = Adresse %s, erwartet leer", photoURL)
		}

		expectedErr := "schüler nicht gefunden"
		if err.Error() != expectedErr {
			t.Errorf("SchreibeLeserFoto(…) = Fehler %s, erwartet %s", err.Error(), expectedErr)
		}
		if !errors.Is(err, ErrFotoLeserUnbekannt) {
			t.Errorf("SchreibeLeserFoto(…) = %v, erwartet ErrFotoLeserUnbekannt", err)
		}

		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("there were unfulfilled expectations: %s", err)
		}
	})

	t.Run("invalid base64 string format", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		if err != nil {
			t.Fatalf("pgxmock.NewPool() = %v", err)
		}
		defer mock.Close()

		mock.ExpectQuery("SELECT COALESCE\\(barcode_id, ''\\) FROM leser WHERE id = \\$1").
			WithArgs("student-123").
			WillReturnRows(pgxmock.NewRows([]string{"barcode_id"}).AddRow("barcode-123"))

		invalidBase64 := "data:image/png;base64,invalid-base-64!!!"
		photoURL, err := SchreibeLeserFoto(context.Background(), mock, "student-123", invalidBase64)
		if !errors.Is(err, ErrFotoUnlesbar) {
			t.Errorf("SchreibeLeserFoto(…) = %v, erwartet ErrFotoUnlesbar", err)
		}
		if photoURL != "" {
			t.Errorf("SchreibeLeserFoto(…) = Adresse %s, erwartet leer", photoURL)
		}

		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("there were unfulfilled expectations: %s", err)
		}
	})

	t.Run("database insertion failure", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		if err != nil {
			t.Fatalf("pgxmock.NewPool() = %v", err)
		}
		defer mock.Close()

		mock.ExpectQuery("SELECT COALESCE\\(barcode_id, ''\\) FROM leser WHERE id = \\$1").
			WithArgs("student-123").
			WillReturnRows(pgxmock.NewRows([]string{"barcode_id"}).AddRow("barcode-123"))

		mock.ExpectExec("INSERT INTO schueler_fotos").
			WithArgs("student-123", pgxmock.AnyArg()).
			WillReturnError(errors.New("db error"))

		photoURL, err := SchreibeLeserFoto(context.Background(), mock, "student-123", validBase64)
		if err == nil {
			t.Error("SchreibeLeserFoto(…) bei Datenbankfehler = nil, erwartet einen Fehler")
		}
		if photoURL != "" {
			t.Errorf("SchreibeLeserFoto(…) = Adresse %s, erwartet leer", photoURL)
		}

		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("there were unfulfilled expectations: %s", err)
		}
	})
}
