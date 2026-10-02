package inventur

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"testing"
)

// Der Titel aus der DNB ist der Sachtitel (245 $a mit Zählung und Teil). Die Verfasserangabe
// dahinter ($c) gehört nicht hinein; fehlt ein Verfasser in 100 und 700, steht sie als Autor.
// Um den Artikel am Anfang setzt die DNB zwei Nichtsortierzeichen (U+0098, U+009C), die nicht
// zu sehen sind und jeden Vergleich mit dem getippten Titel scheitern lassen. Die Felder sind
// gekürzte Antworten der DNB.
func TestSucheDNB_TitelOhneVerfasserangabeUndNichtsortierzeichen(t *testing.T) {
	faelle := []struct {
		name, felder, titel, autor string
	}{
		{
			name: "Verfasser in 100: die Verfasserangabe entfällt",
			felder: `<datafield tag="100" ind1="1" ind2=" "><subfield code="a">Rowling, J. K.</subfield></datafield>
				<datafield tag="245" ind1="1" ind2="0"><subfield code="a">Harry Potter und der Stein der Weisen</subfield><subfield code="c">Joanne K. Rowling. Aus dem Engl. von Klaus Fritz</subfield></datafield>`,
			titel: "Harry Potter und der Stein der Weisen",
			autor: "Rowling, J. K.",
		},
		{
			name: "Artikel in Nichtsortierzeichen",
			felder: `<datafield tag="100" ind1="1" ind2=" "><subfield code="a">Tolkien, J. R. R.</subfield></datafield>
				<datafield tag="245" ind1="1" ind2="0"><subfield code="a">&#152;Der&#156; kleine Hobbit</subfield><subfield code="c">John Ronald R. Tolkien. Aus dem Engl. von Walter Scherf</subfield></datafield>`,
			titel: "Der kleine Hobbit",
			autor: "Tolkien, J. R. R.",
		},
		{
			name:   "kein Verfasser in 100 und 700: die Verfasserangabe ist der Autor",
			felder: `<datafield tag="245" ind1="0" ind2="0"><subfield code="a">Deutschbuch</subfield><subfield code="p">Gymnasium</subfield><subfield code="n">5.</subfield><subfield code="c">hrsg. von Markus Langner ...</subfield></datafield>`,
			titel:  "Deutschbuch Gymnasium 5.",
			autor:  "hrsg. von Markus Langner ...",
		},
		{
			name: "Verfasser erst in 700: die Verfasserangabe entfällt",
			felder: `<datafield tag="245" ind1="1" ind2="0"><subfield code="a">Tintenherz</subfield><subfield code="c">Cornelia Funke. Mit Ill. der Autorin</subfield></datafield>
				<datafield tag="700" ind1="1" ind2=" "><subfield code="a">Funke, Cornelia</subfield></datafield>`,
			titel: "Tintenherz",
			autor: "Funke, Cornelia",
		},
		{
			name:   "ältere Sätze trennen die Verfasser mit Schrägstrich ab",
			felder: `<datafield tag="245" ind1="0" ind2="0"><subfield code="a">Green line Oberstufe</subfield><subfield code="p">[Schülerbuch] / von Dr. Daniel Becker</subfield></datafield>`,
			titel:  "Green line Oberstufe [Schülerbuch]",
			autor:  "von Dr. Daniel Becker",
		},
	}
	for _, f := range faelle {
		t.Run(f.name, func(t *testing.T) {
			client := NeuerMetadatenClient()
			client.SetzeHTTPClientFuerTest(&http.Client{Transport: &mockTransport{
				roundTripFunc: func(*http.Request) (*http.Response, error) {
					koerper := `<searchRetrieveResponse xmlns="http://www.loc.gov/zing/srw/"><records><record><recordData>
						<record xmlns="http://www.loc.gov/MARC21/slim">` + f.felder + `</record>
						</recordData></record></records></searchRetrieveResponse>`
					return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(bytes.NewBufferString(koerper)), Header: make(http.Header)}, nil
				},
			}})

			erg, err := client.sucheDNB(context.Background(), "9783423715669")
			if err != nil {
				t.Fatalf("sucheDNB: %v", err)
			}
			if erg.Titel != f.titel {
				t.Errorf("Titel %q, erwartet %q", erg.Titel, f.titel)
			}
			if erg.Autor != f.autor {
				t.Errorf("Autor %q, erwartet %q", erg.Autor, f.autor)
			}
			for _, zeichen := range erg.Titel + erg.Autor {
				if zeichen >= 0x80 && zeichen <= 0x9f {
					t.Errorf("Steuerzeichen U+%04X in %q / %q", zeichen, erg.Titel, erg.Autor)
				}
			}
		})
	}
}
