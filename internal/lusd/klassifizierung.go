package lusd

import (
	"fmt"
	"regexp"
	"strings"
)

// Zuordnung ist das Ergebnis der Klassifizierung — je CSV-Zeile das Ziel. Die
// Vorschau (Vorschau) ist dieselbe Entscheidung, nur für Menschen erzählt.
type Zuordnung struct {
	zielID        map[int]string     // Zeilenindex → schueler.id: Bestand aktualisieren
	ueberspringen map[int]bool       // Zeilenindex → nicht anfassen (ohne ID, mehrdeutig)
	adoptionen    []VorschauAdoption // ID-Modus: LUSD-ID anheften; Name+Geb-Modus: Geburtsdatum nachtragen
	abgaengerIDs  []string
	// neuZeilen sind die Zeilenindizes, die als Neuzugang enden würden — die eine Seite der
	// Umbenennungs-Paarung (paarung.go). geburtsdatumSetzen markiert bestätigte
	// Paare: Nur dort schreibt der Bestands-Batch das Geburtsdatum aus dem Export (eine
	// Datumskorrektur der LUSD), sonst bliebe der Schlüssel beim nächsten Lauf falsch.
	neuZeilen          []int
	geburtsdatumSetzen map[int]bool
	// datumNachgetragen: Bestandsschüler ohne Geburtsdatum, die im Name+Geb-Modus über
	// den Namen zugeordnet wurden — sie gehören nicht mehr in „nicht abgleichbar".
	datumNachgetragen map[string]bool
}

var fuehrendeNullen = regexp.MustCompile(`^0+(\d)`)

// KlassenSchluessel spiegelt die SQL-Funktion klassen_normkey aus Migration 079 (klein,
// ohne Leerzeichen, ohne führende Nullen vor Ziffern). Der BEFORE-Trigger kanonisiert
// jede geschriebene Klasse auf die registrierte Schreibweise — "05A" aus der LUSD wird
// zu "5a", wenn die Schule so schreibt. Vergliche der Import roh, stünde in jeder
// Vorschau ein Klassenwechsel "5a → 05A", der keiner ist. Die Parität zur SQL-Fassung
// hält internal/lusd/klassen_schluessel_pg_test.go.
func KlassenSchluessel(klasse string) string {
	s := strings.ToLower(strings.ReplaceAll(strings.Trim(klasse, " "), " ", ""))
	return fuehrendeNullen.ReplaceAllString(s, "$1")
}

// IstKlasseGleich sagt, ob zwei Schreibweisen dieselbe Klasse meinen ("05A" und "5a").
func IstKlasseGleich(a, b string) bool { return KlassenSchluessel(a) == KlassenSchluessel(b) }

// vorschauAusZeile baut den Vorschau-Eintrag einer CSV-Zeile. id ist der Listenschlüssel fürs
// Frontend: die LUSD-ID (ID-Modus) oder die schueler-UUID — bei Neuzugängen im
// Namensmodus die Zeilennummer, denn sonst gäbe es nichts Stabiles.
func vorschauAusZeile(id string, zeile Zeile, alteKlasse, neueKlasse string) VorschauSchueler {
	return VorschauSchueler{ID: id, Vorname: zeile.Vorname, Nachname: zeile.Nachname, AlteKlasse: alteKlasse, NeueKlasse: neueKlasse}
}

func vorschauAusBestand(s *bestandsSchueler, id string) VorschauSchueler {
	return VorschauSchueler{ID: id, Vorname: s.Vorname, Nachname: s.Nachname, AlteKlasse: s.Klasse}
}

type klassifizierungsLauf struct {
	index    bestandsIndex
	vorschau *Vorschau
	z        *Zuordnung
	gesehen  map[string]bool
}

