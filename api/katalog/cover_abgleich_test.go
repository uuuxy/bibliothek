package katalog

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"bibliothek/api/tuer"
	"bibliothek/db"
	"bibliothek/internal/service"

	"github.com/jackc/pgx/v5"
)

// auswahlMelder meldet, dass ein Lauf des Abgleichs seine Auswahl liest, und lässt ihn danach
// mit einem Fehler enden.
type auswahlMelder struct {
	db.Pool
	gelesen chan struct{}
}

func (m auswahlMelder) Query(context.Context, string, ...any) (pgx.Rows, error) {
	m.gelesen <- struct{}{}
	return nil, errors.New("attrappe: keine Auswahl")
}

// Der Knopf startet den Abgleich, den die Tür bekommt. Nur wenn Start, Zeitplan und Knopf
// denselben Dienst haben, verhindert dessen Sperre zwei Läufe zugleich; ein Dienst, den die Tür
// selbst baute, liefe neben dem vom Start her.
func TestCoverAbgleichen_StartetDenGegebenenDienst(t *testing.T) {
	gegeben := make(chan struct{}, 1)
	ausDerBasis := make(chan struct{}, 1)
	abgleich := service.NewCoverDienst(auswahlMelder{gelesen: gegeben})
	s := &Tueren{Basis: &tuer.Basis{DB: &db.Verbindung{Pool: auswahlMelder{gelesen: ausDerBasis}}}}

	rec := httptest.NewRecorder()
	s.CoverAbgleichen(abgleich)(rec, httptest.NewRequest(http.MethodPost, "/api/admin/sync-covers", nil))

	if rec.Code != http.StatusOK {
		t.Errorf("CoverAbgleichen: Status %d, erwartet 200", rec.Code)
	}
	select {
	case <-gegeben:
	case <-ausDerBasis:
		t.Fatal("CoverAbgleichen: Die Tür hat einen eigenen Dienst gestartet statt des gegebenen")
	case <-time.After(5 * time.Second):
		t.Fatal("CoverAbgleichen: In fünf Sekunden hat kein Lauf begonnen")
	}
}
