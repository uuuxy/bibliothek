package repository

import (
	"context"
	"strconv"
	"strings"

	"bibliothek/db"
	"bibliothek/pkg/bestelllink"
	"bibliothek/pkg/lmfplan"

	"github.com/jackc/pgx/v5"
)

// SystemEinstellungen ist, was die Verwaltung einstellt und die Oberfläche als Ganzes liest.
// Ein gespeichertes Feld steht als eigene Zeile in system_einstellungen, unter dem Namen
// seines JSON-Felds.
type SystemEinstellungen struct {
	FerienLeseclubAktiv     bool    `json:"ferien_leseclub_aktiv"`
	FerienLeseclubZieldatum *string `json:"ferien_leseclub_zieldatum"` // Kalendertag „JJJJ-MM-TT" oder nil
	LMFStichtag             string  `json:"lmf_stichtag"`              // „MM-TT", etwa „07-31"
	// LMFEingangsjahrgaenge nennt die Jahrgänge, die ihre Bücher erst nach den Sommerferien
	// bekommen, weil ihre Klassen neu gebildet werden („5, 7"; EingangsjahrgaengeAus).
	LMFEingangsjahrgaenge string `json:"lmf_eingangsjahrgaenge"`
	// Sommerferien hält die eigenen Jahre der Schule als JSON-Liste und verlängert die Tabelle
	// des Programms (pkg/lmfplan/ferien_einstellung.go); leer heißt, nur die Tabelle gilt.
	// SommerferienProgramm ist diese Tabelle selbst. Sie wird nur gelesen, damit die Oberfläche
	// zeigt, was schon da ist.
	Sommerferien         string                        `json:"sommerferien"`
	SommerferienProgramm []lmfplan.SommerferienEintrag `json:"sommerferien_programm"`
	MaxAusleihenSchueler int                           `json:"max_ausleihen_schueler"`
	FristBuchTage        int                           `json:"frist_buch_tage"`
	FristMedienTage      int                           `json:"frist_medien_tage"`
	// UeberfaelligKarenzTage und UeberfaelligSperreAbMedien steuern die Sperre wegen
	// Überfälligkeit: Gesperrt ist, wer mindestens so viele Medien länger als die Karenz
	// überfällig hat.
	UeberfaelligKarenzTage     int `json:"max_overdue_days"`
	UeberfaelligSperreAbMedien int `json:"max_overdue_items"`
	// BestellbedarfWarnungAktiv schaltet die Warnung; BestellbedarfSchwelle nennt die Zahl der
	// Exemplare, unter der ein Titel als Bestellbedarf gilt.
	BestellbedarfWarnungAktiv bool `json:"bestellbedarf_warnung_aktiv"`
	BestellbedarfSchwelle     int  `json:"bestellbedarf_schwelle"`
	// BestelllinkGueltigkeitTage ist die Lebensdauer des Links zur Bestätigung, den der
	// Hauptlieferant mit der Bestellmail bekommt (Vorgabe und Begründung in pkg/bestelllink).
	// Sie gilt für neu erzeugte Links; laufende behalten ihr Ablaufdatum.
	BestelllinkGueltigkeitTage int `json:"bestelllink_gueltigkeit_tage"`
	// PreiseErfassen entscheidet, ob das Bestellwesen mit Geld arbeitet. Aus heißt: kein
	// Preisfeld im Warenkorb, keine Spalten mit Beträgen, und die Berichte zählen Exemplare.
	// Der Schalter steuert Erfassung und Anzeige, nicht die Daten: Erfasste Beträge bleiben
	// gespeichert und erscheinen wieder, sobald er zurückgelegt wird. Ohne gepflegte Preise
	// summierten Historie und Berichte Nullen, und eine Spalte voller 0,00 EUR sieht aus wie
	// ein Nachweis.
	PreiseErfassen bool `json:"preise_erfassen"`
	// Name und Anschrift der Schule stehen im Kopf der Briefe.
	SchuleName    string `json:"schule_name"`
	SchuleStrasse string `json:"schule_strasse"`
	SchulePLZ     string `json:"schule_plz"`
	SchuleOrt     string `json:"schule_ort"`
	// EtikettEigentumsvermerk steht als letzte Zeile auf jedem Buchetikett. Er ist einstellbar,
	// weil der Träger je nach Bundesland und Schulform ein anderer ist und ein Vermerk, der
	// nicht stimmt, schlechter ist als keiner.
	EtikettEigentumsvermerk string `json:"etikett_eigentumsvermerk"`
	// EtikettEigentumsvermerkSchuelerbuecherei steht an derselben Stelle auf Büchern aus
	// Mitteln des Schulträgers (ExemplarTopfSQL). Hier gibt es keine Vorgabe des Programms:
	// Wem diese Bücher gehören, weiß nur die Schule, und leer heißt kein Vermerk.
	EtikettEigentumsvermerkSchuelerbuecherei string `json:"etikett_eigentumsvermerk_schuelerbuecherei"`
	// OeffentlicheAdresse ist die Adresse, unter der Dritte das Programm erreichen, etwa
	// „https://bibliothek.schule.de". Aus ihr entsteht der Link zur Bestätigung in der
	// Bestellmail. Der Server kennt sie nicht von selbst: Hinter einem Reverse-Proxy sieht er
	// nur seinen internen Namen. Ein Zeiger, damit „nicht mitgeschickt" (nil, der Wert bleibt)
	// und „geleert" ("") verschieden bleiben.
	OeffentlicheAdresse *string `json:"oeffentliche_adresse"`
	// AlarmEmpfaenger nennt, durch Komma getrennt, die Adressen für die Alarme der
	// Selbstprüfung. Leer heißt alle aktiven Admin-Konten: Ein Alarm, der niemanden erreicht,
	// ist keiner.
	AlarmEmpfaenger *string `json:"alarm_empfaenger"`
	// Fristen des Datenschutzes und der Sitzung, je als Zeiger; Vorgaben und Begründung stehen
	// in einstellungen_datenschutz.go. Die Lesehistorie zählt die Tage nach der Rückgabe, nach
	// denen eine Ausleihe vom Leser getrennt wird, je für Schülerbücherei und Lernmittel; Theke
	// und Sperre zählen Minuten ohne Eingabe. 0 schaltet eine Frist ab, nur die Aufbewahrung des
	// Protokolls hat eine Untergrenze (protokollAufbewahrungMonateMinimum).
	LesehistorieTage            *int `json:"lesehistorie_tage"`
	LesehistorieLernmittelTage  *int `json:"lesehistorie_lernmittel_tage"`
	AnliegenTage                *int `json:"anliegen_tage"`
	ProtokollAufbewahrungMonate *int `json:"audit_aufbewahrung_monate"`
	ThekeLeerenMinuten          *int `json:"theke_leeren_minuten"`
	SperreMinuten               *int `json:"sperre_minuten"`
	// AbgaengerKarenzTage sind die Tage nach dem Abgang, die ein Abgänger ohne offene Vorgänge
	// nur gesperrt bleibt, bevor er anonymisiert wird. 0 heißt sofort.
	AbgaengerKarenzTage *int `json:"abgaenger_karenz_tage"`

	// Angaben für den Schadensersatz-Bescheid. Vorgaben, die Prüfung der Pflichtangaben und der
	// Aufbau der Referenznummer stehen in einstellungen_bescheid.go, dort auch, warum es
	// einzelne Schlüssel sind und kein Block.
	BescheidBereichNr         string `json:"bescheid_bereich_nr"`
	BescheidSchulnummer       string `json:"bescheid_schulnummer"`
	BescheidAufsicht          string `json:"bescheid_aufsicht"`
	BescheidSchulleitung      string `json:"bescheid_schulleitung"`
	BescheidGeschaeftszeichen string `json:"bescheid_geschaeftszeichen"`
	BescheidBearbeiter        string `json:"bescheid_bearbeiter"`
	BescheidDurchwahl         string `json:"bescheid_durchwahl"`
	BescheidZahlstelle        string `json:"bescheid_zahlstelle"`
	BescheidBankverbindung    string `json:"bescheid_bankverbindung"`
	BescheidFristTage         *int   `json:"bescheid_frist_tage"`
	// ErsatzwertImmerKaufpreis legt fest, dass ab dem zweiten Verleihjahr der Einkaufspreis
	// der Schule gilt statt des heutigen Listenpreises. Die Frage ist so gestellt, dass Nein
	// die Vorgabe ist: Ein nicht gesetzter Schlüssel liest sich als false, und false ist die
	// Regel der Arbeitshilfe („80 % des Neupreises zum Zeitpunkt des Verlusts").
	ErsatzwertImmerKaufpreis bool `json:"ersatzwert_immer_kaufpreis"`
}

