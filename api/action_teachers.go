package api

import (
	"context"
	"errors"
	"fmt"

	"bibliothek/repository"

	"github.com/jackc/pgx/v5"
)

// handleTeacherAction loads teacher details by scanning card barcodes.
func (s *Server) handleTeacherAction(ctx context.Context, query string, resp *ActionResponse) error {
	q := `
		SELECT b.id, b.barcode_id, b.vorname, b.nachname, br.rolle 
		FROM benutzer b 
		JOIN benutzer_rollen br ON b.id = br.benutzer_id
		WHERE b.barcode_id = $1 AND br.rolle = 'LEHRER' AND b.aktiv = true
		LIMIT 1
	`
	var u repository.User
	err := s.DB.Pool.QueryRow(ctx, q, query).Scan(&u.ID, &u.BarcodeID, &u.Vorname, &u.Nachname, &u.Rolle)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return fmt.Errorf("%w: Lehrer-Barcode %s ist nicht registriert oder inaktiv", errNotFound, query)
		}
		return err
	}
	resp.Type = "teacher"
	resp.Teacher = &u
	return nil
}
