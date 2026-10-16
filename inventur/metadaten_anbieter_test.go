package inventur

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestDekodiereMARC(t *testing.T) {
	t.Run("Valid XML", func(t *testing.T) {
		xmlDaten := []byte(`
			<root>
			<records>
				<record>
					<recordData>
						<record>
							<datafield tag="245">
								<subfield code="a">Test Title</subfield>
							</datafield>
						</record>
					</recordData>
				</record>
			</records>
			</root>
		`)
		ergebnis, err := dekodiereMARC(xmlDaten)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		if len(ergebnis.Records.Record) != 1 {
			t.Fatalf("dekodiereMARC(xmlDaten): %d Datensätze, erwartet 1", len(ergebnis.Records.Record))
		}
		if len(ergebnis.Records.Record[0].RecordData.Record.Datafield) != 1 {
			t.Fatalf("dekodiereMARC(xmlDaten): %d Datenfelder, erwartet 1", len(ergebnis.Records.Record[0].RecordData.Record.Datafield))
		}
	})

	t.Run("Invalid XML", func(t *testing.T) {
		xmlDaten := []byte(`<invalid>`)
		_, err := dekodiereMARC(xmlDaten)
		if err == nil {
			t.Fatalf("dekodiereMARC(<invalid>) = nil, erwartet einen Fehler")
		}
	})
}

func TestVerarbeiteFeld_Und_Subfelder(t *testing.T) {
	b := &marcBibDaten{}

	// Titel und ausgelesene Autoren.
	b.verarbeiteFeld(marcDatafield{
		Tag: "245",
		Subfield: []marcSubfield{
			{Code: "a", Value: "The Great Book / John Doe"},
			{Code: "b", Value: "A sub title"},
		},
	})
	if len(b.titelTeile) != 2 || b.titelTeile[0] != "The Great Book" || b.titelTeile[1] != "A sub title" {
		t.Errorf("Unexpected titelTeile: %v", b.titelTeile)
	}
	if len(b.extrahierteAutoren) != 1 || b.extrahierteAutoren[0] != "John Doe" {
		t.Errorf("Unexpected extrahierteAutoren: %v", b.extrahierteAutoren)
	}

	// Test Main Author
	b.verarbeiteFeld(marcDatafield{
		Tag: "100",
		Subfield: []marcSubfield{
			{Code: "a", Value: "Jane Doe"},
		},
	})
	if b.hauptAutor != "Jane Doe" {
		t.Errorf("b.hauptAutor = '%s', erwartet 'Jane Doe'", b.hauptAutor)
	}

	// Test Publication Details
	b.verarbeiteFeld(marcDatafield{
		Tag: "260",
		Subfield: []marcSubfield{
			{Code: "b", Value: "Awesome Publisher, "},
			{Code: "c", Value: "[2023]"},
		},
	})
	if b.verlag != "Awesome Publisher" {
		t.Errorf("b.verlag = '%s', erwartet 'Awesome Publisher'", b.verlag)
	}
	if b.jahr != "2023" {
		t.Errorf("b.jahr = '%s', erwartet '2023'", b.jahr)
	}

	// Test Genre
	b.verarbeiteFeld(marcDatafield{
		Tag: "655",
		Subfield: []marcSubfield{
			{Code: "a", Value: "Fantasy"},
		},
	})
	if len(b.genres) != 1 || b.genres[0] != "Fantasy" {
		t.Errorf("b.genres = %v, erwartet 'Fantasy'", b.genres)
	}

	// Test Zielgruppe
	b.verarbeiteFeld(marcDatafield{
		Tag: "653",
		Subfield: []marcSubfield{
			{Code: "a", Value: "(Zielgruppe)ab 12 Jahre"},
		},
	})
	if b.zielgruppe != "ab 12 Jahre" {
		t.Errorf("b.zielgruppe = '%s', erwartet 'ab 12 Jahre'", b.zielgruppe)
	}
}

func TestAutorUndTitel(t *testing.T) {
	b1 := &marcBibDaten{
		hauptAutor: "Jane Doe",
	}
	if b1.autor() != "Jane Doe" {
		t.Errorf("b1.autor() = '%s', erwartet 'Jane Doe'", b1.autor())
	}

	b2 := &marcBibDaten{
		extrahierteAutoren: []string{"John Doe"},
	}
	if b2.autor() != "John Doe" {
		t.Errorf("b2.autor() = '%s', erwartet 'John Doe'", b2.autor())
	}

	b3 := &marcBibDaten{
		hauptAutor:         "Jane Doe",
		extrahierteAutoren: []string{"John Doe"},
	}
	if b3.autor() != "Jane Doe (John Doe)" {
		t.Errorf("b3.autor() = '%s', erwartet 'Jane Doe (John Doe)'", b3.autor())
	}

	b4 := &marcBibDaten{
		titelTeile: []string{"Part 1", "Part 2"},
	}
	if b4.titel() != "Part 1 Part 2" {
		t.Errorf("b4.titel() = '%s', erwartet 'Part 1 Part 2'", b4.titel())
	}
}

func TestSucheDNB_And_SucheTextDNB(t *testing.T) {
	// mockTransport steht in metadaten_client_test.go.
	mockTr := &mockTransport{
		roundTripFunc: func(anfrage *http.Request) (*http.Response, error) {
			if strings.Contains(anfrage.URL.String(), "NUM=9781234567890") || anfrage.URL.Query().Get("query") == `any all "Test Book"` {
				xmlDaten := `
					<searchRetrieveResponse xmlns="http://www.loc.gov/zing/srw/">
					  <records>
						<record>
						  <recordData>
							<record xmlns="http://www.loc.gov/MARC21/slim">
							  <datafield tag="245" ind1="1" ind2="0">
								<subfield code="a">Test Title</subfield>
							  </datafield>
							  <datafield tag="100" ind1="1" ind2="0">
								<subfield code="a">Test Author</subfield>
							  </datafield>
							</record>
						  </recordData>
						</record>
					  </records>
					</searchRetrieveResponse>`
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(xmlDaten)), // Mock response correctly
				}, nil
			}
			return &http.Response{StatusCode: http.StatusOK, Body: http.NoBody}, nil
		},
	}
	client := &MetadatenClient{
		httpClient: &http.Client{Transport: mockTr},
	}

	t.Run("sucheDNB Found", func(t *testing.T) {
		ergebnis, err := client.sucheDNB(context.Background(), "9781234567890")
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		if ergebnis.Titel != "Test Title" || ergebnis.Autor != "Test Author" {
			t.Errorf("Unexpected result: %+v", ergebnis)
		}
	})

	t.Run("SucheTextDNB Found", func(t *testing.T) {
		ergebnis, err := client.SucheTextDNB(context.Background(), "Test Book")
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		if len(ergebnis) != 1 || ergebnis[0].Titel != "Test Title" || ergebnis[0].Autor != "Test Author" {
			t.Errorf("Unexpected result: %+v", ergebnis)
		}
	})
}
