package leser

import (
	"bytes"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"bibliothek/api/tuer"
	"bibliothek/auth"
	"bibliothek/db"
	"bibliothek/internal/lusd"
	"bibliothek/internal/pgtest"
	"bibliothek/internal/testdaten"
	"bibliothek/internal/testweg"
	"bibliothek/internal/xlsxtest"
)

// Eine Klassenliste mit einem Blatt je Klasse, durch Vorschau und Übernahme: Anna steht in
// Zeile 2 ihres Blatts, Cem in Zeile 2 des anderen. Bestätigt der Admin das Paar für Anna, trägt
// Annas Datensatz danach ihren neuen Namen, und Cem kommt als neuer Schüler dazu. Beide
// Reihenfolgen der Blätter, weil eine Verwechslung nur in einer von beiden auffiele.
func TestLUSDUmbenennung_MappeMitEinemBlattJeKlasse(t *testing.T) {
	pool := pgtest.Pool(t)
	s := &Tueren{Basis: &tuer.Basis{DB: &db.Verbindung{Pool: pool}}}
	geb, eintritt := testdaten.Datum(2013, 5, 4), testdaten.Datum(2024, 8, 19)
	kopf := []any{"Schueler_Nachname", "Schueler_Vorname", "Klassen_Klassenbezeichnung", "Schueler_Geburtsdatum", "Schueler_Eintritt_AktuelleSchule"}

	for _, annaZuerst := range []bool{true, false} {
		name := "Anna im zweiten Blatt"
		if annaZuerst {
			name = "Anna im ersten Blatt"
		}
		t.Run(name, func(t *testing.T) {
			testdaten.LeereBestandsdaten(t, pool)
			adminID := testdaten.LegeKontoAn(t, pool, "Ada", "Admin", "ada@blaetter.invalid", "admin", true)
			admin := &auth.Claims{KontoID: adminID, Rolle: auth.RolleAdmin}
			annaID := testweg.LegeLUSDSchuelerAn(t, pool, testweg.LUSDSchuelerNeu{Vorname: "Anna", Nachname: "Müller", Klasse: "05F1", Barcode: "BLATT-1", Geburtsdatum: geb, SchulEintrittAm: eintritt})

			blattAnna := xlsxtest.Blatt{Name: "06F1", Zeilen: [][]any{kopf, {"Mueller-Schmidt", "Anna", "06F1", *geb, *eintritt}}}
			blattCem := xlsxtest.Blatt{Name: "06F2", Zeilen: [][]any{kopf, {"Cato", "Cem", "06F2", time.Date(2013, 3, 3, 0, 0, 0, 0, time.UTC), *eintritt}}}
			// Unbeteiligte, damit die Bremse gegen den Massenabgang nicht anspringt.
			for i := 0; i < 12; i++ {
				nachname := "Kind" + string(rune('A'+i))
				gebKind := testdaten.Datum(2012, 1, 1+i)
				testweg.LegeLUSDSchuelerAn(t, pool, testweg.LUSDSchuelerNeu{Vorname: "Ruhig", Nachname: nachname, Klasse: "07A", Barcode: "BLATT-R" + string(rune('A'+i)), Geburtsdatum: gebKind})
				zeile := []any{nachname, "Ruhig", "08A", *gebKind, ""}
				if i%2 == 0 {
					blattAnna.Zeilen = append(blattAnna.Zeilen, zeile)
				} else {
					blattCem.Zeilen = append(blattCem.Zeilen, zeile)
				}
			}
			blaetter := []xlsxtest.Blatt{blattAnna, blattCem}
			if !annaZuerst {
				blaetter = []xlsxtest.Blatt{blattCem, blattAnna}
			}
			xlsx := xlsxtest.BaueInReihenfolge(t, blaetter...)

			rec := sendeLUSDMappe(t, s, admin, "/api/lusd/preview", xlsx, "")
			if rec.Code != http.StatusOK {
				t.Fatalf("Vorschau = %d: %s", rec.Code, rec.Body.String())
			}
			var vorschau lusd.Vorschau
			if err := json.Unmarshal(rec.Body.Bytes(), &vorschau); err != nil {
				t.Fatal(err)
			}
			kennungen := map[string]bool{}
			for _, neu := range vorschau.Neuzugaenge {
				if kennungen[neu.ID] {
					t.Errorf("Kennung %q steht zweimal unter den Neuzugängen: %+v", neu.ID, vorschau.Neuzugaenge)
				}
				kennungen[neu.ID] = true
			}
			if len(vorschau.Umbenennungen) != 1 || vorschau.Umbenennungen[0].SchuelerID != annaID {
				t.Fatalf("Vorschau schlägt %+v vor, erwartet ein Paar für Anna", vorschau.Umbenennungen)
			}
			wahl, err := json.Marshal([]lusd.UmbenennungWahl{{Zeile: vorschau.Umbenennungen[0].Zeile, SchuelerID: annaID}})
			if err != nil {
				t.Fatal(err)
			}

			if rec := sendeLUSDMappe(t, s, admin, "/api/lusd/import", xlsx, string(wahl)); rec.Code != http.StatusOK {
				t.Fatalf("Übernahme = %d: %s", rec.Code, rec.Body.String())
			}
			var vorname, nachname, klasse string
			if err := pool.QueryRow(t.Context(), `SELECT vorname, nachname, klasse FROM schueler WHERE id = $1`, annaID).Scan(&vorname, &nachname, &klasse); err != nil {
				t.Fatal(err)
			}
			if vorname != "Anna" || nachname != "Mueller-Schmidt" || !lusd.IstKlasseGleich(klasse, "06F1") {
				t.Errorf("Annas Datensatz trägt %s %s, %s; erwartet Anna Mueller-Schmidt, 06F1", vorname, nachname, klasse)
			}
			if n := zaehle(t, pool, "nachname = 'Cato' AND id <> $1", annaID); n != 1 {
				t.Errorf("%d eigene Datensätze für Cem, erwartet 1", n)
			}
			if n := zaehle(t, pool, "nachname = 'Mueller-Schmidt'"); n != 1 {
				t.Errorf("%d Datensätze für Anna Mueller-Schmidt, erwartet 1", n)
			}
		})
	}
}

// sendeLUSDMappe schickt eine Excel-Mappe an Vorschau oder Übernahme, wie die Oberfläche es tut:
// die Datei im Feld csvFile, die bestätigten Paare im Feld umbenennungen.
func sendeLUSDMappe(t *testing.T, s *Tueren, admin *auth.Claims, pfad string, xlsx []byte, umbenennungen string) *httptest.ResponseRecorder {
	t.Helper()
	var koerper bytes.Buffer
	mw := multipart.NewWriter(&koerper)
	teil, err := mw.CreateFormFile("csvFile", "klassenlisten.xlsx")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := teil.Write(xlsx); err != nil {
		t.Fatal(err)
	}
	if umbenennungen != "" {
		if err := mw.WriteField("umbenennungen", umbenennungen); err != nil {
			t.Fatal(err)
		}
	}
	if err := mw.Close(); err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodPost, pfad, &koerper)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	return ladeLUSDDateiHoch(s, req, admin)
}
