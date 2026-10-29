package pdf

import (
	"fmt"
	"regexp"
	"strconv"
	"testing"

	"bibliothek/internal/pdftest"
	"bibliothek/pkg/strichcode"
)

// etikettStueck ist ein Text oder ein Bild auf der Seite, in mm von der linken oberen Ecke. Bei
// einem Text reicht das Rechteck von der Oberlänge bis zur Unterlänge seiner Schrift; seine
// rechte Kante kennt der Strom nicht, die Breite prüft TestBuchEtikett_KeinTextIstBreiterAlsSeineZelle.
type etikettStueck struct {
	text                       string
	bild                       bool
	links, oben, rechts, unten float64
}

// stueckImStrom findet im Inhaltsstrom von gofpdf die Wahl der Schrift (`/F1 8.00 Tf`), einen
// Text an seiner Grundlinie (`BT 18.14 794.99 Td (…)Tj ET`) und ein Bild mit seinem Rechteck
// (`q 34.0 0 0 34.0 61.1 754.9 cm /I1 Do Q`), in der Reihenfolge des Stroms. Die Zahlen sind
// Punkt, gezählt von der linken unteren Ecke.
var stueckImStrom = regexp.MustCompile(`/\w+\s+([0-9.]+)\s+Tf|BT ([0-9.]+) ([0-9.]+) Td \(((?:\\.|[^()\\])*)\)\s*Tj ET|q ([0-9.]+) 0 0 ([0-9.]+) ([0-9.]+) ([0-9.]+) cm /\w+ Do Q`)

const (
	ptJeMM    = 72 / 25.4
	a4HoeheMM = 297.0
	// Ober- und Unterlänge von Helvetica und Courier, gerundet nach oben, als Teil der Größe.
	oberlaenge  = 0.75
	unterlaenge = 0.22
)

// etikettStuecke liest Texte und Bilder der ersten Seite.
func etikettStuecke(t *testing.T, roh []byte) []etikettStueck {
	t.Helper()
	zahl := func(b []byte) float64 {
		wert, err := strconv.ParseFloat(string(b), 64)
		if err != nil {
			t.Fatalf("Zahl %q im Inhaltsstrom nicht lesbar: %v", b, err)
		}
		return wert / ptJeMM
	}
	var stuecke []etikettStueck
	var groesse float64
	for _, m := range stueckImStrom.FindAllSubmatch(pdftest.InhaltJeSeite(t, roh)[0], -1) {
		switch {
		case m[1] != nil:
			groesse = zahl(m[1])
		case m[2] != nil:
			links, grundlinie := zahl(m[2]), a4HoeheMM-zahl(m[3])
			stuecke = append(stuecke, etikettStueck{
				text: string(m[4]), links: links, rechts: links,
				oben: grundlinie - oberlaenge*groesse, unten: grundlinie + unterlaenge*groesse,
			})
		default:
			links, unten := zahl(m[7]), a4HoeheMM-zahl(m[8])
			stuecke = append(stuecke, etikettStueck{
				bild: true, links: links, rechts: links + zahl(m[5]),
				oben: unten - zahl(m[6]), unten: unten,
			})
		}
	}
	return stuecke
}

// Was ein Etikett druckt, steht auf ihm: Kein Text reicht näher als einen Millimeter an die Ober-
// oder Unterkante, kein Code näher an irgendeine Kante, damit ein leicht versetzt eingezogener
// Bogen nichts abschneidet, und nichts liegt übereinander. Was über die Unterkante läuft, steht
// auf dem Etikett darunter. Zur Seite darf ein Text die Zelle füllen, die das ganze Etikett breit
// ist; sonst stünde der Schulname auf dem schmalen Bogen gekürzt da.
func TestBuchEtikett_NichtsReichtUeberSeineKante(t *testing.T) {
	const rand = 1.0
	etikett := BuchEtikett{
		Schulname: "Philipp-Reis-Schule, Friedrichsdorf", Titel: "Seydlitz Geographie Gymnasium Hessen",
		AnschaffungsJahr: "2016", Signatur: "LMF Deu 7 / Bie", Eigentumsvermerk: "Eigentum des Landes Hessen",
	}
	for _, id := range etikettFormatReihenfolge {
		f := etikettFormate[id]
		// Das einzige Etikett des Bogens liegt im ersten Feld.
		links, oben := f.RandLinks, f.RandOben
		rechts, unten := links+f.Breite, oben+f.Hoehe
		for _, nummer := range etikettNummern {
			etikett.BarcodeID = nummer
			for _, alsQR := range []bool{false, true} {
				code := "Strichcode"
				if alsQR {
					code = "QR-Code"
				}
				t.Run(fmt.Sprintf("%s/%s/%d Zeichen", id, code, len(nummer)), func(t *testing.T) {
					stuecke := etikettStuecke(t, buchBogen(t, id, alsQR, etikett))
					var texte, bilder []etikettStueck
					for _, s := range stuecke {
						if s.bild {
							bilder = append(bilder, s)
						} else {
							texte = append(texte, s)
						}
					}
					// Ohne Code und ohne Titel und Nummer läse der Test am Strom vorbei. Eine
					// Nummer, die kein Code fasst, steht ohne ihn da.
					wantBilder := 1
					if _, err := strichcode.PNG(nummer, alsQR, 200, 200); err != nil {
						wantBilder = 0
					}
					if len(bilder) != wantBilder || len(texte) < 2 {
						t.Fatalf("%d Bilder und %d Texte gelesen, erwartet %d Bild und mindestens zwei Texte", len(bilder), len(texte), wantBilder)
					}
					for _, s := range stuecke {
						abstand := 0.0
						if s.bild {
							abstand = rand
						}
						if s.oben < oben+rand || s.unten > unten-rand || s.links < links+abstand || s.rechts > rechts-abstand {
							t.Errorf("%s reicht über die Kante: %.1f–%.1f mm (oben–unten), ab %.1f mm; das Etikett reicht von %.1f bis %.1f, ab %.1f",
								beschreibe(s), s.oben, s.unten, s.links, oben, unten, links)
						}
					}
					for i, a := range texte {
						for _, b := range texte[i+1:] {
							if a.oben < b.unten && b.oben < a.unten {
								t.Errorf("%s und %s liegen übereinander", beschreibe(a), beschreibe(b))
							}
						}
						// Neben dem Code darf ein Text stehen, auf ihm nicht.
						for _, b := range bilder {
							if a.oben < b.unten && b.oben < a.unten && a.links < b.rechts {
								t.Errorf("%s liegt auf dem Code", beschreibe(a))
							}
						}
					}
				})
			}
		}
	}
}

func beschreibe(s etikettStueck) string {
	if s.bild {
		return "der Code"
	}
	return fmt.Sprintf("%q", s.text)
}
