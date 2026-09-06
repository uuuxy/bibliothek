package jobs

import (
	"context"
	"os"
	"testing"

	"bibliothek/repository"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Die Karenz-Uhr am nächtlichen Job (Rasterdurchgang 02.09.2026, Frage 7): Kein Test
// fuhr RunGDPRAnonymizeOldData über eine Zeile mit altem abgaenger_seit und frischem
// aktualisiert_am — das alte Prädikat (nur aktualisiert_am) und ein fest verdrahtetes
// 360 wären unter allen Tests grün geblieben. Hier: die Uhr ist abgaenger_seit, nicht
// aktualisiert_am, und die Frist kommt aus der Einstellung abgaenger_karenz_tage.
func TestGDPRAnonymize_KarenzUhrUndEinstellung(t *testing.T) {
	adminDSN := os.Getenv(drillEnvVar)
	if adminDSN == "" {
		t.Skipf("%s nicht gesetzt — Test übersprungen", drillEnvVar)
	}
	_, dsn := legeProbeDatenbankAn(t, adminDSN, "karenz")
	befuelleQuelle(t, dsn)
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatalf("Pool: %v", err)
	}
	t.Cleanup(pool.Close)

	lege := func(barcode string, seit, aktualisiert string) string {
		t.Helper()
		var id string
		if err := pool.QueryRow(ctx, `
			INSERT INTO schueler (barcode_id, vorname, nachname, klasse, abgaenger_jahr, ist_abgaenger, ist_gesperrt, block_reason, abgaenger_seit, aktualisiert_am)
			VALUES ($1, 'Karenz', $1, 'ABG', 2026, true, true, $4, NOW() - $2::interval, NOW() - $3::interval)
			RETURNING id`, barcode, seit, aktualisiert, repository.AbgaengerSperrgrundKarenz).Scan(&id); err != nil {
			t.Fatal(err)
		}
		return id
	}
	setzeKarenz := func(tage string) {
		t.Helper()
		if _, err := pool.Exec(ctx, `INSERT INTO system_einstellungen (schluessel, wert) VALUES ($1, $2)
			ON CONFLICT (schluessel) DO UPDATE SET wert = EXCLUDED.wert`, repository.AbgaengerKarenzSchluessel, tage); err != nil {
			t.Fatal(err)
		}
	}
	anonymisiert := func(id string) bool {
		t.Helper()
		var ja bool
		if err := pool.QueryRow(ctx, `SELECT anonymized_at IS NOT NULL FROM schueler WHERE id = $1`, id).Scan(&ja); err != nil {
			t.Fatal(err)
		}
		return ja
	}

	// Uhr: A ist seit 100 Tagen Abgänger, aber gestern angefasst; B ist seit heute
	// Abgänger, aber seit 400 Tagen unberührt. Bei Karenz 90 fällt A, B bleibt.
	setzeKarenz("90")
	a := lege("KZ-A", "100 days", "1 day")
	b := lege("KZ-B", "0 days", "400 days")
	s := NewScheduler(pool, repository.NewAuditRepository(pool))
	s.RunGDPRAnonymizeOldData()
	if !anonymisiert(a) {
		t.Error("A (abgaenger_seit vor 100 Tagen) muss bei Karenz 90 anonymisiert sein — rechnet der Job mit aktualisiert_am?")
	}
	if anonymisiert(b) {
		t.Error("B (abgaenger_seit heute) darf nicht anonymisiert sein — rechnet der Job mit aktualisiert_am?")
	}

	// Einstellung: dieselbe Zeile wie A überlebt eine Karenz von 200 Tagen.
	setzeKarenz("200")
	c := lege("KZ-C", "100 days", "1 day")
	s.RunGDPRAnonymizeOldData()
	if anonymisiert(c) {
		t.Error("C darf bei Karenz 200 nicht anonymisiert sein — liest der Job die Einstellung?")
	}
}

