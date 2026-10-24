package repository_test

import (
	"context"
	"testing"

	"bibliothek/db"
	"bibliothek/internal/pgtest"
	"bibliothek/internal/testdaten"
	"bibliothek/internal/testweg"
	"bibliothek/repository"
)

// Name und Freitext der Titel-Löschspur fallen mit der Anonymisierung.
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
	pool := pgtest.Pool(t)
	testdaten.LeereBestandsdaten(t, pool)
	ctx := context.Background()
	const name, freitext = "Spurprobe Klarname", "FREITEXT-PROBE Buch im Bus liegen gelassen"

	schuelerID := testdaten.LegeSchuelerMitNamenAn(t, pool, "TLS-1", "Spurprobe", "Klarname", "10R", 2026)

	// Je Löschweg ein Titel mit einer offenen Forderung und einer Vormerkung desselben Lesers.
	titelMitSpur := func(titel, barcode string) string {
		t.Helper()
		var titelID string
		if err := pool.QueryRow(ctx,
			`INSERT INTO buecher_titel (titel, autor) VALUES ($1, 'Test') RETURNING id`, titel).Scan(&titelID); err != nil {
			t.Fatalf("Titel anlegen: %v", err)
		}
		exemplarID := testdaten.LegeExemplarAn(t, pool, titelID, barcode, true, "")
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

	if err := repository.NewKatalogAbfragen(pool).LoescheTitel(ctx, einzeln, testdaten.LegeBearbeiterAn(t, pool)); err != nil {
		t.Fatalf("einzelnen Titel löschen: %v", err)
	}
	if err := repository.NewTitelAbfragen(pool).LoescheTitelSamtAusleihen(ctx, []string{gesammelt}); err != nil {
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
	defer db.Verwerfe(ctx, tx)
	if err := repository.AnonymisiereAbgaenger(ctx, tx, schuelerID); err != nil {
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

// Die Spur zieht beim Zusammenführen zweier Leserzeilen mit um. Die Tilgung sucht sie über die
// Kennung in den Details; bliebe dort die Kennung der aufgelösten Zeile stehen, fände die
// Anonymisierung der verbliebenen Zeile Name und Freitext nicht mehr.
func TestAnonymisierung_TilgtDieLoeschspurAuchNachDemZusammenfuehren(t *testing.T) {
	pool := pgtest.Pool(t)
	testdaten.LeereBestandsdaten(t, pool)
	ctx := context.Background()
	const name, freitext = "Doppelt Spurprobe", "FREITEXT-PROBE Einband abgerissen"

	ziel := testweg.LegeLUSDSchuelerAn(t, pool, testweg.LUSDSchuelerNeu{Vorname: "Erst", Nachname: "Spurprobe", Klasse: "07A", Barcode: "TLZ-1", Geburtsdatum: testdaten.Datum(2012, 5, 5)})
	quelle := testweg.LegeLUSDSchuelerAn(t, pool, testweg.LUSDSchuelerNeu{Vorname: "Doppelt", Nachname: "Spurprobe", Klasse: "07A", Barcode: "TLZ-2", Geburtsdatum: testdaten.Datum(2012, 5, 5)})

	var titelID string
	if err := pool.QueryRow(ctx,
		`INSERT INTO buecher_titel (titel, autor) VALUES ('Löschspur vor dem Zusammenführen', 'Test') RETURNING id`).Scan(&titelID); err != nil {
		t.Fatalf("Titel anlegen: %v", err)
	}
	exemplarID := testdaten.LegeExemplarAn(t, pool, titelID, "TLZ-EX-1", true, "")
	if _, err := pool.Exec(ctx,
		`INSERT INTO schadensfaelle (exemplar_id, schueler_id, beschreibung, betrag) VALUES ($1, $2, $3, 12.50)`,
		exemplarID, quelle, freitext); err != nil {
		t.Fatalf("Forderung anlegen: %v", err)
	}
	if _, err := pool.Exec(ctx,
		`INSERT INTO vormerkungen (titel_id, schueler_id, status) VALUES ($1, $2, 'wartend')`, titelID, quelle); err != nil {
		t.Fatalf("Vormerkung anlegen: %v", err)
	}
	if err := repository.NewKatalogAbfragen(pool).LoescheTitel(ctx, titelID, testdaten.LegeBearbeiterAn(t, pool)); err != nil {
		t.Fatalf("Titel löschen: %v", err)
	}
	if _, err := repository.FuehreLeserZusammen(ctx, pool, repository.ZusammenfuehrenAuftrag{ZielID: ziel, QuelleID: quelle}); err != nil {
		t.Fatalf("Zusammenführen: %v", err)
	}

	mitPersonenbezug := func() int {
		t.Helper()
		return pgtest.ZaehleZeilen(t, pool,
			`SELECT count(*) FROM audit_log
			 WHERE tabelle IN ('schadensfaelle', 'vormerkungen')
			   AND (details::text LIKE '%' || $1 || '%' OR details::text LIKE '%' || $2 || '%')`, name, freitext)
	}
	// Positivkontrolle: eine Forderung (schuldner, beschreibung) und eine Vormerkung (betrifft).
	if n := mitPersonenbezug(); n != 2 {
		t.Fatalf("vor der Anonymisierung %d Einträge mit Name oder Freitext, erwartet 2", n)
	}

	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Verwerfe(ctx, tx)
	if err := repository.AnonymisiereAbgaenger(ctx, tx, ziel); err != nil {
		t.Fatalf("anonymisieren: %v", err)
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatal(err)
	}

	if n := mitPersonenbezug(); n != 0 {
		t.Errorf("Name oder Freitext der aufgelösten Leserzeile überleben die Anonymisierung der verbliebenen in %d Einträgen", n)
	}
}
