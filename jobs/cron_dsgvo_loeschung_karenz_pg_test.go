package jobs

import (
	"context"
	"os"
	"testing"
	"time"

	"bibliothek/repository"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Die endgültige Löschung wartet die Karenz ab: Gelöscht wird nur, was schon anonymisiert
// ist (Befund-Register, Entscheidung vom 05.09.2026). Bis dahin löschte
// RunGDPRDeleteAbgaenger ab dem 30. Januar des Folgejahres jeden Abgänger ohne offene
// Vorgänge in der nächsten Nacht — und lief im Cron VOR der Anonymisierung. Die Karenz
// (ab dem letzten Vorgang) galt nur für die Anonymisierung; wer nach dem Stichtag
// zurückgab, hatte kein Reparaturfenster, wer im November zurückgab, ein verkürztes.
//
// Gefahren wird die ganze nächtliche Folge (RunNaechtlicheDSGVO), nicht ein einzelner
// Job: Die Regel „nach der Karenz anonymisiert UND in derselben Nacht gelöscht" hängt an
// der Reihenfolge. Die Fälle sind so gebaut, dass das alte Prädikat bei SPAET rot wird
// (gelöscht trotz Rückgabe vor fünf Tagen) und die alte Reihenfolge bei REIF
// (anonymisiert, aber erst in der nächsten Nacht gelöscht). ANON, OFFEN und SCHADEN
// sichern, was unverändert bleibt.
func TestNaechtlicheDSGVO_LoeschungWartetDieKarenzAb(t *testing.T) {
	adminDSN := os.Getenv(drillEnvVar)
	if adminDSN == "" {
		t.Skipf("%s nicht gesetzt — Test übersprungen", drillEnvVar)
	}
	_, dsn := legeProbeDatenbankAn(t, adminDSN, "loeschkarenz")
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
	// Alle sind Abgänger eines Jahrgangs VOR dem Stichjahr — die Löschung ist nach dem
	// Jahr also fällig; was sie hält, ist allein die Karenz oder ein offener Vorgang.
	altesJahr := repository.AbgaengerStichjahr(time.Now()) - 1

	// anonymisiertVor "" = noch nicht anonymisiert.
	legeAbgaenger := func(barcode, anonymisiertVor string) string {
		t.Helper()
		var id string
		if err := pool.QueryRow(ctx, `
			INSERT INTO schueler (barcode_id, vorname, nachname, klasse, abgaenger_jahr, ist_abgaenger,
			                      ist_gesperrt, block_reason, abgaenger_seit, aktualisiert_am, anonymized_at)
			VALUES ($1, 'Karenz', $1, 'ABG', $2, true, true, $3,
			        NOW() - interval '300 days', NOW() - interval '400 days',
			        NOW() - NULLIF($4, '')::interval)
			RETURNING id`, barcode, altesJahr, repository.AbgaengerSperrgrundKarenz, anonymisiertVor).Scan(&id); err != nil {
			t.Fatal(err)
		}
		return id
	}
	legeExemplar := func(barcode string) string {
		t.Helper()
		var id string
		if err := pool.QueryRow(ctx, `
			WITH titel AS (INSERT INTO buecher_titel (titel) VALUES ('Löschkarenz-Titel ' || $1) RETURNING id)
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
			        NOW() - NULLIF($4, '')::interval)`,
			exemplarID, schuelerID, ausgeliehenVor, rueckgabeVor); err != nil {
			t.Fatal(err)
		}
	}
	legeSchaden := func(schuelerID, exemplarID string, bezahlt bool) {
		t.Helper()
		if _, err := pool.Exec(ctx, `
			INSERT INTO schadensfaelle (exemplar_id, schueler_id, beschreibung, betrag, ist_bezahlt, aktualisiert_am)
			VALUES ($1, $2, 'Löschkarenz-Probe', 5.00, $3, NOW() - interval '200 days')`,
			exemplarID, schuelerID, bezahlt); err != nil {
			t.Fatal(err)
		}
	}
	zustand := func(id string) (existiert, anonymisiert bool) {
		t.Helper()
		err := pool.QueryRow(ctx, `SELECT anonymized_at IS NOT NULL FROM schueler WHERE id = $1`, id).Scan(&anonymisiert)
		if err != nil {
			return false, false
		}
		return true, anonymisiert
	}

	// SPAET: Buch vor 5 Tagen zurück — die Karenz hat gerade erst begonnen.
	spaet := legeAbgaenger("LK-SPAET", "")
	legeAusleihe(spaet, legeExemplar("LK-SPAET-EX"), "200 days", "5 days")
	// REIF: Buch vor 100 Tagen zurück — Karenz abgelaufen, noch nicht anonymisiert.
	reif := legeAbgaenger("LK-REIF", "")
	legeAusleihe(reif, legeExemplar("LK-REIF-EX"), "300 days", "100 days")
	// ANON: längst anonymisiert — fällig wie bisher.
	anon := legeAbgaenger("LK-ANON", "10 days")
	// OFFEN: Buch noch draußen — bleibt mit Namen stehen (Mahnwesen).
	offen := legeAbgaenger("LK-OFFEN", "")
	legeAusleihe(offen, legeExemplar("LK-OFFEN-EX"), "200 days", "")
	// SCHADEN: Buch als verloren gebucht, Rechnung offen — bleibt mit Namen stehen.
	schaden := legeAbgaenger("LK-SCHADEN", "")
	legeSchaden(schaden, legeExemplar("LK-SCHADEN-EX"), false)

	NewScheduler(pool, repository.NewAuditRepository(pool)).RunNaechtlicheDSGVO()

	if da, anonym := zustand(spaet); !da || anonym {
		t.Errorf("SPAET: Rückgabe vor 5 Tagen muss die volle Karenz bekommen — existiert=%v, anonymisiert=%v (Löschung kennt die Karenz nicht?)", da, anonym)
	}
	if da, _ := zustand(reif); da {
		t.Error("REIF: Karenz abgelaufen — muss in DERSELBEN Nacht anonymisiert und gelöscht sein (Reihenfolge: erst anonymisieren, dann löschen)")
	}
	if da, _ := zustand(anon); da {
		t.Error("ANON: längst anonymisiert und Stichjahr vorbei — muss gelöscht sein")
	}
	if da, anonym := zustand(offen); !da || anonym {
		t.Errorf("OFFEN: offene Ausleihe schützt — existiert=%v, anonymisiert=%v", da, anonym)
	}
	if da, anonym := zustand(schaden); !da || anonym {
		t.Errorf("SCHADEN: unbezahlter Schadensfall schützt — existiert=%v, anonymisiert=%v", da, anonym)
	}
}
