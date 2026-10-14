package portal

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"bibliothek/api/tuer"
	"bibliothek/auth"
	"bibliothek/db"
	"bibliothek/internal/pgtest"
	"bibliothek/internal/testdaten"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Die Notiz der Bibliothek braucht einen Weg außerhalb der Bereit-Mail.
//
// Die Antwort der Bibliothek („24 von 30, Rest bei der 8a") stünde sonst nur in der Mail:
// Scheitert die Mail oder fehlt die Adresse, sähe die Lehrkraft sie nie. Portal und Theke
// lesen sie deshalb wie beim Anliegen, dazu erledigt_am (Migration 089) für „erledigt am …".
func TestEigeneKlassensatzReservierungen_NotizUndZeitpunktKommenAn(t *testing.T) {
	pool := pgtest.Pool(t)
	testdaten.LeereBestandsdaten(t, pool)
	ctx := context.Background()
	srv := &Tueren{Basis: &tuer.Basis{DB: &db.Database{Pool: pool}}}

	lehrerA := testdaten.LegeKollegiumskontoAn(t, pool, "eigene-a@test.invalid")
	lehrerB := testdaten.LegeKollegiumskontoAn(t, pool, "eigene-b@test.invalid")
	titelID := testdaten.LegeTitelMitCoverUndAlterAn(t, pool, "Die Physiker", "Jug Due", false, 0)

	offenA := legeReservierungAn(t, pool, titelID, lehrerA, "07B")
	fertigA := legeReservierungAn(t, pool, titelID, lehrerA, "09C")
	legeReservierungAn(t, pool, titelID, lehrerB, "05A")

	// Abschluss über die echte Tür, mit Notiz der Bibliothek und scheiterndem Mailversand:
	// der Fall, in dem nur die Zeile die Notiz trägt.
	ersetzeMailversand(t, errors.New("smtp down"))
	body := `{"notiz":"24 von 30, Rest bei der 8a"}`
	req := httptest.NewRequest(http.MethodPut,
		"/api/reservierungen/klassensatz/"+fertigA+"/erledigen", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.SetPathValue("id", fertigA)
	rec := httptest.NewRecorder()
	srv.KlassensatzReservierungErledigen()(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("Erledigen: HTTP %d: %s", rec.Code, rec.Body.String())
	}

	// Migration 089: Der Abschlusszeitpunkt steht in der Zeile.
	var erledigtAmGesetzt bool
	if err := pool.QueryRow(ctx, `SELECT erledigt_am IS NOT NULL FROM klassensatz_reservierungen WHERE id = $1`,
		fertigA).Scan(&erledigtAmGesetzt); err != nil {
		t.Fatalf("erledigt_am lesen: %v", err)
	}
	if !erledigtAmGesetzt {
		t.Fatal("erledigt_am ist NULL — der Abschluss hat keinen Zeitpunkt")
	}

	meine := eigeneReservierungen(t, srv, lehrerA)
	if len(meine) != 2 {
		t.Fatalf("Lehrkraft A: %d Zeilen, erwartet 2 (eigene offene + eigene erledigte): %+v", len(meine), meine)
	}
	nachID := map[string]map[string]any{}
	for _, m := range meine {
		id, istString := m["id"].(string)
		if !istString {
			t.Fatalf("Zeile ohne id: %+v", m)
		}
		nachID[id] = m
	}
	if _, ok := nachID[offenA]; !ok {
		t.Fatalf("eigene offene Reservierung fehlt: %+v", meine)
	}
	fertig, ok := nachID[fertigA]
	if !ok {
		t.Fatalf("eigene erledigte Reservierung fehlt — der Abschluss darf sie nicht aus dem Portal tilgen: %+v", meine)
	}
	if fertig["erledigt_notiz"] != "24 von 30, Rest bei der 8a" {
		t.Fatalf("Bibliotheks-Notiz fehlt im Portal: %+v", fertig)
	}
	if s, istString := fertig["erledigt_am"].(string); !istString || s == "" {
		t.Fatalf("erledigt_am fehlt in der Portal-Antwort: %+v", fertig)
	}

	// Fremde Reservierungen bleiben draußen — B sieht nur die eigene.
	fremde := eigeneReservierungen(t, srv, lehrerB)
	if len(fremde) != 1 {
		t.Fatalf("Lehrkraft B: %d Zeilen, erwartet 1: %+v", len(fremde), fremde)
	}
}

// eigeneReservierungen ruft die echte Tür mit der Sitzung der Lehrkraft auf.
func eigeneReservierungen(t *testing.T, srv *Tueren, benutzerID string) []map[string]any {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/api/reservierungen/klassensatz/eigene", nil)
	req = req.WithContext(context.WithValue(req.Context(), auth.ClaimsContextKey,
		&auth.Claims{UserID: benutzerID}))
	rec := httptest.NewRecorder()
	srv.EigeneKlassensatzReservierungen()(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("eigene: HTTP %d: %s", rec.Code, rec.Body.String())
	}
	var meine []map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &meine); err != nil {
		t.Fatalf("Antwort unlesbar: %v / %s", err, rec.Body.String())
	}
	return meine
}

func legeReservierungAn(t *testing.T, pool *pgxpool.Pool, titelID, benutzerID, klasse string) string {
	t.Helper()
	var id string
	if err := pool.QueryRow(context.Background(), `
		INSERT INTO klassensatz_reservierungen (titel_id, klasse, anzahl, notiz, angefordert_von)
		VALUES ($1, $2, 25, 'bitte bis Montag', $3) RETURNING id`,
		titelID, klasse, benutzerID).Scan(&id); err != nil {
		t.Fatalf("Reservierung anlegen: %v", err)
	}
	return id
}
