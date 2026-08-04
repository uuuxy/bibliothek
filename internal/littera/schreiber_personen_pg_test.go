package littera

import (
	"context"
	"strings"
	"testing"
	"time"

	"bibliothek/repository"
)

func leser(id, nummer, klasse string, art LeserArt) Leser {
	return Leser{ID: id, Lesernummer: nummer, Vorname: "Vor" + id, Nachname: "Nach" + id,
		Klasse: klasse, Art: art}
}

// TestNurSchuelerUndLehrkraefteWerdenGeschrieben: Praktikanten, Sekretariat und die
// Sammelkonten der Fachbereiche sind weder das eine noch das andere. Bei Personendaten
// ist eine ausgelassene Zeile das kleinere Übel als eine falsch einsortierte — aber sie
// muss im Protokoll stehen, sonst verschwinden 20 Personen und 302 Ausleihen lautlos.
func TestNurSchuelerUndLehrkraefteWerdenGeschrieben(t *testing.T) {
	pool := pgTestPool(t)
	leereAlles(t, pool)
	s, protokoll := testSchreiber(t, pool, nil)

	ab := &Altbestand{Leser: []Leser{
		leser("1", "101", "07H1", ArtSchueler),
		leser("2", "102", "", ArtLehrkraft),
		leser("3", "103", "FB Bio", ArtSonstige),
		leser("4", "104", "IMPORT", ArtUnbekannt),
		leser("5", "105", "Ab", ArtAbgegangen),
	}}

	bericht, err := s.SchreibePersonen(context.Background(), ab)
	if err != nil {
		t.Fatalf("SchreibePersonen: %v", err)
	}
	if bericht.Schueler != 2 || bericht.Lehrkraefte != 1 || bericht.Uebersprungen != 2 {
		t.Errorf("2 Schüler / 1 Lehrkraft / 2 ausgelassen erwartet, gemeldet: %d / %d / %d",
			bericht.Schueler, bericht.Lehrkraefte, bericht.Uebersprungen)
	}
	if !bericht.AbgleichOK {
		t.Error("der Abgleich mit der Datenbank muss stimmen")
	}
	if n := zaehle(t, pool, `SELECT count(*) FROM schueler`); n != 2 {
		t.Errorf("2 Schüler in der Datenbank erwartet, gefunden: %d", n)
	}
	text := protokoll()
	for _, id := range []string{"littera_id=3", "littera_id=4"} {
		if !strings.Contains(text, id) {
			t.Errorf("die ausgelassene Person %s muss im Protokoll stehen:\n%s", id, text)
		}
	}
}

// TestAbgaengerBekommenEinJahr: schueler.abgaenger_jahr ist NOT NULL, die Gruppe
// „Abgegangen" trägt als Klassenbezeichnung aber nur „Ab" — daraus rechnet AbgaengerJahr
// nichts aus. Ohne die Sonderregel scheiterten alle 71 Abgänger am NOT NULL.
func TestAbgaengerBekommenEinJahr(t *testing.T) {
	pool := pgTestPool(t)
	leereAlles(t, pool)
	s, _ := testSchreiber(t, pool, func(o *Optionen) { o.SchuljahrEnde = 2027 })

	ab := &Altbestand{Leser: []Leser{leser("5", "105", "Ab", ArtAbgegangen)}}
	if _, err := s.SchreibePersonen(context.Background(), ab); err != nil {
		t.Fatalf("SchreibePersonen: %v", err)
	}

	var jahr int
	var abgaenger bool
	err := pool.QueryRow(context.Background(),
		`SELECT abgaenger_jahr, ist_abgaenger FROM schueler`).Scan(&jahr, &abgaenger)
	if err != nil {
		t.Fatalf("Abfrage: %v", err)
	}
	if jahr != 2027 || !abgaenger {
		t.Errorf("abgaenger_jahr=2027 und ist_abgaenger=true erwartet, gefunden: %d / %v", jahr, abgaenger)
	}
}

