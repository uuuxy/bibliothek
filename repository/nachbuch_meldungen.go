package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

// Nachbuch-Meldungen (Migration 117, Stufe 2 des Offline-Baus): Jede Abweichung vom
// Offline-Scan beim Nachbuchen — umgebucht, nur reaktiviert, nicht gebucht, veraltet —
// bleibt mit Barcode, Grund und Beteiligten stehen, bis jemand aus der Bibliothek sie
// quittiert. Nichts verschwindet still (Entscheidung Peter, 13.09.2026, c).

// Die Ergebnisse des Nachbuchens (OFFEN.md 2.2, Commit 11). `wiederholen` ist keines:
// Es ist die Antwort für einen Eintrag, den der Server nicht beurteilen konnte (503),
// und der bleibt auf dem Theken-Rechner liegen statt hier zu stehen.
const (
	NachbuchAusgeliehen        = "ausgeliehen"
	NachbuchUmgebucht          = "umgebucht"
	NachbuchBereitsAusgeliehen = "bereits_ausgeliehen"
	NachbuchZurueckgegeben     = "zurueckgegeben"
	NachbuchNurReaktiviert     = "nur_reaktiviert"
	NachbuchNichtGebucht       = "nicht_gebucht"
	NachbuchVeraltet           = "veraltet"
)

// NachbuchMeldung ist eine Zeile der Meldungsliste, mit aufgelösten Namen — die Liste
// ist nur mit view_students sichtbar, die Namen sind dort erlaubt.
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

// ErrNachbuchMeldungNichtOffen heißt: quittiert wurde nichts — die Meldung gibt es nicht oder
// sie ist schon quittiert. Kein stiller Erfolg (Phantom-Erfolg-Sweep 31.08.2026).
var ErrNachbuchMeldungNichtOffen = errors.New("nachbuch-meldung nicht offen")

// nachbuchMeldungSQL ist die Zeile der Liste; Namen kommen aus dem Bestand, sind nach der
// Tilgung also weg (FK → NULL), der Barcode-Text bleibt.
const nachbuchMeldungSQL = `
	SELECT m.id, m.barcode, coalesce(t.titel, ''), m.ergebnis, coalesce(m.grund, ''),
	       coalesce(sa.vorname || ' ' || sa.nachname, ba.vorname || ' ' || ba.nachname, ''),
	       coalesce(sa.klasse, ''),
	       coalesce(sv.vorname || ' ' || sv.nachname, bv.vorname || ' ' || bv.nachname, ''),
	       coalesce(m.ausweis_text, ''), m.gescannt_am, m.erstellt_am, m.quittiert_am,
	       coalesce(bq.vorname || ' ' || bq.nachname, '')
	FROM nachbuch_meldungen m
	LEFT JOIN buecher_exemplare e ON e.id = m.exemplar_id
	LEFT JOIN buecher_titel t ON t.id = e.titel_id
	LEFT JOIN schueler sa ON sa.id = m.ausleiher_schueler_id
	LEFT JOIN benutzer ba ON ba.id = m.ausleiher_benutzer_id
	LEFT JOIN schueler sv ON sv.id = m.vorbesitzer_schueler_id
	LEFT JOIN benutzer bv ON bv.id = m.vorbesitzer_benutzer_id
	LEFT JOIN benutzer bq ON bq.id = m.quittiert_von`

// ListeNachbuchMeldungen liest die offenen Meldungen (nurOffen) oder alle, jüngste zuerst.
// Gekappt auf 500 Zeilen: Die Liste ist eine Arbeitsliste, kein Archiv (siehe Löschfrist).
func ListeNachbuchMeldungen(ctx context.Context, q DBQueryer, nurOffen bool) ([]NachbuchMeldung, error) {
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

// QuittiereNachbuchMeldung setzt quittiert_von/am — nur auf einer offenen Meldung.
func QuittiereNachbuchMeldung(ctx context.Context, q DBQueryer, id, benutzerID string) error {
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

// ZaehleOffeneNachbuchMeldungen ist der Zähler fürs Band an der Theke.
func ZaehleOffeneNachbuchMeldungen(ctx context.Context, q DBQueryer) (int, error) {
	var n int
	err := q.QueryRow(ctx, `SELECT count(*) FROM nachbuch_meldungen WHERE quittiert_am IS NULL`).Scan(&n)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, nil
	}
	return n, err
}
