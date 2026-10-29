package bestellung

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"bibliothek/api/tuer"
	"bibliothek/db"
	"bibliothek/internal/pgtest"
	"bibliothek/internal/testdaten"
	"bibliothek/internal/tuertest"

	"github.com/jackc/pgx/v5/pgxpool"
)

// bestellungFuerLieferant legt eine bestellungen_verlauf-Zeile für den Test an.
//
// Den Bestätigungs-Token bekommt die Bestellung genau dann, wenn ihr Lieferant der
// Hauptlieferant ist — dieselbe Kopplung wie in schreibeBestellkopf. Ohne sie prüfte der
// Test einen Zustand, den der Produktivcode nie erzeugt: eine Bestellung bei einem
// Hauptlieferanten, die selbst gar keinen Link bekommen hat.
func bestellungFuerLieferant(t *testing.T, pool *pgxpool.Pool, lieferantID string) string {
	t.Helper()
	var id string
	err := pool.QueryRow(context.Background(), `
		INSERT INTO bestellungen_verlauf
			(lieferant_id, lieferant_name, lieferant_email, kundennummer, anzahl_exemplare,
			 bestaetigungs_token_hash, token_gueltig_bis)
		SELECT $1, 'Test-Lieferant', 'test@example.invalid', 'K-1', 3,
		       CASE WHEN l.ist_hauptlieferant THEN encode(sha256(gen_random_uuid()::text::bytea), 'hex') END,
		       CASE WHEN l.ist_hauptlieferant THEN now() + make_interval(days => 30) END
		FROM lieferanten l WHERE l.id = $1
		RETURNING id
	`, lieferantID).Scan(&id)
	if err != nil {
		t.Fatalf("Bestellung anlegen: %v", err)
	}
	return id
}

func bestaetigeVonHand(srv *Tueren, bestellungID, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPut, "/api/bestellungen/"+bestellungID+"/bestaetigen", strings.NewReader(body))
	req.SetPathValue("id", bestellungID)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	srv.BestellungBestaetigen().ServeHTTP(rec, req)
	return rec
}