// TestSchuelerOhneAbleitbaresJahrWirdGemeldet: ein GERATENES Abgangsjahr archiviert
// irgendwann still den falschen Schüler. Lieber die Zeile auslassen und melden.
func TestSchuelerOhneAbleitbaresJahrWirdGemeldet(t *testing.T) {
	pool := pgTestPool(t)
	leereAlles(t, pool)
	s, protokoll := testSchreiber(t, pool, nil)

	ab := &Altbestand{Leser: []Leser{leser("1", "101", "Sonderklasse", ArtSchueler)}}
	bericht, err := s.SchreibePersonen(context.Background(), ab)
	if err != nil {
		t.Fatalf("SchreibePersonen: %v", err)
	}
	if bericht.Schueler != 0 || bericht.Uebersprungen != 1 {
		t.Errorf("die Zeile muss ausgelassen werden, gemeldet: %+v", bericht)
	}
	if text := protokoll(); !strings.Contains(text, "Abgangsjahr") {
		t.Errorf("das Protokoll muss den Grund nennen:\n%s", text)
	}
}

// TestLehrkraftOhneMailBekommtUnzustellbarenPlatzhalter: benutzer.email ist NOT NULL
// UNIQUE, im Altbestand fehlt sie bei 157 von 158 Lehrkräften. Der Platzhalter MUSS auf
// eine nicht auflösbare Domäne zeigen — ein erfundener Wert unter der Schuldomäne ginge
// irgendwann an eine echte, fremde Person.
func TestLehrkraftOhneMailBekommtUnzustellbarenPlatzhalter(t *testing.T) {
	pool := pgTestPool(t)
	leereAlles(t, pool)
	s, protokoll := testSchreiber(t, pool, nil)

	ab := &Altbestand{Leser: []Leser{
		leser("2", "102", "", ArtLehrkraft),
		leser("3", "103", "", ArtLehrkraft),
	}}
	if _, err := s.SchreibePersonen(context.Background(), ab); err != nil {
		t.Fatalf("SchreibePersonen: %v", err)
	}

	rows, err := pool.Query(context.Background(), `SELECT email, aktiv FROM benutzer ORDER BY email`)
	if err != nil {
		t.Fatalf("Abfrage: %v", err)
	}
	defer rows.Close()
	var adressen []string
	for rows.Next() {
		var mail string
		var aktiv bool
		if err := rows.Scan(&mail, &aktiv); err != nil {
			t.Fatalf("Scan: %v", err)
		}
		if !strings.HasSuffix(mail, ".invalid") {
			t.Errorf("der Platzhalter muss unzustellbar sein (.invalid), gefunden: %q", mail)
		}
		// aktiv=true ist nötig, damit die Omnibox den Ausweis findet — die Lehrer-Abfrage
		// filtert auf aktiv. Den Login sperrt die unzustellbare Adresse, nicht dieses Flag.
		if !aktiv {
			t.Errorf("Lehrkräfte müssen als Entleiher auffindbar sein (aktiv=true): %q", mail)
		}
		adressen = append(adressen, mail)
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("Abfrage: %v", err)
	}
	if len(adressen) != 2 || adressen[0] == adressen[1] {
		t.Errorf("zwei verschiedene Adressen erwartet, gefunden: %v", adressen)
	}
	if text := protokoll(); !strings.Contains(text, "keine E-Mail im Altbestand") {
		t.Errorf("der Platzhalter muss protokolliert werden:\n%s", text)
	}
}

// TestDoppelteAusweisnummerWeichtAus: schueler.barcode_id ist unter aktiven Zeilen
// eindeutig; im Altbestand teilen sich zwei Leser die Nummer 24. Ohne Ausweichen verlöre
// man eine Person samt ihren Ausleihen.
func TestDoppelteAusweisnummerWeichtAus(t *testing.T) {
	pool := pgTestPool(t)
	leereAlles(t, pool)
	s, protokoll := testSchreiber(t, pool, nil)

	ab := &Altbestand{Leser: []Leser{
		leser("1", "24", "07H1", ArtSchueler),
		leser("2", "24", "08R1", ArtSchueler),
	}}
	bericht, err := s.SchreibePersonen(context.Background(), ab)
	if err != nil {
		t.Fatalf("SchreibePersonen: %v", err)
	}
	if bericht.Schueler != 2 {
		t.Fatalf("beide Schüler müssen ankommen, gemeldet: %+v", bericht)
	}
	if n := zaehle(t, pool, `SELECT count(*) FROM schueler WHERE barcode_id = 'L-2'`); n != 1 {
		t.Errorf("der zweite Schüler soll die Ersatznummer L-2 tragen, gefunden: %d", n)
	}
	if text := protokoll(); !strings.Contains(text, "Karte muss neu gedruckt werden") {
		t.Errorf("die Ersatzvergabe muss protokolliert werden — die Karte stimmt nicht mehr:\n%s", text)
	}
}

