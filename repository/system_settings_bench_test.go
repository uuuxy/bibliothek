package repository

import (
	"context"
	"testing"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
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
	settings := &SystemEinstellungen{
		FerienLeseclubAktiv: true,
		LmfStichtag: "08-01",
		MaxAusleihenSchueler: 10,
		FristBuchTage: 30,
		FristMedienTage: 14,
		MaxOverdueDays: 20,
		MaxOverdueItems: 5,
		SchuleName: "Test School",
		SchuleStrasse: "Test Street 1",
		SchulePLZ: "12345",
		SchuleOrt: "Test City",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = repo.SaveSettings(ctx, settings)
	}
}
