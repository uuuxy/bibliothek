package theke

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"bibliothek/internal/tuertest"
	"bibliothek/repository"

	"github.com/google/uuid"
)

// Die Vormerkung eines anderen hält das Buch zurück, an der Online-Theke wie beim Nachbuchen.
// Beide Türen nennen den Satz des Dienstes, ohne die Bezeichnung der Fehlerart davor: die
// Online-Theke als Meldung des 409, das Nachbuchen als Grund des abgewiesenen Eintrags.
func TestVormerkungEinesAnderen_SatzAnBeidenTueren(t *testing.T) {
	w := baueNachbuchWelt(t)
	ctx := context.Background()
	if _, err := w.pool.Exec(ctx, `
		INSERT INTO vormerkungen (titel_id, schueler_id, status, bereitgestellt_exemplar_id, bereitgestellt_bis)
		SELECT titel_id, $2, 'abholbereit', id, CURRENT_TIMESTAMP + interval '2 days'
		  FROM buecher_exemplare WHERE id = $1`, w.exemplarID, w.ben); err != nil {
		t.Fatalf("Vormerkung für Ben: %v", err)
	}
	const reserviert = "Achtung: dieses Exemplar ist noch für Ben Zweiter reserviert"

	rumpf := fmt.Sprintf(`{"query":%q,"idempotency_key":%q,"active_leser_id":%q}`, w.code, uuid.NewString(), w.anna)
	rec := httptest.NewRecorder()
	w.srv.EingabeVerarbeiten(w.online)(rec, w.sitzung(httptest.NewRequest(http.MethodPost, "/api/action", strings.NewReader(rumpf))))
	if rec.Code != http.StatusConflict {
		t.Fatalf("Online-Theke: Status %d, erwartet 409: %s", rec.Code, rec.Body.String())
	}
	if got := tuertest.Fehlermeldung(t, rec); got != reserviert {
		t.Errorf("Online-Theke: Meldung %q, erwartet %q", got, reserviert)
	}

	erg := w.nachbuchen(t, w.eintrag(uuid.NewString(), "ausleihe", &w.anna, time.Now().Add(-time.Minute)))
	if erg[0].Ergebnis != repository.NachbuchNichtGebucht || erg[0].Grund != reserviert {
		t.Errorf("Nachbuchen: %q mit Grund %q, erwartet %q mit %q",
			erg[0].Ergebnis, erg[0].Grund, repository.NachbuchNichtGebucht, reserviert)
	}
}
