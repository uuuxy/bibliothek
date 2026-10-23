package inventur

import (
	"context"
	"encoding/json"
	"fmt"

	"bibliothek/repository"

	"github.com/jackc/pgx/v5"
)

// legeKlassensatzZuordnungenAn schreibt das Kreuzprodukt aus Klassen × Büchern per
// unnest-INSERT in class_books. Leere Eingaben werden ignoriert (kein Insert).
func legeKlassensatzZuordnungenAn(ctx context.Context, tx pgx.Tx, neueKlassen, titelIDs []string) error {
	if len(titelIDs) == 0 || len(neueKlassen) == 0 {
		return nil
	}

	var klassenZeilen []string
	var titelZeilen []string
	for _, klasse := range neueKlassen {
		for _, titelID := range titelIDs {
			klassenZeilen = append(klassenZeilen, klasse)
			titelZeilen = append(titelZeilen, titelID)
		}
	}

	if _, err := tx.Exec(ctx, `
		INSERT INTO class_books (class_name, book_id)
		SELECT class_name, book_id::uuid FROM unnest($1::text[], $2::text[]) AS t(class_name, book_id)`, klassenZeilen, titelZeilen); err != nil {
		return fmt.Errorf("neue zuweisung konnte nicht gespeichert werden: %w", err)
	}
	return nil
}

// klassensatzMindestLeser ist die Mindestzahl: Ab so vielen Kindern derselben Klasse mit demselben Titel — und
// mehr als der Hälfte der Klasse — gilt ein Titel als Klassensatz „aus Ausleihen". Die
// Hälfte allein reichte nicht: In einer Restklasse mit vier Kindern wären zwei Leser schon
// ein Satz.
const klassensatzMindestLeser = 5

// zweigOberstufe wählt in der Klassenübersicht Jahrgang 11 bis 13. Die Oberstufe trägt keinen
// Zweigbuchstaben wie G oder R im Namen, also wählt der Server sie über den Jahrgang.
const zweigOberstufe = "oberstufe"

