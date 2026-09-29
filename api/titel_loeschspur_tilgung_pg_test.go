package api

import (
	"context"
	"testing"

	"bibliothek/db"
	"bibliothek/inventur"
	"bibliothek/repository"
)

// Name und Freitext der Titel-Löschspur fallen mit der Anonymisierung (29.09.2026).
//
// Wer einen Titel löscht, an dem eine offene Forderung oder eine Vormerkung hängt, hinterlässt
// in der Datensatz-Historie (audit_log) eine Spur mit dem Namen des Lesers (schuldner,
// betrifft) und dem Freitext der Forderung (beschreibung). Die Tilgung kannte in audit_log nur
// die Tabellen schueler und ausleihen; Name und Freitext standen nach der Anonymisierung bis
// zur Audit-Aufbewahrung (24 Monate) neben der Kennung. Geprüft über beide Löschwege — den
// einzelnen Titel und die Sammellöschung — und die Anonymisierung des LUSD-Abgangs, die
// dieselbe Liste fährt wie das endgültige Löschen und der Nachtlauf.
//
// Rot gesehen am Rückbau: die Anweisung „audit_log (Personenbezug neben der Leserkennung)"
// aus repository.spurTilgungen entfernt — alle vier Einträge behalten Name und Freitext.
func TestAnonymisierung_TilgtNameUndFreitextDerTitelLoeschspur(t *testing.T) {
	pool := pgTestPool(t)
	resetBestandsdaten(t, pool)
	ctx := context.Background()
	const name, freitext = "Spurprobe Klarname", "FREITEXT-PROBE Buch im Bus liegen gelassen"

	var schuelerID string
	if err := pool.QueryRow(ctx,
		`INSERT INTO schueler (barcode_id, vorname, nachname, klasse, abgaenger_jahr)
		 VALUES ('TLS-1', 'Spurprobe', 'Klarname', '10R', 2026) RETURNING id`).Scan(&schuelerID); err != nil {
		t.Fatalf("Leser anlegen: %v", err)
	}

	// Je Löschweg ein Titel mit einer offenen Forderung und einer Vormerkung desselben Lesers.
	titelMitSpur := func(titel, barcode string) string {
		t.Helper()
		var titelID string
		if err := pool.QueryRow(ctx,
			`INSERT INTO buecher_titel (titel, autor) VALUES ($1, 'Test') RETURNING id`, titel).Scan(&titelID); err != nil {
			t.Fatalf("Titel anlegen: %v", err)
		}
		exemplarID := exemplar(t, pool, titelID, barcode, true, "")
		if _, err := pool.Exec(ctx,
			`INSERT INTO schadensfaelle (exemplar_id, schueler_id, beschreibung, betrag) VALUES ($1, $2, $3, 12.50)`,
			exemplarID, schuelerID, freitext); err != nil {
			t.Fatalf("Forderung anlegen: %v", err)
		}
		if _, err := pool.Exec(ctx,
			`INSERT INTO vormerkungen (titel_id, schueler_id, status) VALUES ($1, $2, 'wartend')`,
			titelID, schuelerID); err != nil {
			t.Fatalf("Vormerkung anlegen: %v", err)
		}
		return titelID
	}
	einzeln := titelMitSpur("Löschspur einzeln", "TLS-EX-1")
	gesammelt := titelMitSpur("Löschspur gesammelt", "TLS-EX-2")

	if err := repository.NewAuditRepository(pool).DeleteTitle(ctx, einzeln, adminFuerAudit(t, pool)); err != nil {
		t.Fatalf("einzelnen Titel löschen: %v", err)
	}
	if err := inventur.NewBookRepository(pool).DeleteBooks(ctx, []string{gesammelt}); err != nil {
		t.Fatalf("Sammellöschung: %v", err)
	}

	mitPersonenbezug := func() int {
		t.Helper()
		var n int
		if err := pool.QueryRow(ctx,
			`SELECT count(*) FROM audit_log
			 WHERE details::text LIKE '%' || $1 || '%' OR details::text LIKE '%' || $2 || '%'`,
			name, freitext).Scan(&n); err != nil {
			t.Fatal(err)
		}
		return n
	}
	// Positivkontrolle: je Weg eine Forderung (schuldner, beschreibung) und eine Vormerkung
	// (betrifft). Ohne diese vier Einträge wäre die Prüfung unten grün, ohne etwas zu prüfen.
	if n := mitPersonenbezug(); n != 4 {
		t.Fatalf("vor der Anonymisierung %d Einträge mit Name oder Freitext, erwartet 4", n)
	}

	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer db.SafeRollback(ctx, tx)
	if err := anonymisiereAbgaenger(ctx, tx, schuelerID); err != nil {
		t.Fatalf("anonymisieren: %v", err)
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatal(err)
	}

	if n := mitPersonenbezug(); n != 0 {
		t.Errorf("Name oder Freitext überleben die Anonymisierung in %d Einträgen der Datensatz-Historie", n)
	}
	// Die Einträge selbst bleiben als Beleg; die Kennung ist nach der Anonymisierung ein Pseudonym.
	var bleiben int
	if err := pool.QueryRow(ctx,
		`SELECT count(*) FROM audit_log WHERE details->>'schueler_id' = $1
		   AND tabelle IN ('schadensfaelle', 'vormerkungen')`, schuelerID).Scan(&bleiben); err != nil {
		t.Fatal(err)
	}
	if bleiben != 4 {
		t.Errorf("%d Einträge mit der Kennung des Lesers nach der Anonymisierung, erwartet 4", bleiben)
	}
}