// AdresseFuerLinks liefert die öffentliche Adresse ohne Leerraum am Rand, "" wenn keine
// hinterlegt ist. Ein Feld aus lauter Leerzeichen ist keine Adresse. Wer einen Link nach außen
// baut oder prüft, ob einer entstehen kann, fragt hier: Mit je eigener Prüfung hielt eine Tür
// für eine Adresse, woraus die nächste keinen Link baute.
func (s *SystemEinstellungen) AdresseFuerLinks() string {
	if s == nil || s.OeffentlicheAdresse == nil {
		return ""
	}
	return strings.TrimSpace(*s.OeffentlicheAdresse)
}

// EigentumsvermerkVorgabe steht auf dem Etikett, solange die Schule nichts eingetragen hat.
// Sie passt zum Betreiber dieser Anlage; jede Schule kann sie überschreiben.
const EigentumsvermerkVorgabe = "Eigentum des Landes Hessen"

// EinstellungsAbfragen liest und schreibt die Einstellungen der Verwaltung.
type EinstellungsAbfragen interface {
	Lade(ctx context.Context) (*SystemEinstellungen, error)
	Schreibe(ctx context.Context, patch *EinstellungenPatch) error
}

type pgEinstellungsAbfragen struct {
	db db.Pool
}

// NewEinstellungsAbfragen liefert die Abfragen der Einstellungen an Postgres.
func NewEinstellungsAbfragen(pool db.Pool) EinstellungsAbfragen {
	return &pgEinstellungsAbfragen{db: pool}
}

