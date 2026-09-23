package repository

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"

	"bibliothek/db"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// Pflege der Schlagworte (Migration 143, docs/OFFEN.md 4.20 Stufe 1): umbenennen,
// zusammenführen, löschen, Verweise und die Filter-Markierung. Das ist die zweite
// Schreib-Tür der Tabelle schlagworte neben SetzeSchlagworte (am Titel); beide halten
// dieselben Regeln, die Migration 143 zusätzlich in der Datenbank festschreibt: kein
// Verweis auf sich selbst, keine Kette, kein Titel an einem Verweis, kein Verweis als Filter.

// schlagwortPflegeListeMax kappt die Pflegeliste. Eine Schülerbücherei hat Hunderte
// Schlagworte, nicht Zehntausende; die Kappung schützt die Tür, und die Antwort sagt, wie
// viele es insgesamt sind.
const schlagwortPflegeListeMax = 5000

var (
	// ErrSchlagwortNichtGefunden meldet eine Kennung, die zu keinem Schlagwort gehört.
	ErrSchlagwortNichtGefunden = errors.New("schlagwort nicht gefunden")
	// ErrSchlagwortGibtEs meldet ein Umbenennen auf ein Wort, das es schon gibt — das ist ein
	// Zusammenführen und muss als solches gewählt werden.
	ErrSchlagwortGibtEs = errors.New("schlagwort gibt es schon")
	// ErrSchlagwortRegel meldet eine verletzte Regel der Verweise (Kette, Filter an einem
	// Verweis, Zusammenführen mit sich selbst).
	ErrSchlagwortRegel = errors.New("regel der schlagworte verletzt")
)

// SchlagwortPflegeZeile ist eine Zeile der Pflegeseite.
type SchlagwortPflegeZeile struct {
	ID   string `json:"id"`
	Wort string `json:"wort"`
	// Titel: wie viele Titel das Wort tragen. Ein Verweis trägt nie welche.
	Titel int `json:"titel"`
	// VerweisAuf: das Ziel, wenn das Wort ein Verweis ist; sonst leer.
	VerweisAufID string `json:"verweis_auf_id,omitempty"`
	VerweisAuf   string `json:"verweis_auf,omitempty"`
	// Verweise: die Wörter, die auf dieses zeigen, alphabetisch; nie nil.
	Verweise  []string `json:"verweise"`
	IstFilter bool     `json:"ist_filter"`
}

// SchlagwortPflegeListe ist die Antwort der Pflegeseite: die Zeilen alphabetisch und die
// Gesamtzahl — liegt sie über der Kappung, sagt die Seite das. Gesamt zählt jede Zeile,
// Verweise die Zeilen, die Verweise sind; die Wörter sind die Differenz.
type SchlagwortPflegeListe struct {
	Zeilen   []SchlagwortPflegeZeile `json:"zeilen"`
	Gesamt   int                     `json:"gesamt"`
	Verweise int                     `json:"verweise"`
}

// SchlagworteZurPflege liefert alle Schlagworte mit Titelzahl, Verweisziel und den
// Verweisen darauf, alphabetisch.
func SchlagworteZurPflege(ctx context.Context, q DBQueryer) (SchlagwortPflegeListe, error) {
	liste := SchlagwortPflegeListe{Zeilen: []SchlagwortPflegeZeile{}}
	if err := q.QueryRow(ctx, `SELECT count(*)::int, count(verweis_auf)::int FROM schlagworte`).
		Scan(&liste.Gesamt, &liste.Verweise); err != nil {
		return liste, fmt.Errorf("schlagworte zählen: %w", err)
	}
	rows, err := q.Query(ctx, `
		SELECT s.id::text, s.wort,
		       (SELECT count(*)::int FROM titel_schlagworte ts WHERE ts.schlagwort_id = s.id),
		       coalesce(z.id::text, ''), coalesce(z.wort, ''),
		       coalesce((SELECT array_agg(v.wort ORDER BY lower(v.wort))
		                 FROM schlagworte v WHERE v.verweis_auf = s.id), '{}'),
		       s.ist_filter
		FROM schlagworte s
		LEFT JOIN schlagworte z ON z.id = s.verweis_auf
		ORDER BY lower(s.wort)
		LIMIT $1`, schlagwortPflegeListeMax)
	if err != nil {
		return liste, fmt.Errorf("schlagworte zur pflege lesen: %w", err)
	}
	zeilen, err := pgx.CollectRows(rows, pgx.RowToStructByPos[SchlagwortPflegeZeile])
	if err != nil {
		return liste, fmt.Errorf("schlagworte zur pflege lesen: %w", err)
	}
	liste.Zeilen = zeilen
	return liste, nil
}

