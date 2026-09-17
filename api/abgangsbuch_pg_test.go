package api

import (
	"context"
	"strings"
	"testing"
	"time"

	"bibliothek/internal/pdftest"
	"bibliothek/pdf"
	"bibliothek/pkg/schulzeit"
	"bibliothek/repository"
)

// Das Abgangsbuch ist ein Nachweis: Es wird ausgedruckt, unterschrieben und abgeheftet.
// Gemessen wird deshalb an den Rändern des Zeitraums und am fertigen Blatt — eine Zeile zu
// viel oder zu wenig fällt niemandem auf, der die Liste nicht selbst nachzählt.
func TestAbgangsbuch_ZeitraumUndToepfe(t *testing.T) {
	pool := pgTestPool(t)
	resetBestandsdaten(t, pool)
	ctx := context.Background()
	loc := schulzeit.Zone()

	lmf := titelMitSignatur(t, pool, "Mathebuch 7", "Mat 7", 0)
	if _, err := pool.Exec(ctx, `UPDATE buecher_titel SET ist_lernmittel = true WHERE id = $1`, lmf); err != nil {
		t.Fatalf("Lernmittel setzen: %v", err)
	}
	buecherei := titelMitSignatur(t, pool, "Gregs Tagebuch", "Jug Gre", 0)

	// Vier Abgänge: zwei im Halbjahr (je Topf einer), einer am Tag davor, einer am Tag
	// danach. Die beiden Ränder sind der Punkt — der 15.9. gehört noch zum Halbjahr, das
	// an ihm endet, der 16.9. schon zum nächsten.
	stempel := func(titelID, barcode, grund string, wann time.Time) {
		t.Helper()
		id := exemplar(t, pool, titelID, barcode, true, "")
		if _, err := pool.Exec(ctx, `
			UPDATE buecher_exemplare
			SET ist_ausgesondert = true, ist_ausleihbar = false, aussonderung_grund = $2
			WHERE id = $1`, id, grund); err != nil {
			t.Fatalf("aussondern %s: %v", barcode, err)
		}
		// Der Trigger setzt „jetzt"; für den Test muss der Abgang datiert werden.
		if _, err := pool.Exec(ctx,
			`UPDATE buecher_exemplare SET ausgesondert_am = $2 WHERE id = $1`, id, wann); err != nil {
			t.Fatalf("datieren %s: %v", barcode, err)
		}
	}
	tag := func(j int, m time.Month, d, stunde int) time.Time {
		return time.Date(j, m, d, stunde, 0, 0, 0, loc)
	}

	stempel(lmf, "AB-LMF", "VERLUST", tag(2026, time.April, 12, 10))
	stempel(buecherei, "AB-BUE", "AUSSORTIERT", tag(2026, time.September, 15, 23)) // letzter Tag
	stempel(lmf, "AB-FRUEH", "AUSSORTIERT", tag(2026, time.March, 15, 12))         // Tag davor
	stempel(lmf, "AB-SPAET", "AUSSORTIERT", tag(2026, time.September, 16, 1))      // Tag danach

	// Und einer ohne Datum: der Altbestand, den es nicht zuzuordnen gibt.
	ohne := exemplar(t, pool, buecherei, "AB-OHNE", true, "")
	if _, err := pool.Exec(ctx, `
		UPDATE buecher_exemplare SET ist_ausgesondert = true, ist_ausleihbar = false,
		       aussonderung_grund = 'BESTANDSKORREKTUR' WHERE id = $1`, ohne); err != nil {
		t.Fatalf("Altbestand aussondern: %v", err)
	}
	if _, err := pool.Exec(ctx, `UPDATE buecher_exemplare SET ausgesondert_am = NULL WHERE id = $1`, ohne); err != nil {
		t.Fatalf("Datum leeren: %v", err)
	}

	von, bis := tag(2026, time.March, 16, 0), tag(2026, time.September, 15, 0)
	buch, err := repository.LadeAbgangsbuch(ctx, pool, von, bis)
	if err != nil {
		t.Fatalf("Abgangsbuch laden: %v", err)
	}

	var barcodes []string
	for _, z := range buch.Zeilen {
		barcodes = append(barcodes, z.Barcode)
	}
	if len(buch.Zeilen) != 2 {
		t.Fatalf("Zeilen im Halbjahr: %v — erwartet AB-LMF und AB-BUE", barcodes)
	}
	// Lernmittel zuerst: Der Ausdruck trägt sie als ersten Abschnitt.
	if buch.Zeilen[0].Barcode != "AB-LMF" || buch.Zeilen[0].Topf != repository.MittelLand {
		t.Errorf("erste Zeile: %+v — erwartet das Lernmittel", buch.Zeilen[0])
	}
	if buch.Zeilen[1].Barcode != "AB-BUE" || buch.Zeilen[1].Topf != repository.MittelSchultraeger {
		t.Errorf("zweite Zeile: %+v — erwartet das Büchereibuch", buch.Zeilen[1])
	}
	if buch.Zeilen[0].GrundText != "Verlust" {
		t.Errorf("Grund im Klartext: %q", buch.Zeilen[0].GrundText)
	}
	if buch.OhneZeitpunkt != 1 {
		t.Errorf("Abgänge ohne Zeitpunkt: %d, erwartet 1 — die Zahl gehört auf das Blatt", buch.OhneZeitpunkt)
	}

	// Gegenprobe: Das nächste Halbjahr beginnt am 16.9. und trägt genau den einen Abgang.
	naechstes, err := repository.LadeAbgangsbuch(ctx, pool, tag(2026, time.September, 16, 0), tag(2027, time.March, 15, 0))
	if err != nil {
		t.Fatalf("zweites Halbjahr: %v", err)
	}
	if len(naechstes.Zeilen) != 1 || naechstes.Zeilen[0].Barcode != "AB-SPAET" {
		t.Errorf("zweites Halbjahr: %+v — erwartet allein AB-SPAET", naechstes.Zeilen)
	}
}