// klassifiziere ordnet die CSV-Zeilen (rein klassifizierend, ohne Schreibzugriff)
// ein und füllt Vorschau und Zuordnung in einem Durchgang.
func klassifiziere(datei Datei, index bestandsIndex, vorschau *Vorschau) Zuordnung {
	z := Zuordnung{zielID: map[int]string{}, ueberspringen: map[int]bool{}, datumNachgetragen: map[string]bool{}, geburtsdatumSetzen: map[int]bool{}}
	lauf := klassifizierungsLauf{
		index:    index,
		vorschau: vorschau,
		z:        &z,
		gesehen:  map[string]bool{},
	}
	namenInDatei := zaehleNamenInDatei(datei)
	for i, zeile := range datei.Zeilen {
		switch datei.Modus {
		case ModusID:
			lauf.klassifiziereZeileID(i, zeile)
		case ModusName:
			lauf.klassifiziereZeileName(i, zeile, datei.Modus)
		default:
			// Nur-Name: Derselbe Name zweimal in der Datei sind zwei Menschen, die sich
			// nicht auseinanderhalten lassen — beide melden, keinen anfassen.
			if namenInDatei[zeile.namensschluessel()] > 1 {
				lauf.vorschau.Mehrdeutig = append(lauf.vorschau.Mehrdeutig, vorschauAusZeile(zeile.zeilenKennung(), zeile, "", zeile.Klasse))
				lauf.z.ueberspringen[i] = true
				// Der Name STEHT im Export — ein bestätigter Bestandsschüler dieses Namens
				// ist also nicht „nicht im Export". Ohne diese Zeile machte sammleAbgaenger
				// ihn zum Abgänger und Apply anonymisierte ihn (Prüfung 22.08.2026, A1).
				lauf.gesehen[zeile.namensschluessel()] = true
				continue
			}
			lauf.klassifiziereZeileName(i, zeile, datei.Modus)
		}
	}
	lauf.sammleAbgaenger(datei.Modus)
	return z
}

// zaehleNamenInDatei zählt im Nur-Name-Modus, wie oft jeder Name in der Datei steht.
func zaehleNamenInDatei(datei Datei) map[string]int {
	n := map[string]int{}
	if datei.Modus != ModusNurName {
		return n
	}
	for _, zeile := range datei.Zeilen {
		n[zeile.namensschluessel()]++
	}
	return n
}

// klassifiziereZeileID: Schlüssel LUSD-ID. Reihenfolge: aktiver Bestand → Rückkehrer
// (Abgänger mit dieser ID) → Adoption (ID-loser Schüler gleichen Namens+Geburtsdatums)
// → Neuzugang.
func (l *klassifizierungsLauf) klassifiziereZeileID(i int, zeile Zeile) {
	if zeile.LUSDID == "" {
		l.vorschau.OhneID++ // ohne LUSD-ID gibt es keinen stabilen Schlüssel — sichtbar zählen
		l.z.ueberspringen[i] = true
		return
	}
	l.gesehen[zeile.LUSDID] = true
	if s := l.index.aktiv[zeile.LUSDID]; s != nil {
		l.z.zielID[i] = s.ID
		if !IstKlasseGleich(s.Klasse, zeile.Klasse) {
			l.vorschau.Klassenwechsel = append(l.vorschau.Klassenwechsel, vorschauAusZeile(zeile.LUSDID, zeile, s.Klasse, zeile.Klasse))
		}
		return
	}
	if s := l.index.abgaenger[zeile.LUSDID]; s != nil {
		l.z.zielID[i] = s.ID
		l.vorschau.Rueckkehrer = append(l.vorschau.Rueckkehrer, vorschauAusZeile(zeile.LUSDID, zeile, s.Klasse, zeile.Klasse))
		return
	}
	schluessel := zeile.schluessel()
	if w := l.index.waisen[schluessel]; schluessel != "" && w != nil {
		l.z.adoptionen = append(l.z.adoptionen, VorschauAdoption{
			SchuelerID: w.ID, LUSDID: zeile.LUSDID, Vorname: zeile.Vorname, Nachname: zeile.Nachname,
			Geburtsdatum: zeile.Geburtsdatum.Format("2006-01-02"), AlteKlasse: w.Klasse, NeueKlasse: zeile.Klasse,
		})
		delete(l.index.waisen, schluessel) // konsumiert — zwei CSV-Zeilen beanspruchen nie denselben Waisen
		return
	}
	l.vorschau.Neuzugaenge = append(l.vorschau.Neuzugaenge, vorschauAusZeile(zeile.LUSDID, zeile, "", zeile.Klasse))
	l.z.neuZeilen = append(l.z.neuZeilen, i)
}

