// Package littera liest einen Littera-Altbestand und bildet ihn auf die Begriffe
// dieser Anwendung ab.
//
// Warum CSV und nicht ODBC: Das frühere Werkzeug cmd/littera_migration sprach die
// Access-Datei über den Treiber „Microsoft Access Driver (*.mdb, *.accdb)" an — den gibt
// es nur unter Windows. Weder der Rechner, auf dem gearbeitet wird, noch der Container,
// in dem die Anwendung läuft, können ihn laden. `mdb-export` erzeugt aus derselben Datei
// CSV, läuft überall, und das Ergebnis lässt sich als Textdatei prüfen und versionieren.
//
// Warum überhaupt neu: Die Abfragen im alten Werkzeug lauteten
// `SELECT TitelID, Titel, Autor, ISBN, Verlag, Jahr, Signatur FROM TITEL`. Keine dieser
// Spalten existiert — bis auf ISBN. Der Kommentar dort sagt es offen („Wir nehmen hier
// Standardnamen an"): Es ist ein Gerüst, das nie an eine echte Littera-Datei angepasst
// wurde. Gemessen am Altbestand vom 03.08.2026 heißen sie:
//
//	Titel.Buchungsnummer      statt TitelID
//	Titel.Haupttitel          statt Titel
//	Titel.Verfasserangabe     statt Autor   (Urheber ist fast durchgängig leer)
//	Titel.Erscheinungsjahr    statt Jahr    (Text, nicht Zahl)
//	Titel.Verlag              ist ein SCHLÜSSEL auf die Tabelle Verlag, kein Freitext
//	Exemplar.Buchungsnummer   statt ExemplarID
//	Exemplar.Titel            statt TitelID
//	Exemplar.Zugangsdatum     statt ErworbenAm
//
// Die Signatur steht überhaupt nicht am Titel, sondern zweiteilig am Exemplar
// (Sig1 = Regaladresse, Sig2 = Kürzel des Titels darin).
package littera

import (
	"encoding/csv"
	"fmt"
	"io"
	"strconv"
	"strings"
)

// Titel ist ein Katalogeintrag aus der Littera-Tabelle `Titel`.
type Titel struct {
	ID               string // Buchungsnummer — Schlüssel für die Exemplar-Zuordnung
	Haupttitel       string
	Untertitel       string
	Autor            string
	ISBN             string
	VerlagID         string // Schlüssel auf Verlag.Buchungsnummer, kein Freitext
	MedienartID      string // Schlüssel auf Medienart.Buchungsnummer, kein Freitext
	Erscheinungsjahr int
	Beschreibung     string
}

// Exemplar ist ein physisches Stück aus der Littera-Tabelle `Exemplar`.
//
// Zwei Nummern, wie beim Leser — und wieder mit verschiedenen Aufgaben:
//
//   - ID = `Buchungsnummer` ist der interne Schlüssel. Darauf zeigt `Verleih.Exemplar`.
//   - Exemplarnummer ist die Nummer, die auf dem Etikett am Buch steht. Sie ist der
//     Kandidat für `buecher_exemplare.barcode_id` (siehe BarcodeInhalt).
type Exemplar struct {
	ID             string // Buchungsnummer — Ziel des Fremdschlüssels in Verleih
	Exemplarnummer string // die Nummer, die im Klartext auf dem Etikett steht
	// Bibliotheksnummer geht in den EAN-13 des Etiketts ein (siehe EtikettBarcode).
	Bibliotheksnummer string
	TitelID           string // Fremdschlüssel auf Titel.Buchungsnummer
	Barcode           string // Litteras Druckzeichenkette, NICHT der Scanwert (siehe BarcodeInhalt)
	Signatur          string // Sig1 + Sig2, zusammengesetzt
	Zugangsdatum      string // Rohform, wie exportiert (MM/DD/YY HH:MM:SS)
	Preis             float64
}

// leseTabelle liest eine mdb-export-CSV in Zeilen-Abbildungen (Spaltenname → Wert).
//
// Bewusst der Standard-CSV-Leser: mdb-export schreibt Felder mit Komma oder Zeilenumbruch
// in Anführungszeichen, und genau daran scheitert jedes selbstgebaute Aufteilen an ",".
// Bei den Annotationen im Altbestand kommt beides vor.
func leseTabelle(r io.Reader) ([]map[string]string, error) {
	leser := csv.NewReader(r)
	leser.FieldsPerRecord = -1 // mdb-export lässt Randfelder gelegentlich weg
	leser.LazyQuotes = true

	kopf, err := leser.Read()
	if err != nil {
		return nil, fmt.Errorf("kopfzeile unlesbar: %w", err)
	}

	var zeilen []map[string]string
	for {
		satz, err := leser.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("zeile unlesbar: %w", err)
		}
		zeile := make(map[string]string, len(kopf))
		for i, name := range kopf {
			if i < len(satz) {
				zeile[name] = satz[i]
			}
		}
		zeilen = append(zeilen, zeile)
	}
	return zeilen, nil
}

