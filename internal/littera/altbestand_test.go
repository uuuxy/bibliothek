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
