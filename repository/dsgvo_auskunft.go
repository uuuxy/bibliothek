package repository

// dsgvo_auskunft.go — die Abfragen der Auskunft nach Art. 15 DSGVO, die am Leser hängen:
// Stammdaten, Foto, Ausleihen, Schadensfälle, Vormerkungen, Bescheide, Nachbuch-Meldungen und
// die Einträge beider Protokolle. Den Teil am Zugangskonto liest dsgvo_konto.go.
// repository/dsgvo_paar_vollstaendigkeit_test.go liest diese Datei und verlangt jede Tabelle mit
// Leserbezug in einer ihrer Abfragen.

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
)

// DSGVOStammdatenZeile ist die Leserzeile, wie die Auskunft sie liest; die Felder in der
// Reihenfolge von DSGVOStammdatenSQL.
type DSGVOStammdatenZeile struct {
	ID                 string
	BarcodeID          string
	Vorname            string
	Nachname           string
	Klasse             string
	Geburtsdatum       *string
	AbgaengerJahr      int
	IstGesperrt        bool
	IstAbgaenger       bool
	LUSDID             *string
	Strasse            string
	Hausnummer         string
	Plz                string
	Ort                string
	ElternEmail        string
	IstVonHandGesperrt bool
	Sperrgrund         *string
	ErstelltAm         time.Time
	AktualisiertAm     time.Time
	GeloeschtAm        *time.Time
	SchulEintrittAm    *string
	AbgaengerSeit      *time.Time
	LetzterVorgangAm   *time.Time
	LUSDBestaetigtAm   *time.Time
	AnonymisiertAm     *time.Time
	Art                string
	HatZugangskonto    bool
}

// DSGVOStammdatenSQL ist die eine Spaltenliste der Auskunft. Sie ist sichtbar, damit das
// Spalten-Gate sie gegen information_schema.columns halten kann
// (repository/dsgvo_spalten_gate_pg_test.go).
//
// Anschrift und Kontakt sind in der Datenbank nullbar und werden in Texte gelesen; ohne
// COALESCE scheiterte das Lesen bei jedem Leser ohne Adresse. Gelesen wird die Tabelle leser,
// nicht die Sicht schueler: Über die Sicht endete die Auskunft für einen Kollegen mit „nicht
// gefunden".
const DSGVOStammdatenSQL = `
		SELECT id, COALESCE(barcode_id, '') AS barcode_id, vorname, nachname,
		       COALESCE(klasse, '') AS klasse, geburtsdatum::text,
		       COALESCE(abgaenger_jahr, 0) AS abgaenger_jahr,
		       ist_gesperrt, ist_abgaenger, lusd_id,
		       COALESCE(strasse, '') AS strasse, COALESCE(hausnummer, '') AS hausnummer,
		       COALESCE(plz, '') AS plz, COALESCE(ort, '') AS ort,
		       COALESCE(eltern_email, '') AS eltern_email,
		       is_manually_blocked, block_reason,
		       erstellt_am, aktualisiert_am, deleted_at,
		       schul_eintritt_am::text, abgaenger_seit, letzter_vorgang_am,
		       lusd_bestaetigt_am, anonymized_at,
		       art,
		       EXISTS (SELECT 1 FROM benutzer b WHERE b.leser_id = leser.id) AS hat_konto
		FROM leser
		WHERE id = $1`

