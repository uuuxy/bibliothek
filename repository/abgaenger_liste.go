package repository

import (
	"context"
	"time"
)

// abgaengerBedingung ist die EINE Definition, wer in der Abgängerliste steht: nicht
// gelöscht, noch an der Schule und in einer Abschlussklasse nach der Regel der Versetzung.
// Liste, Kontoauszug-Druck und Versand lesen alle dieses Prädikat.
var abgaengerBedingung = "s.deleted_at IS NULL AND s.ist_abgaenger = false AND " +
	AbschlussklasseSQL("s.klasse")

// AbgaengerMitBuechern ist ein Schüler einer Abschlussklasse mit der Zahl seiner offenen und
// davon überfälligen Bücher und der Adresse seiner Klassenleitung (leer, wenn keine
// hinterlegt ist).
type AbgaengerMitBuechern struct {
	ID            string
	BarcodeID     string
	Vorname       string
	Nachname      string
	Klasse        string
	AbgaengerJahr int
	IstGesperrt   bool
	OffeneBuecher int
	Ueberfaellig  int
	LehrerEmail   string
}

// ListeAbgaengerMitOffenenBuechern liefert eine Zeile je Abgänger mit offenen Ausleihen, nach
// Klasse und Nachname geordnet.
func ListeAbgaengerMitOffenenBuechern(ctx context.Context, db DBQueryer) ([]AbgaengerMitBuechern, error) {
	query := `
		SELECT s.id, s.barcode_id, s.vorname, s.nachname, s.klasse, s.abgaenger_jahr, s.ist_gesperrt,
		       COUNT(a.id)                                        AS offene_buecher,
		       COUNT(a.id) FILTER (WHERE a.rueckgabe_frist < now()) AS ueberfaellig,
		       coalesce(m.lehrer_email, '')                        AS lehrer_email
		FROM schueler s
		JOIN ausleihen a ON s.id = a.schueler_id
		-- Klassenleitung mitliefern: Ohne sie kann die Oberfläche VOR dem Versand nicht
		-- zeigen, welche Klasse überhaupt eine Adresse hat. Verglichen wird normalisiert
		-- (getrimmt, Kleinschreibung) — „5a", „5A" und „5a " sind dieselbe Klasse, und
		-- ein unsichtbares Leerzeichen darf keinen stillen Nullversand auslösen.
		LEFT JOIN klassen_lehrer_mapping m ON lower(btrim(m.klasse)) = lower(btrim(s.klasse))
		WHERE ` + abgaengerBedingung + `
		  AND a.rueckgabe_am IS NULL
		GROUP BY s.id, s.barcode_id, s.vorname, s.nachname, s.klasse, s.abgaenger_jahr, s.ist_gesperrt, m.lehrer_email
		ORDER BY s.klasse, s.nachname
	`
	rows, err := db.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	zeilen := []AbgaengerMitBuechern{}
	for rows.Next() {
		var z AbgaengerMitBuechern
		if err := rows.Scan(&z.ID, &z.BarcodeID, &z.Vorname, &z.Nachname, &z.Klasse, &z.AbgaengerJahr,
			&z.IstGesperrt, &z.OffeneBuecher, &z.Ueberfaellig, &z.LehrerEmail); err != nil {
			return nil, err
		}
		zeilen = append(zeilen, z)
	}
	return zeilen, rows.Err()
}

// AbgaengerAusleihe ist eine offene Ausleihe eines Abgängers mit seinem Namen und seiner
// Klasse: eine Zeile seines Kontoauszugs.
type AbgaengerAusleihe struct {
	SchuelerID      string
	Vorname         string
	Nachname        string
	Klasse          string
	Titel           string
	ExemplarBarcode string
	AusgeliehenAm   time.Time
	Frist           time.Time
}

// ListeAbgaengerAusleihen liefert die offenen Ausleihen der Abgänger nach Klasse, Nachname und
// Titel geordnet. Eine leere Klasse heißt alle; nurIDs engt auf die genannten Schüler ein
// (nil = keine Einengung) und ist ein Schnitt mit der Abgängerliste, keine zweite Auswahl.
func ListeAbgaengerAusleihen(ctx context.Context, db DBQueryer, klasse string, nurIDs []string) ([]AbgaengerAusleihe, error) {
	detailQuery := `
		SELECT s.id, s.vorname, s.nachname, s.klasse,
		       t.titel,
		       coalesce(e.barcode_id, '') AS ex_barcode,
		       a.ausgeliehen_am,
		       a.rueckgabe_frist
		FROM schueler s
		JOIN ausleihen a ON s.id = a.schueler_id AND a.rueckgabe_am IS NULL
		JOIN buecher_exemplare e ON a.exemplar_id = e.id
		JOIN buecher_titel t ON e.titel_id = t.id
		WHERE ` + abgaengerBedingung + `
		  AND ($1 = '' OR s.klasse = $1)
		  AND ($2::uuid[] IS NULL OR s.id = ANY($2::uuid[]))
		ORDER BY s.klasse, s.nachname, t.titel
	`
	rows, err := db.Query(ctx, detailQuery, klasse, nurIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ausleihen []AbgaengerAusleihe
	for rows.Next() {
		var a AbgaengerAusleihe
		if err := rows.Scan(&a.SchuelerID, &a.Vorname, &a.Nachname, &a.Klasse,
			&a.Titel, &a.ExemplarBarcode, &a.AusgeliehenAm, &a.Frist); err != nil {
			return nil, err
		}
		ausleihen = append(ausleihen, a)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return ausleihen, nil
}