// ladeKlassensatzGruppen liefert je Klasse ihre Titel der Klassensätze aus zwei Quellen (über
// die LUSD weiß das Programm, in welcher Klasse jeder Schüler ist, also kann sich die Liste
// selbst aktualisieren): der von Hand gepflegten Zuordnung (class_books, Quelle „hand"; bleibt
// unangetastet, nichts wird geschrieben oder gelöscht) und den laufenden Ausleihen (Quelle
// „ausleihe", jedes Mal gerechnet, nie gespeichert): Halten mehr als die Hälfte der aktiven
// Kinder einer Klasse, mindestens klassensatzMindestLeser, denselben Titel, ist das ihr
// Klassensatz, Schulbuch oder Lektüre. Gezählt werden die Klassen mit einem Jahrgang
// (repository.JahrgangSQL), auch die Einführungsphase ohne Zahl am Anfang. Ein Titel, der schon
// von Hand zugeordnet ist, erscheint nur einmal (hand). Beide Quellen laufen durch dieselbe
// Sortierung; zweig filtert auf den Bildungsgang (zweigOberstufe auf Jahrgang 11 bis 13),
// reihenfolge bestimmt die Reihenfolge der Klassen.
//
// Auflagen: Gezählt wird am Buch, nicht an der Auflage, mit COALESCE(werk_id, id) wie im
// Bestellbedarf. Je Auflage gezählt, verschwände ein Klassensatz, sobald er auf zwei Auflagen
// verteilt ist (14 Kinder mit der 4., 14 mit der 3. Auflage: keine hat mehr als die Hälfte), und
// bei 20 zu 8 bliebe die Mischung unsichtbar. Für ein Buch aus den Ausleihen steht die Auflage,
// die die meisten Kinder der Klasse haben, bei Gleichstand die neueste: Die Kachel zeigt, was die
// Klasse hat, nicht, was zuletzt erschien. Eine Zuordnung von Hand deckt das ganze Buch ab.
// Haben Kinder der Klasse eine andere Auflage als die der Kachel, trägt die Zeile die
// Aufschlüsselung.
func (a *TitelAbfragen) ladeKlassensatzGruppen(ctx context.Context, zweig string, reihenfolge string) ([]klassensatzGruppe, error) {
	neueste := repository.SQLNeuesteAuflageZuerst("t")
	abfrage := `
		WITH klassen_groesse AS (
			SELECT klassen_normkey(klasse) AS k, min(klasse) AS klasse, count(*) AS n
			FROM schueler
			WHERE deleted_at IS NULL AND ist_abgaenger = false AND ` + repository.JahrgangSQL("klasse") + ` IS NOT NULL
			GROUP BY klassen_normkey(klasse)
		),
		offen AS (
			SELECT klassen_normkey(s.klasse) AS k, s.id AS kind, t.id AS titel_id,
			       COALESCE(t.werk_id, t.id) AS buch
			FROM ausleihen a
			JOIN schueler s ON s.id = a.schueler_id AND s.deleted_at IS NULL AND s.ist_abgaenger = false
			JOIN buecher_exemplare e ON e.id = a.exemplar_id
			JOIN buecher_titel t ON t.id = e.titel_id
			WHERE a.rueckgabe_am IS NULL
		),
		je_auflage AS (
			SELECT k, buch, titel_id, count(DISTINCT kind) AS kinder
			FROM offen GROUP BY k, buch, titel_id
		),
		leser AS (
			SELECT k, buch, count(DISTINCT kind) AS leser
			FROM offen GROUP BY k, buch
		),
		vertreter AS (
			SELECT DISTINCT ON (j.k, j.buch) j.k, j.buch, j.titel_id
			FROM je_auflage j JOIN buecher_titel t ON t.id = j.titel_id
			ORDER BY j.k, j.buch, j.kinder DESC, ` + neueste + `
		),
		mischung AS (
			-- Nur Titel, die zu einem Buch mit mehreren Auflagen gehören: Ohne Werk ist buch = titel_id.
			SELECT j.k, j.buch, array_agg(j.titel_id) AS titel_ids,
			       json_agg(json_build_object('id', t.id, 'auflage', coalesce(t.auflage, ''),
			           'erscheinungsjahr', coalesce(t.erscheinungsjahr, 0), 'kinder', j.kinder)
			           ORDER BY j.kinder DESC, ` + neueste + `) AS auflagen
			FROM je_auflage j JOIN buecher_titel t ON t.id = j.titel_id
			WHERE j.buch <> j.titel_id
			GROUP BY j.k, j.buch
		),
		zuordnung AS (
			SELECT class_name, book_id, 'hand'::text AS quelle, 0::bigint AS leser FROM class_books
			UNION ALL
			SELECT COALESCE((SELECT min(h.class_name) FROM class_books h WHERE klassen_normkey(h.class_name) = l.k), g.klasse),
			       v.titel_id, 'ausleihe', l.leser
			FROM leser l
			JOIN klassen_groesse g ON g.k = l.k
			JOIN vertreter v ON v.k = l.k AND v.buch = l.buch
			WHERE l.leser * 2 > g.n AND l.leser >= $2
			  AND NOT EXISTS (SELECT 1 FROM class_books h JOIN buecher_titel ht ON ht.id = h.book_id
			                  WHERE klassen_normkey(h.class_name) = l.k AND COALESCE(ht.werk_id, ht.id) = l.buch)
		)
		SELECT
			cb.class_name, b.id, b.titel AS title, COALESCE(b.subject, '') AS subject,
			COALESCE(b.track, '') AS track, COALESCE(b.cover_url, '') AS cover_url,
			COALESCE(b.isbn, '') AS isbn,
			COUNT(e.id) FILTER (WHERE e.ist_ausleihbar = true AND e.ist_ausgesondert = false AND a.id IS NULL) AS verfuegbar,
			COUNT(e.id) FILTER (WHERE e.ist_ausgesondert = false AND e.bestellstatus IS NULL) AS gesamt,
			` + repository.SQLFilterImZulauf + ` AS im_zulauf,
			cb.quelle, cb.leser,
			(SELECT m.auflagen FROM mischung m
			 WHERE m.k = klassen_normkey(cb.class_name) AND m.buch = COALESCE(b.werk_id, b.id)
			   AND NOT (m.titel_ids <@ ARRAY[b.id])) AS auflagen
		FROM zuordnung cb
		JOIN buecher_titel b ON cb.book_id = b.id
		LEFT JOIN buecher_exemplare e ON e.titel_id = b.id
		LEFT JOIN ausleihen a ON a.exemplar_id = e.id AND a.rueckgabe_am IS NULL
		WHERE CASE $1 WHEN '' THEN true
		              WHEN '` + zweigOberstufe + `' THEN ` + repository.JahrgangSQL("cb.class_name") + ` >= 11
		              ELSE cb.class_name ILIKE '%' || $1 || '%' END
		GROUP BY cb.class_name, b.id, b.titel, b.subject, b.track, b.cover_url, b.isbn, cb.quelle, cb.leser
		ORDER BY ` + klassenSortierung(zweig, reihenfolge)

	zeilen, err := a.pool.Query(ctx, abfrage, zweig, klassensatzMindestLeser)
	if err != nil {
		return nil, fmt.Errorf("klassen-bücher konnten nicht geladen werden: %w", err)
	}
	defer zeilen.Close()

	gruppenNachName := make(map[string][]klassensatzTitel)
	var klassen []string

	for zeilen.Next() {
		klasse, eintrag, err := leseKlassensatzTitel(zeilen)
		if err != nil {
			return nil, err
		}

		if _, vorhanden := gruppenNachName[klasse]; !vorhanden {
			klassen = append(klassen, klasse)
		}
		gruppenNachName[klasse] = append(gruppenNachName[klasse], eintrag)
	}
	if err := zeilen.Err(); err != nil {
		return nil, fmt.Errorf("klassen-bücher konnten nicht vollständig gelesen werden: %w", err)
	}

	var ergebnis []klassensatzGruppe
	for _, name := range klassen {
		ergebnis = append(ergebnis, klassensatzGruppe{
			Klasse: name,
			Titel:  gruppenNachName[name],
		})
	}

	return ergebnis, nil
}

