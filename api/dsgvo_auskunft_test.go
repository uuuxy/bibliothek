package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"bibliothek/db"

	"github.com/pashagolub/pgxmock/v4"
)

const dsgvoTestID = "11111111-1111-1111-1111-111111111111"

func dsgvoRequest(t *testing.T, mock pgxmock.PgxPoolIface) *httptest.ResponseRecorder {
	t.Helper()
	s := &Server{DB: &db.Database{Pool: mock}}
	req := httptest.NewRequest(http.MethodGet, "/api/schueler/"+dsgvoTestID+"/dsgvo-auskunft", nil)
	req.SetPathValue("id", dsgvoTestID)
	rec := httptest.NewRecorder()
	s.DsgvoAuskunftHandler()(rec, req)
	return rec
}

// expectStammdaten registriert die Stammdaten-Query mit einem vollständigen Schüler.
func expectStammdaten(mock pgxmock.PgxPoolIface) {
	geb := "2010-04-01"
	now := time.Now()
	mock.ExpectQuery(`SELECT id, barcode_id, vorname, nachname, klasse, geburtsdatum::text`).
		WithArgs(dsgvoTestID).
		WillReturnRows(pgxmock.NewRows([]string{
			"id", "barcode_id", "vorname", "nachname", "klasse", "geburtsdatum",
			"abgaenger_jahr", "ist_gesperrt", "ist_abgaenger", "lusd_id",
			"strasse", "hausnummer", "plz", "ort", "eltern_email",
			"is_manually_blocked", "block_reason", "erstellt_am", "aktualisiert_am", "deleted_at",
		}).AddRow(
			dsgvoTestID, "S-0042", "Max", "Muster", "07B", &geb,
			2029, false, false, (*string)(nil),
			"Reisstraße", "1", "61169", "Friedberg", "eltern@example.org",
			false, (*string)(nil), now, now, (*time.Time)(nil),
		))
}

func TestDsgvoAuskunft_HappyPathLiefertAlleSektionen(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	expectStammdaten(mock)
	fotoZeit := time.Now().Add(-24 * time.Hour)
	mock.ExpectQuery(`SELECT aktualisiert_am FROM schueler_fotos`).
		WithArgs(dsgvoTestID).
		WillReturnRows(pgxmock.NewRows([]string{"aktualisiert_am"}).AddRow(fotoZeit))
	mock.ExpectQuery(`FROM ausleihen a`).
		WithArgs(dsgvoTestID).
		WillReturnRows(pgxmock.NewRows([]string{"gegenstand", "barcode", "ausgeliehen_am", "rueckgabe_frist", "rueckgabe_am", "ist_handapparat"}).
			AddRow("Mathe 7", "B-100", time.Now(), time.Now().Add(14*24*time.Hour), (*time.Time)(nil), false))
	mock.ExpectQuery(`FROM schadensfaelle`).
		WithArgs(dsgvoTestID).
		WillReturnRows(pgxmock.NewRows([]string{"beschreibung", "betrag", "ist_bezahlt", "erstellt_am", "storniert_am", "stornierungsgrund"}).
			AddRow("Wasserschaden", "12.50", true, time.Now(), (*time.Time)(nil), (*string)(nil)))
	mock.ExpectQuery(`FROM vormerkungen v`).
		WithArgs(dsgvoTestID).
		WillReturnRows(pgxmock.NewRows([]string{"titel", "status", "notiz", "erstellt_am"}).
			AddRow("Faust I", "offen", (*string)(nil), time.Now()))
	mock.ExpectQuery(`FROM audit_log`).
		WithArgs(dsgvoTestID).
		WillReturnRows(pgxmock.NewRows([]string{"aktion", "akteur", "timestamp", "kontext", "details"}).
			AddRow("update", "USER", time.Now(), (*string)(nil), []byte(`{"feld":"klasse"}`)))
	mock.ExpectExec(`INSERT INTO audit_log`).
		WithArgs(dsgvoTestID, (*string)(nil), "SYSTEM").
		WillReturnResult(pgxmock.NewResult("INSERT", 1))

	rec := dsgvoRequest(t, mock)
	if rec.Code != http.StatusOK {
		t.Fatalf("erwartet 200, bekam %d: %s", rec.Code, rec.Body.String())
	}

	var resp DsgvoAuskunftResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Antwort kein valides JSON: %v", err)
	}
	if resp.Stammdaten.Nachname != "Muster" || !resp.Foto.Vorhanden ||
		len(resp.Ausleihen) != 1 || len(resp.Schadensfaelle) != 1 ||
		len(resp.Vormerkungen) != 1 || len(resp.AuditEintraege) != 1 {
		t.Errorf("Sektionen unvollständig: %+v", resp)
	}
	if len(resp.Verarbeitungsangaben.Zwecke) == 0 || resp.Verarbeitungsangaben.Rechtsgrundlage == "" {
		t.Errorf("Art.-15-Pflichtangaben fehlen: %+v", resp.Verarbeitungsangaben)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unerfüllte Erwartungen: %s", err)
	}
}

func TestDsgvoAuskunft_UnbekannterSchuelerIst404(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	mock.ExpectQuery(`SELECT id, barcode_id, vorname`).
		WithArgs(dsgvoTestID).
		WillReturnRows(pgxmock.NewRows([]string{"id"})) // keine Zeile

	rec := dsgvoRequest(t, mock)
	if rec.Code != http.StatusNotFound {
		t.Errorf("erwartet 404, bekam %d: %s", rec.Code, rec.Body.String())
	}
}

func TestDsgvoAuskunft_AuditFehlerVerhindertAuskunftNicht(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	expectStammdaten(mock)
	mock.ExpectQuery(`SELECT aktualisiert_am FROM schueler_fotos`).
		WithArgs(dsgvoTestID).
		WillReturnRows(pgxmock.NewRows([]string{"aktualisiert_am"})) // kein Foto
	for _, frag := range []string{`FROM ausleihen a`, `FROM schadensfaelle`, `FROM vormerkungen v`, `FROM audit_log`} {
		mock.ExpectQuery(frag).WithArgs(dsgvoTestID).
			WillReturnRows(pgxmock.NewRows([]string{"x"}))
	}
	mock.ExpectExec(`INSERT INTO audit_log`).
		WithArgs(dsgvoTestID, (*string)(nil), "SYSTEM").
		WillReturnError(errors.New("audit kaputt"))

	rec := dsgvoRequest(t, mock)
	if rec.Code != http.StatusOK {
		t.Errorf("Audit-Fehler darf die Auskunft nicht blockieren: %d %s", rec.Code, rec.Body.String())
	}
}
