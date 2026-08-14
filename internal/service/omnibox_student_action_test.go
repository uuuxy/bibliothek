package service

import (
	"context"
	"errors"
	"testing"

	"bibliothek/repository"
)

func TestHandleStudentAction(t *testing.T) {
	tests := []struct {
		name        string
		query       string
		mockStudent *repository.Student
		mockErr     error
		wantErr     bool
		errMatch    error
	}{
		{
			name:        "Happy Path - Student found",
			query:       "S-12345",
			mockStudent: &repository.Student{ID: "student-1", BarcodeID: "S-12345", Vorname: "Max", Nachname: "Mustermann"},
			mockErr:     nil,
			wantErr:     false,
		},
		{
			name:        "Not Found - Student nil",
			query:       "S-99999",
			mockStudent: nil,
			mockErr:     nil,
			wantErr:     true,
			errMatch:    ErrNotFound,
		},
		{
			name:        "Error - DB Error",
			query:       "S-12345",
			mockStudent: nil,
			mockErr:     errors.New("db connection failed"),
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			studentRepo := &routingStudentRepo{
				students: map[string]*repository.Student{},
			}

			if tt.mockStudent != nil {
				studentRepo.students[tt.query] = tt.mockStudent
			}

			svc := &defaultOmniboxService{
				studentRepo: studentRepo,
			}

			// Override GetByBarcode for the DB error case
			if tt.mockErr != nil {
				svc.studentRepo = &errorStudentRepo{err: tt.mockErr}
			}

			resp := &OmniboxResult{}
			err := svc.handleStudentAction(context.Background(), tt.query, resp)

			if (err != nil) != tt.wantErr {
				t.Errorf("handleStudentAction() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr && tt.errMatch != nil && !errors.Is(err, tt.errMatch) {
				t.Errorf("handleStudentAction() error = %v, expected to match %v", err, tt.errMatch)
			}

			if !tt.wantErr {
				if resp.Type != "student" {
					t.Errorf("handleStudentAction() resp.Type = %v, want %v", resp.Type, "student")
				}
				if resp.Student != tt.mockStudent {
					t.Errorf("handleStudentAction() resp.Student = %v, want %v", resp.Student, tt.mockStudent)
				}
			}
		})
	}
}

type errorStudentRepo struct {
	repository.StudentRepository
	err error
}

func (r *errorStudentRepo) GetByBarcode(_ context.Context, barcode string) (*repository.Student, error) {
	return nil, r.err
}
