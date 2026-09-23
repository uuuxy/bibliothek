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

// Schlagworte am Titel (Migration 138, docs/OFFEN.md 4.20) — frei eintragbar wie in
// Littera, mit Vorschlägen aus dem Bestand. Diese Datei ist der EINE Schreibpfad: Das
// Buchformular (PUT/POST /api/books) und der Bestellkorb
// (PUT /api/buecher/titel/{id}/schlagworte) rufen beide SetzeSchlagworte, damit die
// Regeln — Form, Grenzen, „vorhandene Schreibweise gewinnt" — an einer Stelle stehen.

const (
	// SchlagwortMaxZeichen hält die Datenbank selbst (chk_schlagwort_form); hier steht
	// sie, damit ein zu langes Wort als 400 mit Text ankommt statt als 500.
	SchlagwortMaxZeichen = 80
	// SchlagworteJeTitelMax begrenzt die Menge eines Titels. Littera kennt keine Grenze;
	// dreißig reichen für jede Hand-Eintragung, und ohne Grenze wäre ein Tippfehler im
	// Aufrufer (ein ganzer Absatz, am Komma zerlegt) eine Liste von Hunderten Wörtern.
	SchlagworteJeTitelMax = 30
	// schlagwortVorschlaegeMax kappt die Vorschlagsliste. Die Häufigsten kommen zuerst;
	// ein seltenes Wort, das darüber fällt, lässt sich weiter tippen und trifft beim
	// Speichern trotzdem die vorhandene Schreibweise.
	schlagwortVorschlaegeMax = 500
)

// ErrSchlagwortUngueltig meldet eine Eingabe, die eine Grenze verletzt. Die Handler
// antworten mit 400 und reichen den Text weiter.
var ErrSchlagwortUngueltig = errors.New("schlagworte ungültig")

// NormalisiereSchlagworte bringt eine Eingabe in die gespeicherte Form: Leerraum
// außen weg und innen zu einem Leerzeichen, Leeres fällt, Doppelte (ohne Rücksicht auf
// Groß- und Kleinschreibung) fallen — das erste gewinnt. Liefert nie nil, damit „keine
// Wörter" als leere Liste weitergeht und nicht als „nichts gesagt".
func NormalisiereSchlagworte(roh []string) ([]string, error) {
	woerter := make([]string, 0, len(roh))
	gesehen := make(map[string]bool, len(roh))
	for _, eingabe := range roh {
		wort := strings.Join(strings.Fields(eingabe), " ")
		if wort == "" {
			continue
		}
		if utf8.RuneCountInString(wort) > SchlagwortMaxZeichen {
			return nil, fmt.Errorf("%w: „%s …“ ist länger als %d Zeichen",
				ErrSchlagwortUngueltig, kuerzeRunen(wort, 30), SchlagwortMaxZeichen)
		}
		schluessel := strings.ToLower(wort)
		if gesehen[schluessel] {
			continue
		}
		gesehen[schluessel] = true
		woerter = append(woerter, wort)
	}
	if len(woerter) > SchlagworteJeTitelMax {
		return nil, fmt.Errorf("%w: höchstens %d Schlagworte je Titel, genannt waren %d",
			ErrSchlagwortUngueltig, SchlagworteJeTitelMax, len(woerter))
	}
	return woerter, nil
}

