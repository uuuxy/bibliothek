package repository

import "strings"

// Die Angaben, die der Schadensersatz-Bescheid braucht und die nur die Schule kennt.
//
// Warum einzelne Schlüssel und kein JSON-Block: Der Patch-Mechanismus unterscheidet je
// Feld zwischen „nicht mitgeschickt" (bleibt) und „geleert" (leer). Ein Block wäre ein
// Feld, das man ganz oder nicht schreibt — und dann löscht das Speichern einer Zeile
// still die anderen acht.
//
// Vorgaben, wo es welche gibt: Die Schule muss nur die beiden Nummern und die
// Aufsichtsbehörde eintragen; Frist, Zahlstelle und Bankverbindung stehen vorbelegt
// aus dem Musterschreiben. Was leer bleiben darf, bleibt im Brief einfach weg.
const (
	// BescheidFristTageVorgabe: Die Vierwochenfrist des Verfahrens, als Tage ab Briefdatum.
	// Sie MUSS im Brief ein Datum sein, keine Zeitspanne („bis zum 15.05.", nicht „bis
	// Mitte Mai") — gerechnet wird sie beim Erstellen.
	BescheidFristTageVorgabe = 28
	// BescheidZahlstelleVorgabe ist der Empfänger, den das Musterschreiben nennt.
	BescheidZahlstelleVorgabe = "HCC-Schulbereich bei der Landesbank Hessen-Thüringen"
	// BescheidBankverbindungVorgabe sind die vier Zeilen des Musterschreibens. Mehrzeilig,
	// weil sie im Brief als Block untereinander stehen.
	BescheidBankverbindungVorgabe = "Konto-Nr. 1002401\nBankleitzahl 500 500 00\nIBAN DE86500500000001002401\nBIC HELADEFFXXX"
)

// BescheidAngaben bündelt die Einstellungen des Bescheids für Brief und Prüfung.
type BescheidAngaben struct {
	// BereichNr und Schulnummer bilden mit Kassenjahr und laufender Nummer die
	// Referenznummer. Beide vierstellig; ohne sie gibt es keinen Bescheid, weil sich
	// eine Zahlung ohne Referenz nicht zuordnen lässt.
	BereichNr   string
	Schulnummer string
	// Aufsicht: Name und Anschrift der Stelle, an die der Fall nach Fristablauf geht.
	// Sie steht an zwei Stellen im Brief und ist deshalb Pflicht.
	Aufsicht string
	// Schulleitung: die Unterschriftszeile.
	Schulleitung string
	// Freiwillig, dürfen leer bleiben — im Muster sind es graue Felder.
	Geschaeftszeichen string
	Bearbeiter        string
	Durchwahl         string
	// Zahlstelle und Bankverbindung: wohin gezahlt wird.
	Zahlstelle     string
	Bankverbindung string
	// FristTage: Vorgabe BescheidFristTageVorgabe.
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

// FehlendeAngaben nennt die Pflichtangaben, die noch fehlen — in der Sprache der
// Oberfläche, damit die Meldung sagt, was einzutragen ist.
//
// EINE Quelle für beide Verbraucher: die Tür, die den Bescheid erstellt (sie weist ab),
// und die Selbstprüfung der Betriebsbereitschaft (sie warnt vorher). Zwei getrennte
// Listen wären zwei Wahrheiten, und eine davon wäre irgendwann falsch.
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

// SchuleAngaben ist der Ausschnitt der Schul-Identität, den die Prüfung braucht — als
// eigener Typ, damit repository nicht auf das pdf-Paket zeigen muss.
type SchuleAngaben struct {
	Name    string
	Strasse string
	PLZ     string
	Ort     string
}

// SchuleAngabenAus zieht sie aus den Einstellungen.
func SchuleAngabenAus(e *SystemEinstellungen) SchuleAngaben {
	if e == nil {
		return SchuleAngaben{}
	}
	return SchuleAngaben{Name: e.SchuleName, Strasse: e.SchuleStrasse, PLZ: e.SchulePLZ, Ort: e.SchuleOrt}
}

// Referenznummer baut die Nummer des Briefs: Schulamtsbereich, Kassenjahr, Schulnummer,
// laufende Nummer — je vierstellig, durch Leerzeichen getrennt, wie im Musterschreiben
// („5830 2015 1234 0001").
func (a BescheidAngaben) Referenznummer(kassenjahr, laufendeNr int) string {
	return strings.Join([]string{
		vierstellig(a.BereichNr),
		itoa4(kassenjahr),
		vierstellig(a.Schulnummer),
		itoa4(laufendeNr),
	}, " ")
}

// vierstellig füllt eine eingetippte Nummer links mit Nullen auf. Wer „583" einträgt,
// meint 0583 — abschneiden würde die Zuordnung der Zahlung zerstören.
func vierstellig(s string) string {
	s = strings.TrimSpace(s)
	for len(s) < 4 {
		s = "0" + s
	}
	return s
}

// itoa4 formatiert eine Zahl vierstellig mit führenden Nullen.
func itoa4(n int) string {
	if n < 0 {
		n = 0
	}
	ziffern := []byte{byte('0' + (n/1000)%10), byte('0' + (n/100)%10), byte('0' + (n/10)%10), byte('0' + n%10)}
	return string(ziffern)
}
