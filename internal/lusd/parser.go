package lusd

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/xuri/excelize/v2"
)

// Zeile ist eine gelesene Zeile des Exports: ein Schüler, wie die LUSD ihn nennt. LineNum
// ist die Zeilennummer in der Datei; an ihr hängt die Wahl der Umbenennungen zwischen
// Vorschau und Import.
type Zeile struct {
	LusdID      string
	Vorname     string
	Nachname    string
	Klasse      string
	GebDatum    *time.Time
	Strasse     string
	Hausnummer  string
	PLZ         string
	Ort         string
	ElternEmail string
	// EintrittAm ist der Schuleintritt laut Bericht (Schueler_Eintritt_AktuelleSchule) —
	// der zweite Schlüssel der Umbenennungs-Paarung (paarung.go); nil ohne Spalte.
	EintrittAm *time.Time
	LineNum    int
	// geburtsdatumUebernehmen: nur für bestätigte Umbenennungs-Paare — der Bestands-Batch
	// schreibt dann das Geburtsdatum des Exports (Datumskorrektur der LUSD).
	geburtsdatumUebernehmen bool
}

// schluessel ist der Name+Geburtsdatum-Schlüssel der Zeile ("" ohne Datum).
func (r Zeile) schluessel() string {
	return waisenSchluessel(r.Vorname, r.Nachname, r.GebDatum)
}

// namensschluessel ist der Nur-Name-Schlüssel (Vorname + Nachname, normalisiert).
func (r Zeile) namensschluessel() string {
	return namensSchluessel(r.Vorname, r.Nachname)
}

// schluesselFuer liefert den Nachschlage-Schlüssel dieser Zeile für den gegebenen Modus —
// das Gegenstück zu bestandsSchluessel (bestand.go), das dasselbe für die Bestandszeile
// tut. Zwei Seiten, EINE Regel: Solange beide über den Modus gehen, kann niemand versehentlich
// den Namensschlüssel gegen den Name+Datum-Index schlagen (das matchte still niemanden).
func (r Zeile) schluesselFuer(modus Modus) string {
	if modus == ModusNurName {
		return r.namensschluessel()
	}
	return r.schluessel()
}

