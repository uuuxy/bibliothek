package repository

import (
	"strconv"
	"strings"

	"bibliothek/pkg/lmfplan"
)

// EinstellungenPatch ist das, was ein Speichern-Klick schickt: NUR die Felder der
// Kategorie, die gerade gespeichert wurde. Jedes Feld ist ein Zeiger, und nil heißt
// ausnahmslos „diese Kategorie kennt das Feld nicht — den gespeicherten Wert stehen
// lassen". Ein gesetzter Zeiger heißt „so speichern", auch auf "" und auf 0.
//
// Warum ein eigener Typ neben SystemEinstellungen (das die GET-Antwort bleibt): Dort
// sind die meisten Felder Werte statt Zeiger, und false/0/"" ist vom „gar nicht
// mitgeschickt" nicht zu unterscheiden. buildSettingsPairs schrieb deshalb ELF
// Schlüssel bei JEDEM Speichern, gleich was im Rumpf stand. Solange die Oberfläche
// immer alles auf einmal schickte, fiel das nicht auf. Mit einem Speichern je
// Kategorie (23.08.2026) wäre es Datenverlust gewesen: Ein Speichern in „Datenschutz
// & Sitzung" hätte Ferien-Leseclub, Bestellbedarf-Warnung und die Preiserfassung
// ausgeschaltet und fünf Fristen auf die Vorgabe zurückgesetzt — mit einer grünen
// Erfolgsmeldung. Dieselbe Bugklasse wie das Upsert-Blanking beim Import.
//
// omitempty ist hier nicht Kosmetik: Der Audit-Eintrag (UPDATE_SETTINGS) trägt damit
// genau die Schlüssel, die dieser Klick geändert hat, statt aller mit null.
type EinstellungenPatch struct {
	FerienLeseclubAktiv     *bool   `json:"ferien_leseclub_aktiv,omitempty"`
	FerienLeseclubZieldatum *string `json:"ferien_leseclub_zieldatum,omitempty"`
	LmfStichtag             *string `json:"lmf_stichtag,omitempty"`
	LmfEingangsjahrgaenge   *string `json:"lmf_eingangsjahrgaenge,omitempty"`
	// Sommerferien: JSON-Liste eigener Jahre; der Handler bringt sie vor dem Speichern in
	// Normalform (lmfplan.NormalisiereSommerferien) und lehnt Unlesbares mit 400 ab.
	Sommerferien         *string `json:"sommerferien,omitempty"`
	MaxAusleihenSchueler *int    `json:"max_ausleihen_schueler,omitempty"`
	FristBuchTage        *int    `json:"frist_buch_tage,omitempty"`
	FristMedienTage      *int    `json:"frist_medien_tage,omitempty"`
	MaxOverdueDays       *int    `json:"max_overdue_days,omitempty"`
	MaxOverdueItems      *int    `json:"max_overdue_items,omitempty"`

	BestellbedarfWarnungAktiv  *bool `json:"bestellbedarf_warnung_aktiv,omitempty"`
	BestellbedarfSchwelle      *int  `json:"bestellbedarf_schwelle,omitempty"`
	BestelllinkGueltigkeitTage *int  `json:"bestelllink_gueltigkeit_tage,omitempty"`
	PreiseErfassen             *bool `json:"preise_erfassen,omitempty"`

	// Schul-Identität. Bis zum 23.08.2026 hieß ein leeres Feld hier „nicht anfassen" —
	// eine Notbremse gegen das Blanking oben, die aber zugleich das Löschen unmöglich
	// machte (ein falscher Eigentumsvermerk ließ sich nicht mehr entfernen). Mit dem
	// Speichern je Kategorie schickt nur noch die Kategorie „Schule" diese Felder,
	// und dann heißt leer wieder schlicht leer.
	SchuleName              *string `json:"schule_name,omitempty"`
	SchuleStrasse           *string `json:"schule_strasse,omitempty"`
	SchulePLZ               *string `json:"schule_plz,omitempty"`
	SchuleOrt               *string `json:"schule_ort,omitempty"`
	EtikettEigentumsvermerk *string `json:"etikett_eigentumsvermerk,omitempty"`

	OeffentlicheAdresse *string `json:"oeffentliche_adresse,omitempty"`
	AlarmEmpfaenger     *string `json:"alarm_empfaenger,omitempty"`

	LesehistorieTage           *int `json:"lesehistorie_tage,omitempty"`
	LesehistorieLernmittelTage *int `json:"lesehistorie_lernmittel_tage,omitempty"`
	AnliegenTage               *int `json:"anliegen_tage,omitempty"`
	AuditAufbewahrungMonate    *int `json:"audit_aufbewahrung_monate,omitempty"`
	ThekeLeerenMinuten         *int `json:"theke_leeren_minuten,omitempty"`
	SperreMinuten              *int `json:"sperre_minuten,omitempty"`
	AbgaengerKarenzTage        *int `json:"abgaenger_karenz_tage,omitempty"`

	// Kategorie „Schadensersatz" (Migration 110).
	BescheidBereichNr         *string `json:"bescheid_bereich_nr,omitempty"`
	BescheidSchulnummer       *string `json:"bescheid_schulnummer,omitempty"`
	BescheidAufsicht          *string `json:"bescheid_aufsicht,omitempty"`
	BescheidSchulleitung      *string `json:"bescheid_schulleitung,omitempty"`
	BescheidGeschaeftszeichen *string `json:"bescheid_geschaeftszeichen,omitempty"`
	BescheidBearbeiter        *string `json:"bescheid_bearbeiter,omitempty"`
	BescheidDurchwahl         *string `json:"bescheid_durchwahl,omitempty"`
	BescheidZahlstelle        *string `json:"bescheid_zahlstelle,omitempty"`
	BescheidBankverbindung    *string `json:"bescheid_bankverbindung,omitempty"`
	BescheidFristTage         *int    `json:"bescheid_frist_tage,omitempty"`
}

