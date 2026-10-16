package inventur

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/pashagolub/pgxmock/v5"
)

var errTest = errors.New("test error")

func TestAendereKlassensaetze(t *testing.T) {
	faelle := []struct {
		name          string
		oldClassName  string
		newClassNames []string
		bookIDs       []string
		mockSetup     func(mock pgxmock.PgxPoolIface)
		wantErr       bool
		errMsg        string
	}{
		{
			name:          "success with both old and new class names",
			oldClassName:  "5A",
			newClassNames: []string{"6A", "6B"},
			bookIDs:       []string{"b1", "b2"},
			mockSetup: func(mock pgxmock.PgxPoolIface) {
				mock.ExpectBegin()
				mock.ExpectExec(`^DELETE FROM class_books WHERE class_name = \$1$`).
					WithArgs("5A").
					WillReturnResult(pgxmock.NewResult("DELETE", 2))
				mock.ExpectExec(`^DELETE FROM class_books WHERE class_name = ANY\(\$1\)$`).
					WithArgs([]string{"6A", "6B"}).
					WillReturnResult(pgxmock.NewResult("DELETE", 0))
				mock.ExpectExec(`INSERT INTO class_books`).
					WithArgs([]string{"6A", "6A", "6B", "6B"}, []string{"b1", "b2", "b1", "b2"}).
					WillReturnResult(pgxmock.NewResult("INSERT", 4))
				mock.ExpectCommit()
			},
			wantErr: false,
		},
		{
			name:          "success with empty old class name",
			oldClassName:  "",
			newClassNames: []string{"6A"},
			bookIDs:       []string{"b1"},
			mockSetup: func(mock pgxmock.PgxPoolIface) {
				mock.ExpectBegin()
				mock.ExpectExec(`^DELETE FROM class_books WHERE class_name = ANY\(\$1\)$`).
					WithArgs([]string{"6A"}).
					WillReturnResult(pgxmock.NewResult("DELETE", 0))
				mock.ExpectExec(`INSERT INTO class_books`).
					WithArgs([]string{"6A"}, []string{"b1"}).
					WillReturnResult(pgxmock.NewResult("INSERT", 1))
				mock.ExpectCommit()
			},
			wantErr: false,
		},
		{
			name:          "success with empty new class names",
			oldClassName:  "5A",
			newClassNames: []string{},
			bookIDs:       []string{},
			mockSetup: func(mock pgxmock.PgxPoolIface) {
				mock.ExpectBegin()
				mock.ExpectExec(`^DELETE FROM class_books WHERE class_name = \$1$`).
					WithArgs("5A").
					WillReturnResult(pgxmock.NewResult("DELETE", 2))
				mock.ExpectCommit()
			},
			wantErr: false,
		},
		{
			name:          "error starting transaction",
			oldClassName:  "5A",
			newClassNames: []string{"6A"},
			bookIDs:       []string{"b1"},
			mockSetup: func(mock pgxmock.PgxPoolIface) {
				mock.ExpectBegin().WillReturnError(errTest)
			},
			wantErr: true,
			errMsg:  "transaktion konnte nicht gestartet werden: test error",
		},
		{
			name:          "error deleting old bindings",
			oldClassName:  "5A",
			newClassNames: []string{"6A"},
			bookIDs:       []string{"b1"},
			mockSetup: func(mock pgxmock.PgxPoolIface) {
				mock.ExpectBegin()
				mock.ExpectExec(`^DELETE FROM class_books WHERE class_name = \$1$`).
					WithArgs("5A").
					WillReturnError(errTest)
				mock.ExpectRollback()
			},
			wantErr: true,
			errMsg:  "alte zuweisungen konnten nicht gelöscht werden: test error",
		},
		{
			name:          "error deleting existing new bindings",
			oldClassName:  "5A",
			newClassNames: []string{"6A"},
			bookIDs:       []string{"b1"},
			mockSetup: func(mock pgxmock.PgxPoolIface) {
				mock.ExpectBegin()
				mock.ExpectExec(`^DELETE FROM class_books WHERE class_name = \$1$`).
					WithArgs("5A").
					WillReturnResult(pgxmock.NewResult("DELETE", 2))
				mock.ExpectExec(`^DELETE FROM class_books WHERE class_name = ANY\(\$1\)$`).
					WithArgs([]string{"6A"}).
					WillReturnError(errTest)
				mock.ExpectRollback()
			},
			wantErr: true,
			errMsg:  "vorhandene zuweisungen des neuen namens konnten nicht gelöscht werden: test error",
		},
		{
			name:          "error inserting new bindings",
			oldClassName:  "5A",
			newClassNames: []string{"6A"},
			bookIDs:       []string{"b1"},
			mockSetup: func(mock pgxmock.PgxPoolIface) {
				mock.ExpectBegin()
				mock.ExpectExec(`^DELETE FROM class_books WHERE class_name = \$1$`).
					WithArgs("5A").
					WillReturnResult(pgxmock.NewResult("DELETE", 2))
				mock.ExpectExec(`^DELETE FROM class_books WHERE class_name = ANY\(\$1\)$`).
					WithArgs([]string{"6A"}).
					WillReturnResult(pgxmock.NewResult("DELETE", 0))
				mock.ExpectExec(`INSERT INTO class_books`).
					WithArgs([]string{"6A"}, []string{"b1"}).
					WillReturnError(errTest)
				mock.ExpectRollback()
			},
			wantErr: true,
			errMsg:  "neue zuweisung konnte nicht gespeichert werden: test error",
		},
		{
			name:          "error committing transaction",
			oldClassName:  "5A",
			newClassNames: []string{"6A"},
			bookIDs:       []string{"b1"},
			mockSetup: func(mock pgxmock.PgxPoolIface) {
				mock.ExpectBegin()
				mock.ExpectExec(`^DELETE FROM class_books WHERE class_name = \$1$`).
					WithArgs("5A").
					WillReturnResult(pgxmock.NewResult("DELETE", 2))
				mock.ExpectExec(`^DELETE FROM class_books WHERE class_name = ANY\(\$1\)$`).
					WithArgs([]string{"6A"}).
					WillReturnResult(pgxmock.NewResult("DELETE", 0))
				mock.ExpectExec(`INSERT INTO class_books`).
					WithArgs([]string{"6A"}, []string{"b1"}).
					WillReturnResult(pgxmock.NewResult("INSERT", 1))
				mock.ExpectCommit().WillReturnError(errTest)
				mock.ExpectRollback()
			},
			wantErr: true,
			errMsg:  "transaktion konnte nicht abgeschlossen werden: test error",
		},
	}

	for _, f := range faelle {
		t.Run(f.name, func(t *testing.T) {
			mock, err := pgxmock.NewPool()
			if err != nil {
				t.Fatalf("failed to create pgxmock: %v", err)
			}
			defer mock.Close()

			abfragen := NewTitelAbfragen(mock)
			f.mockSetup(mock)

			err = abfragen.aendereKlassensaetze(context.Background(), f.oldClassName, f.newClassNames, f.bookIDs)
			if f.wantErr {
				if err == nil {
					t.Errorf("expected error, got nil")
				} else if err.Error() != f.errMsg {
					t.Errorf("expected error message %q, got %q", f.errMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("unmet mock expectations: %v", err)
			}
		})
	}
}

func TestLoescheKlassensatzGruppe(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("failed to create pgxmock: %v", err)
	}
	defer mock.Close()

	abfragen := NewTitelAbfragen(mock)

	faelle := []struct {
		name       string
		classNames []string
		mockSetup  func()
		wantErr    bool
	}{
		{
			name:       "success single",
			classNames: []string{"5A"},
			mockSetup: func() {
				mock.ExpectExec(`^DELETE FROM class_books WHERE class_name = ANY\(\$1\)$`).
					WithArgs([]string{"5A"}).
					WillReturnResult(pgxmock.NewResult("DELETE", 1))
			},
			wantErr: false,
		},
		{
			name:       "success multiple",
			classNames: []string{"5A", "5B"},
			mockSetup: func() {
				mock.ExpectExec(`^DELETE FROM class_books WHERE class_name = ANY\(\$1\)$`).
					WithArgs([]string{"5A", "5B"}).
					WillReturnResult(pgxmock.NewResult("DELETE", 2))
			},
			wantErr: false,
		},
		{
			name:       "success empty",
			classNames: []string{},
			mockSetup: func() {
				// No DB call expected
			},
			wantErr: false,
		},
		{
			name:       "db error",
			classNames: []string{"10B"},
			mockSetup: func() {
				mock.ExpectExec(`^DELETE FROM class_books WHERE class_name = ANY\(\$1\)$`).
					WithArgs([]string{"10B"}).
					WillReturnError(fmt.Errorf("db error"))
			},
			wantErr: true,
		},
	}

	for _, f := range faelle {
		t.Run(f.name, func(t *testing.T) {
			f.mockSetup()

			err := abfragen.loescheKlassensatzGruppe(context.Background(), f.classNames)
			if (err != nil) != f.wantErr {
				t.Errorf("loescheKlassensatzGruppe() error = %v, wantErr %v", err, f.wantErr)
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("unmet mock expectations: %v", err)
			}
		})
	}
}

func TestLadeKlassensatzGruppen(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("failed to create pgxmock: %v", err)
	}
	defer mock.Close()

	abfragen := NewTitelAbfragen(mock)
	ctx := context.Background()

	faelle := []struct {
		name       string
		branch     string
		sortOrder  string
		mockSetup  func()
		wantErr    bool
		wantGroups int
	}{
		{
			name:      "success default (asc, empty branch)",
			branch:    "",
			sortOrder: "",
			mockSetup: func() {
				zeilen := pgxmock.NewRows([]string{
					"class_name", "id", "title", "subject", "track", "cover_url", "isbn", "verfuegbar", "gesamt", "im_zulauf", "quelle", "leser", "auflagen",
				}).
					AddRow("5A", "550e8400-e29b-41d4-a716-446655440000", "Buch 1", "Mathe", "G", "url1", "123", 5, 10, 0, "hand", 0, nil).
					AddRow("5A", "550e8400-e29b-41d4-a716-446655440001", "Buch 2", "Deutsch", "G", "url2", "456", 2, 5, 0, "hand", 0, nil).
					AddRow("6B", "550e8400-e29b-41d4-a716-446655440002", "Buch 1", "Mathe", "R", "url1", "123", 4, 8, 0, "hand", 0, nil)

				mock.ExpectQuery(`(?s)SELECT.*cb\.class_name, b\.id, b\.titel AS title.*FROM zuordnung cb.*ORDER BY.*CASE.*ASC, b\.titel ASC`).
					WithArgs("", klassensatzMindestLeser).
					WillReturnRows(zeilen)
			},
			wantErr:    false,
			wantGroups: 2, // 5A and 6B
		},
		{
			name:      "success branch and desc",
			branch:    "F",
			sortOrder: "desc",
			mockSetup: func() {
				zeilen := pgxmock.NewRows([]string{
					"class_name", "id", "title", "subject", "track", "cover_url", "isbn", "verfuegbar", "gesamt", "im_zulauf", "quelle", "leser", "auflagen",
				}).
					AddRow("10F", "550e8400-e29b-41d4-a716-446655440003", "Buch 3", "Englisch", "F", "url3", "789", 1, 1, 0, "hand", 0, nil)

				mock.ExpectQuery(`(?s)SELECT.*cb\.class_name, b\.id, b\.titel AS title.*FROM zuordnung cb.*ORDER BY.*CAST.*DESC, cb\.class_name DESC, b\.titel ASC`).
					WithArgs("F", klassensatzMindestLeser).
					WillReturnRows(zeilen)
			},
			wantErr:    false,
			wantGroups: 1, // 10F
		},
		{
			name:      "query error",
			branch:    "",
			sortOrder: "",
			mockSetup: func() {
				mock.ExpectQuery(`(?s)SELECT.*`).
					WithArgs("", klassensatzMindestLeser).
					WillReturnError(fmt.Errorf("db error"))
			},
			wantErr:    true,
			wantGroups: 0,
		},
		{
			name:      "scan error",
			branch:    "",
			sortOrder: "",
			mockSetup: func() {
				// Falscher Typ für gesamt (Text statt Zahl), damit der Scan scheitert.
				zeilen := pgxmock.NewRows([]string{
					"class_name", "id", "title", "subject", "track", "cover_url", "isbn", "verfuegbar", "gesamt", "im_zulauf", "quelle", "leser", "auflagen",
				}).
					AddRow("5A", "550e8400-e29b-41d4-a716-446655440000", "Buch 1", "Mathe", "G", "url1", "123", 5, "invalid", 0, "hand", 0, nil)

				mock.ExpectQuery(`(?s)SELECT.*`).
					WithArgs("", klassensatzMindestLeser).
					WillReturnRows(zeilen)
			},
			wantErr:    true,
			wantGroups: 0,
		},
		{
			name:      "rows error",
			branch:    "",
			sortOrder: "",
			mockSetup: func() {
				zeilen := pgxmock.NewRows([]string{
					"class_name", "id", "title", "subject", "track", "cover_url", "isbn", "verfuegbar", "gesamt", "im_zulauf", "quelle", "leser", "auflagen",
				}).
					AddRow("5A", "550e8400-e29b-41d4-a716-446655440000", "Buch 1", "Mathe", "G", "url1", "123", 5, 10, 0, "hand", 0, nil).
					RowError(0, fmt.Errorf("row error"))

				mock.ExpectQuery(`(?s)SELECT.*`).
					WithArgs("", klassensatzMindestLeser).
					WillReturnRows(zeilen)
			},
			wantErr:    true,
			wantGroups: 0,
		},
		{
			// Die Aufschlüsselung der Auflagen ist JSON aus der Abfrage; ist es nicht lesbar, ist das ein
			// Fehler und keine Kachel ohne Mischung.
			name:      "auflagen error",
			branch:    "",
			sortOrder: "",
			mockSetup: func() {
				zeilen := pgxmock.NewRows([]string{
					"class_name", "id", "title", "subject", "track", "cover_url", "isbn", "verfuegbar", "gesamt", "im_zulauf", "quelle", "leser", "auflagen",
				}).
					AddRow("5A", "550e8400-e29b-41d4-a716-446655440000", "Buch 1", "Mathe", "G", "url1", "123", 5, 10, 0, "ausleihe", 6, []byte(`{kaputt`))

				mock.ExpectQuery(`(?s)SELECT.*`).
					WithArgs("", klassensatzMindestLeser).
					WillReturnRows(zeilen)
			},
			wantErr:    true,
			wantGroups: 0,
		},
	}

	for _, f := range faelle {
		t.Run(f.name, func(t *testing.T) {
			f.mockSetup()

			gruppen, err := abfragen.ladeKlassensatzGruppen(ctx, f.branch, f.sortOrder)
			if (err != nil) != f.wantErr {
				t.Errorf("ladeKlassensatzGruppen() error = %v, wantErr %v", err, f.wantErr)
			}
			if len(gruppen) != f.wantGroups {
				t.Errorf("ladeKlassensatzGruppen() got %d groups, want %d", len(gruppen), f.wantGroups)
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("unmet mock expectations: %v", err)
			}
		})
	}
}

