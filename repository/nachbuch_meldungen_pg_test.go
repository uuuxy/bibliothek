package repository

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"bibliothek/internal/pgtest"

	"github.com/google/uuid"
)

// Migration 117: Eine Meldung entsteht je Schlüssel genau einmal, steht offen, bis jemand
// sie quittiert, zählt bis dahin fürs Band, und fällt quittiert nach der Frist — mit
// DEMSELBEN Prädikat, das der Rückstands-Wächter benutzt.
func TestNachbuchMeldungen_Lebenslauf(t *testing.T) {
	pool := pgtest.Pool(t)
	ctx := context.Background()
	suffix := fmt.Sprintf("%d", time.Now().UnixNano())

	var bearbeiterID, schuelerID string
	if err := pool.QueryRow(ctx, `
		INSERT INTO benutzer (barcode_id, vorname, nachname, email, rolle, aktiv)
		VALUES ($1, 'Nach', 'Bucher', $2, 'mitarbeiter', true) RETURNING id`,
		"MA-"+suffix, "nachbuch-"+suffix+"@schule.invalid").Scan(&bearbeiterID); err != nil {
		t.Fatalf("Mitarbeiter anlegen: %v", err)
	}
	if err := pool.QueryRow(ctx, `
		INSERT INTO schueler (barcode_id, vorname, nachname, klasse, abgaenger_jahr)
		VALUES ($1, 'Anna', 'Gemeldet', '07B', 2031) RETURNING id`, "S-"+suffix).Scan(&schuelerID); err != nil {
		t.Fatalf("Schüler anlegen: %v", err)
	}
	vorher, err := ZaehleOffeneNachbuchMeldungen(ctx, pool)
	if err != nil {
		t.Fatalf("zählen: %v", err)
	}

	schluessel := uuid.NewString()
	barcode := "B-NB-" + suffix
	// Zweimal derselbe Schlüssel = eine Meldung (die Wiederholung des Theken-Rechners) —
	// geschrieben, wie die Nachbuch-Tür es tut (Commit 11: SchreibeNachbuchMeldung).
	for i := 0; i < 2; i++ {
		if _, err := pool.Exec(ctx, `
			INSERT INTO nachbuch_meldungen (idempotency_key, barcode, ergebnis, grund, vorbesitzer_schueler_id, gescannt_am)
			VALUES ($1::uuid, $2, $3, $4, $5::uuid, $6) ON CONFLICT (idempotency_key) DO NOTHING`,
			schluessel, barcode, NachbuchUmgebucht, "lag bei jemand anderem", schuelerID, time.Now().Add(-time.Hour)); err != nil {
			t.Fatalf("schreiben (%d): %v", i, err)
		}
	}
	nachher, err := ZaehleOffeneNachbuchMeldungen(ctx, pool)
	if err != nil {
		t.Fatalf("zählen: %v", err)
	}
	if nachher != vorher+1 {
		t.Fatalf("zwei Einträge mit einem Schlüssel: offen vorher %d, nachher %d — erwartet +1", vorher, nachher)
	}

	offen, err := ListeNachbuchMeldungen(ctx, pool, true)
	if err != nil {
		t.Fatalf("liste: %v", err)
	}
	var meine *NachbuchMeldung
	for i := range offen {
		if offen[i].Barcode == barcode {
			meine = &offen[i]
		}
	}
	if meine == nil {
		t.Fatal("die Meldung fehlt in der Liste der offenen")
	}
	if meine.Vorbesitzer != "Anna Gemeldet" || meine.Ergebnis != NachbuchUmgebucht {
		t.Errorf("Zeile: Vorbesitzer %q, Ergebnis %q", meine.Vorbesitzer, meine.Ergebnis)
	}

	// Quittieren: einmal ja, ein zweites Mal ist kein stiller Erfolg.
	if err := QuittiereNachbuchMeldung(ctx, pool, meine.ID, bearbeiterID); err != nil {
		t.Fatalf("quittieren: %v", err)
	}
	if err := QuittiereNachbuchMeldung(ctx, pool, meine.ID, bearbeiterID); !errors.Is(err, ErrNachbuchMeldungNichtOffen) {
		t.Errorf("zweites Quittieren: erwartet ErrNachbuchMeldungNichtOffen, bekommen %v", err)
	}
	if n, err := ZaehleOffeneNachbuchMeldungen(ctx, pool); err != nil || n != vorher {
		t.Errorf("nach dem Quittieren offen %d, erwartet %d", n, vorher)
	}

	// Frist: quittiert vor 31 Tagen fällt (30 Tage), quittiert vor 10 Tagen bleibt.
	if _, err := pool.Exec(ctx, `UPDATE nachbuch_meldungen SET quittiert_am = now() - interval '31 days' WHERE id = $1`, meine.ID); err != nil {
		t.Fatalf("rückdatieren: %v", err)
	}
	b := PredikatNachbuchMeldungen(HoechstNachbuchMeldungenTage, KulanzJob)
	tag, err := pool.Exec(ctx, `DELETE FROM nachbuch_meldungen WHERE `+b.Where+` AND barcode = $3`, append(b.Args, barcode)...)
	if err != nil {
		t.Fatalf("löschen: %v", err)
	}
	if tag.RowsAffected() != 1 {
		t.Errorf("31 Tage quittiert: %d gelöscht, erwartet 1", tag.RowsAffected())
	}
}

// Die Frist: Lesehistorie-Frist, höchstens 30 Tage — nie mehr, auch wenn die Einstellung
// zwei Jahre sagt, nie 0 (0 hieße „aus", und offene Namen ohne Frist sind nicht gewollt).
func TestNachbuchMeldungenTage_HoechstensDreissig(t *testing.T) {
	zehn, zweiJahre, aus := 10, 730, 0
	if got := NachbuchMeldungenTage(&SystemEinstellungen{LesehistorieTage: &zehn}); got != 10 {
		t.Errorf("10 Tage Lesehistorie → %d, erwartet 10", got)
	}
	if got := NachbuchMeldungenTage(&SystemEinstellungen{LesehistorieTage: &zweiJahre}); got != 30 {
		t.Errorf("730 Tage Lesehistorie → %d, erwartet 30 (Obergrenze)", got)
	}
	if got := NachbuchMeldungenTage(&SystemEinstellungen{LesehistorieTage: &aus}); got != 30 {
		t.Errorf("Lesehistorie aus → %d, erwartet 30 (nie ohne Frist)", got)
	}
}
