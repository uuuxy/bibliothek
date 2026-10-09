package repository

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// KontoauszugBuch ist ein ausgeliehenes Buch auf dem Kontoauszug eines Schülers.
type KontoauszugBuch struct {
	Titel          string
	Barcode        string
	Ausleihdatum   time.Time
	Rueckgabedatum time.Time
}

// ListeKontoauszugBuecher lädt die offenen Ausleihen eines Schülers, die früheste Frist
// zuerst. Eine unlesbare Zeile wird übersprungen.
func ListeKontoauszugBuecher(ctx context.Context, db DBQueryer, schuelerID uuid.UUID) ([]KontoauszugBuch, error) {
	query := `
		SELECT t.titel, e.barcode_id, a.ausgeliehen_am, a.rueckgabe_frist
		FROM ausleihen a
		JOIN buecher_exemplare e ON a.exemplar_id = e.id
		JOIN buecher_titel t ON e.titel_id = t.id
		WHERE a.schueler_id = $1 AND a.rueckgabe_am IS NULL
		ORDER BY a.rueckgabe_frist ASC
	`
	rows, err := db.Query(ctx, query, schuelerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var buecher []KontoauszugBuch
	for rows.Next() {
		var b KontoauszugBuch
		if err := rows.Scan(&b.Titel, &b.Barcode, &b.Ausleihdatum, &b.Rueckgabedatum); err != nil {
			continue
		}
		buecher = append(buecher, b)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return buecher, nil
}

// KontoauszugKopf sind Name und Klasse des Schülers auf seinem Kontoauszug.
type KontoauszugKopf struct {
	Vorname  string
	Nachname string
	Klasse   string
}

// LadeKontoauszugKopf liest Name und Klasse eines Schülers, der nicht im Papierkorb liegt;
// pgx.ErrNoRows sonst.
func LadeKontoauszugKopf(ctx context.Context, db DBQueryer, schuelerID uuid.UUID) (KontoauszugKopf, error) {
	var k KontoauszugKopf
	err := db.QueryRow(ctx, `
			SELECT vorname, nachname, klasse
			FROM schueler WHERE id = $1 AND deleted_at IS NULL
		`, schuelerID).Scan(&k.Vorname, &k.Nachname, &k.Klasse)
	return k, err
}
