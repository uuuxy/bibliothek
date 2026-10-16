package inventurlauf

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"bibliothek/api/tuer"
	"bibliothek/db"
	"bibliothek/internal/pgtest"
	"bibliothek/internal/testdaten"
	"bibliothek/repository"

	"github.com/jackc/pgx/v5/pgxpool"
)

// TestScanVerbuchen_FremderBereichNichtErfasst: Ein Buch außerhalb des Bereichs (ein
// Mathebuch, während eine Inventur für Deutsch läuft) wird mit 409 abgewiesen und nicht im
// Lauf für Deutsch verbucht. Sonst fehlte es beim Abschluss der Inventur für Mathematik und
// würde als Verlust gebucht, obwohl es im Regal steht.
func TestScanVerbuchen_FremderBereichNichtErfasst(t *testing.T) {
	pool := pgtest.Pool(t)
	ctx := context.Background()
	if _, err := pool.Exec(ctx, `TRUNCATE inventur_sessions RESTART IDENTITY CASCADE`); err != nil {
		t.Fatalf("Sessions leeren: %v", err)
	}
	testdaten.LeereBestandsdaten(t, pool)

	suffix := fmt.Sprintf("%d", time.Now().UnixNano())
	barcodeMathe := "B-M-" + suffix
	barcodeDeutsch := "B-D-" + suffix
	sigDeutsch := "INV-Deutsch-" + suffix
	testdaten.LegeExemplarMitSignaturAn(t, pool, sigDeutsch, barcodeDeutsch)
	exMathe := testdaten.LegeExemplarMitSignaturAn(t, pool, "INV-Mathe-"+suffix, barcodeMathe)

	invRepo := repository.NewInventurAbfragen(pool)
	session, err := invRepo.LegeLaufAn(ctx, "signature", repository.InventurBereich{Signatur: &sigDeutsch}, "Deutsch", "")
	if err != nil {
		t.Fatalf("Deutsch-Session anlegen: %v", err)
	}

	srv := &Tueren{Basis: &tuer.Basis{DB: &db.Verbindung{Pool: pool}}}

	// Mathebuch im Lauf für Deutsch: 409, nicht erfasst.
	rec := inventurScan(t, srv, session.ID, barcodeMathe)
	if rec.Code != http.StatusConflict {
		t.Fatalf("Fremd-Scope-Scan = %d, erwartet 409: %s", rec.Code, rec.Body.String())
	}
	if n := erfasstFuer(t, pool, session.ID, exMathe); n != 0 {
		t.Errorf("Mathe-Exemplar wurde trotz Fremd-Scope in der Deutsch-Session verbucht (%d Erfassungen) "+
			"— es würde später als Verlust gebucht", n)
	}

	// Die 409-Antwort muss strukturiert sein: echter Titel + Status "ausser_scope" + Warntext,
	// damit die Oberfläche kein "Unbekanntes Buch" anzeigt, sondern eine Warnung.
	var payload scanAntwort
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("409-Body nicht als scanAntwort lesbar: %v (%s)", err, rec.Body.String())
	}
	if payload.Status != "ausser_scope" {
		t.Errorf("Status = %q, erwartet \"ausser_scope\"", payload.Status)
	}
	if payload.Titel == "" {
		t.Error("409-Antwort trägt keinen Buchtitel — das Frontend würde \"Unbekanntes Buch\" zeigen")
	}
	if len(payload.Warnungen) == 0 {
		t.Error("409-Antwort trägt keinen Warntext")
	}

	// Deutschbuch im Lauf für Deutsch: 200, erfasst (Gegenprobe).
	rec = inventurScan(t, srv, session.ID, barcodeDeutsch)
	if rec.Code != http.StatusOK {
		t.Fatalf("Scope-konformer Scan = %d, erwartet 200: %s", rec.Code, rec.Body.String())
	}
}

