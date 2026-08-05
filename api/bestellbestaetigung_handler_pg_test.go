package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"bibliothek/db"

	"github.com/jackc/pgx/v5/pgxpool"
)

// haendlerMitBestaetigung legt einen Lieferanten mit/ohne bietet_bestellbestaetigung an.
func haendlerMitBestaetigung(t *testing.T, pool *pgxpool.Pool, name string, bietetBestaetigung bool) string {
	t.Helper()
	var id string
	err := pool.QueryRow(context.Background(), `
		INSERT INTO lieferanten (name, email, kundennummer, bietet_bestellbestaetigung)
		VALUES ($1, $2, $3, $4) RETURNING id
	`, name, name+"@example.invalid", "K-"+name, bietetBestaetigung).Scan(&id)
	if err != nil {
		t.Fatalf("Lieferant %s anlegen: %v", name, err)
	}
	return id
}

// bestellungFuerLieferant legt eine bestellungen_verlauf-Zeile für den Test an.
func bestellungFuerLieferant(t *testing.T, pool *pgxpool.Pool, lieferantID string) string {
	t.Helper()
	var id string
	err := pool.QueryRow(context.Background(), `
		INSERT INTO bestellungen_verlauf (lieferant_id, lieferant_name, lieferant_email, kundennummer, anzahl_exemplare)
		VALUES ($1, 'Test-Lieferant', 'test@example.invalid', 'K-1', 3) RETURNING id
	`, lieferantID).Scan(&id)
	if err != nil {
		t.Fatalf("Bestellung anlegen: %v", err)
	}
	return id
}

func putBestaetigen(srv *Server, bestellungID, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPut, "/api/bestellungen/"+bestellungID+"/bestaetigen", strings.NewReader(body))
	req.SetPathValue("id", bestellungID)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	srv.BestaetigenBestellungHandler().ServeHTTP(rec, req)
	return rec
}

// Der externe Bestätigungsschritt (Naacher-Link) lässt sich in Bibliosys nachtragen,
// wenn der Lieferant dafür freigeschaltet ist.
func TestBestaetigenBestellung_Erfolg(t *testing.T) {
	pool := pgTestPool(t)
	resetBestandsdaten(t, pool)
	ctx := context.Background()
	srv := &Server{DB: &db.Database{Pool: pool}}

	lieferant := haendlerMitBestaetigung(t, pool, "Naacher", true)
	bestellung := bestellungFuerLieferant(t, pool, lieferant)

	rec := putBestaetigen(srv, bestellung, `{"etiketten_groesse":"gross"}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("Status = %d, want 200, body: %s", rec.Code, rec.Body.String())
	}

	var bestaetigtAmGesetzt bool
	var groesse string
	if err := pool.QueryRow(ctx, `
		SELECT (bestaetigt_am IS NOT NULL), coalesce(etiketten_groesse, '') FROM bestellungen_verlauf WHERE id = $1
	`, bestellung).Scan(&bestaetigtAmGesetzt, &groesse); err != nil {
		t.Fatalf("Bestellung lesen: %v", err)
	}
	if !bestaetigtAmGesetzt {
		t.Error("bestaetigt_am wurde nicht gesetzt")
	}
	if groesse != "gross" {
		t.Errorf("etiketten_groesse = %q, want %q", groesse, "gross")
	}
}

// Ein Lieferant ohne das Flag bietet den Schritt nicht an — Bestätigen muss abgelehnt
// werden, sonst könnte jede Bestellung fälschlich als "extern bestätigt" markiert werden.
func TestBestaetigenBestellung_LieferantOhneFlag(t *testing.T) {
	pool := pgTestPool(t)
	resetBestandsdaten(t, pool)
	srv := &Server{DB: &db.Database{Pool: pool}}

	lieferant := haendlerMitBestaetigung(t, pool, "Normalhaendler", false)
	bestellung := bestellungFuerLieferant(t, pool, lieferant)

	rec := putBestaetigen(srv, bestellung, `{"etiketten_groesse":"klein"}`)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("Status = %d, want 400, body: %s", rec.Code, rec.Body.String())
	}
}

// Ein zweites Bestätigen derselben Bestellung darf die erste Angabe nicht stillschweigend
// überschreiben — das wäre ein falsches Bild vom tatsächlichen externen Vorgang.
func TestBestaetigenBestellung_DoppeltBestaetigenGibt409(t *testing.T) {
	pool := pgTestPool(t)
	resetBestandsdaten(t, pool)
	srv := &Server{DB: &db.Database{Pool: pool}}

	lieferant := haendlerMitBestaetigung(t, pool, "Naacher2", true)
	bestellung := bestellungFuerLieferant(t, pool, lieferant)

	if rec := putBestaetigen(srv, bestellung, `{"etiketten_groesse":"klein"}`); rec.Code != http.StatusOK {
		t.Fatalf("erste Bestätigung: Status = %d, body: %s", rec.Code, rec.Body.String())
	}

	rec := putBestaetigen(srv, bestellung, `{"etiketten_groesse":"gross"}`)
	if rec.Code != http.StatusConflict {
		t.Fatalf("Status = %d, want 409, body: %s", rec.Code, rec.Body.String())
	}
}

// Ein ungültiger Wert für etiketten_groesse muss vor jeder DB-Abfrage abgelehnt werden.
func TestBestaetigenBestellung_UngueltigeGroesse(t *testing.T) {
	pool := pgTestPool(t)
	resetBestandsdaten(t, pool)
	srv := &Server{DB: &db.Database{Pool: pool}}

	lieferant := haendlerMitBestaetigung(t, pool, "Naacher3", true)
	bestellung := bestellungFuerLieferant(t, pool, lieferant)

	rec := putBestaetigen(srv, bestellung, `{"etiketten_groesse":"mittel"}`)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("Status = %d, want 400, body: %s", rec.Code, rec.Body.String())
	}
}
