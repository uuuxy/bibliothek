package littera

import (
	"os"
	"path/filepath"
	"testing"
)

// TestEchterAltbestand prüft die Zuordnung gegen den ECHTEN Littera-Export statt gegen
// Beispielzeilen — dieselbe Idee wie bei den PG-Tests: gegated über eine Umgebungsvariable,
// damit CI ohne die Dateien grün bleibt.
//
//	mdb-export littera_sav.mdb Titel    > $LITTERA_CSV_DIR/titel.csv
//	mdb-export littera_sav.mdb Exemplar > $LITTERA_CSV_DIR/exemplar.csv
//	mdb-export littera_sav.mdb Verlag   > $LITTERA_CSV_DIR/verlag.csv
//	LITTERA_CSV_DIR=... go test ./internal/littera/
//
// Der Kern ist die Verwaisungs-Prüfung: Wenn auch nur EIN Exemplar seinen Titel nicht
// findet, stimmt die Fremdschlüssel-Zuordnung nicht — und genau daran wäre das alte
// Werkzeug gescheitert, das `TitelID` statt `Titel` las. Beim Lauf gegen den Altbestand
// vom 03.08.2026: 10.732 Titel, 61.520 Exemplare, 0 verwaist.
func TestEchterAltbestand(t *testing.T) {
	basis := os.Getenv("LITTERA_CSV_DIR")
	if basis == "" {
		t.Skip("LITTERA_CSV_DIR nicht gesetzt — Lauf gegen den echten Altbestand übersprungen")
	}

	titel := ladeTitel(t, filepath.Join(basis, "titel.csv"))
	exemplare := ladeExemplare(t, filepath.Join(basis, "exemplar.csv"))

	if len(titel) < 1000 {
		t.Fatalf("nur %d Titel gelesen — sieht nach falscher oder leerer Datei aus", len(titel))
	}
	if len(exemplare) < 1000 {
		t.Fatalf("nur %d Exemplare gelesen — sieht nach falscher oder leerer Datei aus", len(exemplare))
	}

	// DIE Invariante: Jedes Exemplar muss seinen Titel finden.
	bekannt := make(map[string]bool, len(titel))
	for _, ti := range titel {
		bekannt[ti.ID] = true
	}
	var verwaist int
	for _, e := range exemplare {
		if !bekannt[e.TitelID] {
			verwaist++
		}
	}
	if verwaist > 0 {
		t.Errorf("%d Exemplare finden ihren Titel nicht — die Fremdschluessel-Zuordnung stimmt nicht",
			verwaist)
	}

	// Die Signatur ist der Grund, warum der Bestand ueberhaupt auffindbar ist. Wenn sie
	// bei der Mehrheit fehlt, stimmt die Spaltenzuordnung (Sig1/Sig2) nicht.
	signaturen, abweichend := SignaturJeTitel(exemplare)
	if len(signaturen)*10 < len(titel)*9 {
		t.Errorf("nur %d von %d Titeln haben eine Signatur — Sig1/Sig2 pruefen",
			len(signaturen), len(titel))
	}
	t.Logf("Altbestand: %d Titel, %d Exemplare, %d mit Signatur, davon %d uneinheitlich",
		len(titel), len(exemplare), len(signaturen), len(abweichend))

	pruefeAutorenAbdeckung(t, basis, titel)
	leser := pruefeLeserEinordnung(t, basis)
	pruefeAusleihen(t, basis, leser, exemplare)
}

