package repository

import (
	"context"
	"errors"
	"strings"

	"github.com/jackc/pgx/v5"
)

// FindeZuBarcode liest einen Leser — Schüler oder Kollegium — über seine
// Ausweisnummer. Die Abfrage geht an die Tabelle `leser` und nicht an die Sicht
// `schueler`; nur so finden Theke und Suchleiste einen Kollegen.
func (a *pgLeserAbfragen) FindeZuBarcode(ctx context.Context, barcode string) (*Leser, error) {
	return a.findeMit(ctx, `WHERE barcode_id = $1 AND deleted_at IS NULL`, barcode)
}

// Finde liest einen Leser über seine UUID.
func (a *pgLeserAbfragen) Finde(ctx context.Context, id string) (*Leser, error) {
	return a.findeMit(ctx, `WHERE id = $1 AND deleted_at IS NULL`, id)
}

// findeMit führt die beiden Leser-Abfragen aus: dieselbe Spaltenliste, dieselbe Behandlung
// des Nichttreffers (nil, nil), nur eine andere Bedingung.
func (a *pgLeserAbfragen) findeMit(ctx context.Context, bedingung string, arg any) (*Leser, error) {
	row := a.db.QueryRow(ctx, `SELECT `+spaltenLeser+`, art FROM leser `+bedingung+` LIMIT 1`, arg)
	var art string
	s, err := leseLeserMitZusatz(row, &art)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	s.Art = art
	return s, nil
}

