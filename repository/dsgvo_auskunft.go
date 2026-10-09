package repository

// dsgvo_auskunft.go — die Abfragen der Auskunft nach Art. 15 DSGVO, die am Leser hängen:
// Stammdaten, Foto, Ausleihen, Schadensfälle, Vormerkungen, Bescheide, Nachbuch-Meldungen und
// die Einträge beider Protokolle. Den Teil am Zugangskonto liest dsgvo_konto.go.
// api/dsgvo_paar_vollstaendigkeit_test.go liest diese Datei und verlangt jede Tabelle mit
// Leserbezug in einer ihrer Abfragen.

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
)

// DsgvoStammdatenZeile ist die Leserzeile, wie die Auskunft sie liest; die Felder in der
// Reihenfolge von DsgvoStammdatenSQL.
type DsgvoStammdatenZeile struct {
	ID                string
	BarcodeID         string
	Vorname           string
	Nachname          string
	Klasse            string
	Geburtsdatum      *string
	AbgaengerJahr     int
	IstGesperrt       bool
	IstAbgaenger      bool
	LusdID            *string
	Strasse           string
	Hausnummer        string
	Plz               string
	Ort               string
	ElternEmail       string
	IsManuallyBlocked bool
	BlockReason       *string
	ErstelltAm        time.Time
	AktualisiertAm    time.Time
	GeloeschtAm       *time.Time
	SchulEintrittAm   *string
	AbgaengerSeit     *time.Time
	LetzterVorgangAm  *time.Time
	LusdBestaetigtAm  *time.Time
	AnonymisiertAm    *time.Time
	Art               string
	HatZugangskonto   bool
}

