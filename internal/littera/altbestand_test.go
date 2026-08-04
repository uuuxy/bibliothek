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
