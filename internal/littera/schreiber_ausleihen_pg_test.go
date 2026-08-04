package littera

import (
	"context"
	"strings"
	"testing"
	"time"
)

// ausleihWelt legt einen kleinen Bestand mit Personen an und liefert die beiden Berichte,
// die SchreibeAusleihen als Eingang braucht.
func ausleihWelt(t *testing.T, s *Schreiber, ab *Altbestand) (BestandBericht, PersonenBericht) {
	t.Helper()
	ctx := context.Background()
	bestandBericht, err := s.SchreibeBestand(ctx, ab)
	if err != nil {
		t.Fatalf("SchreibeBestand: %v", err)
	}
	personenBericht, err := s.SchreibePersonen(ctx, ab)
	if err != nil {
		t.Fatalf("SchreibePersonen: %v", err)
	}
	return bestandBericht, personenBericht
}

func ausleihe(id, exemplar, leserID string, tage int) Ausleihe {
	start := time.Date(2010, 9, 1, 8, 0, 0, 0, time.UTC)
	return Ausleihe{ID: id, ExemplarID: exemplar, LeserID: leserID,
		AusgeliehenAm: start.AddDate(0, 0, tage), Frist: start.AddDate(1, 0, tage)}
}

// TestAusleiheTrifftDieRichtigeSpalte: ausleihen trägt den Entleiher polymorph, und
// check_loan_borrower lässt genau EINE der beiden Spalten zu. Eine Lehrkraft in
// schueler_id wäre nicht nur falsch, sie käme gar nicht erst durch.
func TestAusleiheTrifftDieRichtigeSpalte(t *testing.T) {
	pool := pgTestPool(t)
	leereAlles(t, pool)
	s, _ := testSchreiber(t, pool, nil)

	ab := bestand(titel("1", "Ein Buch", ""), titel("2", "Noch eins", ""))
	ab.Leser = []Leser{
		leser("S1", "101", "07H1", ArtSchueler),
		leser("L1", "201", "", ArtLehrkraft),
	}
	ab.Ausleihen = []Ausleihe{
		ausleihe("A1", "E1", "S1", 0),
		ausleihe("A2", "E2", "L1", 1),
	}
	bestandBericht, personenBericht := ausleihWelt(t, s, ab)

	bericht, err := s.SchreibeAusleihen(context.Background(), ab, bestandBericht, personenBericht)
	if err != nil {
		t.Fatalf("SchreibeAusleihen: %v", err)
	}
	if bericht.Geschrieben != 2 || !bericht.AbgleichOK {
		t.Fatalf("2 Ausleihen erwartet, gemeldet: %+v", bericht)
	}

	if n := zaehle(t, pool,
		`SELECT count(*) FROM ausleihen WHERE schueler_id IS NOT NULL AND NOT ist_handapparat`); n != 1 {
		t.Errorf("die Schülerausleihe muss an schueler_id hängen, gefunden: %d", n)
	}
	// ist_handapparat ist die Kennzeichnung, die die Anwendung für Lehrerausleihen benutzt.
	if n := zaehle(t, pool,
		`SELECT count(*) FROM ausleihen WHERE ausleiher_benutzer_id IS NOT NULL AND ist_handapparat`); n != 1 {
		t.Errorf("die Lehrerausleihe muss an ausleiher_benutzer_id hängen und Handapparat sein, gefunden: %d", n)
	}
}

// TestZweiteOffeneAusleiheWirdAbgelehnt: uniq_ausleihen_aktiv_exemplar lässt höchstens
// EINE offene Ausleihe je Exemplar zu — ein Buch liegt nicht bei zwei Leuten. Ohne die
// Vorsortierung liefe die zweite Zeile in einen 23505 mitten im Batch.
func TestZweiteOffeneAusleiheWirdAbgelehnt(t *testing.T) {
	pool := pgTestPool(t)
	leereAlles(t, pool)
	s, protokoll := testSchreiber(t, pool, nil)

	ab := bestand(titel("1", "Ein Buch", ""))
	ab.Leser = []Leser{leser("S1", "101", "07H1", ArtSchueler), leser("S2", "102", "07H1", ArtSchueler)}
	ab.Ausleihen = []Ausleihe{
		ausleihe("ALT", "E1", "S1", 0),
		ausleihe("NEU", "E1", "S2", 30), // dieselbe Kopie, später ausgegeben
	}
	bestandBericht, personenBericht := ausleihWelt(t, s, ab)

	bericht, err := s.SchreibeAusleihen(context.Background(), ab, bestandBericht, personenBericht)
	if err != nil {
		t.Fatalf("die Doppelbelegung darf den Lauf nicht abbrechen: %v", err)
	}
	if bericht.Geschrieben != 1 || bericht.Doppelbelegung != 1 {
		t.Errorf("1 geschrieben / 1 Doppelbelegung erwartet, gemeldet: %+v", bericht)
	}
	// Es muss die JÜNGERE gewinnen — die ältere ist die Karteileiche.
	var schuelerBarcode string
	err = pool.QueryRow(context.Background(), `
		SELECT s.barcode_id FROM ausleihen a JOIN schueler s ON s.id = a.schueler_id`).Scan(&schuelerBarcode)
	if err != nil {
		t.Fatalf("Abfrage: %v", err)
	}
	if schuelerBarcode != "102" {
		t.Errorf("die jüngere Ausleihe (Schüler 102) muss gewinnen, gefunden: %s", schuelerBarcode)
	}
	if text := protokoll(); !strings.Contains(text, "littera_id=ALT") {
		t.Errorf("die verworfene Ausleihe muss im Protokoll stehen:\n%s", text)
	}
}

