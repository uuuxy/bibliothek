package service

import (
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"slices"
	"strconv"
	"strings"

	"bibliothek/internal/littera"
	"bibliothek/pkg/isbnutil"
	"bibliothek/pkg/lmf"
	"bibliothek/repository"

	"golang.org/x/net/html/charset"
)

// ErrKeinKatalogisat kennzeichnet eine hochgeladene XML-Datei, die kein
// Littera-MAB2-Katalogisat ist (z. B. eine Schlagwort- oder Systematikliste).
// Der Handler übersetzt diesen Fehler in ein 400 — es ist ein Datei-, kein
// Serverproblem.
var ErrKeinKatalogisat = errors.New("die Datei ist kein Littera-Katalogisat-Export (erwartet wird MAB2-XML mit <Katalogisate>-Wurzel)")

// katalogisate repräsentiert die Wurzel des MAB2 XML-Exports
type katalogisate struct {
	XMLName     xml.Name      `xml:"Katalogisate"`
	Datensaetze []katalogisat `xml:"Katalogisat"`
}

// katalogisat bündelt die Felder für einen Datensatz (ein Buch/Medium).
type katalogisat struct {
	Felder []feld `xml:"Feld"`
}

// feld repräsentiert ein einzelnes XML-feld aus Littera mit MAB-Code und Wert.
type feld struct {
	MAB     string `xml:"MAB,attr"`
	Reihung string `xml:"Reihung,attr"`
	Wert    string `xml:",chardata"`
}

// ImportiereLitteraXML liest die MAB2-XML-Datei ein und speichert die Bücher in der Datenbank.
// Ist die Datei kein Katalogisat (falsche Wurzel, kaputtes XML, keine Datensätze),
// kommt ein ErrKeinKatalogisat zurück.
func (d *ImportDienst) ImportiereLitteraXML(ctx context.Context, datei io.Reader) (int, error) {
	xmlLeser := xml.NewDecoder(datei)
	// Littera exportiert je nach Version auch ISO-8859-1/Windows-1252 —
	// ohne CharsetReader bricht der Decoder bei deklarierten Nicht-UTF-8-Encodings ab.
	xmlLeser.CharsetReader = charset.NewReaderLabel
	var wurzel katalogisate

	// Komplette XML-Struktur einlesen
	if err := xmlLeser.Decode(&wurzel); err != nil {
		return 0, fmt.Errorf("%w: %v", ErrKeinKatalogisat, err)
	}
	if len(wurzel.Datensaetze) == 0 {
		return 0, ErrKeinKatalogisat
	}

	// Titel sammeln und in EINEM gepipelineten Batch schreiben (statt je Titel
	// eine eigene DB-Rundreise) — der frühere N+1 ließ den Import mit ~15.000
	// Titeln gegen eine nicht-lokale DB in Client-Timeouts laufen.
	titelListe := make([]repository.Titel, 0, len(wurzel.Datensaetze))

	for _, kat := range wurzel.Datensaetze {
		titel, ok := titelAusFeldern(leseKatalogisat(kat))
		if !ok {
			continue
		}
		titelListe = append(titelListe, titel)
	}

	anzahl, err := d.katalogAbfragen.SchreibeTitel(ctx, titelListe)
	if err != nil {
		return 0, err
	}

	return anzahl, nil
}

// litteraFelder bündelt die aus einem Katalogisat extrahierten MAB-Rohfelder.
type litteraFelder struct {
	autor, titel, ort, verlag, isbn, jahrText, signatur, standort string
	// zielgruppen (MAB 070b: „Sekundarstufe 1", „Lehrer") und schlagwoerter (MAB 710:
	// „Physik", „Schulbuch", …) — bis zum 02.09.2026 vom Import weggeworfen, obwohl
	// zwei Drittel der Titel Schlagwörter tragen und sie das Fach nennen. Seit dem
	// 30.09.2026 kommen die Schlagwörter selbst mit (repository.Titel.Schlagworte).
	// 070b trägt Litteras Interessenkreise, bis zu fünf je Titel; entschieden am 30.09.2026:
	// Sie kommen als Schlagworte mit (docs/OFFEN.md 4.20), und die Jahrgangsspanne entsteht aus
	// allen. Bis dahin gewann der letzte Wert, und „Lehrer" oder „Referendare" gingen verloren.
	zielgruppen   []string
	schlagwoerter []string
}

