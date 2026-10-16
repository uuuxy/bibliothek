package lusd

import (
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/xuri/excelize/v2"
)

// Zeile ist eine gelesene Zeile des Exports: ein Schüler, wie die LUSD ihn nennt. Nr
// ist die Zeilennummer in der Datei; an ihr hängt die Wahl der Umbenennungen zwischen
// Vorschau und Import.
type Zeile struct {
	LUSDID       string
	Vorname      string
	Nachname     string
	Klasse       string
	Geburtsdatum *time.Time
	Strasse      string
	Hausnummer   string
	PLZ          string
	Ort          string
	ElternEmail  string
	// EintrittAm ist der Schuleintritt laut Bericht (Schueler_Eintritt_AktuelleSchule) —
	// der zweite Schlüssel der Umbenennungs-Paarung (paarung.go); nil ohne Spalte.
	EintrittAm *time.Time
	Nr         int
	// geburtsdatumUebernehmen: nur für bestätigte Umbenennungs-Paare — der Bestands-Batch
	// schreibt dann das Geburtsdatum des Exports (Datumskorrektur der LUSD).
	geburtsdatumUebernehmen bool
}

// schluessel ist der Name+Geburtsdatum-Schlüssel der Zeile ("" ohne Datum).
func (z Zeile) schluessel() string {
	return waisenSchluessel(z.Vorname, z.Nachname, z.Geburtsdatum)
}

// namensschluessel ist der Nur-Name-Schlüssel (Vorname + Nachname, normalisiert).
func (z Zeile) namensschluessel() string {
	return namensSchluessel(z.Vorname, z.Nachname)
}

// schluesselFuer liefert den Nachschlage-Schlüssel dieser Zeile für den gegebenen Modus —
// das Gegenstück zu bestandsSchluessel (bestand.go), das dasselbe für die Bestandszeile
// tut. Zwei Seiten, eine Regel: Solange beide über den Modus gehen, kann niemand versehentlich
// den Namensschlüssel gegen den Name+Datum-Index schlagen (das traf still niemanden).
func (z Zeile) schluesselFuer(modus Modus) string {
	if modus == ModusNurName {
		return z.namensschluessel()
	}
	return z.schluessel()
}

const (
	spalteID           = "lusd_id"
	spalteVorname      = "vorname"
	spalteNachname     = "nachname"
	spalteKlasse       = "klasse"
	spalteGeburtsdatum = "geburtsdatum"
	// Optionale Kontakt-/Adressspalten. Fehlen sie im Export, bleibt der Import
	// gültig; die Felder sind dann leer. Zweck: Schadens-Rechnung (Anschrift) und
	// Elternkontakt (E-Mail). Die Kopfzellen müssen so heißen, gleich in welcher Schreibung.
	spalteStrasse     = "strasse"
	spalteHausnummer  = "hausnummer"
	spaltePLZ         = "plz"
	spalteOrt         = "ort"
	spalteElternEmail = "eltern_email"
	// spalteEintritt: Eintritt an der aktuellen Schule. Optional; ohne die Spalte bleibt
	// die Umbenennungs-Paarung auf Geburtsdatum, Klasse, Name und Anschrift angewiesen.
	spalteEintritt = "eintritt"
)

// Modus sagt, worüber der Import die Schüler zuordnet — die Datei entscheidet.
// Der Export der Schule hat keine Schüler-ID und bekommt auch keine; der LANIS-
// Klassenlisten-Export hat nicht einmal ein Geburtsdatum. Drei Stufen, absteigend
// sicher; die Vorschau sagt dem Sekretariat, welche gilt und was sie kostet.
type Modus int

const (
	// ModusID ordnet über die LUSD-ID zu; Zeilen ohne ID werden übersprungen.
	ModusID Modus = iota
	// ModusName ordnet über Vorname, Nachname und Geburtsdatum zu. Das Datum ist dann in
	// jeder Zeile Pflicht; fehlt es, bricht der Lauf ab, statt die Zeile zu überspringen.
	ModusName
	// ModusNurName ordnet nur über Vorname und Nachname zu, wenn die Datei weder ID noch
	// Geburtsdatum trägt. Namensgleiche werden nicht zugeordnet, sondern gemeldet.
	ModusNurName
)

// String ist der Wert, den die Vorschau der Oberfläche meldet.
func (m Modus) String() string {
	switch m {
	case ModusName:
		return "name_geburtsdatum"
	case ModusNurName:
		return "name"
	default:
		return "lusd_id"
	}
}

