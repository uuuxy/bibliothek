package repository

import "context"

// SystematikKategorie ist eine Sachgruppe der Systematik.
type SystematikKategorie struct {
	ID, Kuerzel, Bezeichnung string
}

// ListeSystematikKategorien liefert alle Sachgruppen nach ihrer Bezeichnung geordnet.
func ListeSystematikKategorien(ctx context.Context, db DBQueryer) ([]SystematikKategorie, error) {
	rows, err := db.Query(ctx, "SELECT id, kuerzel, bezeichnung FROM systematik_kategorien ORDER BY bezeichnung ASC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var kategorien []SystematikKategorie
	for rows.Next() {
		var k SystematikKategorie
		if err := rows.Scan(&k.ID, &k.Kuerzel, &k.Bezeichnung); err == nil {
			kategorien = append(kategorien, k)
		}
	}
	return kategorien, rows.Err()
}

// ListeFaecher liefert die Fächer, die an Titeln eingetragen sind, je einmal und ohne Leerraum
// am Rand: die Auswahl für eine Inventur nach Fach.
func ListeFaecher(ctx context.Context, db DBQueryer) ([]string, error) {
	rows, err := db.Query(ctx, `
			SELECT DISTINCT btrim(subject)
			FROM buecher_titel
			WHERE subject IS NOT NULL AND btrim(subject) <> ''
			ORDER BY 1
		`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	faecher := []string{}
	for rows.Next() {
		var f string
		if err := rows.Scan(&f); err == nil {
			faecher = append(faecher, f)
		}
	}
	return faecher, rows.Err()
}
