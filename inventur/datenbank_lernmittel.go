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
}

const lernmittelZaehlung = `
	COUNT(e.id) FILTER (WHERE e.ist_ausgesondert = false AND e.bestellstatus IS NULL) AS gesamt,
	COUNT(e.id) FILTER (WHERE e.ist_ausgesondert = false AND a.id IS NOT NULL) AS verliehen,
	COUNT(e.id) FILTER (WHERE e.ist_ausleihbar = true AND e.ist_ausgesondert = false AND a.id IS NULL) AS verfuegbar`

const lernmittelJoins = `
	FROM buecher_titel b
	LEFT JOIN buecher_exemplare e ON e.titel_id = b.id
	LEFT JOIN ausleihen a ON a.exemplar_id = e.id AND a.rueckgabe_am IS NULL
	WHERE b.ist_lernmittel`

// GetLernmittelFaecher liefert je Fach die Zahlen; Titel ohne Fach als eigene Kachel
// (Fach ""), am Ende sortiert.
func (repo *BookRepository) GetLernmittelFaecher(ctx context.Context) ([]FachBestand, error) {
	rows, err := repo.db.Query(ctx, `
		SELECT fach, COUNT(*) AS titel, SUM(gesamt)::int, SUM(verliehen)::int, SUM(verfuegbar)::int
		FROM (
			SELECT COALESCE(b.subject, '') AS fach, b.id,`+lernmittelZaehlung+lernmittelJoins+`
			GROUP BY b.subject, b.id
		) t
		GROUP BY fach
		ORDER BY (fach = ''), fach`)
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
// alleFaecher=true alle Lernmittel, nach Fach und Titel sortiert (Export).
func (repo *BookRepository) GetLernmittelTitel(ctx context.Context, fach string, alleFaecher bool) ([]LernmittelTitel, error) {
	rows, err := repo.db.Query(ctx, `
		SELECT b.id, b.titel, COALESCE(b.autor, ''), COALESCE(b.subject, ''), COALESCE(b.cover_url, ''),
		       COALESCE(b.isbn, ''),`+lernmittelZaehlung+lernmittelJoins+`
		  AND ($1 OR COALESCE(b.subject, '') = $2)
		GROUP BY b.id, b.titel, b.autor, b.subject, b.cover_url, b.isbn
		ORDER BY (COALESCE(b.subject, '') = ''), b.subject, b.titel`, alleFaecher, fach)
	if err != nil {
		return nil, fmt.Errorf("lernmittel eines fachs: %w", err)
	}
	defer rows.Close()
	out := []LernmittelTitel{}
	for rows.Next() {
		var t LernmittelTitel
		if err := rows.Scan(&t.ID, &t.Title, &t.Autor, &t.Subject, &t.CoverURL, &t.ISBN, &t.Gesamt, &t.Verliehen, &t.Verfuegbar); err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}
