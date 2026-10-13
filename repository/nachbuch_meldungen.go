package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

// Die Ergebnisse des Nachbuchens. „wiederholen" ist keines: Es ist die Antwort für einen
// Eintrag, den der Server nicht beurteilen konnte (503); der bleibt auf dem Theken-Rechner
// liegen und steht nicht hier.
const (
	NachbuchAusgeliehen        = "ausgeliehen"
	NachbuchUmgebucht          = "umgebucht"
	NachbuchBereitsAusgeliehen = "bereits_ausgeliehen"
	NachbuchZurueckgegeben     = "zurueckgegeben"
	NachbuchNurReaktiviert     = "nur_reaktiviert"
	NachbuchNichtGebucht       = "nicht_gebucht"
	NachbuchVeraltet           = "veraltet"
)

// NachbuchMeldungEingabe ist, was das Nachbuchen je Abweichung festhält. Jede Abweichung vom
// Scan ohne Netz (umgebucht, nur reaktiviert, nicht gebucht, veraltet) bleibt mit Barcode,
// Grund und Beteiligten stehen, bis jemand aus der Bibliothek sie quittiert.
type NachbuchMeldungEingabe struct {
	IdempotencyKey        string
	ExemplarID            *string
	Barcode               string
	Ergebnis              string
	Grund                 string
	AusleiherSchuelerID   *string
	VorbesitzerSchuelerID *string
	AusweisText           *string
	GescanntAm            time.Time
}

// NachbuchMeldung ist eine Zeile der Liste der Meldungen, mit den Namen der Beteiligten. Die
// Liste ist nur mit dem Recht view_students zu sehen; die Namen sind dort erlaubt.
type NachbuchMeldung struct {
	ID              string     `json:"id"`
	Barcode         string     `json:"barcode"`
	Titel           string     `json:"titel"`
	Ergebnis        string     `json:"ergebnis"`
	Grund           string     `json:"grund,omitempty"`
	Ausleiher       string     `json:"ausleiher,omitempty"`
	Vorbesitzer     string     `json:"vorbesitzer,omitempty"`
	AusweisText     string     `json:"ausweis_text,omitempty"`
	GescanntAm      time.Time  `json:"gescannt_am"`
	ErstelltAm      time.Time  `json:"erstellt_am"`
	QuittiertAm     *time.Time `json:"quittiert_am,omitempty"`
	QuittiertVon    string     `json:"quittiert_von,omitempty"`
	AusleiherKlasse string     `json:"ausleiher_klasse,omitempty"`
}

// ErrNachbuchMeldungNichtOffen meldet, dass nichts quittiert wurde: Die Meldung gibt es nicht,
// oder sie ist schon quittiert.
var ErrNachbuchMeldungNichtOffen = errors.New("nachbuch-meldung nicht offen")

// SchreibeNachbuchMeldung legt eine Meldung ab, in der Transaktion des Rufers, damit sie mit
// der Buchung gilt oder fällt. Ein zweiter Eintrag mit demselben Schlüssel wird übergangen (ON
// CONFLICT DO NOTHING): Die Wiederholung eines Theken-Rechners ist kein zweiter Vorgang.
func SchreibeNachbuchMeldung(ctx context.Context, q Datenbank, m NachbuchMeldungEingabe) error {
	if _, err := q.Exec(ctx, `
		INSERT INTO nachbuch_meldungen
			(idempotency_key, exemplar_id, barcode, ergebnis, grund,
			 ausleiher_schueler_id, vorbesitzer_schueler_id,
			 ausweis_text, gescannt_am)
		VALUES ($1::uuid, $2::uuid, $3, $4, NULLIF($5, ''), $6::uuid, $7::uuid, $8, $9)
		ON CONFLICT (idempotency_key) DO NOTHING`,
		m.IdempotencyKey, m.ExemplarID, m.Barcode, m.Ergebnis, m.Grund,
		m.AusleiherSchuelerID, m.VorbesitzerSchuelerID,
		m.AusweisText, m.GescanntAm); err != nil {
		return fmt.Errorf("nachbuch-meldung schreiben: %w", err)
	}
	return nil
}

