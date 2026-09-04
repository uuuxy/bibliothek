package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"bibliothek/db"
	"bibliothek/internal/service"
	"bibliothek/repository"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Befund F1 der unabhängigen Prüfung (bewertung/datenbank-pruefbericht.md):
// Der Bestellstatus lebte als Magie-Text im Freitextfeld zustand_notiz — eine
// harmlose Personal-Notiz wie "Bestellt am 3.9. neu" ließ ein Exemplar still
// aus dem OPAC verschwinden. Seit Migration 071 entscheidet ausschließlich die
// Spalte bestellstatus. Dieser Test spielt genau das Befund-Szenario an der
// echten Datenbank durch.
func TestBestellstatusSpalteStattNotizMagie(t *testing.T) {
	pool := pgTestPool(t)
	ctx := context.Background()
	resetBestandsdaten(t, pool)

	var titelID string
	if err := pool.QueryRow(ctx, `
		INSERT INTO buecher_titel (titel, autor, medientyp)
		VALUES ('F1-Magietext-Titel', 'Pruefbericht', 'Buch') RETURNING id`).Scan(&titelID); err != nil {
		t.Fatalf("Titel: %v", err)
	}

	// 1) Das Befund-Szenario: menschliche Notiz, die wie der Magie-Text aussieht.
	if _, err := pool.Exec(ctx, `
		INSERT INTO buecher_exemplare (titel_id, barcode_id, zustand_notiz, ist_ausleihbar)
		VALUES ($1, 'F1-NOTIZ', 'Bestellt am 3.9. neu, alter Band verloren', true)`, titelID); err != nil {
		t.Fatalf("Exemplar mit Notiz: %v", err)
	}
	if n := opacGesamt(t, pool, "F1-Magietext-Titel"); n != 1 {
		t.Fatalf("Exemplar mit harmloser 'Bestellt…'-NOTIZ fehlt im OPAC (gesamt=%d) — Notiz steuert wieder!", n)
	}

	// 2) Ein echtes Pipeline-Exemplar (Spalte gesetzt) bleibt unsichtbar …
	if _, err := pool.Exec(ctx, `
		INSERT INTO buecher_exemplare (titel_id, barcode_id, zustand_notiz, ist_ausleihbar, bestellstatus)
		VALUES ($1, 'F1-ZULAUF', 'Im Zulauf - Cornelsen', false, 'im_zulauf')`, titelID); err != nil {
		t.Fatalf("Pipeline-Exemplar: %v", err)
	}
	if n := opacGesamt(t, pool, "F1-Magietext-Titel"); n != 1 {
		t.Fatalf("Exemplar im Zulauf darf nicht im OPAC zählen (gesamt=%d)", n)
	}

	// … bis der Wareneingang es freigibt: Status weg, sichtbar.
	var exemplarID string
	if err := pool.QueryRow(ctx, `SELECT id FROM buecher_exemplare WHERE barcode_id='F1-ZULAUF'`).Scan(&exemplarID); err != nil {
		t.Fatalf("Exemplar-ID: %v", err)
	}
	auditRepo := repository.NewAuditRepository(pool)
	if _, err := service.BulkReceiveOrder(ctx, pool, auditRepo, service.BulkReceiveParams{
		ExemplarIDs: []string{exemplarID},
		AdminID:     "00000000-0000-0000-0000-000000000001",
		IPAddr:      "",
	}); err != nil {
		t.Fatalf("Wareneingang: %v", err)
	}
	if n := opacGesamt(t, pool, "F1-Magietext-Titel"); n != 2 {
		t.Fatalf("nach Wareneingang muessen beide Exemplare zaehlen (gesamt=%d)", n)
	}
	var status *string
	if err := pool.QueryRow(ctx, `SELECT bestellstatus FROM buecher_exemplare WHERE id=$1`, exemplarID).Scan(&status); err != nil {
		t.Fatalf("Status lesen: %v", err)
	}
	if status != nil {
		t.Fatalf("Wareneingang muss bestellstatus auf NULL setzen, ist %q", *status)
	}

	// 3) Die Prüfregel verteidigt das Vokabular.
	if _, err := pool.Exec(ctx, `
		INSERT INTO buecher_exemplare (titel_id, barcode_id, bestellstatus)
		VALUES ($1, 'F1-KAPUTT', 'quatsch')`, titelID); err == nil {
		t.Fatal("bestellstatus='quatsch' wurde angenommen — CHECK-Constraint fehlt")
	} else if !strings.Contains(err.Error(), "chk_exemplar_bestellstatus") {
		t.Fatalf("falscher Fehler: %v", err)
	}
}

// opacGesamt fragt die ÖFFENTLICHE Katalogsuche über den echten Handler ab —
// nicht über eine nachgebaute Abfrage, die eine Regression im Handler nie sähe.
func opacGesamt(t *testing.T, pool *pgxpool.Pool, titel string) int {
	t.Helper()
	srv := &Server{DB: &db.Database{Pool: pool}}
	req := httptest.NewRequest("GET", "/api/opac/suche?q="+url.QueryEscape(titel), nil)
	w := httptest.NewRecorder()
	srv.PublicCatalogSearchHandler()(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("OPAC-Suche: HTTP %d: %s", w.Code, w.Body.String())
	}
	var antwort []OpacTitel
	if err := json.Unmarshal(w.Body.Bytes(), &antwort); err != nil {
		t.Fatalf("OPAC-Antwort unlesbar: %v / %s", err, w.Body.String())
	}
	for _, titelZeile := range antwort {
		if titelZeile.Titel == titel {
			return titelZeile.Gesamt
		}
	}
	return 0 // Titel gar nicht gelistet = 0 sichtbare Exemplare
}
