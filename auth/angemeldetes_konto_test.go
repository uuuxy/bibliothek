package auth

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/pashagolub/pgxmock/v5"
)

func rufeAngemeldetesKonto(t *testing.T, a *Anmeldung, mock pgxmock.PgxPoolIface, cookie *http.Cookie) *httptest.ResponseRecorder {
	t.Helper()
	anfrage := httptest.NewRequest(http.MethodGet, "/api/auth/me", nil)
	if cookie != nil {
		anfrage.AddCookie(cookie)
	}
	rec := httptest.NewRecorder()
	AngemeldetesKonto(mock, a)(rec, anfrage)
	return rec
}

func TestAngemeldetesKonto_OhneCookieIst401(t *testing.T) {
	a, mock := newTestAnmeldung(t, 12*time.Hour)
	rec := rufeAngemeldetesKonto(t, a, mock, nil)
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("rec.Code = %d, erwartet 401", rec.Code)
	}
}

func TestAngemeldetesKonto_AktiverAdminBekommtDieAntwortDerAnmeldung(t *testing.T) {
	a, mock := newTestAnmeldung(t, 12*time.Hour)
	token, err := a.ErzeugeToken("user-1", RolleAdmin, "")
	if err != nil {
		t.Fatalf("ErzeugeToken: %v", err)
	}

	erwarteNichtWiderrufen(mock)
	erwarteKontoAktiv(mock, true)
	mock.ExpectQuery(`SELECT rolle, vorname, nachname, aktiv, email`).
		WithArgs("user-1").
		WillReturnRows(pgxmock.NewRows([]string{"rolle", "vorname", "nachname", "aktiv", "email"}).
			AddRow("admin", "Nina", "Berger", true, "nina@example.org"))
	// Der Admin hat alle Rechte ("*"); role_permissions wird nicht gefragt.

	rec := rufeAngemeldetesKonto(t, a, mock, &http.Cookie{Name: "session_token", Value: token})
	if rec.Code != http.StatusOK {
		t.Fatalf("rec.Code = %d, erwartet 200: %s", rec.Code, rec.Body.String())
	}

	var antwort anmeldeAntwort
	if err := json.Unmarshal(rec.Body.Bytes(), &antwort); err != nil {
		t.Fatalf("Antwort ist kein JSON der Anmeldung: %v", err)
	}
	if antwort.KontoID != "user-1" || antwort.Rolle != RolleAdmin || antwort.Vorname != "Nina" || antwort.Nachname != "Berger" || antwort.Email != "nina@example.org" {
		t.Errorf("Stammdaten falsch: %+v", antwort)
	}
	if len(antwort.Rechte) != 1 || antwort.Rechte[0] != "*" {
		t.Errorf("Admin muss implizit '*' bekommen: %+v", antwort.Rechte)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("offene Erwartungen: %v", err)
	}
}

func TestAngemeldetesKonto_OhneAdminRechteAusDerMatrix(t *testing.T) {
	a, mock := newTestAnmeldung(t, 12*time.Hour)
	token, err := a.ErzeugeToken("user-2", RolleMitarbeiter, "")
	if err != nil {
		t.Fatalf("ErzeugeToken: %v", err)
	}

	erwarteNichtWiderrufen(mock)
	erwarteKontoAktiv(mock, true)
	mock.ExpectQuery(`SELECT rolle, vorname, nachname, aktiv, email`).
		WithArgs("user-2").
		WillReturnRows(pgxmock.NewRows([]string{"rolle", "vorname", "nachname", "aktiv", "email"}).
			AddRow("mitarbeiter", "Mia", "Muster", true, "mia@example.org"))
	mock.ExpectQuery(`SELECT permission`).
		WithArgs("mitarbeiter").
		WillReturnRows(pgxmock.NewRows([]string{"permission"}).
			AddRow("view_books").
			AddRow("view_students"))

	rec := rufeAngemeldetesKonto(t, a, mock, &http.Cookie{Name: "session_token", Value: token})
	if rec.Code != http.StatusOK {
		t.Fatalf("rec.Code = %d, erwartet 200: %s", rec.Code, rec.Body.String())
	}

	var antwort anmeldeAntwort
	if err := json.Unmarshal(rec.Body.Bytes(), &antwort); err != nil {
		t.Fatalf("Antwort ist kein JSON der Anmeldung: %v", err)
	}
	if len(antwort.Rechte) != 2 || antwort.Rechte[0] != "view_books" {
		t.Errorf("konfigurierte Rechte nicht durchgereicht: %+v", antwort.Rechte)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("offene Erwartungen: %v", err)
	}
}

