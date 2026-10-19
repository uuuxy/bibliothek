package repository

import (
	"testing"

	"github.com/pashagolub/pgxmock/v5"
)

func TestNewProtokollAbfragen(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("Fehler beim Erstellen von pgxmock: %v", err)
	}
	defer mock.Close()

	repo := NewProtokollAbfragen(mock)
	if repo == nil {
		t.Fatal("NewProtokollAbfragen sollte nicht nil zurückgeben")
	}
	if repo.db != mock {
		t.Error("NewProtokollAbfragen setzt das Feld db nicht")
	}
}
