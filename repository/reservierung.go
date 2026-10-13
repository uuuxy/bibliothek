package repository

import (
	"context"
	"errors"
	"time"

	"bibliothek/db"

	"github.com/jackc/pgx/v5"
)

// Reservierung represents a pending class-set reservation.
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
	// ErledigtNotiz/ErledigtAm: Antwort der Bibliothek und Abschlusszeitpunkt (088/089).
	// Bis zum 31.08.2026 existierte die Notiz nur in der Bereit-Mail — scheiterte die,
	// war sie für immer unsichtbar. ErledigtAm leer = vor Migration 089 abgeschlossen.
	ErledigtNotiz string `json:"erledigt_notiz,omitempty"`
	ErledigtAm    string `json:"erledigt_am,omitempty"`
	ErstelltAm    string `json:"erstellt_am"`
	// Verfuegbar ist der aktuelle Ausleih-Bestand des Titels (gleiche Definition wie
	// im OPAC: ausleihbar, nicht ausgesondert, keine offene Ausleihe) — damit sieht
	// die Bibliothek beim Erledigen, ob der Satz noch vollständig ist, ohne am Regal
	// zu zählen.
	Verfuegbar int `json:"verfuegbar"`
}

// OffeneReservierung ist die schmale Sicht für das Kollegiums-Portal: Wer vor der
// eigenen Reservierung steht (Titel, Klasse, Menge) — bewusst OHNE Personendaten
// des Anfordernden.
//
// Der TITEL fehlte bis zum 23.08.2026, obwohl der Kommentar darüber ihn nannte. Neben
// einem Suchtreffer fiel das nicht auf (das Buch steht ja daneben); auf der Startfläche
// des Portals stand dann „Klasse 8B · 28 Stück" ohne jeden Hinweis, worum es geht. Der
// Titel ist keine Person und steht ohnehin im öffentlichen Katalog.
type OffeneReservierung struct {
	TitelID    string `json:"titel_id"`
	Titel      string `json:"titel"`
	Klasse     string `json:"klasse"`
	Anzahl     int    `json:"anzahl"`
	ErstelltAm string `json:"erstellt_am"`
}

// ReservierungsAbfragen kapselt die Datenbankzugriffe für Reservierungen.
type ReservierungsAbfragen interface {
	IstTitelVorhanden(ctx context.Context, titelID string) (bool, error)
	// ZaehleTitelBestand liefert die Zahl physisch vorhandener (nicht ausgesonderter)
	// Exemplare eines Titels — die Obergrenze für eine Klassensatz-Reservierung.
	ZaehleTitelBestand(ctx context.Context, titelID string) (int, error)
	LegeAn(ctx context.Context, titelID, klasse string, anzahl int, notiz *string, angefordertVon string, idempotenzSchluessel *string) (id string, neu bool, err error)
	LadeAlle(ctx context.Context) ([]Reservierung, error)
	// LadeOffene liefert die offenen Reservierungen in
	// Warteschlangen-Reihenfolge (älteste zuerst) — die Sicht für das Portal.
	LadeOffene(ctx context.Context) ([]OffeneReservierung, error)
	ZaehleOffene(ctx context.Context) (int, error)
	// Erledige schliesst eine offene Reservierung ab und
	// liefert die Angaben fuer die Bereit-Mail an die anfragende Lehrkraft.
	// notiz ist die Antwort der Bibliothek (landet in der Mail, Migration 088).
	// nil ohne Fehler: war nicht (mehr) offen.
	Erledige(ctx context.Context, id, notiz string) (*ErledigteReservierung, error)
	// LadeEigene liefert die EIGENEN Reservierungen einer Lehrkraft
	// fuers Portal — offene plus kuerzlich abgeschlossene samt Bibliotheks-Notiz (088/089).
	LadeEigene(ctx context.Context, benutzerID string) ([]EigeneReservierung, error)
}

type pgReservierungsAbfragen struct {
	db db.PgxPoolIface
}

// NewReservierungsAbfragen bindet die Vormerkungen an einen Verbindungspool. Gibt die
// Schnittstelle zurück, nicht den konkreten Typ — die Aufrufer sollen gegen den Vertrag
// arbeiten, damit Tests ihn ersetzen können.
func NewReservierungsAbfragen(pool db.PgxPoolIface) ReservierungsAbfragen {
	return &pgReservierungsAbfragen{db: pool}
}