// pflegeWort ist ein gesperrtes Schlagwort innerhalb einer Pflege-Transaktion.
type pflegeWort struct {
	id, wort   string
	verweisAuf *string
	istFilter  bool
}

// sperreSchlagwort liest und sperrt ein Schlagwort; unbekannt ist ErrSchlagwortNichtGefunden.
func sperreSchlagwort(ctx context.Context, tx pgx.Tx, id string) (pflegeWort, error) {
	var w pflegeWort
	err := tx.QueryRow(ctx, `
		SELECT id::text, wort, verweis_auf::text, ist_filter FROM schlagworte WHERE id = $1 FOR UPDATE`, id).
		Scan(&w.id, &w.wort, &w.verweisAuf, &w.istFilter)
	if errors.Is(err, pgx.ErrNoRows) {
		return w, ErrSchlagwortNichtGefunden
	}
	if err != nil {
		return w, fmt.Errorf("schlagwort sperren: %w", err)
	}
	return w, nil
}

// einWort bringt eine Eingabe in die gespeicherte Form (dieselbe wie am Titel).
func einWort(roh string) (string, error) {
	woerter, err := NormalisiereSchlagworte([]string{roh})
	if err != nil {
		return "", err
	}
	if len(woerter) == 0 {
		return "", fmt.Errorf("%w: das Wort ist leer", ErrSchlagwortUngueltig)
	}
	return woerter[0], nil
}

// regelFehler übersetzt eine Ausnahme der Trigger aus Migration 143 in ErrSchlagwortRegel.
// Die Tür prüft dieselben Regeln vorher; die Datenbank ist der Rückhalt, etwa wenn zwei
// Pflegende gleichzeitig arbeiten.
func regelFehler(err error, was string) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && strings.HasPrefix(pgErr.Message, "schlagwort_verweis_") {
		// Der Code vor dem Doppelpunkt ist für die Erkennung da, der Satz dahinter für Menschen.
		_, satz, _ := strings.Cut(pgErr.Message, ": ")
		return fmt.Errorf("%w: %s", ErrSchlagwortRegel, satz)
	}
	return fmt.Errorf("%s: %w", was, err)
}

