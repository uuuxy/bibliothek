package repository

// Die fünfzehn Zahlen der Einstellungen und ihre Spanne.
//
// Eine Zahl außerhalb ihrer Spanne wird abgelehnt und nicht ersetzt: Nur dann sagen Anzeige,
// Protokoll und Datenbank dasselbe. Ein stiller Ersatzwert speicherte 5, wo der Mensch 0
// getippt hatte, und meldete „gespeichert".
//
// Die Obergrenze ist keine Meinung über den richtigen Wert, sondern die Grenze, jenseits derer
// die Einstellung ihre Aufgabe verliert: 100 Bücher gleichzeitig sind keine Ausleihe mehr,
// 3650 Tage sind zehn Jahre Aufbewahrung, 1440 Minuten sind ein Tag.

import (
	"fmt"
	"strings"
)

// zahlFeld ist eine Zahl des Patches mit ihrer Spanne und der Beschriftung, unter der die
// Oberfläche sie zeigt.
type zahlFeld struct {
	schluessel   string
	wert         *int
	min, max     int
	beschriftung string
	// einheit steht hinter der Spanne im Satz („1 bis 365 Tagen“), wie das Formular sie rechts im
	// Feld zeigt.
	einheit string
}

// zahlenFelder ist die eine Liste für „was wird geschrieben?" (paareAusPatch) und „was ist
// erlaubt?" (PruefeZahlen); mit zwei Listen liefen die beiden Antworten auseinander.
//
// Untergrenze 0 heißt: Die 0 ist hier ein Wert („aus", „sofort"). Untergrenze 1 heißt: Die 0
// ergibt keinen Sinn, eine Frist von 0 Tagen ist keine Frist.
func zahlenFelder(p *EinstellungenPatch) []zahlFeld {
	return []zahlFeld{
		{"max_ausleihen_schueler", p.MaxAusleihenSchueler, 1, 100, "Max. Ausleihen", ""},
		{"frist_buch_tage", p.FristBuchTage, 1, 365, "Leihfrist Bücher", "Tagen"},
		{"frist_medien_tage", p.FristMedienTage, 1, 365, "Leihfrist Medien", "Tagen"},
		{"frist_geraete_tage", p.FristGeraeteTage, 1, 365, "Leihfrist Geräte", "Tagen"},
		// 0 heißt sofort sperren.
		{"max_overdue_days", p.UeberfaelligKarenzTage, 0, 365, "Tage bis Sperre", ""},
		// Obergrenze 100: Die Sperre wegen Überfälligkeit ist eine Regel des Katalogs. Ein
		// Wert, der sie abschaltet, wäre ein Ausschalter und stünde als solcher in der
		// Oberfläche.
		{"max_overdue_items", p.UeberfaelligSperreAbMedien, 1, 100, "Ab x Medien sperren", ""},
		{"bestelllink_gueltigkeit_tage", p.BestelllinkGueltigkeitTage, 1, 365, "Bestätigungs-Link gültig", "Tagen"},
		// Untergrenze 1 Tag: Mit 0 Tagen wäre der Bescheid am Tag des Drucks abgelaufen.
		{"bescheid_frist_tage", p.BescheidFristTage, 1, 365, "Zahlungsfrist", "Tagen"},
		// 0 heißt hier „aus" und ist damit ein Wert; negativ gibt es nicht.
		{"lesehistorie_tage", p.LesehistorieTage, 0, 3650, "Lesehistorie Schülerbücherei", "Tagen"},
		{"lesehistorie_lernmittel_tage", p.LesehistorieLernmittelTage, 0, 3650, "Lesehistorie Lernmittel", "Tagen"},
		{"anliegen_tage", p.AnliegenTage, 0, 3650, "Erledigte Anliegen aufbewahren", "Tagen"},
		// Untergrenze 6, anders als bei den drei Fristen darüber: Ohne Protokoll ließe sich
		// nicht mehr prüfen, wer etwa eine Gebühr storniert hat.
		{ProtokollAufbewahrungSchluessel, p.ProtokollAufbewahrungMonate, protokollAufbewahrungMonateMinimum, 120, "Prüfprotokoll aufbewahren", "Monaten"},
		{"theke_leeren_minuten", p.ThekeLeerenMinuten, 0, 1440, "Theke leeren nach", "Minuten"},
		{"sperre_minuten", p.SperreMinuten, 0, 1440, "Sperrbildschirm nach", "Minuten"},
		// 0 heißt sofort anonymisieren: der schärfste Wert des Schalters, aber ein gewollter.
		// Er wird angenommen und nicht ersetzt.
		{AbgaengerKarenzSchluessel, p.AbgaengerKarenzTage, 0, 3650, "Abgänger-Karenzzeit", "Tagen"},
	}
}

// ZahlenAusserhalbFehler meldet Zahlen der Einstellungen außerhalb ihrer Spanne. Ein Mangel nennt
// das Feld, wie es das Formular zeigt, den Wert und die Spanne; die Tür setzt die Meldung daraus
// zusammen.
type ZahlenAusserhalbFehler struct{ Maengel []string }

func (e *ZahlenAusserhalbFehler) Error() string {
	return "zahlen außerhalb ihrer spanne: " + strings.Join(e.Maengel, "; ")
}

// PruefeZahlen meldet jede mitgeschickte Zahl außerhalb ihrer Spanne, alle auf einmal, damit
// der Mensch nicht nach jedem Speichern ein neues Feld erfährt. Ein Feld mit nil gehört nicht
// zur gespeicherten Kategorie und wird nicht geprüft.
func (p *EinstellungenPatch) PruefeZahlen() error {
	var maengel []string
	for _, f := range zahlenFelder(p) {
		if f.wert == nil {
			continue
		}
		if *f.wert < f.min || *f.wert > f.max {
			spanne := fmt.Sprintf("%d bis %d", f.min, f.max)
			if f.einheit != "" {
				spanne += " " + f.einheit
			}
			maengel = append(maengel, fmt.Sprintf("%s: %d liegt außerhalb von %s", f.beschriftung, *f.wert, spanne))
		}
	}
	if len(maengel) == 0 {
		return nil
	}
	return &ZahlenAusserhalbFehler{Maengel: maengel}
}
