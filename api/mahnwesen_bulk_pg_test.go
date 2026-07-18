package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"bibliothek/db"

	"github.com/jackc/pgx/v5/pgxpool"
)

// TestBulkPrintMahnungen_NurOffeneHochzaehlen sichert #4 ab: Der Bulk-Mahnlauf zählt
// die Mahnstufe NUR für noch offene Ausleihen hoch und liest die PDF-Daten innerhalb
// derselben Transaktion. Ein zwischen Aufbereiten und Druck zurückgegebenes Buch darf
// weder gemahnt (Mahnstufe) noch aufs PDF geraten.
func TestBulkPrintMahnungen_NurOffeneHochzaehlen(t *testing.T) {
	pool := pgTestPool(t)
	resetBestandsdaten(t, pool)
	ctx := context.Background()

	sid := seedSchueler(t, pool, "MB-1", "Mia", "7a")
	vergangenheit := time.Now().AddDate(0, 0, -30)

	loanOffen := seedAusleihe(t, pool, sid, "Offenes Buch", vergangenheit)
	loanZurueck := seedAusleihe(t, pool, sid, "Zurückgegebenes Buch", vergangenheit)
	// Buch wird zwischen Aufbereiten der Liste und Druck zurückgegeben.
	if _, err := pool.Exec(ctx, `UPDATE ausleihen SET rueckgabe_am = NOW() WHERE id = $1`, loanZurueck); err != nil {
		t.Fatalf("Rückgabe setzen: %v", err)
	}

	srv := &Server{DB: &db.Database{Pool: pool}}
	body, err := json.Marshal(BulkPrintRequest{AusleihIDs: []string{loanOffen, loanZurueck}})
	if err != nil {
		t.Fatalf("Request serialisieren: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, "/api/admin/mahnungen/bulk-print", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	srv.BulkPrintMahnungenHandler()(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("erwartet 200, bekam %d: %s", rec.Code, rec.Body.String())
	}
	if got := rec.Body.Bytes(); len(got) < 4 || string(got[:4]) != "%PDF" {
		t.Fatalf("Antwort ist kein PDF (Kopf: %q)", firstBytes(rec.Body.Bytes(), 8))
	}

	// Offene Ausleihe: Mahnstufe hochgezählt, Mahndatum gesetzt.
	stufe, datum := mahnState(t, pool, loanOffen)
	if stufe != 1 {
		t.Errorf("offene Ausleihe: Mahnstufe erwartet 1, war %d", stufe)
	}
	if datum == nil {
		t.Error("offene Ausleihe: letztes_mahndatum wurde nicht gesetzt")
	}

	// Zurückgegebene Ausleihe: unangetastet.
	stufeZ, datumZ := mahnState(t, pool, loanZurueck)
	if stufeZ != 0 {
		t.Errorf("zurückgegebene Ausleihe: Mahnstufe darf nicht steigen, war %d", stufeZ)
	}
	if datumZ != nil {
		t.Error("zurückgegebene Ausleihe: letztes_mahndatum darf nicht gesetzt werden")
	}
}

func mahnState(t *testing.T, pool *pgxpool.Pool, ausleiheID string) (int, *time.Time) {
	t.Helper()
	var stufe int
	var datum *time.Time
	if err := pool.QueryRow(context.Background(),
		`SELECT mahnstufe, letztes_mahndatum FROM ausleihen WHERE id = $1`, ausleiheID).
		Scan(&stufe, &datum); err != nil {
		t.Fatalf("Mahn-Status lesen: %v", err)
	}
	return stufe, datum
}

func firstBytes(b []byte, n int) []byte {
	if len(b) < n {
		return b
	}
	return b[:n]
}
