package jobs

import (
	"context"
	"os"
	"testing"

	"bibliothek/repository"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Die Karenz-Uhr läuft ab dem letzten abgeschlossenen Vorgang, nicht nur ab dem Abgang.
//
// Die Bedingung schließt Schüler mit offener Ausleihe aus. Liefe abgaenger_seit weiter, während
// das Buch draußen ist, kippte der Schutz mit der Rückgabe: Wer am Tag 10 zurückgab, hätte 80
// Tage Fenster für eine Reparatur, wer am Tag 120 zurückgab, würde in der folgenden Nacht
// anonymisiert. Die Karenz ist der Raum, in dem eine falsche Zuordnung an der Theke auffällt,
// und die Rückgabe ist der Kontakt an der Theke.
//
// Die Fälle sind so gebaut, dass eine Uhr nur aus abgaenger_seit bei RUECK und SCHADEN
// anonymisiert und der Test rot wird; ALT und VOR sichern, dass die Uhr nicht „irgendeine
// Ausleihe je" ist, sondern der Zeitpunkt des Abschlusses.
func TestSchuelerAnonymisierung_KarenzUhrLaeuftAbLetztemVorgang(t *testing.T) {
	adminDSN := os.Getenv(datenbankVariable)
	if adminDSN == "" {
		t.Skipf("%s nicht gesetzt — Test übersprungen", datenbankVariable)
	}
	_, dsn := legeProbeDatenbankAn(t, adminDSN, "karenzrueckgabe")
	befuelleQuelle(t, dsn)
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatalf("Pool: %v", err)
	}
	t.Cleanup(pool.Close)

	if _, err := pool.Exec(ctx, `INSERT INTO system_einstellungen (schluessel, wert) VALUES ($1, '90')
		ON CONFLICT (schluessel) DO UPDATE SET wert = EXCLUDED.wert`, repository.AbgaengerKarenzSchluessel); err != nil {
		t.Fatal(err)
	}

	// Jeder Abgänger ist seit 100 Tagen Abgänger und seit 400 Tagen unberührt; ohne Vorgang fällt
	// er bei Karenz 90 sicher. Was ihn hält, ist allein der Vorgang.
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
	// rueckgabeVor "" = noch offen.
	legeAusleihe := func(schuelerID, exemplarID, ausgeliehenVor, rueckgabeVor string) {
		t.Helper()
		if _, err := pool.Exec(ctx, `
			INSERT INTO ausleihen (exemplar_id, schueler_id, ausgeliehen_am, rueckgabe_frist, rueckgabe_am)
			VALUES ($1, $2, NOW() - $3::interval, NOW() - $3::interval + interval '21 days',
			        NOW() - NULLIF($4, '')::interval)`,
			exemplarID, schuelerID, ausgeliehenVor, rueckgabeVor); err != nil {
			t.Fatal(err)
		}
	}
	legeSchaden := func(schuelerID, exemplarID string, bezahlt bool, aktualisiertVor string) {
		t.Helper()
		if _, err := pool.Exec(ctx, `
			INSERT INTO schadensfaelle (exemplar_id, schueler_id, beschreibung, betrag, ist_bezahlt, aktualisiert_am)
			VALUES ($1, $2, 'Karenz-Probe', 5.00, $3, NOW() - $4::interval)`,
			exemplarID, schuelerID, bezahlt, aktualisiertVor); err != nil {
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

	// RUECK: Buch vor 5 Tagen zurück; die Uhr startet neu, 85 Tage Fenster bleiben.
	rueck := legeAbgaenger("KR-RUECK")
	legeAusleihe(rueck, legeExemplar("KR-RUECK-EX"), "150 days", "5 days")
	// ALT: Buch vor 95 Tagen zurück; auch die neue Uhr ist abgelaufen.
	alt := legeAbgaenger("KR-ALT")
	legeAusleihe(alt, legeExemplar("KR-ALT-EX"), "200 days", "95 days")
	// VOR: Rückgabe vor dem Abgang (120 Tage); der Abgang ist die spätere Uhr, die Zeile fällt.
	vor := legeAbgaenger("KR-VOR")
	legeAusleihe(vor, legeExemplar("KR-VOR-EX"), "150 days", "120 days")
	// SCHADEN: Schadensfall vor 3 Tagen bezahlt; der Abschluss hält die Zeile.
	schaden := legeAbgaenger("KR-SCHADEN")
	legeSchaden(schaden, legeExemplar("KR-SCHADEN-EX"), true, "3 days")
	// OFFEN: Buch noch draußen; die Zeile bleibt stehen.
	offen := legeAbgaenger("KR-OFFEN")
	legeAusleihe(offen, legeExemplar("KR-OFFEN-EX"), "150 days", "")

	NewZeitplan(pool, repository.NewProtokollAbfragen(pool)).FuehreSchuelerAnonymisierungAus()

	if anonymisiert(rueck) {
		t.Error("RUECK: Rückgabe vor 5 Tagen muss die Karenz-Uhr neu starten — anonymisiert trotzdem (Uhr = nur abgaenger_seit?)")
	}
	if !anonymisiert(alt) {
		t.Error("ALT: Rückgabe vor 95 Tagen liegt außerhalb der Karenz 90 — muss anonymisiert sein")
	}
	if !anonymisiert(vor) {
		t.Error("VOR: Rückgabe vor dem Abgang darf die Uhr nicht verlängern — muss anonymisiert sein")
	}
	if anonymisiert(schaden) {
		t.Error("SCHADEN: Bezahlung vor 3 Tagen muss die Karenz-Uhr neu starten — anonymisiert trotzdem")
	}
	if anonymisiert(offen) {
		t.Error("OFFEN: Schüler mit offener Ausleihe darf nie anonymisiert werden (Regression)")
	}
}
