package repository

import (
	"bibliothek/db"
	"context"
	"strconv"
)

// SystemEinstellungen holds configurable system-wide settings.
type SystemEinstellungen struct {
	FerienLeseclubAktiv     bool    `json:"ferien_leseclub_aktiv"`
	FerienLeseclubZieldatum *string `json:"ferien_leseclub_zieldatum"` // ISO date string "YYYY-MM-DD" or null
	LmfStichtag             string  `json:"lmf_stichtag"`              // "MM-DD" format, e.g. "07-31"
	MaxAusleihenSchueler    int     `json:"max_ausleihen_schueler"`
	FristBuchTage           int     `json:"frist_buch_tage"`
	FristMedienTage         int     `json:"frist_medien_tage"`
	MaxOverdueDays          int     `json:"max_overdue_days"`
	MaxOverdueItems         int     `json:"max_overdue_items"`
	// Bestellbedarf: ob überhaupt gewarnt wird und ab welcher Exemplarzahl ein
	// (LMF-)Titel als Bestellbedarf gilt (gesamt < Schwelle). Löst den früheren
	// pauschalen Meldebestand-Default 5 ab, der fast jeden Titel fälschlich meldete.
	BestellbedarfWarnungAktiv bool `json:"bestellbedarf_warnung_aktiv"`
	BestellbedarfSchwelle     int  `json:"bestellbedarf_schwelle"`
	// PreiseErfassen entscheidet, ob das Bestellwesen mit Geld arbeitet.
	//
	// Aus heisst: kein Preisfeld im Warenkorb, keine Betragsspalten in der Historie, und
	// die Berichte zaehlen Exemplare statt Euro zu summieren. Der Schalter steuert
	// ERFASSUNG UND ANZEIGE, nicht die Daten — bereits erfasste Betraege bleiben in der
	// Datenbank und tauchen wieder auf, sobald er zurueckgelegt wird.
	//
	// Anlass: Ohne gepflegte Preise summieren Bestellhistorie und alle drei Berichte
	// Nullen. Spalten voller 0,00 EUR sind schlimmer als keine Spalten — sie sehen aus
	// wie ein Nachweis und sind keiner.
	PreiseErfassen bool `json:"preise_erfassen"`
	// School identity — used in PDF letter headers (set once via settings UI).
	SchuleName    string `json:"schule_name"`
	SchuleStrasse string `json:"schule_strasse"`
	SchulePLZ     string `json:"schule_plz"`
	SchuleOrt     string `json:"schule_ort"`
	// EtikettEigentumsvermerk steht als letzte Zeile auf jedem Buchetikett
	// ("Eigentum des Landes Hessen"). Konfigurierbar und nicht fest verdrahtet, weil
	// der Träger je nach Bundesland und Schulform ein anderer ist — und weil ein
	// Eigentumsvermerk, der nicht stimmt, schlechter ist als keiner.
	EtikettEigentumsvermerk string `json:"etikett_eigentumsvermerk"`
	// OeffentlicheAdresse ist die Adresse, unter der DRITTE dieses System erreichen —
	// z. B. "https://bibliothek.schule.de". Aus ihr entsteht der Bestätigungs-Link, den
	// der Lieferant mit der Bestellmail bekommt.
	//
	// Der Server kennt sie nicht von selbst: Hinter einem Reverse-Proxy sieht er nur
	// seinen internen Namen, und ein Link auf "localhost:8080" wäre beim Lieferanten
	// wertlos. Zeiger statt String, damit die Unterscheidung "nicht mitgeschickt"
	// (nil, Wert bleibt) und "geleert" ("") erhalten bleibt — sonst löscht jedes
	// Speichern einer anderen Einstellungs-Sektion still den Link-Versand.
	OeffentlicheAdresse *string `json:"oeffentliche_adresse"`
	// AlarmEmpfaenger: kommaseparierte Adressen fuer die Kritisch-Alarme der
	// Selbstpruefung. Leer = alle aktiven Admin-Konten (sicherer Rueckfall — ein
	// Alarm, der niemanden erreicht, ist keiner). Betreiber-Wunsch 17.08.2026.
	AlarmEmpfaenger *string `json:"alarm_empfaenger"`
	// Datenschutz (A1): Tage nach der Rückgabe, nach denen die Ausleihe vom Schüler getrennt
	// wird (schueler_id = NULL) — getrennt für Schülerbücherei und Lernmittel. 0 = nie.
	// Sitzung (A4): Minuten Inaktivität bis Theke leeren bzw. Sperrbildschirm. 0 = aus.
	// Zeiger, Vorgaben und Begründung: system_settings_datenschutz.go.
	LesehistorieTage           *int `json:"lesehistorie_tage"`
	LesehistorieLernmittelTage *int `json:"lesehistorie_lernmittel_tage"`
	ThekeLeerenMinuten         *int `json:"theke_leeren_minuten"`
	SperreMinuten              *int `json:"sperre_minuten"`
}

// StandardEigentumsvermerk greift, solange in den Einstellungen nichts hinterlegt ist.
// Die Vorgabe passt zum Betreiber dieses Systems; jede Schule kann sie überschreiben.
const StandardEigentumsvermerk = "Eigentum des Landes Hessen"

// SystemSettingsRepository defines operations for managing global system settings.
type SystemSettingsRepository interface {
	GetSettings(ctx context.Context) (*SystemEinstellungen, error)
	SaveSettings(ctx context.Context, patch *EinstellungenPatch) error
}

type pgSystemSettingsRepository struct {
	db db.PgxPoolIface
}

