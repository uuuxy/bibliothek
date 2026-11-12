package leser

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"bibliothek/api/tuer"
	"bibliothek/auth"
)

// Ein abgewiesener Abgleich steht als Warnung im Log, mit dem Grund aus der Antwort: Den sieht
// sonst nur, wer den Abgleich gerade macht, und die Alarm-Mail nennt ihn mit dem Schalter
// alarm_warnungen. Datei und Auswahl werden gelesen, bevor der Abgleich die Datenbank braucht.
func TestLUSDAbweisung_StehtAlsWarnungImLog(t *testing.T) {
	srv := &Tueren{Basis: &tuer.Basis{}}
	admin := &auth.Claims{Rolle: auth.RolleAdmin}
	faelle := []struct {
		name, pfad, inhalt string
		felder             map[string]string
		grund              string
	}{
		{"Pflichtspalte fehlt", "/api/lusd/preview", "vorname,nachname,geburtsdatum\nKind,Nummer01,2012-01-01\n", nil,
			"In der Kopfzeile fehlt die Pflichtspalte „klasse“. Ist das die Exportdatei aus der LUSD?"},
		{"Auswahl unlesbar", "/api/lusd/import", "vorname,nachname,klasse,geburtsdatum\nKind,Nummer01,7a,2012-01-01\n",
			map[string]string{"umbenennungen": "{kaputt"}, meldungUmbenennungUnlesbar},
	}
	for _, fall := range faelle {
		t.Run(fall.name, func(t *testing.T) {
			protokoll := leseLogMit(t)
			rec := ladeLUSDDateiHoch(srv, lusdFormular(t, fall.pfad, fall.inhalt, fall.felder), admin)
			if rec.Code != http.StatusBadRequest {
				t.Fatalf("Status %d, erwartet 400: %s", rec.Code, rec.Body.String())
			}
			if n := warnungenMitGrund(t, protokoll, fall.grund); n != 1 {
				t.Errorf("%d Warnungen „LUSD-Abgleich abgewiesen“ mit dem Grund %q, erwartet 1:\n%s", n, fall.grund, protokoll)
			}
		})
	}
}

// leseLogMit leitet das Log des Tests in einen Puffer und stellt das vorige danach wieder her.
func leseLogMit(t *testing.T) *bytes.Buffer {
	t.Helper()
	var protokoll bytes.Buffer
	vorher := slog.Default()
	slog.SetDefault(slog.New(slog.NewJSONHandler(&protokoll, nil)))
	t.Cleanup(func() { slog.SetDefault(vorher) })
	return &protokoll
}

// warnungenMitGrund zählt die Zeilen „LUSD-Abgleich abgewiesen" der Stufe WARN mit diesem Grund.
func warnungenMitGrund(t *testing.T, protokoll *bytes.Buffer, grund string) int {
	t.Helper()
	n := 0
	for _, zeile := range strings.Split(strings.TrimSpace(protokoll.String()), "\n") {
		var eintrag map[string]any
		if json.Unmarshal([]byte(zeile), &eintrag) != nil {
			continue
		}
		if eintrag["level"] == "WARN" && eintrag["msg"] == "LUSD-Abgleich abgewiesen" && eintrag["fehler"] == grund {
			n++
		}
	}
	return n
}

// lusdFormular baut das Formular des Uploads mit einer CSV-Datei.
func lusdFormular(t *testing.T, pfad, inhalt string, felder map[string]string) *http.Request {
	t.Helper()
	var koerper bytes.Buffer
	mw := multipart.NewWriter(&koerper)
	teil, err := mw.CreateFormFile("csvFile", "lusd_export.csv")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := teil.Write([]byte(inhalt)); err != nil {
		t.Fatal(err)
	}
	for k, v := range felder {
		if err := mw.WriteField(k, v); err != nil {
			t.Fatal(err)
		}
	}
	if err := mw.Close(); err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodPost, pfad, &koerper)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	return req
}
