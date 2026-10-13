// Package pdftest liest aus einem erzeugten PDF heraus, was auf dem Blatt gedruckt wird.
//
// Geprüft wird am fertigen PDF, weil die Schichten davor einen Fehler im Druck nicht zeigen:
// Ein Feld kann im Struct stehen und von der Abfrage kommen, ohne gezeichnet zu werden. Erst
// der Inhaltsstrom sagt, was auf dem Blatt steht.
//
// Wie internal/pgtest und internal/smtptest ist es ein gewöhnliches Paket, das nur Testdateien
// einbinden. Die Etiketten und die Lernmittelliste lesen ihr PDF mit demselben Leser, damit
// die Fallen der Bauform an einer Stelle behandelt sind.
package pdftest

import (
	"bytes"
	"compress/zlib"
	"io"
	"regexp"
	"sort"
	"strings"
	"testing"

	"golang.org/x/text/encoding/charmap"
)

// tjText findet die gezeichneten Textstücke im Inhaltsstrom: `(Ansch.J. 2016 · LMF-Deutsch 5)Tj`.
//
// `\s*` steht vor dem Tj, weil nicht jeder Erzeuger gleich schreibt: gofpdf setzt `)Tj`, maroto
// (Kontoauszug der Abgänger) `) Tj`. Ohne es fände der Leser in einem PDF von maroto nichts
// und meldete eine leere Seite.
var tjText = regexp.MustCompile(`\(((?:\\.|[^()\\])*)\)\s*Tj`)

// Texte liefert die gedruckten Textstücke des Dokuments, sortiert.
//
// gofpdf packt die Ströme meist (FlateDecode); im rohen PDF steht der Text dann nicht, nach
// dem Entpacken liegt er als `(…)Tj` vor. Sonderzeichen bleiben in der Kodierung des Erzeugers
// (Windows-1252), bis nachUTF8 sie wandelt.
//
// Ein Strom, der sich nicht entpacken lässt, wird roh gelesen: Der Kontoauszug der Abgänger
// kommt ungepackt aus dem Erzeuger, und ohne diesen Weg bräche der Leser bei ihm mit „kein
// lesbarer Inhaltsstrom" ab.
func Texte(t *testing.T, roh []byte) []string {
	t.Helper()
	_, texte := lies(t, roh)
	return texte
}

// TexteJeSeite liefert die Textstücke getrennt nach Inhaltsstrom; bei gofpdf ist das je Seite
// einer.
//
// Damit lässt sich ein Seitenumbruch mitten in einem Block prüfen, den der Text des ganzen
// Dokuments nicht zeigt: Steht „Name des/der" unten auf Seite 1 und „Schülers/Schülerin" oben
// auf Seite 2, ist im Gesamttext beides vorhanden.
func TexteJeSeite(t *testing.T, roh []byte) [][]string {
	t.Helper()
	stroeme := stroemeAus(t, roh)
	seiten := make([][]string, 0, len(stroeme))
	for _, strom := range stroeme {
		texte := texteAus(strom)
		// Ströme ohne Gedrucktes (Schriften, Bilder) sind keine Seiten.
		if len(texte) > 0 {
			seiten = append(seiten, texte)
		}
	}
	if len(seiten) == 0 {
		t.Fatalf("keine Seite mit Text im PDF (%d Bytes) — Textextraktion kaputt", len(roh))
	}
	return seiten
}

// InhaltJeSeite liefert die entpackten Inhaltsströme getrennt, je Seite einen. Für Prüfungen am
// Ort von Text und Bild: Koordinaten gelten je Seite.
func InhaltJeSeite(t *testing.T, roh []byte) [][]byte {
	t.Helper()
	var seiten [][]byte
	for _, strom := range stroemeAus(t, roh) {
		// Ströme ohne Gedrucktes (Schriften, Bilder) sind keine Seiten.
		if len(texteAus(strom)) > 0 {
			seiten = append(seiten, strom)
		}
	}
	if len(seiten) == 0 {
		t.Fatalf("keine Seite mit Text im PDF (%d Bytes) — Textextraktion kaputt", len(roh))
	}
	return seiten
}

// lies entpackt alle Inhaltsströme des Dokuments und liefert sie roh und als Textstücke.
func lies(t *testing.T, roh []byte) ([]byte, []string) {
	t.Helper()

	var inhalt bytes.Buffer
	for _, strom := range stroemeAus(t, roh) {
		inhalt.Write(strom)
	}
	if inhalt.Len() == 0 {
		t.Fatalf("kein lesbarer Inhaltsstrom im PDF (%d Bytes) — Textextraktion kaputt, "+
			"der Test würde ab hier alles durchwinken", len(roh))
	}
	return inhalt.Bytes(), texteAus(inhalt.Bytes())
}

