package inventur

import (
	"bytes"
	"context"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"strings"
)

// sucheDNB fragt die Deutsche Nationalbibliothek (DNB) über die MARC21-XML-Schnittstelle ab.

func (client *MetadatenClient) sucheDNB(kontext context.Context, isbn string) (*MetadatenErgebnis, error) {
	url := fmt.Sprintf("https://services.dnb.de/sru/dnb?version=1.1&operation=searchRetrieve&query=NUM=%s&recordSchema=MARC21-xml", isbn)
	koerper, fehler := client.holeInhalt(kontext, url)
	if fehler != nil {
		return nil, fehler
	}

	var nutzlast struct {
		Records struct {
			Record []struct {
				RecordData struct {
					Record struct {
						Datafield []struct {
							Tag      string `xml:"tag,attr"`
							Subfield []struct {
								Code  string `xml:"code,attr"`
								Value string `xml:",chardata"`
							} `xml:"subfield"`
						} `xml:"datafield"`
					} `xml:"record"`
				} `xml:"recordData"`
			} `xml:"record"`
		} `xml:"records"`
	}

	// LimitReader protects against memory exhaustion during XML parsing (e.g. billion laughs attack)
	// even though the byte array is already bounded, using it on the stream explicitly ensures safety.
	decoder := xml.NewDecoder(io.LimitReader(bytes.NewReader(koerper), 2<<20))
	if fehler := decoder.Decode(&nutzlast); fehler != nil {
		return nil, fehler
	}

	if len(nutzlast.Records.Record) == 0 {
		return nil, fmt.Errorf("nicht gefunden")
	}

	var titelTeile []string
	var hauptAutor string
	var extrahierteAutoren []string

	for _, datenFeld := range nutzlast.Records.Record[0].RecordData.Record.Datafield {
		if datenFeld.Tag == "245" {
			for _, unterFeld := range datenFeld.Subfield {
				if unterFeld.Code == "a" || unterFeld.Code == "b" || unterFeld.Code == "n" || unterFeld.Code == "p" || unterFeld.Code == "c" {
					wert := strings.TrimSpace(unterFeld.Value)

					// Teils im Titel versteckte Autoren extrahieren (durch Space-Slash-Space abgetrennt)
					if idx := strings.Index(wert, " / "); idx != -1 {
						autorInfo := strings.TrimSpace(wert[idx+3:])
						if autorInfo != "" {
							extrahierteAutoren = append(extrahierteAutoren, autorInfo)
						}
						wert = strings.TrimSpace(wert[:idx])
					}

					if wert != "" {
						titelTeile = append(titelTeile, wert)
					}
				}
			}
		}
		if datenFeld.Tag == "100" || datenFeld.Tag == "700" {
			if hauptAutor == "" {
				for _, unterFeld := range datenFeld.Subfield {
					if unterFeld.Code == "a" {
						hauptAutor = strings.TrimSpace(unterFeld.Value)
						break
					}
				}
			}
		}
	}

	titel := strings.Join(titelTeile, " ")

	// Bevorzuge den direkten Autoren-Tag, falls vorhanden
	finalerAutor := hauptAutor
	if len(extrahierteAutoren) > 0 {
		if hauptAutor == "" {
			finalerAutor = strings.Join(extrahierteAutoren, " ; ")
		} else {
			finalerAutor = hauptAutor + " (" + strings.Join(extrahierteAutoren, " ; ") + ")"
		}
	}

	if titel == "" && finalerAutor == "" {
		return nil, fmt.Errorf("nicht gefunden")
	}

	return &MetadatenErgebnis{
		ISBN:  isbn,
		Titel: titel,
		Autor: finalerAutor,
	}, nil
}

// sucheLobid fragt das nordrhein-westfälische Bibliotheksnetzwerk ab.
// Aktuell wird diese Methode faktisch intern übersprungen (ausgeklammert im Router),
// aber bleibt als Backup bestehen.
func (client *MetadatenClient) sucheLobid(kontext context.Context, isbn string) (*MetadatenErgebnis, error) {
	url := fmt.Sprintf("https://lobid.org/resources/search?q=isbn:%s&format=json", isbn)
	koerper, fehler := client.holeInhalt(kontext, url)
	if fehler != nil {
		return nil, fehler
	}

	var nutzlast struct {
		Member []struct {
			Title        string `json:"title"`
			Contribution []struct {
				Agent struct {
					Label string `json:"label"`
				} `json:"agent"`
			} `json:"contribution"`
		} `json:"member"`
	}

	if fehler := json.Unmarshal(koerper, &nutzlast); fehler != nil {
		return nil, fehler
	}

	if len(nutzlast.Member) == 0 {
		return nil, fmt.Errorf("nicht gefunden")
	}

	eintrag := nutzlast.Member[0]
	autor := ""
	if len(eintrag.Contribution) > 0 {
		autor = eintrag.Contribution[0].Agent.Label
	}

	return &MetadatenErgebnis{
		ISBN:  isbn,
		Titel: eintrag.Title,
		Autor: autor,
	}, nil
}
