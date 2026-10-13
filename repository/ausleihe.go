package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"bibliothek/db"

	"github.com/jackc/pgx/v5"
)

// Ausleihe ist eine laufende oder beendete Ausleihe (Tabelle ausleihen).
type Ausleihe struct {
	// ID ist die UUID der Ausleihe.
	ID string `json:"id"`
	// ExemplarID ist das ausgeliehene Exemplar; nil bei einem Gerät.
	ExemplarID *string `json:"exemplar_id,omitempty"`
	// GeraetID ist das ausgeliehene Gerät; nil bei einem Buch.
	GeraetID *string `json:"geraet_id,omitempty"`
	// SchuelerID ist der Leser, der ausleiht, Schüler oder Lehrkraft: Die Spalte heißt
	// schueler_id und zeigt auf die Tabelle leser. nil, wenn die Ausleihe von ihrem Leser
	// getrennt ist.
	SchuelerID *string `json:"schueler_id,omitempty"`
	// AusgeliehenAm ist der Zeitpunkt der Ausleihe.
	AusgeliehenAm time.Time `json:"ausgeliehen_am"`
	// RueckgabeFrist ist der späteste Tag der Rückgabe.
	RueckgabeFrist time.Time `json:"rueckgabe_frist"`
	// RueckgabeAm ist der Zeitpunkt der Rückgabe; nil, solange die Ausleihe läuft.
	RueckgabeAm *time.Time `json:"rueckgabe_am,omitempty"`
	// BearbeiterID ist das Konto, das die Ausleihe erfasst hat. Ein Zeiger, weil die Spalte auf
	// ON DELETE SET NULL steht und die Anonymisierung sie leert: NULL ließe sich nicht in einen
	// Text lesen.
	BearbeiterID *string `json:"bearbeiter_id"`
	// RueckgabeBearbeiterID ist das Konto, das die Rückgabe erfasst hat; nil, solange die
	// Ausleihe läuft.
	RueckgabeBearbeiterID *string `json:"rueckgabe_bearbeiter_id,omitempty"`
	// IstFremdrueckgabe sagt, ob jemand anderes als der Leser das Buch zurückgebracht hat.
	IstFremdrueckgabe bool `json:"ist_fremdrueckgabe"`
	// IstHandapparat kennzeichnet die Dauerleihe für das Schuljahr; sie wird nicht überfällig.
	IstHandapparat bool `json:"ist_handapparat"`
}

// ErrAusleiheKonflikt meldet, dass das INSERT am eindeutigen Index (exemplar_id, rueckgabe_am
// IS NULL) abgeprallt ist: Zwischen der Prüfung „Exemplar ist frei" und dem Schreiben hat ein
// anderer Vorgang es verbucht. Ohne den Fehlerwert bekäme der Rufer keine Zeile und keinen
// Fehler und meldete dem Arbeitsplatz eine Ausleihe, die nie geschrieben wurde.
var ErrAusleiheKonflikt = errors.New("exemplar wurde zwischenzeitlich anderweitig verbucht")

// AusleihAbfragen liest und schreibt die Ausleihen und Rückgaben der Exemplare.
type AusleihAbfragen interface {
	// FindeOffene liefert die offene Ausleihe eines Exemplars; nil, wenn es nicht verliehen
	// ist.
	FindeOffene(ctx context.Context, exemplarID string) (*Ausleihe, error)

	// SperreOffene liefert die offene Ausleihe eines Exemplars und sperrt ihre Zeile bis zum
	// Ende der Transaktion (FOR UPDATE): Zwei Scans desselben Exemplars zur selben Zeit laufen
	// nacheinander.
	SperreOffene(ctx context.Context, tx pgx.Tx, exemplarID string) (*Ausleihe, error)

	// BeginneTx beginnt eine Transaktion (Read Committed). Der Rufer rollt sie mit defer zurück
	// und schreibt sie bei Erfolg fest.
	BeginneTx(ctx context.Context) (pgx.Tx, error)

	// LegeAn legt die Ausleihe eines Lesers in der Transaktion an, zum jetzigen Zeitpunkt.
	// istDauerleihe kennzeichnet die Ausleihe für das Schuljahr (ist_handapparat); sie wird
	// nicht überfällig.
	LegeAn(ctx context.Context, tx pgx.Tx, exemplarID, leserID, bearbeiterID string, rueckgabeFrist time.Time, istDauerleihe bool) (*Ausleihe, error)

	// BucheRueckgabe beendet eine offene Ausleihe in der Transaktion, zum jetzigen Zeitpunkt.
	BucheRueckgabe(ctx context.Context, tx pgx.Tx, ausleiheID, bearbeiterID string, istFremdrueckgabe bool) error
}

type pgAusleihAbfragen struct {
	db db.PgxPoolIface
}

// NewAusleihAbfragen liefert die Abfragen der Ausleihe an Postgres.
func NewAusleihAbfragen(db db.PgxPoolIface) AusleihAbfragen {
	return &pgAusleihAbfragen{db: db}
}