// einstellungenVorgabe liefert die Werte, die gelten, solange zu einem Schlüssel keine Zeile
// gespeichert ist.
func einstellungenVorgabe() *SystemEinstellungen {
	e := &SystemEinstellungen{
		LMFStichtag:                LMFStichtagVorgabe,
		LMFEingangsjahrgaenge:      lmfEingangsjahrgaengeVorgabe,
		SommerferienProgramm:       lmfplan.ProgrammEintraege(),
		MaxAusleihenSchueler:       5,
		FristBuchTage:              21,
		FristMedienTage:            7,
		UeberfaelligKarenzTage:     14,
		UeberfaelligSperreAbMedien: 1,
		// Die Warnung ist an; die Schwelle 3 hält sie ruhig, bis die Schule sie einstellt.
		BestellbedarfWarnungAktiv:  true,
		BestellbedarfSchwelle:      3,
		BestelllinkGueltigkeitTage: bestelllink.TageVorgabe,
		// An: Wer Preise führt, soll sie nach einem Update nicht vermissen.
		PreiseErfassen: true,
	}
	setzeDatenschutzVorgaben(e)
	return e
}

// setzeInt übernimmt eine gespeicherte Zahl; ein leerer oder unlesbarer Wert lässt die Vorgabe
// stehen.
func setzeInt(wert *string, ziel *int) {
	if wert == nil || *wert == "" {
		return
	}
	if n, err := strconv.Atoi(*wert); err == nil {
		*ziel = n
	}
}

// setzeStringNichtLeer übernimmt einen gespeicherten Text; ein leerer lässt die Vorgabe stehen.
func setzeStringNichtLeer(wert *string, ziel *string) {
	if wert != nil && *wert != "" {
		*ziel = *wert
	}
}

// setzeStringRoh übernimmt einen gespeicherten Text, auch den leeren.
func setzeStringRoh(wert *string, ziel *string) {
	if wert != nil {
		*ziel = *wert
	}
}

