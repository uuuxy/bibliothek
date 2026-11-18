package repository

import (
	"context"
	"os"
	"path/filepath"
	"regexp"
	"testing"
	"time"

	"bibliothek/internal/pgtest"
)

// Migration 180 bringt gespeicherte Zeilen auf die Form, die das Programm jetzt schreibt: den
// Sperrgrund der Ehemaligen ohne veraltenden Zustand und die eine Form der Anonymisierung.
//
// Geprüft wird die echte Migrationsdatei, zweimal hintereinander, in einer Transaktion, die am
// Ende zurückgerollt wird. Ein Grund von Hand und ein Schüler, der wirklich Abgänger heißt,
// bleiben, wie sie sind; eine Zeile ohne abgaenger_seit behält ihre Uhr der Karenz.
func TestSperrgrundDerEhemaligen_Migration180(t *testing.T) {
	pool := pgtest.Pool(t)
	ctx := context.Background()
	tx := beginne(t, pool)
	defer func() {
		if err := tx.Rollback(ctx); err != nil {
			t.Errorf("zurückrollen: %v", err)
		}
	}()

	alteUhr := time.Now().Add(-40 * 24 * time.Hour).Truncate(time.Second)
	zeile := func(barcode, vorname, nachname, klasse string, abgaenger, mitSeit, anonymisiert bool, grund string) string {
		t.Helper()
		var id string
		if err := tx.QueryRow(ctx, `
			INSERT INTO schueler (barcode_id, vorname, nachname, klasse, abgaenger_jahr, ist_abgaenger,
			                      ist_gesperrt, block_reason, abgaenger_seit, aktualisiert_am, anonymized_at)
			VALUES ($1, $2, $3, $4, 2026, $5, true, $6,
			        CASE WHEN $7 THEN NOW() - interval '10 days' END, $8,
			        CASE WHEN $9 THEN NOW() END)
			RETURNING id::text`,
			barcode, vorname, nachname, klasse, abgaenger, grund, mitSeit, alteUhr, anonymisiert).Scan(&id); err != nil {
			t.Fatalf("Zeile %s anlegen: %v", barcode, err)
		}
		return id
	}
	offen := zeile("M180-OFFEN", "Ole", "Offen", "07A", true, true, false, "Automatisierte Abgänger-Sperre wegen offener Vorgänge")
	karenz := zeile("M180-KARENZ", "Kai", "Karenz", "07A", true, false, false, "Automatisierte Abgänger-Sperre in der Karenzzeit vor der Anonymisierung")
	versetzt := zeile("M180-SJW", "Sina", "Versetzt", "ABG", true, true, false, "Automatisierte Abgänger-Sperre zum Schuljahreswechsel")
	rueckkehr := zeile("M180-RUECK", "Rita", "Zurueck", "08A", false, false, false, "Automatisierte Abgänger-Sperre wegen offener Vorgänge")
	vonHand := zeile("M180-HAND", "Hanna", "Hand", "07A", true, true, false, "Ausweis gestohlen")
	anonym := zeile("ANON-M180", "Abgänger", "Anonymisiert-alt", "ABG", true, true, true, "Abgänger anonymisiert")
	echt := zeile("M180-ECHT", "Abgänger", "Anonymisiert-Echt", "07A", true, true, false, "Ausweis verloren")

	migration, err := os.ReadFile(filepath.Join("..", "..", "migrations", "180_sperrgrund_der_ehemaligen.sql"))
	if err != nil {
		t.Fatalf("Migration lesen: %v", err)
	}
	for lauf := 1; lauf <= 2; lauf++ {
		if _, err := tx.Exec(ctx, string(migration)); err != nil {
			t.Fatalf("Lauf %d: Migration scheitert: %v", lauf, err)
		}
	}

	type stand struct {
		vorname, nachname, klasse, grund string
		seit                             *time.Time
	}
	lies := func(id string) stand {
		t.Helper()
		var s stand
		if err := tx.QueryRow(ctx,
			`SELECT vorname, nachname, klasse, COALESCE(block_reason, ''), abgaenger_seit FROM schueler WHERE id = $1`, id).
			Scan(&s.vorname, &s.nachname, &s.klasse, &s.grund, &s.seit); err != nil {
			t.Fatalf("Zeile lesen: %v", err)
		}
		return s
	}

	for name, id := range map[string]string{"offen": offen, "karenz": karenz, "versetzt": versetzt} {
		if g := lies(id).grund; g != AbgaengerSperrgrund {
			t.Errorf("%s: Grund %q, erwartet %q", name, g, AbgaengerSperrgrund)
		}
	}
	if g := lies(rueckkehr).grund; g != AbgaengerSperrgrundOffen {
		t.Errorf("Rückkehrer mit offenen Vorgängen: Grund %q, erwartet %q", g, AbgaengerSperrgrundOffen)
	}
	if s := lies(rueckkehr); s.seit != nil {
		t.Errorf("Rückkehrer: abgaenger_seit = %v, erwartet leer — er ist nicht weg", s.seit)
	}
	// Die Uhr der Karenz rechnete ohne abgaenger_seit mit aktualisiert_am; die Migration hält
	// diesen Wert fest, bevor sie aktualisiert_am verschiebt.
	if s := lies(karenz); s.seit == nil || !s.seit.Equal(alteUhr) {
		t.Errorf("Ehemaliger ohne abgaenger_seit: Uhr steht auf %v, erwartet den alten Stempel %v", s.seit, alteUhr)
	}
	for name, id := range map[string]string{"von Hand": vonHand, "echter Name": echt} {
		s := lies(id)
		if s.grund == AbgaengerSperrgrund || s.nachname == "Anonym" {
			t.Errorf("%s: Zeile wurde angefasst: %+v", name, s)
		}
	}
	s := lies(anonym)
	if !regexp.MustCompile(`^[0-9a-f]{8}$`).MatchString(s.vorname) || s.nachname != "Anonym" || s.klasse != "" ||
		s.grund != "Anonymisiert (DSGVO)" {
		t.Errorf("anonymisierte Zeile in alter Form: %+v, erwartet die Form von sqlAnonymisierung", s)
	}
}
