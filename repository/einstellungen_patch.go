package repository

import (
	"strconv"
	"strings"

	"bibliothek/pkg/lmfplan"
)

// EinstellungenPatch ist, was ein Speichern in den Einstellungen schickt: nur die Felder der
// Kategorie, die gerade gespeichert wurde. Jedes Feld ist ein Zeiger. nil heißt „diese
// Kategorie kennt das Feld nicht, der gespeicherte Wert bleibt"; ein gesetzter Zeiger heißt
// „so speichern", auch "" und 0.
//
// Ein eigener Typ neben SystemEinstellungen, weil dort die meisten Felder Werte sind und sich
// false, 0 und "" nicht von „nicht mitgeschickt" unterscheiden lassen: Ein Speichern in einer
// Kategorie setzte sonst die Felder aller anderen zurück und meldete Erfolg.
//
// Mit omitempty trägt der Eintrag im Protokoll genau die Schlüssel, die dieses Speichern
// geändert hat.
type EinstellungenPatch struct {
	FerienLeseclubAktiv     *bool   `json:"ferien_leseclub_aktiv,omitempty"`
	FerienLeseclubZieldatum *string `json:"ferien_leseclub_zieldatum,omitempty"`
	LMFStichtag             *string `json:"lmf_stichtag,omitempty"`
	LMFEingangsjahrgaenge   *string `json:"lmf_eingangsjahrgaenge,omitempty"`
	// Sommerferien ist die JSON-Liste eigener Jahre. Die Tür bringt sie vor dem Speichern in
	// Normalform (lmfplan.NormalisiereSommerferien) und lehnt Unlesbares ab.
	Sommerferien               *string `json:"sommerferien,omitempty"`
	MaxAusleihenSchueler       *int    `json:"max_ausleihen_schueler,omitempty"`
	FristBuchTage              *int    `json:"frist_buch_tage,omitempty"`
	FristMedienTage            *int    `json:"frist_medien_tage,omitempty"`
	UeberfaelligKarenzTage     *int    `json:"max_overdue_days,omitempty"`
	UeberfaelligSperreAbMedien *int    `json:"max_overdue_items,omitempty"`

	BestellbedarfWarnungAktiv  *bool `json:"bestellbedarf_warnung_aktiv,omitempty"`
	BestellbedarfSchwelle      *int  `json:"bestellbedarf_schwelle,omitempty"`
	BestelllinkGueltigkeitTage *int  `json:"bestelllink_gueltigkeit_tage,omitempty"`
	PreiseErfassen             *bool `json:"preise_erfassen,omitempty"`

	// Name, Anschrift und Eigentumsvermerk schickt nur die Kategorie „Schule". Leer heißt dort
	// leer, damit sich ein falscher Vermerk entfernen lässt.
	SchuleName              *string `json:"schule_name,omitempty"`
	SchuleStrasse           *string `json:"schule_strasse,omitempty"`
	SchulePLZ               *string `json:"schule_plz,omitempty"`
	SchuleOrt               *string `json:"schule_ort,omitempty"`
	EtikettEigentumsvermerk *string `json:"etikett_eigentumsvermerk,omitempty"`
	// EtikettEigentumsvermerkSchuelerbuecherei ist der Vermerk für Bücher der Schülerbücherei;
	// leer heißt kein Vermerk.
	EtikettEigentumsvermerkSchuelerbuecherei *string `json:"etikett_eigentumsvermerk_schuelerbuecherei,omitempty"`

	OeffentlicheAdresse *string `json:"oeffentliche_adresse,omitempty"`
	AlarmEmpfaenger     *string `json:"alarm_empfaenger,omitempty"`

	LesehistorieTage            *int `json:"lesehistorie_tage,omitempty"`
	LesehistorieLernmittelTage  *int `json:"lesehistorie_lernmittel_tage,omitempty"`
	AnliegenTage                *int `json:"anliegen_tage,omitempty"`
	ProtokollAufbewahrungMonate *int `json:"audit_aufbewahrung_monate,omitempty"`
	ThekeLeerenMinuten          *int `json:"theke_leeren_minuten,omitempty"`
	SperreMinuten               *int `json:"sperre_minuten,omitempty"`
	AbgaengerKarenzTage         *int `json:"abgaenger_karenz_tage,omitempty"`

	// Kategorie „Schadensersatz".
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
	// ErsatzwertImmerKaufpreis: Nicht mitgeschickt heißt unangetastet, false heißt
	// ausdrücklich „Listenpreis" (siehe SystemEinstellungen).
	ErsatzwertImmerKaufpreis *bool `json:"ersatzwert_immer_kaufpreis,omitempty"`
}