// Unlesbare Einstellungen dürfen NICHT anonymisieren (Rasterdurchgang 06.09.2026).
//
// Die Vorgabe-Karenz von 90 Tagen sieht nach einem sicheren Rückfall aus und ist der
// gefährlichste Wert: Eine KLEINERE Karenz wählt MEHR Zeilen. Bei eingestellten 200 Tagen
// hätte ein einziger Lesefehler alle Abgänger zwischen Tag 91 und 200 anonymisiert —
// unwiederbringlich, und der Löschjob räumt die Hülle in derselben Nacht.
//
// Der Lesefehler wird hier echt hergestellt (die Tabelle ist kurz weg), nicht gemockt:
// Nur so läuft der Job über denselben Weg wie im Betrieb.
func TestGDPRAnonymize_OhneEinstellungenWirdNichtsAnonymisiert(t *testing.T) {
	adminDSN := os.Getenv(drillEnvVar)
	if adminDSN == "" {
		t.Skipf("%s nicht gesetzt — Test übersprungen", drillEnvVar)
	}
	_, dsn := legeProbeDatenbankAn(t, adminDSN, "karenzfehler")
	befuelleQuelle(t, dsn)
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatalf("Pool: %v", err)
	}
	t.Cleanup(pool.Close)

	if _, err := pool.Exec(ctx, `INSERT INTO system_einstellungen (schluessel, wert) VALUES ($1, '200')
		ON CONFLICT (schluessel) DO UPDATE SET wert = EXCLUDED.wert`, repository.AbgaengerKarenzSchluessel); err != nil {
		t.Fatal(err)
	}
	var id string
	if err := pool.QueryRow(ctx, `
		INSERT INTO schueler (barcode_id, vorname, nachname, klasse, abgaenger_jahr, ist_abgaenger, ist_gesperrt, block_reason, abgaenger_seit, aktualisiert_am)
		VALUES ('KZF-1', 'Karenz', 'Fehler', 'ABG', 2026, true, true, $1, NOW() - '150 days'::interval, NOW() - '150 days'::interval)
		RETURNING id`, repository.AbgaengerSperrgrundKarenz).Scan(&id); err != nil {
		t.Fatal(err)
	}

	// Einstellungen unlesbar machen — die Zeile liegt zwischen Vorgabe (90) und
	// eingestellter Karenz (200), fällt also genau dann, wenn der Job die Vorgabe nimmt.
	if _, err := pool.Exec(ctx, `ALTER TABLE system_einstellungen RENAME TO system_einstellungen_weg`); err != nil {
		t.Fatalf("Tabelle umbenennen: %v", err)
	}
	s := NewScheduler(pool, repository.NewAuditRepository(pool))
	s.RunGDPRAnonymizeOldData()
	if _, err := pool.Exec(ctx, `ALTER TABLE system_einstellungen_weg RENAME TO system_einstellungen`); err != nil {
		t.Fatalf("Tabelle zurückbenennen: %v", err)
	}

	var anonym bool
	if err := pool.QueryRow(ctx, `SELECT anonymized_at IS NOT NULL FROM schueler WHERE id = $1`, id).Scan(&anonym); err != nil {
		t.Fatal(err)
	}
	if anonym {
		t.Error("Ohne lesbare Einstellungen wurde anonymisiert — die Vorgabe 90 hat die eingestellten 200 Tage überstimmt")
	}

	// Gegenprobe: mit lesbarer Einstellung und Karenz 90 fällt dieselbe Zeile.
	if _, err := pool.Exec(ctx, `UPDATE system_einstellungen SET wert = '90' WHERE schluessel = $1`,
		repository.AbgaengerKarenzSchluessel); err != nil {
		t.Fatal(err)
	}
	s.RunGDPRAnonymizeOldData()
	if err := pool.QueryRow(ctx, `SELECT anonymized_at IS NOT NULL FROM schueler WHERE id = $1`, id).Scan(&anonym); err != nil {
		t.Fatal(err)
	}
	if !anonym {
		t.Error("Gegenprobe: bei Karenz 90 muss die 150 Tage alte Zeile anonymisiert werden")
	}
}
