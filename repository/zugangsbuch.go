package repository

import (
	"context"
	"time"

	"bibliothek/pkg/schulzeit"
)

// Das Zugangsbuch: welche Exemplare in einem Zeitraum in den Bestand gekommen sind.
//
// Die Gegenrichtung zum Abgangsbuch (abgangsbuch.go) und die zweite Hälfte von Punkt 1 des
// Protokolls vom 16.09.2026. Die Arbeitshilfe verlangt „jede Lieferung mit Eingangsdatum,
// Titel, Anzahl, Lieferant, Inventarnummer; bei EDV-Führung je Schulhalbjahr ein Ausdruck
// der Neuanschaffungen" (docs/mittel_konzept.md 7.1).
//
// Das Datum kommt aus `zugang_am` (Migration 129): der Tag, an dem das Exemplar in den
// Bestand kam. Bis dahin las das Buch `erworben_am` — den Tag, an dem die ZEILE entstand,
// und die entsteht im Bestellweg beim Bestellen. Ein im Dezember bestelltes, im Februar
// geliefertes Buch stand damit im Dezember, und Exemplare im Zulauf standen darin, obwohl
// sie nicht da waren. Bei der Littera-Übernahme ist es weiterhin das echte Datum aus der
// Altanwendung, nicht das des Imports.

// ZugangsZeile ist ein Zugang, wie er im Buch steht.
type ZugangsZeile struct {
	Datum    time.Time `json:"datum"`
	Barcode  string    `json:"barcode"`
	Titel    string    `json:"titel"`
	Signatur string    `json:"signatur"`
	// Lieferant steht im Zugangsbuch, weil die Arbeitshilfe ihn verlangt. Leer, wenn keine
	// Bestellung hinterlegt ist — dann gibt es keinen Lieferanten, den man nennen könnte.
	Lieferant string `json:"lieferant"`
	// Topf: der Topf der BESTELLUNG, aus der das Exemplar kam. Leer, wenn keine hinterlegt
	// ist. Bewusst nicht aus `ist_lernmittel` des Titels abgeleitet: Beim Zugang geht es
	// darum, aus welchem Geld das Buch bezahlt wurde, und das steht an der Bestellung
	// (Migration 109, „eine Bestellung = ein Topf"). Der Titel ist dort nur ein Vorschlag —
	// er darf im Warenkorb umgehängt werden, und genau dann wäre die Ableitung falsch.
	Topf string `json:"topf"`
}

// Zugangsbuch bündelt Zeitraum und Zeilen.
type Zugangsbuch struct {
	Von    time.Time      `json:"von"`
	Bis    time.Time      `json:"bis"`
	Zeilen []ZugangsZeile `json:"zeilen"`
}

// LadeZugangsbuch liest die Zugänge eines Zeitraums. `von` und `bis` sind Kalendertage der
// Schule, beide EINSCHLIESSLICH.
//
// Ausgesonderte Exemplare bleiben drin: Ein Buch, das im Mai kam und im Juli verloren ging,
// ist im Mai trotzdem zugegangen. Ein Zugangsbuch, das solche Zeilen weglässt, ändert
// rückwirkend eine Zahl, die jemand unterschrieben hat — der Abgang steht im Abgangsbuch.
func LadeZugangsbuch(ctx context.Context, q DBQueryer, von, bis time.Time) (Zugangsbuch, error) {
	loc := schulzeit.Zone()
	abVon := time.Date(von.Year(), von.Month(), von.Day(), 0, 0, 0, 0, loc)
	bisTag := time.Date(bis.Year(), bis.Month(), bis.Day(), 0, 0, 0, 0, loc)

	buch := Zugangsbuch{Von: abVon, Bis: bisTag, Zeilen: []ZugangsZeile{}}

	// Gelesen wird `zugang_am` (Migration 129), nicht `erworben_am`: Im Bestellweg entsteht
	// die Exemplarzeile beim BESTELLEN, der Zugang ist erst der Wareneingang. Exemplare im
	// Zulauf tragen NULL und fallen damit heraus — ein Nachweis über den Bestand führt keine
	// Bücher, die noch beim Händler liegen.
	//
	// `zugang_am` ist ein DATUM, kein Zeitpunkt — hier wird deshalb auf Tagen verglichen,
	// anders als beim Abgangsbuch. Ein AT TIME ZONE darum herum wäre nicht nur überflüssig,
	// sondern falsch: Es machte aus dem 16.09. je nach Sitzungszeitzone den 15.
	rows, err := q.Query(ctx, `
		SELECT e.zugang_am, e.barcode_id, t.titel, COALESCE(t.signatur, ''),
		       COALESCE(b.lieferant_name, ''), COALESCE(b.mittel, '')
		FROM buecher_exemplare e
		JOIN buecher_titel t ON t.id = e.titel_id
		LEFT JOIN bestellungen_verlauf b ON b.id = e.bestellung_id
		WHERE e.zugang_am >= $1::date AND e.zugang_am <= $2::date
		ORDER BY e.zugang_am, t.titel, e.barcode_id
	`, abVon, bisTag)
	if err != nil {
		return buch, err
	}
	defer rows.Close()

	for rows.Next() {
		var z ZugangsZeile
		if err := rows.Scan(&z.Datum, &z.Barcode, &z.Titel, &z.Signatur, &z.Lieferant, &z.Topf); err != nil {
			return buch, err
		}
		buch.Zeilen = append(buch.Zeilen, z)
	}
	if err := rows.Err(); err != nil {
		return buch, err
	}
	return buch, nil
}
