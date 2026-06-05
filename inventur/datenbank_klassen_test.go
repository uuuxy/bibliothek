package inventur

import (
	"context"
	"strings"
	"testing"
)

func TestUpdateClassBooksMock(t *testing.T) {
	ctx := context.Background()

	t.Run("Insert new class books", func(t *testing.T) {
		mockPool := &MockPool{}
		repo := &BookRepository{db: mockPool}

		err := repo.UpdateClassBooks(ctx, "", []string{"1A"}, []string{"book1", "book2"})
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}

		if !mockPool.Tx.CommitCalled {
			t.Error("expected commit to be called")
		}

		// 1 for DELETE (if len(newClassNames) > 0), 1 for INSERT
		if mockPool.Tx.ExecCount != 2 {
			t.Errorf("expected 2 Exec calls, got %d", mockPool.Tx.ExecCount)
		}

		deleteSql := mockPool.Tx.ExecArgs[0][0].(string)
		if !strings.Contains(deleteSql, "DELETE FROM class_books WHERE class_name = ANY($1)") {
			t.Errorf("expected delete query, got %s", deleteSql)
		}

		insertSql := mockPool.Tx.ExecArgs[1][0].(string)
		if !strings.Contains(insertSql, "INSERT INTO class_books") {
			t.Errorf("expected insert query, got %s", insertSql)
		}
	})

	t.Run("Rename class and keep books", func(t *testing.T) {
		mockPool := &MockPool{}
		repo := &BookRepository{db: mockPool}

		err := repo.UpdateClassBooks(ctx, "1A", []string{"2A"}, []string{"book1"})
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}

		if !mockPool.Tx.CommitCalled {
			t.Error("expected commit to be called")
		}

		// 1 for old class DELETE, 1 for target class DELETE, 1 for INSERT
		if mockPool.Tx.ExecCount != 3 {
			t.Errorf("expected 3 Exec calls, got %d", mockPool.Tx.ExecCount)
		}

		deleteOldSql := mockPool.Tx.ExecArgs[0][0].(string)
		if !strings.Contains(deleteOldSql, "DELETE FROM class_books WHERE class_name = $1") {
			t.Errorf("expected old delete query, got %s", deleteOldSql)
		}

		deleteTargetSql := mockPool.Tx.ExecArgs[1][0].(string)
		if !strings.Contains(deleteTargetSql, "DELETE FROM class_books WHERE class_name = ANY($1)") {
			t.Errorf("expected target delete query, got %s", deleteTargetSql)
		}

		insertSql := mockPool.Tx.ExecArgs[2][0].(string)
		if !strings.Contains(insertSql, "INSERT INTO class_books") {
			t.Errorf("expected insert query, got %s", insertSql)
		}
	})

	t.Run("Delete old class without new class assignment", func(t *testing.T) {
		mockPool := &MockPool{}
		repo := &BookRepository{db: mockPool}

		err := repo.UpdateClassBooks(ctx, "4A", []string{}, []string{})
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}

		if !mockPool.Tx.CommitCalled {
			t.Error("expected commit to be called")
		}

		// 1 for old class DELETE
		if mockPool.Tx.ExecCount != 1 {
			t.Errorf("expected 1 Exec call, got %d", mockPool.Tx.ExecCount)
		}

		deleteOldSql := mockPool.Tx.ExecArgs[0][0].(string)
		if !strings.Contains(deleteOldSql, "DELETE FROM class_books WHERE class_name = $1") {
			t.Errorf("expected old delete query, got %s", deleteOldSql)
		}
	})

	t.Run("Transaction Rollback on Begin Error", func(t *testing.T) {
		mockPool := &MockPool{BeginErr: ErrMockBegin}
		repo := &BookRepository{db: mockPool}

		err := repo.UpdateClassBooks(ctx, "1A", []string{"2A"}, []string{"book1"})
		if err == nil {
			t.Error("expected error, got nil")
		}

		if !strings.Contains(err.Error(), "transaktion konnte nicht gestartet werden") {
			t.Errorf("unexpected error message: %v", err)
		}
	})

	t.Run("Transaction Rollback on Insert Error", func(t *testing.T) {
		mockTx := &MockTx{ExecErr: ErrMockExec}
		mockPool := &MockPool{Tx: mockTx}
		repo := &BookRepository{db: mockPool}

		err := repo.UpdateClassBooks(ctx, "1A", []string{"2A"}, []string{"book1"})
		if err == nil {
			t.Error("expected error, got nil")
		}

		if !mockTx.RollbackCalled {
			t.Error("expected rollback to be called")
		}

		if mockTx.CommitCalled {
			t.Error("expected commit to not be called")
		}
	})
}