func TestOrdneTitelKlassenZu(t *testing.T) {
	faelle := []struct {
		name       string
		classNames []string
		bookIDs    []string
		mockSetup  func(mock pgxmock.PgxPoolIface)
		wantErr    bool
	}{
		{
			name:       "success single",
			classNames: []string{"5A"},
			bookIDs:    []string{"123e4567-e89b-12d3-a456-426614174000"},
			mockSetup: func(mock pgxmock.PgxPoolIface) {
				mock.ExpectExec(`^[\s\S]*INSERT INTO class_books[\s\S]*ON CONFLICT[\s\S]*DO NOTHING[\s\S]*$`).
					WithArgs([]string{"5A"}, []string{"123e4567-e89b-12d3-a456-426614174000"}).
					WillReturnResult(pgxmock.NewResult("INSERT", 1))
			},
			wantErr: false,
		},
		{
			name:       "success multiple",
			classNames: []string{"5A", "5B"},
			bookIDs:    []string{"123e4567-e89b-12d3-a456-426614174000", "98765432-e89b-12d3-a456-426614174000"},
			mockSetup: func(mock pgxmock.PgxPoolIface) {
				// 05A und 05B mit den zwei Titeln ergeben vier Zeilen.
				mock.ExpectExec(`^[\s\S]*INSERT INTO class_books[\s\S]*ON CONFLICT[\s\S]*DO NOTHING[\s\S]*$`).
					WithArgs(
						[]string{"5A", "5A", "5B", "5B"},
						[]string{
							"123e4567-e89b-12d3-a456-426614174000",
							"98765432-e89b-12d3-a456-426614174000",
							"123e4567-e89b-12d3-a456-426614174000",
							"98765432-e89b-12d3-a456-426614174000",
						},
					).
					WillReturnResult(pgxmock.NewResult("INSERT", 4))
			},
			wantErr: false,
		},
		{
			name:       "empty classNames",
			classNames: []string{},
			bookIDs:    []string{"123e4567-e89b-12d3-a456-426614174000"},
			mockSetup: func(mock pgxmock.PgxPoolIface) {
				// No DB call expected
			},
			wantErr: false,
		},
		{
			name:       "empty bookIDs",
			classNames: []string{"5A"},
			bookIDs:    []string{},
			mockSetup: func(mock pgxmock.PgxPoolIface) {
				// No DB call expected
			},
			wantErr: false,
		},
		{
			name:       "db error",
			classNames: []string{"5A"},
			bookIDs:    []string{"123e4567-e89b-12d3-a456-426614174000"},
			mockSetup: func(mock pgxmock.PgxPoolIface) {
				mock.ExpectExec(`^[\s\S]*INSERT INTO class_books[\s\S]*ON CONFLICT[\s\S]*DO NOTHING[\s\S]*$`).
					WithArgs([]string{"5A"}, []string{"123e4567-e89b-12d3-a456-426614174000"}).
					WillReturnError(fmt.Errorf("db error"))
			},
			wantErr: true,
		},
	}

	for _, f := range faelle {
		t.Run(f.name, func(t *testing.T) {
			mock, err := pgxmock.NewPool()
			if err != nil {
				t.Fatalf("failed to create pgxmock: %v", err)
			}
			defer mock.Close()

			abfragen := NewTitelAbfragen(mock)
			f.mockSetup(mock)

			err = abfragen.ordneTitelKlassenZu(context.Background(), f.classNames, f.bookIDs)
			if (err != nil) != f.wantErr {
				t.Errorf("ordneTitelKlassenZu() error = %v, wantErr %v", err, f.wantErr)
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("unmet mock expectations: %v", err)
			}
		})
	}
}