// paarSammler sammelt die Paare aus Schlüssel und Wert, die ein Patch schreibt. Jedes Feld
// geht durch eine der drei Methoden; so steht die Regel für nil an einer Stelle.
type paarSammler struct{ paare [][2]string }

func (s *paarSammler) text(schluessel string, v *string) {
	if v != nil {
		s.paare = append(s.paare, [2]string{schluessel, *v})
	}
}

func (s *paarSammler) schalter(schluessel string, v *bool) {
	if v == nil {
		return
	}
	wert := "false"
	if *v {
		wert = "true"
	}
	s.paare = append(s.paare, [2]string{schluessel, wert})
}

// zahlen nimmt alle mitgeschickten Zahlen auf. Was erlaubt ist, prüft PruefeZahlen vor dem
// Speichern (einstellungen_zahlen.go); hier wird kein Wert ersetzt.
func (s *paarSammler) zahlen(p *EinstellungenPatch) {
	for _, f := range zahlenFelder(p) {
		if f.wert != nil {
			s.paare = append(s.paare, [2]string{f.schluessel, strconv.Itoa(*f.wert)})
		}
	}
}

// IstLeer meldet, dass der Patch kein einziges Feld trägt. Die Tür lehnt das ab: Eine
// Erfolgsmeldung hinterließe im Protokoll eine Änderung, die es nicht gab, und bescheinigte
// einer kaputten Oberfläche, sie habe gespeichert.
func (p *EinstellungenPatch) IstLeer() bool {
	return len(paareAusPatch(p)) == 0
}

// paareAusPatch übersetzt den Patch in Paare aus Schlüssel und Wert, nur für die Felder, die
// der Rufer mitgeschickt hat.
func paareAusPatch(p *EinstellungenPatch) [][2]string {
	s := &paarSammler{}

	s.schalter("ferien_leseclub_aktiv", p.FerienLeseclubAktiv)
	s.text("ferien_leseclub_zieldatum", p.FerienLeseclubZieldatum)
	if p.LMFStichtag != nil && *p.LMFStichtag == "" {
		s.paare = append(s.paare, [2]string{"lmf_stichtag", LMFStichtagVorgabe})
	} else {
		s.text("lmf_stichtag", p.LMFStichtag)
	}
	// Leer heißt Vorgabe, nie „niemand": Ohne Eingangsjahrgänge hätte der Ausgabe-Plan keine
	// Klasse, und vor den Ferien gäbe niemand „nur zurück".
	if p.LMFEingangsjahrgaenge != nil && strings.TrimSpace(*p.LMFEingangsjahrgaenge) == "" {
		s.paare = append(s.paare, [2]string{"lmf_eingangsjahrgaenge", LMFEingangsjahrgaengeVorgabe})
	} else {
		s.text("lmf_eingangsjahrgaenge", p.LMFEingangsjahrgaenge)
	}
	s.text(lmfplan.SommerferienSchluessel, p.Sommerferien)

	s.schalter("bestellbedarf_warnung_aktiv", p.BestellbedarfWarnungAktiv)
	s.schalter("preise_erfassen", p.PreiseErfassen)

	s.text("schule_name", p.SchuleName)
	s.text("schule_strasse", p.SchuleStrasse)
	s.text("schule_plz", p.SchulePLZ)
	s.text("schule_ort", p.SchuleOrt)
	s.text("etikett_eigentumsvermerk", p.EtikettEigentumsvermerk)
	s.text("etikett_eigentumsvermerk_schuelerbuecherei", p.EtikettEigentumsvermerkSchuelerbuecherei)

	s.text("bescheid_bereich_nr", p.BescheidBereichNr)
	s.text("bescheid_schulnummer", p.BescheidSchulnummer)
	s.text("bescheid_aufsicht", p.BescheidAufsicht)
	s.text("bescheid_schulleitung", p.BescheidSchulleitung)
	s.text("bescheid_geschaeftszeichen", p.BescheidGeschaeftszeichen)
	s.text("bescheid_bearbeiter", p.BescheidBearbeiter)
	s.text("bescheid_durchwahl", p.BescheidDurchwahl)
	s.text("bescheid_zahlstelle", p.BescheidZahlstelle)
	s.text("bescheid_bankverbindung", p.BescheidBankverbindung)
	s.schalter("ersatzwert_immer_kaufpreis", p.ErsatzwertImmerKaufpreis)
	s.text("oeffentliche_adresse", p.OeffentlicheAdresse)
	s.text("alarm_empfaenger", p.AlarmEmpfaenger)

	// Alle Zahlen auf einmal, in der Reihenfolge von zahlenFelder.
	s.zahlen(p)

	return s.paare
}