// LadeDSGVOStammdaten liest die Leserzeile. Ohne Zeile: nil, kein Fehler.
func LadeDSGVOStammdaten(ctx context.Context, q Datenbank, id string) (*DSGVOStammdatenZeile, error) {
	var st DSGVOStammdatenZeile
	err := q.QueryRow(ctx, DSGVOStammdatenSQL, id).Scan(
		&st.ID, &st.BarcodeID, &st.Vorname, &st.Nachname, &st.Klasse, &st.Geburtsdatum,
		&st.AbgaengerJahr, &st.IstGesperrt, &st.IstAbgaenger, &st.LUSDID,
		&st.Strasse, &st.Hausnummer, &st.Plz, &st.Ort, &st.ElternEmail,
		&st.IstVonHandGesperrt, &st.Sperrgrund,
		&st.ErstelltAm, &st.AktualisiertAm, &st.GeloeschtAm,
		&st.SchulEintrittAm, &st.AbgaengerSeit, &st.LetzterVorgangAm,
		&st.LUSDBestaetigtAm, &st.AnonymisiertAm,
		&st.Art, &st.HatZugangskonto,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &st, nil
}

// LadeDSGVOFotoStand nennt, wann das Ausweisfoto zuletzt gespeichert wurde. Ohne Foto: nil,
// kein Fehler.
func LadeDSGVOFotoStand(ctx context.Context, q Datenbank, id string) (*time.Time, error) {
	var aktualisiert time.Time
	err := q.QueryRow(ctx,
		`SELECT aktualisiert_am FROM schueler_fotos WHERE schueler_id = $1`, id,
	).Scan(&aktualisiert)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &aktualisiert, nil
}

// DSGVOAusleiheZeile ist eine Zeile von LadeDSGVOAusleihen.
type DSGVOAusleiheZeile struct {
	Gegenstand     string
	Barcode        string
	AusgeliehenAm  time.Time
	RueckgabeFrist time.Time
	RueckgabeAm    *time.Time
	IstHandapparat bool
}

// LadeDSGVOAusleihen liest die Ausleihen des Lesers, die jüngste zuerst.
func LadeDSGVOAusleihen(ctx context.Context, q Datenbank, id string) ([]DSGVOAusleiheZeile, error) {
	return sammle(ctx, q, `
		SELECT COALESCE(t.titel, g.modellname, 'Unbekannt') AS gegenstand,
		       COALESCE(e.barcode_id, g.barcode_id, '') AS barcode,
		       a.ausgeliehen_am, a.rueckgabe_frist, a.rueckgabe_am, a.ist_handapparat
		FROM ausleihen a
		LEFT JOIN buecher_exemplare e ON e.id = a.exemplar_id
		LEFT JOIN buecher_titel t ON t.id = e.titel_id
		LEFT JOIN geraete g ON g.id = a.geraet_id
		WHERE a.schueler_id = $1
		ORDER BY a.ausgeliehen_am DESC`, func(r pgx.Rows) (DSGVOAusleiheZeile, error) {
		var a DSGVOAusleiheZeile
		return a, r.Scan(&a.Gegenstand, &a.Barcode, &a.AusgeliehenAm, &a.RueckgabeFrist, &a.RueckgabeAm, &a.IstHandapparat)
	}, id)
}

// DSGVOSchadensfallZeile ist eine Zeile von LadeDSGVOSchadensfaelle.
type DSGVOSchadensfallZeile struct {
	Beschreibung      string
	Betrag            string
	IstBezahlt        bool
	ErstelltAm        time.Time
	StorniertAm       *time.Time
	Stornierungsgrund *string
}

// LadeDSGVOSchadensfaelle liest die Schadens- und Verlustfälle des Lesers.
func LadeDSGVOSchadensfaelle(ctx context.Context, q Datenbank, id string) ([]DSGVOSchadensfallZeile, error) {
	return sammle(ctx, q, `
		SELECT beschreibung, betrag::text, ist_bezahlt, erstellt_am, storniert_am, stornierungsgrund
		FROM schadensfaelle
		WHERE schueler_id = $1
		ORDER BY erstellt_am DESC`, func(r pgx.Rows) (DSGVOSchadensfallZeile, error) {
		var f DSGVOSchadensfallZeile
		return f, r.Scan(&f.Beschreibung, &f.Betrag, &f.IstBezahlt, &f.ErstelltAm, &f.StorniertAm, &f.Stornierungsgrund)
	}, id)
}

// DSGVOVormerkungZeile ist eine Zeile von LadeDSGVOVormerkungen.
type DSGVOVormerkungZeile struct {
	Titel      string
	Status     string
	Notiz      *string
	ErstelltAm time.Time
}

// LadeDSGVOVormerkungen liest die Vormerkungen des Lesers.
func LadeDSGVOVormerkungen(ctx context.Context, q Datenbank, id string) ([]DSGVOVormerkungZeile, error) {
	return sammle(ctx, q, `
		SELECT t.titel, v.status, v.notiz, v.erstellt_am
		FROM vormerkungen v
		JOIN buecher_titel t ON t.id = v.titel_id
		WHERE v.schueler_id = $1
		ORDER BY v.erstellt_am DESC`, func(r pgx.Rows) (DSGVOVormerkungZeile, error) {
		var v DSGVOVormerkungZeile
		return v, r.Scan(&v.Titel, &v.Status, &v.Notiz, &v.ErstelltAm)
	}, id)
}

// DSGVONachbuchZeile ist eine Zeile von LadeDSGVONachbuchMeldungen.
type DSGVONachbuchZeile struct {
	Rolle       string
	Barcode     string
	Ergebnis    string
	Grund       *string
	GescanntAm  time.Time
	QuittiertAm *time.Time
}

// LadeDSGVONachbuchMeldungen liest die Nachbuch-Meldungen, in denen die Person als Ausleiher
// oder Vorbesitzer steht. Nach der Tilgung findet die Abfrage nichts mehr: Beide
// Personenspalten sind dann NULL.
func LadeDSGVONachbuchMeldungen(ctx context.Context, q Datenbank, id string) ([]DSGVONachbuchZeile, error) {
	return sammle(ctx, q, `
		SELECT CASE WHEN ausleiher_schueler_id = $1 THEN 'ausleiher' ELSE 'vorbesitzer' END,
		       barcode, ergebnis, grund, gescannt_am, quittiert_am
		FROM nachbuch_meldungen
		WHERE ausleiher_schueler_id = $1 OR vorbesitzer_schueler_id = $1
		ORDER BY gescannt_am DESC`, func(r pgx.Rows) (DSGVONachbuchZeile, error) {
		var m DSGVONachbuchZeile
		return m, r.Scan(&m.Rolle, &m.Barcode, &m.Ergebnis, &m.Grund, &m.GescanntAm, &m.QuittiertAm)
	}, id)
}

// DSGVOBescheidZeile ist eine Zeile von LadeDSGVOBescheide.
type DSGVOBescheidZeile struct {
	Referenznummer string
	BriefDatum     time.Time
	FristBis       time.Time
	Gesamtbetrag   string
	Status         string
}

// LadeDSGVOBescheide liest die Schadensersatz-Bescheide des Lesers. Nach der Anonymisierung
// findet die Abfrage nichts mehr: schueler_id ist dann NULL, der Brief bleibt als Beleg ohne
// Person.
func LadeDSGVOBescheide(ctx context.Context, q Datenbank, id string) ([]DSGVOBescheidZeile, error) {
	return sammle(ctx, q, `
		SELECT referenznummer, brief_datum, frist_bis, gesamtbetrag::text, status
		FROM schadensersatz_bescheide
		WHERE schueler_id = $1
		ORDER BY brief_datum DESC`, func(r pgx.Rows) (DSGVOBescheidZeile, error) {
		var b DSGVOBescheidZeile
		return b, r.Scan(&b.Referenznummer, &b.BriefDatum, &b.FristBis, &b.Gesamtbetrag, &b.Status)
	}, id)
}

// DSGVOProtokollZeile ist eine Zeile von LadeDSGVOProtokollEintraege.
type DSGVOProtokollZeile struct {
	Tabelle    string
	Aktion     string
	Akteur     string
	Zeitpunkt  time.Time
	Kontext    *string
	Gegenstand string
	Barcode    string
	Details    json.RawMessage
}

// LadeDSGVOProtokollEintraege liest jeden Eintrag der Datensatz-Historie, der den Leser nennt: die
// Einträge an seiner Leserzeile (datensatz_id) und die jeder anderen Tabelle, die seine Kennung
// in details tragen — Ausleihe und Rückgabe (datensatz_id ist dort das Exemplar oder Gerät), die
// Stornierung einer Forderung, die Spur einer Ausleihe, Forderung oder Vormerkung, deren Titel
// gelöscht wurde. Den Löscheintrag eines Zugangskontos liest LadeDSGVOFruehereZugangskonten.
func LadeDSGVOProtokollEintraege(ctx context.Context, q Datenbank, id string) ([]DSGVOProtokollZeile, error) {
	return sammle(ctx, q, `
		SELECT al.tabelle, al.aktion, al.akteur, al.timestamp, al.kontext, COALESCE(al.details, 'null'::jsonb),
		       COALESCE(t.titel, g.modellname, ''), COALESCE(e.barcode_id, g.barcode_id, '')
		FROM audit_log al
		LEFT JOIN buecher_exemplare e ON al.tabelle = 'ausleihen' AND e.id = al.datensatz_id
		LEFT JOIN buecher_titel t ON t.id = e.titel_id
		LEFT JOIN geraete g ON al.tabelle = 'ausleihen' AND g.id = al.datensatz_id
		WHERE (al.tabelle = 'schueler' AND al.datensatz_id = $1::uuid)
		   OR (al.tabelle <> 'benutzer' AND al.details->>'schueler_id' = $1::text)
		ORDER BY al.timestamp DESC`, func(r pgx.Rows) (DSGVOProtokollZeile, error) {
		var e DSGVOProtokollZeile
		return e, r.Scan(&e.Tabelle, &e.Aktion, &e.Akteur, &e.Zeitpunkt, &e.Kontext, &e.Details, &e.Gegenstand, &e.Barcode)
	}, id)
}

// DSGVOVerwaltungZeile ist eine Zeile von LadeDSGVOVerwaltungsEintraege.
type DSGVOVerwaltungZeile struct {
	Aktion    string
	Zeitpunkt time.Time
	Details   json.RawMessage
}

// LadeDSGVOVerwaltungsEintraege liest die Einträge des Verwaltungsprotokolls (audit_logs), die
// den Leser über details->>'schueler_id' nennen.
func LadeDSGVOVerwaltungsEintraege(ctx context.Context, q Datenbank, id string) ([]DSGVOVerwaltungZeile, error) {
	return sammle(ctx, q, `
		SELECT aktion, zeitstempel, COALESCE(details, '{}'::jsonb)
		FROM audit_logs
		WHERE details->>'schueler_id' = $1
		ORDER BY zeitstempel DESC`, func(r pgx.Rows) (DSGVOVerwaltungZeile, error) {
		var e DSGVOVerwaltungZeile
		return e, r.Scan(&e.Aktion, &e.Zeitpunkt, &e.Details)
	}, id)
}

// ProtokolliereDSGVOAuskunft hält fest, dass die Auskunft erteilt wurde (Rechenschaft).
// bearbeiterID ist nil, wenn kein angemeldetes Konto sie abgerufen hat.
func ProtokolliereDSGVOAuskunft(ctx context.Context, q Datenbank, id string, bearbeiterID *string, akteur string) error {
	_, err := q.Exec(ctx,
		`INSERT INTO audit_log (tabelle, aktion, datensatz_id, bearbeiter_id, akteur)
		 VALUES ('schueler', 'dsgvo_auskunft', $1::uuid, $2, $3)`,
		id, bearbeiterID, akteur,
	)
	return err
}