// klassenSortierung ist das ORDER BY der Klassenübersicht: ohne Bildungsgang zuerst in der
// Reihenfolge F, G, R, H, dann nach Jahrgang, Klasse und Titel. Die Einführungsphase steht als
// Jahrgang 11 vor 12T und 13T.
func klassenSortierung(zweig, reihenfolge string) string {
	sortierung := ""
	if zweig == "" {
		// Fallback: Workflow-Reihenfolge F, G, R, H
		sortierung += `
			CASE 
				WHEN cb.class_name ILIKE '%F%' THEN 1
				WHEN cb.class_name ILIKE '%G%' THEN 2
				WHEN cb.class_name ILIKE '%R%' THEN 3
				WHEN cb.class_name ILIKE '%H%' THEN 4
				ELSE 5
			END, `
	}

	stufeZahl := repository.JahrgangSQL("cb.class_name")
	if reihenfolge == "desc" {
		return sortierung + stufeZahl + ` DESC, cb.class_name DESC, b.titel ASC`
	}
	return sortierung + stufeZahl + ` ASC, cb.class_name ASC, b.titel ASC`
}

// leseKlassensatzTitel liest eine Zeile der Klassenübersicht: die Klasse und ihr Buch, mit der
// Aufschlüsselung nach Auflagen, wenn die Zeile eine trägt.
func leseKlassensatzTitel(zeilen pgx.Rows) (string, klassensatzTitel, error) {
	var klasse string
	var eintrag klassensatzTitel
	var auflagen []byte
	err := zeilen.Scan(&klasse, &eintrag.ID, &eintrag.Titel, &eintrag.Fach, &eintrag.Zweig, &eintrag.CoverURL, &eintrag.ISBN, &eintrag.Verfuegbar, &eintrag.Gesamt, &eintrag.ImZulauf, &eintrag.Quelle, &eintrag.Leser, &auflagen)
	if err != nil {
		return "", eintrag, fmt.Errorf("daten konnten nicht gelesen werden: %w", err)
	}
	if len(auflagen) > 0 {
		if err := json.Unmarshal(auflagen, &eintrag.Auflagen); err != nil {
			return "", eintrag, fmt.Errorf("auflagen von %s in %s: %w", eintrag.ID, klasse, err)
		}
	}
	return klasse, eintrag, nil
}

