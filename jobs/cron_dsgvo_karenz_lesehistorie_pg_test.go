package jobs

import (
	"context"
	"os"
	"testing"

	"bibliothek/repository"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Der Lesehistorie-Lauf darf die Karenz-Uhr nicht verstellen (OFFEN.md 4.12,
// freigegeben am 22.09.2026).
//
// Zwei Fristen, die sich nichts angehen sollten: Die KARENZ ist das Fenster, in dem eine
// falsche Zuordnung an der Theke noch auffällt — sie läuft ab dem letzten abgeschlossenen
// Vorgang. Die LESEHISTORIE ist die Frist, nach der eine zurückgegebene Ausleihe ihre
// Person verliert (`schueler_id = NULL`). Bis Migration 137 rechnete die Karenz-Uhr den
// letzten Vorgang über genau diese Spalte aus. Ist die Karenz länger eingestellt als die
// Lesehistorie — hier 90 gegen 1 Tag, beide sind einstellbar —, nahm der Lesehistorie-Lauf
// der Karenz ihre Grundlage: Die Uhr fiel auf den Abgang zurück, und die Zeile wurde in
// derselben Nacht anonymisiert, obwohl 85 Tage Fenster eingestellt waren.
//
// Der Lauf ist derselbe wie im Betrieb (RunLesehistorieBefristung, dann
// RunGDPRAnonymizeOldData) und in derselben Reihenfolge wie im Cron.
//
// Am Stand vor Migration 137 ist dieser Test ROT: RUECK wird anonymisiert.
func TestGDPRAnonymize_LesehistorieVerstelltKarenzUhrNicht(t *testing.T) {
	adminDSN := os.Getenv(drillEnvVar)
	if adminDSN == "" {
		t.Skipf("%s nicht gesetzt — Test übersprungen", drillEnvVar)
	}
	_, dsn := legeProbeDatenbankAn(t, adminDSN, "karenzlesehist")
	befuelleQuelle(t, dsn)
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatalf("Pool: %v", err)
	}
	t.Cleanup(pool.Close)

	// Karenz 90 Tage, Lesehistorie 1 Tag: Die Rückgabe vor 5 Tagen wird getrennt, während
	// die Karenz noch 85 Tage laufen muss. Genau die Paarung, die es im Betrieb gibt,
	// sobald jemand die Lesehistorie kürzer stellt als die Karenz.
	setze := func(schluessel, wert string) {
		t.Helper()
		if _, err := pool.Exec(ctx, `INSERT INTO system_einstellungen (schluessel, wert) VALUES ($1, $2)
			ON CONFLICT (schluessel) DO UPDATE SET wert = EXCLUDED.wert`, schluessel, wert); err != nil {
			t.Fatal(err)
		}
	}
	setze(repository.AbgaengerKarenzSchluessel, "90")
	setze("lesehistorie_tage", "1")
	setze("lesehistorie_lernmittel_tage", "1")

	legeAbgaenger := func(barcode string) string {
		t.Helper()
		var id string
		if err := pool.QueryRow(ctx, `
			INSERT INTO schueler (barcode_id, vorname, nachname, klasse, abgaenger_jahr, ist_abgaenger,
			                      ist_gesperrt, block_reason, abgaenger_seit, aktualisiert_am)
			VALUES ($1, 'Karenz', $1, 'ABG', 2026, true, true, $2,
			        NOW() - interval '100 days', NOW() - interval '400 days')
			RETURNING id`, barcode, repository.AbgaengerSperrgrundKarenz).Scan(&id); err != nil {
			t.Fatal(err)
		}
		return id
	}
	legeExemplar := func(barcode string) string {
		t.Helper()
		var id string
		if err := pool.QueryRow(ctx, `
			WITH titel AS (INSERT INTO buecher_titel (titel) VALUES ('Karenz-Titel ' || $1) RETURNING id)
			INSERT INTO buecher_exemplare (titel_id, barcode_id) SELECT id, $1 FROM titel RETURNING id`,
			barcode).Scan(&id); err != nil {
			t.Fatal(err)
		}
		return id
	}
	legeAusleihe := func(schuelerID, exemplarID, ausgeliehenVor, rueckgabeVor string) {
		t.Helper()
		if _, err := pool.Exec(ctx, `
			INSERT INTO ausleihen (exemplar_id, schueler_id, ausgeliehen_am, rueckgabe_frist, rueckgabe_am)
			VALUES ($1, $2, NOW() - $3::interval, NOW() - $3::interval + interval '21 days',
			        NOW() - $4::interval)`,
			exemplarID, schuelerID, ausgeliehenVor, rueckgabeVor); err != nil {
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

	// RUECK: Rückgabe vor 5 Tagen — 85 Tage Karenz stehen noch aus.
	rueck := legeAbgaenger("KL-RUECK")
	legeAusleihe(rueck, legeExemplar("KL-RUECK-EX"), "150 days", "5 days")
	// ALT: Rückgabe vor 95 Tagen — die Karenz ist auch mit der neuen Uhr abgelaufen.
	// Sichert, dass der Stempel nicht einfach jeden vor der Anonymisierung bewahrt.
	alt := legeAbgaenger("KL-ALT")
	legeAusleihe(alt, legeExemplar("KL-ALT-EX"), "200 days", "95 days")

	s := NewScheduler(pool, repository.NewAuditRepository(pool))
	s.RunLesehistorieBefristung()

	// Die Vorbedingung des Tests: Der Lauf hat die Zuordnung wirklich genommen. Ohne
	// diese Probe könnte der Test grün sein, weil gar nichts getrennt wurde.
	var nochZugeordnet int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM ausleihen WHERE schueler_id = $1`, rueck).Scan(&nochZugeordnet); err != nil {
		t.Fatal(err)
	}
	if nochZugeordnet != 0 {
		t.Fatalf("Vorbedingung: Der Lesehistorie-Lauf hat die Ausleihe nicht getrennt (%d noch zugeordnet) — der Test prüft dann nichts", nochZugeordnet)
	}

	s.RunGDPRAnonymizeOldData()

	if anonymisiert(rueck) {
		t.Error("RUECK: Rückgabe vor 5 Tagen, Karenz 90 — die Zeile muss noch 85 Tage stehen bleiben. " +
			"Anonymisiert: Der Lesehistorie-Lauf hat der Karenz-Uhr ihre Grundlage genommen (OFFEN.md 4.12)")
	}
	if !anonymisiert(alt) {
		t.Error("ALT: Rückgabe vor 95 Tagen liegt außerhalb der Karenz 90 — muss anonymisiert sein")
	}
}
