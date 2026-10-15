package repository

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// RechnungsPosition ist eine offene Forderung auf der Ersatzforderung. Land sagt, ob die
// Position ein Buch des Landes betrifft; ein Geräteschaden ist es nie.
type RechnungsPosition struct {
	Titel        string
	Barcode      string
	Ausleihdatum time.Time
	Ersatzpreis  float64
	Land         bool
}

// LadeOffeneForderungenOhneBescheid lädt die unbezahlten Schadensfälle eines Schülers als
// Positionen der Ersatzforderung.
//
// Mit LEFT JOIN, weil exemplar_id und ausleihe_id leer sein können (ausleihe_id steht auf ON
// DELETE SET NULL): Mit INNER JOIN fiele eine Forderung ohne diese Bezüge aus dem Brief,
// während die Akte den Betrag weiter zeigt und der Schüler gesperrt bleibt. Den Geräteschaden
// sieht das Schema vor (check_damage_item verlangt genau eines von Exemplar und Gerät), auch
// wenn ihn noch nichts schreibt; er steht mit Modellname und Barcode des Geräts da. Bleibt
// beides leer, trägt die Beschreibung des Schadensfalls die Zeile, und als Ausleihdatum gilt
// das Datum der Forderung.
//
// Eine Forderung, die schon auf einem Schadensersatz-Bescheid steht, gehört nicht hierher:
// Dieselbe Forderung auf beiden Briefen wäre zweimal dieselbe Zahlungsaufforderung, mit zwei
// Fristen und zwei Nummern.
//
// Eine unlesbare Zeile bricht ab: Eine Rechnung, der eine Position fehlt, nennt einen zu
// kleinen Betrag.
func LadeOffeneForderungenOhneBescheid(ctx context.Context, db Datenbank, schuelerID uuid.UUID) ([]RechnungsPosition, error) {
	query := `
		SELECT COALESCE(t.titel, g.modellname, sf.beschreibung),
		       COALESCE(e.barcode_id, g.barcode_id, ''),
		       COALESCE(a.ausgeliehen_am, sf.erstellt_am),
		       sf.betrag,
		       -- Der Topf der Position und damit ihr Zahlungsweg: das Eigentum des Exemplars
		       -- (repository.ExemplarTopfSQL), dieselbe Regel wie am Etikett und im Bescheid.
		       -- Ohne Exemplar (Geräteschaden) ist es kein Buch des Landes: Die Bedingung auf
		       -- e.id liefert dann false statt NULL (NULL-Scan-Bugklasse).
		       (e.id IS NOT NULL AND ` + ExemplarTopfSQL + ` = 'land')
		FROM schadensfaelle sf
		LEFT JOIN buecher_exemplare e ON sf.exemplar_id = e.id
		LEFT JOIN buecher_titel t ON e.titel_id = t.id
		` + ExemplarTopfJoin + `
		LEFT JOIN geraete g ON sf.geraet_id = g.id
		LEFT JOIN ausleihen a ON sf.ausleihe_id = a.id
		WHERE sf.schueler_id = $1 AND sf.ist_bezahlt = false
		  AND sf.bescheid_id IS NULL
	`
	rows, err := db.Query(ctx, query, schuelerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var positionen []RechnungsPosition
	for rows.Next() {
		var p RechnungsPosition
		if err := rows.Scan(&p.Titel, &p.Barcode, &p.Ausleihdatum, &p.Ersatzpreis, &p.Land); err != nil {
			return nil, err
		}
		positionen = append(positionen, p)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return positionen, nil
}

// HatOffeneForderungAufBescheid sagt, ob eine unbezahlte Forderung des Lesers auf einem
// Schadensersatz-Bescheid steht.
func HatOffeneForderungAufBescheid(ctx context.Context, db Datenbank, schuelerID uuid.UUID) (bool, error) {
	var aufBescheid bool
	err := db.QueryRow(ctx, `
			SELECT EXISTS (SELECT 1 FROM schadensfaelle
			               WHERE schueler_id = $1 AND ist_bezahlt = false AND bescheid_id IS NOT NULL)
		`, schuelerID).Scan(&aufBescheid)
	return aufBescheid, err
}

// BriefAnschrift sind Name und Anschrift eines Schülers für einen Brief im Fensterkuvert.
type BriefAnschrift struct {
	Vorname    string
	Nachname   string
	Strasse    string
	Hausnummer string
	PLZ        string
	Ort        string
}

// LadeBriefAnschrift liest Name und Anschrift eines Schülers, der nicht im Papierkorb liegt;
// pgx.ErrNoRows sonst. Leere Anschriftfelder kommen als leerer Text.
func LadeBriefAnschrift(ctx context.Context, db Datenbank, schuelerID uuid.UUID) (BriefAnschrift, error) {
	var a BriefAnschrift
	err := db.QueryRow(ctx, `
		SELECT vorname, nachname,
		       COALESCE(strasse, ''), COALESCE(hausnummer, ''),
		       COALESCE(plz, ''), COALESCE(ort, '')
		FROM schueler WHERE id = $1 AND deleted_at IS NULL
	`, schuelerID).Scan(&a.Vorname, &a.Nachname, &a.Strasse, &a.Hausnummer, &a.PLZ, &a.Ort)
	return a, err
}
