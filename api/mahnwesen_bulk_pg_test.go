package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"bibliothek/api/tuer"
	"bibliothek/db"
	"bibliothek/internal/pdftest"
	"bibliothek/internal/pgtest"
	"bibliothek/internal/testdaten"
)

// TestBulkPrintMahnungen_NurOffeneHochzaehlen sichert #4 ab: Der Bulk-Mahnlauf zählt
// die Mahnstufe NUR für noch offene Ausleihen hoch und liest die PDF-Daten innerhalb
// derselben Transaktion. Ein zwischen Aufbereiten und Druck zurückgegebenes Buch darf
// weder gemahnt (Mahnstufe) noch aufs PDF geraten.
func TestBulkPrintMahnungen_NurOffeneHochzaehlen(t *testing.T) {
	pool := pgtest.Pool(t)
	testdaten.ResetBestandsdaten(t, pool)
	ctx := context.Background()

	sid := testdaten.SeedSchueler(t, pool, "MB-1", "Mia", "7a")
	vergangenheit := time.Now().AddDate(0, 0, -30)

	loanOffen := testdaten.SeedAusleihe(t, pool, sid, "Offenes Buch", vergangenheit)
	loanZurueck := testdaten.SeedAusleihe(t, pool, sid, "Zurückgegebenes Buch", vergangenheit)
	// Buch wird zwischen Aufbereiten der Liste und Druck zurückgegeben.
	if _, err := pool.Exec(ctx, `UPDATE ausleihen SET rueckgabe_am = NOW() WHERE id = $1`, loanZurueck); err != nil {
		t.Fatalf("Rückgabe setzen: %v", err)
	}

	srv := &Server{Basis: tuer.Basis{DB: &db.Database{Pool: pool}}}
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
		t.Fatalf("Antwort ist kein PDF (Kopf: %q)", pdftest.ErsteBytes(rec.Body.Bytes(), 8))
	}

	// Offene Ausleihe: Mahnstufe hochgezählt, Mahndatum gesetzt.
	stufe, datum := testdaten.MahnState(t, pool, loanOffen)
	if stufe != 1 {
		t.Errorf("offene Ausleihe: Mahnstufe erwartet 1, war %d", stufe)
	}
	if datum == nil {
		t.Error("offene Ausleihe: letztes_mahndatum wurde nicht gesetzt")
	}

	// Zurückgegebene Ausleihe: unangetastet.
	stufeZ, datumZ := testdaten.MahnState(t, pool, loanZurueck)
	if stufeZ != 0 {
		t.Errorf("zurückgegebene Ausleihe: Mahnstufe darf nicht steigen, war %d", stufeZ)
	}
	if datumZ != nil {
		t.Error("zurückgegebene Ausleihe: letztes_mahndatum darf nicht gesetzt werden")
	}
}
