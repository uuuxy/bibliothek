package leser

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"bibliothek/api/tuer"
	"bibliothek/db"
	"bibliothek/internal/pgtest"
	"bibliothek/internal/testdaten"
	"bibliothek/internal/tuertest"
	"bibliothek/repository"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Das Zusammenführen über seine Route. Die Regeln prüft
// api/student_zusammenfuehren_pg_test.go
// an der Funktion; hier steht, was die Route dazutut: die Antwort, die beiden
// Protokolleinträge mit dem angemeldeten Benutzer und zu jeder Abweisung eine Meldung,
// die den Grund nennt. Beide Dialoge der Oberfläche zeigen diese Meldung unverändert.

// zusammenfuehrenUeberRoute ruft POST /api/schueler/{id}/zusammenfuehren.
func zusammenfuehrenUeberRoute(t *testing.T, srv *Tueren, pool *pgxpool.Pool, benutzerID, zielID, rumpf string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/api/schueler/"+zielID+"/zusammenfuehren", strings.NewReader(rumpf))
	req.SetPathValue("id", zielID)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	srv.ZusammenfuehrenSchuelerHandler(repository.NewAuditRepository(pool))(rec, tuertest.AlsBenutzer(req, benutzerID))
	return rec
}

func zrRumpf(quelleID string) string { return fmt.Sprintf(`{"quelle_id":%q}`, quelleID) }

// zrKonto hängt ein Zugangskonto an die Leserzeile.
func zrKonto(t *testing.T, pool *pgxpool.Pool, leserID string) {
	t.Helper()
	if _, err := pool.Exec(context.Background(), `
		INSERT INTO benutzer (vorname, nachname, email, rolle, aktiv, leser_id)
		VALUES ('Zwei', 'Konten', 'konto-' || $1::uuid::text || '@test.invalid', 'kollegium', true, $1::uuid)`,
		leserID); err != nil {
		t.Fatalf("Konto anlegen: %v", err)
	}
}

func TestZusammenfuehrenRoute_AntwortUndProtokolle(t *testing.T) {
	pool := pgtest.Pool(t)
	testdaten.ResetBestandsdaten(t, pool)
	srv := &Tueren{Basis: &tuer.Basis{DB: &db.Database{Pool: pool}}}
	admin := testdaten.AdminFuerAudit(t, pool)

	ziel := testdaten.LegeUmbSchuelerAn(t, pool, testdaten.UmbSchueler{Vorname: "Alt", Nachname: "Route", Klasse: "07A", Barcode: "ZR-1", Geb: testdaten.Datum(2012, 3, 3), Abgaenger: true})
	quelle := testdaten.LegeUmbSchuelerAn(t, pool, testdaten.UmbSchueler{Vorname: "Neu", Nachname: "Route", Klasse: "08A", Barcode: "ZR-2", Geb: testdaten.Datum(2012, 3, 3)})
	testdaten.SeedOffeneAusleihe(t, pool, quelle, "ZRQ")

	rec := zusammenfuehrenUeberRoute(t, srv, pool, admin, ziel, zrRumpf(quelle))
	if rec.Code != http.StatusOK {
		t.Fatalf("Status %d, erwartet 200: %s", rec.Code, rec.Body.String())
	}
	var erg repository.ZusammenfuehrenErgebnis
	if err := json.Unmarshal(rec.Body.Bytes(), &erg); err != nil {
		t.Fatalf("Antwort nicht lesbar: %v (%s)", err, rec.Body.String())
	}
	if erg.ZielID != ziel || erg.BarcodeID != "ZR-1" || erg.QuelleBarcode != "ZR-2" || erg.Ausleihen != 1 {
		t.Errorf("Antwort %+v, erwartet das Ziel mit ZR-1, die Quelle ZR-2 und eine Ausleihe", erg)
	}
	if n := pgtest.ZaehleZeilen(t, pool, `SELECT count(*) FROM leser WHERE id = $1`, quelle); n != 0 {
		t.Error("die Quelle steht nach der Antwort noch in der Datenbank")
	}

	// Das Admin-Protokoll nennt, welcher Datensatz in welchem aufging und wer es tat.
	if n := pgtest.ZaehleZeilen(t, pool, `
		SELECT count(*) FROM audit_logs
		WHERE aktion = 'SCHUELER_ZUSAMMENGEFUEHRT' AND admin_id = $1
		  AND details->>'schueler_id' = $2 AND details->>'aufgeloest_id' = $3
		  AND details->>'barcode' = 'ZR-1' AND details->>'aufgeloest_barcode' = 'ZR-2'
		  AND details->>'ausleihen' = '1'`, admin, ziel, quelle); n != 1 {
		t.Errorf("%d passende Einträge im Admin-Protokoll, erwartet 1", n)
	}
	// Der Rückweg-Eintrag am Ziel trägt den Benutzer aus der Sitzung.
	if n := pgtest.ZaehleZeilen(t, pool, `
		SELECT count(*) FROM audit_log
		WHERE tabelle = 'schueler' AND aktion = 'ZUSAMMENGEFUEHRT' AND datensatz_id = $1
		  AND bearbeiter_id = $2 AND akteur = 'USER'`, ziel, admin); n != 1 {
		t.Errorf("%d Rückweg-Einträge mit dem angemeldeten Benutzer, erwartet 1", n)
	}
}

