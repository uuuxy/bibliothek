package api

import (
	"context"
	"testing"
	"time"
)

// Zwei LUSD-Importe dürfen sich nicht ins Gehege kommen — und der Import darf der
// Handanlage keine Nummer wegnehmen.
//
// Das ist die Zusage, die bis zum 16.09.2026 NIEMAND geprüft hat, obwohl es einen Test
// gab. `generateImportBarcode` baute die Nummer aus `time.Now().Unix()%1000000` plus der
// Zeilennummer. Der Zeitteil wiederholt sich alle 11,6 Tage; zwei Läufe im richtigen
// Abstand und von ähnlicher Größe erzeugen dieselben Nummern. Der eindeutige Index
// (uniq_schueler_barcode_active) hätte das quittiert — mit dem Abbruch des GESAMTEN
// Imports, zu einem Zeitpunkt, den niemand vorhersagen kann.
//
// Der alte Test (TestGenerateImportBarcode_UniqueWithinImport) war grün und konnte es
// nicht sehen: Er prüfte die Eindeutigkeit INNERHALB eines Laufs, und die war nie das
// Problem. Deshalb steht diese Zusage jetzt an der echten Datenbank, über zwei Läufe und
// gegen die zweite Vergabestelle.
func TestLusdAusweisnummern_ZweiLaeufeUndDieHandanlage(t *testing.T) {
	pool := pgTestPool(t)
	resetBestandsdaten(t, pool)
	ctx := context.Background()

	lauf := func(t *testing.T, namen ...string) {
		t.Helper()
		datei := lusdDatei{Modus: lusdModusID}
		z := lusdZuordnung{
			zielID:             map[int]string{},
			ueberspringen:      map[int]bool{},
			geburtsdatumSetzen: map[int]bool{},
		}
		geb := time.Date(2012, 5, 4, 0, 0, 0, 0, time.UTC)
		for i, n := range namen {
			datei.Zeilen = append(datei.Zeilen, parsedStudentRow{
				LusdID: "LUSD-" + n, Vorname: "Vor", Nachname: n, Klasse: "07A",
				GebDatum: &geb, LineNum: i + 2,
			})
		}
		tx, err := pool.Begin(ctx)
		if err != nil {
			t.Fatal(err)
		}
		defer tx.Rollback(ctx) //nolint:errcheck
		if err := wendeLusdAenderungenAn(ctx, tx, datei, z); err != nil {
			t.Fatalf("Import: %v", err)
		}
		if err := tx.Commit(ctx); err != nil {
			t.Fatal(err)
		}
	}

	nummernVon := func(t *testing.T, namen ...string) []string {
		t.Helper()
		out := []string{}
		for _, n := range namen {
			var bc string
			if err := pool.QueryRow(ctx,
				`SELECT barcode_id FROM leser WHERE nachname = $1`, n).Scan(&bc); err != nil {
				t.Fatalf("Ausweis von %s: %v", n, err)
			}
			out = append(out, bc)
		}
		return out
	}

	// Erster Lauf: drei Neuzugänge.
	lauf(t, "Ampel", "Birne", "Coda")
	ersteNummern := nummernVon(t, "Ampel", "Birne", "Coda")

	// Jede Nummer trägt die EINE Vorsilbe, und sie hat dieselbe Form wie die der
	// Handanlage — nicht mehr die der Uhr.
	for _, bc := range ersteNummern {
		if len(bc) < len(AusweisPraefix) || bc[:len(AusweisPraefix)] != AusweisPraefix {
			t.Fatalf("Ausweis %q trägt nicht die Vorsilbe %q", bc, AusweisPraefix)
		}
	}

	// Zweiter Lauf, unmittelbar danach: KEINE Nummer darf sich wiederholen. Genau hier
	// wäre der alte Generator gescheitert, sobald der Zeitteil sich wiederholt.
	lauf(t, "Dattel", "Endivie", "Fenchel")
	zweiteNummern := nummernVon(t, "Dattel", "Endivie", "Fenchel")

	gesehen := map[string]string{}
	for i, bc := range append(append([]string{}, ersteNummern...), zweiteNummern...) {
		if wo, doppelt := gesehen[bc]; doppelt {
			t.Fatalf("Ausweisnummer %q zweimal vergeben (%s und Zeile %d)", bc, wo, i)
		}
		gesehen[bc] = "Zeile " + bc
	}

	// Und die Gegenprobe zur ZWEITEN Vergabestelle: Die Handanlage zieht aus derselben
	// Sequenz und muss über allem liegen, was die Importe vergeben haben. Ein eigener
	// Zähler gäbe hier eine Nummer aus, die schon hängt — der Fehler aus Migration 068,
	// eine Tabelle weiter.
	var hoechste int
	if err := pool.QueryRow(ctx, `
		SELECT COALESCE(MAX((substring(barcode_id from '([0-9]{1,15})$'))::bigint), 0)
		  FROM leser WHERE barcode_id LIKE $1`, AusweisPraefix+"%").Scan(&hoechste); err != nil {
		t.Fatalf("höchste Nummer lesen: %v", err)
	}
	if hoechste == 0 {
		t.Fatal("keine Ausweisnummern gefunden — der Test misst nichts")
	}

	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck
	naechste, ok := resolveNeueBarcodeID(ctx, tx, nil, "")
	if !ok {
		t.Fatal("die Handanlage konnte keine Nummer ziehen")
	}
	if erwartet := AusweisNummer(hoechste + 1); naechste != erwartet {
		t.Fatalf("die Handanlage vergibt %q, erwartet %q — zwei Zähler in einem Nummernkreis",
			naechste, erwartet)
	}
}