// BenenneSchlagwortUm gibt einem Wort eine neue Schreibweise; alle Titel tragen danach die
// neue. Gibt es das neue Wort schon (ohne Rücksicht auf Groß- und Kleinschreibung) als
// ANDERES Schlagwort, ist das ein Zusammenführen und wird abgelehnt (ErrSchlagwortGibtEs) —
// die Pflegeseite fragt dann nach. Nur die Schreibweise zu ändern („fantasy" → „Fantasy")
// geht. Ist die neue Schreibweise ein Verweis auf eben dieses Wort, tauschen die beiden:
// „Krimi" mit dem Verweis „Kriminalroman" wird „Kriminalroman", der Verweis geht im Wort
// auf. Vorher lehnte die Tür das ab und riet zum Zusammenführen — das scheitert an einem
// eigenen Verweis („nicht mit sich selbst").
//
// alteAlsVerweis lässt die alte Schreibweise als Verweis auf das Wort stehen, wie beim
// Zusammenführen (entschieden am 23.09.2026, docs/OFFEN.md 4.20). Das hält eine Maske
// richtig, die beim Umbenennen offen war: Buchformular und Bestellkorb schicken beim
// Speichern die Menge zurück, die sie beim Öffnen gelesen haben, und die alte Schreibweise
// löst sich dann zum Wort auf. Ohne Verweis — so macht es Littera — legt ein solches
// Speichern die alte Schreibweise wieder an. Liefert die gespeicherte Schreibweise und ob ein
// Verweis entstanden ist.
//
// Ein Verweis selbst lässt sich nur ohne alteAlsVerweis umbenennen (ErrSchlagwortRegel, mit
// dem Weg im Satz): Seine alte Schreibweise müsste auf sein Ziel zeigen, und dafür bräuchte
// die Tür nach dem Verweis noch das Ziel — die umgekehrte Reihenfolge des Zusammenführens,
// das erst das Ziel sperrt und dann seine Verweise umhängt. Eine weitere Schreibweise legt
// „Verweis anlegen" am Ziel an.
func BenenneSchlagwortUm(ctx context.Context, q DBQueryer, id, neu string, alteAlsVerweis bool) (string, bool, error) {
	wort, err := einWort(neu)
	if err != nil {
		return "", false, err
	}
	tx, err := q.Begin(ctx)
	if err != nil {
		return "", false, fmt.Errorf("umbenennen: transaktion öffnen: %w", err)
	}
	defer db.SafeRollback(ctx, tx)
	alt, err := sperreSchlagwort(ctx, tx, id)
	if err != nil {
		return "", false, err
	}
	if alteAlsVerweis && alt.verweisAuf != nil {
		return "", false, fmt.Errorf("%w: „%s“ ist ein Verweis — eine weitere Schreibweise legt „Verweis anlegen“ am Ziel an",
			ErrSchlagwortRegel, alt.wort)
	}
	var anderesID, anderes string
	var anderesZiel *string
	err = tx.QueryRow(ctx, `SELECT id::text, wort, verweis_auf::text FROM schlagworte
		WHERE lower(wort) = lower($1) AND id <> $2`, wort, id).Scan(&anderesID, &anderes, &anderesZiel)
	switch {
	case errors.Is(err, pgx.ErrNoRows):
	case err != nil:
		return "", false, fmt.Errorf("umbenennen: vorhandenes wort suchen: %w", err)
	case anderesZiel != nil && *anderesZiel == id:
		// Der eigene Verweis geht im Wort auf. Ohne Sperre gelesen, damit die Tür nur Zeilen
		// sperrt, die sie ändert. Hat ihn inzwischen jemand umgehängt oder gelöscht, trifft
		// das DELETE keine Zeile; die Tür meldet dann „gibt es schon", und ein zweiter Versuch
		// sieht den neuen Stand.
		tag, err := tx.Exec(ctx, `DELETE FROM schlagworte WHERE id = $1 AND verweis_auf = $2`, anderesID, id)
		if err != nil {
			return "", false, fmt.Errorf("umbenennen: eigenen verweis auflösen: %w", err)
		}
		if tag.RowsAffected() != 1 {
			return "", false, fmt.Errorf("%w: „%s“", ErrSchlagwortGibtEs, anderes)
		}
	default:
		return "", false, fmt.Errorf("%w: „%s“", ErrSchlagwortGibtEs, anderes)
	}
	tag, err := tx.Exec(ctx, `UPDATE schlagworte SET wort = $2 WHERE id = $1`, id, wort)
	if err != nil {
		return "", false, fmt.Errorf("umbenennen: %w", err)
	}
	if tag.RowsAffected() != 1 {
		return "", false, ErrSchlagwortNichtGefunden
	}
	verweis := false
	if alteAlsVerweis {
		// Der Konflikt auf lower(wort) ist genau der Fall „nur die Groß- und Kleinschreibung
		// geändert": Dann ist die alte Schreibweise dasselbe Wort, ein Verweis wäre keiner.
		tag, err := tx.Exec(ctx, `
			INSERT INTO schlagworte (wort, verweis_auf) VALUES ($1, $2)
			ON CONFLICT (lower(wort)) DO NOTHING`, alt.wort, id)
		if err != nil {
			return "", false, regelFehler(err, "umbenennen: alte schreibweise als verweis")
		}
		verweis = tag.RowsAffected() == 1
	}
	if err := tx.Commit(ctx); err != nil {
		return "", false, fmt.Errorf("umbenennen: commit: %w", err)
	}
	return wort, verweis, nil
}

