package inventur

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type MockTx struct {
	pgx.Tx
	ExecCount int
	ExecArgs  [][]any
	CommitErr error
	RollbackCalled bool
	CommitCalled bool
	ExecErr error
}

func (m *MockTx) Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error) {
	m.ExecCount++
	m.ExecArgs = append(m.ExecArgs, append([]any{sql}, arguments...))
	return pgconn.CommandTag{}, m.ExecErr
}

func (m *MockTx) Commit(ctx context.Context) error {
	m.CommitCalled = true
	return m.CommitErr
}

func (m *MockTx) Rollback(ctx context.Context) error {
	m.RollbackCalled = true
	return nil
}

type MockPool struct {
	ExecCount int
	ExecArgs  [][]any
	Tx        *MockTx
	BeginErr  error
}

func (m *MockPool) Close() {}

func (m *MockPool) Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error) {
	m.ExecCount++
	m.ExecArgs = append(m.ExecArgs, append([]any{sql}, arguments...))
	return pgconn.CommandTag{}, nil
}

func (m *MockPool) Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error) {
	return nil, nil
}

func (m *MockPool) QueryRow(ctx context.Context, sql string, args ...any) pgx.Row {
	return nil
}

func (m *MockPool) Begin(ctx context.Context) (pgx.Tx, error) {
	if m.BeginErr != nil {
		return nil, m.BeginErr
	}
	if m.Tx == nil {
		m.Tx = &MockTx{}
	}
	return m.Tx, nil
}

func (m *MockPool) BeginTx(ctx context.Context, txOptions pgx.TxOptions) (pgx.Tx, error) {
	return m.Begin(ctx)
}

func (m *MockPool) Ping(ctx context.Context) error {
	return nil
}

var ErrMockBegin = errors.New("mock begin error")
var ErrMockExec = errors.New("mock exec error")
