package inventur

import (
	"bytes"
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"net/url"
	"regexp"
	"strings"

	"bibliothek/pkg/isbnutil"
	"bibliothek/repository"

	"golang.org/x/text/unicode/norm"
)

// --- Gemeinsame Strukturen für MARC21 und SRU und ihr Lesen ---
//
// sucheDNB und SucheTextDNB werten dieselbe Antwort der DNB in MARC21-XML aus. Das Auswerten der
// Felder ist deshalb in einem gemeinsamen Sammler gebündelt.

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
//
// Jeder Wert wird dabei zusammengesetzt (Unicode NFC). Die DNB liefert Umlaute und Akzente
// zerlegt, als Grundbuchstabe mit dem Zeichen dahinter („u" + U+0308 statt „ü"). Das sieht
// gleich aus, trifft aber keine eingetippte Suche und kein Schlagwort der eigenen Liste. Die
// Datenbank setzt Titeltexte ebenfalls zusammen; hier geschieht es für alles,
// was nie gespeichert wird (die Stichwörter des Vorschlags).
//
// Die Nichtsortierzeichen fallen weg: Die DNB umschließt mit ihnen den Artikel am Anfang
// eines Titels. Sie sind nicht zu sehen, und ein Titel mit ihnen gleicht dem getippten nicht.
func dekodiereMARC(koerper []byte) (sruAntwort, error) {
	var nutzlast sruAntwort
	dekodierer := xml.NewDecoder(io.LimitReader(bytes.NewReader(koerper), 2<<20))
	if err := dekodierer.Decode(&nutzlast); err != nil {
		return sruAntwort{}, err
	}
	for i := range nutzlast.Records.Record {
		felder := nutzlast.Records.Record[i].RecordData.Record.Datafield
		for j := range felder {
			for k := range felder[j].Subfield {
				felder[j].Subfield[k].Value = norm.NFC.String(ohneNichtsortierzeichen.Replace(felder[j].Subfield[k].Value))
			}
		}
	}
	return nutzlast, nil
}

// ohneNichtsortierzeichen entfernt U+0098 und U+009C, mit denen MARC21 den Teil eines Werts
// kennzeichnet, der beim Sortieren nicht zählt.
var ohneNichtsortierzeichen = strings.NewReplacer(string(rune(0x98)), "", string(rune(0x9c)), "")

// marcBibDaten sammelt die aus den Datafields extrahierten bibliografischen
// Angaben eines einzelnen Datensatzes.
type marcBibDaten struct {
	titelTeile         []string
	hauptAutor         string
	extrahierteAutoren []string
	// verfasserangabe ist 245 $c („Joanne K. Rowling. Aus dem Engl. von Klaus Fritz"): Autor
	// nur, wenn der Satz keinen Verfasser in 100 oder 700 nennt.
	verfasserangabe string
	verlag          string
	jahr            string
	isbn            string
	genres          []string
	// freieWoerter: 653 $a ohne Vorsatz, die Wörter des Verlags („Burgen", „Erste Liebe").
	// Einträge mit Vorsatz sind Angaben anderer Art: „(Zielgruppe)", „(Lesealter)", „(Produktform)",
	// „(VLB-WN)", „(BISAC Subject Heading)", „(Produktgruppe)" (gesehen an Sätzen der DNB).
	freieWoerter []string
	// normdaten: die Schlagwörter der Gemeinsamen Normdatei (600–651 mit $2 gnd), für den
	// Vorschlag beim Bestellen.
	normdaten  []repository.Normdatenbegriff
	zielgruppe string
	// preis ist der Ladenpreis aus MARC21 020 $c — ein Vorschlag, keine Tatsache:
	// Er gilt zum Erscheinungszeitpunkt und ist nicht der Schulpreis.
	preis float64
}

// verarbeiteFeld gibt ein Datenfeld an die Auswertung seines Tags weiter.
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
		b.verarbeiteFreieWoerter(feld.Subfield)
	case "600", "610", "611", "630", "648", "650", "651":
		b.verarbeiteNormdaten(feld.Tag, feld.Subfield)
	}
}