// FuehreSchlagworteZusammen hängt die Titel von „von" an „in". Mit alteAlsVerweis wird „von"
// zum Verweis auf „in" — wer das alte Wort gewohnt ist, landet weiter richtig —, sonst fällt
// es weg, wie in Littera. Verweise auf „von" zeigen danach auf „in", eine Filter-Markierung
// geht auf „in" über. Ist „in" selbst ein Verweis, gilt sein Ziel. Liefert die Zahl der
// Titel, die „von" trug.
func FuehreSchlagworteZusammen(ctx context.Context, q DBQueryer, vonID, inID string, alteAlsVerweis bool) (int, error) {
	tx, err := q.Begin(ctx)
	if err != nil {
		return 0, fmt.Errorf("zusammenführen: transaktion öffnen: %w", err)
	}
	defer db.SafeRollback(ctx, tx)
	titel, err := fuehreZusammenIn(ctx, tx, vonID, inID, alteAlsVerweis)
	if err != nil {
		return 0, err
	}
	if err := tx.Commit(ctx); err != nil {
		return 0, fmt.Errorf("zusammenführen: commit: %w", err)
	}
	return titel, nil
}

// fuehreZusammenIn ist das Zusammenführen innerhalb einer laufenden Transaktion.
func fuehreZusammenIn(ctx context.Context, tx pgx.Tx, vonID, inID string, alteAlsVerweis bool) (int, error) {
	// In fester Reihenfolge sperren, damit zwei gegenläufige Aufrufe nicht verklemmen.
	erste, zweite := vonID, inID
	if zweite < erste {
		erste, zweite = zweite, erste
	}
	gesperrt := map[string]pflegeWort{}
	for _, id := range []string{erste, zweite} {
		w, err := sperreSchlagwort(ctx, tx, id)
		if err != nil {
			return 0, err
		}
		gesperrt[id] = w
	}
	von, ziel := gesperrt[vonID], gesperrt[inID]
	if ziel.verweisAuf != nil {
		z, err := sperreSchlagwort(ctx, tx, *ziel.verweisAuf)
		if err != nil {
			return 0, err
		}
		ziel = z
	}
	if von.id == ziel.id {
		return 0, fmt.Errorf("%w: ein Wort lässt sich nicht mit sich selbst zusammenführen", ErrSchlagwortRegel)
	}

	var titel int
	if err := tx.QueryRow(ctx,
		`SELECT count(*)::int FROM titel_schlagworte WHERE schlagwort_id = $1`, von.id).Scan(&titel); err != nil {
		return 0, fmt.Errorf("zusammenführen: titel zählen: %w", err)
	}
	// Die Reihenfolge ist die Regel der Trigger: erst Titel und Verweise von „von"
	// wegnehmen, dann wird „von" selbst zum Verweis (oder fällt weg).
	type zusammenSchritt struct {
		was, sql string
		args     []any
	}
	letzter := zusammenSchritt{"zum verweis machen", `UPDATE schlagworte SET verweis_auf = $2, ist_filter = false WHERE id = $1`,
		[]any{von.id, ziel.id}}
	if !alteAlsVerweis {
		letzter = zusammenSchritt{"altes wort löschen", `DELETE FROM schlagworte WHERE id = $1`, []any{von.id}}
	}
	schritte := []zusammenSchritt{
		{"titel umhängen", `INSERT INTO titel_schlagworte (titel_id, schlagwort_id)
			SELECT titel_id, $2 FROM titel_schlagworte WHERE schlagwort_id = $1 ON CONFLICT DO NOTHING`,
			[]any{von.id, ziel.id}},
		{"titel lösen", `DELETE FROM titel_schlagworte WHERE schlagwort_id = $1`, []any{von.id}},
		{"verweise umhängen", `UPDATE schlagworte SET verweis_auf = $2 WHERE verweis_auf = $1`,
			[]any{von.id, ziel.id}},
		{"filter übernehmen", `UPDATE schlagworte SET ist_filter = true WHERE id = $1`, nil},
		letzter,
	}
	for _, schritt := range schritte {
		if schritt.args == nil {
			// Die Markierung geht nur über, wenn „von" sie trug.
			if !von.istFilter {
				continue
			}
			schritt.args = []any{ziel.id}
		}
		if _, err := tx.Exec(ctx, schritt.sql, schritt.args...); err != nil {
			return 0, regelFehler(err, "zusammenführen: "+schritt.was)
		}
	}
	return titel, nil
}