// Der externe Bestätigungsschritt (Naacher-Link) lässt sich in Bibliosys nachtragen,
// wenn der Lieferant dafür freigeschaltet ist.
func TestBestellungBestaetigen_Erfolg(t *testing.T) {
	pool := pgtest.Pool(t)
	testdaten.LeereBestandsdaten(t, pool)
	ctx := context.Background()
	srv := &Tueren{Basis: &tuer.Basis{DB: &db.Verbindung{Pool: pool}}}

	lieferant := testdaten.LegeLieferantAn(t, pool, "Naacher", true)
	bestellung := bestellungFuerLieferant(t, pool, lieferant)

	rec := bestaetigeVonHand(srv, bestellung, `{"etiketten_groesse":"gross"}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("Status = %d, erwartet 200, Rumpf: %s", rec.Code, rec.Body.String())
	}

	var bestaetigtAmGesetzt bool
	var groesse, durch string
	if err := pool.QueryRow(ctx, `
		SELECT (bestaetigt_am IS NOT NULL), coalesce(etiketten_groesse, ''), coalesce(bestaetigt_durch, '')
		FROM bestellungen_verlauf WHERE id = $1
	`, bestellung).Scan(&bestaetigtAmGesetzt, &groesse, &durch); err != nil {
		t.Fatalf("Bestellung lesen: %v", err)
	}
	if !bestaetigtAmGesetzt {
		t.Error("bestaetigt_am wurde nicht gesetzt")
	}
	if groesse != "gross" {
		t.Errorf("etiketten_groesse = %q, erwartet %q", groesse, "gross")
	}
	// Die Historie unterscheidet den Nachtrag aus dem Haus von der Bestätigung des Lieferanten.
	if durch != "bibliothek" {
		t.Errorf("bestaetigt_durch = %q, erwartet %q", durch, "bibliothek")
	}
}

// Ein Lieferant ohne das Flag bietet den Schritt nicht an — Bestätigen muss abgelehnt
// werden, sonst könnte jede Bestellung fälschlich als "extern bestätigt" markiert werden.
func TestBestellungBestaetigen_LieferantOhneFlag(t *testing.T) {
	pool := pgtest.Pool(t)
	testdaten.LeereBestandsdaten(t, pool)
	srv := &Tueren{Basis: &tuer.Basis{DB: &db.Verbindung{Pool: pool}}}

	lieferant := testdaten.LegeLieferantAn(t, pool, "Normalhaendler", false)
	bestellung := bestellungFuerLieferant(t, pool, lieferant)

	rec := bestaetigeVonHand(srv, bestellung, `{"etiketten_groesse":"klein"}`)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("Status = %d, erwartet 400, Rumpf: %s", rec.Code, rec.Body.String())
	}
}

// Ein zweites Bestätigen derselben Bestellung darf die erste Angabe nicht stillschweigend
// überschreiben — das wäre ein falsches Bild vom tatsächlichen externen Vorgang.
func TestBestellungBestaetigen_DoppeltBestaetigenGibt409(t *testing.T) {
	pool := pgtest.Pool(t)
	testdaten.LeereBestandsdaten(t, pool)
	srv := &Tueren{Basis: &tuer.Basis{DB: &db.Verbindung{Pool: pool}}}

	lieferant := testdaten.LegeLieferantAn(t, pool, "Naacher2", true)
	bestellung := bestellungFuerLieferant(t, pool, lieferant)

	if rec := bestaetigeVonHand(srv, bestellung, `{"etiketten_groesse":"klein"}`); rec.Code != http.StatusOK {
		t.Fatalf("erste Bestätigung: Status = %d, body: %s", rec.Code, rec.Body.String())
	}

	rec := bestaetigeVonHand(srv, bestellung, `{"etiketten_groesse":"gross"}`)
	if rec.Code != http.StatusConflict {
		t.Fatalf("Status = %d, erwartet 409, Rumpf: %s", rec.Code, rec.Body.String())
	}
}

// Ein ungültiger Wert für etiketten_groesse muss vor jeder DB-Abfrage abgelehnt werden.
func TestBestellungBestaetigen_UngueltigeGroesse(t *testing.T) {
	pool := pgtest.Pool(t)
	testdaten.LeereBestandsdaten(t, pool)
	srv := &Tueren{Basis: &tuer.Basis{DB: &db.Verbindung{Pool: pool}}}

	lieferant := testdaten.LegeLieferantAn(t, pool, "Naacher3", true)
	bestellung := bestellungFuerLieferant(t, pool, lieferant)

	rec := bestaetigeVonHand(srv, bestellung, `{"etiketten_groesse":"mittel"}`)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("Status = %d, erwartet 400, Rumpf: %s", rec.Code, rec.Body.String())
	}
}

// Ein Raster, das es nicht gibt, und eine unbekannte Bestellung weist der Nachtrag ab, ohne zu
// schreiben. Die unbekannte Kennung ist ein 404 und keine 500: Der Sanitizer ersetzte deren
// Meldung, und die Historie zeigte einen Serverfehler statt „nicht gefunden".
func TestBestellungBestaetigen_UnbekanntesRasterUndUnbekannteBestellung(t *testing.T) {
	pool := pgtest.Pool(t)
	testdaten.LeereBestandsdaten(t, pool)
	srv := &Tueren{Basis: &tuer.Basis{DB: &db.Verbindung{Pool: pool}}}

	lieferant := testdaten.LegeLieferantAn(t, pool, "Naacher4", true)
	bestellung := bestellungFuerLieferant(t, pool, lieferant)

	rec := bestaetigeVonHand(srv, bestellung, `{"etiketten_groesse":"klein","etiketten_format":"gibtesnicht"}`)
	if rec.Code != http.StatusBadRequest || tuertest.Fehlermeldung(t, rec) != "Unbekanntes Format der Etiketten" {
		t.Errorf("unbekanntes Raster: Status %d, erwartet 400 mit Hinweis: %s", rec.Code, rec.Body.String())
	}
	if n := pgtest.ZaehleZeilen(t, pool, `SELECT count(*) FROM bestellungen_verlauf WHERE id = $1 AND bestaetigt_am IS NULL`, bestellung); n != 1 {
		t.Errorf("nach dem abgewiesenen Nachtrag ist die Bestellung nicht mehr offen")
	}

	rec = bestaetigeVonHand(srv, "00000000-0000-0000-0000-00000000dead", `{"etiketten_groesse":"klein"}`)
	if rec.Code != http.StatusNotFound || tuertest.Fehlermeldung(t, rec) != "Bestellung nicht gefunden" {
		t.Errorf("unbekannte Bestellung: Status %d, erwartet 404 mit Hinweis: %s", rec.Code, rec.Body.String())
	}
}
