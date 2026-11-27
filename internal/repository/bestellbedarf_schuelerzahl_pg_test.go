package repository_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"bibliothek/internal/pgtest"
	"bibliothek/internal/pkg/schulzeit"
	"bibliothek/internal/repository"
	"bibliothek/internal/testdaten"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Der Bestellbedarf rechnet nach der Zahl der Kinder: Welcher Jahrgang ein Schulbuch braucht,
// zeigen die Ausleihen dieses Schuljahres, und im nächsten Schuljahr braucht es so viele, wie
// Kinder in den Jahrgang nachrücken. Der Bestand zählt alle nicht ausgesonderten Exemplare,
// auch verliehene und bestellte. Die Tests rechnen mit der Uhr des Programms; die Fristen
// liegen deshalb relativ zum Schuljahr von heute.

// bedarfHeute ist der Tag, für den die Tests rechnen, mit den Fristen des laufenden, des
// vorigen und des nächsten Schuljahres.
func bedarfHeute() (heute, fristDiesesJahr, fristVorjahr, fristNaechstesJahr time.Time) {
	heute = time.Now()
	beginn := schulzeit.SchuljahrBeginn(heute)
	ende := beginn.AddDate(1, 0, 0).Add(-time.Hour)
	return heute, ende, ende.AddDate(-1, 0, 0), ende.AddDate(1, 0, 0)
}

// kinderInKlasse legt anzahl Kinder in der Klasse an.
func kinderInKlasse(t *testing.T, pool *pgxpool.Pool, klasse string, anzahl int) []string {
	t.Helper()
	ids := make([]string, anzahl)
	for i := range ids {
		ids[i] = testdaten.LegeSchuelerAn(t, pool, fmt.Sprintf("BB-%s-%d", klasse, i), fmt.Sprintf("Kind %d", i), klasse)
	}
	return ids
}

// exemplareVon legt anzahl ausleihbare Exemplare des Titels an.
func exemplareVon(t *testing.T, pool *pgxpool.Pool, titelID, praefix string, anzahl int) []string {
	t.Helper()
	ids := make([]string, anzahl)
	for i := range ids {
		ids[i] = testdaten.LegeExemplarAn(t, pool, titelID, fmt.Sprintf("%s-%d", praefix, i), true, "")
	}
	return ids
}

// leiheAus bucht eine Ausleihe mit der Frist, auf Wunsch schon zurückgegeben.
func leiheAus(t *testing.T, pool *pgxpool.Pool, exemplarID, kindID string, frist time.Time, zurueck bool) {
	t.Helper()
	if _, err := pool.Exec(context.Background(), `
		INSERT INTO ausleihen (exemplar_id, schueler_id, ausgeliehen_am, rueckgabe_frist, rueckgabe_am)
		VALUES ($1, $2, now() - interval '1 day', $3, CASE WHEN $4 THEN now() END)`,
		exemplarID, kindID, frist, zurueck); err != nil {
		t.Fatalf("Ausleihe anlegen: %v", err)
	}
}

// leiheJedemEins gibt jedem Kind eines der Exemplare, der Reihe nach.
func leiheJedemEins(t *testing.T, pool *pgxpool.Pool, exemplare, kinder []string, frist time.Time, zurueck bool) {
	t.Helper()
	for i, kind := range kinder {
		leiheAus(t, pool, exemplare[i], kind, frist, zurueck)
	}
}

// ladeBedarfNachTitel liefert die Zeilen des Bestellbedarfs nach Titel.
func ladeBedarfNachTitel(t *testing.T, pool *pgxpool.Pool, heute time.Time) map[string]repository.Bestellbedarf {
	t.Helper()
	zeilen, err := repository.LadeBestellbedarf(context.Background(), pool, heute)
	if err != nil {
		t.Fatalf("LadeBestellbedarf: %v", err)
	}
	m := map[string]repository.Bestellbedarf{}
	for _, z := range zeilen {
		m[z.Titel] = z
	}
	return m
}