// pruefeLeserEinordnung belegt, dass die Weiche Schüler/Lehrkraft am echten Bestand
// greift — und dass keine Lehrkraft in der Schuelermenge landet.
// Liefert die gelesenen Leser zurück, damit die Ausleih-Prüfung sie mitverwenden kann,
// ohne die Datei ein zweites Mal zu lesen.
func pruefeLeserEinordnung(t *testing.T, basis string) []Leser {
	t.Helper()

	gf, err := os.Open(filepath.Join(basis, "leser_ug.csv")) //nolint:gosec // Testumgebung
	if err != nil {
		t.Skipf("leser_ug.csv fehlt — Leser-Pruefung uebersprungen: %v", err)
	}
	t.Cleanup(func() { _ = gf.Close() }) //nolint:errcheck // Testaufraeumen
	gruppen, err := LeseLesergruppen(gf)
	if err != nil {
		t.Fatalf("Lesergruppen: %v", err)
	}

	lf, err := os.Open(filepath.Join(basis, "leser.csv")) //nolint:gosec // Testumgebung
	if err != nil {
		t.Skipf("leser.csv fehlt — Leser-Pruefung uebersprungen: %v", err)
	}
	t.Cleanup(func() { _ = lf.Close() }) //nolint:errcheck // Testaufraeumen
	leser, err := LeseLeser(lf, gruppen)
	if err != nil {
		t.Fatalf("Leser lesen: %v", err)
	}

	schueler := NurArt(leser, ArtSchueler)
	lehrkraefte := NurArt(leser, ArtLehrkraft)
	abgegangen := NurArt(leser, ArtAbgegangen)

	if len(schueler) == 0 || len(lehrkraefte) == 0 {
		t.Fatalf("Einordnung greift nicht: %d Schueler, %d Lehrkraefte",
			len(schueler), len(lehrkraefte))
	}

	// Die eigentliche Zusicherung: Jeder Schueler hat eine Klasse. schueler.klasse ist
	// in der Zieltabelle NOT NULL — eine leere Klasse waere ein Schreibfehler.
	for _, s := range schueler {
		if s.Klasse == "" {
			t.Fatalf("Schueler (ID %s) ohne Klasse — schueler.klasse ist NOT NULL", s.ID)
		}
	}

	// Und die Gegenprobe: keine Lehrkraft in der Schuelermenge.
	lehrerNummern := map[string]bool{}
	for _, l := range lehrkraefte {
		lehrerNummern[l.ID] = true
	}
	for _, s := range schueler {
		if lehrerNummern[s.ID] {
			t.Errorf("Leser-ID %s steht in beiden Mengen", s.ID)
		}
	}

	t.Logf("Leser: %d gesamt — %d Schueler, %d Lehrkraefte, %d abgegangen, %d sonstige, %d unklar",
		len(leser), len(schueler), len(lehrkraefte), len(abgegangen),
		len(NurArt(leser, ArtSonstige)), len(NurArt(leser, ArtUnbekannt)))

	// schueler.abgaenger_jahr ist NOT NULL. Jeder Schueler, der geschrieben werden
	// soll, braucht also einen Wert — sonst bricht der Import auf halber Strecke ab.
	const schuljahrEnde = 2027
	ohneJahr, abschlussklassen := 0, 0
	unbekannteKlassen := map[string]int{}
	for _, s := range schueler {
		if _, ok := AbgaengerJahr(s.Klasse, schuljahrEnde); !ok {
			ohneJahr++
			unbekannteKlassen[s.Klasse]++
			continue
		}
		if IstAbschlussklasse(s.Klasse) {
			abschlussklassen++
		}
	}
	if ohneJahr > 0 {
		t.Errorf("%d Schueler ohne ableitbares Abgangsjahr (Klassen: %v) — abgaenger_jahr ist NOT NULL",
			ohneJahr, unbekannteKlassen)
	}
	t.Logf("Abgangsjahr: fuer alle %d Schueler ableitbar, davon %d in einer Abschlussklasse (9H/10R/13)",
		len(schueler), abschlussklassen)

	return leser
}

// pruefeAusleihen belegt, dass jede Ausleihe ihr Exemplar UND ihren Entleiher findet.
// Eine Ausleihe ins Leere waere beim Schreiben ein Fremdschluesselfehler — und der
// bricht den Import mitten in der Transaktion ab.
func pruefeAusleihen(t *testing.T, basis string, leser []Leser, exemplare []Exemplar) {
	t.Helper()

	vf, err := os.Open(filepath.Join(basis, "verleih.csv")) //nolint:gosec // Testumgebung
	if err != nil {
		t.Skipf("verleih.csv fehlt — Ausleih-Pruefung uebersprungen: %v", err)
	}
	t.Cleanup(func() { _ = vf.Close() }) //nolint:errcheck // Testaufraeumen

	ausleihen, err := LeseAusleihen(vf)
	if err != nil {
		t.Fatalf("Ausleihen lesen: %v", err)
	}
	if len(ausleihen) == 0 {
		t.Fatal("keine Ausleihen gelesen")
	}

	bekannteExemplare := make(map[string]bool, len(exemplare))
	for _, e := range exemplare {
		bekannteExemplare[e.ID] = true
	}
	bekannteLeser := make(map[string]bool, len(leser))
	for _, l := range leser {
		bekannteLeser[l.ID] = true
	}

	ohneExemplar, ohneLeser, ohneDatum := 0, 0, 0
	for _, a := range ausleihen {
		if !bekannteExemplare[a.ExemplarID] {
			ohneExemplar++
		}
		if !bekannteLeser[a.LeserID] {
			ohneLeser++
		}
		if a.AusgeliehenAm.IsZero() {
			ohneDatum++
		}
	}
	// Der Entleiher MUSS immer gefunden werden: Verleih.Leser zeigt auf
	// Leser.Buchungsnummer. Waere hier auch nur eine Zeile offen, haette man den
	// falschen Schluessel erwischt (ueber Lesernummer treffen nur 792 von 15.615).
	if ohneLeser > 0 {
		t.Errorf("%d Ausleihen finden ihren Entleiher nicht — falscher Fremdschluessel?", ohneLeser)
	}

	// Beim Exemplar ist eine verschwindende Zahl Datenfehler in Littera, kein
	// Zuordnungsfehler. Ein nennenswerter Anteil waere dagegen der falsche Schluessel
	// (ueber Exemplarnummer treffen nur 7.985 von 15.615).
	if ohneExemplar*1000 > len(ausleihen) {
		t.Errorf("%d von %d Ausleihen finden ihr Exemplar nicht — das ist zu viel fuer Datenfehler",
			ohneExemplar, len(ausleihen))
	} else if ohneExemplar > 0 {
		t.Logf("%d von %d Ausleihen ohne Exemplar — Datenfehler im Altbestand, beim Import zu melden",
			ohneExemplar, len(ausleihen))
	}
	if ohneDatum > 0 {
		t.Errorf("%d Ausleihen ohne lesbares Verleihdatum", ohneDatum)
	}

	t.Logf("Ausleihen: %d gesamt, %d offen, %d ohne Frist (rueckgabe_frist ist NOT NULL)",
		len(ausleihen), len(NurOffene(ausleihen)), len(OhneFrist(ausleihen)))
}