// verarbeiteISBN liest die ISBN aus Tag 020 $a und nebenbei den Ladenpreis aus $c.
//
// Die ISBN-13 gewinnt vor der ISBN-10; innerhalb einer Länge der erste gültige Wert. Die DNB
// führt beide Formen desselben Buchs in 020, erst die 13-, dann die 10-stellige
// („Dunkelnacht": 9783751200530, dann 3751200533). Gewönne der letzte gültige Wert, entstünde
// ein über die Freitextsuche bestellter Titel zehnstellig (an 30 Sätzen der Freitextsuche
// gemessen trugen 19 beide Formen, alle mit der ISBN-10 zuletzt), und der Strichcode auf dem Buch
// (immer EAN-13) fände ihn später nicht: Die Normalform trennt 10 und 13 absichtlich, und die
// Tür der Bestellung legte den Titel ein zweites Mal an.
//
// Der erste brauchbare Preis gewinnt, nicht der letzte: Die DNB liefert zu einer ISBN mehrere
// Sätze (Auflagen), die neueren zuerst. Bei der Alexander Gesamtausgabe steht im ersten Satz
// "EUR 27.00", in einem späteren "DM 42.00, EUR 21.47"; der ältere darf den jüngeren nicht
// überschreiben.
func (b *marcBibDaten) verarbeiteISBN(subfelder []marcSubfield) {
	for _, unterFeld := range subfelder {
		if unterFeld.Code == "c" && b.preis == 0 {
			b.preis = preisAusVerfuegbarkeit(unterFeld.Value)
			continue
		}
		if unterFeld.Code != "a" {
			continue
		}
		nummer := bereinigeISBN(unterFeld.Value)
		switch {
		case nummer == "":
		case len(nummer) == 13 && len(b.isbn) != 13:
			b.isbn = nummer
		case b.isbn == "":
			b.isbn = nummer
		}
	}
}

// bereinigeISBN schält aus einem 020 $a die reine Nummer heraus und gibt "" zurück,
// wenn nichts Gültiges übrig bleibt.
//
// Das Feld enthält regelmäßig Beiwerk („3-12-345678-9 kart. : EUR 24.90"), deshalb zählt
// nur das erste Wort, und daraus nur Ziffern und das Prüfzeichen X. Was danach keine 10
// oder 13 Stellen hat, ist keine ISBN — eine halb erkannte Nummer wäre schlimmer als gar
// keine, weil sie im Katalog auf ein fremdes Buch zeigen könnte.
func bereinigeISBN(wert string) string {
	teile := strings.Fields(wert)
	if len(teile) == 0 {
		return ""
	}
	var bereinigt strings.Builder
	for _, zeichen := range teile[0] {
		if (zeichen >= '0' && zeichen <= '9') || zeichen == 'X' || zeichen == 'x' {
			bereinigt.WriteRune(zeichen)
		}
	}
	if l := bereinigt.Len(); l == 10 || l == 13 {
		return bereinigt.String()
	}
	return ""
}