// DsgvoStammdatenSQL ist die eine Spaltenliste der Auskunft. Sie ist sichtbar, damit das
// Spalten-Gate sie gegen information_schema.columns halten kann
// (api/dsgvo_spalten_gate_pg_test.go).
//
// Anschrift und Kontakt sind in der Datenbank nullbar und werden in Texte gelesen; ohne
// COALESCE scheiterte das Lesen bei jedem Leser ohne Adresse. Gelesen wird die Tabelle leser,
// nicht die Sicht schueler: Über die Sicht endete die Auskunft für einen Kollegen mit „nicht
// gefunden".
const DsgvoStammdatenSQL = `
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

// LeseDsgvoStammdaten liest die Leserzeile. Ohne Zeile: nil, kein Fehler.
func LeseDsgvoStammdaten(ctx context.Context, q DBQueryer, id string) (*DsgvoStammdatenZeile, error) {
	var st DsgvoStammdatenZeile
	err := q.QueryRow(ctx, DsgvoStammdatenSQL, id).Scan(
		&st.ID, &st.BarcodeID, &st.Vorname, &st.Nachname, &st.Klasse, &st.Geburtsdatum,
		&st.AbgaengerJahr, &st.IstGesperrt, &st.IstAbgaenger, &st.LusdID,
		&st.Strasse, &st.Hausnummer, &st.Plz, &st.Ort, &st.ElternEmail,
		&st.IsManuallyBlocked, &st.BlockReason,
		&st.ErstelltAm, &st.AktualisiertAm, &st.GeloeschtAm,
		&st.SchulEintrittAm, &st.AbgaengerSeit, &st.LetzterVorgangAm,
		&st.LusdBestaetigtAm, &st.AnonymisiertAm,
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

// LeseDsgvoFotoStand nennt, wann das Ausweisfoto zuletzt gespeichert wurde. Ohne Foto: nil,
// kein Fehler.
func LeseDsgvoFotoStand(ctx context.Context, q DBQueryer, id string) (*time.Time, error) {
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

// DsgvoAusleiheZeile ist eine Zeile von LeseDsgvoAusleihen.
type DsgvoAusleiheZeile struct {
	Gegenstand     string
	Barcode        string
	AusgeliehenAm  time.Time
	RueckgabeFrist time.Time
	RueckgabeAm    *time.Time
	IstHandapparat bool
}

// LeseDsgvoAusleihen liest die Ausleihen des Lesers, die jüngste zuerst.
func LeseDsgvoAusleihen(ctx context.Context, q DBQueryer, id string) ([]DsgvoAusleiheZeile, error) {
	return sammle(ctx, q, `
		SELECT COALESCE(t.titel, g.modellname, 'Unbekannt') AS gegenstand,
		       COALESCE(e.barcode_id, g.barcode_id, '') AS barcode,
		       a.ausgeliehen_am, a.rueckgabe_frist, a.rueckgabe_am, a.ist_handapparat
		FROM ausleihen a
		LEFT JOIN buecher_exemplare e ON e.id = a.exemplar_id
		LEFT JOIN buecher_titel t ON t.id = e.titel_id
		LEFT JOIN geraete g ON g.id = a.geraet_id
		WHERE a.schueler_id = $1
		ORDER BY a.ausgeliehen_am DESC`, func(r pgx.Rows) (DsgvoAusleiheZeile, error) {
		var a DsgvoAusleiheZeile
		return a, r.Scan(&a.Gegenstand, &a.Barcode, &a.AusgeliehenAm, &a.RueckgabeFrist, &a.RueckgabeAm, &a.IstHandapparat)
	}, id)
}

// DsgvoSchadensfallZeile ist eine Zeile von LeseDsgvoSchadensfaelle.
type DsgvoSchadensfallZeile struct {
	Beschreibung      string
	Betrag            string
	IstBezahlt        bool
	ErstelltAm        time.Time
	StorniertAm       *time.Time
	Stornierungsgrund *string
}

// LeseDsgvoSchadensfaelle liest die Schadens- und Verlustfälle des Lesers.
func LeseDsgvoSchadensfaelle(ctx context.Context, q DBQueryer, id string) ([]DsgvoSchadensfallZeile, error) {
	return sammle(ctx, q, `
		SELECT beschreibung, betrag::text, ist_bezahlt, erstellt_am, storniert_am, stornierungsgrund
		FROM schadensfaelle
		WHERE schueler_id = $1
		ORDER BY erstellt_am DESC`, func(r pgx.Rows) (DsgvoSchadensfallZeile, error) {
		var f DsgvoSchadensfallZeile
		return f, r.Scan(&f.Beschreibung, &f.Betrag, &f.IstBezahlt, &f.ErstelltAm, &f.StorniertAm, &f.Stornierungsgrund)
	}, id)
}

// DsgvoVormerkungZeile ist eine Zeile von LeseDsgvoVormerkungen.
type DsgvoVormerkungZeile struct {
	Titel      string
	Status     string
	Notiz      *string
	ErstelltAm time.Time
}

// LeseDsgvoVormerkungen liest die Vormerkungen des Lesers.
func LeseDsgvoVormerkungen(ctx context.Context, q DBQueryer, id string) ([]DsgvoVormerkungZeile, error) {
	return sammle(ctx, q, `
		SELECT t.titel, v.status, v.notiz, v.erstellt_am
		FROM vormerkungen v
		JOIN buecher_titel t ON t.id = v.titel_id
		WHERE v.schueler_id = $1
		ORDER BY v.erstellt_am DESC`, func(r pgx.Rows) (DsgvoVormerkungZeile, error) {
		var v DsgvoVormerkungZeile
		return v, r.Scan(&v.Titel, &v.Status, &v.Notiz, &v.ErstelltAm)
	}, id)
}

// DsgvoNachbuchZeile ist eine Zeile von LeseDsgvoNachbuchMeldungen.
type DsgvoNachbuchZeile struct {
	Rolle       string
	Barcode     string
	Ergebnis    string
	Grund       *string
	GescanntAm  time.Time
	QuittiertAm *time.Time
}

// LeseDsgvoNachbuchMeldungen liest die Nachbuch-Meldungen, in denen die Person als Ausleiher
// oder Vorbesitzer steht. Nach der Tilgung findet die Abfrage nichts mehr: Beide
// Personenspalten sind dann NULL.
func LeseDsgvoNachbuchMeldungen(ctx context.Context, q DBQueryer, id string) ([]DsgvoNachbuchZeile, error) {
	return sammle(ctx, q, `
		SELECT CASE WHEN ausleiher_schueler_id = $1 THEN 'ausleiher' ELSE 'vorbesitzer' END,
		       barcode, ergebnis, grund, gescannt_am, quittiert_am
		FROM nachbuch_meldungen
		WHERE ausleiher_schueler_id = $1 OR vorbesitzer_schueler_id = $1
		ORDER BY gescannt_am DESC`, func(r pgx.Rows) (DsgvoNachbuchZeile, error) {
		var m DsgvoNachbuchZeile
		return m, r.Scan(&m.Rolle, &m.Barcode, &m.Ergebnis, &m.Grund, &m.GescanntAm, &m.QuittiertAm)
	}, id)
}

// DsgvoBescheidZeile ist eine Zeile von LeseDsgvoBescheide.
type DsgvoBescheidZeile struct {
	Referenznummer string
	BriefDatum     time.Time
	FristBis       time.Time
	Gesamtbetrag   string
	Status         string
}

// LeseDsgvoBescheide liest die Schadensersatz-Bescheide des Lesers. Nach der Anonymisierung
// findet die Abfrage nichts mehr: schueler_id ist dann NULL, der Brief bleibt als Beleg ohne
// Person.
func LeseDsgvoBescheide(ctx context.Context, q DBQueryer, id string) ([]DsgvoBescheidZeile, error) {
	return sammle(ctx, q, `
		SELECT referenznummer, brief_datum, frist_bis, gesamtbetrag::text, status
		FROM schadensersatz_bescheide
		WHERE schueler_id = $1
		ORDER BY brief_datum DESC`, func(r pgx.Rows) (DsgvoBescheidZeile, error) {
		var b DsgvoBescheidZeile
		return b, r.Scan(&b.Referenznummer, &b.BriefDatum, &b.FristBis, &b.Gesamtbetrag, &b.Status)
	}, id)
}

// DsgvoAuditZeile ist eine Zeile von LeseDsgvoAuditEintraege.
type DsgvoAuditZeile struct {
	Tabelle    string
	Aktion     string
	Akteur     string
	Zeitpunkt  time.Time
	Kontext    *string
	Gegenstand string
	Barcode    string
	Details    json.RawMessage
}

// LeseDsgvoAuditEintraege liest jeden Eintrag der Datensatz-Historie, der den Leser nennt: die
// Einträge an seiner Leserzeile (datensatz_id) und die jeder anderen Tabelle, die seine Kennung
// in details tragen — Ausleihe und Rückgabe (datensatz_id ist dort das Exemplar oder Gerät), die
// Stornierung einer Forderung, die Spur einer Ausleihe, Forderung oder Vormerkung, deren Titel
// gelöscht wurde. Den Löscheintrag eines Zugangskontos liest LeseDsgvoFruehereZugangskonten.
func LeseDsgvoAuditEintraege(ctx context.Context, q DBQueryer, id string) ([]DsgvoAuditZeile, error) {
	return sammle(ctx, q, `
		SELECT al.tabelle, al.aktion, al.akteur, al.timestamp, al.kontext, COALESCE(al.details, 'null'::jsonb),
		       COALESCE(t.titel, g.modellname, ''), COALESCE(e.barcode_id, g.barcode_id, '')
		FROM audit_log al
		LEFT JOIN buecher_exemplare e ON al.tabelle = 'ausleihen' AND e.id = al.datensatz_id
		LEFT JOIN buecher_titel t ON t.id = e.titel_id
		LEFT JOIN geraete g ON al.tabelle = 'ausleihen' AND g.id = al.datensatz_id
		WHERE (al.tabelle = 'schueler' AND al.datensatz_id = $1::uuid)
		   OR (al.tabelle <> 'benutzer' AND al.details->>'schueler_id' = $1::text)
		ORDER BY al.timestamp DESC`, func(r pgx.Rows) (DsgvoAuditZeile, error) {
		var e DsgvoAuditZeile
		return e, r.Scan(&e.Tabelle, &e.Aktion, &e.Akteur, &e.Zeitpunkt, &e.Kontext, &e.Details, &e.Gegenstand, &e.Barcode)
	}, id)
}

// DsgvoVerwaltungZeile ist eine Zeile von LeseDsgvoVerwaltungsEintraege.
type DsgvoVerwaltungZeile struct {
	Aktion    string
	Zeitpunkt time.Time
	Details   json.RawMessage
}

// LeseDsgvoVerwaltungsEintraege liest die Einträge des Verwaltungsprotokolls (audit_logs), die
// den Leser über details->>'schueler_id' nennen.
func LeseDsgvoVerwaltungsEintraege(ctx context.Context, q DBQueryer, id string) ([]DsgvoVerwaltungZeile, error) {
	return sammle(ctx, q, `
		SELECT aktion, zeitstempel, COALESCE(details, '{}'::jsonb)
		FROM audit_logs
		WHERE details->>'schueler_id' = $1
		ORDER BY zeitstempel DESC`, func(r pgx.Rows) (DsgvoVerwaltungZeile, error) {
		var e DsgvoVerwaltungZeile
		return e, r.Scan(&e.Aktion, &e.Zeitpunkt, &e.Details)
	}, id)
}

// SchreibeDsgvoAuskunftProtokoll hält fest, dass die Auskunft erteilt wurde (Rechenschaft).
// bearbeiterID ist nil, wenn kein angemeldetes Konto sie abgerufen hat.
func SchreibeDsgvoAuskunftProtokoll(ctx context.Context, q DBQueryer, id string, bearbeiterID *string, akteur string) error {
	_, err := q.Exec(ctx,
		`INSERT INTO audit_log (tabelle, aktion, datensatz_id, bearbeiter_id, akteur)
		 VALUES ('schueler', 'dsgvo_auskunft', $1::uuid, $2, $3)`,
		id, bearbeiterID, akteur,
	)
	return err
}