func inventurScan(t *testing.T, srv *Tueren, laufID, barcode string) *httptest.ResponseRecorder {
	t.Helper()
	body := fmt.Sprintf(`{"session_id":%q,"barcode_id":%q}`, laufID, barcode)
	req := httptest.NewRequest(http.MethodPost, "/api/inventur/scan", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	srv.ScanVerbuchen().ServeHTTP(rec, req)
	return rec
}

func erfasstFuer(t *testing.T, pool *pgxpool.Pool, laufID, exemplarID string) int {
	t.Helper()
	var n int
	if err := pool.QueryRow(context.Background(),
		`SELECT count(*) FROM inventur_erfassungen WHERE session_id = $1 AND exemplar_id = $2`,
		laufID, exemplarID).Scan(&n); err != nil {
		t.Fatalf("Erfassung zählen: %v", err)
	}
	return n
}

// Ein unbekannter Barcode bekommt am Scanner einen Satz, den das Personal lesen kann. Der
// Fehlertext der Datenbank gehört ins Log, nicht auf den Bildschirm.
func TestScanVerbuchen_UnbekannterBarcodeNenntKeinenDatenbankfehler(t *testing.T) {
	pool := pgtest.Pool(t)
	ctx := context.Background()
	if _, err := pool.Exec(ctx, `TRUNCATE inventur_sessions RESTART IDENTITY CASCADE`); err != nil {
		t.Fatalf("Sessions leeren: %v", err)
	}
	testdaten.LeereBestandsdaten(t, pool)
	session, err := repository.NewInventurAbfragen(pool).
		LegeLaufAn(ctx, "global", repository.InventurBereich{}, "Komplette Bibliothek", "")
	if err != nil {
		t.Fatalf("Inventur anlegen: %v", err)
	}

	rec := inventurScan(t, &Tueren{Basis: &tuer.Basis{DB: &db.Verbindung{Pool: pool}}}, session.ID, "GIBT-ES-NICHT")
	if rec.Code != http.StatusNotFound {
		t.Fatalf("Status %d, erwartet 404: %s", rec.Code, rec.Body.String())
	}
	var antwort struct {
		Error string `json:"error"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &antwort); err != nil {
		t.Fatalf("Antwort lesen: %v", err)
	}
	if antwort.Error != "Zu diesem Barcode gibt es kein Exemplar" {
		t.Errorf("Meldung am Scanner: %q", antwort.Error)
	}
}

// TestScanVerbuchen_Fehlerfaelle prüft die Abweisungen und die Warnung: ein fehlender oder
// unbekannter Lauf, ein unbekanntes oder ausgesondertes Exemplar und ein ausgeliehenes Buch.
func TestScanVerbuchen_Fehlerfaelle(t *testing.T) {
	pool := pgtest.Pool(t)
	ctx := context.Background()
	if _, err := pool.Exec(ctx, `TRUNCATE inventur_sessions RESTART IDENTITY CASCADE`); err != nil {
		t.Fatalf("Sessions leeren: %v", err)
	}
	testdaten.LeereBestandsdaten(t, pool)

	suffix := fmt.Sprintf("%d", time.Now().UnixNano())
	sigDeutsch := "INV-Deutsch-" + suffix
	barcodeDeutsch := "B-D-" + suffix
	testdaten.LegeExemplarMitSignaturAn(t, pool, sigDeutsch, barcodeDeutsch)

	// Ausgesondertes Exemplar, direkt in der Zeile gesetzt
	barcodeAusgesondert := "B-A-" + suffix
	exAusgesondert := testdaten.LegeExemplarMitSignaturAn(t, pool, sigDeutsch, barcodeAusgesondert)
	if _, err := pool.Exec(ctx, `UPDATE buecher_exemplare SET ist_ausgesondert = true, aussonderung_grund = 'AUSSORTIERT' WHERE id = $1`, exAusgesondert); err != nil {
		t.Fatalf("Exemplar aussondern: %v", err)
	}

	// Ausgeliehenes Exemplar
	barcodeLent := "B-L-" + suffix
	exLent := testdaten.LegeExemplarMitSignaturAn(t, pool, sigDeutsch, barcodeLent)
	var userID string
	// Die Ausleihe hängt an der Leserzeile des Kontos (Migration 125).
	if err := pool.QueryRow(ctx, `INSERT INTO benutzer (vorname, nachname, email, rolle, aktiv) VALUES ('Test', 'User', 'test@example.com', 'admin', true) RETURNING leser_id`).Scan(&userID); err != nil {
		t.Fatalf("User anlegen: %v", err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO ausleihen (exemplar_id, schueler_id, ausgeliehen_am, rueckgabe_frist) VALUES ($1, $2, NOW(), NOW() + interval '14 days')`, exLent, userID); err != nil {
		t.Fatalf("Ausleihe anlegen: %v", err)
	}

	invRepo := repository.NewInventurAbfragen(pool)
	session, err := invRepo.LegeLaufAn(ctx, "signature", repository.InventurBereich{Signatur: &sigDeutsch}, "Deutsch", "")
	if err != nil {
		t.Fatalf("Deutsch-Session anlegen: %v", err)
	}

	srv := &Tueren{Basis: &tuer.Basis{DB: &db.Verbindung{Pool: pool}}}

	tests := []struct {
		name       string
		laufID     string
		barcode    string
		wantCode   int
		wantStatus string
	}{
		{"Fehlende Session ID", "", barcodeDeutsch, http.StatusBadRequest, ""},
		{"Unbekannte Session ID", "00000000-0000-0000-0000-000000000000", barcodeDeutsch, http.StatusNotFound, ""},
		{"Unbekannter Barcode", session.ID, "UNKNOWN", http.StatusNotFound, ""},
		{"Ausgesondertes Exemplar", session.ID, barcodeAusgesondert, http.StatusConflict, ""},
		{"Ausgeliehenes Exemplar (Warnung aber OK)", session.ID, barcodeLent, http.StatusOK, "erfasst"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := inventurScan(t, srv, tt.laufID, tt.barcode)
			if rec.Code != tt.wantCode {
				t.Errorf("rec.Code = %d, erwartet %d: %s", rec.Code, tt.wantCode, rec.Body.String())
			}
			if tt.wantStatus != "" {
				var payload scanAntwort
				if err := json.Unmarshal(rec.Body.Bytes(), &payload); err == nil {
					if payload.Status != tt.wantStatus {
						t.Errorf("Status = %q, erwartet %q", payload.Status, tt.wantStatus)
					}
					if tt.name == "Ausgeliehenes Exemplar (Warnung aber OK)" && len(payload.Warnungen) == 0 {
						t.Errorf("Scan eines ausgeliehenen Buchs ohne Warnung")
					}
				}
			}
		})
	}
}
