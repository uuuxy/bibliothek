package inventur

import (
	"bibliothek/pkg/csvutil"
	"bytes"
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"net/url"
	"strings"
)

// --- Gemeinsame MARC21-/SRU-Strukturen und Parsing-Logik ---
//
// sucheDNB und SucheTextDNB werten dieselbe MARC21-XML-Antwort der DNB aus.
// Die Feld-Auswertung ist deshalb in einem gemeinsamen Akkumulator gebündelt
// (früher zwei je ~100er-Cognitive-Complexity-Funktionen, SonarQube go:S3776).

type marcSubfield struct {
	Code  string `xml:"code,attr"`
	Value string `xml:",chardata"`
}

type marcDatafield struct {
	Tag      string         `xml:"tag,attr"`
	Subfield []marcSubfield `xml:"subfield"`
}

type marcRecord struct {
	RecordData struct {
		Record struct {
			Datafield []marcDatafield `xml:"datafield"`
		} `xml:"record"`
	} `xml:"recordData"`
}

type sruAntwort struct {
	Records struct {
		Record []marcRecord `xml:"record"`
	} `xml:"records"`
}

// dekodiereMARC parst die SRU-/MARC21-XML-Antwort. Der LimitReader schützt vor
// Speicher-Erschöpfung beim XML-Parsing (z. B. Billion-Laughs-Angriff).
func dekodiereMARC(koerper []byte) (sruAntwort, error) {
	var nutzlast sruAntwort
	decoder := xml.NewDecoder(io.LimitReader(bytes.NewReader(koerper), 2<<20))
	if err := decoder.Decode(&nutzlast); err != nil {
		return sruAntwort{}, err
	}
	return nutzlast, nil
}

// marcBibDaten sammelt die aus den Datafields extrahierten bibliografischen
// Angaben eines einzelnen Datensatzes.
type marcBibDaten struct {
	titelTeile         []string
	hauptAutor         string
	extrahierteAutoren []string
	verlag             string
	jahr               string
	isbn               string
	genres             []string
	zielgruppe         string
}

// verarbeiteFeld leitet ein Datafield an den passenden Tag-Handler weiter.
func (b *marcBibDaten) verarbeiteFeld(feld marcDatafield) {
	switch feld.Tag {
	case "020":
		b.verarbeiteISBN(feld.Subfield)
	case "245":
		b.verarbeiteTitel(feld.Subfield)
	case "100", "700":
		b.verarbeiteAutor(feld.Subfield)
	case "260", "264":
		b.verarbeitePublikation(feld.Subfield)
	case "655":
		b.verarbeiteGenre(feld.Subfield)
	case "653":
		b.verarbeiteZielgruppe(feld.Subfield)
	}
}

// verarbeiteISBN liest die ISBN aus Tag 020 $a (letzter gültiger Wert gewinnt).
func (b *marcBibDaten) verarbeiteISBN(subfelder []marcSubfield) {
	for _, unterFeld := range subfelder {
		if unterFeld.Code != "a" {
			continue
		}
		parts := strings.Fields(unterFeld.Value)
		if len(parts) == 0 {
			continue
		}
		var clean strings.Builder
		for _, char := range parts[0] {
			if (char >= '0' && char <= '9') || char == 'X' || char == 'x' {
				clean.WriteRune(char)
			}
		}
		if l := clean.Len(); l == 10 || l == 13 {
			b.isbn = clean.String()
		}
	}
}

// verarbeiteTitel wertet Tag 245 aus. Im Titel versteckte Autoren (durch
// " / " abgetrennt) werden extrahiert.
func (b *marcBibDaten) verarbeiteTitel(subfelder []marcSubfield) {
	for _, unterFeld := range subfelder {
		switch unterFeld.Code {
		case "a", "b", "n", "p", "c":
		default:
			continue
		}
		wert := strings.TrimSpace(unterFeld.Value)
		if idx := strings.Index(wert, " / "); idx != -1 {
			autorInfo := strings.TrimSpace(wert[idx+3:])
			if autorInfo != "" {
				b.extrahierteAutoren = append(b.extrahierteAutoren, autorInfo)
			}
			wert = strings.TrimSpace(wert[:idx])
		}
		if wert != "" {
			b.titelTeile = append(b.titelTeile, wert)
		}
	}
}

// verarbeiteAutor übernimmt den ersten Autor aus Tag 100/700 $a.
func (b *marcBibDaten) verarbeiteAutor(subfelder []marcSubfield) {
	if b.hauptAutor != "" {
		return
	}
	for _, unterFeld := range subfelder {
		if unterFeld.Code == "a" {
			b.hauptAutor = strings.TrimSpace(unterFeld.Value)
			return
		}
	}
}

