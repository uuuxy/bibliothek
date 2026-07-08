package inventur

import (
	"context"
	"fmt"
	"testing"
	"reflect"

	"github.com/pashagolub/pgxmock/v4"
)

func TestGetClassGroups(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("failed to create pgxmock: %v", err)
	}
	defer mock.Close()

	repo := NewBookRepository(mock)

	columns := []string{
		"class_name", "id", "title", "subject", "track", "cover_url", "isbn", "verfuegbar", "gesamt",
	}

	tests := []struct {
		name      string
		branch    string
		sortOrder string
		mockFn    func()
		want      []ClassGroup
		wantErr   bool
	}{
		{
			name:      "happy path - no branch, asc sort",
			branch:    "",
			sortOrder: "asc",
			mockFn: func() {
				mock.ExpectQuery(`(?s)SELECT .* FROM class_books cb .*`).
					WithArgs("").
					WillReturnRows(pgxmock.NewRows(columns).
						AddRow("05A", "book1", "Math", "Math", "A", "cover1.jpg", "123", 5, 10).
						AddRow("05A", "book2", "Science", "Science", "B", "cover2.jpg", "456", 2, 5).
						AddRow("06B", "book3", "History", "History", "C", "cover3.jpg", "789", 0, 3))
			},
			want: []ClassGroup{
				{
					ClassName: "05A",
					Books: []ClassBook{
						{ID: "book1", Title: "Math", Subject: "Math", Track: "A", CoverURL: "cover1.jpg", ISBN: "123", Stock: 10, Verfuegbar: 5, Gesamt: 10},
						{ID: "book2", Title: "Science", Subject: "Science", Track: "B", CoverURL: "cover2.jpg", ISBN: "456", Stock: 5, Verfuegbar: 2, Gesamt: 5},
					},
				},
				{
					ClassName: "06B",
					Books: []ClassBook{
						{ID: "book3", Title: "History", Subject: "History", Track: "C", CoverURL: "cover3.jpg", ISBN: "789", Stock: 3, Verfuegbar: 0, Gesamt: 3},
					},
				},
			},
			wantErr: false,
		},
		{
			name:      "happy path - with branch, desc sort",
			branch:    "G",
			sortOrder: "desc",
			mockFn: func() {
				mock.ExpectQuery(`(?s)SELECT .* FROM class_books cb .*`).
					WithArgs("G").
					WillReturnRows(pgxmock.NewRows(columns).
						AddRow("07G", "book4", "Art", "Art", "", "", "", 1, 1))
			},
			want: []ClassGroup{
				{
					ClassName: "07G",
					Books: []ClassBook{
						{ID: "book4", Title: "Art", Subject: "Art", Track: "", CoverURL: "", ISBN: "", Stock: 1, Verfuegbar: 1, Gesamt: 1},
					},
				},
			},
			wantErr: false,
		},
		{
			name:      "db query error",
			branch:    "",
			sortOrder: "",
			mockFn: func() {
				mock.ExpectQuery(`(?s)SELECT .* FROM class_books cb .*`).
					WithArgs("").
					WillReturnError(fmt.Errorf("db connection failed"))
			},
			want:    nil,
			wantErr: true,
		},
		{
			name:      "db scan error",
			branch:    "",
			sortOrder: "",
			mockFn: func() {
				// Missing columns or invalid types to trigger scan error
				mock.ExpectQuery(`(?s)SELECT .* FROM class_books cb .*`).
					WithArgs("").
					WillReturnRows(pgxmock.NewRows([]string{"class_name"}).
						AddRow("05A"))
			},
			want:    nil,
			wantErr: true,
		},
		{
			name:      "db iteration error",
			branch:    "",
			sortOrder: "",
			mockFn: func() {
				mock.ExpectQuery(`(?s)SELECT .* FROM class_books cb .*`).
					WithArgs("").
					WillReturnRows(pgxmock.NewRows(columns).
						AddRow("05A", "book1", "Math", "Math", "A", "cover1.jpg", "123", 5, 10).
						RowError(0, fmt.Errorf("row error")))
			},
			want:    nil,
			wantErr: true,
		},
		{
			name:      "empty result",
			branch:    "",
			sortOrder: "",
			mockFn: func() {
				mock.ExpectQuery(`(?s)SELECT .* FROM class_books cb .*`).
					WithArgs("").
					WillReturnRows(pgxmock.NewRows(columns))
			},
			want:    nil, // Note: the function returns nil, nil when there are no rows because classNames is empty and result is nil.
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mockFn()
			got, err := repo.GetClassGroups(context.Background(), tt.branch, tt.sortOrder)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetClassGroups() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetClassGroups() = %v, want %v", got, tt.want)
			}
		})
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}
}
