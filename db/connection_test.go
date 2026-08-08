package db

import (
	"bytes"
	"context"
	"errors"
	"log"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/pashagolub/pgxmock/v4"
)

func TestConnect_Success(t *testing.T) {
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL nicht gesetzt")
	}

	ctx := context.Background()
	db, err := Connect(ctx, dsn)
	if err != nil {
		t.Fatalf("Erwartete erfolgreiche Verbindung, erhielt Fehler: %v", err)
	}
	defer db.Close()

	if db == nil || db.Pool == nil {
		t.Fatal("Erwartete initialisierte Datenbank")
	}
}

func TestConnect_ParseError(t *testing.T) {
	ctx := context.Background()

	_, err := Connect(ctx, "postgres://user@localhost:invalid_port/db")
	if err == nil {
		t.Fatal("Erwartete Fehler beim Parsen der DSN, erhielt keinen")
	}
	if !strings.Contains(err.Error(), "failed to parse") {
		t.Errorf("Unerwartete Fehlermeldung: %v", err)
	}
}

func TestConnect_PingError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	_, err := Connect(ctx, "postgres://user:pass@127.0.0.1:1/nonexistentdb?connect_timeout=1")
	if err == nil {
		t.Fatal("Erwartete Fehler beim Ping, erhielt keinen")
	}
	if !strings.Contains(err.Error(), "failed to ping database") {
		t.Errorf("Unerwartete Fehlermeldung: %v", err)
	}
}

func TestDatabase_Close(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("failed to create mock: %v", err)
	}

	db := &Database{Pool: mock}
	db.Close()

	dbNil := &Database{Pool: nil}
	dbNil.Close()
}

func TestSafeRollback(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("failed to create mock: %v", err)
	}
	defer mock.Close()

	ctx := context.Background()

	// 1. Rollback success
	mock.ExpectBegin()
	mock.ExpectRollback()

	tx1, err := mock.Begin(ctx)
	if err != nil {
		t.Fatalf("failed to begin tx: %v", err)
	}
	SafeRollback(ctx, tx1)

	// 2. Rollback error (ErrTxClosed) - should not log error
	mock.ExpectBegin()
	mock.ExpectRollback().WillReturnError(pgx.ErrTxClosed)

	tx2, err := mock.Begin(ctx)
	if err != nil {
		t.Fatalf("failed to begin tx: %v", err)
	}

	var buf bytes.Buffer
	log.SetOutput(&buf)
	defer log.SetOutput(os.Stderr) // Reset logger

	SafeRollback(ctx, tx2)

	if buf.Len() > 0 {
		t.Errorf("erwartete keine Log-Ausgabe für ErrTxClosed, erhielt: %s", buf.String())
	}
	buf.Reset()

	// 3. Rollback error (other error) - should log error
	mock.ExpectBegin()
	testErr := errors.New("some unexpected error")
	mock.ExpectRollback().WillReturnError(testErr)

	tx3, err := mock.Begin(ctx)
	if err != nil {
		t.Fatalf("failed to begin tx: %v", err)
	}

	SafeRollback(ctx, tx3)

	if !strings.Contains(buf.String(), "db: transaction rollback failed") {
		t.Errorf("erwartete Log-Ausgabe für unerwarteten Fehler, erhielt: %s", buf.String())
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}
}
