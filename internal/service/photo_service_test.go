package service

import (
	"context"
	"errors"
	"testing"

	"bibliothek/internal/crypto"
	"github.com/jackc/pgx/v5"
	"github.com/pashagolub/pgxmock/v4"
)

func TestUploadStudentPhoto(t *testing.T) {
	// Set the required master key for encryption
	t.Setenv(crypto.SchluesselVariable, "12345678901234567890123456789012")

	// 10x10 red pixel png image base64
	validBase64 := "data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAAAoAAAAKCAIAAAACUFjqAAAAF0lEQVR4nGL5z4APMOGVHbHSgAAAAP//RM4BFjLZ0j4AAAAASUVORK5CYII="

	t.Run("successful upload", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		if err != nil {
			t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
		}
		defer mock.Close()

		mock.ExpectQuery("SELECT barcode_id FROM schueler WHERE id = \\$1").
			WithArgs("student-123").
			WillReturnRows(pgxmock.NewRows([]string{"barcode_id"}).AddRow("barcode-123"))

		mock.ExpectExec("INSERT INTO schueler_fotos").
			WithArgs("student-123", pgxmock.AnyArg()).
			WillReturnResult(pgxmock.NewResult("INSERT", 1))

		photoURL, err := UploadStudentPhoto(context.Background(), mock, "student-123", validBase64)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		expectedURL := "/api/schueler/barcode-123/photo"
		if photoURL != expectedURL {
			t.Errorf("expected url %s, got %s", expectedURL, photoURL)
		}

		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("there were unfulfilled expectations: %s", err)
		}
	})

	t.Run("missing student", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		if err != nil {
			t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
		}
		defer mock.Close()

		mock.ExpectQuery("SELECT barcode_id FROM schueler WHERE id = \\$1").
			WithArgs("student-not-found").
			WillReturnError(pgx.ErrNoRows)

		photoURL, err := UploadStudentPhoto(context.Background(), mock, "student-not-found", validBase64)
		if err == nil {
			t.Error("expected error, got nil")
		}
		if photoURL != "" {
			t.Errorf("expected empty url, got %s", photoURL)
		}

		expectedErr := "schüler nicht gefunden"
		if err.Error() != expectedErr {
			t.Errorf("expected error %s, got %s", expectedErr, err.Error())
		}

		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("there were unfulfilled expectations: %s", err)
		}
	})

	t.Run("invalid base64 string format", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		if err != nil {
			t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
		}
		defer mock.Close()

		mock.ExpectQuery("SELECT barcode_id FROM schueler WHERE id = \\$1").
			WithArgs("student-123").
			WillReturnRows(pgxmock.NewRows([]string{"barcode_id"}).AddRow("barcode-123"))

		invalidBase64 := "data:image/png;base64,invalid-base-64!!!"
		photoURL, err := UploadStudentPhoto(context.Background(), mock, "student-123", invalidBase64)
		if err == nil {
			t.Error("expected error, got nil")
		}
		if photoURL != "" {
			t.Errorf("expected empty url, got %s", photoURL)
		}

		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("there were unfulfilled expectations: %s", err)
		}
	})

	t.Run("database insertion failure", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		if err != nil {
			t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
		}
		defer mock.Close()

		mock.ExpectQuery("SELECT barcode_id FROM schueler WHERE id = \\$1").
			WithArgs("student-123").
			WillReturnRows(pgxmock.NewRows([]string{"barcode_id"}).AddRow("barcode-123"))

		mock.ExpectExec("INSERT INTO schueler_fotos").
			WithArgs("student-123", pgxmock.AnyArg()).
			WillReturnError(errors.New("db error"))

		photoURL, err := UploadStudentPhoto(context.Background(), mock, "student-123", validBase64)
		if err == nil {
			t.Error("expected error, got nil")
		}
		if photoURL != "" {
			t.Errorf("expected empty url, got %s", photoURL)
		}

		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("there were unfulfilled expectations: %s", err)
		}
	})
}
