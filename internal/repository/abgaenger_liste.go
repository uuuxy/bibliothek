package repository

import (
	"context"
	"time"
)

// abgaengerBedingung ist die eine Definition, wer in der Abgängerliste steht: nicht
// gelöscht, noch an der Schule und in einer Abschlussklasse nach der Regel der Versetzung.
// Liste, Kontoauszug-Druck und Versand lesen alle dieses Prädikat.
var abgaengerBedingung = "s.deleted_at IS NULL AND s.ist_abgaenger = false AND " +
	AbschlussklasseSQL("s.klasse")

// AbgaengerMitAusleihen ist ein Schüler einer Abschlussklasse mit der Zahl seiner offenen und
// davon überfälligen Bücher und der Adresse seiner Klassenleitung (leer, wenn keine
// hinterlegt ist). IstGesperrt ist die Sperre des Programms, IstVonHandGesperrt die von Hand;
// den Vermerk zeigt die Oberfläche nach beiden (sperrStatus.ausleiheGesperrt).
type AbgaengerMitAusleihen struct {
	ID                 string
	BarcodeID          string
	Vorname            string
	Nachname           string
	Klasse             string
	AbgaengerJahr      int
	IstGesperrt        bool
	IstVonHandGesperrt bool
	OffeneAusleihen    int
	Ueberfaellig       int
	LehrerEmail        string
}

// LadeAbgaengerMitOffenenAusleihen liefert eine Zeile je Abgänger mit offenen Ausleihen, nach
// Klasse und Nachname geordnet.
func LadeAbgaengerMitOffenenAusleihen(ctx context.Context, db Datenbank) ([]AbgaengerMitAusleihen, error) {
	query := `
		SELECT s.id, s.barcode_id, s.vorname, s.nachname, s.klasse, s.abgaenger_jahr, s.ist_gesperrt,
		       coalesce(s.is_manually_blocked, false),
		       COUNT(a.id)                                        AS offene_buecher,
		       COUNT(a.id) FILTER (WHERE a.rueckgabe_frist < now()) AS ueberfaellig,
		       coalesce(m.lehrer_email, '')                        AS lehrer_email
		FROM schueler s
		JOIN ausleihen a ON s.id = a.schueler_id
		-- Klassenleitung mitliefern: Ohne sie kann die Oberfläche vor dem Versand nicht
		-- zeigen, welche Klasse überhaupt eine Adresse hat. Verglichen wird normalisiert
		-- (getrimmt, Kleinschreibung) — „5a", „5A" und „5a " sind dieselbe Klasse, und
		-- ein unsichtbares Leerzeichen darf keinen stillen Nullversand auslösen.
		LEFT JOIN klassen_lehrer_mapping m ON lower(btrim(m.klasse)) = lower(btrim(s.klasse))
		WHERE ` + abgaengerBedingung + `
		  AND a.rueckgabe_am IS NULL
		GROUP BY s.id, s.barcode_id, s.vorname, s.nachname, s.klasse, s.abgaenger_jahr, s.ist_gesperrt, s.is_manually_blocked, m.lehrer_email
		ORDER BY s.klasse COLLATE deutsch, s.nachname COLLATE deutsch
	`
	rows, err := db.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	zeilen := []AbgaengerMitAusleihen{}
	for rows.Next() {
		var z AbgaengerMitAusleihen
		if err := rows.Scan(&z.ID, &z.BarcodeID, &z.Vorname, &z.Nachname, &z.Klasse, &z.AbgaengerJahr,
			&z.IstGesperrt, &z.IstVonHandGesperrt, &z.OffeneAusleihen, &z.Ueberfaellig, &z.LehrerEmail); err != nil {
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
	// IstGeraet heißt: Die Zeile ist ein verliehenes Gerät; Titel trägt den Modellnamen.
	IstGeraet bool
}

// LadeAbgaengerAusleihen liefert die offenen Ausleihen der Abgänger nach Klasse, Nachname und
// Titel geordnet. Eine leere Klasse heißt alle; nurIDs engt auf die genannten Schüler ein
// (nil = keine Einengung) und ist ein Schnitt mit der Abgängerliste, keine zweite Auswahl.
func LadeAbgaengerAusleihen(ctx context.Context, db Datenbank, klasse string, nurIDs []string) ([]AbgaengerAusleihe, error) {
	detailQuery := `
		SELECT s.id, s.vorname, s.nachname, s.klasse,
		       COALESCE(t.titel, g.modellname, '') AS titel,
		       COALESCE(e.barcode_id, g.barcode_id, '') AS ex_barcode,
		       a.ausgeliehen_am,
		       a.rueckgabe_frist,
		       a.geraet_id IS NOT NULL
		FROM schueler s
		JOIN ausleihen a ON s.id = a.schueler_id AND a.rueckgabe_am IS NULL
		LEFT JOIN buecher_exemplare e ON a.exemplar_id = e.id
		LEFT JOIN buecher_titel t ON e.titel_id = t.id
		LEFT JOIN geraete g ON a.geraet_id = g.id
		WHERE ` + abgaengerBedingung + `
		  AND ($1 = '' OR s.klasse = $1)
		  AND ($2::uuid[] IS NULL OR s.id = ANY($2::uuid[]))
		ORDER BY s.klasse COLLATE deutsch, s.nachname COLLATE deutsch, COALESCE(t.titel, g.modellname, '') COLLATE deutsch
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
			&a.Titel, &a.ExemplarBarcode, &a.AusgeliehenAm, &a.Frist, &a.IstGeraet); err != nil {
			return nil, err
		}
		ausleihen = append(ausleihen, a)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return ausleihen, nil
}