// Jede Abweisung der Regeln kommt mit ihrem Grund an und ändert nichts. Eine 500 trüge
// nur „Zusammenführen fehlgeschlagen" nach außen, und wer davorsitzt, wüsste nicht weiter.
func TestZusammenfuehrenRoute_AbweisungenNennenDenGrund(t *testing.T) {
	pool := pgtest.Pool(t)
	srv := &Tueren{Basis: &tuer.Basis{DB: &db.Database{Pool: pool}}}
	ctx := context.Background()

	schueler := func(t *testing.T, nachname, barcode string) string {
		return testdaten.LegeUmbSchuelerAn(t, pool, testdaten.UmbSchueler{Vorname: "Zr", Nachname: nachname, Klasse: "07A", Barcode: barcode, Geb: testdaten.Datum(2012, 5, 5)})
	}
	faelle := []struct {
		name   string
		lage   func(t *testing.T) (ziel, quelle string)
		status int
		grund  string
	}{
		{"mit sich selbst", func(t *testing.T) (string, string) {
			a := schueler(t, "Selbst", "ZR-A1")
			return a, a
		}, http.StatusBadRequest, "mit sich selbst"},
		{"unbekannte Quelle", func(t *testing.T) (string, string) {
			return schueler(t, "Allein", "ZR-A2"), "00000000-0000-0000-0000-000000000000"
		}, http.StatusNotFound, "nicht gefunden"},
		{"Quelle im Papierkorb", func(t *testing.T) (string, string) {
			ziel, quelle := schueler(t, "Bleibt", "ZR-A3"), schueler(t, "Papierkorb", "ZR-A4")
			if _, err := pool.Exec(ctx, `UPDATE leser SET deleted_at = NOW() WHERE id = $1`, quelle); err != nil {
				t.Fatal(err)
			}
			return ziel, quelle
		}, http.StatusNotFound, "Papierkorb"},
		{"anonymisierte Quelle", func(t *testing.T) (string, string) {
			ziel, quelle := schueler(t, "Bleibt", "ZR-A5"), schueler(t, "Anonym", "ZR-A6")
			if _, err := pool.Exec(ctx, `UPDATE leser SET anonymized_at = NOW() WHERE id = $1`, quelle); err != nil {
				t.Fatal(err)
			}
			return ziel, quelle
		}, http.StatusConflict, "anonymisiert"},
		{"Schüler mit Kollege", func(t *testing.T) (string, string) {
			var kollege string
			if err := pool.QueryRow(ctx,
				`INSERT INTO leser (vorname, nachname, art, barcode_id) VALUES ('Zr', 'Kollege', 'lehrkraft', 'ZR-A8') RETURNING id`).Scan(&kollege); err != nil {
				t.Fatalf("Kollege anlegen: %v", err)
			}
			return schueler(t, "Kind", "ZR-A7"), kollege
		}, http.StatusConflict, "Kolleg"},
		{"beide mit Zugangskonto", func(t *testing.T) (string, string) {
			ziel, quelle := schueler(t, "KontoEins", "ZR-A9"), schueler(t, "KontoZwei", "ZR-A10")
			zrKonto(t, pool, ziel)
			zrKonto(t, pool, quelle)
			return ziel, quelle
		}, http.StatusConflict, "Zugangskonto"},
	}
	for _, f := range faelle {
		t.Run(f.name, func(t *testing.T) {
			testdaten.ResetBestandsdaten(t, pool)
			admin := testdaten.AdminFuerAudit(t, pool)
			ziel, quelle := f.lage(t)
			vorher := pgtest.ZaehleZeilen(t, pool, `SELECT count(*) FROM leser`)

			rec := zusammenfuehrenUeberRoute(t, srv, pool, admin, ziel, zrRumpf(quelle))
			if rec.Code != f.status {
				t.Errorf("Status %d, erwartet %d: %s", rec.Code, f.status, rec.Body.String())
			}
			if meldung := tuertest.Fehlermeldung(t, rec); !strings.Contains(meldung, f.grund) {
				t.Errorf("Meldung %q nennt den Grund nicht (erwartet %q darin)", meldung, f.grund)
			}
			if nachher := pgtest.ZaehleZeilen(t, pool, `SELECT count(*) FROM leser`); nachher != vorher {
				t.Errorf("Leserzeilen vorher %d, nachher %d: die Abweisung hat etwas gelöscht", vorher, nachher)
			}
			if n := pgtest.ZaehleZeilen(t, pool, `SELECT count(*) FROM audit_logs WHERE aktion = 'SCHUELER_ZUSAMMENGEFUEHRT'`); n != 0 {
				t.Errorf("das Admin-Protokoll meldet %d Zusammenführungen, erwartet keine", n)
			}
		})
	}
}

