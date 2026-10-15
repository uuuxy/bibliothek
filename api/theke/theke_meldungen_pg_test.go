package theke_test

import (
	"net/http"
	"testing"

	"bibliothek/internal/pgtest"
	"bibliothek/internal/routertest"
	"bibliothek/internal/testdaten"
	"bibliothek/internal/tuertest"

	"github.com/google/uuid"
)

// Die Theke zeigt zu einem Scan, der nicht weiterführt, den Satz des Dienstes, ohne die
// Bezeichnung der Fehlerart davor. Wiederholt der Arbeitsplatz die Anfrage mit demselben
// Schlüssel, kommt derselbe Satz aus der Ablage.
func TestEingabeVerarbeiten_SatzDesDienstes(t *testing.T) {
	pool := pgtest.Pool(t)
	testdaten.LeereBestandsdaten(t, pool)
	_, rufe := routertest.MitAufruf(t, pool)

	for _, f := range []struct{ name, eingabe, meldung string }{
		{"unbekannter Ausweis", "S-UNBEKANNT-1", "Ausweis S-UNBEKANNT-1 ist nicht registriert"},
		{"Leser nicht mehr da", "leser:00000000-0000-4000-8000-0000000000e2", "Dieser Leser steht nicht mehr in der Leserdatei"},
	} {
		t.Run(f.name, func(t *testing.T) {
			rumpf := `{"query":"` + f.eingabe + `","idempotency_key":"` + uuid.NewString() + `"}`
			for _, versuch := range []string{"erster Versuch", "Wiederholung aus der Ablage"} {
				rec := rufe(t, http.MethodPost, "/api/action", rumpf)
				if rec.Code != http.StatusNotFound {
					t.Fatalf("%s: Status %d, erwartet 404: %s", versuch, rec.Code, rec.Body.String())
				}
				if got := tuertest.Fehlermeldung(t, rec); got != f.meldung {
					t.Errorf("%s: Meldung %q, erwartet %q", versuch, got, f.meldung)
				}
			}
		})
	}
}
