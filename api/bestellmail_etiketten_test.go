package api

// Eigentumsvermerk auf den gemailten Etikettenbögen (Sweep 01.09.2026, Fund P2):
// Bis dahin nagelte der Mailweg den Vermerk auf die Werksvorgabe „Eigentum des
// Landes Hessen" fest, während Selbstdruck (s.etikettKopf) und Lieferanten-Link
// die Einstellung etikett_eigentumsvermerk lasen — zwei Wege zum selben Buch,
// zwei verschiedene Aufkleber. Geprüft am fertigen PDF-Inhaltsstrom (pdfText),
// wie alle Etiketten-Gates.

import (
	"strings"
	"testing"

	"bibliothek/repository"
)

func testEtikettLabels() []BarcodeLabelDetail {
	return []BarcodeLabelDetail{{
		BarcodeID: "BM-TEST-1", Titel: "Bestellmail-Testband", Autor: "Prüfer", ISBN: "9783123456789",
	}}
}

func TestBestellmailEtikettenTragenKonfiguriertenVermerk(t *testing.T) {
	kopf := etikettKopfAus(&repository.SystemEinstellungen{
		SchuleName: "Testschule", EtikettEigentumsvermerk: "Eigentum des Kreises Wetterau"})
	boegen, err := etikettenboegen(testEtikettLabels(), kopf, false, repository.MittelLand)
	if err != nil {
		t.Fatalf("etikettenboegen: %v", err)
	}
	if len(boegen) == 0 {
		t.Fatal("kein Bogen erzeugt")
	}
	for _, b := range boegen {
		text := pdfText(t, b.Data)
		if !strings.Contains(text, "Eigentum des Kreises Wetterau") {
			t.Errorf("%s: konfigurierter Eigentumsvermerk fehlt auf dem Bogen", b.Name)
		}
		if strings.Contains(text, repository.StandardEigentumsvermerk) {
			t.Errorf("%s: Werksvorgabe steht trotz konfiguriertem Vermerk auf dem Bogen", b.Name)
		}
	}
}

func TestBestellmailEtikettenFallenOhneKonfigurationAufWerksvorgabe(t *testing.T) {
	kopf := etikettKopfAus(&repository.SystemEinstellungen{SchuleName: "Testschule"})
	boegen, err := etikettenboegen(testEtikettLabels(), kopf, false, repository.MittelLand)
	if err != nil {
		t.Fatalf("etikettenboegen: %v", err)
	}
	for _, b := range boegen {
		if !strings.Contains(pdfText(t, b.Data), repository.StandardEigentumsvermerk) {
			t.Errorf("%s: ohne Konfiguration muss die Werksvorgabe drucken", b.Name)
		}
	}
}

// Das große Lernmittel-Etikett („Eigentum des Landes") geht nur mit, wenn die Bestellung
// nicht der Schülerbücherei gilt (OFFEN.md 4.11, entschieden am 21.09.2026). Die
// Alt-Bestellung ohne Zuordnung behält beide Größen.
func TestBestellmailGrossesEtikettFolgtDemTopf(t *testing.T) {
	faelle := []struct {
		name              string
		istHauptlieferant bool
		mittel            string
		wantGross         bool
	}{
		{"Hauptlieferant, Lernmittelfreiheit", true, repository.MittelLand, true},
		{"Hauptlieferant, Schülerbücherei", true, repository.MittelSchultraeger, false},
		{"Hauptlieferant, ohne Zuordnung", true, "", true},
		{"anderer Lieferant, Lernmittelfreiheit", false, repository.MittelLand, false},
	}
	for _, f := range faelle {
		t.Run(f.name, func(t *testing.T) {
			kopf := etikettKopfAus(&repository.SystemEinstellungen{SchuleName: "Testschule"})
			boegen, err := etikettenboegen(testEtikettLabels(), kopf, f.istHauptlieferant, f.mittel)
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

// Der Eigentumsvermerk folgt dem Topf des Exemplars (OFFEN.md 4.21, entschieden am
// 21.09.2026) — am fertigen PDF beider Erzeuger. Bis dahin trug ein Buch der
// Schülerbücherei, bezahlt vom Schulträger, den Aufdruck „Eigentum des Landes Hessen".
func TestEtikettEigentumsvermerkFolgtDemTopf(t *testing.T) {
	const land, stadt = "Eigentum des Landes Hessen", "Eigentum der Stadt Friedrichsdorf"

	faelle := []struct {
		name        string
		vermerkSB   string // Einstellung „Eigentumsvermerk Schülerbücherei"
		topf        string
		want, nicht string
	}{
		{"Lernmittel trägt den allgemeinen Vermerk", stadt, repository.MittelLand, land, stadt},
		{"Schülerbücherei trägt ihren eigenen", stadt, repository.MittelSchultraeger, stadt, land},
		{"Schülerbücherei ohne Einstellung: kein Vermerk", "", repository.MittelSchultraeger, "", "Eigentum"},
		{"unbekanntes Exemplar (Vorab-Druck): wie bisher", stadt, "", land, stadt},
	}
	for _, f := range faelle {
		t.Run(f.name, func(t *testing.T) {
			kopf := etikettKopfAus(&repository.SystemEinstellungen{
				SchuleName: "Testschule", EtikettEigentumsvermerkSchuelerbuecherei: f.vermerkSB})
			labels := testEtikettLabels()
			labels[0].Topf = f.topf

			// Hauptlieferant + Land-Bestellung: beide Bögen entstehen, klein UND groß.
			boegen, err := etikettenboegen(labels, kopf, true, repository.MittelLand)
			if err != nil {
				t.Fatalf("etikettenboegen: %v", err)
			}
			if len(boegen) != 2 {
				t.Fatalf("erwartet kleinen und großen Bogen, bekam %d", len(boegen))
			}
			for _, b := range boegen {
				text := pdfText(t, b.Data)
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
