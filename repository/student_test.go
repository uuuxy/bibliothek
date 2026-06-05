package repository

import (
	"bibliothek/db"
	"testing"
)

func TestNewStudentRepository(t *testing.T) {
	tests := []struct {
		name string
		db   db.PgxPoolIface
		want bool // we check if it returned a non-nil StudentRepository
	}{
		{
			name: "Constructor returns non-nil repository with nil db",
			db:   nil,
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := NewStudentRepository(tt.db)
			if (repo != nil) != tt.want {
				t.Errorf("NewStudentRepository() returned %v, want non-nil: %v", repo, tt.want)
			}

			// Additional check: cast back to underlying type and verify the db matches
			if pgRepo, ok := repo.(*pgStudentRepository); ok {
				if pgRepo.db != tt.db {
					t.Errorf("Expected db to be %v, got %v", tt.db, pgRepo.db)
				}
			} else {
				t.Errorf("Expected repository to be of type *pgStudentRepository")
			}
		})
	}
}
