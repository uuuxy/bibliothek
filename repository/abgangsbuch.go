package repository

import (
	"context"
	"time"

	"bibliothek/pkg/schulzeit"
)

// Das Abgangsbuch: welche Exemplare in einem Zeitraum aus dem Bestand gegangen sind. Die Schule
// druckt den Nachweis zum Stichtag aus und heftet ihn ab.
//
// Getrennt nach Topf, weil die Finanzen getrennt geführt werden: Lernmittel gehören dem Land,
// der Bücherei-Bestand dem Schulträger (docs/mittel_konzept.md 1.1/1.2). Ein Nachweis, der
// beides mischt, muss von Hand sortiert werden, bevor er irgendwo hingeht.

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
	// Topf ist 'land' oder 'schultraeger', dasselbe Vokabular wie bei Bestellung und Bescheid
	// und dieselbe Regel wie am Etikett (ExemplarTopfSQL): Eigentum am Exemplar, sonst Topf der
	// Bestellung, sonst der Titel. Entschiede allein der Titel, ginge ein Buch, das der
	// Schulträger bezahlt hat, als Abgang des Landes ab, obwohl sein Etikett den Schulträger
	// nennt.
	Topf string `json:"topf"`
}

// abgangsgrundText übersetzt den gespeicherten Grund (chk_aussonderung_grund) in die
// Sprache des Nachweises.
func abgangsgrundText(grund string) string {
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
		// Kommt nur bei Altdaten vor: chk_aussonderung_grund verlangt einen Grund. Ein leeres
		// Feld im Nachweis sähe nach Druckfehler aus.
		return "ohne Angabe"
	default:
		return grund
	}
}

// Abgangsbuch bündelt Zeitraum, Zeilen und die beiden Zahlen, die sagen, was nicht in der Liste
// steht.
//
// OhneZeitpunkt ist die wichtigste Zahl auf dem Blatt: Ein Exemplar aus dem Altbestand, das
// ohne Abgangsdatum ausgesondert ist, lässt sich keinem Zeitraum zuordnen und steht in keiner
// Halbjahresliste. Wer die Zahl nicht hinschreibt, behauptet Vollständigkeit, die es nicht
// gibt.
type Abgangsbuch struct {
	Von           time.Time      `json:"von"`
	Bis           time.Time      `json:"bis"`
	Zeilen        []AbgangsZeile `json:"zeilen"`
	OhneZeitpunkt int            `json:"ohne_zeitpunkt"`
	// AusKatalogGeloescht ist die zweite Lücke dieses Nachweises: Drei Türen entfernen ein
	// Exemplar körperlich, statt es auszusondern, nämlich Titel löschen (einzeln und als
	// Massenaktion) und „Verlust endgültig löschen" in der Inventur. Danach steht die Zeile in
	// keiner Abfrage über `buecher_exemplare` mehr, auch nicht in diesem Buch: Ein Exemplar,
	// das mit Abgangsdatum hier stand, verschwindet mit seinem Titel rückwirkend aus einem
	// Halbjahr, das vielleicht schon unterschrieben ist.
	//
	// Gezählt wird, was das Protokoll hergibt. Titel, Signatur und Abgangsgrund sind mit dem
	// Exemplar gegangen; sie aus dem Protokolltext zu rekonstruieren hieße, einen Nachweis aus
	// Bruchstücken zu bauen.
	AusKatalogGeloescht int `json:"aus_katalog_geloescht"`
}

// sqlWarImBestand sagt, ob ein Exemplar je im Bestand war: Es trägt ein Zugangsdatum.
// Die Löschspuren vermerken dasselbe (ProtokollDetailWarImBestand).
const sqlWarImBestand = `e.zugang_am IS NOT NULL`

// sqlIstAbgang ist die Grenze beider Abfragen und der Verlustzahlen der Statistik:
// ausgesondert und vorher im Bestand gewesen.
// Ein bestelltes Exemplar, das nie eintraf und ausgebucht wurde, trägt kein Zugangsdatum
// und steht damit in keinem der beiden Bücher.
const sqlIstAbgang = `e.ist_ausgesondert = true AND ` + sqlWarImBestand