// Zurückgeholt heißt: Der Abgang war ein Irrtum. Das Buch steht wieder im Regal und darf
// in keinem Nachweis mehr als Abgang stehen.
func TestAbgangsbuch_ZurueckgeholtesStehtNichtDrin(t *testing.T) {
	pool := pgTestPool(t)
	resetBestandsdaten(t, pool)
	ctx := context.Background()
	bookRepo := repository.NewBookRepository(pool)

	titelID := titelMitSignatur(t, pool, "Irrtum", "Irr 1", 0)
	id := exemplar(t, pool, titelID, "AB-IRRTUM", true, "")
	if err := bookRepo.DecommissionCopy(ctx, id); err != nil {
		t.Fatalf("aussondern: %v", err)
	}
	von, bis := schulzeit.Halbjahr(schulzeit.Jetzt())
	if buch, err := repository.LadeAbgangsbuch(ctx, pool, von, bis); err != nil || len(buch.Zeilen) != 1 {
		t.Fatalf("vor dem Zurückholen: %v (%v) — der Test misst sonst nichts", buch.Zeilen, err)
	}

	if _, err := repository.HoleExemplarZurueck(ctx, pool, id, "", nil); err != nil {
		t.Fatalf("zurückholen: %v", err)
	}
	buch, err := repository.LadeAbgangsbuch(ctx, pool, von, bis)
	if err != nil {
		t.Fatalf("Abgangsbuch laden: %v", err)
	}
	if len(buch.Zeilen) != 0 {
		t.Errorf("zurückgeholtes Exemplar steht weiter im Abgangsbuch: %+v", buch.Zeilen)
	}
}

// Das Blatt selbst: zwei Abschnitte mit eigener Stückzahl, und der Hinweis auf die
// Abgänge ohne Zeitpunkt. Gelesen wird der Inhaltsstrom des fertigen PDFs — im Struct
// stand schon manches, was nie gedruckt wurde.
func TestAbgangsbuchPDF_ZweiAbschnitteUndHinweis(t *testing.T) {
	buch := repository.Abgangsbuch{
		Von: time.Date(2026, time.March, 16, 0, 0, 0, 0, schulzeit.Zone()),
		Bis: time.Date(2026, time.September, 15, 0, 0, 0, 0, schulzeit.Zone()),
		Zeilen: []repository.AbgangsZeile{
			{Datum: time.Date(2026, time.April, 12, 10, 0, 0, 0, schulzeit.Zone()),
				Barcode: "B-00042", Titel: "Mathebuch 7", Signatur: "Mat 7",
				Grund: "VERLUST", GrundText: "Verlust", Topf: repository.MittelLand},
			{Datum: time.Date(2026, time.May, 3, 10, 0, 0, 0, schulzeit.Zone()),
				Barcode: "B-00815", Titel: "Gregs Tagebuch", Signatur: "Jug Gre",
				Grund: "AUSSORTIERT", GrundText: "Aussortiert", Topf: repository.MittelSchultraeger},
		},
		OhneZeitpunkt: 7,
	}

	roh, err := generateAbgangsbuchPDF(buch, pdf.SchuleInfo{Name: "Philipp-Reis-Schule", Strasse: "Schulstr. 1", PLZ: "61440", Ort: "Oberursel"})
	if err != nil {
		t.Fatalf("Abgangsbuch drucken: %v", err)
	}
	blatt := strings.Join(pdftest.Texte(t, roh), " ")

	for _, muss := range []string{
		"Abgangsbuch",
		"16.03.2026", "15.09.2026", // der Zeitraum steht auf dem Blatt
		"B-00042", "B-00815",
		"Verlust", "Aussortiert",
		"Summe " + mittelBeschriftung(repository.MittelLand) + ": 1 Exemplare",
		"Summe " + mittelBeschriftung(repository.MittelSchultraeger) + ": 1 Exemplare",
		"7 weitere Exemplare", // der Hinweis auf die Abgänge ohne Zeitpunkt
	} {
		if !strings.Contains(blatt, muss) {
			t.Errorf("auf dem Blatt fehlt %q:\n%s", muss, blatt)
		}
	}
}

// Ohne Abgänge darf kein leeres Gerüst herauskommen, das wie ein Fehler aussieht — und
// der Hinweis auf die undatierten Altabgänge fehlt dann auch nicht.
func TestAbgangsbuchPDF_LeererZeitraumSagtEs(t *testing.T) {
	buch := repository.Abgangsbuch{
		Von:    time.Date(2026, time.March, 16, 0, 0, 0, 0, schulzeit.Zone()),
		Bis:    time.Date(2026, time.September, 15, 0, 0, 0, 0, schulzeit.Zone()),
		Zeilen: []repository.AbgangsZeile{},
	}
	roh, err := generateAbgangsbuchPDF(buch, pdf.SchuleInfo{Name: "Philipp-Reis-Schule"})
	if err != nil {
		t.Fatalf("Abgangsbuch drucken: %v", err)
	}
	blatt := strings.Join(pdftest.Texte(t, roh), " ")
	if !strings.Contains(blatt, "kein Exemplar aus dem Bestand gegangen") {
		t.Errorf("leerer Zeitraum sagt es nicht:\n%s", blatt)
	}
}