// SchlagwortLoeschung sagt, was ein Löschen getroffen hat: die gewählten Wörter, die Titel,
// die dadurch Schlagworte verloren (jeder Titel einmal), und die Verweise, die mitfielen, ohne
// selbst gewählt zu sein.
type SchlagwortLoeschung struct {
	Woerter  int
	Titel    int
	Verweise int
}

// LoescheSchlagworte löscht die gewählten Wörter in einer Transaktion — alle oder keins; die
// Titel verlieren sie, die Verweise darauf fallen mit (ON DELETE CASCADE). Mehrere auf einmal
// wie in Littera („Datenbearbeitung", Markieren und Löschen), für das Aufräumen nach dem
// Einlesen der Littera-Schlagworte (docs/OFFEN.md 4.20). Ein einzelnes Wort ist eine Liste mit
// einem Eintrag: eine Tür, eine Regel. Gibt es eine Kennung nicht (mehr), löscht die Tür
// nichts (ErrSchlagwortNichtGefunden) — die Pflegeseite lädt dann den neuen Stand.
func LoescheSchlagworte(ctx context.Context, q DBQueryer, ids []string) (SchlagwortLoeschung, error) {
	var ergebnis SchlagwortLoeschung
	ids = slices.Compact(slices.Sorted(slices.Values(ids)))
	if len(ids) == 0 {
		return ergebnis, fmt.Errorf("%w: kein Schlagwort gewählt", ErrSchlagwortUngueltig)
	}
	if len(ids) > schlagwortPflegeListeMax {
		return ergebnis, fmt.Errorf("%w: höchstens %d Schlagworte auf einmal", ErrSchlagwortUngueltig, schlagwortPflegeListeMax)
	}
	tx, err := q.Begin(ctx)
	if err != nil {
		return ergebnis, fmt.Errorf("löschen: transaktion öffnen: %w", err)
	}
	defer db.SafeRollback(ctx, tx)
	// In fester Reihenfolge sperren (LockRows steht über dem Sort), wie fuehreZusammenIn —
	// zwei Löschende mit überlappender Auswahl verklemmen sich nicht.
	var gesperrt int
	if err := tx.QueryRow(ctx, `
		SELECT count(*)::int FROM (
			SELECT id FROM schlagworte WHERE id = ANY($1::uuid[]) ORDER BY id FOR UPDATE
		) g`, ids).Scan(&gesperrt); err != nil {
		return ergebnis, fmt.Errorf("löschen: sperren: %w", err)
	}
	if gesperrt != len(ids) {
		return ergebnis, ErrSchlagwortNichtGefunden
	}
	if err := tx.QueryRow(ctx, `
		SELECT (SELECT count(DISTINCT titel_id)::int FROM titel_schlagworte WHERE schlagwort_id = ANY($1::uuid[])),
		       (SELECT count(*)::int FROM schlagworte
		        WHERE verweis_auf = ANY($1::uuid[]) AND NOT id = ANY($1::uuid[]))`, ids).
		Scan(&ergebnis.Titel, &ergebnis.Verweise); err != nil {
		return ergebnis, fmt.Errorf("löschen: zählen: %w", err)
	}
	tag, err := tx.Exec(ctx, `DELETE FROM schlagworte WHERE id = ANY($1::uuid[])`, ids)
	if err != nil {
		return ergebnis, fmt.Errorf("löschen: %w", err)
	}
	// Die Zeilen sind gesperrt; ein gewählter Verweis, dessen Ziel ebenfalls gewählt ist, fällt
	// hier selbst, nicht erst über die Kaskade (die läuft am Ende der Anweisung).
	if tag.RowsAffected() != int64(len(ids)) {
		return ergebnis, fmt.Errorf("löschen: %d Zeilen statt %d", tag.RowsAffected(), len(ids))
	}
	if err := tx.Commit(ctx); err != nil {
		return ergebnis, fmt.Errorf("löschen: commit: %w", err)
	}
	ergebnis.Woerter = len(ids)
	return ergebnis, nil
}

