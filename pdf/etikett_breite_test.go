package pdf

import (
	"bytes"
	"strings"
	"testing"

	"bibliothek/internal/pdftest"
)

// etikettZeile ist eine gekürzte Zeile eines Etiketts mit dem, was der Test über sie weiß, ohne
// den Erzeuger zu fragen: der ungekürzte Text, die Schrift, die Breite der Zelle und die
// Obergrenze in Zeichen.
type etikettZeile struct {
	name    string
	quelle  string
	fett    bool
	groesse float64
	zelle   float64
	zeichen int
}

// breitesEtikett trägt in jedem Feld den breitesten Text, den die Schrift hergibt. Jedes Feld
// beginnt mit einem eigenen Wort, damit der Test seine Zeile findet und ein vertauschtes Feld
// auffällt.
var breitesEtikett = BuchEtikett{
	Schulname:        "Schule " + strings.Repeat("W", 60),
	BarcodeID:        "1000000000001",
	Titel:            "Titel " + strings.Repeat("M", 60),
	Autor:            "Autor " + strings.Repeat("W", 40),
	AnschaffungsJahr: "2026",
	Signatur:         "Signatur " + strings.Repeat("W", 60),
	Eigentumsvermerk: "Vermerk " + strings.Repeat("M", 60),
}

// aufZeichen kürzt wie die Obergrenze der Zeile: höchstens grenze Zeichen, das Auslassungszeichen
// eingerechnet.
func aufZeichen(s string, grenze int) string {
	r := []rune(s)
	if len(r) <= grenze {
		return s
	}
	return string(r[:grenze-1]) + "…"
}

func buchBogen(t *testing.T, format string, qr bool, etiketten ...BuchEtikett) []byte {
	t.Helper()
	dokument, err := ErzeugeBuchEtiketten(format, 1, qr, etiketten)
	if err != nil {
		t.Fatalf("Bogen %s: %v", format, err)
	}
	var buf bytes.Buffer
	if err := dokument.Output(&buf); err != nil {
		t.Fatalf("Bogen %s ausgeben: %v", format, err)
	}
	return buf.Bytes()
}

// pruefeEtikettZeilen sucht jede Zeile am ersten Wort ihrer Quelle und hält sie gegen ihre
// Zelle: nicht breiter, nicht länger als die Obergrenze und nicht mehr gekürzt als nötig.
func pruefeEtikettZeilen(t *testing.T, roh []byte, zeilen []etikettZeile) {
	t.Helper()
	for _, z := range zeilen {
		wort, _, _ := strings.Cut(z.quelle, " ")
		gedruckt := gedruckteZeilen(t, roh, wort)
		if len(gedruckt) != 1 {
			t.Errorf("%s: %d gedruckte Zeilen mit %q, erwartet eine: %q", z.name, len(gedruckt), wort, gedruckt)
			continue
		}
		t.Run(z.name, func(t *testing.T) {
			if z.fett {
				pdftest.InSpalteFett(t, gedruckt[0], aufZeichen(z.quelle, z.zeichen), z.groesse, z.zelle)
				return
			}
			pdftest.InSpalte(t, gedruckt[0], aufZeichen(z.quelle, z.zeichen), z.groesse, z.zelle)
		})
	}
}

