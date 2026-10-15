package repository

import (
	"context"
	"errors"
	"time"

	"bibliothek/db"

	"github.com/jackc/pgx/v5"
)

// Reservierung ist die Reservierung eines Klassensatzes durch eine Lehrkraft, wie die
// Bibliothek sie in ihrer Liste sieht.
type Reservierung struct {
	ID             string  `json:"id"`
	TitelID        string  `json:"titel_id"`
	TitelName      string  `json:"titel_name"`
	CoverURL       string  `json:"cover_url,omitempty"`
	Klasse         string  `json:"klasse"`
	Anzahl         int     `json:"anzahl"`
	Notiz          *string `json:"notiz,omitempty"`
	AngefordertVon *string `json:"angefordert_von,omitempty"`
	Erledigt       bool    `json:"erledigt"`
	// ErledigtNotiz und ErledigtAm sind die Antwort der Bibliothek und der Zeitpunkt des
	// Abschlusses. Die Notiz steht hier und nicht nur in der Mail an die Lehrkraft: Scheitert
	// die Mail, bliebe sie sonst unsichtbar. ErledigtAm ist leer bei alten Reservierungen ohne
	// Zeitpunkt.
	ErledigtNotiz string `json:"erledigt_notiz,omitempty"`
	ErledigtAm    string `json:"erledigt_am,omitempty"`
	ErstelltAm    string `json:"erstellt_am"`
	// Verfuegbar ist der ausleihbare Bestand des Titels, gezählt wie im Katalog (ausleihbar,
	// nicht ausgesondert, keine offene Ausleihe): Die Bibliothek sieht beim Erledigen, ob der
	// Satz vollständig ist, ohne am Regal zu zählen.
	Verfuegbar int `json:"verfuegbar"`
}

// OffeneReservierung ist die schmale Sicht für das Portal des Kollegiums: wer vor der eigenen
// Reservierung steht (Titel, Klasse, Menge), ohne Personendaten des Anfordernden. Der Titel ist
// keine Person und steht ohnehin im öffentlichen Katalog; ohne ihn sagte die Zeile nicht, worum
// es geht.
type OffeneReservierung struct {
	TitelID    string `json:"titel_id"`
	Titel      string `json:"titel"`
	Klasse     string `json:"klasse"`
	Anzahl     int    `json:"anzahl"`
	ErstelltAm string `json:"erstellt_am"`
}

// ReservierungsAbfragen liest und schreibt die Reservierungen von Klassensätzen.
type ReservierungsAbfragen interface {
	IstTitelVorhanden(ctx context.Context, titelID string) (bool, error)
	// ZaehleTitelBestand zählt die vorhandenen (nicht ausgesonderten) Exemplare eines Titels:
	// die Obergrenze einer Reservierung.
	ZaehleTitelBestand(ctx context.Context, titelID string) (int, error)
	LegeAn(ctx context.Context, titelID, klasse string, anzahl int, notiz *string, angefordertVon string, idempotenzSchluessel *string) (id string, neu bool, err error)
	LadeAlle(ctx context.Context) ([]Reservierung, error)
	// LadeOffene liefert die offenen Reservierungen in der Reihenfolge der Warteschlange (die
	// älteste zuerst), für das Portal.
	LadeOffene(ctx context.Context) ([]OffeneReservierung, error)
	ZaehleOffene(ctx context.Context) (int, error)
	// Erledige schließt eine offene Reservierung ab und liefert die Angaben für die Mail an die
	// Lehrkraft; notiz ist die Antwort der Bibliothek. nil ohne Fehler heißt: war nicht mehr
	// offen.
	Erledige(ctx context.Context, id, notiz string) (*ErledigteReservierung, error)
	// LadeEigene liefert die eigenen Reservierungen einer Lehrkraft für das Portal: offene und
	// kürzlich abgeschlossene mit der Notiz der Bibliothek.
	LadeEigene(ctx context.Context, benutzerID string) ([]EigeneReservierung, error)
}

