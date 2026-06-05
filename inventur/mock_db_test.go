package inventur

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type MockDB struct {
	ExecFunc     func(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
	QueryRowFunc func(ctx context.Context, sql string, args ...any) pgx.Row
	QueryFunc    func(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
}

func (m *MockDB) Close() {}

func (m *MockDB) Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error) {
	if m.ExecFunc != nil {
		return m.ExecFunc(ctx, sql, arguments...)
	}
	return pgconn.CommandTag{}, fmt.Errorf("not implemented")
}

func (m *MockDB) QueryRow(ctx context.Context, sql string, args ...any) pgx.Row {
	if m.QueryRowFunc != nil {
		return m.QueryRowFunc(ctx, sql, args...)
	}
	return nil
}

func (m *MockDB) Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error) {
	if m.QueryFunc != nil {
		return m.QueryFunc(ctx, sql, args...)
	}
	return nil, fmt.Errorf("not implemented")
}

func (m *MockDB) Begin(ctx context.Context) (pgx.Tx, error) {
	return nil, fmt.Errorf("not implemented")
}

func (m *MockDB) BeginTx(ctx context.Context, txOptions pgx.TxOptions) (pgx.Tx, error) {
	return nil, fmt.Errorf("not implemented")
}

func (m *MockDB) Ping(ctx context.Context) error {
	return nil
}
