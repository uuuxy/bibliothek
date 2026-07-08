package repository

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/pashagolub/pgxmock/v4"
)

func BenchmarkQueryUeberfaelligeNachKlasse(b *testing.B) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		b.Fatalf("pgxmock: %v", err)
	}
	defer mock.Close()

	repo := NewMahnwesenRepository(mock)
	frist := time.Date(2026, 6, 20, 12, 0, 0, 0, time.UTC)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Mock the first query
		rows := pgxmock.NewRows([]string{
			"id", "s_id", "name", "klasse",
			"titel", "autor", "isbn", "cover_url",
			"rueckgabe_frist", "tage_ueberfaellig",
		})

        // Let's mock a larger number of classes to see the impact of ANY($1) more clearly, or simulate many classes
        for j := 0; j < 100; j++ {
			klasse := fmt.Sprintf("Klasse%d", j)
			rows.AddRow("a1", "s1", "Anna", klasse, "Faust", "Goethe", "978-1", "", frist, 17)
		}
		mock.ExpectQuery(`SELECT a\.id, s\.id`).WillReturnRows(rows)

		// Mock the second query (the one we want to optimize)
		// With ANY we need to match any query
		mock.ExpectQuery(`SELECT klasse, lehrer_email FROM klassen_lehrer_mapping`).
			WillReturnRows(pgxmock.NewRows([]string{"klasse", "lehrer_email"}))

		_, _ = repo.QueryUeberfaelligeNachKlasse(context.Background(), "")
	}
}