// setzeEinstellung überträgt eine Zeile aus system_einstellungen in ihr Feld. Einen Schlüssel,
// den hier niemand kennt, übergeht sie.
func setzeEinstellung(e *SystemEinstellungen, schluessel string, wert *string) {
	if setzeDatenschutzEinstellung(e, schluessel, wert) || setzeBescheidEinstellung(e, schluessel, wert) {
		return
	}
	switch schluessel {
	case "ersatzwert_immer_kaufpreis":
		e.ErsatzwertImmerKaufpreis = wert != nil && *wert == "true"
	case "ferien_leseclub_aktiv":
		e.FerienLeseclubAktiv = wert != nil && *wert == "true"
	case "ferien_leseclub_zieldatum":
		if wert != nil && *wert != "" {
			v := *wert
			e.FerienLeseclubZieldatum = &v
		}
	case "lmf_stichtag":
		setzeStringNichtLeer(wert, &e.LMFStichtag)
	case "lmf_eingangsjahrgaenge":
		setzeStringNichtLeer(wert, &e.LMFEingangsjahrgaenge)
	case lmfplan.SommerferienSchluessel:
		setzeStringRoh(wert, &e.Sommerferien)
	case "max_ausleihen_schueler":
		setzeInt(wert, &e.MaxAusleihenSchueler)
	case "frist_buch_tage":
		setzeInt(wert, &e.FristBuchTage)
	case "frist_medien_tage":
		setzeInt(wert, &e.FristMedienTage)
	case "max_overdue_days":
		setzeInt(wert, &e.UeberfaelligKarenzTage)
	case "max_overdue_items":
		setzeInt(wert, &e.UeberfaelligSperreAbMedien)
	case "bestellbedarf_warnung_aktiv":
		e.BestellbedarfWarnungAktiv = wert != nil && *wert == "true"
	case "bestellbedarf_schwelle":
		setzeInt(wert, &e.BestellbedarfSchwelle)
	case "bestelllink_gueltigkeit_tage":
		setzeInt(wert, &e.BestelllinkGueltigkeitTage)
	case "preise_erfassen":
		// Gegen „false" geprüft statt für „true": Die Vorgabe ist an, und ein fehlender oder
		// unlesbarer Wert darf die Preise nicht abschalten.
		e.PreiseErfassen = wert == nil || *wert != "false"
	case "schule_name":
		setzeStringRoh(wert, &e.SchuleName)
	case "schule_strasse":
		setzeStringRoh(wert, &e.SchuleStrasse)
	case "schule_plz":
		setzeStringRoh(wert, &e.SchulePLZ)
	case "schule_ort":
		setzeStringRoh(wert, &e.SchuleOrt)
	case "etikett_eigentumsvermerk":
		setzeStringRoh(wert, &e.EtikettEigentumsvermerk)
	case "etikett_eigentumsvermerk_schuelerbuecherei":
		setzeStringRoh(wert, &e.EtikettEigentumsvermerkSchuelerbuecherei)
	case "alarm_empfaenger":
		if wert != nil {
			v := *wert
			e.AlarmEmpfaenger = &v
		}
	case "oeffentliche_adresse":
		if wert != nil {
			v := *wert
			e.OeffentlicheAdresse = &v
		}
	}
}

// Lade liest die Einstellungen. Bei einem Lesefehler liefert es die Vorgaben des Programms
// zusammen mit dem Fehler; der Rufer verweigert dann die Antwort oder wählt die Vorgabe
// ausdrücklich.
func (a *pgEinstellungsAbfragen) Lade(ctx context.Context) (*SystemEinstellungen, error) {
	return LadeEinstellungen(ctx, a.db)
}

// LadeEinstellungen liest die Einstellungen über q, den Pool oder die Transaktion des Rufers,
// und liefert bei einem Lesefehler dasselbe wie Lade. Die Abholfrist liest so die Sommerferien
// über den Zugang, auf dem sie die Vormerkung zuteilt.
func LadeEinstellungen(ctx context.Context, q interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
}) (*SystemEinstellungen, error) {
	e := einstellungenVorgabe()

	rows, err := q.Query(ctx, `SELECT schluessel, wert FROM system_einstellungen`)
	if err != nil {
		return e, err
	}
	defer rows.Close()

	for rows.Next() {
		var schluessel string
		var wert *string
		if scanErr := rows.Scan(&schluessel, &wert); scanErr != nil {
			continue
		}
		setzeEinstellung(e, schluessel, wert)
	}
	return e, rows.Err()
}

// Schreibe schreibt die Felder eines Patches und nur diese. Was der Rufer nicht mitgeschickt
// hat, bleibt in der Datenbank, wie es ist (einstellungen_patch.go).
func (a *pgEinstellungsAbfragen) Schreibe(ctx context.Context, patch *EinstellungenPatch) error {
	paare := paareAusPatch(patch)
	if len(paare) == 0 {
		return nil
	}

	upsert := `
		INSERT INTO system_einstellungen (schluessel, wert)
		SELECT * FROM UNNEST($1::varchar[], $2::text[])
		ON CONFLICT (schluessel) DO UPDATE
		  SET wert = EXCLUDED.wert, aktualisiert_am = CURRENT_TIMESTAMP
	`

	schluessel := make([]string, len(paare))
	werte := make([]string, len(paare))
	for i, p := range paare {
		schluessel[i] = p[0]
		werte[i] = p[1]
	}

	_, err := a.db.Exec(ctx, upsert, schluessel, werte)
	return err
}
