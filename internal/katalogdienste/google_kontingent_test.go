package katalogdienste

import (
	"bytes"
	"context"
	"image"
	"image/color"
	"image/png"
	"io"
	"net/http"
	"strings"
	"testing"
)

// Google Books wird ohne Schlüssel gefragt; das tägliche Kontingent teilt die Schule mit allen
// Nutzern. Die Suche fragt Google deshalb nur, wenn sie es braucht: nicht, wenn die DNB Titel
// und Cover liefert, und nur einmal, wenn der Titel von Google kommt. Vorher fragte die
// Cover-Suche Google vor jedem Kandidaten, nach einem Treffer bei Google ein zweites Mal.
func TestSucheNachISBN_FragtGoogleNurWennNoetig(t *testing.T) {
	t.Chdir(t.TempDir()) // speichereCoverLokal legt das Cover unter uploads/ ab
	bild := image.NewRGBA(image.Rect(0, 0, 15, 15))
	for x := 0; x < 15; x++ {
		for y := 0; y < 15; y++ {
			bild.Set(x, y, color.RGBA{0, 0, 255, 255})
		}
	}
	var puffer bytes.Buffer
	if err := png.Encode(&puffer, bild); err != nil {
		t.Fatalf("png encode: %v", err)
	}
	const dnbTreffer = `<?xml version="1.0" encoding="UTF-8"?>
<searchRetrieveResponse xmlns="http://www.loc.gov/zing/srw/"><records><record><recordData>
<record xmlns="http://www.loc.gov/MARC21/slim"><datafield tag="245" ind1="1" ind2="0"><subfield code="a">DNB Buch</subfield></datafield></record>
</recordData></record></records></searchRetrieveResponse>`
	const googleTreffer = `{"items":[{"volumeInfo":{"title":"Google Buch","imageLinks":{"thumbnail":"https://books.google.com/books/content?id=x&img=1"}}}]}`

	faelle := []struct {
		name                                 string
		dnbKennt, dnbHatCover, googleBildGut bool
		googleAnfragen                       int
		mitCover                             bool
	}{
		{"DNB mit Titel und Cover", true, true, true, 0, true},
		{"DNB mit Titel, ohne Cover", true, false, true, 1, true},
		{"Titel von Google", false, false, true, 1, true},
		{"Titel von Google, sein Bild unbrauchbar", false, false, false, 1, false},
	}
	for _, f := range faelle {
		t.Run(f.name, func(t *testing.T) {
			googleAnfragen := 0
			antwort := func(status int, rumpf []byte) *http.Response {
				return &http.Response{StatusCode: status, Body: io.NopCloser(bytes.NewReader(rumpf)), Header: make(http.Header)}
			}
			client := NewMetadatenClient()
			client.SetzeHTTPClientFuerTest(&http.Client{Transport: &mockTransport{
				roundTripFunc: func(anfrage *http.Request) (*http.Response, error) {
					switch adresse := anfrage.URL.String(); {
					case strings.Contains(adresse, "services.dnb.de") && f.dnbKennt:
						return antwort(http.StatusOK, []byte(dnbTreffer)), nil
					case strings.Contains(adresse, "portal.dnb.de/opac/mvb/cover") && f.dnbHatCover,
						strings.Contains(adresse, "books.google.com/books/content") && f.googleBildGut:
						return antwort(http.StatusOK, puffer.Bytes()), nil
					case anfrage.URL.Host == "www.googleapis.com":
						googleAnfragen++
						return antwort(http.StatusOK, []byte(googleTreffer)), nil
					}
					return antwort(http.StatusNotFound, nil), nil
				},
			}})

			ergebnis, err := client.SucheNachISBN(context.Background(), "9783161484100")
			if err != nil {
				t.Fatalf("SucheNachISBN: %v", err)
			}
			if (ergebnis.CoverURL != "") != f.mitCover {
				t.Errorf("Cover %q, erwartet eines: %v", ergebnis.CoverURL, f.mitCover)
			}
			if googleAnfragen != f.googleAnfragen {
				t.Errorf("%d Anfragen an Google, erwartet %d", googleAnfragen, f.googleAnfragen)
			}
		})
	}
}
