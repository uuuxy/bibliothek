package repository_test

import (
	"context"
	"testing"

	"bibliothek/internal/pgtest"
	"bibliothek/internal/testdaten"
	"bibliothek/repository"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Der Bestellbedarf misst sich am Gesamtbestand (nicht ausgesondert) und nicht am gerade
// verfügbaren: Sonst gälte jeder verliehene Klassensatz, im Schuljahr der Normalfall, als
// Bedarf, und die vorgeschlagene Menge wäre zu hoch.
func TestLadeBestellbedarf_GesamtNichtVerfuegbar(t *testing.T) {
	pool := pgtest.Pool(t)
	testdaten.LeereBestandsdaten(t, pool)
	ctx := context.Background()

	schueler := testdaten.LegeSchuelerAn(t, pool, "R-1", "Ida", "5a")

	// A) Voll verliehener Klassensatz: 30 Exemplare, alle ausgeliehen (verfügbar 0, gesamt 30),
	// Schwelle 5: kein Grund zu bestellen.
	tVoll := testdaten.LegeTitelAn(t, pool, "LMF-Mathe 7", 5)
	for i := 0; i < 30; i++ {
		e := testdaten.LegeExemplarAn(t, pool, tVoll, barcodeN("V", i), true, "")
		leiheExemplarAus(t, pool, e, schueler)
	}

	// B) Echter Fehlbestand: nur 3 eigene Exemplare (gesamt 3 < Meldebestand 5).
	tKnapp := testdaten.LegeTitelAn(t, pool, "LMF-Deutsch 5", 5)
	for i := 0; i < 3; i++ {
		testdaten.LegeExemplarAn(t, pool, tKnapp, barcodeN("K", i), true, "")
	}

	// C) Schon bestellt: 2 eigene und 3 als bestellt vermerkte Exemplare ergeben gesamt 5. Der
	// Bedarf ist gedeckt, weil die bestellten im Gesamtbestand mitzählen.
	tBestellt := testdaten.LegeTitelAn(t, pool, "LMF-Bio 6", 5)
	testdaten.LegeExemplarAn(t, pool, tBestellt, "BE-1", true, "")
	testdaten.LegeExemplarAn(t, pool, tBestellt, "BE-2", true, "")
	for i := 0; i < 3; i++ {
		testdaten.LegeExemplarAn(t, pool, tBestellt, barcodeN("BE", i+3), false, "bestellt")
	}

	// Schwelle 5, wie der Meldebestand der drei Titel.
	reorders, err := repository.LadeBestellbedarf(ctx, pool, bestellbedarfFilterLMF(), 5)
	if err != nil {
		t.Fatalf("LadeBestellbedarf: %v", err)
	}

	got := map[string]repository.Bestellbedarf{}
	for _, r := range reorders {
		got[r.Titel] = r
	}

	if _, drin := got["LMF-Mathe 7"]; drin {
		t.Error("voll verliehener Klassensatz (gesamt 30) wurde als Bestellbedarf gemeldet — Fehlalarm")
	}
	if _, drin := got["LMF-Bio 6"]; drin {
		t.Error("bereits vollständig bestellter Titel (gesamt 5) wurde erneut gemeldet")
	}
	k, drin := got["LMF-Deutsch 5"]
	if !drin {
		t.Fatal("Titel mit echtem Fehlbestand (gesamt 3 < Meldebestand 5) fehlt in der Liste")
	}
	if k.GesamtBestand != 3 || k.Meldebestand != 5 {
		t.Errorf("Fehlbestand-Zahlen falsch: gesamt=%d meldebestand=%d (erwartet 3 / 5)", k.GesamtBestand, k.Meldebestand)
	}
}

// Die eingestellte Schwelle steuert die Aufnahme, nicht der Meldebestand am Titel: Ein Titel
// mit gesamt 3 erscheint bei Schwelle 5 und fehlt bei Schwelle 2.
func TestLadeBestellbedarf_SchwelleSteuert(t *testing.T) {
	pool := pgtest.Pool(t)
	testdaten.LeereBestandsdaten(t, pool)
	ctx := context.Background()

	tKnapp := testdaten.LegeTitelAn(t, pool, "LMF-Physik 8", 5) // meldebestand pauschal 5
	for i := 0; i < 3; i++ {
		testdaten.LegeExemplarAn(t, pool, tKnapp, barcodeN("P", i), true, "")
	}

	filter := bestellbedarfFilterLMF()
	r, err := repository.LadeBestellbedarf(ctx, pool, filter, 5)
	if err != nil {
		t.Fatalf("LadeBestellbedarf error: %v", err)
	}
	if len(r) != 1 {
		t.Errorf("Schwelle 5 = %d, erwartet 1 Treffer (gesamt 3 < 5)", len(r))
	}

	r2, err := repository.LadeBestellbedarf(ctx, pool, filter, 2)
	if err != nil {
		t.Fatalf("LadeBestellbedarf error: %v", err)
	}
	if len(r2) != 0 {
		t.Errorf("Schwelle 2 = %d, erwartet 0 Treffer (gesamt 3 ≥ 2) — Schwelle steuert, nicht Meldebestand 5", len(r2))
	}
}

// Ein von Hand angelegtes Lernmittel trägt oft einen Titel in Klartext („Mathematik Neue Wege
// 9") und das Kennzeichen der Lernmittel nur in der Signatur („LMF Ma", der Vorschlag der
// Maske). Auch ein solches Buch fällt unter die Schwelle.
func TestLadeBestellbedarf_LMFNurInSignatur(t *testing.T) {
	pool := pgtest.Pool(t)
	testdaten.LeereBestandsdaten(t, pool)
	ctx := context.Background()

	tID := testdaten.LegeTitelMitSignaturAn(t, pool, "Mathematik Neue Wege 9", "LMF Ma", 5)
	for i := 0; i < 3; i++ {
		testdaten.LegeExemplarAn(t, pool, tID, barcodeN("S", i), true, "")
	}

	reorders, err := repository.LadeBestellbedarf(ctx, pool, bestellbedarfFilterLMF(), 5)
	if err != nil {
		t.Fatalf("LadeBestellbedarf: %v", err)
	}

	got := map[string]repository.Bestellbedarf{}
	for _, r := range reorders {
		got[r.Titel] = r
	}
	if _, drin := got["Mathematik Neue Wege 9"]; !drin {
		t.Error("LMF-Kennzeichen nur in der Signatur wurde nicht erkannt — Titel fehlt in der Bestellbedarf-Liste")
	}
}

// bestellbedarfFilterLMF liefert die Bedingung der Vorgabe (Lernmittel), ohne eine Anfrage zu
// bauen.
func bestellbedarfFilterLMF() string {
	frag, _ := repository.BestandsFilterBedingung("lmf")
	return frag
}

func leiheExemplarAus(t *testing.T, pool *pgxpool.Pool, exemplarID, schuelerID string) {
	t.Helper()
	if _, err := pool.Exec(context.Background(),
		`INSERT INTO ausleihen (exemplar_id, schueler_id, rueckgabe_frist)
		 VALUES ($1, $2, CURRENT_DATE)`, exemplarID, schuelerID); err != nil {
		t.Fatalf("Ausleihe anlegen: %v", err)
	}
}

func barcodeN(prefix string, i int) string {
	return prefix + "-" + string(rune('a'+i%26)) + string(rune('0'+i/26))
}

// Zwei Auflagen desselben Buchs stehen als eine Zeile. Ihre Summe steht gegen die Schwelle,
// gezeigt wird die neueste Auflage, denn die wird bestellt, und die Zeile trägt die
// Aufschlüsselung. Getrennt gezählt läge jede für sich unter der Schwelle, und dasselbe Buch
// stünde zweimal da, auch wenn beide zusammen reichen. Ein Titel ohne Werk ist sein eigenes
// Buch.
func TestLadeBestellbedarf_ZaehltAmBuch(t *testing.T) {
	pool := pgtest.Pool(t)
	testdaten.LeereBestandsdaten(t, pool)
	ctx := context.Background()
	// CASCADE nimmt buecher_titel mit (werk_id verweist auf werke); deshalb vor dem Anlegen.
	if _, err := pool.Exec(ctx, `TRUNCATE werke CASCADE`); err != nil {
		t.Fatal(err)
	}

	alt := testdaten.LegeTitelAn(t, pool, "LMF-Lambacher Schweizer 7", 5)
	neu := testdaten.LegeTitelAn(t, pool, "LMF-Lambacher Schweizer 7 (Neubearbeitung)", 5)
	einzeln := testdaten.LegeTitelAn(t, pool, "LMF-Physik 8", 5)
	if _, err := pool.Exec(ctx, `
		UPDATE buecher_titel
		SET erscheinungsjahr = CASE WHEN id = $1::uuid THEN 2019 ELSE 2023 END,
		    auflage = CASE WHEN id = $1::uuid THEN '3. Aufl.' ELSE '4. Aufl.' END,
		    isbn = CASE WHEN id = $1::uuid THEN '9783120000019' ELSE '9783120000026' END
		WHERE id IN ($1::uuid, $2::uuid)`, alt, neu); err != nil {
		t.Fatalf("Auflagen beschriften: %v", err)
	}
	testdaten.LegeExemplarAn(t, pool, alt, "ALT-1", true, "")
	testdaten.LegeExemplarAn(t, pool, alt, "ALT-2", true, "")
	testdaten.LegeExemplarAn(t, pool, neu, "NEU-1", true, "")
	testdaten.LegeExemplarAn(t, pool, einzeln, "EIN-1", true, "")
	if _, err := repository.FasseAuflagenZusammen(ctx, pool, alt, neu); err != nil {
		t.Fatalf("zusammenfassen: %v", err)
	}
	filter := bestellbedarfFilterLMF()
	nachID := func(liste []repository.Bestellbedarf) map[string]repository.Bestellbedarf {
		m := map[string]repository.Bestellbedarf{}
		for _, r := range liste {
			m[r.ID] = r
		}
		return m
	}

	// Schwelle 3: Zusammen sind es 3, also kein Bedarf. Getrennt stünden beide da (2 und 1).
	r, err := repository.LadeBestellbedarf(ctx, pool, filter, 3)
	if err != nil {
		t.Fatalf("LadeBestellbedarf: %v", err)
	}
	got := nachID(r)
	if _, drin := got[alt]; drin {
		t.Error("Schwelle 3: die alte Auflage (2) steht allein da — gezählt wird am Buch (2 + 1 = 3)")
	}
	if _, drin := got[neu]; drin {
		t.Error("Schwelle 3: die neue Auflage (1) steht allein da — gezählt wird am Buch (2 + 1 = 3)")
	}
	if e, drin := got[einzeln]; !drin || e.GesamtBestand != 1 || e.Auflagen != nil {
		t.Errorf("Schwelle 3: Titel ohne Werk %+v — erwartet wie bisher: da, Bestand 1, ohne Aufschlüsselung", e)
	}

	// Schwelle 5: eine Zeile für das Buch mit der neuesten Auflage, der Summe und der
	// Aufschlüsselung.
	r, err = repository.LadeBestellbedarf(ctx, pool, filter, 5)
	if err != nil {
		t.Fatalf("LadeBestellbedarf: %v", err)
	}
	got = nachID(r)
	if _, drin := got[alt]; drin {
		t.Error("Schwelle 5: die alte Auflage hat eine eigene Zeile — das Buch steht zweimal da")
	}
	z, drin := got[neu]
	if !drin {
		t.Fatal("Schwelle 5: keine Zeile für das Buch (Summe 3 < 5)")
	}
	if z.GesamtBestand != 3 || z.VerfuegbarBestand != 3 || z.ISBN != "9783120000026" {
		t.Errorf("Zeile des Buchs: gesamt %d, verfügbar %d, ISBN %q — erwartet 3, 3 und die ISBN der neuesten Auflage",
			z.GesamtBestand, z.VerfuegbarBestand, z.ISBN)
	}
	if len(z.Auflagen) != 2 || z.Auflagen[0].ID != neu || z.Auflagen[0].GesamtBestand != 1 ||
		z.Auflagen[1].ID != alt || z.Auflagen[1].GesamtBestand != 2 || z.Auflagen[1].ISBN != "9783120000019" ||
		z.Auflagen[1].Auflage != "3. Aufl." || z.Auflagen[1].Erscheinungsjahr != 2019 {
		t.Errorf("Aufschlüsselung %+v — erwartet die neue (1) vor der alten (2, 3. Aufl. 2019, mit ISBN)", z.Auflagen)
	}
}