// verarbeiteTitel wertet Tag 245 aus: Sachtitel, Zusatz, Zählung und Teil ergeben den Titel.
// Ältere Sätze hängen die Verfasser mit " / " an einen dieser Werte an, neuere führen sie
// als Verfasserangabe in $c — beides gehört nicht in den Titel.
func (b *marcBibDaten) verarbeiteTitel(subfelder []marcSubfield) {
	for _, unterFeld := range subfelder {
		switch unterFeld.Code {
		case "a", "b", "n", "p":
		case "c":
			if b.verfasserangabe == "" {
				b.verfasserangabe = strings.TrimSpace(unterFeld.Value)
			}
			continue
		default:
			continue
		}
		wert := strings.TrimSpace(unterFeld.Value)
		if stelle := strings.Index(wert, " / "); stelle != -1 {
			autorInfo := strings.TrimSpace(wert[stelle+3:])
			if autorInfo != "" {
				b.extrahierteAutoren = append(b.extrahierteAutoren, autorInfo)
			}
			wert = strings.TrimSpace(wert[:stelle])
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

// verarbeiteGenre sammelt Angaben zu Genre und Form aus Tag 655 $a (Vokabular der GND), die
// Gattungsbegriffe. Eine Signatur leitet sich daraus nicht ab: Die Signaturen der
// Schülerbücherei sind die Codes von Littera am Regal. Die gesammelten Begriffe sind eine Quelle
// des Vorschlags für Schlagworte (stichwoerter): Vorgeschlagen wird nur, was es schon als
// Schlagwort oder Verweis gibt.
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

// verarbeiteFreieWoerter sammelt die Verlagswörter aus Tag 653 $a — alles ohne Vorsatz in
// Klammern. Sie sind ungeprüfter Text und taugen nur als Treffer gegen die eigene Liste.
func (b *marcBibDaten) verarbeiteFreieWoerter(subfelder []marcSubfield) {
	for _, unterFeld := range subfelder {
		wert := strings.TrimSpace(unterFeld.Value)
		if unterFeld.Code == "a" && wert != "" && !strings.HasPrefix(wert, "(") {
			b.freieWoerter = append(b.freieWoerter, wert)
		}
	}
}

// zeitMitJahren trennt ein Zeitschlagwort der Normdatei in Wort und Jahre: „Geschichte 1918-1933",
// „Geschichte 1250 v. Chr.".
var zeitMitJahren = regexp.MustCompile(`^(.*\S) (\d{1,4}( v\. Chr\.)?(-\d{1,4}( v\. Chr\.)?)?)$`)

// verarbeiteNormdaten liest ein Schlagwort der Gemeinsamen Normdatei: Person (600), Körperschaft
// (610, 611), Werk (630), Zeit (648), Sache (650) und Ort (651), jeweils nur mit $2 gnd; andere
// Vokabulare stehen in denselben Feldern. Feld 689 wiederholt dieselben Begriffe als Kette und
// wird nicht gelesen.
//
// Angeboten wird in der Schreibweise der Liste aus Littera, in der ein Zusatz in spitzen Klammern
// steht: „Kafka, Franz" wird „Kafka <Franz>", „Friedrich" mit „II." und „Preußen, König" wird
// „Friedrich <Preußen, König, II.>", „Geschichte 1918-1933" wird „Geschichte <1918-1933>", der
// Zusatz $g einer Sache oder eines Orts kommt in die Klammer. Gesucht wird unter beiden Formen.
func (b *marcBibDaten) verarbeiteNormdaten(tag string, subfelder []marcSubfield) {
	werte := map[string]string{}
	for _, s := range subfelder {
		if _, da := werte[s.Code]; !da {
			werte[s.Code] = strings.TrimSpace(s.Value)
		}
	}
	a := werte["a"]
	if werte["2"] != "gnd" || a == "" {
		return
	}
	anzeige := a
	switch nach, vor, komma := strings.Cut(a, ", "); {
	case tag == "600" && komma && vor != "":
		anzeige = nach + " <" + vor + ">"
	case tag == "600" && werte["c"] != "":
		zusatz := werte["c"]
		if werte["b"] != "" {
			zusatz += ", " + werte["b"]
		}
		anzeige = a + " <" + zusatz + ">"
	case tag == "648" && zeitMitJahren.MatchString(a):
		teile := zeitMitJahren.FindStringSubmatch(a)
		anzeige = teile[1] + " <" + teile[2] + ">"
	case werte["g"] != "":
		anzeige = a + " <" + werte["g"] + ">"
	}
	formen := []string{anzeige}
	if anzeige != a {
		formen = append(formen, a)
	}
	b.normdaten = append(b.normdaten, repository.Normdatenbegriff{Anzeige: anzeige, Formen: formen})
}

// stichwoerter sind die Kandidaten des Schlagwort-Vorschlags: Gattungsbegriffe und
// Verlagswörter in der Reihenfolge des Satzes.
func (b *marcBibDaten) stichwoerter() []string {
	return append(append([]string{}, b.genres...), b.freieWoerter...)
}

func (b *marcBibDaten) titel() string {
	return strings.Join(b.titelTeile, " ")
}

// autor bevorzugt den direkten Autoren-Tag; im Titel gefundene Autoren werden
// ergänzend in Klammern angehängt. Nennt der Satz keinen von beiden, steht die
// Verfasserangabe.
func (b *marcBibDaten) autor() string {
	if b.hauptAutor == "" && len(b.extrahierteAutoren) == 0 {
		return b.verfasserangabe
	}
	if len(b.extrahierteAutoren) == 0 {
		return b.hauptAutor
	}
	if b.hauptAutor == "" {
		return strings.Join(b.extrahierteAutoren, " ; ")
	}
	return b.hauptAutor + " (" + strings.Join(b.extrahierteAutoren, " ; ") + ")"
}

// sucheDNB fragt die Deutsche Nationalbibliothek (DNB) über die MARC21-XML-Schnittstelle ab.
func (c *MetadatenClient) sucheDNB(kontext context.Context, isbn string) (*MetadatenErgebnis, error) {
	adresse := fmt.Sprintf("https://services.dnb.de/sru/dnb?version=1.1&operation=searchRetrieve&query=NUM=%s&recordSchema=MARC21-xml", isbn)
	koerper, fehler := c.holeInhalt(kontext, adresse)
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
		Preis:        akk.preis,
		Stichwoerter: akk.stichwoerter(),
		Normdaten:    akk.normdaten,
	}, nil
}

// cqlAlleWoerter baut aus einer Eingabe die Abfrage in CQL „jedes Wort irgendwo im Satz":
// any all "…". Als any=Dunkelnacht+Boie lieferte die DNB mit zwei Wörtern 0 Sätze, mit
// „Dunkelnacht" allein 15, mit any all "Dunkelnacht Boie" 14 (gemessen). In Anführungszeichen
// ist jedes Wort ein Suchwort, auch „and" oder „or"; maskiert werden nur Backslash und
// Anführungszeichen, die die Zeichenkette sonst beendeten.
func cqlAlleWoerter(text string) string {
	return `any all "` + strings.NewReplacer(`\`, `\\`, `"`, `\"`).Replace(text) + `"`
}

// SucheTextDNB fragt die DNB per Freitext ab — für die Bestellsuche, wenn keine ISBN
// vorliegt. Liefert mehrere Treffer; die Auswahl trifft die Oberfläche.
func (c *MetadatenClient) SucheTextDNB(kontext context.Context, abfrage string) ([]MetadatenErgebnis, error) {
	bereinigt := strings.TrimSpace(abfrage)
	if bereinigt == "" {
		return nil, nil
	}

	bereinigteSuche := isbnutil.OhneTrenner(bereinigt)
	var sruAbfrage string
	if isbnutil.HatForm(bereinigteSuche) {
		sruAbfrage = "NUM=" + bereinigteSuche
	} else {
		sruAbfrage = url.QueryEscape(cqlAlleWoerter(bereinigt))
	}

	adresse := fmt.Sprintf("https://services.dnb.de/sru/dnb?version=1.1&operation=searchRetrieve&query=%s&recordSchema=MARC21-xml&maximumRecords=10", sruAbfrage)
	koerper, fehler := c.holeInhalt(kontext, adresse)
	if fehler != nil {
		return nil, fehler
	}

	antwort, fehler := dekodiereMARC(koerper)
	if fehler != nil {
		return nil, fehler
	}

	var ergebnisse []MetadatenErgebnis
	for _, eintrag := range antwort.Records.Record {
		var akk marcBibDaten
		for _, datenFeld := range eintrag.RecordData.Record.Datafield {
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
			Preis:  akk.preis,
		})
	}

	return ergebnisse, nil
}