// TestGeburtsdatumLandetNichtInDerZukunft: die Jahrhundertgrenze bei zweistelligen Jahren
// liegt in Go fest auf 69. Am Altbestand landen dadurch 69 Personen in den Jahren 2046
// bis 2068 — ein Geburtsdatum in der Zukunft ist immer falsch.
func TestGeburtsdatumLandetNichtInDerZukunft(t *testing.T) {
	pool := pgTestPool(t)
	leereAlles(t, pool)
	s, _ := testSchreiber(t, pool, nil)

	l := leser("1", "101", "07H1", ArtSchueler)
	l.Geburtsdatum = "03/17/63" // gemeint ist 1963, Go liest 2063
	if _, err := s.SchreibePersonen(context.Background(), &Altbestand{Leser: []Leser{l}}); err != nil {
		t.Fatalf("SchreibePersonen: %v", err)
	}

	var geboren time.Time
	if err := pool.QueryRow(context.Background(),
		`SELECT geburtsdatum FROM schueler`).Scan(&geboren); err != nil {
		t.Fatalf("Abfrage: %v", err)
	}
	if geboren.Year() != 1963 {
		t.Errorf("1963 erwartet, gespeichert: %d", geboren.Year())
	}
}

// TestAdressenWerdenNichtUebernommen hält die Datenminimierung fest, damit sie nicht
// beiläufig wieder aufgeweicht wird. Litteras Adressfelder sind bei 1.927 von 1.991
// Lesern gefüllt; ihr Zweck laut schema.sql ist der Rechnungsversand, und die gepflegte
// Quelle dafür ist die LUSD — nicht ein Altbestand.
func TestAdressenWerdenNichtUebernommen(t *testing.T) {
	pool := pgTestPool(t)
	leereAlles(t, pool)
	s, _ := testSchreiber(t, pool, nil)

	l := leser("1", "101", "07H1", ArtSchueler)
	l.Strasse, l.PLZ, l.Ort, l.EMail = "Hauptstr. 1", "35390", "Gießen", "eltern@example.org"
	if _, err := s.SchreibePersonen(context.Background(), &Altbestand{Leser: []Leser{l}}); err != nil {
		t.Fatalf("SchreibePersonen: %v", err)
	}

	n := zaehle(t, pool, `SELECT count(*) FROM schueler
		WHERE strasse IS NOT NULL OR plz IS NOT NULL OR ort IS NOT NULL OR eltern_email IS NOT NULL`)
	if n != 0 {
		t.Errorf("Anschrift und Eltern-Mail dürfen nicht aus Littera kommen, gefunden: %d Zeilen", n)
	}
}

// TestImportierteLehrkraftIstAmScannerAuffindbar schliesst den Kreis zwischen Import und
// Ausleihpfad: Die Omnibox filtert Lehrkraefte auf aktiv=true. Legte der Import sie
// inaktiv an, waere jeder gedruckte Lehrerausweis wertlos — und gemerkt haette man es
// erst an der Theke.
//
// Der Login bleibt trotzdem gesperrt: Er laeuft ausschliesslich ueber IMAP gegen den
// Schul-Mailserver, und die Platzhalter-Adresse gibt es dort nicht.
func TestImportierteLehrkraftIstAmScannerAuffindbar(t *testing.T) {
	pool := pgTestPool(t)
	leereAlles(t, pool)
	s, _ := testSchreiber(t, pool, nil)

	l := leser("2", "B97601826458", "", ArtLehrkraft)
	if _, err := s.SchreibePersonen(context.Background(), &Altbestand{Leser: []Leser{l}}); err != nil {
		t.Fatalf("SchreibePersonen: %v", err)
	}

	// Genau die Abfrage, die die Omnibox stellt.
	lehrkraft, err := repository.NewUserRepository(pool).
		GetLehrerByBarcode(context.Background(), "B97601826458")
	if err != nil {
		t.Fatalf("Lehrer-Abfrage: %v", err)
	}
	if lehrkraft == nil {
		t.Fatal("die importierte Lehrkraft ist ueber ihren Ausweis nicht auffindbar")
	}
	if lehrkraft.Nachname != "Nach2" {
		t.Errorf("falsche Lehrkraft geladen: %+v", lehrkraft)
	}
}
