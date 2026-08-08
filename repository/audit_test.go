package repository

import (
	"testing"

	"github.com/pashagolub/pgxmock/v4"
)

func TestNewAuditRepository(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("Fehler beim Erstellen von pgxmock: %v", err)
	}
	defer mock.Close()

	repo := NewAuditRepository(mock)
	if repo == nil {
		t.Fatal("NewAuditRepository sollte nicht nil zurückgeben")
	}

	// Typprüfung, dass es wirklich der gewünschte Typ ist
	_, ok := repo.(*pgAuditRepository)
	if !ok {
		t.Errorf("NewAuditRepository gibt falschen Typ zurück")
	}
}