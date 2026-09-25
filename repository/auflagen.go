package repository

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"

	"bibliothek/db"

	"github.com/jackc/pgx/v5"
)

// Auflagen eines Schulbuchs (Migration 148, docs/OFFEN.md 4.18). Eine neue Auflage ist ein
// eigener Titel mit eigener ISBN und bleibt es; zusammengefasst werden die Titel, die
// dasselbe Buch sind, über eine gemeinsame werk_id. Gelesen wird über
// COALESCE(werk_id, id) — ein Titel ohne Werk ist sein eigenes.
//
// Diese Datei ist der EINE Schreibpfad von werk_id und werke. Die Titelmaske und der
// Bestellkorb rufen dieselben Funktionen, damit die Regeln an einer Stelle stehen:
//
//   - nur Lernmittel (docs/OFFEN.md 4.18: „Es geht ausdrücklich NUR um Schulbücher");
//   - zwei Gruppen werden eine, wenn zwei ihrer Titel zusammengefasst werden;
//   - ein Werk, an dem weniger als zwei Titel hängen, fällt samt dem letzten Verweis.

// auflagenLockKey reiht das Zusammenfassen und Lösen hintereinander. Beide ändern oft
// mehrere Titel (beim Vereinen zweier Gruppen alle Titel der einen); mit Sperren je Titel
// käme es auf die Reihenfolge an, in der zwei gleichzeitige Aufrufe sie nehmen. Die
// Handlung ist selten und geschieht von Hand — eine Sperre für alle kostet nichts.
const auflagenLockKey int64 = 750_2026

// SQLNeuesteAuflageZuerst ordnet die Auflagen eines Buchs: die jüngste zuerst, nach dem
// Erscheinungsjahr, bei gleichem oder fehlendem Jahr die zuletzt angelegte. Die eine
// Stelle für „die neueste Auflage" — die Liste der Titelmaske zeigt sie oben, und die
// Nachbestell-Liste bestellt sie. Der Titel muss als `b` gebunden sein.
const SQLNeuesteAuflageZuerst = `b.erscheinungsjahr DESC NULLS LAST, b.erstellt_am DESC, b.id`

// ErrAuflageUngueltig meldet eine Zuordnung, die eine Regel verletzt. Die Handler
// antworten mit 400 und reichen den Text weiter.
var ErrAuflageUngueltig = errors.New("auflagen ungültig")

// Auflage ist ein Titel in der Liste der Auflagen eines Buchs, mit seinem Bestand in
// denselben drei Zahlen wie im Katalog (book_bestand.go).
type Auflage struct {
	ID               string `json:"id"`
	Titel            string `json:"titel"`
	Auflage          string `json:"auflage"`
	ISBN             string `json:"isbn"`
	Verlag           string `json:"verlag"`
	Erscheinungsjahr int    `json:"erscheinungsjahr"`
	IstLernmittel    bool   `json:"ist_lernmittel"`
	Gesamt           int    `json:"gesamt"`
	Verfuegbar       int    `json:"verfuegbar"`
	ImZulauf         int    `json:"im_zulauf"`
}

// auflagenKopf ist, was die Regeln über einen Titel wissen müssen.
type auflagenKopf struct {
	id            string
	titel         string
	istLernmittel bool
	werkID        *string
}

// AuflagenDesTitels liefert alle Auflagen des Buchs, zu dem titelID gehört, die neueste
// zuerst — ohne Werk nur den Titel selbst. Unbekannte Titel melden ErrTitelNichtGefunden.
func AuflagenDesTitels(ctx context.Context, q DBQueryer, titelID string) ([]Auflage, error) {
	rows, err := q.Query(ctx, `
		SELECT b.id, b.titel, coalesce(b.auflage, ''), coalesce(b.isbn, ''), coalesce(b.verlag, ''),
		       coalesce(b.erscheinungsjahr, 0), b.ist_lernmittel,
		       `+SQLBestandGesamt+`, `+SQLBestandVerfuegbar+`, `+SQLBestandImZulauf+`
		FROM buecher_titel b
		WHERE b.id = $1
		   OR b.werk_id = (SELECT werk_id FROM buecher_titel WHERE id = $1)
		ORDER BY `+SQLNeuesteAuflageZuerst, titelID)
	if err != nil {
		return nil, fmt.Errorf("auflagen lesen: %w", err)
	}
	defer rows.Close()

	auflagen := make([]Auflage, 0, 2)
	for rows.Next() {
		var a Auflage
		if err := rows.Scan(&a.ID, &a.Titel, &a.Auflage, &a.ISBN, &a.Verlag, &a.Erscheinungsjahr,
			&a.IstLernmittel, &a.Gesamt, &a.Verfuegbar, &a.ImZulauf); err != nil {
			return nil, fmt.Errorf("auflagen lesen: %w", err)
		}
		auflagen = append(auflagen, a)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("auflagen lesen: %w", err)
	}
	if len(auflagen) == 0 {
		return nil, ErrTitelNichtGefunden
	}
	return auflagen, nil
}

