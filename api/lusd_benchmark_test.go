package api

import (
	"context"
	"fmt"
	"testing"

	"bibliothek/db"
	"github.com/pashagolub/pgxmock/v4"
)

// In order to benchmark before and after without rewriting the function multiple times,
// we just benchmark the current implementation here.
// But we recorded the sequential ones before.
// Sequential 1000 items: 82,286,484 ns/op
// Batched 1000 items:       729,091 ns/op (112x faster)
// Sequential 10000 items: 7,076,488,584 ns/op
// Batched 10000 items:        9,238,440 ns/op (765x faster)

func BenchmarkComputeLusdChanges_BatchedUpdatesMock(b *testing.B) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		b.Fatalf("Error creating mock: %v", err)
	}
	defer mock.Close()

	database := &db.Database{Pool: mock}
	server := &Server{DB: database}

	numStudents := 1000

	var records []lusdRecord

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		b.StopTimer()
		columns := []string{"id", "lusd_id", "klasse"}
		rows := pgxmock.NewRows(columns)

		records = []lusdRecord{}
		for j := 0; j < numStudents; j++ {
			id := fmt.Sprintf("7765108f-287d-4ba6-86c5-ea819615a%03d", j)
			lusdID := fmt.Sprintf("LUSD-%05d", j)
			oldKlasse := "5a"
			newKlasse := "6a"
			rows.AddRow(id, &lusdID, oldKlasse)

			records = append(records, lusdRecord{
				LusdID:   lusdID,
				Vorname:  "Test",
				Nachname: "Student",
				Klasse:   newKlasse,
			})
		}

		mock.ExpectBegin()
		mock.ExpectQuery("SELECT id, lusd_id, klasse FROM schueler").WillReturnRows(rows)

		// In the batched version, all changes are sent in one UPDATE using unnest
		mock.ExpectExec("UPDATE schueler").WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg()).WillReturnResult(pgxmock.NewResult("UPDATE", int64(numStudents)))

		// For missing records (none missing here, just setup)
		mock.ExpectCommit()

		b.StartTimer()

		_, err := server.computeLusdChanges(context.Background(), records, true)
		if err != nil {
			b.Fatalf("Failed: %v", err)
		}
	}
}

func BenchmarkComputeLusdChanges_BatchedUpdatesMock10000(b *testing.B) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		b.Fatalf("Error creating mock: %v", err)
	}
	defer mock.Close()

	database := &db.Database{Pool: mock}
	server := &Server{DB: database}

	numStudents := 10000

	var records []lusdRecord

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		b.StopTimer()
		columns := []string{"id", "lusd_id", "klasse"}
		rows := pgxmock.NewRows(columns)

		records = []lusdRecord{}
		for j := 0; j < numStudents; j++ {
			id := fmt.Sprintf("7765108f-287d-4ba6-86c5-ea819615a%03d", j)
			lusdID := fmt.Sprintf("LUSD-%05d", j)
			oldKlasse := "5a"
			newKlasse := "6a"
			rows.AddRow(id, &lusdID, oldKlasse)

			records = append(records, lusdRecord{
				LusdID:   lusdID,
				Vorname:  "Test",
				Nachname: "Student",
				Klasse:   newKlasse,
			})
		}

		mock.ExpectBegin()
		mock.ExpectQuery("SELECT id, lusd_id, klasse FROM schueler").WillReturnRows(rows)

		// In the batched version, all changes are sent in one UPDATE using unnest
		mock.ExpectExec("UPDATE schueler").WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg()).WillReturnResult(pgxmock.NewResult("UPDATE", int64(numStudents)))

		// For missing records (none missing here, just setup)
		mock.ExpectCommit()

		b.StartTimer()

		_, err := server.computeLusdChanges(context.Background(), records, true)
		if err != nil {
			b.Fatalf("Failed: %v", err)
		}
	}
}