// Datei ist die gelesene Datei: die Zeilen plus der Modus, in dem sie zugeordnet
// werden, plus das, was beim Zusammenlegen doppelter Zeilen verloren ging.
type Datei struct {
	Zeilen []Zeile
	Modus  Modus
	// DublettenInDatei zählt Zeilen, die auf denselben Schlüssel fielen und von der
	// späteren überschrieben wurden (ID- und Name+Geburtsdatum-Modus: letzte gewinnt).
	// Im Nur-Name-Modus wird nicht zusammengelegt — gleiche Namen sind dort mehrdeutig.
	DublettenInDatei int
	// Zusammengelegt nennt die Dubletten, deren Zeilen sich in der Klasse unterschieden
	// (AlteKlasse = überschriebene Zeile, NeueKlasse = gewonnene, ID = Zeilennummer der
	// gewonnenen). Im Name+Geburtsdatum-Modus kann das ein Schulformwechsler sein, den die LUSD
	// doppelt führt, oder zwei Schüler mit demselben Namen und einem Tippfehler im Datum; die
	// Vorschau nennt sie deshalb beim Namen. Gleiche Klasse ist eine bloße Doppelzeile und bleibt
	// eine Zahl.
	Zusammengelegt []VorschauSchueler
}

// spaltenWert liest eine optionale Spalte getrimmt aus; fehlt sie oder ist die
// Zeile zu kurz, wird "" zurückgegeben.
func spaltenWert(zellen []string, spalten map[string]int, spalte string) string {
	i, ok := spalten[spalte]
	if !ok || i >= len(zellen) {
		return ""
	}
	return strings.TrimSpace(zellen[i])
}

var excelSerienzahl = regexp.MustCompile(`^\d{4,6}(\.\d+)?$`)

// leseDatum liest eine Datumsspalte (Geburtsdatum, Schuleintritt). Excel liefert
// Datumszellen als Serienzahl (Tage seit 1899-12-30), CSV als Text in mehreren
// Schreibweisen. nil, wenn die Spalte fehlt oder der Wert unlesbar ist.
func leseDatum(zellen []string, spalten map[string]int, spalte string) *time.Time {
	roh := spaltenWert(zellen, spalten, spalte)
	if roh == "" {
		return nil
	}
	if excelSerienzahl.MatchString(roh) {
		if serie, err := strconv.ParseFloat(roh, 64); err == nil {
			if t, err := excelize.ExcelDateToTime(serie, false); err == nil {
				t = time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
				return &t
			}
		}
		return nil
	}
	for _, format := range []string{datumsformatDE, "2.1.2006", datumsformatISO, "01/02/2006", "2006-01-02T15:04:05Z07:00", "2006-01-02 15:04:05"} {
		if t, parseErr := time.ParseInLocation(format, roh, time.UTC); parseErr == nil {
			t = time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
			return &t
		}
	}
	return nil
}

// plausiblerEintritt verwirft Platzhalter- und Unsinnsdaten im Schuleintritt: vor 1990,
// in der Zukunft oder nicht nach dem Geburtsdatum. Ein Platzhalter wie 01.01.1900 in
// jeder Zeile machte sonst jeden Namensvetter zum „sicheren" Paar, und bestätigt überschriebe
// der Batch das Geburtsdatum.
func plausiblerEintritt(eintritt, geburtsdatum *time.Time) *time.Time {
	if eintritt == nil {
		return nil
	}
	heute := time.Now().UTC()
	if eintritt.Year() < 1990 || eintritt.After(heute) {
		return nil
	}
	if geburtsdatum != nil && !eintritt.After(*geburtsdatum) {
		return nil
	}
	return eintritt
}

// leseZeile parst eine Datenzeile und validiert die Pflichtfelder. Alle Spalten
// laufen über spaltenWert, weil ID- und Geburtsdatum-Spalte fehlen dürfen.
func leseZeile(zellen []string, spalten map[string]int, nr int) (Zeile, error) {
	vorname := spaltenWert(zellen, spalten, spalteVorname)
	nachname := spaltenWert(zellen, spalten, spalteNachname)
	klasse := spaltenWert(zellen, spalten, spalteKlasse)

	if vorname == "" || nachname == "" || klasse == "" {
		return Zeile{}, &ZeileFehler{Zeile: nr, Art: ErrPflichtfeldLeer}
	}

	geb := leseDatum(zellen, spalten, spalteGeburtsdatum)
	return Zeile{
		LUSDID:       spaltenWert(zellen, spalten, spalteID),
		Vorname:      vorname,
		Nachname:     nachname,
		Klasse:       klasse,
		Geburtsdatum: geb,
		EintrittAm:   plausiblerEintritt(leseDatum(zellen, spalten, spalteEintritt), geb),
		Strasse:      spaltenWert(zellen, spalten, spalteStrasse),
		Hausnummer:   spaltenWert(zellen, spalten, spalteHausnummer),
		PLZ:          spaltenWert(zellen, spalten, spaltePLZ),
		Ort:          spaltenWert(zellen, spalten, spalteOrt),
		ElternEmail:  spaltenWert(zellen, spalten, spalteElternEmail),
		Nr:           nr,
	}, nil
}

