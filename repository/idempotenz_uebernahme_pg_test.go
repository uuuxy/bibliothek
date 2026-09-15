package repository

import (
	"context"
	"testing"

	"bibliothek/internal/pgtest"

	"github.com/google/uuid"
)

// Die Nachbuch-Tür übernimmt eine gespeicherte Online-Antwort, bevor sie unter dem Schlüssel
// bucht. Nur wer genau die gelesene Antwort vorfindet, übernimmt: Ein zweiter Aufruf mit derselben
// Portion darf nicht ebenfalls übernehmen und daneben buchen. Und nach einem Serverfehler kommt die
// Antwort unverändert zurück.
func TestIdempotenz_UebernahmeNurEinmalUndZuruecklegen(t *testing.T) {
	pool := pgtest.Pool(t)
	ctx := context.Background()
	k := uuid.NewString()
	if _, err := pool.Exec(ctx, `INSERT INTO idempotency_keys (idempotency_key, response_data, status_code)
		VALUES ($1, '{"type": "rueckgabe", "fremdrueckgabe": true}', 200)`, k); err != nil {
		t.Fatalf("Online-Antwort: %v", err)
	}
	alt, err := LiesIdempotenzAntwort(ctx, pool, k)
	if err != nil {
		t.Fatalf("lesen: %v", err)
	}

	anders := IdempotenzAntwort{Daten: []byte(`{"type": "ausleihe"}`), Status: 200}
	if ok, err := UebernimmIdempotenzAntwort(ctx, pool, k, anders); err != nil || ok {
		t.Fatalf("Übernahme mit einer anderen Antwort: ok=%v err=%v — erwartet abgelehnt", ok, err)
	}
	if ok, err := UebernimmIdempotenzAntwort(ctx, pool, k, *alt); err != nil || !ok {
		t.Fatalf("erste Übernahme: ok=%v err=%v", ok, err)
	}
	if ok, err := UebernimmIdempotenzAntwort(ctx, pool, k, *alt); err != nil || ok {
		t.Fatalf("zweite Übernahme derselben Antwort: ok=%v err=%v — erwartet abgelehnt", ok, err)
	}
	jetzt, err := LiesIdempotenzAntwort(ctx, pool, k)
	if err != nil || !jetzt.InArbeit() {
		t.Fatalf("nach der Übernahme: %+v %v — erwartet eine Reservierung", jetzt, err)
	}

	if err := StelleIdempotenzAntwortWiederHer(ctx, pool, k, *alt); err != nil {
		t.Fatalf("zurücklegen: %v", err)
	}
	zurueck, err := LiesIdempotenzAntwort(ctx, pool, k)
	if err != nil || zurueck.Status != alt.Status || string(zurueck.Daten) != string(alt.Daten) {
		t.Fatalf("zurückgelegt: %+v %v, erwartet %+v", zurueck, err, alt)
	}
	if err := StelleIdempotenzAntwortWiederHer(ctx, pool, k, *alt); err != ErrIdempotenzNichtReserviert {
		t.Errorf("zurücklegen ohne Reservierung: %v, erwartet ErrIdempotenzNichtReserviert", err)
	}
}