// Kein Text des Buchetiketts ist breiter als seine Zelle. Die Zelle reicht über das ganze
// Etikett; was übersteht, steht auf dem Nachbaretikett.
func TestBuchEtikett_KeinTextIstBreiterAlsSeineZelle(t *testing.T) {
	e := breitesEtikett
	if len(etikettFormatReihenfolge) == 0 {
		t.Fatal("kein Bogenformat in etikettFormatReihenfolge: der Test prüfte nichts")
	}
	for _, id := range etikettFormatReihenfolge {
		f := etikettFormate[id]
		t.Run(id+"/Strichcode", func(t *testing.T) {
			zeilen := []etikettZeile{
				{name: "Schulname", quelle: e.Schulname, fett: true, groesse: 8, zelle: f.Breite, zeichen: 42},
				{name: "Titel", quelle: e.Titel, fett: true, groesse: 8, zelle: f.Breite, zeichen: 40},
			}
			// Unter 30 mm Höhe druckt das Etikett weder Jahr und Signatur noch den Vermerk.
			if f.Hoehe >= 30 {
				zeilen = append(zeilen,
					etikettZeile{name: "Jahr und Signatur", quelle: zweiteZeile(e.AnschaffungsJahr, e.Signatur), groesse: 7, zelle: f.Breite, zeichen: 45},
					etikettZeile{name: "Eigentumsvermerk", quelle: e.Eigentumsvermerk, groesse: 7, zelle: f.Breite, zeichen: 45},
				)
			}
			pruefeEtikettZeilen(t, buchBogen(t, id, false, e), zeilen)
		})
		t.Run(id+"/QR-Code", func(t *testing.T) {
			pruefeEtikettZeilen(t, buchBogen(t, id, true, e), []etikettZeile{
				{name: "Titel", quelle: e.Titel, fett: true, groesse: 8, zelle: f.Breite - 4, zeichen: 40},
				{name: "Autor", quelle: e.Autor, groesse: 7, zelle: f.Breite - 4, zeichen: 30},
			})
		})
	}
}

// Dasselbe am großen Lernmittel-Etikett: Die Zelle ist das Feld ohne seinen Rand.
func TestLernmittelEtikett_KeinTextIstBreiterAlsSeineZelle(t *testing.T) {
	e := breitesEtikett
	roh, err := ErzeugeLernmittelEtiketten([]BuchEtikett{e})
	if err != nil {
		t.Fatalf("Lernmittel-Etikett: %v", err)
	}
	pruefeEtikettZeilen(t, roh, []etikettZeile{
		{name: "Schulname", quelle: e.Schulname, fett: true, groesse: 11, zelle: lernmittelInhalt, zeichen: 45},
		{name: "Titel", quelle: e.Titel, fett: true, groesse: 12, zelle: lernmittelInhalt, zeichen: 42},
		{name: "Jahr und Signatur", quelle: zweiteZeile(e.AnschaffungsJahr, e.Signatur), groesse: 9, zelle: lernmittelInhalt, zeichen: 48},
		{name: "Eigentumsvermerk", quelle: e.Eigentumsvermerk, groesse: 8, zelle: lernmittelInhalt, zeichen: 48},
	})
}

// Was in seine Zelle passt, bleibt ungekürzt: der Schulname auch auf dem schmalen Bogen, wo er
// die Zelle fast füllt, und ein langer Titel bis zur Obergrenze in Zeichen. Gekürzt wird nur,
// was sonst über den Rand liefe.
func TestBuchEtikett_WasPasstBleibtWieEsIst(t *testing.T) {
	const schule = "Philipp-Reis-Schule, Friedrichsdorf"
	e := BuchEtikett{
		Schulname: schule, BarcodeID: "1000000000002",
		Titel: "Mathematik heute – Wahrscheinlichkeitsrechnung und Statistik",
	}
	faelle := []struct {
		format string
		titel  string
	}{
		{"zweckform_l4760", "Mathematik heute – Wahrscheinlichkeitsr…"},
		{"avery_3475", "Mathematik heute – Wahrscheinlichkeitsr…"},
		{"standard_52", "Mathematik heute – Wahrscheinli…"},
	}
	for _, f := range faelle {
		roh := buchBogen(t, f.format, false, e)
		if ist := gedruckteZeilen(t, roh, "Schule"); len(ist) != 1 || ist[0] != schule {
			t.Errorf("%s: gedruckter Schulname %q, erwartet %q", f.format, ist, schule)
		}
		if ist := gedruckteZeilen(t, roh, "Mathematik"); len(ist) != 1 || ist[0] != f.titel {
			t.Errorf("%s: gedruckter Titel %q, erwartet %q", f.format, ist, f.titel)
		}
	}
}
