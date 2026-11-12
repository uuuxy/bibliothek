package main

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"bibliothek/internal/littera"
	"bibliothek/internal/pgtest"
	"bibliothek/repository"
)

// Der Befehl legt das Ergebnis des schreibenden Laufs so ab, wie die Selbstprüfung es liest
// (littera.ErgebnisSchluessel); ein späterer Lauf ersetzt es.
func TestLegeErgebnisAb(t *testing.T) {
	pool := pgtest.Pool(t)
	ctx := context.Background()
	t.Cleanup(func() {
		if _, err := pool.Exec(context.Background(), `DELETE FROM system_einstellungen WHERE schluessel = $1`, littera.ErgebnisSchluessel); err != nil {
			t.Logf("Aufräumen: %v", err)
		}
	})
	lies := func() littera.Ergebnis {
		t.Helper()
		roh, err := repository.NewBetriebszustandAbfragen(pool).LadeEinstellungswert(ctx, littera.ErgebnisSchluessel)
		if err != nil {
			t.Fatalf("Ergebnis lesen: %v", err)
		}
		var e littera.Ergebnis
		if err := json.Unmarshal([]byte(roh), &e); err != nil {
			t.Fatalf("Ergebnis %q unlesbar: %v", roh, err)
		}
		return e
	}

	vorher := time.Now()
	legeErgebnisAb(pool, littera.Bericht{Abbruch: errors.New("beim Bestand: verbindung weg"), Fehler: 1, Warnungen: 4})
	e := lies()
	if !e.Abgebrochen || e.Vollstaendig || e.Fehler != 1 || e.Warnungen != 4 || e.Protokoll != protokollPfad || e.Zeitpunkt.Before(vorher.Add(-time.Second)) {
		t.Errorf("abgebrochener Lauf: %+v", e)
	}

	gut := littera.Bericht{Warnungen: 2}
	gut.Bestand.AbgleichOK, gut.Schlagworte.AbgleichOK, gut.Personen.AbgleichOK, gut.Ausleihen.AbgleichOK = true, true, true, true
	legeErgebnisAb(pool, gut)
	if e := lies(); e.Abgebrochen || !e.Vollstaendig || e.Warnungen != 2 {
		t.Errorf("späterer vollständiger Lauf ersetzt das Ergebnis nicht: %+v", e)
	}
}