// FasseAuflagenZusammen erklärt zwei Titel zu Auflagen desselben Buchs und liefert danach
// die Auflagen des Buchs, zu dem titelID gehört. Hat keiner ein Werk, entsteht eins; hat
// einer eins, kommt der andere dazu; haben beide verschiedene, werden sie eins — alle
// Titel des Werks von andereID wechseln in das von titelID. Sind beide schon zusammen,
// wird nichts geschrieben.
//
// q ist der Pool oder eine laufende Transaktion; die Funktion öffnet ihre eigene (auf einer
// Transaktion ist das ein Savepoint).
func FasseAuflagenZusammen(ctx context.Context, q DBQueryer, titelID, andereID string) ([]Auflage, error) {
	// Eine Kennung kann in Großbuchstaben ankommen (kennung.IstUUID nimmt A-F an); Postgres
	// sieht darin denselben Titel, der Vergleich hier und die Schlüssel unten müssen es auch.
	titelID, andereID = strings.ToLower(titelID), strings.ToLower(andereID)
	if titelID == andereID {
		return nil, fmt.Errorf("%w: ein Titel ist keine andere Auflage seiner selbst", ErrAuflageUngueltig)
	}

	tx, err := q.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("auflagen: transaktion öffnen: %w", err)
	}
	defer db.SafeRollback(ctx, tx)

	koepfe, err := sperreAuflagen(ctx, tx, titelID, andereID)
	if err != nil {
		return nil, err
	}
	ziel, quelle := koepfe[titelID], koepfe[andereID]
	for _, k := range []auflagenKopf{ziel, quelle} {
		if !k.istLernmittel {
			return nil, fmt.Errorf("%w: „%s“ ist kein Lernmittel — zusammengefasst werden nur Auflagen eines Schulbuchs",
				ErrAuflageUngueltig, k.titel)
		}
	}

	switch {
	case ziel.werkID == nil && quelle.werkID == nil:
		var werkID string
		if err := tx.QueryRow(ctx, `INSERT INTO werke DEFAULT VALUES RETURNING id::text`).Scan(&werkID); err != nil {
			return nil, fmt.Errorf("werk anlegen: %w", err)
		}
		if err := setzeWerk(ctx, tx, werkID, ziel.id, quelle.id); err != nil {
			return nil, err
		}
	case quelle.werkID == nil:
		if err := setzeWerk(ctx, tx, *ziel.werkID, quelle.id); err != nil {
			return nil, err
		}
	case ziel.werkID == nil:
		if err := setzeWerk(ctx, tx, *quelle.werkID, ziel.id); err != nil {
			return nil, err
		}
	case *ziel.werkID != *quelle.werkID:
		if err := vereineWerke(ctx, tx, *ziel.werkID, *quelle.werkID); err != nil {
			return nil, err
		}
	}

	auflagen, err := AuflagenDesTitels(ctx, tx, titelID)
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("auflagen: commit: %w", err)
	}
	return auflagen, nil
}

// LoeseAuflage nimmt einen Titel aus seinem Werk und liefert danach seine Auflagen — also
// nur ihn selbst. Bleibt am Werk höchstens ein Titel übrig, wird auch der gelöst und das
// Werk gelöscht: Ein Buch mit einer Auflage ist keine Gruppe. Ein Titel ohne Werk bleibt,
// wie er ist; lösen lässt sich auch ein Titel, der inzwischen kein Lernmittel mehr ist.
func LoeseAuflage(ctx context.Context, q DBQueryer, titelID string) ([]Auflage, error) {
	titelID = strings.ToLower(titelID)
	tx, err := q.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("auflagen: transaktion öffnen: %w", err)
	}
	defer db.SafeRollback(ctx, tx)

	koepfe, err := sperreAuflagen(ctx, tx, titelID)
	if err != nil {
		return nil, err
	}
	if werkID := koepfe[titelID].werkID; werkID != nil {
		if err := setzeWerk(ctx, tx, "", titelID); err != nil {
			return nil, err
		}
		if err := raeumeWerkAuf(ctx, tx, *werkID); err != nil {
			return nil, err
		}
	}

	auflagen, err := AuflagenDesTitels(ctx, tx, titelID)
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("auflagen: commit: %w", err)
	}
	return auflagen, nil
}

