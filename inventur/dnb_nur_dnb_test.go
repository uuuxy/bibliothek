package inventur

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"slices"
	"testing"
)

// SucheDNBNachISBN ist die Quelle des Schlagwort-Vorschlags im Buchformular und beim
// Nachbestellen (entschieden am 30.09.2026). Sie fragt NUR die DNB: Schlagworte hat keine
// andere Quelle, und SucheNachISBN fragte danach noch Google Books (anonymes Tageskontingent)
// und lüde das Cover herunter. Die drei Ausgänge tragen je ein eigenes Sentinel, weil die
// Oberfläche sie verschieden ansagt: kennt die ISBN nicht, nicht erreichbar, keine ISBN.
func TestSucheDNBNachISBN_FragtNurDieDNB(t *testing.T) {
	const satz = `<searchRetrieveResponse xmlns="http://www.loc.gov/zing/srw/"><records><record><recordData>
		<record xmlns="http://www.loc.gov/MARC21/slim">
		  <datafield tag="245" ind1="1" ind2="0"><subfield code="a">Dunkelnacht</subfield></datafield>
		  <datafield tag="653" ind1=" " ind2=" "><subfield code="a">Krieg</subfield></datafield>
		  <datafield tag="650" ind1=" " ind2="7"><subfield code="a">Schulstress</subfield><subfield code="2">gnd</subfield></datafield>
		</record>
	</recordData></record></records></searchRetrieveResponse>`
	const leer = `<searchRetrieveResponse xmlns="http://www.loc.gov/zing/srw/"><numberOfRecords>0</numberOfRecords></searchRetrieveResponse>`

	faelle := []struct {
		name      string
		isbn      string
		status    int
		koerper   string
		sentinel  error
		gefragt   []string
		stichwort string
	}{
		{"Satz da", "978-3-7512-0053-0", http.StatusOK, satz, nil, []string{"services.dnb.de"}, "Krieg"},
		{"DNB kennt die ISBN nicht, kein Rückfall auf Google", "9783751200530", http.StatusOK, leer, ErrNichtInDerDNB, []string{"services.dnb.de"}, ""},
		{"DNB kaputt", "9783751200530", http.StatusServiceUnavailable, "", ErrDNBNichtErreichbar, []string{"services.dnb.de"}, ""},
		{"keine ISBN", "Dunkelnacht", http.StatusOK, satz, ErrUngueltigeISBN, nil, ""},
	}
	for _, f := range faelle {
		t.Run(f.name, func(t *testing.T) {
			var gefragt []string
			client := NeuerMetadatenClient()
			client.SetzeHTTPClientFuerTest(&http.Client{Transport: &mockTransport{
				roundTripFunc: func(req *http.Request) (*http.Response, error) {
					gefragt = append(gefragt, req.URL.Host)
					if req.URL.Host != "services.dnb.de" {
						return &http.Response{StatusCode: http.StatusNotFound, Body: http.NoBody, Header: make(http.Header)}, nil
					}
					return &http.Response{StatusCode: f.status, Body: io.NopCloser(bytes.NewBufferString(f.koerper)), Header: make(http.Header)}, nil
				},
			}})

			erg, err := client.SucheDNBNachISBN(context.Background(), f.isbn)

			if f.sentinel != nil {
				if !errors.Is(err, f.sentinel) {
					t.Errorf("Fehler %v, erwartet %v", err, f.sentinel)
				}
			} else if err != nil {
				t.Fatalf("Fehler: %v", err)
			}
			if !slices.Equal(gefragt, f.gefragt) {
				t.Errorf("gefragt %v, erwartet %v — nur die DNB, ohne Cover und ohne Rückfallquellen", gefragt, f.gefragt)
			}
			if f.stichwort == "" {
				return
			}
			if !slices.Contains(erg.Stichwoerter, f.stichwort) {
				t.Errorf("Stichwörter %v ohne %q", erg.Stichwoerter, f.stichwort)
			}
			if len(erg.Normdaten) != 1 || erg.Normdaten[0].Anzeige != "Schulstress" {
				t.Errorf("Normdaten %+v, erwartet Schulstress", erg.Normdaten)
			}
			if erg.CoverURL != "" {
				t.Errorf("Cover %q — der Vorschlag lädt keins", erg.CoverURL)
			}
		})
	}
}