// pruefeAutorenAbdeckung belegt, dass die Aufloesung ueber Personen/Personen_Zuordnung
// deutlich mehr Titel mit einem Autor versorgt als die Freitext-Verfasserangabe.
//
// Ohne sie waere der Katalog fuer die Autorensuche weitgehend blind: Nur gut ein
// Viertel der Titel traegt eine Verfasserangabe.
func pruefeAutorenAbdeckung(t *testing.T, basis string, titel []Titel) {
	t.Helper()

	pf, err := os.Open(filepath.Join(basis, "personen.csv")) //nolint:gosec // Testumgebung
	if err != nil {
		t.Skipf("personen.csv fehlt — Autoren-Pruefung uebersprungen: %v", err)
	}
	t.Cleanup(func() { _ = pf.Close() }) //nolint:errcheck // Testaufraeumen

	personen, err := LesePersonen(pf)
	if err != nil {
		t.Fatalf("Personen lesen: %v", err)
	}

	zf, err := os.Open(filepath.Join(basis, "personen_zuordnung.csv")) //nolint:gosec // Testumgebung
	if err != nil {
		t.Skipf("personen_zuordnung.csv fehlt — Autoren-Pruefung uebersprungen: %v", err)
	}
	t.Cleanup(func() { _ = zf.Close() }) //nolint:errcheck // Testaufraeumen

	autoren, err := AutorenJeTitel(personen, zf)
	if err != nil {
		t.Fatalf("Zuordnung lesen: %v", err)
	}

	vorher := 0
	for _, ti := range titel {
		if ti.Autor != "" {
			vorher++
		}
	}
	nachher := 0
	for _, ti := range MitAutoren(titel, autoren) {
		if ti.Autor != "" {
			nachher++
		}
	}

	if nachher <= vorher {
		t.Errorf("Aufloesung bringt nichts: vorher %d, nachher %d Titel mit Autor", vorher, nachher)
	}
	// Deutlich mehr als die Haelfte muss abgedeckt sein, sonst stimmt die
	// Funktions-Auswahl (Verfasser = 0) nicht.
	if nachher*2 < len(titel) {
		t.Errorf("nur %d von %d Titeln haben einen Autor — Funktionsschluessel pruefen",
			nachher, len(titel))
	}
	t.Logf("Autoren: %d von %d ueber Verfasserangabe, %d nach Personen-Aufloesung",
		vorher, len(titel), nachher)
}

func ladeTitel(t *testing.T, pfad string) []Titel {
	t.Helper()
	f, err := os.Open(pfad) //nolint:gosec // Pfad kommt aus der Testumgebung
	if err != nil {
		t.Fatalf("%s: %v", pfad, err)
	}
	t.Cleanup(func() {
		if err := f.Close(); err != nil {
			t.Logf("Datei schliessen: %v", err)
		}
	})
	titel, err := LeseTitel(f)
	if err != nil {
		t.Fatalf("Titel lesen: %v", err)
	}
	return titel
}

func ladeExemplare(t *testing.T, pfad string) []Exemplar {
	t.Helper()
	f, err := os.Open(pfad) //nolint:gosec // Pfad kommt aus der Testumgebung
	if err != nil {
		t.Fatalf("%s: %v", pfad, err)
	}
	t.Cleanup(func() {
		if err := f.Close(); err != nil {
			t.Logf("Datei schliessen: %v", err)
		}
	})
	ex, err := LeseExemplare(f)
	if err != nil {
		t.Fatalf("Exemplare lesen: %v", err)
	}
	return ex
}
