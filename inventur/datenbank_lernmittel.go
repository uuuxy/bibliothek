package inventur

import (
	"context"
	"fmt"
)

// Schulbücher je Fach fürs Lehrerportal (Peter, 03.09.2026): Die Fachsprecher wollen
// wissen, wie viele Mathebücher die Schule hat. Grundlage ist ausschließlich der
// Lernmittel-Schalter (buecher_titel.ist_lernmittel, Migration 093) und das Fach
// (buecher_titel.subject, FK auf systematik_kategorien). Zählweise wie bei den
// Klassensätzen (GetClassGroups): gesamt = nicht ausgesondert und nicht nur bestellt;
// verliehen = offene Ausleihe; verfügbar = ausleihbar, nicht ausgesondert, nicht verliehen.

// FachBestand ist eine Fach-Kachel: Titel und Exemplare eines Fachs. Fach "" = ohne Fach.
type FachBestand struct {
	Fach       string `json:"fach"`
	Titel      int    `json:"titel"`
	Gesamt     int    `json:"gesamt"`
	Verliehen  int    `json:"verliehen"`
	Verfuegbar int    `json:"verfuegbar"`
}

// LernmittelTitel ist ein Schulbuch in der Fach-Liste — Feldnamen wie ClassBook, damit
// die Kachel des Klassensatzes (KlassenBuchKachel) sie unverändert zeigt.
type LernmittelTitel struct {
	ID         string `json:"id"`
	Title      string `json:"title"`
	Autor      string `json:"autor"`
	Subject    string `json:"subject"`
	CoverURL   string `json:"coverUrl"`
	ISBN       string `json:"isbn"`
	Gesamt     int    `json:"gesamt"`
	Verliehen  int    `json:"verliehen"`
	Verfuegbar int    `json:"verfuegbar"`
	// Jahrgang aus Signatur/Import (Migration 093); 5–10 ist die Spalten-Vorgabe und
	// damit von „unbekannt" nicht zu unterscheiden — die Oberfläche zeigt sie nicht.
	JahrgangVon int `json:"jahrgangVon"`
	JahrgangBis int `json:"jahrgangBis"`
	// Schulzweig (buecher_titel.track): am Schulbuch gepflegt, seit 03.09.2026 wieder in
	// der Maske. Leer heißt „gilt für alle Zweige", nicht „unbekannt".
	Track string `json:"track"`
	// Gezaehlt ist das Datum der letzten Zählung (buecher_titel.last_counted) als
	// „TT.MM.JJJJ"; leer heißt „noch nie gezählt". Peter, 03.09.2026: Eine Bestandszahl
	// ohne Datum sagt nicht, wie alt sie ist — im Ausdruck steht sie sonst so da, als
	// wäre sie von heute.
	Gezaehlt string `json:"gezaehlt"`
}

// LernmittelFilter sind die drei Einschränkungen des Portal-Reiters. Nullwerte heißen
// „alles": Jahrgang 0, Zweig "" (ZweigOhne = nur Bücher ohne Zweig), Suche "".
type LernmittelFilter struct {
	Jahrgang int
	Zweig    string
	Suche    string
}

// ZweigOhne ist der Filterwert für „kein Schulzweig gesetzt" — ein Zeichen, das als
// Zweigname nicht vorkommt, damit es keinen echten Wert verdeckt.
const ZweigOhne = "-"

// lernmittelFilterSQL: $1 Jahrgang (0 = alle; ein Atlas 5–10 zählt bei Jahrgang 7 mit),
// $2 Zweig, $3 Suchtext über Titel, ISBN, Autor und Fach.
//
// Zum Zweig: Ein leerer track heißt „gilt für ALLE Zweige" — so sagt es die Buchmaske
// („leer heißt gilt für alle"), und so ist der Altbestand: Littera hat den Zweig nie
// geliefert, 470 von 578 Schulbüchern tragen keinen. Ein Filter auf „Gymnasium", der
// nur `track = 'Gymnasium'` prüft, versteckte deshalb fünf Sechstel des Bestands vor dem
// Fachsprecher (Befund 03.09.2026). Gesucht wird darum „dieser Zweig ODER für alle";
// ZweigOhne ist die Gegenrichtung — nur die ohne Angabe.
const lernmittelFilterSQL = `
	AND ($1 = 0 OR (b.jahrgang_von <= $1 AND b.jahrgang_bis >= $1))
	AND ($2 = '' OR ($2 = '` + ZweigOhne + `' AND COALESCE(b.track, '') = '')
	     OR ($2 <> '` + ZweigOhne + `' AND (COALESCE(b.track, '') = '' OR b.track = $2)))
	AND ($3 = '' OR b.titel ILIKE '%' || $3 || '%' OR COALESCE(b.isbn, '') ILIKE '%' || $3 || '%'
	     OR COALESCE(b.autor, '') ILIKE '%' || $3 || '%' OR COALESCE(b.subject, '') ILIKE '%' || $3 || '%')`