// aendereKlassensaetze schreibt die Buchzuordnung einer Klasse neu. alteKlasse wird dabei
// durch neueKlassen ersetzt — so lässt sich derselbe Satz in einem Zug auf mehrere
// Klassen verteilen, ohne ihn mehrfach anzulegen.
func (a *TitelAbfragen) aendereKlassensaetze(ctx context.Context, alteKlasse string, neueKlassen []string, titelIDs []string) error {
	tx, err := a.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("transaktion konnte nicht gestartet werden: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }() //nolint:errcheck

	// Gibt es einen alten Namen der Klasse, wird er gelöscht.
	if alteKlasse != "" {
		if _, err := tx.Exec(ctx, `DELETE FROM class_books WHERE class_name = $1`, alteKlasse); err != nil {
			return fmt.Errorf("alte zuweisungen konnten nicht gelöscht werden: %w", err)
		}
	}

	// Die bisherigen Zuordnungen aller Zielklassen in einem DELETE entfernen (überschreiben).
	if len(neueKlassen) > 0 {
		if _, err := tx.Exec(ctx, `DELETE FROM class_books WHERE class_name = ANY($1)`, neueKlassen); err != nil {
			return fmt.Errorf("vorhandene zuweisungen des neuen namens konnten nicht gelöscht werden: %w", err)
		}
	}

	// Die neuen Zuordnungen in einem INSERT mit unnest anlegen.
	if err := legeKlassensatzZuordnungenAn(ctx, tx, neueKlassen, titelIDs); err != nil {
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("transaktion konnte nicht abgeschlossen werden: %w", err)
	}

	return nil
}

// loescheKlassensatzGruppe löst die Klassensätze der genannten Klassen auf. Betroffen ist nur
// die Zuordnung — die Titel selbst bleiben im Katalog.
func (a *TitelAbfragen) loescheKlassensatzGruppe(ctx context.Context, klassen []string) error {
	if len(klassen) == 0 {
		return nil
	}
	_, err := a.pool.Exec(ctx, `DELETE FROM class_books WHERE class_name = ANY($1)`, klassen)
	if err != nil {
		return fmt.Errorf("klassen konnten nicht gelöscht werden: %w", err)
	}
	return nil
}

// ordneTitelKlassenZu ergänzt Titel in mehreren Klassensätzen, ohne die bestehende
// Zuordnung zu ersetzen — anders als aendereKlassensaetze, das sie neu schreibt.
func (a *TitelAbfragen) ordneTitelKlassenZu(ctx context.Context, klassen []string, titelIDs []string) error {
	if len(klassen) == 0 || len(titelIDs) == 0 {
		return nil
	}

	var klassenZeilen []string
	var titelZeilen []string

	for _, klasse := range klassen {
		for _, titelID := range titelIDs {
			klassenZeilen = append(klassenZeilen, klasse)
			titelZeilen = append(titelZeilen, titelID)
		}
	}

	// ON CONFLICT DO NOTHING: Vorhandene Zuordnungen bleiben und entstehen nicht doppelt.
	abfrage := `
		INSERT INTO class_books (class_name, book_id)
		SELECT class_name, book_id::uuid FROM unnest($1::text[], $2::text[]) AS t(class_name, book_id)
		ON CONFLICT (class_name, book_id) DO NOTHING
	`

	_, err := a.pool.Exec(ctx, abfrage, klassenZeilen, titelZeilen)
	if err != nil {
		return fmt.Errorf("fehler beim hinzufügen der bücher zu den klassen: %w", err)
	}

	return nil
}
