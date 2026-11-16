package repository

import (
	"bibliothek/internal/testdaten"
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Zahlung und Storno einer Forderung am echten Postgres: Die Fehlerwerte für „nicht gefunden"
// und „schon erledigt" und das Lesen des Betrags unter FOR UPDATE sind Verhalten der Datenbank.
// Der Betrag im Protokoll stammt aus der Datenbank; der Test übergibt ihn nirgends und findet
// ihn trotzdem im Eintrag.

func legeOffenenSchadensfallAn(t *testing.T, pool *pgxpool.Pool, schuelerID, exemplarID string, betrag float64) string {
	t.Helper()
	var id string
	if err := pool.QueryRow(context.Background(),
		`INSERT INTO schadensfaelle (exemplar_id, schueler_id, beschreibung, betrag, ist_bezahlt)
		 VALUES ($1, $2, 'Wasserschaden', $3, false) RETURNING id`,
		exemplarID, schuelerID, betrag).Scan(&id); err != nil {
		t.Fatalf("Schadensfall anlegen: %v", err)
	}
	return id
}

func TestForderungErledigen_BezahltUndStorno(t *testing.T) {
	pool := pgTestPool(t)
	ctx := context.Background()

	schueler := legeSchuelerAn(t, pool, "S-GEB-1", "Mia", "8b")
	exemplare := legeSignaturMitExemplarenAn(t, pool, "GebErl", 2)

	// Eigener Bearbeiter statt legeBearbeiterAn: Dessen fester Barcode gehört den Tests in
	// schaden_melden_ausleihe_pg_test.go, und mit einem geteilten Konto hinge das Ergebnis an
	// der Reihenfolge der Tests.
	bearbeiter := testdaten.LegeKontoAn(t, pool, "Gebuehren", "Kraft", "geb@example.org", "mitarbeiter", true)

	// (1) Storno: setzt ist_bezahlt und die Spalten des Stornos, der Eintrag trägt den Betrag
	// aus der Datenbank.
	storno := legeOffenenSchadensfallAn(t, pool, schueler, exemplare[0], 12.50)
	if err := NewSchadensAbfragen(pool).StorniereForderung(ctx, storno, bearbeiter, "Buch wiedergefunden"); err != nil {
		t.Fatalf("Storno: %v", err)
	}
	var istBezahlt bool
	var grund *string
	if err := pool.QueryRow(ctx,
		`SELECT ist_bezahlt, stornierungsgrund FROM schadensfaelle WHERE id = $1`,
		storno).Scan(&istBezahlt, &grund); err != nil {
		t.Fatalf("Storno nachlesen: %v", err)
	}
	if !istBezahlt || grund == nil || *grund != "Buch wiedergefunden" {
		t.Errorf("Storno-Zustand: ist_bezahlt=%v grund=%v", istBezahlt, grund)
	}
	var auditBetrag string
	if err := pool.QueryRow(ctx,
		`SELECT details->>'betrag' FROM audit_log
		 WHERE tabelle = 'schadensfaelle' AND aktion = 'STORNIERUNG' AND datensatz_id = $1::uuid`,
		storno).Scan(&auditBetrag); err != nil {
		t.Fatalf("Storno-Audit fehlt: %v", err)
	}
	if auditBetrag != "12.50" {
		t.Errorf("Audit-Betrag = %q, erwartet 12.50 (aus der DB, nicht vom Aufrufer)", auditBetrag)
	}

	// (2) Ein zweites Storno und das Bezahlen eines erledigten Falls liefern
	// ErrSchadensfallErledigt.
	if err := NewSchadensAbfragen(pool).StorniereForderung(ctx, storno, bearbeiter, "nochmal"); !errors.Is(err, ErrSchadensfallErledigt) {
		t.Errorf("Doppel-Storno = %v, erwartet ErrSchadensfallErledigt", err)
	}
	if err := NewSchadensAbfragen(pool).VerbucheZahlung(ctx, storno, bearbeiter); !errors.Is(err, ErrSchadensfallErledigt) {
		t.Errorf("Bezahlen nach Storno = %v, erwartet ErrSchadensfallErledigt", err)
	}

	// (3) Bezahlt: setzt ist_bezahlt und lässt die Spalten des Stornos unberührt. Bezahlt und
	// storniert bleiben unterscheidbar (Auskunft, Kontoauszug).
	bezahlt := legeOffenenSchadensfallAn(t, pool, schueler, exemplare[1], 7.00)
	if err := NewSchadensAbfragen(pool).VerbucheZahlung(ctx, bezahlt, bearbeiter); err != nil {
		t.Fatalf("Bezahlen: %v", err)
	}
	var bezahltFlag bool
	var storniertAm *string
	if err := pool.QueryRow(ctx,
		`SELECT ist_bezahlt, storniert_am::text FROM schadensfaelle WHERE id = $1`,
		bezahlt).Scan(&bezahltFlag, &storniertAm); err != nil {
		t.Fatalf("Zahlung nachlesen: %v", err)
	}
	if !bezahltFlag || storniertAm != nil {
		t.Errorf("Zahlung: ist_bezahlt=%v storniert_am=%v (muss NULL bleiben)", bezahltFlag, storniertAm)
	}

	// (4) Eine unbekannte ID liefert ErrSchadensfallNichtGefunden und nicht den Konflikt.
	err := NewSchadensAbfragen(pool).VerbucheZahlung(ctx, "00000000-0000-0000-0000-000000000000", bearbeiter)
	if !errors.Is(err, ErrSchadensfallNichtGefunden) {
		t.Errorf("unbekannte ID = %v, erwartet ErrSchadensfallNichtGefunden", err)
	}

	// (5) Die Liste der Schülerakte führt beide Fälle, neueste zuerst.
	liste, err := NewSchadensAbfragen(pool).LadeZuLeser(ctx, schueler)
	if err != nil {
		t.Fatalf("Liste: %v", err)
	}
	if len(liste) != 2 {
		t.Fatalf("Liste: %d Einträge, erwartet 2", len(liste))
	}
	if liste[0].ID != bezahlt || liste[0].StorniertAm != nil || liste[1].StorniertAm == nil {
		t.Errorf("Liste: Reihenfolge/Storno-Kennzeichnung falsch: %+v", liste)
	}
}