// TestAusleiheOhneEntleiherWirdGemeldet: 341 Ausleihen des Altbestands hängen an
// Sammelkonten und unklaren Gruppen, die bewusst nicht angelegt werden. Sie blind zu
// schreiben scheiterte am Fremdschlüssel und risse den ganzen Lauf mit; sie
// stillschweigend zu verwerfen verschwiege, dass ein Buch verliehen war.
func TestAusleiheOhneEntleiherWirdGemeldet(t *testing.T) {
	pool := pgTestPool(t)
	leereAlles(t, pool)
	s, protokoll := testSchreiber(t, pool, nil)

	ab := bestand(titel("1", "Ein Buch", ""), titel("2", "Noch eins", ""))
	ab.Leser = []Leser{leser("FB", "301", "FB Bio", ArtSonstige)}
	ab.Ausleihen = []Ausleihe{
		ausleihe("A1", "E1", "FB", 0),          // Entleiher nicht übernommen
		ausleihe("A2", "GIBTS NICHT", "FB", 1), // Exemplar unbekannt
	}
	bestandBericht, personenBericht := ausleihWelt(t, s, ab)

	bericht, err := s.SchreibeAusleihen(context.Background(), ab, bestandBericht, personenBericht)
	if err != nil {
		t.Fatalf("SchreibeAusleihen: %v", err)
	}
	if bericht.Geschrieben != 0 || bericht.OhneEntleiher != 1 || bericht.OhneExemplar != 1 {
		t.Errorf("0 geschrieben, je 1 ohne Entleiher/Exemplar erwartet, gemeldet: %+v", bericht)
	}
	text := protokoll()
	for _, erwartet := range []string{"littera_id=A1", "littera_id=A2", "nicht übernommen"} {
		if !strings.Contains(text, erwartet) {
			t.Errorf("das Protokoll nennt %q nicht:\n%s", erwartet, text)
		}
	}
}

// TestRueckgabeVorAusgabeWirdVerworfen: check_return_date verlangt rueckgabe_am >=
// ausgeliehen_am. Ein Rückgabedatum davor ist ein Datenfehler in Littera — die Zeile
// deswegen zu verlieren wäre unnötig.
func TestRueckgabeVorAusgabeWirdVerworfen(t *testing.T) {
	pool := pgTestPool(t)
	leereAlles(t, pool)
	s, protokoll := testSchreiber(t, pool, nil)

	ab := bestand(titel("1", "Ein Buch", ""))
	ab.Leser = []Leser{leser("S1", "101", "07H1", ArtSchueler)}
	a := ausleihe("A1", "E1", "S1", 0)
	a.Zurueckgegeben = true
	a.RueckgabeAm = a.AusgeliehenAm.AddDate(0, 0, -5)
	ab.Ausleihen = []Ausleihe{a}
	bestandBericht, personenBericht := ausleihWelt(t, s, ab)

	bericht, err := s.SchreibeAusleihen(context.Background(), ab, bestandBericht, personenBericht)
	if err != nil {
		t.Fatalf("SchreibeAusleihen: %v", err)
	}
	if bericht.Geschrieben != 1 {
		t.Fatalf("die Ausleihe muss ankommen, gemeldet: %+v", bericht)
	}
	if n := zaehle(t, pool, `SELECT count(*) FROM ausleihen WHERE rueckgabe_am IS NULL`); n != 1 {
		t.Errorf("das unmögliche Rückgabedatum muss verworfen sein, gefunden: %d offene", n)
	}
	if text := protokoll(); !strings.Contains(text, "vor dem Verleihdatum") {
		t.Errorf("die Verwerfung muss protokolliert werden:\n%s", text)
	}
}

// TestZurueckgegebeneAusleiheBelegtDasExemplarNicht: „zurückgegeben" ohne Rückgabedatum
// steht in dieser Anwendung für „läuft noch". Die Doppelbelegungsprüfung muss dieselbe
// Auslegung benutzen wie der INSERT — sonst schreibt sie zwei Zeilen, die der partielle
// Unique-Index anschließend nicht nebeneinander duldet.
func TestZurueckgegebeneAusleiheBelegtDasExemplarNicht(t *testing.T) {
	pool := pgTestPool(t)
	leereAlles(t, pool)
	s, _ := testSchreiber(t, pool, nil)

	ab := bestand(titel("1", "Ein Buch", ""))
	ab.Leser = []Leser{leser("S1", "101", "07H1", ArtSchueler), leser("S2", "102", "07H1", ArtSchueler)}

	alt := ausleihe("ALT", "E1", "S1", 0)
	alt.Zurueckgegeben = true
	alt.RueckgabeAm = alt.AusgeliehenAm.AddDate(0, 0, 10) // sauber zurückgegeben

	ab.Ausleihen = []Ausleihe{alt, ausleihe("NEU", "E1", "S2", 30)}
	bestandBericht, personenBericht := ausleihWelt(t, s, ab)

	bericht, err := s.SchreibeAusleihen(context.Background(), ab, bestandBericht, personenBericht)
	if err != nil {
		t.Fatalf("SchreibeAusleihen: %v", err)
	}
	if bericht.Geschrieben != 2 || bericht.Doppelbelegung != 0 {
		t.Errorf("beide Ausleihen dürfen nebeneinander stehen, gemeldet: %+v", bericht)
	}
}
