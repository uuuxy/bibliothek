package theke

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

// Die Tür nimmt die volle Menge an; erst der Eintrag darüber weist den ganzen Aufruf ab, bevor etwas
// gebucht ist.
func TestNachbuchen_Hoechstmenge(t *testing.T) {
	w := baueNachbuchWelt(t)
	eintraege := func(n int) []map[string]any {
		alle := make([]map[string]any, n)
		for i := range alle {
			alle[i] = w.eintrag(uuid.NewString(), "rueckgabe", nil, time.Now().Add(-time.Minute))
		}
		return alle
	}

	if erg := w.nachbuchen(t, eintraege(nachbuchenHoechstens)...); len(erg) != nachbuchenHoechstens {
		t.Errorf("%d Ergebnisse, erwartet %d", len(erg), nachbuchenHoechstens)
	}

	rumpf, err := json.Marshal(map[string]any{"eintraege": eintraege(nachbuchenHoechstens + 1), "gesendet_am": time.Now()})
	if err != nil {
		t.Fatalf("Rumpf: %v", err)
	}
	rec := httptest.NewRecorder()
	w.srv.WarteschlangeNachbuchen(w.nachbuch)(rec, w.sitzung(httptest.NewRequest(http.MethodPost, "/api/action/nachbuchen", strings.NewReader(string(rumpf)))))
	if rec.Code != http.StatusBadRequest {
		t.Errorf("Ein Eintrag zu viel: Status %d, erwartet 400", rec.Code)
	}
}