// SetzeSchlagworte ersetzt die Schlagworte eines Titels durch die genannten und liefert
// sie in der gespeicherten Schreibweise zurück. Eine leere Liste entfernt alle; wer
// nichts ändern will, ruft die Funktion nicht auf (die Aufrufer unterscheiden „nicht
// mitgeschickt" von „leer", siehe inventur.BuchEingabe).
//
// Unbekannte Wörter werden angelegt. Ein Wort, das es in anderer Groß- und
// Kleinschreibung schon gibt, wird NICHT neu angelegt: Wer „fantasy" tippt, bekommt das
// vorhandene „Fantasy". Sonst stünden nach einem Jahr drei Schreibweisen nebeneinander,
// und die Suche fände jeweils nur einen Teil.
//
// q ist der Pool oder eine laufende Transaktion; die Funktion öffnet ihre eigene (auf
// einer Transaktion ist das ein Savepoint). Der Titel wird gesperrt, damit zwei
// gleichzeitige Speichervorgänge nacheinander ersetzen statt sich zu vermischen.
func SetzeSchlagworte(ctx context.Context, q DBQueryer, titelID string, roh []string) ([]string, error) {
	woerter, err := NormalisiereSchlagworte(roh)
	if err != nil {
		return nil, err
	}

	tx, err := q.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("schlagworte: transaktion öffnen: %w", err)
	}
	defer db.SafeRollback(ctx, tx)

	var gesperrt int
	err = tx.QueryRow(ctx, `SELECT 1 FROM buecher_titel WHERE id = $1 FOR UPDATE`, titelID).Scan(&gesperrt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrTitelNichtGefunden
	}
	if err != nil {
		return nil, fmt.Errorf("schlagworte: titel sperren: %w", err)
	}

	// Anlegen, was es noch nicht gibt. Der Konflikt auf lower(wort) ist genau der Fall
	// „gibt es schon in anderer Schreibweise" — dann bleibt die vorhandene Zeile stehen.
	if _, err := tx.Exec(ctx, `
		INSERT INTO schlagworte (wort)
		SELECT unnest($1::text[])
		ON CONFLICT (lower(wort)) DO NOTHING`, woerter); err != nil {
		return nil, fmt.Errorf("schlagworte anlegen: %w", err)
	}

	// Die Menge des Titels ersetzen: Was nicht mehr genannt ist, fällt; was neu ist,
	// kommt dazu. Zwei Anweisungen statt einer schreibenden CTE — ein INSERT in einer CTE
	// ist für die übrige Anweisung unsichtbar, die neuen Wörter fehlten dann hier.
	//
	// Seit Migration 143 kann ein genanntes Wort ein Verweis sein („Tierfantasy" →
	// „Fantasy"): Am Titel hängt dann sein Ziel. COALESCE(verweis_auf, id) ist die eine
	// Stelle, an der das aufgelöst wird — in beiden Anweisungen gleich.
	if _, err := tx.Exec(ctx, `
		DELETE FROM titel_schlagworte ts
		WHERE ts.titel_id = $1
		  AND ts.schlagwort_id NOT IN (
		      SELECT coalesce(s.verweis_auf, s.id) FROM schlagworte s
		      WHERE lower(s.wort) = ANY (SELECT lower(w) FROM unnest($2::text[]) AS w))`,
		titelID, woerter); err != nil {
		return nil, fmt.Errorf("schlagworte entfernen: %w", err)
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO titel_schlagworte (titel_id, schlagwort_id)
		SELECT DISTINCT $1::uuid, coalesce(s.verweis_auf, s.id)
		FROM schlagworte s
		WHERE lower(s.wort) = ANY (SELECT lower(w) FROM unnest($2::text[]) AS w)
		ON CONFLICT DO NOTHING`,
		titelID, woerter); err != nil {
		return nil, fmt.Errorf("schlagworte verbinden: %w", err)
	}

	gespeichert, err := SchlagworteDesTitels(ctx, tx, titelID)
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("schlagworte: commit: %w", err)
	}
	return gespeichert, nil
}

// SchlagworteDesTitels liefert die Schlagworte eines Titels alphabetisch, in der
// gespeicherten Schreibweise. Ohne Wörter eine leere Liste, nie nil: Das Formular
// schickt, was es gelesen hat, zurück — und „leer" muss dort als [] ankommen, nicht als
// „nichts gesagt".
//
// Ein unbekannter Titel ist ErrTitelNichtGefunden, keine leere Liste: Die Abfrage geht
// vom Titel aus, und ohne Titel gibt es keine Zeile. Sonst bekäme ein Aufrufer mit einer
// falschen Kennung „keine Schlagworte" als Erfolg gemeldet.
func SchlagworteDesTitels(ctx context.Context, q DBQueryer, titelID string) ([]string, error) {
	var woerter []string
	err := q.QueryRow(ctx, `
		SELECT coalesce(array_agg(s.wort ORDER BY lower(s.wort)) FILTER (WHERE s.id IS NOT NULL), '{}')
		FROM buecher_titel t
		LEFT JOIN titel_schlagworte ts ON ts.titel_id = t.id
		LEFT JOIN schlagworte s ON s.id = ts.schlagwort_id
		WHERE t.id = $1
		GROUP BY t.id`, titelID).Scan(&woerter)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrTitelNichtGefunden
	}
	if err != nil {
		return nil, fmt.Errorf("schlagworte des titels lesen: %w", err)
	}
	return woerter, nil
}

// SchlagwortZahl ist ein Vorschlag: das Wort und wie viele Titel es tragen.
type SchlagwortZahl struct {
	Wort  string `json:"wort"`
	Titel int    `json:"titel"`
}

// SchlagwortVorschlaege liefert die Wörter, die mindestens ein Titel trägt, die
// häufigsten zuerst, gekappt auf schlagwortVorschlaegeMax. Ein Wort ohne Titel fehlt mit
// Absicht: Es entsteht, wenn jemand ein Wort anlegt und wieder entfernt — meist ein
// Tippfehler, der nicht als Vorschlag weiterleben soll.
func SchlagwortVorschlaege(ctx context.Context, q DBQueryer) ([]SchlagwortZahl, error) {
	rows, err := q.Query(ctx, `
		SELECT s.wort, count(*)::int AS titel
		FROM schlagworte s
		JOIN titel_schlagworte ts ON ts.schlagwort_id = s.id
		GROUP BY s.id, s.wort
		ORDER BY count(*) DESC, lower(s.wort)
		LIMIT $1`, schlagwortVorschlaegeMax)
	if err != nil {
		return nil, fmt.Errorf("schlagwort-vorschläge lesen: %w", err)
	}
	vorschlaege, err := pgx.CollectRows(rows, pgx.RowToStructByPos[SchlagwortZahl])
	if err != nil {
		return nil, fmt.Errorf("schlagwort-vorschläge lesen: %w", err)
	}
	return vorschlaege, nil
}