// LeseTitel liest die Tabelle `Titel`.
func LeseTitel(r io.Reader) ([]Titel, error) {
	zeilen, err := leseTabelle(r)
	if err != nil {
		return nil, err
	}

	titel := make([]Titel, 0, len(zeilen))
	for _, z := range zeilen {
		id := strings.TrimSpace(z["Buchungsnummer"])
		if id == "" {
			continue // ohne Schlüssel lässt sich kein Exemplar zuordnen
		}
		titel = append(titel, Titel{
			ID:               id,
			Haupttitel:       strings.TrimSpace(z["Haupttitel"]),
			Untertitel:       strings.TrimSpace(z["Untertitel"]),
			Autor:            autorAus(z),
			ISBN:             strings.TrimSpace(z["ISBN"]),
			VerlagID:         strings.TrimSpace(z["Verlag"]),
			MedienartID:      strings.TrimSpace(z["Medienart"]),
			Erscheinungsjahr: jahrAus(z["Erscheinungsjahr"]),
			Beschreibung:     strings.TrimSpace(z["Annotation"]),
		})
	}
	return titel, nil
}

// autorAus wählt das Autorenfeld.
//
// Littera führt drei Kandidaten. Im Altbestand ist `Urheber` fast durchgängig leer,
// `Verfasserangabe` trägt den Namen in Lesefassung („Reinhard Neebe"), und
// `Haupteintrag` wiederholt bei Sachbüchern schlicht den Titel — der taugt deshalb
// NICHT als Rückfall, sonst stünde der Buchtitel im Autorenfeld.
//
// Bestandsvermerke fliegen raus (siehe ohneBestandsmarken): In 2.316 Titeln steht in
// diesem Freitextfeld „Buchbestand Bibliothek" oder „LMF" statt oder neben einem Namen.
func autorAus(z map[string]string) string {
	if v := ohneBestandsmarken(strings.TrimSpace(z["Verfasserangabe"])); v != "" {
		return v
	}
	return ohneBestandsmarken(strings.TrimSpace(z["Urheber"]))
}

// jahrAus wandelt das als TEXT geführte Erscheinungsjahr in eine Zahl.
//
// Im Altbestand steht dort auch „[1997]", „ca. 2003" oder Leerstring. Ein hartes
// strconv.Atoi würde solche Zeilen verwerfen; hier werden die ersten vier
// zusammenhängenden Ziffern genommen und der Rest verworfen. 0 heißt „unbekannt".
func jahrAus(roh string) int {
	ziffern := strings.Builder{}
	for _, r := range roh {
		if r >= '0' && r <= '9' {
			ziffern.WriteRune(r)
			if ziffern.Len() == 4 {
				break
			}
			continue
		}
		if ziffern.Len() > 0 {
			break // angefangene Zahl endet — kein Zusammenkleben über Trenner hinweg
		}
	}
	if ziffern.Len() != 4 {
		return 0
	}
	jahr, err := strconv.Atoi(ziffern.String())
	if err != nil {
		return 0
	}
	return jahr
}

// LeseExemplare liest die Tabelle `Exemplar`.
func LeseExemplare(r io.Reader) ([]Exemplar, error) {
	zeilen, err := leseTabelle(r)
	if err != nil {
		return nil, err
	}

	exemplare := make([]Exemplar, 0, len(zeilen))
	for _, z := range zeilen {
		id := strings.TrimSpace(z["Buchungsnummer"])
		titelID := strings.TrimSpace(z["Titel"])
		if id == "" || titelID == "" {
			continue // ein Exemplar ohne Titel ist im Katalog nicht darstellbar
		}
		exemplare = append(exemplare, Exemplar{
			ID:                id,
			Exemplarnummer:    strings.TrimSpace(z["Exemplarnummer"]),
			Bibliotheksnummer: strings.TrimSpace(z["Bibliotheksnummer"]),
			TitelID:           titelID,
			Barcode:           strings.TrimSpace(z["Barcode"]),
			Signatur:          SignaturAus(z["Sig1"], z["Sig2"]),
			Zugangsdatum:      strings.TrimSpace(z["Zugangsdatum"]),
			Preis:             preisAus(z["Preis"]),
		})
	}
	return exemplare, nil
}

