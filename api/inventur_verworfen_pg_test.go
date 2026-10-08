package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"bibliothek/db"
	"bibliothek/repository"

	"github.com/jackc/pgx/v5/pgconn"
)

// Eine verworfene Inventur steht nicht als „vollständig" in der Liste früherer Inventuren
// (docs/OFFEN.md 5.32, Migration 149).
//
// Zwei Bereiche, beide ohne Verlust beendet: der eine abgeschlossen, nachdem sein einziges
// Buch gescannt war; der andere verworfen, nachdem eines von zwei Büchern gescannt war. Die
// Liste lieferte für beide dasselbe (verluste 0), und der Bildschirm schrieb bei beiden
// „vollständig". Geprüft über die Routen, die der Bildschirm ruft.
func TestInventurVerworfen_StehtNichtAlsVollstaendigInDerListe(t *testing.T) {
	pool := pgTestPool(t)
	ctx := context.Background()
	if _, err := pool.Exec(ctx, `TRUNCATE inventur_sessions RESTART IDENTITY CASCADE`); err != nil {
		t.Fatalf("Sessions leeren: %v", err)
	}
	resetBestandsdaten(t, pool)

	suffix := fmt.Sprintf("%d", time.Now().UnixNano())
	sigGeprueft := "INV-Geprueft-" + suffix
	sigVerworfen := "INV-Verworfen-" + suffix
	barcodeGeprueft := "B-G-" + suffix
	barcodeGescannt := "B-V1-" + suffix
	barcodeLiegtNoch := "B-V2-" + suffix
	seedSignaturMitExemplar(t, pool, sigGeprueft, barcodeGeprueft)
	seedSignaturMitExemplar(t, pool, sigVerworfen, barcodeGescannt)
	seedSignaturMitExemplar(t, pool, sigVerworfen, barcodeLiegtNoch)

	invRepo := repository.NewInventoryRepository(pool)
	geprueft, err := invRepo.CreateInventurSession(ctx, "signature", repository.InventurScope{Signatur: &sigGeprueft}, "Geprüft", "")
	if err != nil {
		t.Fatalf("Session anlegen: %v", err)
	}
	verworfen, err := invRepo.CreateInventurSession(ctx, "signature", repository.InventurScope{Signatur: &sigVerworfen}, "Verworfen", "")
	if err != nil {
		t.Fatalf("Session anlegen: %v", err)
	}

	srv := &Server{DB: &db.Database{Pool: pool}}
	for _, scan := range []struct{ session, barcode string }{
		{geprueft.ID, barcodeGeprueft}, {verworfen.ID, barcodeGescannt},
	} {
		if rec := inventurScan(t, srv, scan.session, scan.barcode); rec.Code != http.StatusOK {
			t.Fatalf("Scan %s: erwartet 200, war %d: %s", scan.barcode, rec.Code, rec.Body.String())
		}
	}
	if rec := inventurPost(t, srv.InventurFinishHandler(), "/api/inventur/finish", geprueft.ID, adminFuerAudit(t, pool)); rec.Code != http.StatusOK {
		t.Fatalf("Abschluss: erwartet 200, war %d: %s", rec.Code, rec.Body.String())
	}
	if rec := inventurPost(t, srv.InventurAbortHandler(), "/api/inventur/abort", verworfen.ID, ""); rec.Code != http.StatusOK {
		t.Fatalf("Verwerfen: erwartet 200, war %d: %s", rec.Code, rec.Body.String())
	}

	liste := abgeschlosseneInventuren(t, srv)
	if len(liste) != 2 {
		t.Fatalf("Liste früherer Inventuren: erwartet 2 Zeilen, waren %d: %+v", len(liste), liste)
	}
	nachID := map[string]map[string]any{}
	for _, zeile := range liste {
		nachID[fmt.Sprint(zeile["session_id"])] = zeile
	}
	// Das Feld wird roh gelesen, nicht über die Go-Struktur: Der Bildschirm liest den
	// JSON-Namen, und ein fehlendes Feld wäre in einer Struktur still false.
	for _, fall := range []struct {
		id, name  string
		verworfen bool
	}{
		{geprueft.ID, "abgeschlossene", false},
		{verworfen.ID, "verworfene", true},
	} {
		zeile, da := nachID[fall.id]
		if !da {
			t.Fatalf("Die %s Inventur fehlt in der Liste: %+v", fall.name, liste)
		}
		wert, hatFeld := zeile["verworfen"]
		if !hatFeld {
			t.Fatalf("Die %s Inventur trägt kein Feld verworfen: %+v", fall.name, zeile)
		}
		if wert != fall.verworfen {
			t.Errorf("Die %s Inventur: verworfen = %v, erwartet %v", fall.name, wert, fall.verworfen)
		}
		if zeile["verluste"] != float64(0) || zeile["erfasst"] != float64(1) {
			t.Errorf("Die %s Inventur: verluste %v, erfasst %v, erwartet 0 und 1", fall.name, zeile["verluste"], zeile["erfasst"])
		}
	}

	// Das Verwerfen bucht keinen Verlust: Das ungescannte Buch bleibt im Umlauf.
	var ausgesondert bool
	if err := pool.QueryRow(ctx, `SELECT ist_ausgesondert FROM buecher_exemplare WHERE barcode_id = $1`,
		barcodeLiegtNoch).Scan(&ausgesondert); err != nil {
		t.Fatalf("Exemplar lesen: %v", err)
	}
	if ausgesondert {
		t.Error("Das ungescannte Buch der verworfenen Inventur wurde ausgesondert")
	}

	// Ein zweites Verwerfen der abgeschlossenen Inventur bleibt 200 (der Bereich ist frei),
	// macht aus dem Abschluss aber kein Verwerfen.
	if rec := inventurPost(t, srv.InventurAbortHandler(), "/api/inventur/abort", geprueft.ID, ""); rec.Code != http.StatusOK {
		t.Fatalf("Verwerfen einer abgeschlossenen Inventur: erwartet 200, war %d: %s", rec.Code, rec.Body.String())
	}
	for _, zeile := range abgeschlosseneInventuren(t, srv) {
		if zeile["session_id"] == geprueft.ID && zeile["verworfen"] != false {
			t.Errorf("Verwerfen nach dem Abschluss hat die Inventur umgeschrieben: %+v", zeile)
		}
	}
}

