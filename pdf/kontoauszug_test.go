package pdf

import (
	"slices"
	"testing"
	"time"

	"bibliothek/internal/pdftest"
	"bibliothek/pkg/schulzeit"
)

// TestKontoauszug_JedeSeiteNenntDieSchule: Oben steht die Schule mit Anschrift wie auf
// Bestandsbuch und Bestellbericht, auf dem Auszug zum Lesen und auf jeder Seite des Laufzettels
// der Abgänger.
func TestKontoauszug_JedeSeiteNenntDieSchule(t *testing.T) {
	eintraege := []KontoauszugEintrag{
		{Schueler: KontoauszugSchueler{Vorname: "Ali", Nachname: "Öztürk", Klasse: "10a"},
			Buecher: []KontoauszugBuch{{Titel: "Rest", Barcode: "B-9", Ausleihdatum: time.Now(), Rueckgabedatum: time.Now()}}},
		{Schueler: KontoauszugSchueler{Vorname: "Bea", Nachname: "Schäfer", Klasse: "10b"}},
	}
	schule := testSchule()

	for _, mitUnterschrift := range []bool{false, true} {
		roh, err := ErzeugeKontoauszuege(eintraege, schule, mitUnterschrift)
		if err != nil {
			t.Fatalf("ErzeugeKontoauszuege(mitUnterschrift=%v): %v", mitUnterschrift, err)
		}
		seiten := pdftest.TexteJeSeite(t, roh)
		if len(seiten) != len(eintraege) {
			t.Fatalf("ErzeugeKontoauszuege(mitUnterschrift=%v) druckt %d Seiten, erwartet %d", mitUnterschrift, len(seiten), len(eintraege))
		}
		for i, texte := range seiten {
			for _, kopf := range []string{schule.Name, schule.Absenderzeile()} {
				if !slices.Contains(texte, kopf) {
					t.Errorf("ErzeugeKontoauszuege(mitUnterschrift=%v): Seite %d ohne %q", mitUnterschrift, i+1, kopf)
				}
			}
		}
	}
}

// TestKontoauszug_KennzeichnetUeberfaelliges: Ist die Frist vorbei, steht hinter dem Datum
// „überfällig", und der Satz unter der Liste bittet um die Rückgabe, statt vor Mahnungen zu
// warnen. Ein Wort und keine Farbe, weil die Schule schwarz-weiß druckt.
func TestKontoauszug_KennzeichnetUeberfaelliges(t *testing.T) {
	jetzt := schulzeit.Jetzt()
	abgelaufen := jetzt.AddDate(0, 0, -7)
	laufend := jetzt.AddDate(0, 1, 0)
	datum := func(z time.Time) string { return z.In(schulzeit.Zone()).Format(datumsformat) }
	buch := func(titel string, frist time.Time) KontoauszugBuch {
		return KontoauszugBuch{Titel: titel, Barcode: "105785", Ausleihdatum: frist.AddDate(0, 0, -21), Rueckgabedatum: frist}
	}
	schueler := KontoauszugSchueler{Vorname: "Max", Nachname: "Mustermann", Klasse: "05F"}

	faelle := []struct {
		name     string
		buecher  []KontoauszugBuch
		gedruckt []string
		fehlt    []string
	}{
		{
			name:     "eines überfällig",
			buecher:  []KontoauszugBuch{buch("Fokus Physik 8", abgelaufen), buch("Zeitreise Band 1", laufend)},
			gedruckt: []string{datum(abgelaufen) + " überfällig", datum(laufend), "Ein Buch ist überfällig. Bitte gib es so bald wie möglich zurück."},
			fehlt:    []string{datum(laufend) + " überfällig", "Bitte beachte die Rückgabefristen, um Mahnungen zu vermeiden."},
		},
		{
			name:     "zwei überfällig",
			buecher:  []KontoauszugBuch{buch("Fokus Physik 8", abgelaufen), buch("Fokus Physik 8", abgelaufen)},
			gedruckt: []string{datum(abgelaufen) + " überfällig", "2 Bücher sind überfällig. Bitte gib sie so bald wie möglich zurück."},
		},
		{
			name:     "nichts überfällig",
			buecher:  []KontoauszugBuch{buch("Zeitreise Band 1", laufend)},
			gedruckt: []string{datum(laufend), "Bitte beachte die Rückgabefristen, um Mahnungen zu vermeiden."},
			fehlt:    []string{datum(laufend) + " überfällig"},
		},
	}
	for _, f := range faelle {
		t.Run(f.name, func(t *testing.T) {
			roh, err := ErzeugeKontoauszug(schueler, f.buecher, testSchule())
			if err != nil {
				t.Fatalf("ErzeugeKontoauszug: %v", err)
			}
			texte := pdftest.Texte(t, roh)
			for _, s := range f.gedruckt {
				if !slices.Contains(texte, s) {
					t.Errorf("ErzeugeKontoauszug druckt %q nicht, gedruckt ist %q", s, texte)
				}
			}
			for _, s := range f.fehlt {
				if slices.Contains(texte, s) {
					t.Errorf("ErzeugeKontoauszug druckt %q", s)
				}
			}
		})
	}
}

// TestKontoauszug_SpaltenFassenIhrenText: Ein Text, der nicht in seine Spalte passt, bricht maroto
// an der Lücke um, und die zweite Zeile läge über der nächsten Ausleihe. Gemessen an der längsten
// Rückgabe (fett, mit „überfällig") und an einer 13-stelligen Nummer, wie sie Littera vergab;
// die steht mittig und behält zu jeder Seite mindestens 4 mm bis zum Text daneben.
func TestKontoauszug_SpaltenFassenIhrenText(t *testing.T) {
	teil := (210.0 - 2*kontoauszugRand) / kontoauszugRaster
	pdftest.InSpalteFett(t, "22.10.2026 überfällig", "22.10.2026 überfällig", 10, kontoauszugSpalteRueckgabe*teil)
	pdftest.InSpalte(t, "4012345678901", "4012345678901", 10, kontoauszugSpalteBarcode*teil-2*4)
	if summe := kontoauszugSpalteTitel + kontoauszugSpalteBarcode + kontoauszugSpalteAusgeliehen + kontoauszugSpalteRueckgabe; summe != kontoauszugRaster {
		t.Errorf("Spalten des Kontoauszugs ergeben %d Teile, erwartet %d", summe, kontoauszugRaster)
	}
}
