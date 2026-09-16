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

// TestLehrkraftUndLiVBekommenIhrePersonenart: Littera führt Referendare in einer eigenen Gruppe.
// Bis zum 15.09.2026 übersprang der Lauf sie als unklar; jetzt kommen sie ins Kollegium, mit der
// Personenart „liv" (Lehrkraft im Vorbereitungsdienst), Lehrkräfte mit „lehrkraft".
func TestLehrkraftUndLiVBekommenIhrePersonenart(t *testing.T) {
	pool := pgTestPool(t)
	leereAlles(t, pool)
	s, _ := testSchreiber(t, pool, nil)

	ab := &Altbestand{Leser: []Leser{
		leser("1", "31", "", ArtLehrkraft),
		leser("2", "32", "", ArtLiV),
	}}
	bericht, err := s.SchreibePersonen(context.Background(), ab)
	if err != nil {
		t.Fatalf("SchreibePersonen: %v", err)
	}
	if bericht.Lehrkraefte != 2 || !bericht.AbgleichOK {
		t.Fatalf("beide gehören ins Kollegium, gemeldet: %+v", bericht)
	}
	// Ausweis und Art stehen seit Migration 125 an der Leserzeile des Kontos, nicht am Konto.
	if n := zaehle(t, pool, `SELECT count(*) FROM leser WHERE barcode_id = '31' AND art = 'lehrkraft'`); n != 1 {
		t.Errorf("die Lehrkraft soll als Art „lehrkraft“ geführt werden, gefunden: %d", n)
	}
	if n := zaehle(t, pool, `SELECT count(*) FROM leser WHERE barcode_id = '32' AND art = 'liv'`); n != 1 {
		t.Errorf("die LiV soll als Art „liv“ geführt werden, gefunden: %d", n)
	}
	// Und beide hängen an einem Konto — sonst könnten sie sich nie anmelden.
	if n := zaehle(t, pool, `SELECT count(*) FROM benutzer b JOIN leser l ON l.id = b.leser_id
		WHERE l.barcode_id IN ('31','32')`); n != 2 {
		t.Errorf("beide Konten müssen auf ihre Leserzeile zeigen, gefunden: %d", n)
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

// TestFremdLeserNummerGewinnt: Der Ausweis liefert beim Scannen die Nummer seines Herstellers
// (gemessen „B97601826457"); Littera hält sie in FremdLeserNummer. Steht eine da, wird SIE die
// Ausweisnummer — nicht die Lesernummer. Bis zum 15.09.2026 wurde die Tabelle eingelesen und
// nie geschrieben: Nach dem Personenlauf hätte kein alter Ausweis seine Person gefunden
// (OFFEN.md 5.15).
func TestFremdLeserNummerGewinnt(t *testing.T) {
	pool := pgTestPool(t)
	leereAlles(t, pool)
	s, protokoll := testSchreiber(t, pool, nil)

	ab := &Altbestand{
		Leser: []Leser{
			leser("1", "24", "07H1", ArtSchueler),
			leser("2", "25", "07H1", ArtSchueler),
			leser("3", "26", "", ArtLehrkraft),
		},
		Ausweisnummern: map[string]string{"1": "B97601826457", "3": "B97601826458"},
	}
	if _, err := s.SchreibePersonen(context.Background(), ab); err != nil {
		t.Fatalf("SchreibePersonen: %v", err)
	}
	if n := zaehle(t, pool, `SELECT count(*) FROM schueler WHERE barcode_id = 'B97601826457'`); n != 1 {
		t.Errorf("die Herstellernummer muss der Ausweis des Schülers sein, gefunden: %d", n)
	}
	if n := zaehle(t, pool, `SELECT count(*) FROM schueler WHERE barcode_id = '24'`); n != 0 {
		t.Errorf("neben einer Herstellernummer darf die Lesernummer nicht stehen, gefunden: %d", n)
	}
	if n := zaehle(t, pool, `SELECT count(*) FROM schueler WHERE barcode_id = '25'`); n != 1 {
		t.Errorf("ohne Herstellernummer bleibt die Lesernummer der Ausweis, gefunden: %d", n)
	}
	if n := zaehle(t, pool, `SELECT count(*) FROM leser WHERE barcode_id = 'B97601826458' AND art = 'lehrkraft'`); n != 1 {
		t.Errorf("auch die Lehrkraft bekommt die Herstellernummer, gefunden: %d", n)
	}
	// Leser 2 hat keine Karte, obwohl die Schule Herstellerausweise nutzt: Ein Ausweis in
	// seiner Hand findet ihn nicht — das muss jemand erfahren, bevor er an der Theke steht.
	if text := protokoll(); !strings.Contains(text, "keine Karte in FremdLeserNummer") {
		t.Errorf("wer ohne Karte übernommen wird, muss im Protokoll stehen:\n%s", text)
	}
}

// TestAusweisnummerGleichBuchBarcodeWeichtAus: Lesernummern sind kleine Zahlen, und im Bestand
// tragen Tausende Bücher nackte Mediennummern (Server, 15.09.2026: 30.658). Die Theke löst eine
// Nummer ohne Vorsilbe zuerst als Buch auf (resolveOhnePraefix) — ein Schüler mit derselben
// Nummer buchte beim Klick in der Namenssuche oder beim Scan seines neuen Ausweises das Buch.
// Eine Nummer, die schon ein Buch trägt, ist deshalb vergeben (OFFEN.md 5.15).
func TestAusweisnummerGleichBuchBarcodeWeichtAus(t *testing.T) {
	pool := pgTestPool(t)
	leereAlles(t, pool)
	s, protokoll := testSchreiber(t, pool, nil)
	ctx := context.Background()

	var titelID string
	if err := pool.QueryRow(ctx, `INSERT INTO buecher_titel (titel, autor, medientyp, ist_lernmittel)
		VALUES ('Kollisionsband', 'Prüfer', 'Buch', false) RETURNING id`).Scan(&titelID); err != nil {
		t.Fatalf("Titel anlegen: %v", err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO buecher_exemplare (titel_id, barcode_id, ist_ausleihbar, einkaufspreis)
		VALUES ($1, '24', true, 10.00)`, titelID); err != nil {
		t.Fatalf("Exemplar anlegen: %v", err)
	}

	ab := &Altbestand{Leser: []Leser{leser("1", "24", "07H1", ArtSchueler)}}
	if _, err := s.SchreibePersonen(ctx, ab); err != nil {
		t.Fatalf("SchreibePersonen: %v", err)
	}
	if n := zaehle(t, pool, `SELECT count(*) FROM schueler WHERE barcode_id = '24'`); n != 0 {
		t.Errorf("die Buchnummer 24 darf kein Ausweis werden, gefunden: %d", n)
	}
	if n := zaehle(t, pool, `SELECT count(*) FROM schueler WHERE barcode_id = 'L-1'`); n != 1 {
		t.Errorf("der Schüler soll die Ersatznummer L-1 tragen, gefunden: %d", n)
	}
	if text := protokoll(); !strings.Contains(text, "Barcode eines Buchs") {
		t.Errorf("die Kollision mit dem Buch muss protokolliert werden:\n%s", text)
	}
}

// TestErsatznummerWeichtVergebenerNummerAus: Die Ersatznummer „L-<Littera-Nummer>" kann schon
// vergeben sein, etwa von Hand an eine Lehrkraft. Vor Migration 125 galt die Eindeutigkeit nur
// je Tabelle, und die Theke suchte bei „L-" zuerst unter den Schülern — der Ausweis der
// Lehrkraft lud still den Schüler (Rasterdurchgang 15.09.2026). Heute wäre es kein stiller
// Fehlgriff mehr, sondern ein abgewiesener Import; ausweichen soll er trotzdem.
func TestErsatznummerWeichtVergebenerNummerAus(t *testing.T) {
	pool := pgTestPool(t)
	leereAlles(t, pool)
	s, protokoll := testSchreiber(t, pool, nil)
	ctx := context.Background()

	// Zwei Anweisungen: Eine schreibende CTE sähe das äußere UPDATE nicht (die Zeile liegt
	// außerhalb seines Schnappschusses) und änderte still 0 Zeilen.
	var handLeserID string
	if err := pool.QueryRow(ctx, `
		INSERT INTO benutzer (vorname, nachname, email, rolle, aktiv)
		VALUES ('Hand', 'Vergeben', 'hand@schule.invalid', 'kollegium', true)
		RETURNING leser_id::text`).Scan(&handLeserID); err != nil {
		t.Fatalf("Lehrkraft anlegen: %v", err)
	}
	if _, err := pool.Exec(ctx,
		`UPDATE leser SET barcode_id = 'L-2' WHERE id = $1`, handLeserID); err != nil {
		t.Fatalf("Ausweis der Lehrkraft eintragen: %v", err)
	}
	ab := &Altbestand{Leser: []Leser{
		leser("1", "24", "07H1", ArtSchueler),
		leser("2", "24", "08R1", ArtSchueler),
	}}
	bericht, err := s.SchreibePersonen(ctx, ab)
	if err != nil {
		t.Fatalf("SchreibePersonen: %v", err)
	}
	if bericht.Schueler != 2 {
		t.Fatalf("beide Schüler müssen ankommen, gemeldet: %+v", bericht)
	}
	if n := zaehle(t, pool, `SELECT count(*) FROM schueler WHERE barcode_id = 'L-2'`); n != 0 {
		t.Errorf("L-2 trägt schon die Lehrkraft, kein Schüler darf sie bekommen, gefunden: %d", n)
	}
	if n := zaehle(t, pool, `SELECT count(*) FROM schueler WHERE barcode_id = 'L-2-2'`); n != 1 {
		t.Errorf("der zweite Schüler soll auf L-2-2 ausweichen, gefunden: %d", n)
	}
	if text := protokoll(); !strings.Contains(text, "Ausweis L-2-2 vergeben") {
		t.Errorf("die ausgewichene Ersatznummer muss im Protokoll stehen:\n%s", text)
	}
}

// TestMehrereKartenWerdenProtokolliert: Hat ein Leser mehrere Karten hinterlegt, gilt die
// zuletzt angelegte (LeseAusweisnummern) — mit einer älteren Karte findet die Theke niemanden.
// Das gehört ins Protokoll; bis zum 15.09.2026 las niemand AusweisMehrfach.
func TestMehrereKartenWerdenProtokolliert(t *testing.T) {
	pool := pgTestPool(t)
	leereAlles(t, pool)
	s, protokoll := testSchreiber(t, pool, nil)

	ab := &Altbestand{
		Leser:           []Leser{leser("1", "24", "07H1", ArtSchueler)},
		Ausweisnummern:  map[string]string{"1": "B97601826459"},
		AusweisMehrfach: []string{"1"},
	}
	if _, err := s.SchreibePersonen(context.Background(), ab); err != nil {
		t.Fatalf("SchreibePersonen: %v", err)
	}
	if text := protokoll(); !strings.Contains(text, "mehrere Karten") {
		t.Errorf("mehrere hinterlegte Karten müssen protokolliert werden:\n%s", text)
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
// Ausleihpfad: Die Theke sucht bei jedem Ausweis unter den LESERN (Migration 125). Landete
// die importierte Lehrkraft nur in der Kontentabelle, waere jeder gedruckte Lehrerausweis
// wertlos — und gemerkt haette man es erst an der Theke.
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
	lehrkraft, err := repository.NewStudentRepository(pool).
		GetLeserByBarcode(context.Background(), "B97601826458")
	if err != nil {
		t.Fatalf("Ausweis-Abfrage: %v", err)
	}
	if lehrkraft == nil {
		t.Fatal("die importierte Lehrkraft ist ueber ihren Ausweis nicht auffindbar")
	}
	if lehrkraft.Nachname != "Nach2" {
		t.Errorf("falsche Lehrkraft geladen: %+v", lehrkraft)
	}
	if lehrkraft.Art != "lehrkraft" {
		t.Errorf("die Theke muss die Art am Treffer sehen, gelesen: %q", lehrkraft.Art)
	}
}
