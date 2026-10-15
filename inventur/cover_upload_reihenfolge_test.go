package inventur

import (
	"bytes"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/pashagolub/pgxmock/v5"
)

// Reihenfolge beim Hochladen eines Covers: Schriebe die Tür die neue Datei, bevor feststeht, dass
// es den Titel gibt, und löschte sie das alte Cover, bevor das UPDATE gelungen ist, antwortete
// POST /api/books/<unbekannte UUID>/cover-upload 404, und die Datei läge trotzdem in uploads/.
// Scheiterte das UPDATE, zeigte die Datenbank auf ein Cover, das gerade gelöscht wurde.

const reihenfolgeBuchID = "9f1c2b3a-4d5e-4f60-8a7b-1c2d3e4f5a6b"

var reihenfolgeBuchSpalten = []string{
	"id", "isbn", "title", "author", "signatur", "cover_url", "subject", "track", "stock",
	"last_counted", "sort_order", "medientyp", "jahrgang_von", "jahrgang_bis", "erweiterte_eigenschaften", "auflage",
}

func coverUploadAnfrage(t *testing.T, id string) *http.Request {
	t.Helper()
	var rumpf bytes.Buffer
	w := multipart.NewWriter(&rumpf)
	feld, err := w.CreateFormFile("cover", "neu.jpg")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := feld.Write(erzeugeTestbild("jpeg", 20, 20)); err != nil {
		t.Fatal(err)
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	anfrage := httptest.NewRequest(http.MethodPost, "/api/books/"+id+"/cover-upload", &rumpf)
	anfrage.Header.Set("Content-Type", w.FormDataContentType())
	anfrage.SetPathValue("id", id)
	return anfrage
}

// coverDateien: alles, was in uploads/ zu dieser Kennung liegt (cover_<id>_…). Aufgeräumt
// wird am Testende, damit ein roter Lauf den nächsten nicht mitfärbt.
func coverDateien(t *testing.T, id string) []string {
	t.Helper()
	treffer, err := filepath.Glob(filepath.Join(uploadsVerzeichnis, "cover_"+id+"_*"))
	if err != nil {
		t.Fatal(err)
	}
	return treffer
}

func raeumeCoverDateienAuf(t *testing.T, id string) {
	t.Cleanup(func() {
		treffer, _ := filepath.Glob(filepath.Join(uploadsVerzeichnis, "cover_"+id+"_*")) //nolint:errcheck
		for _, datei := range treffer {
			_ = os.Remove(datei) //nolint:errcheck
		}
	})
}

// altesCover legt ein echtes Cover in uploads/ an und liefert die Zeile, mit der
// LadeTitel darauf zeigt.
func altesCover(t *testing.T, id string) (name string, zeile *pgxmock.Rows) {
	t.Helper()
	name = "cover_" + id + "_alt.jpg"
	if err := schreibeUploadDatei(name, []byte("alt")); err != nil {
		t.Fatal(err)
	}
	zeile = pgxmock.NewRows(reihenfolgeBuchSpalten).AddRow(
		id, "9781234567890", "Titel", "Autor", "", "/uploads/"+name, "", "", 1, nil, 1, "Buch", 5, 10, nil, "",
	)
	return name, zeile
}

func TestCoverUpload_UnbekanntesBuchHinterlaesstKeineDatei(t *testing.T) {
	raeumeCoverDateienAuf(t, reihenfolgeBuchID)
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock: %v", err)
	}
	defer mock.Close()
	mock.ExpectQuery("(?s)SELECT id, COALESCE.*").WithArgs(reihenfolgeBuchID).WillReturnError(pgx.ErrNoRows)

	rec := httptest.NewRecorder()
	(&Tueren{abfragen: NewTitelAbfragen(mock)}).CoverHochladen(rec, coverUploadAnfrage(t, reihenfolgeBuchID))

	if rec.Code != http.StatusNotFound {
		t.Errorf("Status %d, erwartet 404: %s", rec.Code, rec.Body.String())
	}
	if dateien := coverDateien(t, reihenfolgeBuchID); len(dateien) > 0 {
		t.Errorf("verwaiste Datei für ein Buch, das es nicht gibt: %v", dateien)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("Datenbank: %v", err)
	}
}

func TestCoverUpload_GescheitertesUpdateLaesstAltesCoverStehen(t *testing.T) {
	raeumeCoverDateienAuf(t, reihenfolgeBuchID)
	alt, zeile := altesCover(t, reihenfolgeBuchID)
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock: %v", err)
	}
	defer mock.Close()
	mock.ExpectQuery("(?s)SELECT id, COALESCE.*").WithArgs(reihenfolgeBuchID).WillReturnRows(zeile)
	mock.ExpectExec("(?s)UPDATE buecher_titel.*").
		WithArgs("", "", pgxmock.AnyArg(), reihenfolgeBuchID).
		WillReturnError(errTest)

	rec := httptest.NewRecorder()
	(&Tueren{abfragen: NewTitelAbfragen(mock)}).CoverHochladen(rec, coverUploadAnfrage(t, reihenfolgeBuchID))

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("Status %d, erwartet 500: %s", rec.Code, rec.Body.String())
	}
	dateien := coverDateien(t, reihenfolgeBuchID)
	if len(dateien) != 1 || filepath.Base(dateien[0]) != alt {
		t.Errorf("nach gescheitertem UPDATE erwartet: nur das alte Cover %s, gefunden: %v", alt, dateien)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("Datenbank: %v", err)
	}
}

// Gegenprobe: Gelingt das UPDATE, ersetzt das neue Cover das alte.
func TestCoverUpload_ErfolgErsetztAltesCover(t *testing.T) {
	raeumeCoverDateienAuf(t, reihenfolgeBuchID)
	alt, zeile := altesCover(t, reihenfolgeBuchID)
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock: %v", err)
	}
	defer mock.Close()
	mock.ExpectQuery("(?s)SELECT id, COALESCE.*").WithArgs(reihenfolgeBuchID).WillReturnRows(zeile)
	mock.ExpectExec("(?s)UPDATE buecher_titel.*").
		WithArgs("", "", pgxmock.AnyArg(), reihenfolgeBuchID).
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))

	rec := httptest.NewRecorder()
	(&Tueren{abfragen: NewTitelAbfragen(mock)}).CoverHochladen(rec, coverUploadAnfrage(t, reihenfolgeBuchID))

	if rec.Code != http.StatusOK {
		t.Fatalf("Status %d, erwartet 200: %s", rec.Code, rec.Body.String())
	}
	dateien := coverDateien(t, reihenfolgeBuchID)
	if len(dateien) != 1 || filepath.Base(dateien[0]) == alt {
		t.Errorf("nach Erfolg erwartet: genau das neue Cover, gefunden: %v", dateien)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("Datenbank: %v", err)
	}
}
