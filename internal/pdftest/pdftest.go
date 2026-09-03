// Package pdftest liest heraus, was auf einem erzeugten PDF wirklich GEDRUCKT wird.
//
// Der Umweg über das fertige PDF ist Absicht. Beide Etiketten-Bugs von 2026 waren an den
// Zwischenschichten grün: Das Feld stand im Struct, die Abfrage lieferte es — nur
// gezeichnet wurde es nicht. Erst der Inhaltsstrom beantwortet die Frage, die die
// Bibliothek stellt: Was steht auf dem Blatt?
//
// Wie internal/pgtest und internal/smtptest ist das ein normales Paket, das nur von Tests
// importiert wird: Testdateien (_test.go) lassen sich nicht paketübergreifend importieren.
// Es lag bis zum 03.09.2026 als Helfer in api/etiketten_pdf_paritaet_pg_test.go; der
// Schulbuch-Export (Paket inventur) braucht denselben Leser, und eine zweite Fassung wäre
// die Gelegenheit gewesen, die Fallen unten erneut hineinzubauen.
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
// `\s*` vor dem Tj, weil nicht jeder Erzeuger gleich schreibt: gofpdf setzt `)Tj`, maroto
// (Kontoauszug der Abgänger) `) Tj`. Ohne diese drei Zeichen fand der Leser in einem
// maroto-PDF NICHTS und meldete eine leere Seite — der Test hätte daraus geschlossen, auf
// dem Blatt stehe nichts, statt zu merken, dass er nur nicht hinsehen kann.
var tjText = regexp.MustCompile(`\(((?:\\.|[^()\\])*)\)\s*Tj`)

// Texte liest heraus, was auf einem erzeugten PDF wirklich GEDRUCKT wird.
//
// Der Umweg über das fertige PDF ist Absicht. Beide bisherigen Etiketten-Bugs waren an
// den Zwischenschichten grün: Das Feld stand im Struct, die Abfrage lieferte es — nur
// gezeichnet wurde es nicht bzw. kam auf dem zweiten Weg nie im Struct an. Erst der
// Inhaltsstrom beantwortet die Frage, die die Bibliothek stellt: Was klebt am Buch?
//
// gofpdf komprimiert die Ströme meistens (FlateDecode); im Rohbyte-PDF steht der Text
// dann nicht. Nach dem Inflaten liegt er als `(…)Tj` vor. Sonderzeichen bleiben in der
// Kodierung des Erzeugers (Windows-1252) — für den Vergleich zweier Bögen und die Suche
// nach ASCII-Signaturen ist das gleichgültig, beide Seiten werden gleich behandelt.
//
// „Meistens" ist der Grund für den Rückfall auf die Rohbytes: Der Kontoauszug der
// Abgänger kommt UNKOMPRIMIERT aus dem Generator. Ohne diesen Zweig war der Leser für ihn
// blind und brach mit „kein lesbarer Inhaltsstrom" ab — was nach kaputtem PDF aussieht,
// aber nur hiess, dass der Leser eine Bauform nicht kannte.
func Texte(t *testing.T, roh []byte) []string {
	t.Helper()

	var inhalt bytes.Buffer
	rest := roh
	for {
		i := bytes.Index(rest, []byte("stream"))
		if i < 0 {
			break
		}
		body := bytes.TrimLeft(rest[i+len("stream"):], "\r\n")
		j := bytes.Index(body, []byte("endstream"))
		if j < 0 {
			break
		}
		// Nicht jeder Strom ist ein komprimierter Inhaltsstrom (Schriften, Bilder) — was
		// zlib nicht öffnet, wird übersprungen. Was es öffnet, muss aber vollständig
		// lesbar sein, sonst prüfte der Test nur den Teil, den er zufällig entpacken konnte.
		zr, err := zlib.NewReader(bytes.NewReader(body[:j]))
		if err == nil {
			entpackt, leseErr := io.ReadAll(zr)
			if leseErr != nil {
				t.Fatalf("Inhaltsstrom nicht vollständig lesbar: %v", leseErr)
			}
			inhalt.Write(entpackt)
			if closeErr := zr.Close(); closeErr != nil {
				t.Fatalf("Inhaltsstrom nicht sauber abgeschlossen: %v", closeErr)
			}
		} else {
			// Unkomprimiert (oder Schrift/Bild): roh mitnehmen. Der Tj-Ausdruck unten
			// entscheidet, ob etwas Gedrucktes darin steht — Binärmüll trifft er nicht.
			inhalt.Write(body[:j])
		}
		// Hinter das "endstream", nicht davor. Mit rest = body[j:] fand der nächste
		// Durchlauf das "stream" IN "endstream" wieder, las ab dort Unsinn, den zlib
		// verwarf — und übersprang dabei den echten nächsten Strom. Gelesen wurde damit
		// immer nur die ERSTE Seite: Ein Bogen mit einem Etikett und ein Bogen mit
		// hundert sahen für diesen Test gleich aus.
		rest = body[j+len("endstream"):]
	}
	if inhalt.Len() == 0 {
		t.Fatalf("kein lesbarer Inhaltsstrom im PDF (%d Bytes) — Textextraktion kaputt, "+
			"der Test würde ab hier alles durchwinken", len(roh))
	}

	var texte []string
	for _, treffer := range tjText.FindAllSubmatch(inhalt.Bytes(), -1) {
		if s := strings.TrimSpace(nachUTF8(treffer[1])); s != "" {
			texte = append(texte, s)
		}
	}
	sort.Strings(texte)
	return texte
}

// nachUTF8 macht aus einem PDF-Textstück lesbares Go: erst die PDF-Escapes auflösen,
// dann von Windows-1252 nach UTF-8 wandeln.
//
// Ohne diesen Schritt ist jede Erwartung mit Umlaut STILL FALSCH-GRÜN. Genau das passierte
// am 03.09.2026: Eine Gegenprobe verlangte, dass die Spalte „Gezählt" NICHT gedruckt wird —
// sie hätte auch dann nicht angeschlagen, wenn die Spalte da gewesen wäre, denn im
// Inhaltsstrom steht „Gez\xe4hlt", und das trifft kein UTF-8-Literal. Der Test sah aus wie
// ein Gate und war eine Zusicherung, die nie zuschlagen konnte.
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
			// Oktal-Escape (\344 = ä in Windows-1252) — bis zu drei Ziffern.
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
	// Fehler wären nur bei defekten Eingaben möglich; dann zählt der Rohtext mehr als nichts.
	if utf8Text, err := charmap.Windows1252.NewDecoder().Bytes(b); err == nil {
		return string(utf8Text)
	}
	return string(b)
}

// IstPDF prüft Signatur und Abschluss-Marker. Ein abgeschnittenes Dokument beginnt zwar
// mit %PDF, hat aber kein %%EOF — genau so sähe ein Ausgabefehler aus.
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