// Vier Kinder in der 7 haben Mathe 7, sechs rücken aus der 6 nach: Bei vier Exemplaren fehlen
// zwei. Deutsch 7 hat sechs Exemplare und reicht genau; bei Englisch 7 zählen die beiden
// bestellten Exemplare mit. Ein Titel ohne eigenes Cover trägt die Adresse der DNB zu seiner
// ISBN, wie Zulauf und Bestellsuche (sqlCoverOderDNB).
func TestLadeBestellbedarf_JahrgangRuecktNach(t *testing.T) {
	pool := pgtest.Pool(t)
	testdaten.LeereBestandsdaten(t, pool)
	ctx := context.Background()
	heute, frist, _, _ := bedarfHeute()

	siebte := kinderInKlasse(t, pool, "07a", 4)
	kinderInKlasse(t, pool, "06a", 6)

	mathe := testdaten.LegeTitelAn(t, pool, "LMF-Mathe 7")
	if _, err := pool.Exec(ctx, `UPDATE buecher_titel SET isbn = '9780000577092' WHERE id = $1`, mathe); err != nil {
		t.Fatalf("ISBN setzen: %v", err)
	}
	leiheJedemEins(t, pool, exemplareVon(t, pool, mathe, "MA", 4), siebte, frist, false)

	deutsch := testdaten.LegeTitelAn(t, pool, "LMF-Deutsch 7")
	leiheJedemEins(t, pool, exemplareVon(t, pool, deutsch, "DE", 6), siebte, frist, false)

	englisch := testdaten.LegeTitelAn(t, pool, "LMF-Englisch 7")
	leiheJedemEins(t, pool, exemplareVon(t, pool, englisch, "EN", 4), siebte, frist, false)
	testdaten.LegeExemplarImZulaufAn(t, pool, englisch, "EN-Z1")
	testdaten.LegeExemplarImZulaufAn(t, pool, englisch, "EN-Z2")

	got := ladeBedarfNachTitel(t, pool, heute)
	m, drin := got["LMF-Mathe 7"]
	if !drin {
		t.Fatalf("Mathe 7 fehlt: 6 Kinder rücken nach, 4 Exemplare sind da; Liste %v", got)
	}
	if m.Bedarf != 6 || m.GesamtBestand != 4 || m.Fehlen != 2 {
		t.Errorf("Mathe 7: Bedarf %d, Bestand %d, fehlen %d; erwartet 6, 4, 2", m.Bedarf, m.GesamtBestand, m.Fehlen)
	}
	if want := "https://portal.dnb.de/opac/mvb/cover?isbn=9780000577092"; m.CoverURL != want {
		t.Errorf("Cover %q, erwartet %q", m.CoverURL, want)
	}
	if !m.IstLernmittel {
		t.Error("IstLernmittel: erwartet true, sonst schlägt der Warenkorb den Topf der Bücherei vor")
	}
	if _, drin := got["LMF-Deutsch 7"]; drin {
		t.Error("Deutsch 7 steht da, obwohl 6 Exemplare für 6 Kinder reichen")
	}
	if _, drin := got["LMF-Englisch 7"]; drin {
		t.Error("Englisch 7 steht da, obwohl 4 eigene und 2 bestellte Exemplare für 6 Kinder reichen")
	}

	kinder, mitSchulbuch, err := repository.LadeBestellbedarfGrundlage(ctx, pool, heute)
	if err != nil {
		t.Fatalf("LadeBestellbedarfGrundlage: %v", err)
	}
	if kinder != 10 || mitSchulbuch != 4 {
		t.Errorf("Grundlage: %d Kinder, %d mit Schulbuch; erwartet 10 und 4", kinder, mitSchulbuch)
	}
}

