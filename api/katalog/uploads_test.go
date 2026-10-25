package katalog

import (
	"bibliothek/api/tuer"
	"bibliothek/db"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pashagolub/pgxmock/v5"
)

// TestNeuteredFileSystem_Open hält die Aufgabe dieses Dateisystems fest: Ein Verzeichnis
// lässt sich nicht öffnen, mit oder ohne index.html. Öffnet es sich, zeigt der
// Go-Dateiserver an seiner Stelle die Dateinamen darin als Liste.
func TestNeuteredFileSystem_Open(t *testing.T) {
	// t.TempDir raeumt selbst auf — kein deferter RemoveAll, dessen Fehler niemand liest.
	testVerzeichnis := t.TempDir()

	if err := os.WriteFile(filepath.Join(testVerzeichnis, "file.txt"), []byte("test"), 0644); err != nil {
		t.Fatalf("Datei anlegen: %v", err)
	}

	unterordnerMitIndex := filepath.Join(testVerzeichnis, "with_index")
	if err := os.Mkdir(unterordnerMitIndex, 0755); err != nil {
		t.Fatalf("Verzeichnis anlegen: %v", err)
	}
	if err := os.WriteFile(filepath.Join(unterordnerMitIndex, "index.html"), []byte("index"), 0644); err != nil {
		t.Fatalf("index.html anlegen: %v", err)
	}

	unterordnerOhneIndex := filepath.Join(testVerzeichnis, "without_index")
	if err := os.Mkdir(unterordnerOhneIndex, 0755); err != nil {
		t.Fatalf("Verzeichnis anlegen: %v", err)
	}

	fs := neuteredFileSystem{http.Dir(testVerzeichnis)}

	// schliesse macht aus dem Close ein gepruefte Handlung: bliebe hier ein Handle
	// offen, faende man die Ursache spaeter nur noch als Datei-Limit im Betrieb.
	schliesse := func(t *testing.T, f http.File) {
		t.Helper()
		if f == nil {
			return
		}
		if err := f.Close(); err != nil {
			t.Errorf("Schliessen fehlgeschlagen: %v", err)
		}
	}

	t.Run("vorhandene Datei", func(t *testing.T) {
		f, err := fs.Open("file.txt")
		if err != nil {
			t.Errorf("fs.Open(…) = %v, erwartet kein Fehler", err)
		}
		schliesse(t, f)
	})

	t.Run("nicht vorhandene Datei", func(t *testing.T) {
		_, err := fs.Open("nonexistent.txt")
		if err == nil {
			t.Error("fs.Open(\"nonexistent.txt\") = nil, erwartet einen Fehler")
		}
	})

	t.Run("Verzeichnis mit index.html", func(t *testing.T) {
		f, err := fs.Open("with_index")
		if err == nil {
			schliesse(t, f)
			t.Fatal("das Verzeichnis liess sich oeffnen — der Dateiserver zeigt dann seine Dateinamen als Liste")
		}
		if !errors.Is(err, os.ErrNotExist) {
			t.Errorf("fs.Open(…) = %v, erwartet os.ErrNotExist", err)
		}
	})

	t.Run("Open directory without index.html", func(t *testing.T) {
		f, err := fs.Open("without_index")
		if err == nil {
			schliesse(t, f)
			t.Fatal("das Verzeichnis liess sich oeffnen — der Dateiserver wuerde jetzt ein Listing ausliefern")
		}
		// Es muss ein „existiert nicht" sein: nur daraus macht net/http ein 404.
		// Ein anderer Fehler wuerde als 500 durchschlagen und die SPA-Weiche stoeren.
		if !errors.Is(err, os.ErrNotExist) {
			t.Errorf("fs.Open(…) = %v, erwartet os.ErrNotExist", err)
		}
	})
}