type pgReservierungsAbfragen struct {
	db db.Pool
}

// NewReservierungsAbfragen liefert die Abfragen der Reservierungen an Postgres.
func NewReservierungsAbfragen(pool db.Pool) ReservierungsAbfragen {
	return &pgReservierungsAbfragen{db: pool}
}

// IstTitelVorhanden meldet, ob es den Titel gibt.
func (a *pgReservierungsAbfragen) IstTitelVorhanden(ctx context.Context, titelID string) (bool, error) {
	var vorhanden bool
	err := a.db.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM buecher_titel WHERE id = $1)`, titelID).Scan(&vorhanden)
	return vorhanden, err
}

// ZaehleTitelBestand zählt die vorhandenen (nicht ausgesonderten) Exemplare eines Titels. Eine
// Reservierung über diese Zahl hinaus ließe sich nie erfüllen und bliebe als Aufgabe in der
// Liste stehen.
func (a *pgReservierungsAbfragen) ZaehleTitelBestand(ctx context.Context, titelID string) (int, error) {
	var anzahl int
	err := a.db.QueryRow(ctx, `
		SELECT COUNT(*) FROM buecher_exemplare
		WHERE titel_id = $1 AND ist_ausgesondert = false
	`, titelID).Scan(&anzahl)
	return anzahl, err
}

// LegeAn legt eine Reservierung an; der zweite Wert ist false, wenn dieselbe Anfrage schon
// angelegt war.
func (a *pgReservierungsAbfragen) LegeAn(ctx context.Context, titelID, klasse string, anzahl int, notiz *string, angefordertVon string, idempotenzSchluessel *string) (string, bool, error) {
	// Schutz vor dem Doppelklick: Mit demselben idempotenz_schluessel läuft das zweite INSERT
	// auf den partiellen Unique-Index, DO NOTHING liefert keine Zeile, und zurück geht die
	// schon angelegte Reservierung (keine zweite Mail). Ohne Schlüssel (NULL) kollidiert
	// nichts.
	var id string
	err := a.db.QueryRow(ctx, `
		INSERT INTO klassensatz_reservierungen
			(titel_id, klasse, anzahl, notiz, angefordert_von, idempotenz_schluessel)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (idempotenz_schluessel) WHERE idempotenz_schluessel IS NOT NULL DO NOTHING
		RETURNING id
	`, titelID, klasse, anzahl, notiz, angefordertVon, idempotenzSchluessel).Scan(&id)
	if err == nil {
		return id, true, nil
	}
	if errors.Is(err, pgx.ErrNoRows) && idempotenzSchluessel != nil {
		// Dieselbe Anfrage lief schon durch: die ID der bestehenden Reservierung liefern.
		errBestehend := a.db.QueryRow(ctx,
			`SELECT id FROM klassensatz_reservierungen WHERE idempotenz_schluessel = $1`,
			*idempotenzSchluessel).Scan(&id)
		return id, false, errBestehend
	}
	return "", false, err
}

// LadeAlle liefert alle Reservierungen: die offenen in der Reihenfolge der Warteschlange (die
// älteste zuerst), dahinter die erledigten, die neuesten zuerst.
func (a *pgReservierungsAbfragen) LadeAlle(ctx context.Context) ([]Reservierung, error) {
	// Verfuegbar zählt wie der Katalog: ausleihbar, nicht ausgesondert, keine offene Ausleihe.
	//
	// Die Zählung steht als LATERAL-Join und nicht in der SELECT-Liste: So kann Postgres sie je
	// Titel zwischenspeichern (Memoize), und mehrere Reservierungen desselben Titels zählen
	// einmal. Gemessen an 2.000 Reservierungen über 252 Titel: 17 ms statt 98 ms, 252 statt
	// 2.000 Zählungen.
	rows, err := a.db.Query(ctx, `
		SELECT r.id, r.titel_id, t.titel, coalesce(t.cover_url,''),
		       r.klasse, r.anzahl, r.notiz, r.erledigt, r.erstellt_am,
		       CASE WHEN b.id IS NULL THEN NULL
		            ELSE btrim(b.vorname || ' ' || b.nachname) END AS angefordert_von,
		       v.verfuegbar,
		       r.erledigt_notiz, r.erledigt_am
		FROM klassensatz_reservierungen r
		JOIN buecher_titel t ON r.titel_id = t.id
		LEFT JOIN benutzer b ON r.angefordert_von = b.id
		CROSS JOIN LATERAL (
		       SELECT COUNT(*) AS verfuegbar FROM buecher_exemplare e
		       WHERE e.titel_id = r.titel_id
		         AND e.ist_ausleihbar = true AND e.ist_ausgesondert = false
		         AND NOT EXISTS (SELECT 1 FROM ausleihen a
		                         WHERE a.exemplar_id = e.id AND a.rueckgabe_am IS NULL)
		) v
		ORDER BY r.erledigt ASC,
		         CASE WHEN r.erledigt THEN r.erstellt_am END DESC,
		         CASE WHEN NOT r.erledigt THEN r.erstellt_am END ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var reservierungen []Reservierung
	for rows.Next() {
		var r Reservierung
		var t time.Time
		var erledigtAm *time.Time // NULL bei alten Reservierungen ohne Zeitpunkt
		if err := rows.Scan(
			&r.ID, &r.TitelID, &r.TitelName, &r.CoverURL,
			&r.Klasse, &r.Anzahl, &r.Notiz, &r.Erledigt, &t, &r.AngefordertVon, &r.Verfuegbar,
			&r.ErledigtNotiz, &erledigtAm,
		); err != nil {
			continue
		}
		r.ErstelltAm = t.Format(datumsformat)
		if erledigtAm != nil {
			r.ErledigtAm = erledigtAm.Format(datumsformat)
		}
		reservierungen = append(reservierungen, r)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return reservierungen, nil
}

