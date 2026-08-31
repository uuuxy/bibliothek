package repository

import (
	"context"
	"errors"
	"time"

	"bibliothek/db"

	"github.com/jackc/pgx/v5"
)

// KlassensatzReservierung represents a pending class-set reservation.
type KlassensatzReservierung struct {
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

// KlassensatzOffen ist die schmale Sicht für das Kollegiums-Portal: Wer vor der
// eigenen Reservierung steht (Titel, Klasse, Menge) — bewusst OHNE Personendaten
// des Anfordernden.
//
// Der TITEL fehlte bis zum 23.08.2026, obwohl der Kommentar darüber ihn nannte. Neben
// einem Suchtreffer fiel das nicht auf (das Buch steht ja daneben); auf der Startfläche
// des Portals stand dann „Klasse 8B · 28 Stück" ohne jeden Hinweis, worum es geht. Der
// Titel ist keine Person und steht ohnehin im öffentlichen Katalog.
type KlassensatzOffen struct {
	TitelID    string `json:"titel_id"`
	Titel      string `json:"titel"`
	Klasse     string `json:"klasse"`
	Anzahl     int    `json:"anzahl"`
	ErstelltAm string `json:"erstellt_am"`
}

// ReservationRepository kapselt die Datenbankzugriffe für Reservierungen.
type ReservationRepository interface {
	CheckTitleExists(ctx context.Context, titelID string) (bool, error)
	// CountTitleStock liefert die Zahl physisch vorhandener (nicht ausgesonderter)
	// Exemplare eines Titels — die Obergrenze für eine Klassensatz-Reservierung.
	CountTitleStock(ctx context.Context, titelID string) (int, error)
	CreateKlassensatzReservierung(ctx context.Context, titelID, klasse string, anzahl int, notiz *string, angefordertVon string, idempotenzSchluessel *string) (id string, neu bool, err error)
	GetKlassensatzReservierungen(ctx context.Context) ([]KlassensatzReservierung, error)
	// OffeneKlassensatzReservierungen liefert die offenen Reservierungen in
	// Warteschlangen-Reihenfolge (älteste zuerst) — die Sicht für das Portal.
	OffeneKlassensatzReservierungen(ctx context.Context) ([]KlassensatzOffen, error)
	GetKlassensatzReservierungenAnzahl(ctx context.Context) (int, error)
	// ErledigeKlassensatzReservierung schliesst eine offene Reservierung ab und
	// liefert die Angaben fuer die Bereit-Mail an die anfragende Lehrkraft.
	// notiz ist die Antwort der Bibliothek (landet in der Mail, Migration 088).
	// nil ohne Fehler: war nicht (mehr) offen.
	ErledigeKlassensatzReservierung(ctx context.Context, id, notiz string) (*KlassensatzErledigt, error)
	// MeineKlassensatzReservierungen liefert die EIGENEN Reservierungen einer Lehrkraft
	// fuers Portal — offene plus kuerzlich abgeschlossene samt Bibliotheks-Notiz (088/089).
	MeineKlassensatzReservierungen(ctx context.Context, benutzerID string) ([]KlassensatzMeine, error)
}

type pgReservationRepository struct {
	db db.PgxPoolIface
}

// NewReservationRepository bindet die Vormerkungen an einen Verbindungspool. Gibt die
// Schnittstelle zurück, nicht den konkreten Typ — die Aufrufer sollen gegen den Vertrag
// arbeiten, damit Tests ihn ersetzen können.
func NewReservationRepository(pool db.PgxPoolIface) ReservationRepository {
	return &pgReservationRepository{db: pool}
}

func (r *pgReservationRepository) CheckTitleExists(ctx context.Context, titelID string) (bool, error) {
	var exists bool
	err := r.db.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM buecher_titel WHERE id = $1)`, titelID).Scan(&exists)
	return exists, err
}

// CountTitleStock zählt die physisch vorhandenen (nicht ausgesonderten) Exemplare eines
// Titels. Eine Klassensatz-Reservierung über diese Zahl hinaus ist grundsätzlich
// unerfüllbar (die Bibliothek besitzt die Bücher schlicht nicht) und bliebe als
// unlösbare Aufgabe im Dashboard hängen.
func (r *pgReservationRepository) CountTitleStock(ctx context.Context, titelID string) (int, error) {
	var count int
	err := r.db.QueryRow(ctx, `
		SELECT COUNT(*) FROM buecher_exemplare
		WHERE titel_id = $1 AND ist_ausgesondert = false
	`, titelID).Scan(&count)
	return count, err
}

func (r *pgReservationRepository) CreateKlassensatzReservierung(ctx context.Context, titelID, klasse string, anzahl int, notiz *string, angefordertVon string, idempotenzSchluessel *string) (string, bool, error) {
	// Doppelklick-Schutz (Migration 076): Mit demselben idempotenz_schluessel läuft der
	// zweite INSERT am partiellen Unique-Index auf und DO NOTHING liefert keine Zeile —
	// dann geben wir die bereits angelegte Reservierung zurück (No-op, keine zweite Mail).
	// Ohne Schlüssel (NULL) kollidiert nichts, jede Anfrage legt regulär an.
	var newID string
	err := r.db.QueryRow(ctx, `
		INSERT INTO klassensatz_reservierungen
			(titel_id, klasse, anzahl, notiz, angefordert_von, idempotenz_schluessel)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (idempotenz_schluessel) WHERE idempotenz_schluessel IS NOT NULL DO NOTHING
		RETURNING id
	`, titelID, klasse, anzahl, notiz, angefordertVon, idempotenzSchluessel).Scan(&newID)
	if err == nil {
		return newID, true, nil
	}
	if errors.Is(err, pgx.ErrNoRows) && idempotenzSchluessel != nil {
		// Konflikt: dieselbe Anfrage lief schon durch — bestehende ID zurückgeben.
		errBestehend := r.db.QueryRow(ctx,
			`SELECT id FROM klassensatz_reservierungen WHERE idempotenz_schluessel = $1`,
			*idempotenzSchluessel).Scan(&newID)
		return newID, false, errBestehend
	}
	return "", false, err
}

func (r *pgReservationRepository) GetKlassensatzReservierungen(ctx context.Context) ([]KlassensatzReservierung, error) {
	// Offene Reservierungen in WARTESCHLANGEN-Reihenfolge (älteste zuerst) — wer
	// zuerst reserviert hat, ist zuerst dran; erledigte dahinter, neueste zuerst.
	// verfuegbar zählt wie der OPAC (Paarung!): ausleihbar, nicht ausgesondert,
	// keine offene Ausleihe.
	rows, err := r.db.Query(ctx, `
		SELECT r.id, r.titel_id, t.titel, coalesce(t.cover_url,''),
		       r.klasse, r.anzahl, r.notiz, r.erledigt, r.erstellt_am,
		       CASE WHEN b.id IS NULL THEN NULL
		            ELSE btrim(b.vorname || ' ' || b.nachname) END AS angefordert_von,
		       (SELECT COUNT(*) FROM buecher_exemplare e
		        WHERE e.titel_id = r.titel_id
		          AND e.ist_ausleihbar = true AND e.ist_ausgesondert = false
		          AND NOT EXISTS (SELECT 1 FROM ausleihen a
		                          WHERE a.exemplar_id = e.id AND a.rueckgabe_am IS NULL)
		       ) AS verfuegbar,
		       r.erledigt_notiz, r.erledigt_am
		FROM klassensatz_reservierungen r
		JOIN buecher_titel t ON r.titel_id = t.id
		LEFT JOIN benutzer b ON r.angefordert_von = b.id
		ORDER BY r.erledigt ASC,
		         CASE WHEN r.erledigt THEN r.erstellt_am END DESC,
		         CASE WHEN NOT r.erledigt THEN r.erstellt_am END ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []KlassensatzReservierung
	for rows.Next() {
		var res KlassensatzReservierung
		var t time.Time
		var erledigtAm *time.Time // NULL = vor Migration 089 abgeschlossen
		if err := rows.Scan(
			&res.ID, &res.TitelID, &res.TitelName, &res.CoverURL,
			&res.Klasse, &res.Anzahl, &res.Notiz, &res.Erledigt, &t, &res.AngefordertVon, &res.Verfuegbar,
			&res.ErledigtNotiz, &erledigtAm,
		); err != nil {
			continue
		}
		res.ErstelltAm = t.Format("02.01.2006")
		if erledigtAm != nil {
			res.ErledigtAm = erledigtAm.Format("02.01.2006")
		}
		result = append(result, res)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return result, nil
}

// OffeneKlassensatzReservierungen — die Warteschlange aus Sicht der Lehrkräfte.
func (r *pgReservationRepository) OffeneKlassensatzReservierungen(ctx context.Context) ([]KlassensatzOffen, error) {
	// LEFT JOIN, nicht INNER: Ein gelöschter Titel darf die Warteschlange der anderen
	// nicht verschwinden lassen — dann steht dort eben ein leerer Name.
	rows, err := r.db.Query(ctx, `
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

	offene := []KlassensatzOffen{}
	for rows.Next() {
		var o KlassensatzOffen
		var t time.Time
		if err := rows.Scan(&o.TitelID, &o.Titel, &o.Klasse, &o.Anzahl, &t); err != nil {
			return nil, err
		}
		o.ErstelltAm = t.Format("02.01.2006")
		offene = append(offene, o)
	}
	return offene, rows.Err()
}

// KlassensatzMeine ist die Portal-Sicht einer Lehrkraft auf die EIGENEN Reservierungen —
// offene und kürzlich abgeschlossene, samt Antwort der Bibliothek. Bis zum 31.08.2026
// verschwand eine Reservierung mit dem Abschluss aus dem Portal, und die Notiz der
// Bibliothek existierte nur in der Bereit-Mail; scheiterte die, war sie unsichtbar.
type KlassensatzMeine struct {
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

// MeineKlassensatzReservierungen — eigene offene plus die in den letzten 60 Tagen
// abgeschlossenen (das Portal ist kein Archiv, wie ListEigene beim Anliegen; Altfälle
// ohne erledigt_am bleiben draußen, ihr Zeitpunkt ist nicht rekonstruierbar).
// Offene zuerst, dann neueste zuerst, höchstens 50.
func (r *pgReservationRepository) MeineKlassensatzReservierungen(ctx context.Context, benutzerID string) ([]KlassensatzMeine, error) {
	rows, err := r.db.Query(ctx, `
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

	meine := []KlassensatzMeine{}
	for rows.Next() {
		var m KlassensatzMeine
		var erstellt time.Time
		var erledigtAm *time.Time
		if err := rows.Scan(&m.ID, &m.TitelID, &m.Titel, &m.Klasse, &m.Anzahl,
			&m.Notiz, &m.Erledigt, &m.ErledigtNotiz, &erledigtAm, &erstellt); err != nil {
			return nil, err
		}
		m.ErstelltAm = erstellt.Format("02.01.2006")
		if erledigtAm != nil {
			m.ErledigtAm = erledigtAm.Format("02.01.2006")
		}
		meine = append(meine, m)
	}
	return meine, rows.Err()
}

func (r *pgReservationRepository) GetKlassensatzReservierungenAnzahl(ctx context.Context) (int, error) {
	var count int
	err := r.db.QueryRow(ctx, `SELECT COUNT(*) FROM klassensatz_reservierungen WHERE erledigt = false`).Scan(&count)
	return count, err
}

// KlassensatzErledigt sind die Angaben fuer die Bereit-Mail nach dem Abschliessen.
type KlassensatzErledigt struct {
	TitelName      string
	Klasse         string
	Anzahl         int
	Notiz          string  // was die Lehrkraft beim Reservieren schrieb ('' wenn nichts)
	ErledigtNotiz  string  // Antwort der Bibliothek beim Abschliessen ('' wenn nichts)
	AnfragendeMail *string // nil: ohne Konto angefragt oder Konto geloescht
}

func (r *pgReservationRepository) ErledigeKlassensatzReservierung(ctx context.Context, id, notiz string) (*KlassensatzErledigt, error) {
	// erledigt = false in der WHERE-Klausel: Der zweite Klick (zweiter Admin) darf
	// weder doppelt abschliessen noch eine zweite Bereit-Mail ausloesen.
	row := r.db.QueryRow(ctx, `
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

	var e KlassensatzErledigt
	if err := row.Scan(&e.TitelName, &e.Klasse, &e.Anzahl, &e.Notiz, &e.ErledigtNotiz, &e.AnfragendeMail); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &e, nil
}
