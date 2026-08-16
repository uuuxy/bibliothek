package repository

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Die Gebühren-Erledigung (Bezahlt/Storno) am echten Postgres: Die 404/409-Sentinels
// und die FOR-UPDATE-Betragslesung sind SQL-Verhalten — pgxmock würde sie nur
// nachspielen. Der Betrag im Audit-Protokoll MUSS aus der Datenbank stammen; genau
// das belegt dieser Test, indem er ihn nirgends übergibt und trotzdem im Log findet.

func seedOffenerSchadensfall(t *testing.T, pool *pgxpool.Pool, schuelerID, exemplarID string, betrag float64) string {
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

func TestGebuehrErledigung_BezahltUndStorno(t *testing.T) {
	pool := pgTestPool(t)
	ctx := context.Background()
	repo := NewAuditRepository(pool)

	schueler := seedSchueler(t, pool, "S-GEB-1", "Mia", "8b")
	exemplare := seedSignaturMitExemplaren(t, pool, "GebErl", 2)

	// Eigener Bearbeiter statt seedBearbeiter: dessen fester Barcode gehört dem
	// Race-Test — ein geteilter Testnutzer macht die Reihenfolge zum Schicksal.
	var bearbeiter string
	if err := pool.QueryRow(ctx,
		`INSERT INTO benutzer (barcode_id, vorname, nachname, email, rolle, aktiv)
		 VALUES ('GEB-B', 'Gebuehren', 'Kraft', 'geb@example.org', 'mitarbeiter', true) RETURNING id`).Scan(&bearbeiter); err != nil {
		t.Fatalf("Bearbeiter anlegen: %v", err)
	}

	// (1) Storno: setzt ist_bezahlt UND die Storno-Spalten, Audit trägt den DB-Betrag.
	storno := seedOffenerSchadensfall(t, pool, schueler, exemplare[0], 12.50)
	if err := repo.StornierungGebuehr(ctx, storno, bearbeiter, "Buch wiedergefunden"); err != nil {
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
	if auditBetrag != "12.5" {
		t.Errorf("Audit-Betrag = %q, erwartet 12.5 (aus der DB, nicht vom Aufrufer)", auditBetrag)
	}

	// (2) Doppel-Storno und Bezahlen eines erledigten Falls → Konflikt-Sentinel.
	if err := repo.StornierungGebuehr(ctx, storno, bearbeiter, "nochmal"); !errors.Is(err, ErrSchadensfallErledigt) {
		t.Errorf("Doppel-Storno: erwartet ErrSchadensfallErledigt, bekam %v", err)
	}
	if err := repo.BezahltGebuehr(ctx, storno, bearbeiter); !errors.Is(err, ErrSchadensfallErledigt) {
		t.Errorf("Bezahlen nach Storno: erwartet ErrSchadensfallErledigt, bekam %v", err)
	}

	// (3) Bezahlt: setzt ist_bezahlt, lässt die Storno-Spalten unberührt —
	// bezahlt und storniert bleiben unterscheidbar (DSGVO-PDF, Kontoauszug).
	bezahlt := seedOffenerSchadensfall(t, pool, schueler, exemplare[1], 7.00)
	if err := repo.BezahltGebuehr(ctx, bezahlt, bearbeiter); err != nil {
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

	// (4) Unbekannte ID → 404-Sentinel, nicht der Konflikt.
	err := repo.BezahltGebuehr(ctx, "00000000-0000-0000-0000-000000000000", bearbeiter)
	if !errors.Is(err, ErrSchadensfallNichtGefunden) {
		t.Errorf("unbekannte ID: erwartet ErrSchadensfallNichtGefunden, bekam %v", err)
	}

	// (5) Die Liste der Schülerakte führt beide Fälle, neueste zuerst.
	liste, err := NewDamageRepository(pool).ListSchadensfaelleVonSchueler(ctx, schueler)
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