// Eine Anfrage ohne brauchbare Quelle erreicht die Datenbank nicht.
func TestZusammenfuehrenRoute_UnbrauchbareAnfrage(t *testing.T) {
	pool := pgtest.Pool(t)
	testdaten.ResetBestandsdaten(t, pool)
	srv := &Tueren{Basis: &tuer.Basis{DB: &db.Database{Pool: pool}}}
	admin := testdaten.AdminFuerAudit(t, pool)
	ziel := testdaten.LegeUmbSchuelerAn(t, pool, testdaten.UmbSchueler{Vorname: "Zr", Nachname: "Ziel", Klasse: "07A", Barcode: "ZR-U1", Geb: testdaten.Datum(2012, 6, 6)})

	for name, rumpf := range map[string]string{
		"Quelle fehlt":      `{}`,
		"Quelle leer":       `{"quelle_id":"   "}`,
		"kein JSON":         `quelle_id`,
		"keine Kennung":     `{"quelle_id":"ZR-U1"}`,
		"Kennung als Liste": `{"quelle_id":["` + ziel + `"]}`,
	} {
		t.Run(name, func(t *testing.T) {
			rec := zusammenfuehrenUeberRoute(t, srv, pool, admin, ziel, rumpf)
			if rec.Code != http.StatusBadRequest {
				t.Errorf("Status %d, erwartet 400: %s", rec.Code, rec.Body.String())
			}
		})
	}
	if n := pgtest.ZaehleZeilen(t, pool, `SELECT count(*) FROM leser WHERE id = $1 AND deleted_at IS NULL`, ziel); n != 1 {
		t.Error("das Ziel steht nach den abgewiesenen Anfragen nicht mehr da")
	}
}