// leseAusleihe liest eine Zeile der Tabelle ausleihen in eine Ausleihe.
func leseAusleihe(row scanner) (*Ausleihe, error) {
	var l Ausleihe
	err := row.Scan(
		&l.ID, &l.ExemplarID, &l.SchuelerID, &l.AusgeliehenAm, &l.RueckgabeFrist, &l.RueckgabeAm, &l.BearbeiterID, &l.RueckgabeBearbeiterID, &l.IstFremdrueckgabe, &l.IstHandapparat,
	)
	if err != nil {
		return nil, err
	}
	return &l, nil
}

// BeginneTx beginnt eine Transaktion mit Read Committed: Sie sieht keine ungeschriebenen Daten
// anderer, und Scans an mehreren Arbeitsplätzen halten einander nicht auf.
func (a *pgAusleihAbfragen) BeginneTx(ctx context.Context) (pgx.Tx, error) {
	return a.db.BeginTx(ctx, pgx.TxOptions{
		IsoLevel:   pgx.ReadCommitted,
		AccessMode: pgx.ReadWrite,
	})
}

// FindeOffene liest die offene Ausleihe ohne Sperre.
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

// SperreOffene liest die offene Ausleihe mit FOR UPDATE: Zwei Scans desselben Exemplars im
// selben Augenblick verarbeiten es nicht doppelt.
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

// sqlAusleihe schreibt die Ausleihe für den Scan an der Theke und für das Nachbuchen: Der
// Zeitpunkt ist ein Parameter. nil heißt jetzt (Scan an der Theke, Vorgabe der Spalten); das
// Nachbuchen gibt den Zeitpunkt des Scans vom Theken-Rechner mit, höchstens die Serverzeit, das
// prüft der Rufer. Zwei Anweisungen für dieselbe Buchung (eine mit, eine ohne Zeit) wären zwei
// Wege zum selben Zustand.
const sqlAusleihe = `
		INSERT INTO ausleihen (exemplar_id, schueler_id, rueckgabe_frist, bearbeiter_id, ist_handapparat, ausgeliehen_am, erfasst_am)
		VALUES ($1, $2, $3, $4, $5, COALESCE($6::timestamptz, CURRENT_TIMESTAMP), COALESCE($6::timestamptz, CURRENT_TIMESTAMP))
		ON CONFLICT DO NOTHING
		RETURNING id, exemplar_id, schueler_id, ausgeliehen_am, rueckgabe_frist, rueckgabe_am, bearbeiter_id, rueckgabe_bearbeiter_id, ist_fremdrueckgabe, ist_handapparat
	`

// AusleiheNeu trägt die Angaben einer neuen Ausleihe. Eine Struktur, weil Exemplar, Leser und
// Bearbeiter drei Kennungen vom selben Typ sind: In der Reihenfolge vertauscht, liefe die
// Buchung ohne Fehler auf die falsche Person. Zeitpunkt nil heißt jetzt.
type AusleiheNeu struct {
	ExemplarID     string
	LeserID        string
	BearbeiterID   string
	RueckgabeFrist time.Time
	IstDauerleihe  bool
	Zeitpunkt      *time.Time
}

// LegeAn legt die Ausleihe in der Transaktion an, zum jetzigen Zeitpunkt.
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

// LegeAusleiheAn legt die Ausleihe eines Lesers zum genannten Zeitpunkt an (nil heißt jetzt)
// und stempelt die Bewegung des Exemplars mit demselben Zeitpunkt.
func LegeAusleiheAn(ctx context.Context, tx pgx.Tx, neu AusleiheNeu) (*Ausleihe, error) {
	return schreibeAusleihe(ctx, tx, neu.ExemplarID, neu.Zeitpunkt, sqlAusleihe, neu.ExemplarID, neu.LeserID, neu.RueckgabeFrist, neu.BearbeiterID, neu.IstDauerleihe, neu.Zeitpunkt)
}

// schreibeAusleihe führt das INSERT aus und stempelt die Bewegung des Exemplars.
func schreibeAusleihe(ctx context.Context, tx pgx.Tx, exemplarID string, zeitpunkt *time.Time, query string, args ...any) (*Ausleihe, error) {
	l, err := leseAusleihe(tx.QueryRow(ctx, query, args...))
	if err != nil {
		// Keine Zeile heißt hier nicht „nichts gefunden": ON CONFLICT hat das INSERT verworfen,
		// ein Konflikt.
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

// BucheRueckgabe beendet die offene Ausleihe in der Transaktion, zum jetzigen Zeitpunkt.
func (a *pgAusleihAbfragen) BucheRueckgabe(ctx context.Context, tx pgx.Tx, ausleiheID, bearbeiterID string, istFremdrueckgabe bool) error {
	return BucheRueckgabeZum(ctx, tx, ausleiheID, bearbeiterID, istFremdrueckgabe, nil)
}

// BucheRueckgabeZum bucht die Rückgabe zum genannten Zeitpunkt (nil heißt jetzt). Liegt der
// Zeitpunkt vor der Ausleihe, weist check_return_date (23514) die Rückgabe ab; beim Nachbuchen
// ist das ein veralteter Scan, kein Serverfehler.
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
	// Der Bewegungsstempel am Exemplar. Diese Funktion bucht Bücher, Geräte gehen über
	// internal/service/device_service.go; die Ausleihe trägt hier also immer ein Exemplar, und
	// keine getroffene Zeile ist ein Widerspruch.
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