// sperreAuflagen nimmt die Sperre für alle Auflagen-Änderungen und dann die Zeilen der
// genannten Titel, in fester Reihenfolge. Unbekannte Titel melden ErrTitelNichtGefunden.
func sperreAuflagen(ctx context.Context, tx pgx.Tx, ids ...string) (map[string]auflagenKopf, error) {
	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock($1)`, auflagenLockKey); err != nil {
		return nil, fmt.Errorf("auflagen sperren: %w", err)
	}
	sortiert := append([]string(nil), ids...)
	sort.Strings(sortiert)
	koepfe := make(map[string]auflagenKopf, len(ids))
	for _, id := range sortiert {
		k := auflagenKopf{id: id}
		err := tx.QueryRow(ctx,
			`SELECT titel, ist_lernmittel, werk_id::text FROM buecher_titel WHERE id = $1 FOR UPDATE`, id).
			Scan(&k.titel, &k.istLernmittel, &k.werkID)
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrTitelNichtGefunden
		}
		if err != nil {
			return nil, fmt.Errorf("titel sperren: %w", err)
		}
		koepfe[id] = k
	}
	return koepfe, nil
}

// setzeWerk hängt die genannten Titel an ein Werk; werkID "" löst sie. Jeder Titel ist
// gesperrt und existiert — trifft die Anweisung weniger Zeilen, stimmt etwas nicht.
func setzeWerk(ctx context.Context, tx pgx.Tx, werkID string, titelIDs ...string) error {
	tag, err := tx.Exec(ctx, `UPDATE buecher_titel SET werk_id = NULLIF($1, '')::uuid WHERE id = ANY($2::uuid[])`,
		werkID, titelIDs)
	if err != nil {
		return fmt.Errorf("werk setzen: %w", err)
	}
	if int(tag.RowsAffected()) != len(titelIDs) {
		return fmt.Errorf("werk setzen: %d von %d titeln geändert", tag.RowsAffected(), len(titelIDs))
	}
	return nil
}

// vereineWerke hängt alle Titel des Werks quelle an das Werk ziel und löscht quelle.
func vereineWerke(ctx context.Context, tx pgx.Tx, ziel, quelle string) error {
	tag, err := tx.Exec(ctx, `UPDATE buecher_titel SET werk_id = $1 WHERE werk_id = $2`, ziel, quelle)
	if err != nil {
		return fmt.Errorf("werke vereinen: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("werke vereinen: am werk %s hängt kein titel", quelle)
	}
	return loescheWerk(ctx, tx, quelle)
}

// raeumeWerkAuf löst den letzten Titel eines Werks und löscht es, sobald weniger als zwei
// Titel daran hängen.
func raeumeWerkAuf(ctx context.Context, tx pgx.Tx, werkID string) error {
	var rest int
	if err := tx.QueryRow(ctx, `SELECT count(*) FROM buecher_titel WHERE werk_id = $1`, werkID).Scan(&rest); err != nil {
		return fmt.Errorf("werk zählen: %w", err)
	}
	if rest >= 2 {
		return nil
	}
	// 0 Zeilen sind hier richtig: Hing nur noch der eben gelöste Titel daran, ist keiner übrig.
	if _, err := tx.Exec(ctx, `UPDATE buecher_titel SET werk_id = NULL WHERE werk_id = $1`, werkID); err != nil {
		return fmt.Errorf("letzten titel lösen: %w", err)
	}
	return loescheWerk(ctx, tx, werkID)
}

// loescheWerk löscht ein Werk, an dem kein Titel mehr hängt.
func loescheWerk(ctx context.Context, tx pgx.Tx, werkID string) error {
	tag, err := tx.Exec(ctx, `DELETE FROM werke WHERE id = $1`, werkID)
	if err != nil {
		return fmt.Errorf("werk löschen: %w", err)
	}
	if tag.RowsAffected() != 1 {
		return fmt.Errorf("werk löschen: werk %s nicht gefunden", werkID)
	}
	return nil
}
