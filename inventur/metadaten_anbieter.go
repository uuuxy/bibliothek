package inventur

import (
	"bytes"
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"net/url"
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
	var verlag string
	var jahr string
	var genres []string
	var zielgruppe string

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
		if datenFeld.Tag == "260" || datenFeld.Tag == "264" {
			for _, unterFeld := range datenFeld.Subfield {
				if unterFeld.Code == "b" && verlag == "" {
					verlag = strings.TrimSpace(strings.TrimRight(unterFeld.Value, ",;/ "))
				}
				if unterFeld.Code == "c" && jahr == "" {
					// DNB year might have brackets like [2021] or 2021.
					jahrStr := strings.TrimSpace(unterFeld.Value)
					jahrStr = strings.Trim(jahrStr, "[]().,;")
					jahr = jahrStr
				}
			}
		}
		// Genre-/Formangaben (z. B. "Kinderbuch", "Jugendbücher ab 12 Jahre")
		// aus den GND-Vokabularen — Basis für den Signatur-Vorschlag.
		if datenFeld.Tag == "655" {
			for _, unterFeld := range datenFeld.Subfield {
				if unterFeld.Code == "a" {
					if genre := strings.TrimSpace(unterFeld.Value); genre != "" {
						genres = append(genres, genre)
					}
				}
			}
		}
		// Verlagsangabe zur Zielgruppe: 653 $a mit Präfix "(Zielgruppe)",
		// z. B. "(Zielgruppe)ab 10 Jahre".
		if datenFeld.Tag == "653" {
			for _, unterFeld := range datenFeld.Subfield {
				wert := strings.TrimSpace(unterFeld.Value)
				if unterFeld.Code == "a" && zielgruppe == "" && strings.HasPrefix(wert, "(Zielgruppe)") {
					zielgruppe = strings.TrimSpace(strings.TrimPrefix(wert, "(Zielgruppe)"))
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
		ISBN:         isbn,
		Titel:        titel,
		Autor:        finalerAutor,
		Verlag:       verlag,
		Jahr:         jahr,
		Zielgruppe:   zielgruppe,
		BibKategorie: leiteBibKategorieAb(genres, zielgruppe),
	}, nil
}

// sucheLobid fragt das nordrhein-westfälische Bibliotheksnetzwerk ab.
// Aktuell wird diese Methode faktisch intern übersprungen (ausgeklammert im Router),
// aber bleibt als Backup bestehen.
/*
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
*/

func (client *MetadatenClient) SucheTextDNB(kontext context.Context, query string) ([]MetadatenErgebnis, error) {
	trimmed := strings.TrimSpace(query)
	if trimmed == "" {
		return nil, nil
	}

	cleanQuery := strings.ReplaceAll(trimmed, "-", "")
	cleanQuery = strings.ReplaceAll(cleanQuery, " ", "")
	var sruQuery string
	if validiereISBN(cleanQuery) {
		sruQuery = "NUM=" + cleanQuery
	} else {
		sruQuery = "any=" + url.QueryEscape(trimmed)
	}

	apiURL := fmt.Sprintf("https://services.dnb.de/sru/dnb?version=1.1&operation=searchRetrieve&query=%s&recordSchema=MARC21-xml&maximumRecords=10", sruQuery)
	koerper, fehler := client.holeInhalt(kontext, apiURL)
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

	decoder := xml.NewDecoder(io.LimitReader(bytes.NewReader(koerper), 2<<20))
	if fehler := decoder.Decode(&nutzlast); fehler != nil {
		return nil, fehler
	}

	var ergebnisse []MetadatenErgebnis

	for _, rec := range nutzlast.Records.Record {
		var titelTeile []string
		var hauptAutor string
		var extrahierteAutoren []string
		var verlag string
		var jahr string
		var isbn string

		for _, datenFeld := range rec.RecordData.Record.Datafield {
			if datenFeld.Tag == "020" {
				for _, unterFeld := range datenFeld.Subfield {
					if unterFeld.Code == "a" {
						parts := strings.Fields(unterFeld.Value)
						if len(parts) > 0 {
							clean := ""
							for _, char := range parts[0] {
								if (char >= '0' && char <= '9') || char == 'X' || char == 'x' {
									clean += string(char)
								}
							}
							if len(clean) == 10 || len(clean) == 13 {
								isbn = clean
							}
						}
					}
				}
			}
			if datenFeld.Tag == "245" {
				for _, unterFeld := range datenFeld.Subfield {
					if unterFeld.Code == "a" || unterFeld.Code == "b" || unterFeld.Code == "n" || unterFeld.Code == "p" || unterFeld.Code == "c" {
						wert := strings.TrimSpace(unterFeld.Value)
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
			if datenFeld.Tag == "260" || datenFeld.Tag == "264" {
				for _, unterFeld := range datenFeld.Subfield {
					if unterFeld.Code == "b" && verlag == "" {
						verlag = strings.TrimSpace(strings.TrimRight(unterFeld.Value, ",;/ "))
					}
					if unterFeld.Code == "c" && jahr == "" {
						jahrStr := strings.TrimSpace(unterFeld.Value)
						jahrStr = strings.Trim(jahrStr, "[]().,;")
						jahr = jahrStr
					}
				}
			}
		}

		titel := strings.Join(titelTeile, " ")
		finalerAutor := hauptAutor
		if len(extrahierteAutoren) > 0 {
			if hauptAutor == "" {
				finalerAutor = strings.Join(extrahierteAutoren, " ; ")
			} else {
				finalerAutor = hauptAutor + " (" + strings.Join(extrahierteAutoren, " ; ") + ")"
			}
		}

		if titel == "" && finalerAutor == "" {
			continue
		}

		ergebnisse = append(ergebnisse, MetadatenErgebnis{
			ISBN:   isbn,
			Titel:  titel,
			Autor:  finalerAutor,
			Verlag: verlag,
			Jahr:   jahr,
		})
	}

	return ergebnisse, nil
}

