package bestellung

// Eigentumsvermerk auf den gemailten Etikettenbögen: Der Mailweg liest die Einstellung
// etikett_eigentumsvermerk wie Selbstdruck (buchetikett.LadeKopf) und Lieferanten-Link. Hielte
// er die Werksvorgabe „Eigentum des Landes Hessen" fest, ergäben zwei Wege zum selben Buch
// zwei verschiedene Aufkleber. Geprüft am fertigen PDF-Inhaltsstrom (pdftest.Suchtext), wie
// alle Etiketten-Gates.

import (
	"strings"
	"testing"

	"bibliothek/internal/buchetikett"
	"bibliothek/internal/pdftest"
	"bibliothek/pkg/mitteltopf"
	"bibliothek/repository"
)

func einTestEtikett() []buchetikett.Etikett {
	return []buchetikett.Etikett{{
		BarcodeID: "BM-TEST-1", Titel: "Bestellmail-Testband", Autor: "Prüfer", ISBN: "9783123456789",
	}}
}

func TestBestellmailEtikettenTragenKonfiguriertenVermerk(t *testing.T) {
	kopf := buchetikett.KopfAus(&repository.SystemEinstellungen{
		SchuleName: "Testschule", EtikettEigentumsvermerk: "Eigentum des Kreises Wetterau"})
	boegen, err := etikettenboegen(einTestEtikett(), kopf, false, mitteltopf.Land)
	if err != nil {
		t.Fatalf("etikettenboegen: %v", err)
	}
	if len(boegen) == 0 {
		t.Fatal("kein Bogen erzeugt")
	}
	for _, b := range boegen {
		text := pdftest.Suchtext(t, b.Data)
		if !strings.Contains(text, "Eigentum des Kreises Wetterau") {
			t.Errorf("%s: konfigurierter Eigentumsvermerk fehlt auf dem Bogen", b.Name)
		}
		if strings.Contains(text, repository.EigentumsvermerkVorgabe) {
			t.Errorf("%s: Werksvorgabe steht trotz konfiguriertem Vermerk auf dem Bogen", b.Name)
		}
		if !strings.Contains(text, "Testschule") {
			t.Errorf("%s: der Schulname aus den Einstellungen fehlt auf dem Bogen", b.Name)
		}
	}
}

func TestBestellmailEtikettenFallenOhneKonfigurationAufWerksvorgabe(t *testing.T) {
	kopf := buchetikett.KopfAus(&repository.SystemEinstellungen{SchuleName: "Testschule"})
	boegen, err := etikettenboegen(einTestEtikett(), kopf, false, mitteltopf.Land)
	if err != nil {
		t.Fatalf("etikettenboegen: %v", err)
	}
	for _, b := range boegen {
		if !strings.Contains(pdftest.Suchtext(t, b.Data), repository.EigentumsvermerkVorgabe) {
			t.Errorf("%s: ohne Konfiguration muss die Werksvorgabe drucken", b.Name)
		}
	}
}

// Das große Lernmittel-Etikett („Eigentum des Landes") geht nur mit, wenn die Bestellung
// nicht der Schülerbücherei gilt. Die
// Alt-Bestellung ohne Zuordnung behält beide Größen.
func TestBestellmailGrossesEtikettFolgtDemTopf(t *testing.T) {
	faelle := []struct {
		name              string
		istHauptlieferant bool
		mittel            string
		wantGross         bool
	}{
		{"Hauptlieferant, Lernmittelfreiheit", true, mitteltopf.Land, true},
		{"Hauptlieferant, Schülerbücherei", true, mitteltopf.Schultraeger, false},
		{"Hauptlieferant, ohne Zuordnung", true, "", true},
		{"anderer Lieferant, Lernmittelfreiheit", false, mitteltopf.Land, false},
	}
	for _, f := range faelle {
		t.Run(f.name, func(t *testing.T) {
			kopf := buchetikett.KopfAus(&repository.SystemEinstellungen{SchuleName: "Testschule"})
			boegen, err := etikettenboegen(einTestEtikett(), kopf, f.istHauptlieferant, f.mittel)
			if err != nil {
				t.Fatalf("etikettenboegen: %v", err)
			}
			hatGross, hatKlein := false, false
			for _, b := range boegen {
				hatGross = hatGross || strings.HasPrefix(b.Name, "etiketten_gross")
				hatKlein = hatKlein || strings.HasPrefix(b.Name, "etiketten_klein")
			}
			if !hatKlein {
				t.Error("der kleine Bogen fehlt — er geht in jedem Fall mit")
			}
			if hatGross != f.wantGross {
				t.Errorf("großes Lernmittel-Etikett im Anhang = %v, erwartet %v", hatGross, f.wantGross)
			}
		})
	}
}

// Der Eigentumsvermerk folgt dem Topf des Exemplars, am fertigen PDF beider Erzeuger: Ein
// Buch der Schülerbücherei, bezahlt vom Schulträger, trägt nicht den Aufdruck „Eigentum des
// Landes Hessen".
func TestEtikettEigentumsvermerkFolgtDemTopf(t *testing.T) {
	const land, stadt = "Eigentum des Landes Hessen", "Eigentum der Stadt Friedrichsdorf"

	faelle := []struct {
		name        string
		vermerkSB   string // Einstellung „Eigentumsvermerk Schülerbücherei"
		topf        string
		want, nicht string
	}{
		{"Lernmittel trägt den allgemeinen Vermerk", stadt, mitteltopf.Land, land, stadt},
		{"Schülerbücherei trägt ihren eigenen", stadt, mitteltopf.Schultraeger, stadt, land},
		{"Schülerbücherei ohne Einstellung: kein Vermerk", "", mitteltopf.Schultraeger, "", "Eigentum"},
		{"unbekanntes Exemplar (Vorab-Druck): wie bisher", stadt, "", land, stadt},
	}
	for _, f := range faelle {
		t.Run(f.name, func(t *testing.T) {
			kopf := buchetikett.KopfAus(&repository.SystemEinstellungen{
				SchuleName: "Testschule", EtikettEigentumsvermerkSchuelerbuecherei: f.vermerkSB})
			labels := einTestEtikett()
			labels[0].Topf = f.topf

			// Hauptlieferant + Land-Bestellung: beide Bögen entstehen, klein und groß.
			boegen, err := etikettenboegen(labels, kopf, true, mitteltopf.Land)
			if err != nil {
				t.Fatalf("etikettenboegen: %v", err)
			}
			if len(boegen) != 2 {
				t.Fatalf("erwartet kleinen und großen Bogen, bekam %d", len(boegen))
			}
			for _, b := range boegen {
				text := pdftest.Suchtext(t, b.Data)
				if f.want != "" && !strings.Contains(text, f.want) {
					t.Errorf("%s: Vermerk %q fehlt", b.Name, f.want)
				}
				if strings.Contains(text, f.nicht) {
					t.Errorf("%s: %q steht auf dem Etikett und gehört dort nicht hin", b.Name, f.nicht)
				}
			}
		})
	}
}
