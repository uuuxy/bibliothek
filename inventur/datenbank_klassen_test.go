package inventur

import (
	"context"
	"fmt"
	"testing"

	"github.com/pashagolub/pgxmock/v4"
)

func TestDeleteClassGroup(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("failed to create pgxmock: %v", err)
	}
	defer mock.Close()

	repo := NewBookRepository(mock)

	tests := []struct {
		name      string
		className string
		mockSetup func()
		wantErr   bool
	}{
		{
			name:      "success",
			className: "5A",
			mockSetup: func() {
				mock.ExpectExec(`^DELETE FROM class_books WHERE class_name = \$1$`).
					WithArgs("5A").
					WillReturnResult(pgxmock.NewResult("DELETE", 1))
			},
			wantErr: false,
		},
		{
			name:      "db error",
			className: "10B",
			mockSetup: func() {
				mock.ExpectExec(`^DELETE FROM class_books WHERE class_name = \$1$`).
					WithArgs("10B").
					WillReturnError(fmt.Errorf("db error"))
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mockSetup()

			err := repo.DeleteClassGroup(context.Background(), tt.className)
			if (err != nil) != tt.wantErr {
				t.Errorf("DeleteClassGroup() error = %v, wantErr %v", err, tt.wantErr)
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("unmet mock expectations: %v", err)
			}
		})
	}
}
