package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"unicode/utf8"

	"bibliothek/db"

	"github.com/jackc/pgx/v5"
)

// Der Standort eines Exemplars (docs/OFFEN.md 5.53, Migration 158): wo es steht, wenn nicht an
// seinem Platz nach der Signatur. Die Übernahme aus Littera schreibt ihn beim Anlegen
// (internal/littera); von Hand wird er geändert wie das Eigentum: Exemplare eines Titels
// markieren, einen Wert für alle setzen. Ein einzelnes Exemplar ist eine Liste mit einem
// Eintrag.
//
// Der Standort wirkt auf keine Rechnung und kein Papier. Einen Grund verlangt die Tür deshalb
// nicht; der alte und der neue Wert stehen im Protokoll, damit eine Änderung für viele
// Exemplare zurückzuverfolgen ist.

// ExemplarStandortMaxZeichen ist die Länge, die chk_exemplar_standort zulässt (wie Litteras Feld).
const ExemplarStandortMaxZeichen = 255

// ExemplarStandortListeMax begrenzt eine Änderung wie ExemplarEigentumListeMax: Beide ändern,
// was die Buchakte markiert.
const ExemplarStandortListeMax = ExemplarEigentumListeMax

// ExemplarStandorteMax kappt die Vorschlagsliste: Freitext kann sie wachsen lassen, und die
// seltensten Werte fallen dann zuerst heraus.
const ExemplarStandorteMax = 1000

// ErrStandortUngueltig ist eine Eingabe, die die Tür nicht annimmt (400).
var ErrStandortUngueltig = errors.New("standort ungültig")

// StandortAenderung ist eine Änderung für mehrere Exemplare.
type StandortAenderung struct {
	ExemplarIDs []string
	// Standort leer nimmt die Angabe weg: Das Exemplar steht dann wieder nach der Signatur.
	Standort     string
	BearbeiterID string
}

// StandortZahl ist ein Standort mit der Zahl der Exemplare im Bestand, die ihn tragen.
type StandortZahl struct {
	Standort string `json:"standort"`
	Anzahl   int    `json:"anzahl"`
}

// SetzeExemplarStandort setzt den Standort der genannten Exemplare und liefert, wie viele sich
// geändert haben. Alle oder keins: Fehlt eine Kennung, ändert sich nichts
// (ErrExemplarNichtGefunden). Ein Exemplar, das schon so steht, wird nicht noch einmal
// geschrieben und nicht protokolliert.
func SetzeExemplarStandort(ctx context.Context, q DBQueryer, a StandortAenderung) (int, error) {
	// Die Datenbank lässt Leerraum am Rand eines Werts zu; ohne das Kürzen stünde derselbe
	// Standort zweimal in der Vorschlagsliste.
	a.Standort = strings.TrimSpace(a.Standort)
	if err := pruefeStandortAenderung(a); err != nil {
		return 0, err
	}

	tx, err := q.Begin(ctx)
	if err != nil {
		return 0, fmt.Errorf("standort: transaktion öffnen: %w", err)
	}
	defer db.SafeRollback(ctx, tx)

	// Wie viele verschiedene Exemplare gewählt sind, zählt die Datenbank — sie liest Groß-
	// und Kleinschreibung einer Kennung als dieselbe (wie SetzeExemplarEigentum).
	var gewaehlt int
	if err := tx.QueryRow(ctx, `SELECT count(DISTINCT k)::int FROM unnest($1::uuid[]) k`, a.ExemplarIDs).
		Scan(&gewaehlt); err != nil {
		return 0, fmt.Errorf("standort: kennungen zählen: %w", err)
	}
	alle, err := sperreStandortStand(ctx, tx, a.ExemplarIDs)
	if err != nil {
		return 0, err
	}
	if len(alle) != gewaehlt {
		return 0, ErrExemplarNichtGefunden
	}

	geaendert := 0
	for _, s := range alle {
		if s.standort == a.Standort {
			continue
		}
		if err := schreibeStandort(ctx, tx, s, a); err != nil {
			return 0, err
		}
		geaendert++
	}
	if err := tx.Commit(ctx); err != nil {
		return 0, fmt.Errorf("standort: commit: %w", err)
	}
	return geaendert, nil
}

// standortStand ist der Standort eines Exemplars vor der Änderung.
type standortStand struct{ id, standort string }

// sperreStandortStand sperrt die Exemplare in fester Reihenfolge, damit sich zwei Änderungen
// mit überlappender Auswahl nicht verklemmen, und liest dabei den alten Wert fürs Protokoll.
func sperreStandortStand(ctx context.Context, tx pgx.Tx, exemplarIDs []string) ([]standortStand, error) {
	rows, err := tx.Query(ctx, `
		SELECT id::text, coalesce(standort, '')
		FROM buecher_exemplare WHERE id = ANY($1::uuid[]) ORDER BY id FOR UPDATE`, exemplarIDs)
	if err != nil {
		return nil, fmt.Errorf("standort: sperren: %w", err)
	}
	var alle []standortStand
	for rows.Next() {
		var s standortStand
		if err := rows.Scan(&s.id, &s.standort); err != nil {
			rows.Close()
			return nil, fmt.Errorf("standort: alten stand lesen: %w", err)
		}
		alle = append(alle, s)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("standort: alten stand lesen: %w", err)
	}
	return alle, nil
}