// LadeAbgangsbuch liest die Abgänge eines Zeitraums. `von` und `bis` sind Kalendertage der
// Schule, beide einschließlich — der 15.9. gehört noch in das Halbjahr, das an ihm endet.
//
// Verglichen wird auf Zeitpunkten statt auf `(ausgesondert_am AT TIME ZONE …)::date`: Der
// Ausdruck um die Spalte herum macht den Teilindex unbrauchbar. Die
// Grenzen entstehen deshalb hier, in der Zeitzone der Schule — die Datenbanksitzung läuft
// in UTC, und zwischen Mitternacht in Berlin und Mitternacht UTC läge ein Abgang sonst im
// falschen Halbjahr.
func LadeAbgangsbuch(ctx context.Context, q Datenbank, von, bis time.Time) (Abgangsbuch, error) {
	loc := schulzeit.Zone()
	abVon := time.Date(von.Year(), von.Month(), von.Day(), 0, 0, 0, 0, loc)
	bisAusschliesslich := time.Date(bis.Year(), bis.Month(), bis.Day(), 0, 0, 0, 0, loc).AddDate(0, 0, 1)

	buch := Abgangsbuch{Von: abVon, Bis: bisAusschliesslich.AddDate(0, 0, -1), Zeilen: []AbgangsZeile{}}

	rows, err := q.Query(ctx, `
		SELECT e.ausgesondert_am, e.barcode_id, t.titel,
		       COALESCE(t.signatur, ''), COALESCE(e.aussonderung_grund, ''),
		       `+ExemplarTopfSQL+`
		FROM buecher_exemplare e
		JOIN buecher_titel t ON t.id = e.titel_id
		`+ExemplarTopfJoin+`
		WHERE `+sqlIstAbgang+`
		  AND e.ausgesondert_am >= $1 AND e.ausgesondert_am < $2
		-- Spalte 6 ist der Topf: 'land' vor 'schultraeger', die Reihenfolge des Ausdrucks.
		ORDER BY 6, e.ausgesondert_am, t.titel COLLATE deutsch, e.barcode_id
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
		z.GrundText = abgangsgrundText(z.Grund)
		buch.Zeilen = append(buch.Zeilen, z)
	}
	if err := rows.Err(); err != nil {
		return buch, err
	}

	if err := q.QueryRow(ctx, `
		SELECT count(*) FROM buecher_exemplare e
		WHERE `+sqlIstAbgang+` AND e.ausgesondert_am IS NULL
	`).Scan(&buch.OhneZeitpunkt); err != nil {
		return buch, err
	}

	// Die körperlich gelöschten Exemplare des Zeitraums — die einzige Spur, die von ihnen
	// bleibt. Gezählt wird am Zeitpunkt der Löschung, und das ist hier auch der richtige
	// Zeitpunkt: Für ein Exemplar, das aus der Tabelle verschwindet, ist die Löschung der
	// Abgang. Ein Ausbuchen davor stünde ohnehin schon in der Liste oben.
	if err := q.QueryRow(ctx, `
		SELECT count(*) FROM audit_log
		WHERE tabelle = 'buecher_exemplare' AND aktion = 'DELETE'
		  AND details->>'action' = ANY($1)
		  AND timestamp >= $2 AND timestamp < $3
		  -- Ältere Spuren tragen den Vermerk nicht und zählen weiter mit.
		  AND COALESCE((details->>'`+ProtokollDetailWarImBestand+`')::boolean, true)
	`, []string{ProtokollAktionTitelGeloescht, ProtokollAktionVerlustEndgueltigGeloescht},
		abVon, bisAusschliesslich).Scan(&buch.AusKatalogGeloescht); err != nil {
		return buch, err
	}
	return buch, nil
}