// leseKatalogisat extrahiert die relevanten MAB-Felder eines Datensatzes.
func leseKatalogisat(kat katalogisat) litteraFelder {
	var f litteraFelder
	for _, eintrag := range kat.Felder {
		mab := strings.TrimSpace(eintrag.MAB)
		// Erst die Zeichen, dann der Rand, wie in der Übernahme: Ein Titel nur aus
		// Nichtsortierzeichen ist leer und fällt weg.
		wert := strings.TrimSpace(littera.OhneNichtsortierzeichen(eintrag.Wert))

		switch mab {
		case "100":
			f.autor = wert
		case "108a":
			f.standort = wert
		case "310":
			f.titel = wert
		case "410":
			f.ort = wert
		case "412":
			f.verlag = wert
		case "425":
			f.jahrText = wert
		case "540":
			f.isbn = isbnutil.OhneTrenner(wert)
		case "700":
			if eintrag.Reihung == "1" {
				f.signatur = wert
			}
		case "710":
			f.schlagwoerter = append(f.schlagwoerter, wert)
		case "070b":
			f.zielgruppen = append(f.zielgruppen, wert)
		}
	}
	return f
}

// titelAusFeldern wandelt die Rohfelder in einen Titel um. ok=false bedeutet:
// der Datensatz hat keinen Titel und wird übersprungen.
func titelAusFeldern(f litteraFelder) (repository.Titel, bool) {
	if f.titel == "" {
		return repository.Titel{}, false // Ein Titel ist zwingend erforderlich
	}

	// Lernmittelfreiheit: Litteras Kennung steht in der Signatur („LMF Bio 7") oder im
	// Standortfeld 108a („LMF", „LMF/Bibliothek"). Daraus wird das Feld (Migration
	// 093); Titel und Signatur bleiben, wie Littera sie liefert.
	lernmittel := lmf.HatKennung(f.signatur) || hatLMFKennung(f.standort)
	fach, von, bis := kategorisiere(f, lernmittel)

	erscheinungsjahr := 0
	if len(f.jahrText) >= 4 {
		// Versuche die ersten 4 Zeichen als Jahr zu parsen
		if j, err := strconv.Atoi(f.jahrText[:4]); err == nil {
			erscheinungsjahr = j
		}
	}

	// Optional: Wenn Ort vorhanden ist, diesen mit dem Verlag kombinieren
	verlag := f.verlag
	if f.ort != "" && verlag != "" {
		verlag = f.ort + " : " + verlag
	} else if f.ort != "" {
		verlag = f.ort
	}

	return repository.Titel{
		Titel:            f.titel,
		Autor:            f.autor,
		ISBN:             f.isbn,
		Verlag:           verlag,
		Erscheinungsjahr: erscheinungsjahr,
		Signatur:         f.signatur,
		IstLernmittel:    lernmittel,
		Fach:             fach,
		JahrgangVon:      von,
		JahrgangBis:      bis,
		// Die Interessenkreise hinter den Schlagwörtern, wie in der Übernahme aus der Sicherung;
		// ein gleichlautendes Wort („U plus") zählt einmal (repository.SchlagworteAusFremddaten).
		Schlagworte: append(slices.Clone(f.schlagwoerter), f.zielgruppen...),
	}, true
}

// kategorisiere leitet Fach und Jahrgangsspanne ab — in der Reihenfolge der
// Verlässlichkeit: (1) die Lernmittelsignatur „LMF Bio 7", von Hand vergeben und
// eindeutig; (2) Litteras Schlagwörter, wenn sie genau ein Fach nennen; (3) nur bei
// Lernmitteln der Titeltext („Biologie heute 7") — bei Romanen und Sachbüchern
// wäre das Raten; (4) die Zielgruppen für die Spanne, wo die Signatur keine nennt
// (lmf.JahrgangAusZielgruppen, alle Werte). Das Fach kommt nie aus den Zielgruppen,
// auch wenn sie als Schlagworte mitkommen — dieselbe Regel wie in der Übernahme.
// 0 bzw. "" heißt „unbekannt" und lässt im Bestand alles, wie es ist.
func kategorisiere(f litteraFelder, lernmittel bool) (fach string, von, bis int) {
	if z, ok := lmf.Zerlege(f.signatur); ok {
		fach, von, bis = z.Fach, z.JahrgangVon, z.JahrgangBis
	}
	if fach == "" {
		fach = lmf.FachAusSchlagworten(f.schlagwoerter)
	}
	if fach == "" && lernmittel {
		fach = lmf.FachAusText(f.titel)
	}
	if von == 0 {
		von, bis = lmf.JahrgangAusZielgruppen(f.zielgruppen)
	}
	return fach, von, bis
}
