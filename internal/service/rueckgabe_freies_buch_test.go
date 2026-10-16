package service

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"bibliothek/repository"

	"github.com/jackc/pgx/v5"
	"github.com/pashagolub/pgxmock/v5"
)

// Wer ein freies Buch scannt, ohne dass ein Ausweis vorliegt, bekommt eine Meldung — keine
// Ausleihe auf den eigenen Namen.
//
// Buchte das System ein freies Buch, das ein angemeldetes Kollegiumskonto scannt, sofort auf
// diese Person, sammelte, wer Rückläufer sortiert, still Bücher auf seinem eigenen Namen — und
// merkte es erst, wenn die Mahnung käme oder die Inventur das Buch vermisste. Ausgeliehen wird
// über den Ausweis. Dieser Test hält die Tür zu.
type selbstausleiheAttrappe struct {
	repository.AusleihAbfragen
	pool      pgxmock.PgxPoolIface
	geschrieb bool
}

func (r *selbstausleiheAttrappe) BeginneTx(ctx context.Context) (pgx.Tx, error) {
	return r.pool.Begin(ctx)
}

func (r *selbstausleiheAttrappe) SperreOffene(_ context.Context, _ pgx.Tx, _ string) (*repository.Ausleihe, error) {
	return nil, nil // Buch ist frei — genau der Fall, in dem der Sonderweg zuschlug
}

func (r *selbstausleiheAttrappe) LegeAn(_ context.Context, _ pgx.Tx, _, _, _ string, _ time.Time, _ bool) (*repository.Ausleihe, error) {
	r.geschrieb = true
	return &repository.Ausleihe{}, nil
}

func TestFreiesBuchWirdNichtAufDenScannerGebucht(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()
	mock.ExpectBegin()
	mock.ExpectRollback()

	repo := &selbstausleiheAttrappe{pool: mock}
	svc := &ausleihDienst{ausleihAbfragen: repo}
	exemplar := &repository.Exemplar{ID: "c1", TitelID: "t1", IstAusleihbar: true}

	_, err = svc.NimmZurueck(context.Background(), exemplar, "staff1")

	if !errors.Is(err, ErrUngueltigerZustand) || !strings.Contains(err.Error(), "nicht ausgeliehen") {
		t.Errorf("ein freies Buch ohne Ausweis muss „nicht ausgeliehen\" melden, bekam: %v", err)
	}
	if repo.geschrieb {
		t.Error("es wurde eine Ausleihe geschrieben — der Scanner hat das Buch auf sich selbst gebucht")
	}
}
