package repository

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
)

// HasPhoto checks if an encrypted photo exists for the student.
func (repo *pgStudentRepository) HasPhoto(ctx context.Context, studentID string) (bool, error) {
	var hasPhoto bool
	err := repo.db.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM schueler_fotos WHERE schueler_id = $1)", studentID).Scan(&hasPhoto)
	return hasPhoto, err
}

// HasOpenDamages checks if the student has any unpaid damage fees.
func (repo *pgStudentRepository) HasOpenDamages(ctx context.Context, studentID string) (bool, error) {
	var hasOpenDamages bool
	err := repo.db.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM schadensfaelle WHERE schueler_id = $1 AND ist_bezahlt = false)", studentID).Scan(&hasOpenDamages)
	return hasOpenDamages, err
}

// GetActiveBorrowedBooks retrieves all books currently borrowed by the student.
func (repo *pgStudentRepository) GetActiveBorrowedBooks(ctx context.Context, studentID string) ([]BorrowedBook, error) {
	query := `
		SELECT 
			e.id, 
			a.id AS ausleihe_id,
			e.barcode_id, 
			t.titel, 
			coalesce(t.autor, ''), 
			coalesce(t.cover_url, ''),
			a.ausgeliehen_am, 
			a.rueckgabe_frist
		FROM ausleihen a
		JOIN buecher_exemplare e ON a.exemplar_id = e.id
		JOIN buecher_titel t ON e.titel_id = t.id
		WHERE a.schueler_id = $1 AND a.rueckgabe_am IS NULL
		ORDER BY a.ausgeliehen_am DESC
	`
	rows, err := repo.db.Query(ctx, query, studentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var books []BorrowedBook
	for rows.Next() {
		var b BorrowedBook
		if err := rows.Scan(
			&b.ID,
			&b.AusleiheID,
			&b.BarcodeID,
			&b.Titel,
			&b.Autor,
			&b.CoverURL,
			&b.AusgeliehenAm,
			&b.RueckgabeFrist,
		); err != nil {
			return nil, err
		}
		books = append(books, b)
	}
	return books, rows.Err()
}

// GetDistinctClasses returns a list of all active classes.
func (repo *pgStudentRepository) GetDistinctClasses(ctx context.Context) ([]string, error) {
	rows, err := repo.db.Query(ctx, "SELECT DISTINCT klasse FROM schueler WHERE klasse != '' AND deleted_at IS NULL ORDER BY klasse")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var classes []string
	for rows.Next() {
		var k string
		if err := rows.Scan(&k); err == nil {
			classes = append(classes, k)
		}
	}
	return classes, rows.Err()
}

// ListStudentsWithStats returns a list of students with loan statistics.
func (repo *pgStudentRepository) ListStudentsWithStats(ctx context.Context, klasse string) ([]StudentListStat, error) {
	var rows pgx.Rows
	var err error

	if klasse != "" {
		rows, err = repo.db.Query(ctx, `
			SELECT id, barcode_id, vorname, nachname, klasse, abgaenger_jahr, ist_gesperrt,
				(SELECT COUNT(*) FROM ausleihen a WHERE a.schueler_id = schueler.id AND a.rueckgabe_am IS NULL) as ausgeliehen_anzahl,
				(SELECT COUNT(*) FROM ausleihen a WHERE a.schueler_id = schueler.id AND a.rueckgabe_am IS NULL AND a.rueckgabe_frist < CURRENT_TIMESTAMP) as ueberfaellig_anzahl,
				EXISTS(SELECT 1 FROM schueler_fotos sf WHERE sf.schueler_id = schueler.id) as has_foto
			FROM schueler 
			WHERE klasse = $1 AND deleted_at IS NULL
			ORDER BY nachname, vorname
		`, klasse)
	} else {
		rows, err = repo.db.Query(ctx, `
			SELECT id, barcode_id, vorname, nachname, klasse, abgaenger_jahr, ist_gesperrt,
				(SELECT COUNT(*) FROM ausleihen a WHERE a.schueler_id = schueler.id AND a.rueckgabe_am IS NULL) as ausgeliehen_anzahl,
				(SELECT COUNT(*) FROM ausleihen a WHERE a.schueler_id = schueler.id AND a.rueckgabe_am IS NULL AND a.rueckgabe_frist < CURRENT_TIMESTAMP) as ueberfaellig_anzahl,
				EXISTS(SELECT 1 FROM schueler_fotos sf WHERE sf.schueler_id = schueler.id) as has_foto
			FROM schueler 
			WHERE deleted_at IS NULL
			ORDER BY klasse, nachname, vorname 
			LIMIT 500
		`)
	}

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var stats []StudentListStat
	for rows.Next() {
		var s StudentListStat
		if err := rows.Scan(&s.ID, &s.BarcodeID, &s.Vorname, &s.Nachname, &s.Klasse, &s.AbgaengerJahr, &s.IstGesperrt, &s.AusgeliehenCount, &s.UeberfaelligCount, &s.HasFoto); err != nil {
			return nil, err
		}
		if s.BarcodeID != "" && s.HasFoto {
			s.FotoURL = fmt.Sprintf("/api/schueler/%s/photo", s.BarcodeID)
		}
		stats = append(stats, s)
	}
	return stats, rows.Err()
}
