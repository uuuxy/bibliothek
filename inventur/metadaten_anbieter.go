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

type dnbSubfield struct {
	Code  string `xml:"code,attr"`
	Value string `xml:",chardata"`
}

type dnbDatafield struct {
	Tag      string        `xml:"tag,attr"`
	Subfield []dnbSubfield `xml:"subfield"`
}

type dnbRecord struct {
	Datafield []dnbDatafield `xml:"datafield"`
}

type dnbRecordData struct {
	Record dnbRecord `xml:"record"`
}

type dnbRecordWrapper struct {
	RecordData dnbRecordData `xml:"recordData"`
}

type dnbRecords struct {
	Record []dnbRecordWrapper `xml:"record"`
}

type dnbPayload struct {
	Records dnbRecords `xml:"records"`
}

func parseDNBDatenfelder(datenFelder []dnbDatafield, fallbackISBN string) *MetadatenErgebnis {
	var titelTeile []string
	var hauptAutor string
	var extrahierteAutoren []string
	var verlag string
	var jahr string
	isbn := fallbackISBN

	for _, datenFeld := range datenFelder {
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
		return nil
	}

	return &MetadatenErgebnis{
		ISBN:   isbn,
		Titel:  titel,
		Autor:  finalerAutor,
		Verlag: verlag,
		Jahr:   jahr,
	}
}

// sucheDNB fragt die Deutsche Nationalbibliothek (DNB) über die MARC21-XML-Schnittstelle ab.

func (client *MetadatenClient) sucheDNB(kontext context.Context, isbn string) (*MetadatenErgebnis, error) {
	url := fmt.Sprintf("https://services.dnb.de/sru/dnb?version=1.1&operation=searchRetrieve&query=NUM=%s&recordSchema=MARC21-xml", isbn)
	koerper, fehler := client.holeInhalt(kontext, url)
	if fehler != nil {
		return nil, fehler
	}

	var nutzlast dnbPayload

	// LimitReader protects against memory exhaustion during XML parsing (e.g. billion laughs attack)
	// even though the byte array is already bounded, using it on the stream explicitly ensures safety.
	decoder := xml.NewDecoder(io.LimitReader(bytes.NewReader(koerper), 2<<20))
	if fehler := decoder.Decode(&nutzlast); fehler != nil {
		return nil, fehler
	}

	if len(nutzlast.Records.Record) == 0 {
		return nil, fmt.Errorf("nicht gefunden")
	}

	ergebnis := parseDNBDatenfelder(nutzlast.Records.Record[0].RecordData.Record.Datafield, isbn)
	if ergebnis == nil {
		return nil, fmt.Errorf("nicht gefunden")
	}
	return ergebnis, nil
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

	var nutzlast dnbPayload

	decoder := xml.NewDecoder(io.LimitReader(bytes.NewReader(koerper), 2<<20))
	if fehler := decoder.Decode(&nutzlast); fehler != nil {
		return nil, fehler
	}

	var ergebnisse []MetadatenErgebnis

	for _, rec := range nutzlast.Records.Record {
		ergebnis := parseDNBDatenfelder(rec.RecordData.Record.Datafield, "")
		if ergebnis != nil {
			ergebnisse = append(ergebnisse, *ergebnis)
		}
	}

	return ergebnisse, nil
}
