package service

import (
	"context"
	"errors"
	"testing"

	"github.com/pashagolub/pgxmock/v5"
)

// Lässt sich der Abholfach-Hinweis nicht laden, bleibt er leer und der Scan geht weiter: Die
// Theke soll wegen eines Hinweises nicht stehen.
func TestAbholfachHinweis_FehlerLaesstIhnLeer(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock: %v", err)
	}
	defer mock.Close()
	mock.ExpectQuery(`FROM vormerkungen v`).WithArgs("leser-1").WillReturnError(errors.New("Datenbank nicht erreichbar"))

	svc := &defaultOmniboxService{pool: mock}
	if hinweis := svc.ladeAbholbereiteVormerkungen(context.Background(), "leser-1"); hinweis != nil {
		t.Errorf("Hinweis %+v, erwartet keinen", hinweis)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("offene Erwartungen: %v", err)
	}
}
