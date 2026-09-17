package repository

import (
	"context"
	"time"

	"bibliothek/pkg/schulzeit"
)

// Das Abgangsbuch: welche Exemplare in einem Zeitraum aus dem Bestand gegangen sind.
//
// Protokoll des Medienzentrums vom 16.09.2026, Punkt 1 („Zugangs- und Abgangsbuch
// fehlen"). Das Datum dazu steht seit Migration 128 am Exemplar; hier ist der Nachweis,
// den die Schule zum Stichtag ausdruckt und abheftet.
//
// Getrennt nach Topf, weil die Finanzen getrennt geführt werden: Lernmittel gehören dem
// Land, der Bücherei-Bestand dem Schulträger (docs/mittel_konzept.md 1.1/1.2). Ein
// Nachweis, der beides mischt, muss von Hand sortiert werden, bevor er irgendwo hingeht.

// AbgangsZeile ist ein Abgang, wie er im Buch steht.
type AbgangsZeile struct {
	Datum    time.Time `json:"datum"`
	Barcode  string    `json:"barcode"`
	Titel    string    `json:"titel"`
	Signatur string    `json:"signatur"`
	Grund    string    `json:"grund"`
	// GrundText ist derselbe Grund in Klartext. Er kommt vom Server, damit Ausdruck und
	// Bildschirm dieselben vier Wörter benutzen — zwei Übersetzungen desselben
	// Schlüssels laufen auseinander, sobald einer davon geändert wird.
	GrundText string `json:"grund_text"`
	// Topf: 'land' für ein Lernmittel, sonst 'schultraeger' — dasselbe Vokabular wie bei
	// Bestellung und Bescheid (Migrationen 109/110). Beim Abgang entscheidet ihn der
	// Titel: Ein Buch des Landes bleibt eines, egal wie es einmal beschafft wurde.
	Topf string `json:"topf"`
}

// AbgangsgrundText übersetzt den gespeicherten Grund (chk_aussonderung_grund) in die
// Sprache des Nachweises.
func AbgangsgrundText(grund string) string {
	switch grund {
	case "VERLUST":
		return "Verlust"
	case "BESCHAEDIGUNG":
		return "Beschädigung"
	case "AUSSORTIERT":
		return "Aussortiert"
	case "BESTANDSKORREKTUR":
		return "Bestandskorrektur"
	case "":
		// Kommt nur bei Altdaten vor: chk_aussonderung_grund verlangt seit Migration 043
		// einen Grund. Ein leeres Feld im Nachweis sähe nach Druckfehler aus.
		return "ohne Angabe"
	default:
		return grund
	}
}

// Abgangsbuch bündelt Zeitraum, Zeilen und die Zahl der Abgänge OHNE bekannten Zeitpunkt.
//
// OhneZeitpunkt ist kein Schönheitsfehler, sondern die wichtigste Zahl auf dem Blatt: Was
// vor Migration 128 ausgesondert wurde, hat kein Abgangsdatum und kann keinem Zeitraum
// zugeordnet werden. Diese Exemplare stehen in KEINER Halbjahresliste. Wer die Zahl nicht
// hinschreibt, behauptet Vollständigkeit, die es nicht gibt.
type Abgangsbuch struct {
	Von           time.Time      `json:"von"`
	Bis           time.Time      `json:"bis"`
	Zeilen        []AbgangsZeile `json:"zeilen"`
	OhneZeitpunkt int            `json:"ohne_zeitpunkt"`
}

// LadeAbgangsbuch liest die Abgänge eines Zeitraums. `von` und `bis` sind Kalendertage der
// Schule, beide EINSCHLIESSLICH — der 15.9. gehört noch in das Halbjahr, das an ihm endet.
//
// Verglichen wird auf Zeitpunkten statt auf `(ausgesondert_am AT TIME ZONE …)::date`: Der
// Ausdruck um die Spalte herum macht den Teilindex aus Migration 128 unbrauchbar. Die
// Grenzen entstehen deshalb hier, in der Zeitzone der Schule — die Datenbanksitzung läuft
// in UTC, und zwischen Mitternacht in Berlin und Mitternacht UTC läge ein Abgang sonst im
// falschen Halbjahr.
func LadeAbgangsbuch(ctx context.Context, q DBQueryer, von, bis time.Time) (Abgangsbuch, error) {
	loc := schulzeit.Zone()
	abVon := time.Date(von.Year(), von.Month(), von.Day(), 0, 0, 0, 0, loc)
	bisAusschliesslich := time.Date(bis.Year(), bis.Month(), bis.Day(), 0, 0, 0, 0, loc).AddDate(0, 0, 1)

	buch := Abgangsbuch{Von: abVon, Bis: bisAusschliesslich.AddDate(0, 0, -1), Zeilen: []AbgangsZeile{}}

	rows, err := q.Query(ctx, `
		SELECT e.ausgesondert_am, e.barcode_id, t.titel,
		       COALESCE(t.signatur, ''), COALESCE(e.aussonderung_grund, ''),
		       CASE WHEN t.ist_lernmittel THEN 'land' ELSE 'schultraeger' END
		FROM buecher_exemplare e
		JOIN buecher_titel t ON t.id = e.titel_id
		WHERE e.ist_ausgesondert = true
		  AND e.ausgesondert_am >= $1 AND e.ausgesondert_am < $2
		ORDER BY t.ist_lernmittel DESC, e.ausgesondert_am, t.titel, e.barcode_id
	`, abVon, bisAusschliesslich)
	if err != nil {
		return buch, err
	}
	defer rows.Close()

	for rows.Next() {
		var z AbgangsZeile
		if err := rows.Scan(&z.Datum, &z.Barcode, &z.Titel, &z.Signatur, &z.Grund, &z.Topf); err != nil {
			return buch, err
		}
		z.Datum = z.Datum.In(loc)
		z.GrundText = AbgangsgrundText(z.Grund)
		buch.Zeilen = append(buch.Zeilen, z)
	}
	if err := rows.Err(); err != nil {
		return buch, err
	}

	if err := q.QueryRow(ctx, `
		SELECT count(*) FROM buecher_exemplare
		WHERE ist_ausgesondert = true AND ausgesondert_am IS NULL
	`).Scan(&buch.OhneZeitpunkt); err != nil {
		return buch, err
	}
	return buch, nil
}
