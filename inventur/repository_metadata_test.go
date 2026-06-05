package inventur

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/pashagolub/pgxmock/v3"
)

func TestUpdateBookMetadata(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer mock.Close()

	repo := NewBookRepository(mock)
	ctx := context.Background()
	testID := "123e4567-e89b-12d3-a456-426614174000"

	queryRegex := `(?i)UPDATE buecher_titel\s+SET titel = COALESCE\(NULLIF\(\$1, ''\), titel\),\s+autor = COALESCE\(NULLIF\(\$2, ''\), autor\),\s+cover_url = COALESCE\(NULLIF\(\$3, ''\), cover_url\)\s+WHERE id = \$4::uuid`

	tests := []struct {
		name     string
		id       string
		title    string
		author   string
		coverURL string
		mockFunc func()
		wantErr  bool
		errIs    error
	}{
		{
			name:     "Success all fields",
			id:       testID,
			title:    "New Title",
			author:   "New Author",
			coverURL: "new.jpg",
			mockFunc: func() {
				mock.ExpectExec(queryRegex).
					WithArgs("New Title", "New Author", "new.jpg", testID).
					WillReturnResult(pgxmock.NewResult("UPDATE", 1))
			},
			wantErr: false,
		},
		{
			name:     "Success empty fields",
			id:       testID,
			title:    "",
			author:   "",
			coverURL: "new.jpg",
			mockFunc: func() {
				mock.ExpectExec(queryRegex).
					WithArgs("", "", "new.jpg", testID).
					WillReturnResult(pgxmock.NewResult("UPDATE", 1))
			},
			wantErr: false,
		},
		{
			name:     "Not found",
			id:       testID,
			title:    "Title",
			author:   "Author",
			coverURL: "cover.jpg",
			mockFunc: func() {
				mock.ExpectExec(queryRegex).
					WithArgs("Title", "Author", "cover.jpg", testID).
					WillReturnResult(pgxmock.NewResult("UPDATE", 0))
			},
			wantErr: true,
			errIs:   ErrBookNotFound,
		},
		{
			name:     "DB Error",
			id:       testID,
			title:    "Title",
			author:   "Author",
			coverURL: "cover.jpg",
			mockFunc: func() {
				mock.ExpectExec(queryRegex).
					WithArgs("Title", "Author", "cover.jpg", testID).
					WillReturnError(fmt.Errorf("db error"))
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.mockFunc != nil {
				tt.mockFunc()
			}

			err := repo.UpdateBookMetadata(ctx, tt.id, tt.title, tt.author, tt.coverURL)

			if (err != nil) != tt.wantErr {
				t.Errorf("UpdateBookMetadata() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr && tt.errIs != nil && !errors.Is(err, tt.errIs) {
				t.Errorf("UpdateBookMetadata() error = %v, wantErrIs %v", err, tt.errIs)
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("there were unfulfilled expectations: %s", err)
			}
		})
	}
}

func TestUpdateBookCategory(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer mock.Close()

	repo := NewBookRepository(mock)
	ctx := context.Background()
	testID := "123e4567-e89b-12d3-a456-426614174000"

	queryRegex := `(?i)UPDATE buecher_titel\s+SET subject = \$1,\s+grade_level = \$2\s+WHERE id = \$3::uuid`

	tests := []struct {
		name       string
		id         string
		subject    string
		gradeLevel int16
		mockFunc   func()
		wantErr    bool
		errIs      error
	}{
		{
			name:       "Success",
			id:         testID,
			subject:    "Math",
			gradeLevel: 10,
			mockFunc: func() {
				mock.ExpectExec(queryRegex).
					WithArgs("Math", int16(10), testID).
					WillReturnResult(pgxmock.NewResult("UPDATE", 1))
			},
			wantErr: false,
		},
		{
			name:       "Not found",
			id:         testID,
			subject:    "Math",
			gradeLevel: 10,
			mockFunc: func() {
				mock.ExpectExec(queryRegex).
					WithArgs("Math", int16(10), testID).
					WillReturnResult(pgxmock.NewResult("UPDATE", 0))
			},
			wantErr: true,
			errIs:   ErrBookNotFound,
		},
		{
			name:       "DB Error",
			id:         testID,
			subject:    "Math",
			gradeLevel: 10,
			mockFunc: func() {
				mock.ExpectExec(queryRegex).
					WithArgs("Math", int16(10), testID).
					WillReturnError(fmt.Errorf("db error"))
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.mockFunc != nil {
				tt.mockFunc()
			}

			err := repo.UpdateBookCategory(ctx, tt.id, tt.subject, tt.gradeLevel)

			if (err != nil) != tt.wantErr {
				t.Errorf("UpdateBookCategory() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr && tt.errIs != nil && !errors.Is(err, tt.errIs) {
				t.Errorf("UpdateBookCategory() error = %v, wantErrIs %v", err, tt.errIs)
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("there were unfulfilled expectations: %s", err)
			}
		})
	}
}

func TestGetBookByID(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer mock.Close()

	repo := NewBookRepository(mock)
	ctx := context.Background()
	testID := "123e4567-e89b-12d3-a456-426614174000"

	queryRegex := `(?i)SELECT id, COALESCE\(isbn, ''\) AS isbn, titel AS title, COALESCE\(autor, ''\) AS author, COALESCE\(cover_url, ''\) AS cover_url, COALESCE\(subject, ''\) AS subject, COALESCE\(grade_level, 0\) AS grade_level, COALESCE\(track, ''\) AS track, stock, TO_CHAR\(last_counted, 'YYYY-MM-DD'\) as last_counted, sort_order, COALESCE\(medientyp, 'Buch'\) AS medientyp, erweiterte_eigenschaften\s+FROM buecher_titel\s+WHERE id = \$1::uuid`

	columns := []string{"id", "isbn", "title", "author", "cover_url", "subject", "grade_level", "track", "stock", "last_counted", "sort_order", "medientyp", "erweiterte_eigenschaften"}
	lastCountedStr := "2023-01-01"

	tests := []struct {
		name     string
		id       string
		mockFunc func()
		wantErr  bool
	}{
		{
			name: "Success",
			id:   testID,
			mockFunc: func() {
				mock.ExpectQuery(queryRegex).
					WithArgs(testID).
					WillReturnRows(pgxmock.NewRows(columns).AddRow(
						testID, "1234567890", "Test Title", "Test Author", "cover.jpg", "Math", int16(10), "A", 5, &lastCountedStr, 1, "Buch", nil,
					))
			},
			wantErr: false,
		},
		{
			name: "Not found",
			id:   testID,
			mockFunc: func() {
				mock.ExpectQuery(queryRegex).
					WithArgs(testID).
					WillReturnError(fmt.Errorf("no rows in result set"))
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.mockFunc != nil {
				tt.mockFunc()
			}

			book, err := repo.GetBookByID(ctx, tt.id)

			if (err != nil) != tt.wantErr {
				t.Errorf("GetBookByID() error = %v, wantErr %v", err, tt.wantErr)
			}

			if !tt.wantErr && book == nil {
				t.Errorf("GetBookByID() expected book to be returned, got nil")
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("there were unfulfilled expectations: %s", err)
			}
		})
	}
}
