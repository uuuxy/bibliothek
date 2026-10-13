package repository

import "strings"

// Die Angaben, die der Schadensersatz-Bescheid braucht und die nur die Schule kennt.
//
// Es sind einzelne Schlüssel und kein JSON-Block, weil der Patch je Feld zwischen „nicht
// mitgeschickt" (bleibt) und „geleert" unterscheidet. Ein Block wäre ein Feld, das man ganz
// oder gar nicht schreibt: Das Speichern einer Zeile löschte die anderen.
//
// Welche Angaben Pflicht sind, nennt FehlendeAngaben. Frist, Zahlstelle und Bankverbindung
// sind aus dem Musterschreiben vorbelegt; was leer bleiben darf, fehlt im Brief.
const (
	// BescheidFristTageVorgabe ist die Vierwochenfrist des Verfahrens in Tagen ab dem Datum
	// des Briefs. Im Brief steht sie als Datum („bis zum 15.05."), gerechnet beim Erstellen.
	BescheidFristTageVorgabe = 28
	// BescheidZahlstelleVorgabe ist der Empfänger, den das Musterschreiben nennt.
	BescheidZahlstelleVorgabe = "HCC-Schulbereich bei der Landesbank Hessen-Thüringen"
	// BescheidBankverbindungVorgabe sind die vier Zeilen des Musterschreibens; im Brief stehen
	// sie als Block untereinander.
	BescheidBankverbindungVorgabe = "Konto-Nr. 1002401\nBankleitzahl 500 500 00\nIBAN DE86500500000001002401\nBIC HELADEFFXXX"
)

// setzeBescheidEinstellung überträgt eine Zeile in ihr Feld und meldet, ob der Schlüssel zum
// Bescheid gehört.
func setzeBescheidEinstellung(e *SystemEinstellungen, schluessel string, wert *string) bool {
	switch schluessel {
	case "bescheid_bereich_nr":
		setzeStringRoh(wert, &e.BescheidBereichNr)
	case "bescheid_schulnummer":
		setzeStringRoh(wert, &e.BescheidSchulnummer)
	case "bescheid_aufsicht":
		setzeStringRoh(wert, &e.BescheidAufsicht)
	case "bescheid_schulleitung":
		setzeStringRoh(wert, &e.BescheidSchulleitung)
	case "bescheid_geschaeftszeichen":
		setzeStringRoh(wert, &e.BescheidGeschaeftszeichen)
	case "bescheid_bearbeiter":
		setzeStringRoh(wert, &e.BescheidBearbeiter)
	case "bescheid_durchwahl":
		setzeStringRoh(wert, &e.BescheidDurchwahl)
	case "bescheid_zahlstelle":
		setzeStringRoh(wert, &e.BescheidZahlstelle)
	case "bescheid_bankverbindung":
		setzeStringRoh(wert, &e.BescheidBankverbindung)
	case "bescheid_frist_tage":
		setzeIntZeiger(wert, &e.BescheidFristTage)
	default:
		return false
	}
	return true
}

// BescheidAngaben bündelt die Einstellungen des Bescheids für Brief und Prüfung.
type BescheidAngaben struct {
	// BereichNr und Schulnummer bilden mit Kassenjahr und laufender Nummer die
	// Referenznummer. Beide sind vierstellig; ohne sie gibt es keinen Bescheid, weil sich eine
	// Zahlung ohne Referenz nicht zuordnen lässt.
	BereichNr   string
	Schulnummer string
	// Aufsicht nennt Name und Anschrift der Stelle, an die der Fall nach Ablauf der Frist
	// geht. Sie steht an zwei Stellen im Brief und ist deshalb Pflicht.
	Aufsicht string
	// Schulleitung ist die Zeile unter der Unterschrift.
	Schulleitung string
	// Geschäftszeichen, Bearbeiter und Durchwahl sind freiwillig und dürfen leer bleiben.
	Geschaeftszeichen string
	Bearbeiter        string
	Durchwahl         string
	// Zahlstelle und Bankverbindung sagen, wohin gezahlt wird.
	Zahlstelle     string
	Bankverbindung string
	// FristTage hat die Vorgabe BescheidFristTageVorgabe.
	FristTage int
}

