package api

import (
	"context"
	"testing"

	"bibliothek/repository"
)

// Die Löschung muss tilgen, was die Auskunft zurechnet.
//
// Fund des Komplett-Durchgangs 31.08.2026: Zwei Prädikate beantworten dieselbe Frage
// „welche audit_log-Zeilen gehören diesem Schüler?" und waren sich nicht einig. Die
// Art.-15-Auskunft (api/dsgvo_auskunft.go) zählt ausdrücklich auch die Ausleih-Zeilen
// dazu („Ohne diesen Zweig fehlte die Lesehistorie"): tabelle='ausleihen' AND
// details->>'schueler_id' = id. TilgeSchuelerSpuren tilgte nur tabelle='schueler'.
//
// Folge: Nach Purge oder LUSD-Abgang blieb die vollständige Lesehistorie
// (CHECKOUT/RETURN mit schueler_id im details-Blob) stehen — auffindbar mit exakt der
// Abfrage, die der eigene Auskunftshandler benutzt. Sie fiel erst, wenn der
// Lesehistorie-Cron sie Monate später zufällig noch einholte; nach dem Löschen der
// Schülerzeile gab es dafür keine Garantie mehr.
func TestTilgeSchuelerSpuren_TilgtAuchDieLesehistorie(t *testing.T) {
	pool := pgTestPool(t)
	resetBestandsdaten(t, pool)
	ctx := context.Background()

	sid := seedSchueler(t, pool, "S-TILG-1", "Mia", "5a")
	fremd := seedSchueler(t, pool, "S-TILG-2", "Ben", "5a")
	titelID := seedMonitorTitel(t, pool, "Tilgungs-Titel", "Jug Til", false, 0)
	exID := exemplar(t, pool, titelID, "TILG-1", true, "")

	// Zwei Protokollzeilen wie sie die Ausleihe/Rückgabe schreibt (audit_books.go):
	// die des Schülers und die eines anderen — die fremde muss stehen bleiben.
	for _, s := range []string{sid, fremd} {
		if _, err := pool.Exec(ctx, `
			INSERT INTO audit_log (tabelle, aktion, datensatz_id, akteur, kontext, details, timestamp)
			VALUES ('ausleihen', 'CHECKOUT', $1::uuid, 'USER', 'Ausleihe',
			        jsonb_build_object('exemplar_id', $2::text, 'schueler_id', $3::text, 'entleiher', 'Mia Test'),
			        NOW() - interval '3 days')`, exID, exID, s); err != nil {
			t.Fatalf("Protokollzeile anlegen: %v", err)
		}
	}

	if err := repository.TilgeSchuelerSpuren(ctx, pool, sid, "Test-Löschung"); err != nil {
		t.Fatalf("TilgeSchuelerSpuren: %v", err)
	}

	// Genau die Abfrage, mit der die Art.-15-Auskunft die Lesehistorie einsammelt.
	var verbleibend int
	if err := pool.QueryRow(ctx, `
		SELECT count(*) FROM audit_log
		WHERE tabelle = 'ausleihen' AND details->>'schueler_id' = $1`, sid).Scan(&verbleibend); err != nil {
		t.Fatalf("Lesehistorie zählen: %v", err)
	}
	if verbleibend != 0 {
		t.Errorf("%d Protokollzeilen mit schueler_id überleben die Löschung — die Lesehistorie ist über die Abfrage der eigenen Auskunft weiter auffindbar", verbleibend)
	}

	// Die Zeile selbst bleibt (Nachweis, dass ausgeliehen wurde), nur ohne Personenbezug.
	var zeilen, mitEntleiher int
	if err := pool.QueryRow(ctx, `
		SELECT count(*), count(*) FILTER (WHERE details ? 'entleiher')
		FROM audit_log WHERE tabelle = 'ausleihen'`).Scan(&zeilen, &mitEntleiher); err != nil {
		t.Fatalf("Protokollzeilen zählen: %v", err)
	}
	if zeilen != 2 {
		t.Errorf("%d Protokollzeilen — die Buchungshistorie darf nicht verschwinden, nur ihr Personenbezug", zeilen)
	}
	if mitEntleiher != 1 {
		t.Errorf("%d Zeilen mit Entleiher-Namen, erwartet 1 (die des ANDEREN Schülers)", mitEntleiher)
	}

	// Die fremde Zeile ist unangetastet.
	var fremdeDa int
	if err := pool.QueryRow(ctx, `
		SELECT count(*) FROM audit_log
		WHERE tabelle = 'ausleihen' AND details->>'schueler_id' = $1`, fremd).Scan(&fremdeDa); err != nil {
		t.Fatalf("fremde Zeile zählen: %v", err)
	}
	if fremdeDa != 1 {
		t.Errorf("fremde Protokollzeile: %d — die Tilgung greift zu weit", fremdeDa)
	}
}