// klassifiziereZeileName ordnet eine Zeile über Name + Geburtsdatum oder nur über den Namen
// zu. Mehrdeutige Treffer werden übersprungen und gemeldet: Ein Neuzugang an ihrer Stelle
// liefe am Unique-Index unique_schueler_name_gebdatum auf oder wäre im Nur-Name-Modus geraten.
//
// Im Nur-Name-Modus gilt ein Abgänger mit demselben Namen nicht als Rückkehrer. Es kann
// ebenso ein neuer Fünftklässler sein, der sonst auf dem Datensatz des Abgegangenen landete
// (Sperre, Schulden, Lesehistorie); das Sekretariat entscheidet von Hand.
func (l *klassifizierungsLauf) klassifiziereZeileName(i int, zeile Zeile, modus Modus) {
	// Der Schlüssel kommt aus dem Modus, wie auf der Gegenseite (bestandsSchluessel in
	// bestand.go). Käme er vom Aufrufer, könnte ein Namensschlüssel gegen den Index aus
	// Name und Datum laufen und träfe still niemanden.
	nurName := modus == ModusNurName
	schluessel := zeile.schluesselFuer(modus)
	if l.ordneAktivemZu(i, zeile, schluessel) || l.ordneAbgaengerZu(i, zeile, schluessel, nurName) {
		return
	}
	if !nurName && l.ordneUeberNamenZu(i, zeile) {
		return
	}
	l.vorschau.Neuzugaenge = append(l.vorschau.Neuzugaenge, vorschauAusZeile(zeile.zeilenKennung(), zeile, "", zeile.Klasse))
	l.z.neuZeilen = append(l.z.neuZeilen, i)
}

// zeilenKennung steht in den Listen der Vorschau, wo eine Zeile noch keinen Datensatz hat.
func (z Zeile) zeilenKennung() string {
	return fmt.Sprintf("zeile-%d", z.Nr)
}

// ordneAktivemZu: Der Schlüssel trifft den aktiven Bestand. Ein eindeutiger Treffer wird
// zugeordnet, ein mehrdeutiger gemeldet und übersprungen; false heißt kein Treffer.
func (l *klassifizierungsLauf) ordneAktivemZu(i int, zeile Zeile, schluessel string) bool {
	s, ok := l.index.aktiv[schluessel]
	if !ok {
		return false
	}
	if s == nil {
		l.vorschau.Mehrdeutig = append(l.vorschau.Mehrdeutig, vorschauAusZeile(zeile.zeilenKennung(), zeile, "", zeile.Klasse))
		l.z.ueberspringen[i] = true
		return true
	}
	l.gesehen[schluessel] = true
	l.z.zielID[i] = s.ID
	if !IstKlasseGleich(s.Klasse, zeile.Klasse) {
		l.vorschau.Klassenwechsel = append(l.vorschau.Klassenwechsel, vorschauAusZeile(s.ID, zeile, s.Klasse, zeile.Klasse))
	}
	return true
}

// ordneAbgaengerZu: Der Schlüssel trifft einen Abgänger. Eindeutig ist er ein Rückkehrer; im
// Nur-Name-Modus und bei mehreren Treffern wird die Zeile gemeldet und übersprungen.
func (l *klassifizierungsLauf) ordneAbgaengerZu(i int, zeile Zeile, schluessel string, nurName bool) bool {
	s, ok := l.index.abgaenger[schluessel]
	if !ok {
		return false
	}
	if s == nil || nurName {
		alteKlasse := ""
		if s != nil {
			alteKlasse = s.Klasse
		}
		l.vorschau.Mehrdeutig = append(l.vorschau.Mehrdeutig, vorschauAusZeile(zeile.zeilenKennung(), zeile, alteKlasse, zeile.Klasse))
		l.z.ueberspringen[i] = true
		return true
	}
	l.z.zielID[i] = s.ID
	l.vorschau.Rueckkehrer = append(l.vorschau.Rueckkehrer, vorschauAusZeile(s.ID, zeile, s.Klasse, zeile.Klasse))
	return true
}