const lernmittelZaehlung = `
	COUNT(e.id) FILTER (WHERE e.ist_ausgesondert = false AND e.bestellstatus IS NULL) AS gesamt,
	COUNT(e.id) FILTER (WHERE e.ist_ausgesondert = false AND a.id IS NOT NULL) AS verliehen,
	COUNT(e.id) FILTER (WHERE e.ist_ausleihbar = true AND e.ist_ausgesondert = false AND a.id IS NULL) AS verfuegbar`

const lernmittelJoins = `
	FROM buecher_titel b
	LEFT JOIN buecher_exemplare e ON e.titel_id = b.id
	LEFT JOIN ausleihen a ON a.exemplar_id = e.id AND a.rueckgabe_am IS NULL
	WHERE b.ist_lernmittel`

// GetLernmittelFaecher liefert je Fach die Zahlen; Titel ohne Fach als eigene Zeile
// (Fach ""), am Ende sortiert.
func (repo *BookRepository) GetLernmittelFaecher(ctx context.Context, f LernmittelFilter) ([]FachBestand, error) {
	rows, err := repo.db.Query(ctx, `
		SELECT fach, COUNT(*) AS titel, SUM(gesamt)::int, SUM(verliehen)::int, SUM(verfuegbar)::int
		FROM (
			SELECT COALESCE(b.subject, '') AS fach, b.id,`+lernmittelZaehlung+lernmittelJoins+lernmittelFilterSQL+`
			GROUP BY b.subject, b.id
		) t
		GROUP BY fach
		ORDER BY (fach = ''), fach`, f.Jahrgang, f.Zweig, f.Suche)
	if err != nil {
		return nil, fmt.Errorf("lernmittel je fach: %w", err)
	}
	defer rows.Close()
	out := []FachBestand{}
	for rows.Next() {
		var f FachBestand
		if err := rows.Scan(&f.Fach, &f.Titel, &f.Gesamt, &f.Verliehen, &f.Verfuegbar); err != nil {
			return nil, err
		}
		out = append(out, f)
	}
	return out, rows.Err()
}

// GetLernmittelTitel liefert die Schulbücher eines Fachs (fach "" = ohne Fach); mit
// alleFaecher=true alle Lernmittel, nach Fach und Titel sortiert.
func (repo *BookRepository) GetLernmittelTitel(ctx context.Context, fach string, alleFaecher bool, f LernmittelFilter) ([]LernmittelTitel, error) {
	rows, err := repo.db.Query(ctx, `
		SELECT b.id, b.titel, COALESCE(b.autor, ''), COALESCE(b.subject, ''), COALESCE(b.cover_url, ''),
		       COALESCE(b.isbn, ''), b.jahrgang_von, b.jahrgang_bis, COALESCE(b.track, ''),
		       COALESCE(TO_CHAR(b.last_counted, 'DD.MM.YYYY'), ''),`+lernmittelZaehlung+lernmittelJoins+lernmittelFilterSQL+`
		  AND ($4 OR COALESCE(b.subject, '') = $5)
		GROUP BY b.id, b.titel, b.autor, b.subject, b.cover_url, b.isbn, b.jahrgang_von, b.jahrgang_bis, b.track, b.last_counted
		ORDER BY (COALESCE(b.subject, '') = ''), b.subject, b.jahrgang_von, b.titel`,
		f.Jahrgang, f.Zweig, f.Suche, alleFaecher, fach)
	if err != nil {
		return nil, fmt.Errorf("lernmittel eines fachs: %w", err)
	}
	defer rows.Close()
	out := []LernmittelTitel{}
	for rows.Next() {
		var t LernmittelTitel
		if err := rows.Scan(&t.ID, &t.Title, &t.Autor, &t.Subject, &t.CoverURL, &t.ISBN, &t.JahrgangVon, &t.JahrgangBis, &t.Track, &t.Gezaehlt, &t.Gesamt, &t.Verliehen, &t.Verfuegbar); err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}
