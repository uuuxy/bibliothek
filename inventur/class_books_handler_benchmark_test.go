package inventur

import (
	"context"
	"fmt"
	"testing"
	"github.com/pashagolub/pgxmock/v4"
)

func BenchmarkDeleteClassGroupSingle(b *testing.B) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		b.Fatalf("failed to create pgxmock: %v", err)
	}
	defer mock.Close()

	repo := NewBookRepository(mock)
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		className := []string{fmt.Sprintf("Class%d", i)}
		mock.ExpectExec("DELETE FROM class_books WHERE class_name = ANY\\(\\$1\\)").
			WithArgs(className).
			WillReturnResult(pgxmock.NewResult("DELETE", 1))

		_ = repo.DeleteClassGroup(ctx, className)
	}
}

func BenchmarkDeleteClassGroupBatch(b *testing.B) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		b.Fatalf("failed to create pgxmock: %v", err)
	}
	defer mock.Close()

	repo := NewBookRepository(mock)
	ctx := context.Background()

	classNames := make([]string, 100)
	for i := 0; i < 100; i++ {
		classNames[i] = fmt.Sprintf("Class%d", i)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		mock.ExpectExec("DELETE FROM class_books WHERE class_name = ANY\\(\\$1\\)").
			WithArgs(classNames).
			WillReturnResult(pgxmock.NewResult("DELETE", 100))

		_ = repo.DeleteClassGroup(ctx, classNames)
	}
}
