package jobs

import (
	"context"
	"os"
	"testing"

	"bibliothek/repository"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Gelöschte Kollegen fallen nach 180 Tagen im Papierkorb endgültig, über denselben Weg wie
// „Endgültig löschen" von Hand (LoescheLeserEndgueltig).
//
// Erwartung nach dem Lauf:
//
//	K-ALT        Kollege, 200 Tage im Papierkorb                    → gelöscht; seine
//	                                                                   zurückgegebene Ausleihe
//	                                                                   bleibt ohne Person
//	K-JUNG       Kollege, 10 Tage im Papierkorb                     → bleibt
//	K-AUSLEIHE   Kollege, 200 Tage, offene Ausleihe                 → bleibt
//	K-FORDERUNG  Kollege, 200 Tage, unbezahlte Forderung            → bleibt
//	K-AKTIV      Kollege, nicht im Papierkorb                       → bleibt
//	S-ALT        Schüler, 200 Tage im Papierkorb                    → bleibt (ihn anonymisiert
//	                                                                   ein anderer Lauf)
func TestPapierkorbLoeschung_LoeschtNachFristUndHaeltOffeneVorgaenge(t *testing.T) {
	adminDSN := os.Getenv(datenbankVariable)
	if adminDSN == "" {
		t.Skipf("%s nicht gesetzt — Test übersprungen", datenbankVariable)
	}
	_, dsn := legeProbeDatenbankAn(t, adminDSN, "papierkorb")
	befuelleQuelle(t, dsn)

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatalf("Pool: %v", err)
	}
	t.Cleanup(pool.Close)

	eins := func(was, sql string, argumente ...any) string {
		t.Helper()
		var id string
		if err := pool.QueryRow(ctx, sql, argumente...).Scan(&id); err != nil {
			t.Fatalf("%s: %v", was, err)
		}
		return id
	}
	kollege := func(name string, tageImPapierkorb *int) string {
		t.Helper()
		return eins("Kollege "+name, `
			INSERT INTO leser (art, vorname, nachname, deleted_at)
			VALUES ('lehrkraft', 'Probe', $1,
			        CASE WHEN $2::int IS NULL THEN NULL ELSE NOW() - make_interval(days => $2::int) END)
			RETURNING id`, name, tageImPapierkorb)
	}
	tage := func(n int) *int { return &n }

	ids := map[string]string{
		"K-ALT":       kollege("K-ALT", tage(200)),
		"K-JUNG":      kollege("K-JUNG", tage(10)),
		"K-AUSLEIHE":  kollege("K-AUSLEIHE", tage(200)),
		"K-FORDERUNG": kollege("K-FORDERUNG", tage(200)),
		"K-AKTIV":     kollege("K-AKTIV", nil),
		"S-ALT": eins("Schüler im Papierkorb", `
			INSERT INTO schueler (barcode_id, vorname, nachname, klasse, abgaenger_jahr, deleted_at)
			VALUES ('S-PK-ALT', 'Probe', 'S-ALT', '9c', 2030, NOW() - interval '200 days') RETURNING id`),
	}

	titel := eins("Titel", `INSERT INTO buecher_titel (titel) VALUES ('Papierkorb-Titel') RETURNING id`)
	exemplar := func(barcode string) string {
		return eins("Exemplar "+barcode,
			`INSERT INTO buecher_exemplare (titel_id, barcode_id) VALUES ($1, $2) RETURNING id`, titel, barcode)
	}
	zurueck := exemplar("PK-ZURUECK")
	eins("zurückgegebene Ausleihe von K-ALT", `
		INSERT INTO ausleihen (exemplar_id, schueler_id, ausgeliehen_am, rueckgabe_frist, rueckgabe_am)
		VALUES ($1, $2, NOW() - interval '300 days', NOW() - interval '270 days', NOW() - interval '260 days')
		RETURNING id`, zurueck, ids["K-ALT"])
	eins("offene Ausleihe von K-AUSLEIHE", `
		INSERT INTO ausleihen (exemplar_id, schueler_id, ausgeliehen_am, rueckgabe_frist)
		VALUES ($1, $2, NOW() - interval '210 days', NOW() + interval '30 days') RETURNING id`,
		exemplar("PK-OFFEN"), ids["K-AUSLEIHE"])
	eins("unbezahlte Forderung von K-FORDERUNG", `
		INSERT INTO schadensfaelle (exemplar_id, schueler_id, beschreibung, betrag)
		VALUES ($1, $2, 'Wasserschaden', 12.50) RETURNING id`, exemplar("PK-SCHADEN"), ids["K-FORDERUNG"])

	NewZeitplan(pool, repository.NewProtokollAbfragen(pool)).FuehrePapierkorbLoeschungAus()

	for name, bleibt := range map[string]bool{
		"K-ALT": false, "K-JUNG": true, "K-AUSLEIHE": true, "K-FORDERUNG": true, "K-AKTIV": true, "S-ALT": true,
	} {
		var n int
		if err := pool.QueryRow(ctx, `SELECT count(*) FROM leser WHERE id = $1`, ids[name]).Scan(&n); err != nil {
			t.Fatalf("lesen: %v", err)
		}
		if got := n > 0; got != bleibt {
			t.Errorf("%s: vorhanden = %v, erwartet %v", name, got, bleibt)
		}
	}

	// Der Vorgang bleibt für Statistik und Bestandskartei, nur ohne Person, wie beim endgültigen
	// Löschen von Hand.
	var ohnePerson bool
	if err := pool.QueryRow(ctx,
		`SELECT schueler_id IS NULL FROM ausleihen WHERE exemplar_id = $1`, zurueck).Scan(&ohnePerson); err != nil {
		t.Fatalf("Ausleihe von K-ALT: %v", err)
	}
	if !ohnePerson {
		t.Error("die zurückgegebene Ausleihe von K-ALT hängt nach dem Löschen noch an einer Person")
	}

	// Im Protokoll: die Löschung selbst (Akteur SYSTEM, wie beim endgültigen Löschen ohne
	// Bearbeiter) und die Zeile des Laufs mit der Zahl.
	var loescheEndgueltig, lauf string
	if err := pool.QueryRow(ctx, `
		SELECT coalesce(akteur, '') FROM audit_log
		WHERE tabelle = 'schueler' AND aktion = 'DELETE' AND datensatz_id = $1`, ids["K-ALT"]).Scan(&loescheEndgueltig); err != nil {
		t.Fatalf("Protokoll der Löschung: %v", err)
	}
	if loescheEndgueltig != "SYSTEM" {
		t.Errorf("Akteur der Löschung: %q, erwartet SYSTEM", loescheEndgueltig)
	}
	if err := pool.QueryRow(ctx, `
		SELECT coalesce(details->>'geloescht', '') FROM audit_log
		WHERE tabelle = 'schueler' AND aktion = 'BATCH_DELETE'
		  AND kontext LIKE '%Kollegen nach 180 Tagen im Papierkorb%'`).Scan(&lauf); err != nil {
		t.Fatalf("Protokollzeile des Laufs: %v", err)
	}
	if lauf != "1" {
		t.Errorf("Protokoll nennt %q gelöschte Kollegen, erwartet 1", lauf)
	}
}
