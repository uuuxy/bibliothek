package repository

import (
	"context"
	"fmt"
	"time"
)

// QueryUeberfaelligeNachKlasse ermittelt alle Ausleihen, deren Frist überschritten ist,
// gruppiert nach Klasse und Schüler. Ein optionaler Filter schränkt die Abfrage auf eine Klasse ein.
func (repo *MahnwesenRepository) QueryUeberfaelligeNachKlasse(ctx context.Context, klasseFilter string) ([]MahnwesenKlasse, error) {
	q := `
		SELECT a.id, s.id, s.vorname || ' ' || s.nachname, s.klasse,
		       t.titel, coalesce(t.autor,''), coalesce(t.isbn,''), coalesce(t.cover_url,''),
		       a.rueckgabe_frist,
		       GREATEST(0, EXTRACT(DAY FROM (CURRENT_TIMESTAMP - a.rueckgabe_frist))::int) AS tage_ueberfaellig
		FROM ausleihen a
		JOIN buecher_exemplare e ON a.exemplar_id = e.id
		JOIN buecher_titel t    ON e.titel_id = t.id
		JOIN schueler s         ON a.schueler_id = s.id
		WHERE a.rueckgabe_am IS NULL
		  AND a.rueckgabe_frist < CURRENT_TIMESTAMP
	`
	args := []any{}
	if klasseFilter != "" {
		q += " AND s.klasse = $1"
		args = append(args, klasseFilter)
	}
	q += " ORDER BY s.klasse, s.nachname, s.vorname, a.rueckgabe_frist"

	rows, err := repo.db.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	klassenMap := map[string]*MahnwesenKlasse{}
	schuelerMap := map[string]*UeberfaelligerSchueler{}
	klassen := make([]MahnwesenKlasse, 0)

	for rows.Next() {
		var ausleiheID, schuelerID, name, klasse string
		var titel, autor, isbn, coverURL string
		var frist time.Time
		var tage int
		if err := rows.Scan(&ausleiheID, &schuelerID, &name, &klasse,
			&titel, &autor, &isbn, &coverURL,
			&frist, &tage); err != nil {
			continue
		}

		if _, ok := klassenMap[klasse]; !ok {
			klassen = append(klassen, MahnwesenKlasse{Klasse: klasse})
			klassenMap[klasse] = &klassen[len(klassen)-1]
		}

		schuelerKey := klasse + "|" + schuelerID
		if _, ok := schuelerMap[schuelerKey]; !ok {
			sch := UeberfaelligerSchueler{
				SchuelerID: schuelerID,
				Name:       name,
				Klasse:     klasse,
			}
			k := klassenMap[klasse]
			k.Schueler = append(k.Schueler, sch)
			schuelerMap[schuelerKey] = &k.Schueler[len(k.Schueler)-1]
		}

		schuelerMap[schuelerKey].Medien = append(schuelerMap[schuelerKey].Medien, UeberfaelligesMedium{
			AusleiheID:       ausleiheID,
			Titel:            titel,
			Autor:            autor,
			ISBN:             isbn,
			CoverURL:         coverURL,
			FaelligAm:        frist.Format("02.01.2006"),
			TageUeberfaellig: tage,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	// Falls Klassen existieren, ordnen wir ihnen die Lehrer-E-Mails aus dem Mapping zu
	if len(klassen) > 0 {
		mRows, err := repo.db.Query(ctx, `SELECT klasse, lehrer_email FROM klassen_lehrer_mapping`)
		if err == nil {
			defer mRows.Close()
			emailMap := map[string]string{}
			for mRows.Next() {
				var k, e string
				if err := mRows.Scan(&k, &e); err == nil {
					emailMap[k] = e
				}
			}
			for i := range klassen {
				klassen[i].LehrerEmail = emailMap[klassen[i].Klasse]
			}
		}
	}

	return klassen, nil
}

// QueryUeberfaelligeNachJahrgang ermittelt Bücher, die über die Jahrgangsstufe hinaus ausgeliehen wurden
// (z. B. wenn ein Buch nur bis Klasse 6 gedacht ist, der Schüler nun aber in Klasse 7 ist) oder die
// von Schülern behalten wurden, die die Schule bereits verlassen haben (Abgänger).
func (repo *MahnwesenRepository) QueryUeberfaelligeNachJahrgang(ctx context.Context, klasseFilter string) ([]MahnwesenKlasse, error) {
	q := `
		SELECT a.id, s.id, s.vorname || ' ' || s.nachname, s.klasse,
		       t.titel, coalesce(t.autor,''), coalesce(t.isbn,''), coalesce(t.cover_url,''),
		       a.ausgeliehen_am,
		       t.jahrgang_bis,
		       NULLIF(regexp_replace(s.klasse, '\D', '', 'g'), '')::int AS schueler_jahrgang,
			   s.ist_abgaenger
		FROM ausleihen a
		JOIN buecher_exemplare e ON a.exemplar_id = e.id
		JOIN buecher_titel t    ON e.titel_id = t.id
		JOIN schueler s         ON a.schueler_id = s.id
		WHERE a.rueckgabe_am IS NULL
		  AND (
		      (NULLIF(regexp_replace(s.klasse, '\D', '', 'g'), '')::int > t.jahrgang_bis)
		      OR s.ist_abgaenger = true
		  )
	`
	args := []any{}
	if klasseFilter != "" {
		q += " AND s.klasse = $1"
		args = append(args, klasseFilter)
	}
	q += " ORDER BY s.klasse, s.nachname, s.vorname, t.titel"

	rows, err := repo.db.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	klassenMap := map[string]*MahnwesenKlasse{}
	schuelerMap := map[string]*UeberfaelligerSchueler{}
	klassen := make([]MahnwesenKlasse, 0)

	for rows.Next() {
		var ausleiheID, schuelerID, name, klasse string
		var titel, autor, isbn, coverURL string
		var ausgeliehenAm time.Time
		var jahrgangBis int
		var schuelerJahrgang *int
		var istAbgaenger bool

		if err := rows.Scan(&ausleiheID, &schuelerID, &name, &klasse,
			&titel, &autor, &isbn, &coverURL,
			&ausgeliehenAm, &jahrgangBis, &schuelerJahrgang, &istAbgaenger); err != nil {
			continue
		}

		if _, ok := klassenMap[klasse]; !ok {
			klassen = append(klassen, MahnwesenKlasse{Klasse: klasse})
			klassenMap[klasse] = &klassen[len(klassen)-1]
		}

		schuelerKey := klasse + "|" + schuelerID
		if _, ok := schuelerMap[schuelerKey]; !ok {
			sch := UeberfaelligerSchueler{
				SchuelerID: schuelerID,
				Name:       name,
				Klasse:     klasse,
			}
			k := klassenMap[klasse]
			k.Schueler = append(k.Schueler, sch)
			schuelerMap[schuelerKey] = &k.Schueler[len(k.Schueler)-1]
		}

		ueberschreitung := 0
		if schuelerJahrgang != nil {
			ueberschreitung = *schuelerJahrgang - jahrgangBis
		}

		schuelerMap[schuelerKey].Medien = append(schuelerMap[schuelerKey].Medien, UeberfaelligesMedium{
			AusleiheID:       ausleiheID,
			Titel:            titel,
			Autor:            autor,
			ISBN:             isbn,
			CoverURL:         coverURL,
			FaelligAm:        fmt.Sprintf("bis Kl. %d", jahrgangBis),
			TageUeberfaellig: ueberschreitung,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return klassen, nil
}

// QueryUeberfaelligeByAusleiheIDs ermittelt spezifische überfällige Ausleihen für den Bulk-Print.
func (repo *MahnwesenRepository) QueryUeberfaelligeByAusleiheIDs(ctx context.Context, ids []string) ([]MahnwesenKlasse, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	
	q := `
		SELECT a.id, s.id, s.vorname || ' ' || s.nachname, s.klasse,
		       t.titel, coalesce(t.autor,''), coalesce(t.isbn,''), coalesce(t.cover_url,''),
		       a.rueckgabe_frist,
		       GREATEST(0, EXTRACT(DAY FROM (CURRENT_TIMESTAMP - a.rueckgabe_frist))::int) AS tage_ueberfaellig
		FROM ausleihen a
		JOIN buecher_exemplare e ON a.exemplar_id = e.id
		JOIN buecher_titel t    ON e.titel_id = t.id
		JOIN schueler s         ON a.schueler_id = s.id
		WHERE a.id = ANY($1)
		ORDER BY s.klasse, s.nachname, s.vorname, a.rueckgabe_frist
	`

	rows, err := repo.db.Query(ctx, q, ids)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	klassenMap := map[string]*MahnwesenKlasse{}
	schuelerMap := map[string]*UeberfaelligerSchueler{}
	klassen := make([]MahnwesenKlasse, 0)

	for rows.Next() {
		var ausleiheID, schuelerID, name, klasse string
		var titel, autor, isbn, coverURL string
		var frist time.Time
		var tage int
		if err := rows.Scan(&ausleiheID, &schuelerID, &name, &klasse,
			&titel, &autor, &isbn, &coverURL,
			&frist, &tage); err != nil {
			continue
		}

		if _, ok := klassenMap[klasse]; !ok {
			klassen = append(klassen, MahnwesenKlasse{Klasse: klasse})
			klassenMap[klasse] = &klassen[len(klassen)-1]
		}

		schuelerKey := klasse + "|" + schuelerID
		if _, ok := schuelerMap[schuelerKey]; !ok {
			sch := UeberfaelligerSchueler{
				SchuelerID: schuelerID,
				Name:       name,
				Klasse:     klasse,
			}
			k := klassenMap[klasse]
			k.Schueler = append(k.Schueler, sch)
			schuelerMap[schuelerKey] = &k.Schueler[len(k.Schueler)-1]
		}

		schuelerMap[schuelerKey].Medien = append(schuelerMap[schuelerKey].Medien, UeberfaelligesMedium{
			AusleiheID:       ausleiheID,
			Titel:            titel,
			Autor:            autor,
			ISBN:             isbn,
			CoverURL:         coverURL,
			FaelligAm:        frist.Format("02.01.2006"),
			TageUeberfaellig: tage,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return klassen, nil
}
