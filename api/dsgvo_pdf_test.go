package api

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"bibliothek/db"
	"bibliothek/pdf"

	"github.com/pashagolub/pgxmock/v4"
)

// TestGenerateDsgvoAuskunftPDF stellt sicher, dass der Auskunfts-Generator aus
// vollständigen Daten ein valides, nicht-triviales PDF erzeugt — inkl. gefüllter
// Listen (die MultiCell-Umbrüche und optionale Zeiger-Felder auslösen).
func TestGenerateDsgvoAuskunftPDF(t *testing.T) {
	gebdatum := "2010-05-01"
	lusd := "LUSD-4711"
	sperrgrund := "Test-Sperrgrund"
	rueckgabe := time.Date(2025, 3, 1, 0, 0, 0, 0, time.UTC)
	notiz := "bitte zurücklegen"
	kontext := "Import"

	daten := &dsgvoDaten{
		stammdaten: &DsgvoStammdaten{
			ID: "abc-123", BarcodeID: "S-10001", Vorname: "Erika", Nachname: "Mustermann",
			Klasse: "7b", Geburtsdatum: &gebdatum, AbgaengerJahr: 2030, LusdID: &lusd,
			Strasse: "Hauptstr.", Hausnummer: "5", Plz: "60311", Ort: "Frankfurt",
			ElternEmail: "eltern@example.org", IstGesperrt: true, IsManuallyBlocked: true,
			BlockReason: &sperrgrund, ErstelltAm: time.Now(), AktualisiertAm: time.Now(),
		},
		foto:           DsgvoFoto{Vorhanden: true, AktualisiertAm: &rueckgabe, Hinweis: "verschlüsselt"},
		ausleihen:      []DsgvoAusleihe{{Gegenstand: "Mathebuch 7", Barcode: "B-500", AusgeliehenAm: time.Now(), RueckgabeFrist: time.Now(), RueckgabeAm: &rueckgabe}},
		schaeden:       []DsgvoSchadensfall{{Beschreibung: "Wasserschaden", Betrag: "12.50", IstBezahlt: false, ErstelltAm: time.Now()}},
		vormerkungen:   []DsgvoVormerkung{{Titel: "Deutschbuch 7", Status: "wartend", Notiz: &notiz, ErstelltAm: time.Now()}},
		auditEintraege: []DsgvoAuditEintrag{{Aktion: "update", Akteur: "USER", Zeitpunkt: time.Now(), Kontext: &kontext}},
	}

	out, err := generateDsgvoAuskunftPDF(daten, pdf.SchuleInfo{Name: "Testschule", Ort: "Frankfurt"})
	if err != nil {
		t.Fatalf("generateDsgvoAuskunftPDF: %v", err)
	}
	if !bytes.HasPrefix(out, []byte("%PDF")) {
		t.Errorf("Ausgabe ist kein PDF (Prefix %q)", out[:min(8, len(out))])
	}
	if len(out) < 1500 {
		t.Errorf("PDF verdächtig klein: %d Bytes", len(out))
	}
}

func dsgvoPDFRequest(t *testing.T, mock pgxmock.PgxPoolIface, pathID string) *httptest.ResponseRecorder {
	t.Helper()
	s := &Server{DB: &db.Database{Pool: mock}}
	req := httptest.NewRequest(http.MethodGet, "/api/schueler/"+pathID+"/dsgvo-auskunft/pdf", nil)
	req.SetPathValue("id", pathID)
	rec := httptest.NewRecorder()
	s.DsgvoAuskunftPDFHandler()(rec, req)
	return rec
}

func TestDsgvoAuskunftPDFHandler_Success(t *testing.T) {
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
	mock.ExpectQuery(`FROM audit_logs`).
		WithArgs(dsgvoTestID).
		WillReturnRows(pgxmock.NewRows([]string{"aktion", "zeitstempel", "details"}).
			AddRow("RESTORE_STUDENT", time.Now(), []byte(`{"schueler_id":"`+dsgvoTestID+`"}`)))

	mock.ExpectExec(`INSERT INTO audit_log`).
		WithArgs(dsgvoTestID, (*string)(nil), "SYSTEM").
		WillReturnResult(pgxmock.NewResult("INSERT", 1))

	mock.ExpectQuery(`SELECT schluessel, wert FROM system_einstellungen`).
		WillReturnRows(pgxmock.NewRows([]string{"schluessel", "wert"}).
			AddRow("schule_name", "Test School").
			AddRow("schule_ort", "Test City"))

	rec := dsgvoPDFRequest(t, mock, dsgvoTestID)

	if rec.Code != http.StatusOK {
		t.Fatalf("erwartet 200, bekam %d: %s", rec.Code, rec.Body.String())
	}
	if rec.Header().Get("Content-Type") != "application/pdf" {
		t.Errorf("falscher Content-Type: %s", rec.Header().Get("Content-Type"))
	}
	if !bytes.HasPrefix(rec.Body.Bytes(), []byte("%PDF")) {
		t.Errorf("Ausgabe ist kein PDF (Prefix %q)", rec.Body.Bytes()[:min(8, rec.Body.Len())])
	}
	if len(rec.Body.Bytes()) < 1500 {
		t.Errorf("PDF verdächtig klein: %d Bytes", rec.Body.Len())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unerfüllte Erwartungen: %s", err)
	}
}

func TestDsgvoAuskunftPDFHandler_MissingID(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	rec := dsgvoPDFRequest(t, mock, "")
	if rec.Code != http.StatusBadRequest {
		t.Errorf("erwartet 400, bekam %d: %s", rec.Code, rec.Body.String())
	}
}

func TestDsgvoAuskunftPDFHandler_NotFound(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	mock.ExpectQuery(`SELECT id, barcode_id, vorname`).
		WithArgs(dsgvoTestID).
		WillReturnRows(pgxmock.NewRows([]string{"id"})) // keine Zeile

	rec := dsgvoPDFRequest(t, mock, dsgvoTestID)
	if rec.Code != http.StatusNotFound {
		t.Errorf("erwartet 404, bekam %d: %s", rec.Code, rec.Body.String())
	}
}
