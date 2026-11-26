package repository_test

import (
	"context"
	"testing"

	"bibliothek/internal/pgtest"
	"bibliothek/internal/repository"
)

// Die Freigabe nach einem Serverfehler löscht nur die eigene Reservierung. Die Reservierung eines
// anderen Arbeitsplatzes bleibt, sonst bucht dessen Wiederholung ein zweites Mal; eine schon
// geschriebene Antwort bleibt auch, sonst bucht die Wiederholung desselben Scans neu.
func TestGibIdempotenzSchluesselFrei_NurDieEigeneReservierung(t *testing.T) {
	pool := pgtest.Pool(t)
	ctx := context.Background()

	var eigen, fremd, beantwortet string
	if err := pool.QueryRow(ctx, `SELECT gen_random_uuid()::text, gen_random_uuid()::text, gen_random_uuid()::text`).
		Scan(&eigen, &fremd, &beantwortet); err != nil {
		t.Fatalf("Schlüssel erzeugen: %v", err)
	}
	for _, s := range []string{eigen, fremd, beantwortet} {
		if ok, _, err := repository.ReserviereIdempotenzSchluessel(ctx, pool, s); err != nil || !ok {
			t.Fatalf("Reservieren = %v, %v; erwartet true", ok, err)
		}
	}
	if err := repository.SchreibeIdempotenzAntwort(ctx, pool, beantwortet, []byte(`{"art": "ausleihe"}`), 200); err != nil {
		t.Fatalf("Antwort schreiben: %v", err)
	}

	if frei, err := repository.GibIdempotenzSchluesselFrei(ctx, pool, eigen); err != nil || !frei {
		t.Fatalf("eigene Reservierung freigeben = %v, %v; erwartet true", frei, err)
	}
	if frei, err := repository.GibIdempotenzSchluesselFrei(ctx, pool, beantwortet); err != nil || frei {
		t.Errorf("beantworteten Schlüssel freigeben = %v, %v; erwartet false", frei, err)
	}
	if _, err := repository.LadeIdempotenzAntwort(ctx, pool, fremd); err != nil {
		t.Errorf("die Reservierung eines anderen Arbeitsplatzes ist weg: %v", err)
	}
	if a, err := repository.LadeIdempotenzAntwort(ctx, pool, beantwortet); err != nil || a.Status != 200 {
		t.Errorf("die geschriebene Antwort ist weg: %+v, %v", a, err)
	}
}