// SetzeSchlagwortVerweis leitet eine Schreibweise auf ein Wort („Tierfantasy" → „Fantasy").
// Gibt es die Schreibweise noch nicht, entsteht sie als Verweis. Gibt es sie und trägt sie
// Titel oder Verweise, wird sie mit dem Ziel zusammengeführt — das ist, was ein Verweis
// bedeutet. Ist das Ziel selbst ein Verweis, gilt dessen Ziel.
func SetzeSchlagwortVerweis(ctx context.Context, q DBQueryer, roh, zielID string) error {
	wort, err := einWort(roh)
	if err != nil {
		return err
	}
	tx, err := q.Begin(ctx)
	if err != nil {
		return fmt.Errorf("verweis: transaktion öffnen: %w", err)
	}
	defer db.SafeRollback(ctx, tx)

	var vorhandenID string
	err = tx.QueryRow(ctx, `SELECT id::text FROM schlagworte WHERE lower(wort) = lower($1)`, wort).Scan(&vorhandenID)
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		ziel, err := sperreSchlagwort(ctx, tx, zielID)
		if err != nil {
			return err
		}
		if ziel.verweisAuf != nil {
			zielID = *ziel.verweisAuf
		}
		tag, err := tx.Exec(ctx,
			`INSERT INTO schlagworte (wort, verweis_auf) VALUES ($1, $2)`, wort, zielID)
		if err != nil {
			return regelFehler(err, "verweis anlegen")
		}
		if tag.RowsAffected() != 1 {
			return fmt.Errorf("verweis anlegen: %d Zeilen statt 1", tag.RowsAffected())
		}
	case err != nil:
		return fmt.Errorf("verweis: vorhandenes wort suchen: %w", err)
	default:
		if _, err := fuehreZusammenIn(ctx, tx, vorhandenID, zielID, true); err != nil {
			return err
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("verweis: commit: %w", err)
	}
	return nil
}

// SetzeSchlagwortFilter markiert ein Wort als Filter im Portal (oder nimmt die Markierung
// weg). Ein Verweis ist nie Filter — markiert wird sein Ziel.
func SetzeSchlagwortFilter(ctx context.Context, q DBQueryer, id string, an bool) error {
	tx, err := q.Begin(ctx)
	if err != nil {
		return fmt.Errorf("filter: transaktion öffnen: %w", err)
	}
	defer db.SafeRollback(ctx, tx)
	w, err := sperreSchlagwort(ctx, tx, id)
	if err != nil {
		return err
	}
	if an && w.verweisAuf != nil {
		return fmt.Errorf("%w: „%s“ ist ein Verweis — als Filter markiert wird sein Ziel", ErrSchlagwortRegel, w.wort)
	}
	tag, err := tx.Exec(ctx, `UPDATE schlagworte SET ist_filter = $2 WHERE id = $1`, id, an)
	if err != nil {
		return regelFehler(err, "filter setzen")
	}
	if tag.RowsAffected() != 1 {
		return ErrSchlagwortNichtGefunden
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("filter: commit: %w", err)
	}
	return nil
}
