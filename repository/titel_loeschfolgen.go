package repository

// Was das Löschen von Titeln mit sich bringt, gelesen vor der Rückfrage: wie viele Exemplare im
// Haus mit dem Grund ins Abgangsbuch gehen und welche verliehen sind, mit dem Leser, der sie hat.
// Es gelten dieselben Bedingungen wie beim Löschen (sqlIstImHaus, sqlIstVerliehen), damit der
// Dialog sagt, was geschieht.

import (
	"context"
	"fmt"
)

// Loeschfolgen ist die Vorschau des Löschdialogs.
type Loeschfolgen struct {
	// Titel ist die Zahl der gefundenen Titel.
	Titel  int `json:"titel"`
	ImHaus int `json:"im_haus"`
	// Verliehen nennt jedes verliehene Exemplar der Titel. Die Liste ist so lang wie die
	// laufenden Ausleihen der gewählten Titel; die Tür begrenzt die Zahl der Titel.
	Verliehen []VerliehenesExemplar `json:"verliehen"`
}

// VerliehenesExemplar ist ein verliehenes Exemplar mit dem Leser, der es hat.
type VerliehenesExemplar struct {
	ExemplarID string `json:"exemplar_id"`
	Barcode    string `json:"barcode"`
	Titel      string `json:"titel"`
	Leser      string `json:"leser"`
	Klasse     string `json:"klasse"`
}

// LadeLoeschfolgen liest die Vorschau für die genannten Titel. Eine unbekannte Kennung zählt
// nicht mit.
func (a *TitelAbfragen) LadeLoeschfolgen(ctx context.Context, ids []string) (Loeschfolgen, error) {
	folgen := Loeschfolgen{Verliehen: []VerliehenesExemplar{}}
	if err := a.pool.QueryRow(ctx, `
		SELECT (SELECT count(*) FROM buecher_titel WHERE id = ANY($1::uuid[])),
		       (SELECT count(*) FROM buecher_exemplare e
		        WHERE e.titel_id = ANY($1::uuid[]) AND `+sqlIstImHaus+`)`,
		ids).Scan(&folgen.Titel, &folgen.ImHaus); err != nil {
		return folgen, fmt.Errorf("löschfolgen lesen: %w", err)
	}
	zeilen, err := a.pool.Query(ctx, `
		SELECT e.id::text, e.barcode_id, t.titel, `+sqlEntleiherName+`, coalesce(l.klasse, '')
		FROM buecher_exemplare e
		JOIN buecher_titel t ON t.id = e.titel_id
		JOIN ausleihen a ON a.exemplar_id = e.id AND a.rueckgabe_am IS NULL
		LEFT JOIN leser l ON l.id = a.schueler_id
		WHERE e.titel_id = ANY($1::uuid[])
		ORDER BY t.titel COLLATE deutsch, e.barcode_id`, ids)
	if err != nil {
		return folgen, fmt.Errorf("verliehene exemplare lesen: %w", err)
	}
	defer zeilen.Close()
	for zeilen.Next() {
		var v VerliehenesExemplar
		if err := zeilen.Scan(&v.ExemplarID, &v.Barcode, &v.Titel, &v.Leser, &v.Klasse); err != nil {
			return folgen, fmt.Errorf("verliehene exemplare lesen: %w", zeileUnlesbar(err))
		}
		folgen.Verliehen = append(folgen.Verliehen, v)
	}
	// Bräche die Iteration ab, fehlte im Dialog ein verliehenes Buch, und sein Titel bliebe.
	if err := zeilen.Err(); err != nil {
		return folgen, fmt.Errorf("verliehene exemplare lesen: %w", err)
	}
	return folgen, nil
}