const (
	lusdColID           = "lusd_id"
	lusdColVorname      = "vorname"
	lusdColNachname     = "nachname"
	lusdColKlasse       = "klasse"
	lusdColGeburtsdatum = "geburtsdatum"
	// Optionale Kontakt-/Adressspalten. Fehlen sie im Export, bleibt der Import
	// gültig; die Felder sind dann leer. Zweck: Schadens-Rechnung (Anschrift) und
	// Elternkontakt (E-Mail). Header müssen exakt (case-insensitiv) so heißen.
	lusdColStrasse     = "strasse"
	lusdColHausnummer  = "hausnummer"
	lusdColPLZ         = "plz"
	lusdColOrt         = "ort"
	lusdColElternEmail = "eltern_email"
	// lusdColEintritt: Eintritt an der aktuellen Schule. Optional; ohne die Spalte bleibt
	// die Umbenennungs-Paarung auf Geburtsdatum, Klasse, Name und Anschrift angewiesen.
	lusdColEintritt = "eintritt"
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

// String ist der Wert, den die Vorschau dem Frontend meldet.
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

// Datei ist das Parse-Ergebnis: die Zeilen plus der Modus, in dem sie zugeordnet
// werden, plus das, was beim Zusammenlegen doppelter Zeilen verloren ging.
type Datei struct {
	Zeilen []Zeile
	Modus  Modus
	// DublettenInDatei zählt Zeilen, die auf denselben Schlüssel fielen und von der
	// späteren überschrieben wurden (ID- und Name+Geburtsdatum-Modus: letzte gewinnt).
	// Im Nur-Name-Modus wird NICHT zusammengelegt — gleiche Namen sind dort mehrdeutig.
	DublettenInDatei int
	// Zusammengelegt nennt die Dubletten, deren Zeilen sich in der KLASSE unterschieden
	// (AlteKlasse = überschriebene Zeile, NeueKlasse = gewonnene, ID = Zeilennummer der
	// gewonnenen). Im Name+Geburtsdatum-Modus kann das ein Schulformwechsler sein, den
	// LUSD doppelt führt — oder zwei Schüler mit demselben Namen und einem Tippfehler im
	// Datum. Die Zahl allein (DublettenInDatei) verschwieg den Fall; die Vorschau nennt
	// ihn beim Namen (OFFEN.md 5.6, 22.09.2026). Gleiche Klasse ist eine bloße
	// Doppelzeile und bleibt eine Zahl.
	Zusammengelegt []StudentDiff
}

// spaltenWert liest eine optionale Spalte getrimmt aus; fehlt sie oder ist die
// Zeile zu kurz, wird "" zurückgegeben.
func spaltenWert(row []string, headerMap map[string]int, col string) string {
	idx, ok := headerMap[col]
	if !ok || idx >= len(row) {
		return ""
	}
	return strings.TrimSpace(row[idx])
}

var excelSerienzahl = regexp.MustCompile(`^\d{4,6}(\.\d+)?$`)

// parseLUSDDatum liest eine Datumsspalte (Geburtsdatum, Schuleintritt). Excel liefert
// Datumszellen als Serienzahl (Tage seit 1899-12-30), CSV als Text in mehreren
// Schreibweisen. nil, wenn die Spalte fehlt oder der Wert unlesbar ist.
func parseLUSDDatum(row []string, headerMap map[string]int, col string) *time.Time {
	raw := spaltenWert(row, headerMap, col)
	if raw == "" {
		return nil
	}
	if excelSerienzahl.MatchString(raw) {
		if serie, err := strconv.ParseFloat(raw, 64); err == nil {
			if t, err := excelize.ExcelDateToTime(serie, false); err == nil {
				t = time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
				return &t
			}
		}
		return nil
	}
	for _, layout := range []string{dateFormatDE, "2.1.2006", dateFormatISO, "01/02/2006", "2006-01-02T15:04:05Z07:00", "2006-01-02 15:04:05"} {
		if t, parseErr := time.ParseInLocation(layout, raw, time.UTC); parseErr == nil {
			t = time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
			return &t
		}
	}
	return nil
}

// plausiblerEintritt verwirft Platzhalter- und Unsinnsdaten im Schuleintritt: vor 1990,
// in der Zukunft oder nicht nach dem Geburtsdatum. Ein Platzhalter wie 01.01.1900 in
// jeder Zeile machte sonst jeden Namensvetter zum „sicheren" Paar (Rasterdurchgang
// 02.09.2026, Sonde des Prüfers) — und bestätigt überschriebe der Batch das Geburtsdatum.
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

// parseLUSDRow parst eine Datenzeile und validiert die Pflichtfelder. Alle Spalten
// laufen über spaltenWert, weil ID- und Geburtsdatum-Spalte fehlen dürfen.
func parseLUSDRow(row []string, headerMap map[string]int, lineNum int) (Zeile, error) {
	vorname := spaltenWert(row, headerMap, lusdColVorname)
	nachname := spaltenWert(row, headerMap, lusdColNachname)
	klasse := spaltenWert(row, headerMap, lusdColKlasse)

	if vorname == "" || nachname == "" || klasse == "" {
		return Zeile{}, fmt.Errorf("zeile %d enthält ein leeres Pflichtfeld (Vorname/Nachname/Klasse)", lineNum)
	}

	geb := parseLUSDDatum(row, headerMap, lusdColGeburtsdatum)
	return Zeile{
		LusdID:      spaltenWert(row, headerMap, lusdColID),
		Vorname:     vorname,
		Nachname:    nachname,
		Klasse:      klasse,
		GebDatum:    geb,
		EintrittAm:  plausiblerEintritt(parseLUSDDatum(row, headerMap, lusdColEintritt), geb),
		Strasse:     spaltenWert(row, headerMap, lusdColStrasse),
		Hausnummer:  spaltenWert(row, headerMap, lusdColHausnummer),
		PLZ:         spaltenWert(row, headerMap, lusdColPLZ),
		Ort:         spaltenWert(row, headerMap, lusdColOrt),
		ElternEmail: spaltenWert(row, headerMap, lusdColElternEmail),
		LineNum:     lineNum,
	}, nil
}

// istLeereZeile: Excel-Berichte enden oft mit Leer- oder Summenzeilen; eine Zeile ohne
// jeden Wert ist kein Schüler und kein Fehler.
func istLeereZeile(row []string) bool {
	for _, c := range row {
		if strings.TrimSpace(c) != "" {
			return false
		}
	}
	return true
}

// ParseDatei liest die Datei (CSV oder Excel) vollständig, bestimmt den Modus und
// legt doppelte Zeilen zusammen. Harte Fehler statt stillem Überspringen: Das
// Sekretariat soll eine kaputte Datei als Meldung sehen, nicht als halb importierten
// Bestand.
func ParseDatei(content []byte) (Datei, error) {
	rows, err := leseLusdTabelle(content)
	if err != nil {
		return Datei{}, err
	}
	kopfIdx, headerMap, err := findeKopfzeile(rows)
	if err != nil {
		return Datei{}, err
	}

	var zeilen []Zeile
	irgendeineID, irgendeinDatum := false, false
	for i := kopfIdx + 1; i < len(rows); i++ {
		if istLeereZeile(rows[i].zellen) {
			continue
		}
		sRow, err := parseLUSDRow(rows[i].zellen, headerMap, rows[i].nr)
		if err != nil {
			return Datei{}, err
		}
		irgendeineID = irgendeineID || sRow.LusdID != ""
		irgendeinDatum = irgendeinDatum || sRow.GebDatum != nil
		zeilen = append(zeilen, sRow)
	}

	// Eine Spalte, in der kein einziger Wert steht, zählt nicht — LUSD exportiert
	// Spalten mitunter leer. Was die Datei wirklich hergibt, bestimmt den Modus.
	if _, hatIDSpalte := headerMap[lusdColID]; hatIDSpalte && irgendeineID {
		return legeDublettenZusammen(zeilen, ModusID, func(r Zeile) string { return r.LusdID }), nil
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
		if z.GebDatum == nil {
			// Nur die Zeilennummer, kein Name: Die Meldung geht über SendHTTPError auch ins
			// Server-Log. Die Zeile findet das Sekretariat in der Datei.
			return fmt.Errorf("zeile %d: Geburtsdatum fehlt oder ist unlesbar — ohne LUSD-ID ist Name + Geburtsdatum der Zuordnungsschlüssel, er muss in jeder Zeile stehen", z.LineNum)
		}
	}
	return nil
}

// legeDublettenZusammen lässt von mehreren Zeilen mit demselben Schlüssel die LETZTE
// gewinnen (an ihrem ersten Platz) und zählt, was dabei überschrieben wurde. Zeilen
// ohne Schlüssel (ID-Modus: leere ID) bleiben einzeln erhalten — die Klassifizierung
// meldet sie als übersprungen.
func legeDublettenZusammen(zeilen []Zeile, modus Modus, schluessel func(Zeile) string) Datei {
	datei := Datei{Modus: modus}
	platz := make(map[string]int)
	for _, z := range zeilen {
		key := schluessel(z)
		if key == "" {
			datei.Zeilen = append(datei.Zeilen, z)
			continue
		}
		if idx, gesehen := platz[key]; gesehen {
			if vorher := datei.Zeilen[idx]; !KlassenGleich(vorher.Klasse, z.Klasse) {
				datei.Zusammengelegt = append(datei.Zusammengelegt, diffZeile(z.zeilenKennung(), z, vorher.Klasse, z.Klasse))
			}
			datei.Zeilen[idx] = z
			datei.DublettenInDatei++
			continue
		}
		platz[key] = len(datei.Zeilen)
		datei.Zeilen = append(datei.Zeilen, z)
	}
	return datei
}