// SignaturAus setzt die Aufschrift des Buchrückens aus den beiden Littera-Feldern
// zusammen: Sig1 ist die Regaladresse („LMF Deu 7"), Sig2 das Kürzel des Titels darin
// („Bie"). Auf dem Etikett steht beides untereinander, geschrieben „LMF Deu 7 / Bie".
//
// Die Trennung mit " / " ist kein Schmuck: Der Inventur-Scope liest die Signatur als
// Präfix mit Grenze am LEERZEICHEN (repository.SignaturPraefixBedingung). Ein Regal
// „LMF Deu 7" trifft damit „LMF Deu 7 / Bie" — ohne die Leerzeichen um den Schrägstrich
// hieße die Adresse „LMF Deu 7/Bie" und das Regal fände sein eigenes Buch nicht.
//
// Der im Altbestand vorkommende Platzhalter „0" bedeutet „keine Angabe" und wird
// verworfen; er stünde sonst als Signatur am Buch.
func SignaturAus(sig1, sig2 string) string {
	a := bereinigeSignaturteil(sig1)
	b := bereinigeSignaturteil(sig2)
	switch {
	case a != "" && b != "":
		return a + " / " + b
	case a != "":
		return a
	default:
		return b
	}
}

func bereinigeSignaturteil(s string) string {
	s = strings.TrimSpace(s)
	if s == "0" {
		return ""
	}
	return s
}

// preisAus liest den Preis; unlesbare oder leere Angaben ergeben 0.
func preisAus(roh string) float64 {
	roh = strings.TrimSpace(roh)
	if roh == "" {
		return 0
	}
	// Littera exportiert mit Punkt als Dezimaltrenner; ein Komma kommt aus
	// nachbearbeiteten Dateien trotzdem vor.
	roh = strings.Replace(roh, ",", ".", 1)
	wert, err := strconv.ParseFloat(roh, 64)
	if err != nil {
		return 0
	}
	return wert
}

// VerlagNamen liest die Nachschlagetabelle `Verlag` als Schlüssel → Name.
//
// Titel.Verlag ist eine Zahl, kein Freitext — ohne diese Auflösung stünde im Katalog
// „11" statt „Klett".
func VerlagNamen(r io.Reader) (map[string]string, error) {
	zeilen, err := leseTabelle(r)
	if err != nil {
		return nil, err
	}
	namen := make(map[string]string, len(zeilen))
	for _, z := range zeilen {
		id := strings.TrimSpace(z["Buchungsnummer"])
		if id == "" {
			continue
		}
		namen[id] = strings.TrimSpace(z["Verlag"])
	}
	return namen, nil
}

// SignaturJeTitel verdichtet die Exemplar-Signaturen auf eine je Titel.
//
// Nötig, weil Littera die Signatur am EXEMPLAR führt, diese Anwendung sie aber am Titel
// (buecher_titel.signatur, seit Migration 038 die einzige Quelle). Gemessen am
// Altbestand tragen 98 von 10.454 Titeln uneinheitliche Exemplar-Signaturen — knapp 1 %.
//
// Für diese Fälle gewinnt der häufigste Wert, und der Titel wird in `abweichend`
// zurückgegeben. Bewusst nicht „der erste gewinnt": Das hinge von der Zeilenreihenfolge
// des Exports ab und wäre nicht reproduzierbar. Die Liste ist da, damit die Abweichungen
// protokolliert werden können statt still zu verschwinden.
func SignaturJeTitel(exemplare []Exemplar) (signaturen map[string]string, abweichend []string) {
	haeufigkeit := map[string]map[string]int{}
	for _, e := range exemplare {
		if e.Signatur == "" {
			continue
		}
		if haeufigkeit[e.TitelID] == nil {
			haeufigkeit[e.TitelID] = map[string]int{}
		}
		haeufigkeit[e.TitelID][e.Signatur]++
	}

	signaturen = make(map[string]string, len(haeufigkeit))
	for titelID, zaehler := range haeufigkeit {
		if len(zaehler) > 1 {
			abweichend = append(abweichend, titelID)
		}
		beste, besteZahl := "", -1
		for sig, n := range zaehler {
			// Bei Gleichstand entscheidet der alphabetisch kleinere Wert — irgendeine
			// feste Regel muss es geben, sonst ist das Ergebnis lauf-abhängig.
			if n > besteZahl || (n == besteZahl && sig < beste) {
				beste, besteZahl = sig, n
			}
		}
		signaturen[titelID] = beste
	}
	return signaturen, abweichend
}