// nachbuchMeldungSQL ist die Zeile der Liste. Die Namen kommen aus dem Bestand und sind nach
// der Tilgung weg (der Fremdschlüssel wird NULL); der Barcode als Text bleibt.
const nachbuchMeldungSQL = `
	SELECT m.id, m.barcode, coalesce(t.titel, ''), m.ergebnis, coalesce(m.grund, ''),
	       coalesce(la.vorname || ' ' || la.nachname, ''),
	       coalesce(la.klasse, ''),
	       coalesce(lv.vorname || ' ' || lv.nachname, ''),
	       coalesce(m.ausweis_text, ''), m.gescannt_am, m.erstellt_am, m.quittiert_am,
	       coalesce(bq.vorname || ' ' || bq.nachname, '')
	FROM nachbuch_meldungen m
	LEFT JOIN buecher_exemplare e ON e.id = m.exemplar_id
	LEFT JOIN buecher_titel t ON t.id = e.titel_id
	-- leser und nicht die Sicht schueler: In der Meldungsliste stünde ein Kollege sonst
	-- ohne Namen da, obwohl sein Scan der Anlass der Meldung war (Migration 125).
	LEFT JOIN leser la ON la.id = m.ausleiher_schueler_id
	LEFT JOIN leser lv ON lv.id = m.vorbesitzer_schueler_id
	LEFT JOIN benutzer bq ON bq.id = m.quittiert_von`

// LadeNachbuchMeldungen liest die offenen Meldungen (nurOffen) oder alle, die jüngsten zuerst,
// höchstens 500: Die Liste ist eine Arbeitsliste, kein Archiv.
func LadeNachbuchMeldungen(ctx context.Context, q Datenbank, nurOffen bool) ([]NachbuchMeldung, error) {
	where := ""
	if nurOffen {
		where = " WHERE m.quittiert_am IS NULL"
	}
	rows, err := q.Query(ctx, nachbuchMeldungSQL+where+` ORDER BY m.erstellt_am DESC LIMIT 500`)
	if err != nil {
		return nil, fmt.Errorf("nachbuch-meldungen lesen: %w", err)
	}
	defer rows.Close()
	out := []NachbuchMeldung{}
	for rows.Next() {
		var m NachbuchMeldung
		if err := rows.Scan(&m.ID, &m.Barcode, &m.Titel, &m.Ergebnis, &m.Grund, &m.Ausleiher, &m.AusleiherKlasse,
			&m.Vorbesitzer, &m.AusweisText, &m.GescanntAm, &m.ErstelltAm, &m.QuittiertAm, &m.QuittiertVon); err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

// QuittiereNachbuchMeldung setzt quittiert_von und quittiert_am, nur auf einer offenen Meldung.
func QuittiereNachbuchMeldung(ctx context.Context, q Datenbank, id, benutzerID string) error {
	tag, err := q.Exec(ctx, `
		UPDATE nachbuch_meldungen SET quittiert_von = $2, quittiert_am = CURRENT_TIMESTAMP
		WHERE id = $1 AND quittiert_am IS NULL`, id, benutzerID)
	if err != nil {
		return fmt.Errorf("nachbuch-meldung quittieren: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNachbuchMeldungNichtOffen
	}
	return nil
}

// ZaehleOffeneNachbuchMeldungen zählt die offenen Meldungen für das Band an der Theke.
func ZaehleOffeneNachbuchMeldungen(ctx context.Context, q Datenbank) (int, error) {
	var n int
	err := q.QueryRow(ctx, `SELECT count(*) FROM nachbuch_meldungen WHERE quittiert_am IS NULL`).Scan(&n)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, nil
	}
	return n, err
}
