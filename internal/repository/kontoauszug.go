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
	// Dauerleihe ist die Ausleihe an jemanden, der kein Schüler ist (ausleihen.ist_handapparat):
	// Sie hat keine Frist und wird nie überfällig.
	Dauerleihe bool
	// IstGeraet heißt: Die Zeile ist ein verliehenes Gerät; Titel trägt den Modellnamen.
	IstGeraet bool
}

// LadeKontoauszugBuecher lädt die offenen Ausleihen eines Lesers, Bücher und Geräte, die
// früheste Frist zuerst. Eine unlesbare Zeile wird übersprungen.
func LadeKontoauszugBuecher(ctx context.Context, db Datenbank, schuelerID uuid.UUID) ([]KontoauszugBuch, error) {
	query := `
		SELECT COALESCE(t.titel, g.modellname, ''), COALESCE(e.barcode_id, g.barcode_id, ''), a.ausgeliehen_am, a.rueckgabe_frist, a.ist_handapparat,
		       a.geraet_id IS NOT NULL
		FROM ausleihen a
		LEFT JOIN buecher_exemplare e ON a.exemplar_id = e.id
		LEFT JOIN buecher_titel t ON e.titel_id = t.id
		LEFT JOIN geraete g ON a.geraet_id = g.id
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
		if err := rows.Scan(&b.Titel, &b.Barcode, &b.Ausleihdatum, &b.Rueckgabedatum, &b.Dauerleihe, &b.IstGeraet); err != nil {
			return nil, zeileUnlesbar(err)
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

// LadeKontoauszugKopf liest Name und Klasse eines Lesers, der nicht im Papierkorb liegt, Schüler
// oder Kollegium; ohne Klasse ist sie leer. pgx.ErrNoRows sonst.
func LadeKontoauszugKopf(ctx context.Context, db Datenbank, leserID uuid.UUID) (KontoauszugKopf, error) {
	var k KontoauszugKopf
	err := db.QueryRow(ctx, `
			SELECT vorname, nachname, coalesce(klasse, '')
			FROM leser WHERE id = $1 AND deleted_at IS NULL
		`, leserID).Scan(&k.Vorname, &k.Nachname, &k.Klasse)
	return k, err
}
