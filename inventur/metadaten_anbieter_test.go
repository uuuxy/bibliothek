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
		xmlData := []byte(`
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
		res, err := dekodiereMARC(xmlData)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		if len(res.Records.Record) != 1 {
			t.Fatalf("Expected 1 record, got %d", len(res.Records.Record))
		}
		if len(res.Records.Record[0].RecordData.Record.Datafield) != 1 {
			t.Fatalf("Expected 1 datafield")
		}
	})

	t.Run("Invalid XML", func(t *testing.T) {
		xmlData := []byte(`<invalid>`)
		_, err := dekodiereMARC(xmlData)
		if err == nil {
			t.Fatalf("Expected error for invalid XML, got nil")
		}
	})
}

func TestVerarbeiteFeld_Und_Subfelder(t *testing.T) {
	b := &marcBibDaten{}

	// Test Title and Extracted Authors
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
		t.Errorf("Expected hauptAutor 'Jane Doe', got '%s'", b.hauptAutor)
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
		t.Errorf("Expected verlag 'Awesome Publisher', got '%s'", b.verlag)
	}
	if b.jahr != "2023" {
		t.Errorf("Expected jahr '2023', got '%s'", b.jahr)
	}

	// Test Genre
	b.verarbeiteFeld(marcDatafield{
		Tag: "655",
		Subfield: []marcSubfield{
			{Code: "a", Value: "Fantasy"},
		},
	})
	if len(b.genres) != 1 || b.genres[0] != "Fantasy" {
		t.Errorf("Expected genre 'Fantasy', got %v", b.genres)
	}

	// Test Zielgruppe
	b.verarbeiteFeld(marcDatafield{
		Tag: "653",
		Subfield: []marcSubfield{
			{Code: "a", Value: "(Zielgruppe)ab 12 Jahre"},
		},
	})
	if b.zielgruppe != "ab 12 Jahre" {
		t.Errorf("Expected zielgruppe 'ab 12 Jahre', got '%s'", b.zielgruppe)
	}
}

func TestAutorUndTitel(t *testing.T) {
	b1 := &marcBibDaten{
		hauptAutor: "Jane Doe",
	}
	if b1.autor() != "Jane Doe" {
		t.Errorf("Expected 'Jane Doe', got '%s'", b1.autor())
	}

	b2 := &marcBibDaten{
		extrahierteAutoren: []string{"John Doe"},
	}
	if b2.autor() != "John Doe" {
		t.Errorf("Expected 'John Doe', got '%s'", b2.autor())
	}

	b3 := &marcBibDaten{
		hauptAutor:         "Jane Doe",
		extrahierteAutoren: []string{"John Doe"},
	}
	if b3.autor() != "Jane Doe (John Doe)" {
		t.Errorf("Expected 'Jane Doe (John Doe)', got '%s'", b3.autor())
	}

	b4 := &marcBibDaten{
		titelTeile: []string{"Part 1", "Part 2"},
	}
	if b4.titel() != "Part 1 Part 2" {
		t.Errorf("Expected 'Part 1 Part 2', got '%s'", b4.titel())
	}
}

func TestSucheDNB_And_SucheTextDNB(t *testing.T) {
	// mockTransport is defined in metadaten_client_test.go
	mockTr := &mockTransport{
		roundTripFunc: func(req *http.Request) (*http.Response, error) {
			if strings.Contains(req.URL.String(), "NUM=9781234567890") || strings.Contains(req.URL.String(), "any=Test+Book") {
				xmlData := `
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
					Body:       io.NopCloser(strings.NewReader(xmlData)), // Mock response correctly
				}, nil
			}
			return &http.Response{StatusCode: http.StatusOK, Body: http.NoBody}, nil
		},
	}
	client := &MetadatenClient{
		httpClient: &http.Client{Transport: mockTr},
	}

	t.Run("sucheDNB Found", func(t *testing.T) {
		res, err := client.sucheDNB(context.Background(), "9781234567890")
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		if res.Titel != "Test Title" || res.Autor != "Test Author" {
			t.Errorf("Unexpected result: %+v", res)
		}
	})

	t.Run("SucheTextDNB Found", func(t *testing.T) {
		res, err := client.SucheTextDNB(context.Background(), "Test Book")
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		if len(res) != 1 || res[0].Titel != "Test Title" || res[0].Autor != "Test Author" {
			t.Errorf("Unexpected result: %+v", res)
		}
	})
}