func TestAngemeldetesKonto_DeaktiviertesKontoIst401(t *testing.T) {
	// Rolle und Status kommen aus der Datenbank, nicht aus den Claims: Ein inzwischen deaktiviertes
	// Konto darf seine Sitzung nicht wiederherstellen.
	a, mock := newTestAnmeldung(t, 12*time.Hour)
	token, err := a.ErzeugeToken("user-3", RolleMitarbeiter, "")
	if err != nil {
		t.Fatalf("ErzeugeToken: %v", err)
	}

	erwarteNichtWiderrufen(mock)
	// PruefeToken lehnt das deaktivierte Konto schon ab (aktiv=false); die eigene Abfrage der
	// Stammdaten in AngemeldetesKonto wird gar nicht erreicht.
	erwarteKontoAktiv(mock, false)

	rec := rufeAngemeldetesKonto(t, a, mock, &http.Cookie{Name: "session_token", Value: token})
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("rec.Code = %d, erwartet 401 für deaktivierten Benutzer", rec.Code)
	}
}

// Die Abfrage der Stammdaten hinter einer gültigen Sitzung hat drei Ausgänge mit Fehler: Ein
// fehlendes und ein deaktiviertes Konto sind beide keine aktive Sitzung (401). Ein Fehler der
// Datenbank ist es nicht (500), sonst meldete die Oberfläche ab.
func TestAngemeldetesKonto_KontoAbfrageScheitert(t *testing.T) {
	faelle := []struct {
		name    string
		antwort func(q *pgxmock.ExpectedQuery)
		status  int
	}{
		{"kein Konto", func(q *pgxmock.ExpectedQuery) {
			q.WillReturnError(pgx.ErrNoRows)
		}, http.StatusUnauthorized},
		{"Datenbankfehler", func(q *pgxmock.ExpectedQuery) {
			q.WillReturnError(errors.New("connection reset by peer"))
		}, http.StatusInternalServerError},
		{"Konto deaktiviert", func(q *pgxmock.ExpectedQuery) {
			q.WillReturnRows(pgxmock.NewRows([]string{"rolle", "vorname", "nachname", "aktiv", "email"}).
				AddRow("mitarbeiter", "Mia", "Muster", false, "mia@example.org"))
		}, http.StatusUnauthorized},
	}
	for _, f := range faelle {
		t.Run(f.name, func(t *testing.T) {
			a, mock := newTestAnmeldung(t, 12*time.Hour)
			token, err := a.ErzeugeToken("user-4", RolleMitarbeiter, "")
			if err != nil {
				t.Fatalf("ErzeugeToken: %v", err)
			}

			erwarteNichtWiderrufen(mock)
			erwarteKontoAktiv(mock, true)
			f.antwort(mock.ExpectQuery(`SELECT rolle, vorname, nachname, aktiv, email`).WithArgs("user-4"))

			rec := rufeAngemeldetesKonto(t, a, mock, &http.Cookie{Name: "session_token", Value: token})
			if rec.Code != f.status {
				t.Errorf("rec.Code = %d, erwartet %d: %s", rec.Code, f.status, rec.Body.String())
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("offene Erwartungen: %v", err)
			}
		})
	}
}
