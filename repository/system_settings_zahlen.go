package repository

// system_settings_zahlen.go — die fünfzehn Zahlen der Einstellungen und ihre Spanne.
//
// Anlass (Rasterdurchgang 16.09.2026, OFFEN.md 5.18): Bis hierher hatte jede Zahl eine
// Untergrenze und KEINE Obergrenze, und ein Wert unterhalb der Untergrenze wurde still
// durch einen Ersatzwert getauscht — gemeldet wurde „gespeichert". Zwei Folgen, beide
// gemessen:
//
//   - Eingabe 0 bei „Max. Ausleihen je Schüler" wurde als 5 gespeichert. Die Oberfläche
//     zeigte danach 5, der Mensch hatte 0 getippt, und niemand erfuhr davon.
//   - Eingabe 999999 bei „Max. überfällige Titel" wurde unverändert übernommen. Damit
//     ist die Sperr-Automatik — eine Invariante des Katalogs — aus der Oberfläche
//     abschaltbar, ohne dass irgendwo „aus" steht.
//
// Dazu protokollierte api/settings.go den Request, also die EINGABE, nicht das
// Gespeicherte: Stand 0 im Rumpf, stand 0 im Protokoll, gespeichert war 5.
//
// Die Antwort ist dieselbe, die dieselbe Datei für Sommerferien, Eingangsjahrgänge und
// LMF-Stichtag schon gibt (Rasterdurchgang 06.09.2026): Unlesbares wird ABGELEHNT, nicht
// ersetzt. Ein 400 mit dem Namen des Feldes ist die einzige Antwort, nach der Anzeige,
// Protokoll und Datenbank dasselbe sagen.
//
// Die Obergrenze ist keine Meinung über den richtigen Wert, sondern die Grenze, jenseits
// derer die Einstellung ihre Funktion verliert: 100 Bücher gleichzeitig sind keine
// Ausleihe mehr, 3650 Tage sind zehn Jahre Aufbewahrung, 1440 Minuten sind ein Tag.

import (
	"fmt"
	"strings"
)

// zahlFeld ist eine Zahl des Patches samt ihrer Spanne und der Beschriftung, unter der
// der Mensch sie in der Oberfläche kennt.
type zahlFeld struct {
	schluessel string
	wert       *int
	min, max   int
	label      string
}

// zahlenFelder ist die EINE Liste: Sie beantwortet „was wird geschrieben?" (pairsAusPatch)
// und „was ist erlaubt?" (PruefeZahlen). Getrennte Listen wären zwei Wahrheiten über
// dasselbe Feld — die Klasse, an der dieses Projekt schon zweimal hing.
//
// Untergrenze 0 heißt: Die 0 ist hier ein echter Wert („aus", „sofort"). Untergrenze 1
// heißt: Die 0 ergibt keinen Sinn (eine Frist von 0 Tagen ist keine Frist).
func zahlenFelder(p *EinstellungenPatch) []zahlFeld {
	return []zahlFeld{
		{"max_ausleihen_schueler", p.MaxAusleihenSchueler, 1, 100, "Max. Ausleihen je Schüler"},
		{"frist_buch_tage", p.FristBuchTage, 1, 365, "Leihfrist Bücher (Tage)"},
		{"frist_medien_tage", p.FristMedienTage, 1, 365, "Leihfrist Medien (Tage)"},
		// 0 = sofort sperren.
		{"max_overdue_days", p.MaxOverdueDays, 0, 365, "Karenztage bis zur Sperre"},
		// Obergrenze 100: Die Sperr-Automatik ist eine Invariante des Katalogs. Ein Wert,
		// der sie faktisch abschaltet, ist keine Einstellung, sondern ein Ausschalter —
		// und der stünde dann als solcher in der Oberfläche.
		{"max_overdue_items", p.MaxOverdueItems, 1, 100, "Max. überfällige Titel"},
		{"bestellbedarf_schwelle", p.BestellbedarfSchwelle, 1, 1000, "Schwelle Bestellbedarf"},
		{"bestelllink_gueltigkeit_tage", p.BestelllinkGueltigkeitTage, 1, 365, "Gültigkeit des Bestell-Links (Tage)"},
		// Untergrenze 1 Tag: Eine Frist von 0 Tagen wäre ein Bescheid, der am Tag des
		// Drucks bereits abgelaufen ist.
		{"bescheid_frist_tage", p.BescheidFristTage, 1, 365, "Zahlungsfrist des Bescheids (Tage)"},
		// 0 ist hier „aus" und damit ein Wert; negativ gibt es nicht.
		{"lesehistorie_tage", p.LesehistorieTage, 0, 3650, "Lesehistorie Schülerbücherei (Tage)"},
		{"lesehistorie_lernmittel_tage", p.LesehistorieLernmittelTage, 0, 3650, "Lesehistorie Lernmittel (Tage)"},
		{"anliegen_tage", p.AnliegenTage, 0, 3650, "Erledigte Anliegen (Tage)"},
		// Untergrenze 6 — anders als bei den drei Fristen darüber ist 0 hier KEIN gültiger
		// Wert: Ein abgeschaltetes Prüfprotokoll nähme dem System die Revisionsfähigkeit
		// (wer hat die Gebühr storniert?).
		{AuditAufbewahrungSchluessel, p.AuditAufbewahrungMonate, MindestAuditAufbewahrungMonate, 120, "Aufbewahrung Prüfprotokoll (Monate)"},
		{"theke_leeren_minuten", p.ThekeLeerenMinuten, 0, 1440, "Theke leeren nach (Minuten)"},
		{"sperre_minuten", p.SperreMinuten, 0, 1440, "Bildschirmsperre nach (Minuten)"},
		// 0 = sofort anonymisieren. Der destruktivste Wert des Schalters, aber ein
		// gewollter (Verhalten bis 02.09.2026) — er wird deshalb angenommen, nicht ersetzt.
		{AbgaengerKarenzSchluessel, p.AbgaengerKarenzTage, 0, 3650, "Karenz für Abgänger (Tage)"},
	}
}

// PruefeZahlen meldet jede mitgeschickte Zahl, die außerhalb ihrer Spanne liegt — alle
// auf einmal, damit der Mensch nicht nach jedem Speichern ein neues Feld erfährt.
// nil-Felder gehören nicht zu dieser Kategorie und werden nicht geprüft.
func (p *EinstellungenPatch) PruefeZahlen() error {
	var maengel []string
	for _, f := range zahlenFelder(p) {
		if f.wert == nil {
			continue
		}
		if *f.wert < f.min || *f.wert > f.max {
			maengel = append(maengel, fmt.Sprintf("%s: %d liegt außerhalb von %d bis %d",
				f.label, *f.wert, f.min, f.max))
		}
	}
	if len(maengel) == 0 {
		return nil
	}
	//nolint:staticcheck // ST1005: ganzer Satz — die Meldung steht so vor dem Menschen.
	return fmt.Errorf("Nichts gespeichert. %s.", strings.Join(maengel, "; "))
}
