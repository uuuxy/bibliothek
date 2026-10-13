package repository

import (
	"bibliothek/db"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

// Ausleihe repräsentiert einen aktiven oder historischen Ausleiheintrag in der Datenbank (Tabelle `ausleihen`).
type Ausleihe struct {
	// ID ist die eindeutige ID (UUID) des Ausleihvorgangs.
	ID string `json:"id"`
	// ExemplarID verweist auf das ausgeliehene Buch (null bei Geräteausleihe).
	ExemplarID *string `json:"exemplar_id,omitempty"`
	// GeraetID verweist auf das ausgeliehene Hardware-Gerät (null bei Buchausleihe).
	GeraetID *string `json:"geraet_id,omitempty"`
	// SchuelerID verweist auf den ausleihenden Schüler (null bei Ausleihe an Lehrkraft).
	SchuelerID *string `json:"schueler_id,omitempty"`
	// AusleiherBenutzerID verweist auf die ausleihende Lehrkraft (null bei Ausleihe an Schüler).
	// AusgeliehenAm ist der genaue Zeitpunkt der Ausleihe.
	AusgeliehenAm time.Time `json:"ausgeliehen_am"`
	// RueckgabeFrist definiert den spätesten Abgabetermin.
	RueckgabeFrist time.Time `json:"rueckgabe_frist"`
	// RueckgabeAm ist der tatsächliche Rückgabezeitpunkt (null bei laufenden Ausleihen).
	RueckgabeAm *time.Time `json:"rueckgabe_am,omitempty"`
	// BearbeiterID verweist auf den Bibliotheksmitarbeiter, der die Ausleihe erfasst hat.
	// Nullbar: die DB-Spalte ist ON DELETE SET NULL, und der GDPR-Anonymisierungs-Job
	// setzt sie auf NULL. Als nicht-nullbarer string scheiterte der Scan mit
	// "cannot scan NULL into *string" → HTTP 500 im Kiosk (Scan/Rückgabe).
	BearbeiterID *string `json:"bearbeiter_id"`
	// RueckgabeBearbeiterID verweist auf den Mitarbeiter, der die Rückgabe erfasst hat (null bei laufenden Ausleihen).
	RueckgabeBearbeiterID *string `json:"rueckgabe_bearbeiter_id,omitempty"`
	// IstFremdrueckgabe zeigt an, ob das Buch von einer anderen Person zurückgebracht wurde.
	IstFremdrueckgabe bool `json:"ist_fremdrueckgabe"`
	// IstHandapparat markiert Dauerleihen (meist an Lehrer) zur Präsenznutzung im Unterricht.
	IstHandapparat bool `json:"ist_handapparat"`
}

// ErrAusleiheKonflikt meldet, dass der INSERT am eindeutigen Index
// (exemplar_id, rueckgabe_am IS NULL) abgeprallt ist: Zwischen der Prüfung
// "Exemplar ist frei" und dem Schreiben hat ein anderer Vorgang es verbucht.
//
// Vorher gaben die Create*-Funktionen in diesem Fall (nil, nil) zurück — kein Fehler,
// keine Zeile. Der Aufrufer prüfte nur err, hielt das für Erfolg, committete und
// meldete dem Arbeitsplatz eine Ausleihe, die nie geschrieben wurde. Genau diese
// Bugklasse: HTTP 200 trotz Datenverlust.
var ErrAusleiheKonflikt = errors.New("exemplar wurde zwischenzeitlich anderweitig verbucht")

// AusleihAbfragen verwaltet alle Datenbank-Interaktionen für Ausleihen und Rückgaben (Bücher und Geräte).
type AusleihAbfragen interface {
	// FindeOffene sucht die aktuell aktive (nicht zurückgegebene) Ausleihe für ein Buchexemplar.
	// Gibt nil zurück, wenn das Exemplar aktuell nicht verliehen ist.
	FindeOffene(ctx context.Context, exemplarID string) (*Ausleihe, error)

	// SperreOffene sucht die aktive Ausleihe innerhalb einer Transaktion und setzt
	// einen Row-Level-Lock (SELECT ... FOR UPDATE). Dies verhindert Race Conditions bei zeitgleichen Scans.
	SperreOffene(ctx context.Context, tx pgx.Tx, exemplarID string) (*Ausleihe, error)

	// BeginneTx startet eine neue Datenbanktransaktion mit dem Isolationslevel 'Read Committed'.
	// Aufrufer müssen defer tx.Rollback(ctx) aufrufen und bei Erfolg tx.Commit(ctx) ausführen.
	BeginneTx(ctx context.Context) (pgx.Tx, error)

	// LegeAn legt einen neuen Ausleihdatensatz für einen LESER innerhalb einer
	// laufenden Transaktion an. istDauerleihe kennzeichnet die Ausleihe fürs Schuljahr
	// (ist_handapparat) — sie zählt nicht in die Überfällig-Automatik.
	//
	// Bis Migration 125 gab es hier zwei Funktionen mit zwei SQL-Anweisungen, weil ein
	// Schüler in einer anderen Spalte stand als eine Lehrkraft.
	LegeAn(ctx context.Context, tx pgx.Tx, exemplarID, leserID, bearbeiterID string, rueckgabeFrist time.Time, istDauerleihe bool) (*Ausleihe, error)

	// BucheRueckgabe markiert eine aktive Ausleihe als zurückgegeben innerhalb einer Transaktion.
	BucheRueckgabe(ctx context.Context, tx pgx.Tx, ausleiheID, bearbeiterID string, istFremdrueckgabe bool) error
}

// pgAusleihAbfragen implementiert das AusleihAbfragen für PostgreSQL.
type pgAusleihAbfragen struct {
	db db.PgxPoolIface
}

// NewAusleihAbfragen erstellt eine neue Instanz des PostgreSQL-basierten Loan-Repositorys.
func NewAusleihAbfragen(db db.PgxPoolIface) AusleihAbfragen {
	return &pgAusleihAbfragen{db: db}
}

// leseAusleihe liest eine Tabellenzeile in ein Loan-Modellobjekt ein.
func leseAusleihe(row Scanner) (*Ausleihe, error) {
	var l Ausleihe
	err := row.Scan(
		&l.ID, &l.ExemplarID, &l.SchuelerID, &l.AusgeliehenAm, &l.RueckgabeFrist, &l.RueckgabeAm, &l.BearbeiterID, &l.RueckgabeBearbeiterID, &l.IstFremdrueckgabe, &l.IstHandapparat,
	)
	if err != nil {
		return nil, err
	}
	return &l, nil
}

// BeginneTx startet eine Transaktion mit dem Isolationslevel 'Read Committed'.
// Dieses Level ist ideal für das Ausleihsystem, da es Schmutzdaten (Dirty Reads) verhindert,
// gleichzeitig aber hohen Durchsatz bei parallelen Scanner-Anfragen ermöglicht.
func (a *pgAusleihAbfragen) BeginneTx(ctx context.Context) (pgx.Tx, error) {
	return a.db.BeginTx(ctx, pgx.TxOptions{
		IsoLevel:   pgx.ReadCommitted,
		AccessMode: pgx.ReadWrite,
	})
}

// FindeOffene ruft die aktive Ausleihe ohne Sperre (schreibgeschützt) ab.
func (a *pgAusleihAbfragen) FindeOffene(ctx context.Context, exemplarID string) (*Ausleihe, error) {
	query := `
		SELECT id, exemplar_id, schueler_id, ausgeliehen_am, rueckgabe_frist, rueckgabe_am, bearbeiter_id, rueckgabe_bearbeiter_id, ist_fremdrueckgabe, ist_handapparat
		FROM ausleihen
		WHERE exemplar_id = $1 AND rueckgabe_am IS NULL
		LIMIT 1
	`
	l, err := leseAusleihe(a.db.QueryRow(ctx, query, exemplarID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return l, nil
}

// SperreOffene ruft die aktive Ausleihe innerhalb einer Transaktion mit 'SELECT ... FOR UPDATE' ab.
// Diese Zeilensperrung (Row-Level-Lock) verhindert, dass parallele Scanner-Anfragen
// dasselbe Exemplar innerhalb desselben Millisekundenfensters doppelt verarbeiten.
func (a *pgAusleihAbfragen) SperreOffene(ctx context.Context, tx pgx.Tx, exemplarID string) (*Ausleihe, error) {
	query := `
		SELECT id, exemplar_id, schueler_id, ausgeliehen_am, rueckgabe_frist, rueckgabe_am, bearbeiter_id, rueckgabe_bearbeiter_id, ist_fremdrueckgabe, ist_handapparat
		FROM ausleihen
		WHERE exemplar_id = $1 AND rueckgabe_am IS NULL
		LIMIT 1
		FOR UPDATE
	`
	l, err := leseAusleihe(tx.QueryRow(ctx, query, exemplarID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return l, nil
}

// Die drei Schreiber der Ausleihe teilen sich je EINE SQL-Formulierung mit dem Online-Scan
// und dem Nachbuchen (Stufe 2 des Offline-Baus, Commit 11): Der Zeitpunkt ist ein Parameter.
// nil heißt „jetzt" (Online-Scan, Default der Spalten); das Nachbuchen gibt den Scan-Zeitpunkt
// vom Theken-Rechner mit — höchstens Serverzeit, das prüft der Aufrufer. Zwei Formulierungen
// derselben Buchung (eine mit, eine ohne Zeit) wären zwei Türen zum selben Zustand.

const sqlAusleihe = `
		INSERT INTO ausleihen (exemplar_id, schueler_id, rueckgabe_frist, bearbeiter_id, ist_handapparat, ausgeliehen_am, erfasst_am)
		VALUES ($1, $2, $3, $4, $5, COALESCE($6::timestamptz, CURRENT_TIMESTAMP), COALESCE($6::timestamptz, CURRENT_TIMESTAMP))
		ON CONFLICT DO NOTHING
		RETURNING id, exemplar_id, schueler_id, ausgeliehen_am, rueckgabe_frist, rueckgabe_am, bearbeiter_id, rueckgabe_bearbeiter_id, ist_fremdrueckgabe, ist_handapparat
	`

// AusleiheNeu trägt die Angaben einer Ausleihe. Als Struktur, weil Exemplar, Leser
// und Bearbeiter drei Kennungen vom selben Typ sind — positionsweise vertauscht, liefe die
// Buchung ohne Fehler auf die falsche Person. Zeitpunkt nil heißt jetzt.
type AusleiheNeu struct {
	ExemplarID     string
	LeserID        string
	BearbeiterID   string
	RueckgabeFrist time.Time
	IstDauerleihe  bool
	Zeitpunkt      *time.Time
}

// LegeAn erzeugt einen neuen Ausleiheintrag innerhalb einer Transaktion — jetzt.
func (a *pgAusleihAbfragen) LegeAn(ctx context.Context, tx pgx.Tx, exemplarID, leserID, bearbeiterID string, rueckgabeFrist time.Time, istDauerleihe bool) (*Ausleihe, error) {
	return LegeAusleiheAn(ctx, tx, AusleiheNeu{
		ExemplarID:     exemplarID,
		LeserID:        leserID,
		BearbeiterID:   bearbeiterID,
		RueckgabeFrist: rueckgabeFrist,
		IstDauerleihe:  istDauerleihe,
		Zeitpunkt:      nil,
	})
}

// LegeAusleiheAn erzeugt die Ausleihe eines Lesers zum gegebenen Zeitpunkt (nil = jetzt)
// und stempelt die Bewegung des Exemplars mit demselben Zeitpunkt.
func LegeAusleiheAn(ctx context.Context, tx pgx.Tx, neu AusleiheNeu) (*Ausleihe, error) {
	return schreibeAusleihe(ctx, tx, neu.ExemplarID, neu.Zeitpunkt, sqlAusleihe, neu.ExemplarID, neu.LeserID, neu.RueckgabeFrist, neu.BearbeiterID, neu.IstDauerleihe, neu.Zeitpunkt)
}

func schreibeAusleihe(ctx context.Context, tx pgx.Tx, exemplarID string, zeitpunkt *time.Time, query string, args ...any) (*Ausleihe, error) {
	l, err := leseAusleihe(tx.QueryRow(ctx, query, args...))
	if err != nil {
		// Keine Zeile heißt hier NICHT "nichts gefunden", sondern "ON CONFLICT hat
		// den INSERT verworfen" — ein Konflikt, kein Normalfall.
		if errors.Is(err, sql.ErrNoRows) || errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrAusleiheKonflikt
		}
		return nil, err
	}
	if err := stempleBewegungZum(ctx, tx, exemplarID, zeitpunkt); err != nil {
		return nil, err
	}
	return l, nil
}

// BucheRueckgabe bucht ein ausgeliehenes Buch innerhalb einer Transaktion zurück — jetzt.
func (a *pgAusleihAbfragen) BucheRueckgabe(ctx context.Context, tx pgx.Tx, ausleiheID, bearbeiterID string, istFremdrueckgabe bool) error {
	return BucheRueckgabeZum(ctx, tx, ausleiheID, bearbeiterID, istFremdrueckgabe, nil)
}

// BucheRueckgabeZum bucht die Rückgabe zum gegebenen Zeitpunkt (nil = jetzt). Liegt der
// Zeitpunkt vor der Ausleihe, weist check_return_date (23514) die Rückgabe ab — beim
// Nachbuchen ist das ein veralteter Scan, kein Serverfehler.
func BucheRueckgabeZum(ctx context.Context, tx pgx.Tx, ausleiheID, bearbeiterID string, istFremdrueckgabe bool, zeitpunkt *time.Time) error {
	tag, err := tx.Exec(ctx, `
		UPDATE ausleihen
		SET rueckgabe_am = COALESCE($4::timestamptz, CURRENT_TIMESTAMP), rueckgabe_bearbeiter_id = $1, ist_fremdrueckgabe = $2
		WHERE id = $3 AND rueckgabe_am IS NULL
	`, bearbeiterID, istFremdrueckgabe, ausleiheID, zeitpunkt)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return errors.New("loan not active or already returned")
	}
	// Bewegungsstempel am Exemplar (Migration 116). Diese Funktion bucht Bücher; Geräte gehen
	// über device_service — die Ausleihe hier trägt also immer ein Exemplar, 0 Zeilen wäre ein
	// Widerspruch und kein Normalfall.
	stempel, err := tx.Exec(ctx, `
		UPDATE buecher_exemplare e SET letzte_bewegung_am = `+sqlStempelVor(`$2`)+`
		FROM ausleihen a WHERE a.id = $1 AND e.id = a.exemplar_id`, ausleiheID, zeitpunkt)
	if err != nil {
		return fmt.Errorf("bewegung stempeln: %w", err)
	}
	if stempel.RowsAffected() == 0 {
		return ErrExemplarNichtGefunden
	}
	return nil
}