// Beim Büchertausch im Juni geben die Kinder die Bücher dieses Schuljahres zurück und bekommen die
// des nächsten, mit dessen Frist. Die zurückgegebenen zählen weiter; die neuen zählen erst im
// nächsten Schuljahr. Ein Buch aus dem vorigen Schuljahr, ein Kind, das abgegangen ist, eine
// Lehrkraft (die Sicht schueler kennt sie nicht) und ein Roman aus der Bücherei zählen nicht.
func TestLadeBestellbedarf_BuechertauschUndWasNichtZaehlt(t *testing.T) {
	pool := pgtest.Pool(t)
	testdaten.LeereBestandsdaten(t, pool)
	heute, frist, fristVorjahr, fristNaechstesJahr := bedarfHeute()

	siebte := kinderInKlasse(t, pool, "07a", 4)
	kinderInKlasse(t, pool, "06a", 6)

	alt := testdaten.LegeTitelAn(t, pool, "LMF-Englisch 7")
	leiheJedemEins(t, pool, exemplareVon(t, pool, alt, "EA", 4), siebte, frist, true)
	neu := testdaten.LegeTitelAn(t, pool, "LMF-Englisch 8")
	leiheJedemEins(t, pool, exemplareVon(t, pool, neu, "EB", 4), siebte, fristNaechstesJahr, false)

	vorjahr := testdaten.LegeTitelAn(t, pool, "LMF-Englisch 6")
	leiheJedemEins(t, pool, exemplareVon(t, pool, vorjahr, "EC", 4), siebte, fristVorjahr, false)

	abgaenger := testdaten.LegeAbgaengerAn(t, pool, "BB-ABG-1", "Abgang", "07a")
	geschichte := testdaten.LegeTitelAn(t, pool, "LMF-Geschichte 7")
	leiheAus(t, pool, exemplareVon(t, pool, geschichte, "GE", 1)[0], abgaenger, frist, false)

	roman := testdaten.LegeTitelAn(t, pool, "Roman für die 7")
	leiheJedemEins(t, pool, exemplareVon(t, pool, roman, "RO", 4), siebte, frist, false)

	lehrkraft := testdaten.LegeLeserAn(t, pool, "BB-L-1", "Lehr", "Kraft", "lehrkraft")
	if _, err := pool.Exec(context.Background(), `UPDATE leser SET klasse = '07a' WHERE id = $1`, lehrkraft); err != nil {
		t.Fatalf("Klasse der Lehrkraft setzen: %v", err)
	}
	// Ohne Bestand stünde der Band da, sobald die Lehrkraft mitzählte.
	lehrerband := testdaten.LegeTitelAn(t, pool, "LMF-Mathe 7 Lehrerband")
	band := exemplareVon(t, pool, lehrerband, "LB", 1)[0]
	leiheAus(t, pool, band, lehrkraft, frist, false)
	if _, err := pool.Exec(context.Background(), `UPDATE buecher_exemplare SET ist_ausgesondert = true,
		ist_ausleihbar = false, aussonderung_grund = 'VERLUST' WHERE id = $1`, band); err != nil {
		t.Fatalf("Lehrerband aussondern: %v", err)
	}

	got := ladeBedarfNachTitel(t, pool, heute)
	if z, drin := got["LMF-Englisch 7"]; !drin || z.Bedarf != 6 || z.Fehlen != 2 {
		t.Errorf("Englisch 7, im Büchertausch zurückgegeben: %+v; erwartet Bedarf 6, fehlen 2", z)
	}
	for _, titel := range []string{"LMF-Englisch 8", "LMF-Englisch 6", "LMF-Geschichte 7", "Roman für die 7", "LMF-Mathe 7 Lehrerband"} {
		if z, drin := got[titel]; drin {
			t.Errorf("%s steht da (Bedarf %d), zählt aber nicht für dieses Schuljahr", titel, z.Bedarf)
		}
	}
}