// stroemeAus entpackt die Inhaltsströme einzeln, gepackte wie ungepackte.
func stroemeAus(t *testing.T, roh []byte) [][]byte {
	t.Helper()

	var stroeme [][]byte
	rest := roh
	for {
		i := bytes.Index(rest, []byte("stream"))
		if i < 0 {
			break
		}
		rumpf := bytes.TrimLeft(rest[i+len("stream"):], "\r\n")
		j := bytes.Index(rumpf, []byte("endstream"))
		if j < 0 {
			break
		}
		// Nicht jeder Strom ist ein gepackter Inhaltsstrom (Schriften, Bilder); was zlib nicht
		// öffnet, gilt als ungepackt. Was zlib öffnet, muss ganz lesbar sein; sonst prüfte der
		// Test nur den Teil, den er entpacken konnte.
		entpacker, err := zlib.NewReader(bytes.NewReader(rumpf[:j]))
		if err == nil {
			entpackt, leseFehler := io.ReadAll(entpacker)
			if leseFehler != nil {
				t.Fatalf("Inhaltsstrom nicht vollständig lesbar: %v", leseFehler)
			}
			stroeme = append(stroeme, entpackt)
			if schliessFehler := entpacker.Close(); schliessFehler != nil {
				t.Fatalf("Inhaltsstrom nicht sauber abgeschlossen: %v", schliessFehler)
			}
		} else {
			// Ungepackt, oder Schrift oder Bild: roh mitnehmen. Ob etwas Gedrucktes darin
			// steht, entscheidet tjText; Binärdaten trifft der Ausdruck nicht.
			stroeme = append(stroeme, rumpf[:j])
		}
		// Weiter hinter dem "endstream" und nicht davor: Sonst fände der nächste Durchlauf
		// das "stream" in "endstream", läse ab dort Unsinn und überspränge den nächsten
		// Strom. Gelesen würde dann nur die erste Seite.
		rest = rumpf[j+len("endstream"):]
	}
	return stroeme
}

// Inhalt liefert den entpackten Inhaltsstrom roh: für Prüfungen, die nicht am Text hängen,
// etwa am Ort eingebetteter Bilder.
func Inhalt(t *testing.T, roh []byte) []byte {
	t.Helper()
	strom, _ := lies(t, roh)
	return strom
}

func texteAus(inhalt []byte) []string {
	texte := texteInReihenfolge(inhalt)
	sort.Strings(texte)
	return texte
}

func texteInReihenfolge(inhalt []byte) []string {
	var texte []string
	for _, treffer := range tjText.FindAllSubmatch(inhalt, -1) {
		if s := strings.TrimSpace(nachUTF8(treffer[1])); s != "" {
			texte = append(texte, s)
		}
	}
	return texte
}

// TexteInReihenfolge liefert die Textstücke so, wie der Erzeuger sie setzt: Seite für Seite, von
// oben nach unten. Für Prüfungen, die einen Abschnitt des Blatts meinen. Texte sortiert sie und
// taugt dafür nicht.
func TexteInReihenfolge(t *testing.T, roh []byte) []string {
	t.Helper()
	inhalt, _ := lies(t, roh)
	return texteInReihenfolge(inhalt)
}

// nachUTF8 macht aus einem Textstück des PDF lesbaren Text: Es löst die Escapes des PDF auf
// und wandelt von Windows-1252 nach UTF-8.
//
// Ohne diesen Schritt träfe keine Erwartung mit Umlaut: Im Inhaltsstrom steht „Gez\xe4hlt",
// und ein Test, der verlangt, dass „Gezählt" nicht gedruckt wird, bliebe auch dann grün, wenn
// es dasteht.
func nachUTF8(roh []byte) string {
	var b []byte
	for i := 0; i < len(roh); i++ {
		c := roh[i]
		if c != '\\' || i+1 >= len(roh) {
			b = append(b, c)
			continue
		}
		i++
		switch n := roh[i]; n {
		case 'n':
			b = append(b, '\n')
		case 'r':
			b = append(b, '\r')
		case 't':
			b = append(b, '\t')
		case '(', ')', '\\':
			b = append(b, n)
		default:
			// Ein Escape in Oktal (\344 ist ä in Windows-1252) hat bis zu drei Ziffern.
			if n < '0' || n > '7' {
				b = append(b, n)
				break
			}
			wert := int(n - '0')
			for k := 0; k < 2 && i+1 < len(roh) && roh[i+1] >= '0' && roh[i+1] <= '7'; k++ {
				i++
				wert = wert*8 + int(roh[i]-'0')
			}
			b = append(b, byte(wert))
		}
	}
	// Ein Fehler ist nur bei kaputter Eingabe möglich; dann ist der rohe Text besser als
	// keiner.
	if text, err := charmap.Windows1252.NewDecoder().Bytes(b); err == nil {
		return string(text)
	}
	return string(b)
}

// IstPDF prüft Signatur und Abschluss. Ein abgeschnittenes Dokument beginnt mit %PDF, hat
// aber kein %%EOF; so sähe ein Fehler bei der Ausgabe aus.
func IstPDF(t *testing.T, b []byte, was string) {
	t.Helper()
	if len(b) == 0 {
		t.Fatalf("%s: leeres Dokument", was)
	}
	if !bytes.HasPrefix(b, []byte("%PDF-")) {
		t.Fatalf("%s: fehlende PDF-Signatur, Anfang: %q", was, string(b[:min(16, len(b))]))
	}
	if !bytes.Contains(b[max(0, len(b)-1024):], []byte("%%EOF")) {
		t.Fatalf("%s: kein %%%%EOF am Ende — Dokument abgeschnitten (%d Bytes)", was, len(b))
	}
}