// verarbeitePublikation liest Verlag ($b) und Jahr ($c) aus Tag 260/264.
func (b *marcBibDaten) verarbeitePublikation(subfelder []marcSubfield) {
	for _, unterFeld := range subfelder {
		if unterFeld.Code == "b" && b.verlag == "" {
			b.verlag = strings.TrimSpace(strings.TrimRight(unterFeld.Value, ",;/ "))
		}
		if unterFeld.Code == "c" && b.jahr == "" {
			// DNB-Jahr kann Klammern enthalten, z. B. [2021] oder 2021.
			jahrStr := strings.TrimSpace(unterFeld.Value)
			jahrStr = strings.Trim(jahrStr, "[]().,;")
			b.jahr = jahrStr
		}
	}
}

// verarbeiteGenre sammelt Genre-/Formangaben aus Tag 655 $a (GND-Vokabular).
func (b *marcBibDaten) verarbeiteGenre(subfelder []marcSubfield) {
	for _, unterFeld := range subfelder {
		if unterFeld.Code == "a" {
			if genre := strings.TrimSpace(unterFeld.Value); genre != "" {
				b.genres = append(b.genres, genre)
			}
		}
	}
}

// verarbeiteZielgruppe liest die Zielgruppe aus Tag 653 $a mit Präfix
// "(Zielgruppe)", z. B. "(Zielgruppe)ab 10 Jahre".
func (b *marcBibDaten) verarbeiteZielgruppe(subfelder []marcSubfield) {
	for _, unterFeld := range subfelder {
		wert := strings.TrimSpace(unterFeld.Value)
		if unterFeld.Code == "a" && b.zielgruppe == "" && strings.HasPrefix(wert, "(Zielgruppe)") {
			b.zielgruppe = strings.TrimSpace(strings.TrimPrefix(wert, "(Zielgruppe)"))
		}
	}
}

func (b *marcBibDaten) titel() string {
	return strings.Join(b.titelTeile, " ")
}

// autor bevorzugt den direkten Autoren-Tag; im Titel gefundene Autoren werden
// ergänzend in Klammern angehängt.
func (b *marcBibDaten) autor() string {
	if len(b.extrahierteAutoren) == 0 {
		return b.hauptAutor
	}
	if b.hauptAutor == "" {
		return strings.Join(b.extrahierteAutoren, " ; ")
	}
	return b.hauptAutor + " (" + strings.Join(b.extrahierteAutoren, " ; ") + ")"
}

// sucheDNB fragt die Deutsche Nationalbibliothek (DNB) über die MARC21-XML-Schnittstelle ab.
func (client *MetadatenClient) sucheDNB(kontext context.Context, isbn string) (*MetadatenErgebnis, error) {
	url := fmt.Sprintf("https://services.dnb.de/sru/dnb?version=1.1&operation=searchRetrieve&query=NUM=%s&recordSchema=MARC21-xml", isbn)
	koerper, fehler := client.holeInhalt(kontext, url)
	if fehler != nil {
		return nil, fehler
	}

	antwort, fehler := dekodiereMARC(koerper)
	if fehler != nil {
		return nil, fehler
	}
	if len(antwort.Records.Record) == 0 {
		return nil, fmt.Errorf("nicht gefunden")
	}

	var akk marcBibDaten
	for _, datenFeld := range antwort.Records.Record[0].RecordData.Record.Datafield {
		akk.verarbeiteFeld(datenFeld)
	}

	titel := akk.titel()
	finalerAutor := akk.autor()
	if titel == "" && finalerAutor == "" {
		return nil, fmt.Errorf("nicht gefunden")
	}

	return &MetadatenErgebnis{
		ISBN:         isbn,
		Titel:        titel,
		Autor:        finalerAutor,
		Verlag:       akk.verlag,
		Jahr:         akk.jahr,
		Zielgruppe:   akk.zielgruppe,
		BibKategorie: leiteBibKategorieAb(akk.genres, akk.zielgruppe),
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

	cleanQuery := csvutil.CleanISBN(trimmed)
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

	antwort, fehler := dekodiereMARC(koerper)
	if fehler != nil {
		return nil, fehler
	}

	var ergebnisse []MetadatenErgebnis
	for _, rec := range antwort.Records.Record {
		var akk marcBibDaten
		for _, datenFeld := range rec.RecordData.Record.Datafield {
			akk.verarbeiteFeld(datenFeld)
		}

		titel := akk.titel()
		finalerAutor := akk.autor()
		if titel == "" && finalerAutor == "" {
			continue
		}

		ergebnisse = append(ergebnisse, MetadatenErgebnis{
			ISBN:   akk.isbn,
			Titel:  titel,
			Autor:  finalerAutor,
			Verlag: akk.verlag,
			Jahr:   akk.jahr,
		})
	}

	return ergebnisse, nil
}
