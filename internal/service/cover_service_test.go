package service

import (
	"testing"

	"github.com/pashagolub/pgxmock/v4"
)

func TestNewCoverService(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock: %v", err)
	}
	defer mock.Close()

	service := NewCoverService(mock)
	if service == nil {
		t.Fatal("Erwartete ein initialisiertes CoverService, erhielt nil")
	}

	if service.db != mock {
		t.Errorf("Erwartete, dass die db-Instanz übereinstimmt")
	}
}