// suchTokens zerlegt eine Eingabe in einzelne Suchbegriffe und entschärft die
// LIKE-Metazeichen. Ohne das Escaping wäre ein getipptes "%" eine Wildcard, die
// die halbe Schülerschaft zurückgibt, und "_" ein Platzhalter für ein Zeichen.
func suchTokens(suchtext string) []string {
	felder := strings.Fields(suchtext)
	tokens := make([]string, 0, len(felder))
	for _, f := range felder {
		entschaerft := strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`).Replace(f)
		tokens = append(tokens, entschaerft)
	}
	return tokens
}

// Die drei folgenden SQL-Bausteine sind die eine Lesersuche der Anwendung. Sie stehen hier als
// Konstanten, weil zwei Ansichten dieselbe Suche brauchen, aber verschiedene Spalten
// zurückgeben: die Omnibox an der Theke (FindeUnscharf) und die Leserdatei mit ihren
// Ausleihzahlen (LadeSchuelerListe, LadeLeserListe).
//
// Belegung der Platzhalter in beiden Abfragen gleich: $1 = Tokens (text[]), $2 = erstes Token.
// Der Tabellen-Alias muss s sein.

// leserSuchCTE zerlegt die Tokens in Normalform (diakritikfrei) und Rohform.
const leserSuchCTE = `
	WITH tokens AS (
		SELECT suchnorm(t) AS norm, lower(t) AS roh FROM unnest($1::text[]) AS t
	)`

// leserSuchBedingung ist das eigentliche Prädikat: Jedes Token muss irgendeine Spalte
// treffen (UND-Verknüpfung), verglichen über die Normalform.
//
// mitKlasse schaltet die Klasse als zusätzliches Suchfeld frei. Die Schülerdatei braucht
// das (ihr Filter konnte es immer schon, "10a" listet die Klasse), die Omnibox an der
// Theke nicht — dort sucht man einen Menschen, keine Klasse, und 30 Namen im Vorschlag
// wären dort im Weg. Weil jedes Token einzeln geprüft wird, bedeutet "10a Müller"
// übrigens genau das Richtige: Klasse 10a und Name Müller.
//
// Die vorangestellte Ankerbedingung auf $2 ist Absicht und kein Duplikat: Als direkte
// LIKE-Bedingung kann der Planer dafür die Trigramm-Indizes ziehen
// (BitmapOr über alle drei). Sie muss buchstäblich dem Index entsprechen — lower(barcode_id),
// nicht lower(coalesce(…)) auf den leeren Text. Mit dem coalesce ist dieser eine OR-Zweig nicht
// indexierbar, und ein einziger nicht-indexierbarer Zweig kippt die gesamte Abfrage in den
// Seq Scan (gemessen: 30 ms statt 0,1 ms bei 20.000 Schülern). Ein NULL-Barcode ergibt hier
// NULL statt false, was in der OR-Verknüpfung dasselbe Ergebnis liefert. Die Klasse steht
// bewusst nicht im Anker: Sie hat keinen Trigramm-Index, sie erweitert nur die Nachprüfung.
func leserSuchBedingung(mitKlasse bool) string {
	klasse := ""
	if mitKlasse {
		klasse = "\n\t\t\tOR suchnorm(coalesce(s.klasse, ''))     LIKE '%' || tokens.norm || '%'"
	}
	return `(
		   suchnorm(s.vorname)   LIKE '%' || suchnorm($2::text) || '%'
		OR suchnorm(s.nachname)  LIKE '%' || suchnorm($2::text) || '%'
		OR lower(s.barcode_id)   LIKE '%' || lower($2::text) || '%'` + klasseImAnker(mitKlasse) + `
	  )
	  AND (
		SELECT bool_and(
			   suchnorm(coalesce(s.vorname, ''))    LIKE '%' || tokens.norm || '%'
			OR suchnorm(coalesce(s.nachname, ''))   LIKE '%' || tokens.norm || '%'
			OR lower(coalesce(s.barcode_id, ''))    LIKE '%' || tokens.roh || '%'` + klasse + `)
		FROM tokens
	  )`
}

// klasseImAnker erweitert die Ankerbedingung um die Klasse. Ohne das könnte die Suche
// nach "10a" gar nichts finden: Der Anker filtert vor der Nachprüfung, und ein
// Klassenname steht in keinem der drei Namensfelder.
func klasseImAnker(mitKlasse bool) string {
	if !mitKlasse {
		return ""
	}
	return "\n\t\tOR suchnorm(s.klasse)    LIKE '%' || suchnorm($2::text) || '%'"
}

// leserSuchRang sortiert nach Trefferqualität: ein Token am Wortanfang des Nachnamens
// wiegt schwerer als eines im Vornamen, beides zusammen schlägt alles. Sonst stünde bei
// "max hoffmann" der gesuchte Schüler irgendwo alphabetisch in der Liste.
const leserSuchRang = `
		  (SELECT count(*) FROM tokens WHERE suchnorm(coalesce(s.nachname, '')) LIKE tokens.norm || '%') * 2
		+ (SELECT count(*) FROM tokens WHERE suchnorm(coalesce(s.vorname, ''))  LIKE tokens.norm || '%') DESC`

// FindeUnscharf durchsucht die Leserschaft nach Namen oder Ausweisnummern und liefert
// zusätzlich die Gesamtzahl der Treffer, nicht nur die des Limits.
//
// Gelesen wird die Tabelle `leser`, nicht die Sicht `schueler`: Über die Sicht fände niemand
// einen Kollegen an seinem Namen, und wer die Karte gerade nicht zur Hand hat, fände ihn an der
// Theke gar nicht. Die Art wandert als eigene Spalte mit; ohne sie stünde ein Kollege ohne
// Klasse in der Trefferliste wie ein Schüler mit fehlender Angabe. Wer nur Schüler meint
// (Klassenlisten, Mahnlauf, LUSD), fragt die Sicht.
//
// Zwei Eigenschaften der Suche:
//
//  1. Jedes Token muss irgendeine Spalte treffen (UND-Verknüpfung), nicht die
//     komplette Eingabe eine einzelne. Damit ist die Reihenfolge von Vor- und
//     Nachname bedeutungslos — "Lena Hoffmann" und "Hoffmann Lena" finden dieselbe
//     Person, und mehrteilige Namen ("Anna Maria", "García Rodríguez") ebenso.
//  2. Verglichen wird über die Normalform suchnorm() statt über den Rohtext. Sie
//     faltet Diakritika weg ("Garcia" findet García, "Ozturk" findet Öztürk) und
//     zieht die deutsche Ersatzschreibung auf denselben Nenner ("Mueller" findet
//     Müller, "Oeztuerk" findet Öztürk, "Strasse" findet Straße) — in beide
//     Richtungen, denn mal steht der Umlaut in der Datenbank und wird ohne getippt,
//     mal ist es umgekehrt.
//
// Die SQL-Bausteine stehen oben als Konstanten; dieselbe Suche bedient die Leserdatei.
func (a *pgLeserAbfragen) FindeUnscharf(ctx context.Context, suchtext string, limit int) ([]Leser, int, error) {
	tokens := suchTokens(suchtext)
	if len(tokens) == 0 {
		return nil, 0, nil
	}

	query := leserSuchCTE + `
		SELECT ` + spaltenLeser + `, art,
		       count(*) OVER () AS gesamt
		FROM leser s
		WHERE s.deleted_at IS NULL
		  AND ` + leserSuchBedingung(false) + `
		ORDER BY ` + leserSuchRang + `,
			s.nachname ASC, s.vorname ASC
		LIMIT $3
	`
	rows, err := a.db.Query(ctx, query, tokens, tokens[0], limit)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var treffer []Leser
	gesamt := 0
	for rows.Next() {
		var art string
		var zeilenGesamt int
		s, err := leseLeserMitZusatz(rows, &art, &zeilenGesamt)
		if err != nil {
			return nil, 0, err
		}
		s.Art = art
		gesamt = zeilenGesamt
		treffer = append(treffer, *s)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	return treffer, gesamt, nil
}
