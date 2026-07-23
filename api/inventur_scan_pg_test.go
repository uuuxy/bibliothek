package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"bibliothek/db"
	"bibliothek/repository"

	"github.com/jackc/pgx/v5/pgxpool"
)

// seedSignaturMitExemplar legt eine Signatur samt Titel und einem ausleihbaren Exemplar an
// und liefert signature_id und exemplar_id. Der Signaturname wird eindeutig gehalten
// (signatures.name ist UNIQUE, die Tabelle wird zwischen Tests nicht geleert).
func seedSignaturMitExemplar(t *testing.T, pool *pgxpool.Pool, sigName, barcode string) (int, string) {
	t.Helper()
	ctx := context.Background()
	var sigID int
	if err := pool.QueryRow(ctx,
		`INSERT INTO signatures (name) VALUES ($1) RETURNING id`, sigName).Scan(&sigID); err != nil {
		t.Fatalf("Signatur %q anlegen: %v", sigName, err)
	}
	var titelID string
	if err := pool.QueryRow(ctx,
		`INSERT INTO buecher_titel (titel, signature_id) VALUES ($1, $2) RETURNING id`,
		sigName+"-Buch", sigID).Scan(&titelID); err != nil {
		t.Fatalf("Titel anlegen: %v", err)
	}
	exID := exemplar(t, pool, titelID, barcode, true, "")
	return sigID, exID
}

// TestInventurScan_FremderScopeNichtErfasst sichert #3 (Cross-Contamination) ab: Ein Buch
// außerhalb des Session-Scopes (Mathe-Buch, während eine Deutsch-Inventur läuft) wird mit
// 409 abgewiesen und NICHT in der Deutsch-Session verbucht. Vorher speicherte der Handler
// den Scan trotz Scope-Warnung ab — beim Abschluss der Mathe-Inventur fehlte das Exemplar
// dann in deren Erfassungen und wurde als VERLUST gebucht, obwohl es physisch vorlag.
func TestInventurScan_FremderScopeNichtErfasst(t *testing.T) {
	pool := pgTestPool(t)
	ctx := context.Background()
	if _, err := pool.Exec(ctx, `TRUNCATE inventur_sessions RESTART IDENTITY CASCADE`); err != nil {
		t.Fatalf("Sessions leeren: %v", err)
	}
	resetBestandsdaten(t, pool)

	suffix := fmt.Sprintf("%d", time.Now().UnixNano())
	barcodeMathe := "B-M-" + suffix
	barcodeDeutsch := "B-D-" + suffix
	sigDeutsch, _ := seedSignaturMitExemplar(t, pool, "INV-Deutsch-"+suffix, barcodeDeutsch)
	_, exMathe := seedSignaturMitExemplar(t, pool, "INV-Mathe-"+suffix, barcodeMathe)

	invRepo := repository.NewInventoryRepository(pool)
	session, err := invRepo.CreateInventurSession(ctx, "signature", repository.InventurScope{SignatureID: &sigDeutsch}, "Deutsch", "")
	if err != nil {
		t.Fatalf("Deutsch-Session anlegen: %v", err)
	}

	srv := &Server{DB: &db.Database{Pool: pool}}

	// Mathe-Buch in Deutsch-Session -> 409, NICHT erfasst.
	rec := inventurScan(t, srv, session.ID, barcodeMathe)
	if rec.Code != http.StatusConflict {
		t.Fatalf("Fremd-Scope-Scan: erwartet 409, war %d: %s", rec.Code, rec.Body.String())
	}
	if n := erfasstFuer(t, pool, session.ID, exMathe); n != 0 {
		t.Errorf("Mathe-Exemplar wurde trotz Fremd-Scope in der Deutsch-Session verbucht (%d Erfassungen) "+
			"— es würde später als Verlust gebucht", n)
	}

	// Die 409-Antwort muss strukturiert sein: echter Titel + Status "ausser_scope" + Warntext,
	// damit das Frontend kein "Unbekanntes Buch" anzeigt, sondern eine klare Warnung.
	var payload InventurScanResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("409-Body nicht als InventurScanResponse lesbar: %v (%s)", err, rec.Body.String())
	}
	if payload.Status != "ausser_scope" {
		t.Errorf("Status: erwartet \"ausser_scope\", war %q", payload.Status)
	}
	if payload.Titel == "" {
		t.Error("409-Antwort trägt keinen Buchtitel — das Frontend würde \"Unbekanntes Buch\" zeigen")
	}
	if len(payload.Warnungen) == 0 {
		t.Error("409-Antwort trägt keinen Warntext")
	}

	// Deutsch-Buch in Deutsch-Session -> 200, regulär erfasst (Gegenprobe).
	rec = inventurScan(t, srv, session.ID, barcodeDeutsch)
	if rec.Code != http.StatusOK {
		t.Fatalf("Scope-konformer Scan: erwartet 200, war %d: %s", rec.Code, rec.Body.String())
	}
}

func inventurScan(t *testing.T, srv *Server, sessionID, barcode string) *httptest.ResponseRecorder {
	t.Helper()
	body := fmt.Sprintf(`{"session_id":%q,"barcode_id":%q}`, sessionID, barcode)
	req := httptest.NewRequest(http.MethodPost, "/api/inventur/scan", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	srv.InventurScanHandler().ServeHTTP(rec, req)
	return rec
}

func erfasstFuer(t *testing.T, pool *pgxpool.Pool, sessionID, exemplarID string) int {
	t.Helper()
	var n int
	if err := pool.QueryRow(context.Background(),
		`SELECT count(*) FROM inventur_erfassungen WHERE session_id = $1 AND exemplar_id = $2`,
		sessionID, exemplarID).Scan(&n); err != nil {
		t.Fatalf("Erfassung zählen: %v", err)
	}
	return n
}