func (a *pgReservierungsAbfragen) IstTitelVorhanden(ctx context.Context, titelID string) (bool, error) {
	var vorhanden bool
	err := a.db.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM buecher_titel WHERE id = $1)`, titelID).Scan(&vorhanden)
	return vorhanden, err
}

// ZaehleTitelBestand zählt die physisch vorhandenen (nicht ausgesonderten) Exemplare eines
// Titels. Eine Klassensatz-Reservierung über diese Zahl hinaus ist grundsätzlich
// unerfüllbar (die Bibliothek besitzt die Bücher schlicht nicht) und bliebe als
// unlösbare Aufgabe im Dashboard hängen.
func (a *pgReservierungsAbfragen) ZaehleTitelBestand(ctx context.Context, titelID string) (int, error) {
	var anzahl int
	err := a.db.QueryRow(ctx, `
		SELECT COUNT(*) FROM buecher_exemplare
		WHERE titel_id = $1 AND ist_ausgesondert = false
	`, titelID).Scan(&anzahl)
	return anzahl, err
}

func (a *pgReservierungsAbfragen) LegeAn(ctx context.Context, titelID, klasse string, anzahl int, notiz *string, angefordertVon string, idempotenzSchluessel *string) (string, bool, error) {
	// Doppelklick-Schutz (Migration 076): Mit demselben idempotenz_schluessel läuft der
	// zweite INSERT am partiellen Unique-Index auf und DO NOTHING liefert keine Zeile —
	// dann geben wir die bereits angelegte Reservierung zurück (No-op, keine zweite Mail).
	// Ohne Schlüssel (NULL) kollidiert nichts, jede Anfrage legt regulär an.
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
		// Konflikt: dieselbe Anfrage lief schon durch — bestehende ID zurückgeben.
		errBestehend := a.db.QueryRow(ctx,
			`SELECT id FROM klassensatz_reservierungen WHERE idempotenz_schluessel = $1`,
			*idempotenzSchluessel).Scan(&id)
		return id, false, errBestehend
	}
	return "", false, err
}

func (a *pgReservierungsAbfragen) LadeAlle(ctx context.Context) ([]Reservierung, error) {
	// Offene Reservierungen in WARTESCHLANGEN-Reihenfolge (älteste zuerst) — wer
	// zuerst reserviert hat, ist zuerst dran; erledigte dahinter, neueste zuerst.
	// verfuegbar zählt wie der OPAC (Paarung!): ausleihbar, nicht ausgesondert,
	// keine offene Ausleihe.
	//
	// Die Zählung steht als LATERAL-Join, nicht in der SELECT-Liste: So kann Postgres sie je
	// Titel zwischenspeichern (Memoize), und mehrere Reservierungen desselben Titels zählen
	// einmal. Eine Unterabfrage in der SELECT-Liste läuft je Zeile. Gemessen am 30.09.2026
	// (PR #694) an 2.000 Reservierungen über 252 Titel: 98 ms → 17 ms, 2.000 → 252 Zählungen.
	// Ein N+1 war es nie — es bleibt eine Abfrage.
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
		var erledigtAm *time.Time // NULL = vor Migration 089 abgeschlossen
		if err := rows.Scan(
			&r.ID, &r.TitelID, &r.TitelName, &r.CoverURL,
			&r.Klasse, &r.Anzahl, &r.Notiz, &r.Erledigt, &t, &r.AngefordertVon, &r.Verfuegbar,
			&r.ErledigtNotiz, &erledigtAm,
		); err != nil {
			continue
		}
		r.ErstelltAm = t.Format(dateFormatDE)
		if erledigtAm != nil {
			r.ErledigtAm = erledigtAm.Format(dateFormatDE)
		}
		reservierungen = append(reservierungen, r)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return reservierungen, nil
}

// LadeOffene — die Warteschlange aus Sicht der Lehrkräfte.
func (a *pgReservierungsAbfragen) LadeOffene(ctx context.Context) ([]OffeneReservierung, error) {
	// LEFT JOIN, nicht INNER: Ein gelöschter Titel darf die Warteschlange der anderen
	// nicht verschwinden lassen — dann steht dort eben ein leerer Name.
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
		o.ErstelltAm = t.Format(dateFormatDE)
		offene = append(offene, o)
	}
	return offene, rows.Err()
}

// EigeneReservierung ist die Portal-Sicht einer Lehrkraft auf die EIGENEN Reservierungen —
// offene und kürzlich abgeschlossene, samt Antwort der Bibliothek. Bis zum 31.08.2026
// verschwand eine Reservierung mit dem Abschluss aus dem Portal, und die Notiz der
// Bibliothek existierte nur in der Bereit-Mail; scheiterte die, war sie unsichtbar.
type EigeneReservierung struct {
	ID            string `json:"id"`
	TitelID       string `json:"titel_id"`
	Titel         string `json:"titel"`
	Klasse        string `json:"klasse"`
	Anzahl        int    `json:"anzahl"`
	Notiz         string `json:"notiz,omitempty"`
	Erledigt      bool   `json:"erledigt"`
	ErledigtNotiz string `json:"erledigt_notiz,omitempty"`
	ErledigtAm    string `json:"erledigt_am,omitempty"` // leer = vor Migration 089 abgeschlossen
	ErstelltAm    string `json:"erstellt_am"`
}

// LadeEigene — eigene offene plus die in den letzten 60 Tagen
// abgeschlossenen (das Portal ist kein Archiv, wie ListEigene beim Anliegen; Altfälle
// ohne erledigt_am bleiben draußen, ihr Zeitpunkt ist nicht rekonstruierbar).
// Offene zuerst, dann neueste zuerst, höchstens 50.
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
		m.ErstelltAm = erstellt.Format(dateFormatDE)
		if erledigtAm != nil {
			m.ErledigtAm = erledigtAm.Format(dateFormatDE)
		}
		meine = append(meine, m)
	}
	return meine, rows.Err()
}

func (a *pgReservierungsAbfragen) ZaehleOffene(ctx context.Context) (int, error) {
	var anzahl int
	err := a.db.QueryRow(ctx, `SELECT COUNT(*) FROM klassensatz_reservierungen WHERE erledigt = false`).Scan(&anzahl)
	return anzahl, err
}

// ErledigteReservierung sind die Angaben fuer die Bereit-Mail nach dem Abschliessen.
type ErledigteReservierung struct {
	TitelName      string
	Klasse         string
	Anzahl         int
	Notiz          string  // was die Lehrkraft beim Reservieren schrieb ('' wenn nichts)
	ErledigtNotiz  string  // Antwort der Bibliothek beim Abschliessen ('' wenn nichts)
	AnfragendeMail *string // nil: ohne Konto angefragt oder Konto geloescht
}

func (a *pgReservierungsAbfragen) Erledige(ctx context.Context, id, notiz string) (*ErledigteReservierung, error) {
	// erledigt = false in der WHERE-Klausel: Der zweite Klick (zweiter Admin) darf
	// weder doppelt abschliessen noch eine zweite Bereit-Mail ausloesen.
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