// NewSystemSettingsRepository returns a PostgreSQL implementation of SystemSettingsRepository.
func NewSystemSettingsRepository(db db.PgxPoolIface) SystemSettingsRepository {
	return &pgSystemSettingsRepository{db: db}
}

// standardEinstellungen liefert die Default-Konfiguration (Fallback, wenn ein
// Wert nicht in der DB steht).
func standardEinstellungen() *SystemEinstellungen {
	s := &SystemEinstellungen{
		LmfStichtag:          "07-31",
		MaxAusleihenSchueler: 5,
		FristBuchTage:        21,
		FristMedienTage:      7,
		MaxOverdueDays:       14,
		MaxOverdueItems:      1,
		// Warnung standardmäßig an; Schwelle 3 (statt des früheren Default 5) als
		// ruhigerer Startwert — der Betreiber justiert sie in den Einstellungen.
		BestellbedarfWarnungAktiv: true,
		BestellbedarfSchwelle:     3,
		// An als Vorgabe: Das ist das bisherige Verhalten, und wer Preise fuehrt,
		// soll sie nach einem Update nicht ploetzlich vermissen.
		PreiseErfassen: true,
	}
	datenschutzStandards(s)
	return s
}

func setzeIntEinstellung(val *string, ziel *int) {
	if val == nil || *val == "" {
		return
	}
	if v, err := strconv.Atoi(*val); err == nil {
		*ziel = v
	}
}

func setzeStringNichtLeer(val *string, ziel *string) {
	if val != nil && *val != "" {
		*ziel = *val
	}
}

func setzeStringRoh(val *string, ziel *string) {
	if val != nil {
		*ziel = *val
	}
}

// applyEinstellung überträgt einen einzelnen Key/Value-Eintrag aus der DB auf
// die Settings-Struktur.
func applyEinstellung(settings *SystemEinstellungen, key string, val *string) {
	if anwendenDatenschutzEinstellung(settings, key, val) {
		return
	}
	switch key {
	case "ferien_leseclub_aktiv":
		settings.FerienLeseclubAktiv = val != nil && *val == "true"
	case "ferien_leseclub_zieldatum":
		if val != nil && *val != "" {
			v := *val
			settings.FerienLeseclubZieldatum = &v
		}
	case "lmf_stichtag":
		setzeStringNichtLeer(val, &settings.LmfStichtag)
	case "max_ausleihen_schueler":
		setzeIntEinstellung(val, &settings.MaxAusleihenSchueler)
	case "frist_buch_tage":
		setzeIntEinstellung(val, &settings.FristBuchTage)
	case "frist_medien_tage":
		setzeIntEinstellung(val, &settings.FristMedienTage)
	case "max_overdue_days":
		setzeIntEinstellung(val, &settings.MaxOverdueDays)
	case "max_overdue_items":
		setzeIntEinstellung(val, &settings.MaxOverdueItems)
	case "bestellbedarf_warnung_aktiv":
		settings.BestellbedarfWarnungAktiv = val != nil && *val == "true"
	case "bestellbedarf_schwelle":
		setzeIntEinstellung(val, &settings.BestellbedarfSchwelle)
	case "preise_erfassen":
		// Gegen "false" statt fuer "true": Die Vorgabe ist AN, ein fehlender oder
		// unlesbarer Wert darf die Preise also nicht abschalten.
		settings.PreiseErfassen = val == nil || *val != "false"
	case "schule_name":
		setzeStringRoh(val, &settings.SchuleName)
	case "schule_strasse":
		setzeStringRoh(val, &settings.SchuleStrasse)
	case "schule_plz":
		setzeStringRoh(val, &settings.SchulePLZ)
	case "schule_ort":
		setzeStringRoh(val, &settings.SchuleOrt)
	case "etikett_eigentumsvermerk":
		setzeStringRoh(val, &settings.EtikettEigentumsvermerk)
	case "alarm_empfaenger":
		if val != nil {
			v := *val
			settings.AlarmEmpfaenger = &v
		}
	case "oeffentliche_adresse":
		if val != nil {
			v := *val
			settings.OeffentlicheAdresse = &v
		}
	}
}

// GetSettings reads system settings from the database.
func (repo *pgSystemSettingsRepository) GetSettings(ctx context.Context) (*SystemEinstellungen, error) {
	settings := standardEinstellungen()

	rows, err := repo.db.Query(ctx, `SELECT schluessel, wert FROM system_einstellungen`)
	if err != nil {
		return settings, err
	}
	defer rows.Close()

	for rows.Next() {
		var key string
		var val *string
		if scanErr := rows.Scan(&key, &val); scanErr != nil {
			continue
		}
		applyEinstellung(settings, key, val)
	}
	return settings, rows.Err()
}

// SaveSettings schreibt die Felder EINES Patches — und nur die. Was der Aufrufer
// nicht mitgeschickt hat, bleibt in der Datenbank unangetastet (siehe
// system_settings_patch.go).
func (repo *pgSystemSettingsRepository) SaveSettings(ctx context.Context, patch *EinstellungenPatch) error {
	pairs := pairsAusPatch(patch)
	if len(pairs) == 0 {
		return nil
	}

	upsert := `
		INSERT INTO system_einstellungen (schluessel, wert)
		SELECT * FROM UNNEST($1::varchar[], $2::text[])
		ON CONFLICT (schluessel) DO UPDATE
		  SET wert = EXCLUDED.wert, aktualisiert_am = CURRENT_TIMESTAMP
	`

	schluessels := make([]string, len(pairs))
	werts := make([]string, len(pairs))
	for i, p := range pairs {
		schluessels[i] = p[0]
		werts[i] = p[1]
	}

	_, err := repo.db.Exec(ctx, upsert, schluessels, werts)
	return err
}
