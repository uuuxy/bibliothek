package repository

import (
	"context"
	"fmt"
)

// SignaturGruppe ist eine Regaladresse mit der Zahl ihrer Titel und Exemplare.
type SignaturGruppe struct {
	Signatur  string
	Titel     int
	Exemplare int
}

// LadeSignaturGruppen liefert die Regaladressen, die an Titeln vorkommen, mit ihrem Umfang.
// Ausgesonderte Exemplare zählen nicht mit.
func LadeSignaturGruppen(ctx context.Context, db Datenbank) ([]SignaturGruppe, error) {
	regal := sqlSignaturRegaladresse("t.signatur")
	rows, err := db.Query(ctx, `
			SELECT `+regal+` AS signatur,
			       count(DISTINCT t.id) AS titel,
			       count(e.id) FILTER (WHERE e.ist_ausgesondert = false) AS exemplare
			FROM buecher_titel t
			LEFT JOIN buecher_exemplare e ON e.titel_id = t.id
			WHERE COALESCE(btrim(t.signatur), '') <> ''
			GROUP BY `+regal+`
			ORDER BY `+regal+`
		`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	gruppen := []SignaturGruppe{}
	for rows.Next() {
		var g SignaturGruppe
		if err := rows.Scan(&g.Signatur, &g.Titel, &g.Exemplare); err != nil {
			return nil, zeileUnlesbar(err)
		}
		gruppen = append(gruppen, g)
	}
	return gruppen, rows.Err()
}

// SignaturTitel ist ein Titel der Regalansicht mit der Zahl seiner Exemplare und der davon
// verliehenen.
type SignaturTitel struct {
	TitelID   string
	Signatur  string
	Titel     string
	Autor     string
	ISBN      string
	Exemplare int
	Verliehen int
}

// LadeTitelUnterSignatur liefert die Titel, deren Signatur mit dem Präfix beginnt, in der
// Reihenfolge des Regals und höchstens limit Zeilen. Das Prädikat ist signaturPraefixBedingung,
// dasselbe wie beim Bereich einer Inventur.
func LadeTitelUnterSignatur(ctx context.Context, db Datenbank, signatur string, limit int) ([]SignaturTitel, error) {
	rows, err := db.Query(ctx, fmt.Sprintf(`
			SELECT t.id::text,
			       btrim(t.signatur),
			       t.titel,
			       COALESCE(t.autor, ''),
			       COALESCE(t.isbn, ''),
			       count(e.id) FILTER (WHERE e.ist_ausgesondert = false) AS exemplare,
			       count(e.id) FILTER (
			           WHERE e.ist_ausgesondert = false
			             AND EXISTS (SELECT 1 FROM ausleihen a
			                         WHERE a.exemplar_id = e.id AND a.rueckgabe_am IS NULL)
			       ) AS verliehen
			FROM buecher_titel t
			LEFT JOIN buecher_exemplare e ON e.titel_id = t.id
			WHERE %s
			GROUP BY t.id, t.signatur, t.titel, t.autor, t.isbn
			ORDER BY btrim(t.signatur), t.titel
			LIMIT $2
		`, signaturPraefixBedingung("t.signatur", 1)), signatur, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	buecher := []SignaturTitel{}
	for rows.Next() {
		var b SignaturTitel
		if err := rows.Scan(&b.TitelID, &b.Signatur, &b.Titel, &b.Autor, &b.ISBN, &b.Exemplare, &b.Verliehen); err != nil {
			return nil, zeileUnlesbar(err)
		}
		buecher = append(buecher, b)
	}
	return buecher, rows.Err()
}
