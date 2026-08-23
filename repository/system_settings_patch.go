package repository

import "strconv"

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
	MaxAusleihenSchueler    *int    `json:"max_ausleihen_schueler,omitempty"`
	FristBuchTage           *int    `json:"frist_buch_tage,omitempty"`
	FristMedienTage         *int    `json:"frist_medien_tage,omitempty"`
	MaxOverdueDays          *int    `json:"max_overdue_days,omitempty"`
	MaxOverdueItems         *int    `json:"max_overdue_items,omitempty"`

	BestellbedarfWarnungAktiv *bool `json:"bestellbedarf_warnung_aktiv,omitempty"`
	BestellbedarfSchwelle     *int  `json:"bestellbedarf_schwelle,omitempty"`
	PreiseErfassen            *bool `json:"preise_erfassen,omitempty"`

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
	ThekeLeerenMinuten         *int `json:"theke_leeren_minuten,omitempty"`
	SperreMinuten              *int `json:"sperre_minuten,omitempty"`
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

// zahl schreibt eine Zahl; ersatz springt ein, wenn der Wert unterhalb von min liegt.
// Damit bleiben die bisherigen Vorgaben erhalten (eine 0 in „Tage/Buch" ist keine
// Frist, sondern ein leer geräumtes Feld), während min=0 die Felder kennzeichnet, in
// denen die 0 ein echter Wert ist — „sofort sperren", „Befristung aus".
func (s *paarSammler) zahl(key string, v *int, min int, ersatz int) {
	if v == nil {
		return
	}
	n := *v
	if n < min {
		n = ersatz
	}
	s.paare = append(s.paare, [2]string{key, strconv.Itoa(n)})
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
		s.paare = append(s.paare, [2]string{"lmf_stichtag", "07-31"})
	} else {
		s.text("lmf_stichtag", p.LmfStichtag)
	}
	s.zahl("max_ausleihen_schueler", p.MaxAusleihenSchueler, 1, 5)
	s.zahl("frist_buch_tage", p.FristBuchTage, 1, 21)
	s.zahl("frist_medien_tage", p.FristMedienTage, 1, 7)
	s.zahl("max_overdue_days", p.MaxOverdueDays, 0, 0)
	s.zahl("max_overdue_items", p.MaxOverdueItems, 1, 1)

	s.schalter("bestellbedarf_warnung_aktiv", p.BestellbedarfWarnungAktiv)
	s.zahl("bestellbedarf_schwelle", p.BestellbedarfSchwelle, 1, 3)
	s.schalter("preise_erfassen", p.PreiseErfassen)

	s.text("schule_name", p.SchuleName)
	s.text("schule_strasse", p.SchuleStrasse)
	s.text("schule_plz", p.SchulePLZ)
	s.text("schule_ort", p.SchuleOrt)
	s.text("etikett_eigentumsvermerk", p.EtikettEigentumsvermerk)

	s.text("oeffentliche_adresse", p.OeffentlicheAdresse)
	s.text("alarm_empfaenger", p.AlarmEmpfaenger)

	// 0 ist hier „aus" und damit ein Wert; negativ gibt es nicht.
	s.zahl("lesehistorie_tage", p.LesehistorieTage, 0, 0)
	s.zahl("lesehistorie_lernmittel_tage", p.LesehistorieLernmittelTage, 0, 0)
	s.zahl("anliegen_tage", p.AnliegenTage, 0, 0)
	s.zahl("theke_leeren_minuten", p.ThekeLeerenMinuten, 0, 0)
	s.zahl("sperre_minuten", p.SperreMinuten, 0, 0)

	return s.paare
}
