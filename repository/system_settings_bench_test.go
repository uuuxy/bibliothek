package repository

import (
	"context"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"testing"
)

type mockDB struct{}

func (m *mockDB) Close() {}
func (m *mockDB) Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error) {
	return pgconn.CommandTag{}, nil
}
func (m *mockDB) Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error) {
	return nil, nil
}
func (m *mockDB) QueryRow(ctx context.Context, sql string, args ...any) pgx.Row {
	return nil
}
func (m *mockDB) Begin(ctx context.Context) (pgx.Tx, error) {
	return nil, nil
}
func (m *mockDB) BeginTx(ctx context.Context, txOptions pgx.TxOptions) (pgx.Tx, error) {
	return nil, nil
}
func (m *mockDB) Ping(ctx context.Context) error {
	return nil
}
func (m *mockDB) CopyFrom(ctx context.Context, tableName pgx.Identifier, columnNames []string, rowSrc pgx.CopyFromSource) (int64, error) {
	return 0, nil
}

func BenchmarkSaveSettings(b *testing.B) {
	repo := NewSystemSettingsRepository(&mockDB{})
	ctx := context.Background()
	settings := &EinstellungenPatch{
		FerienLeseclubAktiv:  ptr(true),
		LmfStichtag:          ptr("08-01"),
		MaxAusleihenSchueler: ptr(10),
		FristBuchTage:        ptr(30),
		FristMedienTage:      ptr(14),
		MaxOverdueDays:       ptr(20),
		MaxOverdueItems:      ptr(5),
		SchuleName:           ptr("Test School"),
		SchuleStrasse:        ptr("Test Street 1"),
		SchulePLZ:            ptr("12345"),
		SchuleOrt:            ptr("Test City"),
	}

	for b.Loop() {
		_ = repo.SaveSettings(ctx, settings) //nolint:errcheck
	}
}
