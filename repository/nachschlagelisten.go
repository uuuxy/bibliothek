package repository

import "context"

// SystematikKategorie ist eine Sachgruppe der Systematik.
type SystematikKategorie struct {
	ID, Kuerzel, Bezeichnung string
}

// LadeSystematikKategorien liefert alle Sachgruppen nach ihrer Bezeichnung geordnet.
func LadeSystematikKategorien(ctx context.Context, db Datenbank) ([]SystematikKategorie, error) {
	rows, err := db.Query(ctx, "SELECT id, kuerzel, bezeichnung FROM systematik_kategorien ORDER BY bezeichnung COLLATE deutsch ASC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var kategorien []SystematikKategorie
	for rows.Next() {
		var k SystematikKategorie
		if err := rows.Scan(&k.ID, &k.Kuerzel, &k.Bezeichnung); err != nil {
			return nil, zeileUnlesbar(err)
		}
		kategorien = append(kategorien, k)
	}
	return kategorien, rows.Err()
}

// LadeFaecher liefert die Fächer, die an Titeln eingetragen sind, je einmal und ohne Leerraum
// am Rand: die Auswahl für eine Inventur nach Fach.
func LadeFaecher(ctx context.Context, db Datenbank) ([]string, error) {
	rows, err := db.Query(ctx, `
			SELECT btrim(subject)
			FROM buecher_titel
			WHERE subject IS NOT NULL AND btrim(subject) <> ''
			GROUP BY btrim(subject)
			ORDER BY btrim(subject) COLLATE deutsch
		`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	faecher := []string{}
	for rows.Next() {
		var f string
		if err := rows.Scan(&f); err != nil {
			return nil, zeileUnlesbar(err)
		}
		faecher = append(faecher, f)
	}
	return faecher, rows.Err()
}
