package api

import (
	"bytes"
	"testing"
	"time"

	"bibliothek/pdf"
)

// TestGenerateDsgvoAuskunftPDF stellt sicher, dass der Auskunfts-Generator aus
// vollständigen Daten ein valides, nicht-triviales PDF erzeugt — inkl. gefüllter
// Listen (die MultiCell-Umbrüche und optionale Zeiger-Felder auslösen).
func TestGenerateDsgvoAuskunftPDF(t *testing.T) {
	gebdatum := "2010-05-01"
	lusd := "LUSD-4711"
	sperrgrund := "Test-Sperrgrund"
	rueckgabe := time.Date(2025, 3, 1, 0, 0, 0, 0, time.UTC)
	notiz := "bitte zurücklegen"
	kontext := "Import"

	daten := &dsgvoDaten{
		stammdaten: &DsgvoStammdaten{
			ID: "abc-123", BarcodeID: "S-10001", Vorname: "Erika", Nachname: "Mustermann",
			Klasse: "7b", Geburtsdatum: &gebdatum, AbgaengerJahr: 2030, LusdID: &lusd,
			Strasse: "Hauptstr.", Hausnummer: "5", Plz: "60311", Ort: "Frankfurt",
			ElternEmail: "eltern@example.org", IstGesperrt: true, IsManuallyBlocked: true,
			BlockReason: &sperrgrund, ErstelltAm: time.Now(), AktualisiertAm: time.Now(),
		},
		foto:      DsgvoFoto{Vorhanden: true, AktualisiertAm: &rueckgabe, Hinweis: "verschlüsselt"},
		ausleihen: []DsgvoAusleihe{{Gegenstand: "Mathebuch 7", Barcode: "B-500", AusgeliehenAm: time.Now(), RueckgabeFrist: time.Now(), RueckgabeAm: &rueckgabe}},
		schaeden:  []DsgvoSchadensfall{{Beschreibung: "Wasserschaden", Betrag: "12.50", IstBezahlt: false, ErstelltAm: time.Now()}},
		vormerkungen: []DsgvoVormerkung{{Titel: "Deutschbuch 7", Status: "wartend", Notiz: &notiz, ErstelltAm: time.Now()}},
		auditEintraege: []DsgvoAuditEintrag{{Aktion: "update", Akteur: "USER", Zeitpunkt: time.Now(), Kontext: &kontext}},
	}

	out, err := generateDsgvoAuskunftPDF(daten, pdf.SchuleInfo{Name: "Testschule", Ort: "Frankfurt"})
	if err != nil {
		t.Fatalf("generateDsgvoAuskunftPDF: %v", err)
	}
	if !bytes.HasPrefix(out, []byte("%PDF")) {
		t.Errorf("Ausgabe ist kein PDF (Prefix %q)", out[:min(8, len(out))])
	}
	if len(out) < 1500 {
		t.Errorf("PDF verdächtig klein: %d Bytes", len(out))
	}
}