// Ein Buch, das heute nur ein Teil des Jahrgangs hat, ein Zweig oder ein Wahlfach, braucht im
// nächsten Schuljahr denselben Anteil der nachrückenden Kinder. Eine Abschlussklasse rückt nicht
// nach, und für Jahrgang 5 ohne Jahrgang darunter gilt die Zahl von heute.
func TestLadeBestellbedarf_AnteilAbschlussUndJahrgangFuenf(t *testing.T) {
	pool := pgtest.Pool(t)
	testdaten.LeereBestandsdaten(t, pool)
	heute, frist, _, _ := bedarfHeute()

	siebte := kinderInKlasse(t, pool, "07a", 4)
	kinderInKlasse(t, pool, "06a", 8)
	franz := testdaten.LegeTitelAn(t, pool, "LMF-Französisch 7")
	leiheJedemEins(t, pool, exemplareVon(t, pool, franz, "FR", 2), siebte[:2], frist, false)

	zehnte := kinderInKlasse(t, pool, "10R1", 2)
	kinderInKlasse(t, pool, "09R1", 3)
	kinderInKlasse(t, pool, "09H1", 5)
	mathe10 := testdaten.LegeTitelAn(t, pool, "LMF-Mathe 10 R")
	leiheJedemEins(t, pool, exemplareVon(t, pool, mathe10, "MZ", 2), zehnte, frist, false)

	fuenfte := kinderInKlasse(t, pool, "05a", 3)
	mathe5 := testdaten.LegeTitelAn(t, pool, "LMF-Mathe 5")
	ex5 := exemplareVon(t, pool, mathe5, "MF", 3)
	leiheJedemEins(t, pool, ex5[:2], fuenfte[:2], frist, false)
	leiheAus(t, pool, ex5[2], fuenfte[2], frist, true)
	if _, err := pool.Exec(context.Background(), `UPDATE buecher_exemplare SET ist_ausgesondert = true,
		ist_ausleihbar = false, aussonderung_grund = 'VERLUST' WHERE id = $1`, ex5[2]); err != nil {
		t.Fatalf("Exemplar aussondern: %v", err)
	}

	got := ladeBedarfNachTitel(t, pool, heute)
	if z := got["LMF-Französisch 7"]; z.Bedarf != 4 || z.Fehlen != 2 {
		t.Errorf("Französisch 7 (2 von 4 Kindern, 8 rücken nach): %+v; erwartet Bedarf 4, fehlen 2", z)
	}
	if z := got["LMF-Mathe 10 R"]; z.Bedarf != 3 || z.Fehlen != 1 {
		t.Errorf("Mathe 10 R (aus der 9 rücken nur die 3 der 09R1 nach, die 09H1 geht ab): %+v; erwartet Bedarf 3, fehlen 1", z)
	}
	if z := got["LMF-Mathe 5"]; z.Bedarf != 3 || z.GesamtBestand != 2 || z.Fehlen != 1 {
		t.Errorf("Mathe 5 (kein Jahrgang 4, ein Exemplar verloren): %+v; erwartet Bedarf 3, Bestand 2, fehlen 1", z)
	}
}

// Ein Band über die Jahrgänge 7 und 8 bleibt zwei Jahre beim Kind. Er zählt in beiden Jahrgängen,
// auch vor seiner Frist; ein Band, den ein Kind der 6 schon im Büchertausch bekommen hat, zählt
// erst im nächsten Schuljahr.
func TestLadeBestellbedarf_MehrjahresBand(t *testing.T) {
	pool := pgtest.Pool(t)
	testdaten.LeereBestandsdaten(t, pool)
	heute, frist, _, fristNaechstesJahr := bedarfHeute()

	sechste := kinderInKlasse(t, pool, "06a", 4)
	siebte := kinderInKlasse(t, pool, "07a", 2)
	achte := kinderInKlasse(t, pool, "08a", 2)
	band := testdaten.LegeTitelAn(t, pool, "LMF-Atlas 7/8")
	if _, err := pool.Exec(context.Background(), `UPDATE buecher_titel
		SET mehrjahresband = true, jahrgang_von = 7, jahrgang_bis = 8 WHERE id = $1`, band); err != nil {
		t.Fatalf("Band einrichten: %v", err)
	}
	ex := exemplareVon(t, pool, band, "AT", 5)
	leiheJedemEins(t, pool, ex[:2], siebte, fristNaechstesJahr, false)
	leiheJedemEins(t, pool, ex[2:4], achte, frist, false)
	leiheAus(t, pool, ex[4], sechste[0], fristNaechstesJahr, false)

	// Jahrgang 7: 2 von 2 Kindern, 4 rücken nach; Jahrgang 8: 2 von 2, 2 rücken nach.
	got := ladeBedarfNachTitel(t, pool, heute)
	if z := got["LMF-Atlas 7/8"]; z.Bedarf != 6 || z.GesamtBestand != 5 || z.Fehlen != 1 {
		t.Errorf("Band 7/8: %+v; erwartet Bedarf 6 (4 + 2), Bestand 5, fehlen 1", z)
	}
}