// istLeereZeile: Excel-Berichte enden oft mit Leer- oder Summenzeilen; eine Zeile ohne
// jeden Wert ist kein Schüler und kein Fehler.
func istLeereZeile(zellen []string) bool {
	for _, c := range zellen {
		if strings.TrimSpace(c) != "" {
			return false
		}
	}
	return true
}

// LeseDatei liest die Datei (CSV oder Excel) vollständig, bestimmt den Modus und
// legt doppelte Zeilen zusammen. Harte Fehler statt stillem Überspringen: Das
// Sekretariat soll eine kaputte Datei als Meldung sehen, nicht als halb importierten
// Bestand.
func LeseDatei(inhalt []byte) (Datei, error) {
	tabelle, err := leseTabelle(inhalt)
	if err != nil {
		return Datei{}, err
	}
	kopfNr, spalten, err := findeKopfzeile(tabelle)
	if err != nil {
		return Datei{}, err
	}

	var zeilen []Zeile
	irgendeineID, irgendeinDatum := false, false
	for i := kopfNr + 1; i < len(tabelle); i++ {
		if istLeereZeile(tabelle[i].zellen) {
			continue
		}
		zeile, err := leseZeile(tabelle[i].zellen, spalten, tabelle[i].nr)
		if err != nil {
			return Datei{}, err
		}
		irgendeineID = irgendeineID || zeile.LUSDID != ""
		irgendeinDatum = irgendeinDatum || zeile.Geburtsdatum != nil
		zeilen = append(zeilen, zeile)
	}

	// Eine Spalte, in der kein einziger Wert steht, zählt nicht — LUSD exportiert
	// Spalten mitunter leer. Was die Datei wirklich hergibt, bestimmt den Modus.
	if _, hatIDSpalte := spalten[spalteID]; hatIDSpalte && irgendeineID {
		return legeDublettenZusammen(zeilen, ModusID, func(z Zeile) string { return z.LUSDID }), nil
	}
	if irgendeinDatum {
		if err := pruefeGeburtsdatumJeZeile(zeilen); err != nil {
			return Datei{}, err
		}
		return legeDublettenZusammen(zeilen, ModusName, Zeile.schluessel), nil
	}
	return Datei{Zeilen: zeilen, Modus: ModusNurName}, nil
}

// pruefeGeburtsdatumJeZeile meldet die erste Zeile ohne lesbares Geburtsdatum.
func pruefeGeburtsdatumJeZeile(zeilen []Zeile) error {
	for _, z := range zeilen {
		if z.Geburtsdatum == nil {
			return &ZeileFehler{Zeile: z.Nr, Art: ErrGeburtsdatumFehlt}
		}
	}
	return nil
}

// legeDublettenZusammen lässt von mehreren Zeilen mit demselben Schlüssel die letzte
// gewinnen (an ihrem ersten Platz) und zählt, was dabei überschrieben wurde. Zeilen
// ohne Schlüssel (ID-Modus: leere ID) bleiben einzeln erhalten — die Klassifizierung
// meldet sie als übersprungen.
func legeDublettenZusammen(zeilen []Zeile, modus Modus, schluesselVon func(Zeile) string) Datei {
	datei := Datei{Modus: modus}
	platz := make(map[string]int)
	for _, z := range zeilen {
		schluessel := schluesselVon(z)
		if schluessel == "" {
			datei.Zeilen = append(datei.Zeilen, z)
			continue
		}
		if stelle, gesehen := platz[schluessel]; gesehen {
			if vorher := datei.Zeilen[stelle]; !IstKlasseGleich(vorher.Klasse, z.Klasse) {
				datei.Zusammengelegt = append(datei.Zusammengelegt, vorschauAusZeile(z.zeilenKennung(), z, vorher.Klasse, z.Klasse))
			}
			datei.Zeilen[stelle] = z
			datei.DublettenInDatei++
			continue
		}
		platz[schluessel] = len(datei.Zeilen)
		datei.Zeilen = append(datei.Zeilen, z)
	}
	return datei
}

// Datumsformate in der Schreibweise von Go (Referenzzeit "Mon Jan 2 15:04:05 MST 2006").
const (
	datumsformatDE  = "02.01.2006" // TT.MM.JJJJ
	datumsformatISO = "2006-01-02" // JJJJ-MM-TT
)