// BescheidAngabenAus liest die Angaben aus den Einstellungen und setzt die Vorgaben ein.
func BescheidAngabenAus(e *SystemEinstellungen) BescheidAngaben {
	if e == nil {
		return BescheidAngaben{
			Zahlstelle:     BescheidZahlstelleVorgabe,
			Bankverbindung: BescheidBankverbindungVorgabe,
			FristTage:      BescheidFristTageVorgabe,
		}
	}
	a := BescheidAngaben{
		BereichNr:         strings.TrimSpace(e.BescheidBereichNr),
		Schulnummer:       strings.TrimSpace(e.BescheidSchulnummer),
		Aufsicht:          strings.TrimSpace(e.BescheidAufsicht),
		Schulleitung:      strings.TrimSpace(e.BescheidSchulleitung),
		Geschaeftszeichen: strings.TrimSpace(e.BescheidGeschaeftszeichen),
		Bearbeiter:        strings.TrimSpace(e.BescheidBearbeiter),
		Durchwahl:         strings.TrimSpace(e.BescheidDurchwahl),
		Zahlstelle:        strings.TrimSpace(e.BescheidZahlstelle),
		Bankverbindung:    strings.TrimSpace(e.BescheidBankverbindung),
		FristTage:         BescheidFristTageVorgabe,
	}
	if a.Zahlstelle == "" {
		a.Zahlstelle = BescheidZahlstelleVorgabe
	}
	if a.Bankverbindung == "" {
		a.Bankverbindung = BescheidBankverbindungVorgabe
	}
	if e.BescheidFristTage != nil && *e.BescheidFristTage > 0 {
		a.FristTage = *e.BescheidFristTage
	}
	return a
}

// FehlendeAngaben nennt die Pflichtangaben, die noch fehlen, in den Worten der Oberfläche:
// Die Meldung sagt, was einzutragen ist. Die Tür, die den Bescheid erstellt, und die
// Selbstprüfung fragen beide hier; mit zwei Listen wiese die eine ab, wovor die andere nicht
// gewarnt hat.
func (a BescheidAngaben) FehlendeAngaben(schule SchuleAngaben) []string {
	var fehlt []string
	if a.BereichNr == "" {
		fehlt = append(fehlt, "Nummer des Schulamtsbereichs")
	}
	if a.Schulnummer == "" {
		fehlt = append(fehlt, "Schulnummer")
	}
	if a.Aufsicht == "" {
		fehlt = append(fehlt, "Aufsichtsbehörde (Name und Anschrift)")
	}
	if a.Schulleitung == "" {
		fehlt = append(fehlt, "Name der Schulleitung")
	}
	if schule.Name == "" || schule.Strasse == "" || schule.Ort == "" {
		fehlt = append(fehlt, "Anschrift der Schule (Einstellungen → Schule)")
	}
	return fehlt
}

// SchuleAngaben sind Name und Anschrift der Schule, soweit die Prüfung sie braucht. Ein
// eigener Typ, damit repository nicht auf das Paket pdf zeigt.
type SchuleAngaben struct {
	Name    string
	Strasse string
	PLZ     string
	Ort     string
}

// SchuleAngabenAus liest Name und Anschrift der Schule aus den Einstellungen.
func SchuleAngabenAus(e *SystemEinstellungen) SchuleAngaben {
	if e == nil {
		return SchuleAngaben{}
	}
	return SchuleAngaben{Name: e.SchuleName, Strasse: e.SchuleStrasse, PLZ: e.SchulePLZ, Ort: e.SchuleOrt}
}

// Referenznummer baut die Nummer des Briefs aus Schulamtsbereich, Kassenjahr, Schulnummer und
// laufender Nummer, je vierstellig und durch Leerzeichen getrennt wie im Musterschreiben
// („5830 2015 1234 0001").
func (a BescheidAngaben) Referenznummer(kassenjahr, laufendeNr int) string {
	return strings.Join([]string{
		vierstellig(a.BereichNr),
		zahlVierstellig(kassenjahr),
		vierstellig(a.Schulnummer),
		zahlVierstellig(laufendeNr),
	}, " ")
}

// vierstellig füllt eine eingetippte Nummer links mit Nullen auf. Wer „583" einträgt, meint
// 0583; abschneiden zerstörte die Zuordnung der Zahlung.
func vierstellig(s string) string {
	s = strings.TrimSpace(s)
	for len(s) < 4 {
		s = "0" + s
	}
	return s
}

// zahlVierstellig schreibt eine Zahl vierstellig mit führenden Nullen.
func zahlVierstellig(n int) string {
	if n < 0 {
		n = 0
	}
	ziffern := []byte{byte('0' + (n/1000)%10), byte('0' + (n/100)%10), byte('0' + (n/10)%10), byte('0' + n%10)}
	return string(ziffern)
}