// schreibeStandort setzt den Standort eines Exemplars und schreibt den Eintrag mit altem und
// neuem Wert ins Protokoll.
func schreibeStandort(ctx context.Context, tx pgx.Tx, s standortStand, a StandortAenderung) error {
	tag, err := tx.Exec(ctx, `UPDATE buecher_exemplare SET standort = NULLIF($2, '') WHERE id = $1`,
		s.id, a.Standort)
	if err != nil {
		return fmt.Errorf("standort: exemplar %s: %w", s.id, err)
	}
	if tag.RowsAffected() != 1 {
		return fmt.Errorf("standort: exemplar %s: %d zeilen statt 1", s.id, tag.RowsAffected())
	}
	audit := &pgAuditRepository{}
	kontext := "Standort geändert"
	if err := audit.insertAuditLog(ctx, tx, auditEntry{
		Tabelle: "buecher_exemplare", Aktion: "UPDATE", DatensatzID: s.id,
		BearbeiterID: &a.BearbeiterID, Akteur: "USER", Kontext: &kontext,
		Details: map[string]any{"standort_alt": s.standort, "standort_neu": a.Standort},
	}); err != nil {
		return fmt.Errorf("standort: protokoll: %w", err)
	}
	return nil
}

// pruefeStandortAenderung prüft die Eingabe, bevor die Datenbank gefragt wird. Die Länge
// zählt in Zeichen wie char_length in chk_exemplar_standort.
func pruefeStandortAenderung(a StandortAenderung) error {
	switch {
	case len(a.ExemplarIDs) == 0:
		return fmt.Errorf("%w: kein Exemplar gewählt", ErrStandortUngueltig)
	case len(a.ExemplarIDs) > ExemplarStandortListeMax:
		return fmt.Errorf("%w: höchstens %d Exemplare auf einmal", ErrStandortUngueltig, ExemplarStandortListeMax)
	case utf8.RuneCountInString(a.Standort) > ExemplarStandortMaxZeichen:
		return fmt.Errorf("%w: höchstens %d Zeichen", ErrStandortUngueltig, ExemplarStandortMaxZeichen)
	case a.BearbeiterID == "":
		return fmt.Errorf("%w: ohne Bearbeiter", ErrStandortUngueltig)
	}
	return nil
}

// ExemplarStandorte nennt die Standorte, die an Exemplaren im Bestand vorkommen, den
// häufigsten zuerst — die Vorschläge des Dialogs „Standort ändern". Ein Wert, den nur noch
// ausgesonderte Exemplare tragen, wird nicht mehr vorgeschlagen.
func ExemplarStandorte(ctx context.Context, q DBQueryer) ([]StandortZahl, error) {
	rows, err := q.Query(ctx, `
		SELECT e.standort, count(*)::int
		FROM buecher_exemplare e
		WHERE e.standort IS NOT NULL AND `+SQLExemplarImBestand+`
		GROUP BY e.standort
		ORDER BY count(*) DESC, e.standort
		LIMIT $1`, ExemplarStandorteMax)
	if err != nil {
		return nil, fmt.Errorf("standorte lesen: %w", err)
	}
	defer rows.Close()
	liste := []StandortZahl{}
	for rows.Next() {
		var z StandortZahl
		if err := rows.Scan(&z.Standort, &z.Anzahl); err != nil {
			return nil, fmt.Errorf("standorte lesen: %w", err)
		}
		liste = append(liste, z)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("standorte lesen: %w", err)
	}
	return liste, nil
}

// StandorteDerTitel zählt je Titel die Standorte seiner Exemplare im Bestand, für die Spalte
// „Standort" der Titel-Verwaltung. Dieselbe Grenze wie die Zahl „Bestand" daneben
// (SQLExemplarImBestand): Die Zahlen einer Zeile sind höchstens so groß wie ihr Bestand. Ein
// Titel ohne Standort steht nicht in der Antwort.
func StandorteDerTitel(ctx context.Context, q DBQueryer, titelIDs []string) (map[string][]StandortZahl, error) {
	jeTitel := make(map[string][]StandortZahl)
	if len(titelIDs) == 0 {
		return jeTitel, nil
	}
	rows, err := q.Query(ctx, `
		SELECT e.titel_id::text, e.standort, count(*)::int
		FROM buecher_exemplare e
		WHERE e.titel_id = ANY($1::uuid[]) AND e.standort IS NOT NULL AND `+SQLExemplarImBestand+`
		GROUP BY e.titel_id, e.standort
		ORDER BY e.titel_id, count(*) DESC, e.standort`, titelIDs)
	if err != nil {
		return nil, fmt.Errorf("standorte der titel lesen: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var titelID string
		var z StandortZahl
		if err := rows.Scan(&titelID, &z.Standort, &z.Anzahl); err != nil {
			return nil, fmt.Errorf("standorte der titel lesen: %w", err)
		}
		jeTitel[titelID] = append(jeTitel[titelID], z)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("standorte der titel lesen: %w", err)
	}
	return jeTitel, nil
}