// ordneUeberNamenZu ist die Rückfallstufe im Modus Name + Geburtsdatum: Ein Bestandsschüler
// ohne Geburtsdatum wird über den Namen zugeordnet und bekommt das Datum aus dem Export; bei
// mehreren Treffern wird gemeldet und nichts angelegt. Ohne diese Stufe entstünde nach dem
// Wechsel von der Liste ohne Datum zum Export mit Datum jeder Bestandsschüler doppelt.
func (l *klassifizierungsLauf) ordneUeberNamenZu(i int, zeile Zeile) bool {
	if zeile.Geburtsdatum == nil {
		return false
	}
	namensSchluessel := zeile.namensschluessel()
	w, ok := l.index.ohneDatumNachName[namensSchluessel]
	if !ok {
		return false
	}
	if w == nil {
		l.vorschau.Mehrdeutig = append(l.vorschau.Mehrdeutig, vorschauAusZeile(zeile.zeilenKennung(), zeile, "", zeile.Klasse))
		l.z.ueberspringen[i] = true
		return true
	}
	l.z.zielID[i] = w.ID
	l.z.datumNachgetragen[w.ID] = true
	l.z.adoptionen = append(l.z.adoptionen, VorschauAdoption{
		SchuelerID: w.ID, Vorname: zeile.Vorname, Nachname: zeile.Nachname,
		Geburtsdatum: zeile.Geburtsdatum.Format("2006-01-02"), AlteKlasse: w.Klasse, NeueKlasse: zeile.Klasse,
	})
	delete(l.index.ohneDatumNachName, namensSchluessel) // verbraucht: zwei Zeilen beanspruchen nie denselben
	return true
}

// sammleAbgaenger: Wer aktiv ist und nicht im Export steht, geht ab — im ID-Modus jeder
// mit echter LUSD-ID, in den Namensmodi nur, wer schon einmal von einem Export BESTÄTIGT
// wurde (lusd_bestaetigt_am). Nie bestätigte Handanlagen bleiben stehen und werden als
// „nicht im Export" gemeldet; Schüler ohne Geburtsdatum sind nicht abgleichbar.
func (l *klassifizierungsLauf) sammleAbgaenger(modus Modus) {
	for schluessel, s := range l.index.aktiv {
		if s == nil || l.gesehen[schluessel] {
			continue
		}
		if modus != ModusID && !s.LUSDBestaetigt {
			l.vorschau.NichtImExport = append(l.vorschau.NichtImExport, vorschauAusBestand(s, s.ID))
			continue
		}
		listenID := schluessel // ID-Modus: die LUSD-ID
		if modus != ModusID {
			listenID = s.ID
		}
		l.vorschau.Abgaenger = append(l.vorschau.Abgaenger, vorschauAusBestand(s, listenID))
		l.z.abgaengerIDs = append(l.z.abgaengerIDs, s.ID)
	}
	for i := range l.index.ohneSchluessel {
		if l.z.datumNachgetragen[l.index.ohneSchluessel[i].ID] {
			continue // über den Namen zugeordnet, Datum wird nachgetragen
		}
		l.vorschau.NichtAbgleichbar = append(l.vorschau.NichtAbgleichbar, vorschauAusBestand(&l.index.ohneSchluessel[i], l.index.ohneSchluessel[i].ID))
	}
	for i := range l.index.mehrdeutigAktiv {
		l.vorschau.Mehrdeutig = append(l.vorschau.Mehrdeutig, vorschauAusBestand(&l.index.mehrdeutigAktiv[i], l.index.mehrdeutigAktiv[i].ID))
	}
}
