package katalog

import (
	"context"
	"strings"
	"testing"
	"time"

	"bibliothek/internal/pdftest"
	"bibliothek/internal/pgtest"
	"bibliothek/internal/testdaten"
	"bibliothek/pdf"
	"bibliothek/pkg/mitteltopf"
	"bibliothek/pkg/schulzeit"
	"bibliothek/repository"
)

// Das Abgangsbuch ist ein Nachweis: Es wird ausgedruckt, unterschrieben und abgeheftet.
// Gemessen wird deshalb an den Rändern des Zeitraums und am fertigen Blatt — eine Zeile zu
// viel oder zu wenig fällt niemandem auf, der die Liste nicht selbst nachzählt.
func TestAbgangsbuch_ZeitraumUndToepfe(t *testing.T) {
	pool := pgtest.Pool(t)
	testdaten.LeereBestandsdaten(t, pool)
	ctx := context.Background()
	loc := schulzeit.Zone()

	lmf := testdaten.LegeTitelMitSignaturAn(t, pool, "Mathebuch 7", "Mat 7", 0)
	if _, err := pool.Exec(ctx, `UPDATE buecher_titel SET ist_lernmittel = true WHERE id = $1`, lmf); err != nil {
		t.Fatalf("Lernmittel setzen: %v", err)
	}
	buecherei := testdaten.LegeTitelMitSignaturAn(t, pool, "Gregs Tagebuch", "Jug Gre", 0)

	// Vier Abgänge: zwei im Halbjahr (je Topf einer), einer am Tag davor, einer am Tag
	// danach. Die beiden Ränder sind der Punkt — der 15.9. gehört noch zum Halbjahr, das
	// an ihm endet, der 16.9. schon zum nächsten.
	stempel := func(titelID, barcode, grund string, wann time.Time) {
		t.Helper()
		id := testdaten.LegeExemplarAn(t, pool, titelID, barcode, true, "")
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
	ohne := testdaten.LegeExemplarAn(t, pool, buecherei, "AB-OHNE", true, "")
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
	if buch.Zeilen[0].Barcode != "AB-LMF" || buch.Zeilen[0].Topf != mitteltopf.Land {
		t.Errorf("erste Zeile: %+v — erwartet das Lernmittel", buch.Zeilen[0])
	}
	if buch.Zeilen[1].Barcode != "AB-BUE" || buch.Zeilen[1].Topf != mitteltopf.Schultraeger {
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
	pool := pgtest.Pool(t)
	testdaten.LeereBestandsdaten(t, pool)
	ctx := context.Background()
	bookRepo := repository.NewKatalogAbfragen(pool)

	titelID := testdaten.LegeTitelMitSignaturAn(t, pool, "Irrtum", "Irr 1", 0)
	id := testdaten.LegeExemplarAn(t, pool, titelID, "AB-IRRTUM", true, "")
	if err := bookRepo.AendereExemplarStatus(ctx, id, nil, repository.ExemplarStatusStand{IstAusgesondert: true}, "", nil, testdaten.LegeBearbeiterAn(t, pool)); err != nil {
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

// Ein Abgang ist nur, was im Bestand war. Ein bestelltes Exemplar, das nie eintraf, steht
// nicht im Zugangsbuch; wird es in der Buchakte gelöscht oder im Status-Editor ausgesondert,
// gehört es auch nicht ins Abgangsbuch.
func TestAbgangsbuch_NieEingetroffenesIstKeinAbgang(t *testing.T) {
	pool := pgtest.Pool(t)
	testdaten.LeereBestandsdaten(t, pool)
	ctx := context.Background()
	bearbeiter := testdaten.LegeKollegiumskontoAn(t, pool, "abgang-zulauf@test.invalid")
	titelID := testdaten.LegeTitelMitSignaturAn(t, pool, "Nie geliefert", "Nie 1", 0)

	geloescht := testdaten.LegeExemplarImZulaufAn(t, pool, titelID, "AB-ZULAUF-LOESCHEN")
	if err := repository.NewKatalogAbfragen(pool).SondereExemplarAus(ctx, geloescht, "AUSSORTIERT", bearbeiter); err != nil {
		t.Fatalf("bestelltes Exemplar löschen: %v", err)
	}
	ausgesondert := testdaten.LegeExemplarImZulaufAn(t, pool, titelID, "AB-ZULAUF-STATUS")
	if err := repository.NewKatalogAbfragen(pool).AendereExemplarStatus(ctx, ausgesondert, nil, repository.ExemplarStatusStand{IstAusgesondert: true}, "", nil, testdaten.LegeBearbeiterAn(t, pool)); err != nil {
		t.Fatalf("bestelltes Exemplar aussondern: %v", err)
	}
	// Gegenprobe: Dieselbe Tür an einem Exemplar aus dem Bestand ist ein Abgang.
	imBestand := testdaten.LegeExemplarAn(t, pool, titelID, "AB-BESTAND", true, "")
	if err := repository.NewKatalogAbfragen(pool).SondereExemplarAus(ctx, imBestand, "AUSSORTIERT", bearbeiter); err != nil {
		t.Fatalf("Exemplar aus dem Bestand löschen: %v", err)
	}

	von, bis := schulzeit.Halbjahr(schulzeit.Jetzt())
	buch, err := repository.LadeAbgangsbuch(ctx, pool, von, bis)
	if err != nil {
		t.Fatalf("Abgangsbuch laden: %v", err)
	}
	var barcodes []string
	for _, z := range buch.Zeilen {
		barcodes = append(barcodes, z.Barcode)
	}
	if len(barcodes) != 1 || barcodes[0] != "AB-BESTAND" {
		t.Errorf("Zeilen: %v — erwartet allein AB-BESTAND, die zwei anderen waren nie im Bestand", barcodes)
	}

	// Dieselbe Grenze gilt für die Zahl unter der Liste.
	if _, err := pool.Exec(ctx, `UPDATE buecher_exemplare SET ausgesondert_am = NULL WHERE id = ANY($1)`,
		[]string{geloescht, imBestand}); err != nil {
		t.Fatalf("Datum leeren: %v", err)
	}
	buch, err = repository.LadeAbgangsbuch(ctx, pool, von, bis)
	if err != nil {
		t.Fatalf("Abgangsbuch laden: %v", err)
	}
	if buch.OhneZeitpunkt != 1 {
		t.Errorf("Abgänge ohne Zeitpunkt: %d, erwartet 1 — das nie eingetroffene zählt nicht mit", buch.OhneZeitpunkt)
	}
}

// Ein gelöschtes Exemplar aus dem Bestand steht mit seiner Zeile im Buch; ein bestelltes, nie
// eingetroffenes steht weder darin noch in der Zahl darunter. Beide Wege, die Exemplare löschen:
// Titel löschen und „Verlust endgültig löschen".
func TestAbgangsbuch_GeloeschtesStehtMitZeileDrin(t *testing.T) {
	pool := pgtest.Pool(t)
	testdaten.LeereBestandsdaten(t, pool)
	ctx := context.Background()
	bearbeiter := testdaten.LegeKollegiumskontoAn(t, pool, "abgang-geloescht@test.invalid")
	von, bis := schulzeit.Halbjahr(schulzeit.Jetzt())
	lade := func() repository.Abgangsbuch {
		t.Helper()
		buch, err := repository.LadeAbgangsbuch(ctx, pool, von, bis)
		if err != nil {
			t.Fatalf("Abgangsbuch laden: %v", err)
		}
		return buch
	}
	zeile := func(buch repository.Abgangsbuch, barcode string) (repository.AbgangsZeile, bool) {
		for _, z := range buch.Zeilen {
			if z.Barcode == barcode {
				return z, true
			}
		}
		return repository.AbgangsZeile{}, false
	}

	// Titel löschen: ein Exemplar aus dem Bestand, eines aus dem Zulauf.
	titelID := testdaten.LegeTitelMitSignaturAn(t, pool, "Mit Zulauf geloescht", "Zul 1", 0)
	testdaten.LegeExemplarAn(t, pool, titelID, "AG-BESTAND", true, "")
	testdaten.LegeExemplarImZulaufAn(t, pool, titelID, "AG-ZULAUF")
	if _, err := repository.NewTitelAbfragen(pool).LoescheTitel(ctx, repository.TitelLoeschauftrag{TitelIDs: []string{titelID}, Grund: "AUSSORTIERT", BearbeiterID: bearbeiter}); err != nil {
		t.Fatalf("Titel löschen: %v", err)
	}
	buch := lade()
	z, ok := zeile(buch, "AG-BESTAND")
	if !ok || z.Titel != "Mit Zulauf geloescht" || z.Signatur != "Zul 1" || z.Grund != "AUSSORTIERT" {
		t.Errorf("AG-BESTAND nach dem Löschen des Titels: %+v (gefunden %v), erwartet Titel, Signatur und Grund", z, ok)
	}
	if _, ok := zeile(buch, "AG-ZULAUF"); ok {
		t.Error("AG-ZULAUF steht im Abgangsbuch, war aber nie im Bestand")
	}
	if buch.AusKatalogGeloescht != 0 {
		t.Errorf("nach dem Löschen des Titels: %d ohne Zeile, erwartet 0", buch.AusKatalogGeloescht)
	}

	// Verlust endgültig löschen: ein bestelltes Exemplar, im Status-Editor als verloren ausgebucht.
	zweiterTitel := testdaten.LegeTitelMitSignaturAn(t, pool, "Nie geliefert, verloren", "Zul 2", 0)
	verloren := testdaten.LegeExemplarImZulaufAn(t, pool, zweiterTitel, "AG-ZULAUF-VERLOREN")
	if err := repository.NewKatalogAbfragen(pool).AendereExemplarStatus(ctx, verloren, nil, repository.ExemplarStatusStand{IstAusgesondert: true}, "", nil, testdaten.LegeBearbeiterAn(t, pool)); err != nil {
		t.Fatalf("bestelltes Exemplar aussondern: %v", err)
	}
	entfernt, err := repository.NewInventurAbfragen(pool).LoescheVerlusteEndgueltig(ctx, []string{verloren}, bearbeiter)
	if err != nil || len(entfernt) != 1 {
		t.Fatalf("Verlust endgültig löschen: %v, entfernt %v — dann misst dieser Test nichts", err, entfernt)
	}
	buch = lade()
	if _, ok := zeile(buch, "AG-ZULAUF-VERLOREN"); ok || buch.AusKatalogGeloescht != 0 {
		t.Errorf("AG-ZULAUF-VERLOREN war nie im Bestand: Zeile %v, ohne Zeile %d, erwartet keins von beidem", ok, buch.AusKatalogGeloescht)
	}

	// Eine Spur ohne Vermerk und ohne Zeile zählt weiter mit: So sehen Löschungen aus der Zeit vor
	// den Buchzeilen aus. Hat das Exemplar eine Zeile, zählt seine Spur nicht noch einmal.
	for _, barcode := range []string{"AG-ZULAUF", "AG-BESTAND"} {
		if _, err := pool.Exec(ctx, `UPDATE audit_log SET details = details - $1 WHERE details->>'barcode_id' = $2`,
			repository.ProtokollDetailWarImBestand, barcode); err != nil {
			t.Fatalf("Vermerk entfernen: %v", err)
		}
	}
	if n := lade().AusKatalogGeloescht; n != 1 {
		t.Errorf("Spuren ohne Vermerk: %d ohne Zeile, erwartet 1 (AG-ZULAUF; AG-BESTAND hat seine Zeile)", n)
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
				Grund: "VERLUST", GrundText: "Verlust", Topf: mitteltopf.Land},
			{Datum: time.Date(2026, time.May, 3, 10, 0, 0, 0, schulzeit.Zone()),
				Barcode: "B-00815", Titel: "Gregs Tagebuch", Signatur: "Jug Gre",
				Grund: "AUSSORTIERT", GrundText: "Aussortiert", Topf: mitteltopf.Schultraeger},
		},
		OhneZeitpunkt: 7,
		// Die zweite Lücke des Nachweises: Körperlich gelöschte Exemplare stehen in keiner Liste
		// oben. Gemessen wird sie am fertigen Blatt, weil genau dieses Blatt unterschrieben und
		// abgeheftet wird.
		AusKatalogGeloescht: 3,
	}

	roh, err := pdf.ErzeugeAbgangsbuch(abgangsbuchBlatt(abgangsbuchAntwortAus(buch)), pdf.SchuleInfo{Name: "Philipp-Reis-Schule", Strasse: "Schulstr. 1", PLZ: "61440", Ort: "Oberursel"})
	if err != nil {
		t.Fatalf("Abgangsbuch drucken: %v", err)
	}
	blatt := strings.Join(pdftest.Texte(t, roh), " ")

	for _, muss := range []string{
		"Abgangsbuch",
		"16.03.2026", "15.09.2026", // der Zeitraum steht auf dem Blatt
		"B-00042", "B-00815",
		"Verlust", "Aussortiert",
		"Summe " + mitteltopf.Beschriftung(mitteltopf.Land) + ": 1 Exemplar",
		"Summe " + mitteltopf.Beschriftung(mitteltopf.Schultraeger) + ": 1 Exemplar",
		"7 weitere Exemplare", // der Hinweis auf die Abgänge ohne Zeitpunkt
		"3 Exemplare wurden in diesem Zeitraum aus dem Katalog gelöscht", // die zweite Lücke
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
	roh, err := pdf.ErzeugeAbgangsbuch(abgangsbuchBlatt(abgangsbuchAntwortAus(buch)), pdf.SchuleInfo{Name: "Philipp-Reis-Schule"})
	if err != nil {
		t.Fatalf("Abgangsbuch drucken: %v", err)
	}
	blatt := strings.Join(pdftest.Texte(t, roh), " ")
	if !strings.Contains(blatt, "kein Exemplar aus dem Bestand gegangen") {
		t.Errorf("leerer Zeitraum sagt es nicht:\n%s", blatt)
	}
}

// Ein Exemplar, das schon im vorigen Halbjahr ausgesondert war, bleibt dort stehen, wenn später
// sein Titel gelöscht wird: Ein Halbjahr, das jemand unterschrieben hat, darf keine Zeile
// verlieren, und im laufenden steht es nicht noch einmal.
func TestAbgangsbuch_GeloeschterTitelBehaeltSeineZeile(t *testing.T) {
	pool := pgtest.Pool(t)
	testdaten.LeereBestandsdaten(t, pool)
	ctx := context.Background()
	loc := schulzeit.Zone()
	bearbeiter := testdaten.LegeKollegiumskontoAn(t, pool, "abgangsbuch-loeschen@test.invalid")

	jetzt := schulzeit.Jetzt()
	vonJetzt, bisJetzt := schulzeit.Halbjahr(jetzt)
	vonVorher, bisVorher := schulzeit.Halbjahr(vonJetzt.AddDate(0, 0, -1))
	ausgesondertAm := time.Date(vonVorher.Year(), vonVorher.Month(), vonVorher.Day(), 10, 0, 0, 0, loc).AddDate(0, 0, 7)

	titelID := testdaten.LegeTitelMitSignaturAn(t, pool, "Bald geloescht", "Loe 1", 0)
	exID := testdaten.LegeExemplarAn(t, pool, titelID, "AB-WEG", true, "")
	if _, err := pool.Exec(ctx, `
		UPDATE buecher_exemplare
		SET ist_ausgesondert = true, ist_ausleihbar = false, aussonderung_grund = 'BESCHAEDIGUNG'
		WHERE id = $1`, exID); err != nil {
		t.Fatalf("aussondern: %v", err)
	}
	// Ein eigenes UPDATE: trg_exemplar_abgangsdatum stempelt beim Aussondern die Uhrzeit.
	if _, err := pool.Exec(ctx, `UPDATE buecher_exemplare SET ausgesondert_am = $2 WHERE id = $1`,
		exID, ausgesondertAm); err != nil {
		t.Fatalf("Abgangsdatum ins vorige Halbjahr legen: %v", err)
	}
	lade := func(von, bis time.Time) repository.Abgangsbuch {
		t.Helper()
		buch, err := repository.LadeAbgangsbuch(ctx, pool, von, bis)
		if err != nil {
			t.Fatalf("Abgangsbuch %s bis %s: %v", von.Format("02.01.2006"), bis.Format("02.01.2006"), err)
		}
		return buch
	}
	if n := len(lade(vonVorher, bisVorher).Zeilen); n != 1 {
		t.Fatalf("voriges Halbjahr vor dem Löschen: %d Zeilen, erwartet eine — dann misst dieser Test nichts", n)
	}

	// Durch die echte Tür löschen, nicht per SQL: Gemessen wird der Weg, den das Personal nimmt.
	if _, err := repository.NewTitelAbfragen(pool).LoescheTitel(ctx, repository.TitelLoeschauftrag{TitelIDs: []string{titelID}, Grund: "AUSSORTIERT", BearbeiterID: bearbeiter}); err != nil {
		t.Fatalf("Titel löschen: %v", err)
	}

	vorher := lade(vonVorher, bisVorher)
	if len(vorher.Zeilen) != 1 {
		t.Fatalf("voriges Halbjahr nach dem Löschen: %d Zeilen, erwartet eine", len(vorher.Zeilen))
	}
	z := vorher.Zeilen[0]
	if z.Barcode != "AB-WEG" || z.Grund != "BESCHAEDIGUNG" || !z.Datum.Equal(ausgesondertAm) || z.Titel != "Bald geloescht" {
		t.Errorf("Zeile nach dem Löschen: %+v, erwartet AB-WEG, BESCHAEDIGUNG, %s, Bald geloescht", z, ausgesondertAm)
	}
	if vorher.AusKatalogGeloescht != 0 {
		t.Errorf("voriges Halbjahr: %d ohne Zeile, erwartet 0", vorher.AusKatalogGeloescht)
	}
	if laufend := lade(vonJetzt, bisJetzt); len(laufend.Zeilen) != 0 || laufend.AusKatalogGeloescht != 0 {
		t.Errorf("laufendes Halbjahr: %d Zeilen, %d ohne Zeile — das Exemplar ging im vorigen ab",
			len(laufend.Zeilen), laufend.AusKatalogGeloescht)
	}
}