// paarSammler sammelt die Upsert-Paare eines Patches. Jede Hinzufügung geht durch
// eine der drei Methoden — damit steht die nil-Regel an EINER Stelle statt
// zweiundzwanzig Mal als `if x != nil`.
type paarSammler struct{ paare [][2]string }

func (s *paarSammler) text(key string, v *string) {
	if v != nil {
		s.paare = append(s.paare, [2]string{key, *v})
	}
}

func (s *paarSammler) schalter(key string, v *bool) {
	if v == nil {
		return
	}
	wert := "false"
	if *v {
		wert = "true"
	}
	s.paare = append(s.paare, [2]string{key, wert})
}

// zahlen schreibt alle mitgeschickten Zahlen. Was erlaubt ist, steht in
// system_settings_zahlen.go und wird VOR dem Speichern geprüft (PruefeZahlen) — hier
// wird nichts mehr ersetzt: Ein stiller Ersatzwert war genau der Fehler, den der
// Rasterdurchgang am 16.09.2026 gefunden hat.
func (s *paarSammler) zahlen(p *EinstellungenPatch) {
	for _, f := range zahlenFelder(p) {
		if f.wert != nil {
			s.paare = append(s.paare, [2]string{f.schluessel, strconv.Itoa(*f.wert)})
		}
	}
}

// IstLeer meldet, dass der Patch kein einziges Feld trägt.
//
// Das ist ein Aufruferfehler und keine leere Speicherung: Der Handler antwortet
// darauf mit 400 statt mit „ok". Ein 200 auf einen Rumpf ohne Felder hätte einen
// Audit-Eintrag „UPDATE_SETTINGS" hinterlassen, der eine Änderung behauptet, die nie
// stattgefunden hat — und einer kaputten Oberfläche bescheinigt, sie habe gespeichert.
func (p *EinstellungenPatch) IstLeer() bool {
	return len(pairsAusPatch(p)) == 0
}

// pairsAusPatch übersetzt den Patch in Upsert-Paare — ausschließlich für Felder, die
// der Aufrufer tatsächlich mitgeschickt hat.
func pairsAusPatch(p *EinstellungenPatch) [][2]string {
	s := &paarSammler{}

	s.schalter("ferien_leseclub_aktiv", p.FerienLeseclubAktiv)
	s.text("ferien_leseclub_zieldatum", p.FerienLeseclubZieldatum)
	if p.LmfStichtag != nil && *p.LmfStichtag == "" {
		s.paare = append(s.paare, [2]string{"lmf_stichtag", StandardLmfStichtag})
	} else {
		s.text("lmf_stichtag", p.LmfStichtag)
	}
	// Leer heißt Vorgabe, nie „niemand": Ohne Eingangsjahrgänge hätte der Ausgabe-Plan
	// keine Klasse und vor den Ferien gäbe niemand „nur zurück".
	if p.LmfEingangsjahrgaenge != nil && strings.TrimSpace(*p.LmfEingangsjahrgaenge) == "" {
		s.paare = append(s.paare, [2]string{"lmf_eingangsjahrgaenge", LmfEingangsjahrgaengeVorgabe})
	} else {
		s.text("lmf_eingangsjahrgaenge", p.LmfEingangsjahrgaenge)
	}
	s.text(lmfplan.SommerferienSchluessel, p.Sommerferien)

	s.schalter("bestellbedarf_warnung_aktiv", p.BestellbedarfWarnungAktiv)
	s.schalter("preise_erfassen", p.PreiseErfassen)

	s.text("schule_name", p.SchuleName)
	s.text("schule_strasse", p.SchuleStrasse)
	s.text("schule_plz", p.SchulePLZ)
	s.text("schule_ort", p.SchuleOrt)
	s.text("etikett_eigentumsvermerk", p.EtikettEigentumsvermerk)

	s.text("bescheid_bereich_nr", p.BescheidBereichNr)
	s.text("bescheid_schulnummer", p.BescheidSchulnummer)
	s.text("bescheid_aufsicht", p.BescheidAufsicht)
	s.text("bescheid_schulleitung", p.BescheidSchulleitung)
	s.text("bescheid_geschaeftszeichen", p.BescheidGeschaeftszeichen)
	s.text("bescheid_bearbeiter", p.BescheidBearbeiter)
	s.text("bescheid_durchwahl", p.BescheidDurchwahl)
	s.text("bescheid_zahlstelle", p.BescheidZahlstelle)
	s.text("bescheid_bankverbindung", p.BescheidBankverbindung)
	s.text("oeffentliche_adresse", p.OeffentlicheAdresse)
	s.text("alarm_empfaenger", p.AlarmEmpfaenger)

	// Alle Zahlen auf einmal, in der Reihenfolge von zahlenFelder().
	s.zahlen(p)

	return s.paare
}
