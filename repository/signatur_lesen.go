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

// ListeSignaturGruppen liefert die Regaladressen, die an Titeln vorkommen, mit ihrem Umfang.
// Ausgesonderte Exemplare zählen nicht mit.
func ListeSignaturGruppen(ctx context.Context, db DBQueryer) ([]SignaturGruppe, error) {
	regal := SQLSignaturRegaladresse("t.signatur")
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

// SignaturBuch ist ein Titel der Regalansicht mit der Zahl seiner Exemplare und der davon
// verliehenen.
type SignaturBuch struct {
	TitelID   string
	Signatur  string
	Titel     string
	Autor     string
	ISBN      string
	Exemplare int
	Verliehen int
}

// ListeBuecherUnterSignatur liefert die Titel, deren Signatur mit dem Präfix beginnt, in der
// Reihenfolge des Regals und höchstens limit Zeilen. Das Prädikat ist SignaturPraefixBedingung,
// dasselbe wie beim Bereich einer Inventur.
func ListeBuecherUnterSignatur(ctx context.Context, db DBQueryer, signatur string, limit int) ([]SignaturBuch, error) {
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
		`, SignaturPraefixBedingung("t.signatur", 1)), signatur, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	buecher := []SignaturBuch{}
	for rows.Next() {
		var b SignaturBuch
		if err := rows.Scan(&b.TitelID, &b.Signatur, &b.Titel, &b.Autor, &b.ISBN, &b.Exemplare, &b.Verliehen); err != nil {
			return nil, zeileUnlesbar(err)
		}
		buecher = append(buecher, b)
	}
	return buecher, rows.Err()
}