// Zwei Auflagen desselben Buchs stehen als eine Zeile: Die Kinder beider Auflagen zählen
// zusammen, ihr Bestand auch, gezeigt wird die neueste Auflage, denn die wird bestellt, und die
// Zeile trägt die Aufschlüsselung. Ein Titel ohne Werk ist sein eigenes Buch.
func TestLadeBestellbedarf_ZaehltAmBuch(t *testing.T) {
	pool := pgtest.Pool(t)
	testdaten.LeereBestandsdaten(t, pool)
	ctx := context.Background()
	// CASCADE nimmt buecher_titel mit (werk_id verweist auf werke); deshalb vor dem Anlegen.
	if _, err := pool.Exec(ctx, `TRUNCATE werke CASCADE`); err != nil {
		t.Fatal(err)
	}
	heute, frist, _, _ := bedarfHeute()

	siebte := kinderInKlasse(t, pool, "07a", 3)
	kinderInKlasse(t, pool, "06a", 5)
	achte := kinderInKlasse(t, pool, "08a", 1)

	alt := testdaten.LegeTitelAn(t, pool, "LMF-Lambacher Schweizer 7")
	neu := testdaten.LegeTitelAn(t, pool, "LMF-Lambacher Schweizer 7 (Neubearbeitung)")
	if _, err := pool.Exec(ctx, `
		UPDATE buecher_titel
		SET erscheinungsjahr = CASE WHEN id = $1::uuid THEN 2019 ELSE 2023 END,
		    auflage = CASE WHEN id = $1::uuid THEN '3. Aufl.' ELSE '4. Aufl.' END,
		    isbn = CASE WHEN id = $1::uuid THEN '9783120000019' ELSE '9783120000026' END
		WHERE id IN ($1::uuid, $2::uuid)`, alt, neu); err != nil {
		t.Fatalf("Auflagen beschriften: %v", err)
	}
	leiheJedemEins(t, pool, exemplareVon(t, pool, alt, "ALT", 2), siebte[:2], frist, false)
	leiheJedemEins(t, pool, exemplareVon(t, pool, neu, "NEU", 1), siebte[2:], frist, false)
	if _, err := repository.FasseAuflagenZusammen(ctx, pool, alt, neu); err != nil {
		t.Fatalf("zusammenfassen: %v", err)
	}
	einzeln := testdaten.LegeTitelAn(t, pool, "LMF-Physik 8")
	leiheJedemEins(t, pool, exemplareVon(t, pool, einzeln, "EIN", 1), achte, frist, false)

	zeilen, err := repository.LadeBestellbedarf(ctx, pool, heute)
	if err != nil {
		t.Fatalf("LadeBestellbedarf: %v", err)
	}
	nachID := map[string]repository.Bestellbedarf{}
	for _, z := range zeilen {
		nachID[z.ID] = z
	}
	if _, drin := nachID[alt]; drin {
		t.Error("die alte Auflage hat eine eigene Zeile: Das Buch steht zweimal da")
	}
	z, drin := nachID[neu]
	if !drin {
		t.Fatalf("keine Zeile für das Buch (3 Kinder, 5 rücken nach, Bestand 3): %+v", zeilen)
	}
	if z.Bedarf != 5 || z.GesamtBestand != 3 || z.Fehlen != 2 || z.ISBN != "9783120000026" {
		t.Errorf("Zeile des Buchs: Bedarf %d, Bestand %d, fehlen %d, ISBN %q; erwartet 5, 3, 2 und die ISBN der neuesten Auflage",
			z.Bedarf, z.GesamtBestand, z.Fehlen, z.ISBN)
	}
	if len(z.Auflagen) != 2 || z.Auflagen[0].ID != neu || z.Auflagen[0].GesamtBestand != 1 ||
		z.Auflagen[1].ID != alt || z.Auflagen[1].GesamtBestand != 2 || z.Auflagen[1].ISBN != "9783120000019" ||
		z.Auflagen[1].Auflage != "3. Aufl." || z.Auflagen[1].Erscheinungsjahr != 2019 {
		t.Errorf("Aufschlüsselung %+v; erwartet die neue (1) vor der alten (2, 3. Aufl. 2019, mit ISBN)", z.Auflagen)
	}
	// Physik 8: 1 von 1 Kind der 8, 3 rücken aus der 7 nach.
	if e, drin := nachID[einzeln]; !drin || e.Bedarf != 3 || e.Fehlen != 2 || e.Auflagen != nil {
		t.Errorf("Titel ohne Werk %+v; erwartet Bedarf 3, fehlen 2, ohne Aufschlüsselung", e)
	}
}