// LadeOffene liefert die Warteschlange, wie die Lehrkräfte sie sehen.
func (a *pgReservierungsAbfragen) LadeOffene(ctx context.Context) ([]OffeneReservierung, error) {
	// Mit LEFT JOIN: Ein gelöschter Titel nimmt seine Reservierung nicht aus der Warteschlange
	// der anderen; dort steht dann ein leerer Name.
	rows, err := a.db.Query(ctx, `
		SELECT k.titel_id, COALESCE(t.titel, ''), k.klasse, k.anzahl, k.erstellt_am
		FROM klassensatz_reservierungen k
		LEFT JOIN buecher_titel t ON t.id = k.titel_id
		WHERE k.erledigt = false
		ORDER BY k.erstellt_am ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	offene := []OffeneReservierung{}
	for rows.Next() {
		var o OffeneReservierung
		var t time.Time
		if err := rows.Scan(&o.TitelID, &o.Titel, &o.Klasse, &o.Anzahl, &t); err != nil {
			return nil, err
		}
		o.ErstelltAm = t.Format(datumsformat)
		offene = append(offene, o)
	}
	return offene, rows.Err()
}

// EigeneReservierung ist die Sicht einer Lehrkraft im Portal auf ihre Reservierungen: offene
// und kürzlich abgeschlossene, mit der Antwort der Bibliothek. So bleibt die Antwort sichtbar,
// wenn die Mail scheitert.
type EigeneReservierung struct {
	ID            string `json:"id"`
	TitelID       string `json:"titel_id"`
	Titel         string `json:"titel"`
	Klasse        string `json:"klasse"`
	Anzahl        int    `json:"anzahl"`
	Notiz         string `json:"notiz,omitempty"`
	Erledigt      bool   `json:"erledigt"`
	ErledigtNotiz string `json:"erledigt_notiz,omitempty"`
	ErledigtAm    string `json:"erledigt_am,omitempty"` // leer bei alten Reservierungen ohne Zeitpunkt
	ErstelltAm    string `json:"erstellt_am"`
}

// LadeEigene liefert die eigenen offenen Reservierungen und die in den letzten 60 Tagen
// abgeschlossenen; das Portal ist kein Archiv. Alte Reservierungen ohne erledigt_am bleiben
// draußen. Offene zuerst, dann die neuesten, höchstens 50.
func (a *pgReservierungsAbfragen) LadeEigene(ctx context.Context, benutzerID string) ([]EigeneReservierung, error) {
	rows, err := a.db.Query(ctx, `
		SELECT k.id, k.titel_id, COALESCE(t.titel, ''), k.klasse, k.anzahl,
		       COALESCE(k.notiz, ''), k.erledigt, k.erledigt_notiz, k.erledigt_am, k.erstellt_am
		FROM klassensatz_reservierungen k
		LEFT JOIN buecher_titel t ON t.id = k.titel_id
		WHERE k.angefordert_von = $1
		  AND (k.erledigt = false OR k.erledigt_am > NOW() - INTERVAL '60 days')
		ORDER BY k.erledigt ASC, k.erstellt_am DESC
		LIMIT 50`, benutzerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	meine := []EigeneReservierung{}
	for rows.Next() {
		var m EigeneReservierung
		var erstellt time.Time
		var erledigtAm *time.Time
		if err := rows.Scan(&m.ID, &m.TitelID, &m.Titel, &m.Klasse, &m.Anzahl,
			&m.Notiz, &m.Erledigt, &m.ErledigtNotiz, &erledigtAm, &erstellt); err != nil {
			return nil, err
		}
		m.ErstelltAm = erstellt.Format(datumsformat)
		if erledigtAm != nil {
			m.ErledigtAm = erledigtAm.Format(datumsformat)
		}
		meine = append(meine, m)
	}
	return meine, rows.Err()
}

// ZaehleOffene zählt die offenen Reservierungen.
func (a *pgReservierungsAbfragen) ZaehleOffene(ctx context.Context) (int, error) {
	var anzahl int
	err := a.db.QueryRow(ctx, `SELECT COUNT(*) FROM klassensatz_reservierungen WHERE erledigt = false`).Scan(&anzahl)
	return anzahl, err
}

// ErledigteReservierung sind die Angaben für die Mail an die Lehrkraft nach dem Abschluss.
type ErledigteReservierung struct {
	TitelName      string
	Klasse         string
	Anzahl         int
	Notiz          string  // was die Lehrkraft beim Reservieren schrieb ('' wenn nichts)
	ErledigtNotiz  string  // Antwort der Bibliothek beim Abschließen ('' wenn nichts)
	AnfragendeMail *string // nil: ohne Konto angefragt oder Konto gelöscht
}

// Erledige schließt eine offene Reservierung ab und liefert die Angaben für die Mail; nil ohne
// Fehler heißt, sie war nicht mehr offen.
func (a *pgReservierungsAbfragen) Erledige(ctx context.Context, id, notiz string) (*ErledigteReservierung, error) {
	// Die Bedingung erledigt = false: Ein zweiter Klick schließt nicht doppelt ab und löst
	// keine zweite Mail aus.
	row := a.db.QueryRow(ctx, `
		WITH abgeschlossen AS (
			UPDATE klassensatz_reservierungen
			SET erledigt = true, erledigt_notiz = $2, erledigt_am = CURRENT_TIMESTAMP
			WHERE id = $1 AND erledigt = false
			RETURNING titel_id, klasse, anzahl, notiz, erledigt_notiz, angefordert_von
		)
		SELECT t.titel, a.klasse, a.anzahl, COALESCE(a.notiz, ''), a.erledigt_notiz, b.email
		FROM abgeschlossen a
		JOIN buecher_titel t ON t.id = a.titel_id
		LEFT JOIN benutzer b ON b.id = a.angefordert_von
	`, id, notiz)

	var e ErledigteReservierung
	if err := row.Scan(&e.TitelName, &e.Klasse, &e.Anzahl, &e.Notiz, &e.ErledigtNotiz, &e.AnfragendeMail); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &e, nil
}
