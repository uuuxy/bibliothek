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
}

// zahlenFelder ist die eine Liste für „was wird geschrieben?" (paareAusPatch) und „was ist
// erlaubt?" (PruefeZahlen); mit zwei Listen liefen die beiden Antworten auseinander.
//
// Untergrenze 0 heißt: Die 0 ist hier ein Wert („aus", „sofort"). Untergrenze 1 heißt: Die 0
// ergibt keinen Sinn, eine Frist von 0 Tagen ist keine Frist.
func zahlenFelder(p *EinstellungenPatch) []zahlFeld {
	return []zahlFeld{
		{"max_ausleihen_schueler", p.MaxAusleihenSchueler, 1, 100, "Max. Ausleihen je Schüler"},
		{"frist_buch_tage", p.FristBuchTage, 1, 365, "Leihfrist Bücher (Tage)"},
		{"frist_medien_tage", p.FristMedienTage, 1, 365, "Leihfrist Medien (Tage)"},
		// 0 heißt sofort sperren.
		{"max_overdue_days", p.UeberfaelligKarenzTage, 0, 365, "Karenztage bis zur Sperre"},
		// Obergrenze 100: Die Sperre wegen Überfälligkeit ist eine Regel des Katalogs. Ein
		// Wert, der sie abschaltet, wäre ein Ausschalter und stünde als solcher in der
		// Oberfläche.
		{"max_overdue_items", p.UeberfaelligSperreAbMedien, 1, 100, "Max. überfällige Titel"},
		{"bestellbedarf_schwelle", p.BestellbedarfSchwelle, 1, 1000, "Schwelle Bestellbedarf"},
		{"bestelllink_gueltigkeit_tage", p.BestelllinkGueltigkeitTage, 1, 365, "Gültigkeit des Bestell-Links (Tage)"},
		// Untergrenze 1 Tag: Mit 0 Tagen wäre der Bescheid am Tag des Drucks abgelaufen.
		{"bescheid_frist_tage", p.BescheidFristTage, 1, 365, "Zahlungsfrist des Bescheids (Tage)"},
		// 0 heißt hier „aus" und ist damit ein Wert; negativ gibt es nicht.
		{"lesehistorie_tage", p.LesehistorieTage, 0, 3650, "Lesehistorie Schülerbücherei (Tage)"},
		{"lesehistorie_lernmittel_tage", p.LesehistorieLernmittelTage, 0, 3650, "Lesehistorie Lernmittel (Tage)"},
		{"anliegen_tage", p.AnliegenTage, 0, 3650, "Erledigte Anliegen (Tage)"},
		// Untergrenze 6, anders als bei den drei Fristen darüber: Ohne Protokoll ließe sich
		// nicht mehr prüfen, wer etwa eine Gebühr storniert hat.
		{ProtokollAufbewahrungSchluessel, p.ProtokollAufbewahrungMonate, ProtokollAufbewahrungMonateMinimum, 120, "Aufbewahrung Prüfprotokoll (Monate)"},
		{"theke_leeren_minuten", p.ThekeLeerenMinuten, 0, 1440, "Theke leeren nach (Minuten)"},
		{"sperre_minuten", p.SperreMinuten, 0, 1440, "Bildschirmsperre nach (Minuten)"},
		// 0 heißt sofort anonymisieren: der schärfste Wert des Schalters, aber ein gewollter.
		// Er wird angenommen und nicht ersetzt.
		{AbgaengerKarenzSchluessel, p.AbgaengerKarenzTage, 0, 3650, "Karenz für Abgänger (Tage)"},
	}
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
			maengel = append(maengel, fmt.Sprintf("%s: %d liegt außerhalb von %d bis %d",
				f.beschriftung, *f.wert, f.min, f.max))
		}
	}
	if len(maengel) == 0 {
		return nil
	}
	//nolint:staticcheck // ST1005: ganzer Satz — die Meldung steht so vor dem Menschen.
	return fmt.Errorf("Nichts gespeichert. %s.", strings.Join(maengel, "; "))
}