// Eine laufende Inventur ist nie verworfen — die Datenbank hält die Regel für jeden
// Schreiber neben AbortInventurSession (chk_inv_session_verworfen_beendet, Migration 149).
func TestInventurVerworfen_DatenbankHaeltDieRegel(t *testing.T) {
	pool := pgTestPool(t)
	ctx := context.Background()
	if _, err := pool.Exec(ctx, `TRUNCATE inventur_sessions RESTART IDENTITY CASCADE`); err != nil {
		t.Fatalf("Sessions leeren: %v", err)
	}
	resetBestandsdaten(t, pool)

	sig := fmt.Sprintf("INV-Regel-%d", time.Now().UnixNano())
	session, err := repository.NewInventoryRepository(pool).CreateInventurSession(ctx, "signature",
		repository.InventurScope{Signatur: &sig}, "Regel", "")
	if err != nil {
		t.Fatalf("Session anlegen: %v", err)
	}

	_, err = pool.Exec(ctx, `UPDATE inventur_sessions SET verworfen = true WHERE id = $1`, session.ID)
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) || pgErr.ConstraintName != "chk_inv_session_verworfen_beendet" {
		t.Fatalf("Eine laufende Inventur als verworfen markieren: erwartet chk_inv_session_verworfen_beendet, war %v", err)
	}
}

func inventurPost(t *testing.T, h http.Handler, pfad, sessionID, benutzerID string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, pfad, strings.NewReader(fmt.Sprintf(`{"session_id":%q}`, sessionID)))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, alsBenutzer(req, benutzerID))
	return rec
}

func abgeschlosseneInventuren(t *testing.T, srv *Server) []map[string]any {
	t.Helper()
	rec := httptest.NewRecorder()
	srv.ListAbgeschlosseneInventurenHandler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/inventur/abgeschlossen", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("Liste früherer Inventuren: erwartet 200, war %d: %s", rec.Code, rec.Body.String())
	}
	var liste []map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &liste); err != nil {
		t.Fatalf("Liste nicht lesbar: %v (%s)", err, rec.Body.String())
	}
	return liste
}
