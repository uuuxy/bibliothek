package api

import (
	"context"
	"fmt"

	"bibliothek/repository"
)

// handleStudentAction loads student details by scanning card barcodes.
func (s *Server) handleStudentAction(ctx context.Context, query string, repo repository.StudentRepository, resp *ActionResponse) error {
	student, err := repo.GetByBarcode(ctx, query)
	if err != nil {
		return err
	}
	if student == nil {
		return fmt.Errorf("%w: Schüler-Barcode %s ist nicht registriert", errNotFound, query)
	}
	resp.Type = "student"
	resp.Student = student
	return nil
}