// /uploads/ ist ohne Anmeldung lesbar. Ausgeliefert wird deshalb nur, was direkt im
// Verzeichnis liegt: Dort legen Upload und Cover-Abruf ihre Dateien ab.
func TestUploadsLiefertNurDateienDirektImVerzeichnis(t *testing.T) {
	verzeichnis := t.TempDir()
	t.Chdir(verzeichnis)
	lege := func(rel, inhalt string) {
		t.Helper()
		pfad := filepath.Join(verzeichnis, rel)
		if err := os.MkdirAll(filepath.Dir(pfad), 0o750); err != nil {
			t.Fatalf("Verzeichnis anlegen: %v", err)
		}
		if err := os.WriteFile(pfad, []byte(inhalt), 0o600); err != nil {
			t.Fatalf("Datei anlegen: %v", err)
		}
	}
	lege("uploads/cover_1.webp", "cover-inhalt")
	lege("uploads/fotos/S-10041.jpg", "foto-inhalt")
	// Eine index.html im Unterordner öffnet das Verzeichnis nicht: Der Dateiserver zeigte
	// sonst die Dateinamen darin als Liste.
	lege("uploads/fotos/index.html", "index-inhalt")
	lege("uploads/covers/abc.webp", "zwischenspeicher-inhalt")

	tueren := katalogMux(&Tueren{})

	faelle := []struct {
		pfad   string
		status int
	}{
		{"/uploads/cover_1.webp", http.StatusOK},
		{"/uploads/fotos/S-10041.jpg", http.StatusNotFound},
		{"/uploads/covers/abc.webp", http.StatusNotFound},
		{"/uploads/fotos/", http.StatusNotFound},
		{"/uploads/fotos", http.StatusNotFound},
		{"/uploads/", http.StatusNotFound},
	}
	for _, f := range faelle {
		rec := httptest.NewRecorder()
		tueren.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, f.pfad, nil))
		if rec.Code != f.status {
			t.Errorf("GET %s: Status %d, erwartet %d", f.pfad, rec.Code, f.status)
		}
		// Der Treffer belegt, dass der Server aus diesem Verzeichnis liest — sonst wären
		// die 404 darunter auch ohne die Regel grün.
		if f.status == http.StatusOK && rec.Body.String() != "cover-inhalt" {
			t.Errorf("GET %s: Rumpf %q, erwartet den Inhalt der Datei", f.pfad, rec.Body.String())
		}
		if f.status == http.StatusNotFound && strings.Contains(rec.Body.String(), "-inhalt") {
			t.Errorf("GET %s: die Antwort trägt den Inhalt einer Datei aus einem Unterordner", f.pfad)
		}
		if f.status == http.StatusNotFound && strings.Contains(rec.Body.String(), "S-10041") {
			t.Errorf("GET %s: die Antwort nennt eine Datei aus einem Unterordner", f.pfad)
		}
	}
}

// Was der Upload als Adresse zurückgibt, liefert /uploads/ aus. Die Regel oben hängt daran, dass
// er direkt in uploads/ ablegt: Mit einer Adresse im Unterordner zeigte die Oberfläche für den
// Titel kein Cover. Für den Cover-Abruf hält es TestSpeichereCoverLokal bei den Katalogdiensten.
func TestUploadsLiefertWasDerUploadAblegt(t *testing.T) {
	t.Chdir(t.TempDir())

	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock: %v", err)
	}
	defer mock.Close()
	zeile := pgxmock.NewRows(reihenfolgeBuchSpalten).AddRow(
		reihenfolgeBuchID, "9781234567890", "Titel", "Autor", "", "", "", "", 1, nil, 1, "Buch", 5, 10, nil, "",
	)
	mock.ExpectQuery("(?s)SELECT id, COALESCE.*").WithArgs(reihenfolgeBuchID).WillReturnRows(zeile)
	mock.ExpectExec("(?s)UPDATE buecher_titel.*").
		WithArgs("", "", pgxmock.AnyArg(), reihenfolgeBuchID).
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))

	upload := httptest.NewRecorder()
	(&Tueren{Basis: &tuer.Basis{DB: &db.Verbindung{Pool: mock}}}).CoverHochladen()(upload, coverUploadAnfrage(t, reihenfolgeBuchID))
	if upload.Code != http.StatusOK {
		t.Fatalf("Upload: Status %d, erwartet 200: %s", upload.Code, upload.Body.String())
	}
	var antwort struct {
		Data struct {
			CoverURL string `json:"coverUrl"`
		} `json:"data"`
	}
	if err := json.Unmarshal(upload.Body.Bytes(), &antwort); err != nil {
		t.Fatalf("Upload: Antwort nicht lesbar: %v", err)
	}

	tueren := katalogMux(&Tueren{})
	if antwort.Data.CoverURL == "" {
		t.Fatal("Upload: keine Adresse — ohne sie misst der Test nichts")
	}
	rec := httptest.NewRecorder()
	tueren.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, antwort.Data.CoverURL, nil))
	if rec.Code != http.StatusOK || rec.Body.Len() == 0 {
		t.Errorf("Upload legte %s ab, der Abruf darauf: Status %d, %d Bytes — die Oberfläche zeigt für diesen Titel kein Cover",
			antwort.Data.CoverURL, rec.Code, rec.Body.Len())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("Datenbank: %v", err)
	}
}
