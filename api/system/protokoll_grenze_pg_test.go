package system

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"bibliothek/api/tuer"
	"bibliothek/db"
	"bibliothek/internal/pgtest"
	"bibliothek/internal/testdaten"
)

// TestProtokolleSindBegrenzt belegt für Logbuch und Verwaltungsprotokoll, dass die Antwort nicht
// die ganze Tabelle trägt, sondern die jüngsten Einträge bis zur Grenze, und dass der Kopf
// tuer.HeaderGrenze genau die Zahl nennt, bei der gekappt wurde. Aus diesem Kopf nimmt die
// Oberfläche den Satz „Die 1000 jüngsten Einträge" (ProtokollGrenze.svelte).
//
// Ohne LIMIT ergäben 247.000 Zeilen eine Antwort von 72 MB: Der Server lieferte sie in
// 0,6 s, aber der Browser käme mit dem Lesen und dem Bauen ebenso vieler Tabellenzeilen nicht
// mehr zurecht, und die Seite bliebe leer. Beide Protokolle wachsen im Betrieb unbegrenzt, der
// Fall tritt also zwangsläufig ein.
//
// Geprüft wird an einer echten Datenbank mit mehr Zeilen, als die Grenze zulässt. Ein
// Test gegen eine kleine Tabelle wäre wertlos — er wäre auch ohne LIMIT grün.
func TestProtokolleSindBegrenzt(t *testing.T) {
	pool := pgtest.Pool(t)
	testdaten.LeereBestandsdaten(t, pool)
	ctx := context.Background()

	// Ein Bearbeiter, an dem der JOIN hängt. Ohne ihn liefert die Abfrage des Logbuchs gar
	// nichts, und der Test wäre auch bei kaputter Grenze grün.
	var bearbeiterID string
	if err := pool.QueryRow(ctx, `
		INSERT INTO benutzer (vorname, nachname, email, rolle, aktiv)
		VALUES ('Audit', 'Limit', 'audit-limit@test.invalid', 'admin', true)
		ON CONFLICT (email) DO UPDATE SET vorname = EXCLUDED.vorname
		RETURNING id
	`).Scan(&bearbeiterID); err != nil {
		t.Fatalf("Bearbeiter anlegen: %v", err)
	}
	// Über die Kennung des Bearbeiters: Die Zeilen gehören nur diesem Test.
	t.Cleanup(func() {
		for _, loesche := range []string{
			`DELETE FROM audit_log WHERE bearbeiter_id = $1`,
			`DELETE FROM audit_logs WHERE admin_id = $1`,
		} {
			if _, err := pool.Exec(ctx, loesche, bearbeiterID); err != nil {
				t.Errorf("aufräumen: %v", err)
			}
		}
	})

	srv := &Tueren{Basis: &tuer.Basis{DB: &db.Verbindung{Pool: pool}}}
	const ueberschuss = protokollMaxZeilen + 250

	// Je Protokoll absichtlich mehr Zeilen als die Grenze, mit fallenden Zeitstempeln.
	faelle := []struct {
		name, fuelle, url, zeitFeld string
		tuer                        http.HandlerFunc
	}{
		{
			name: "Logbuch",
			fuelle: `
				INSERT INTO audit_log (tabelle, aktion, datensatz_id, bearbeiter_id, akteur, timestamp)
				SELECT 'buecher', 'DELETE', gen_random_uuid(), $1, 'USER',
				       CURRENT_TIMESTAMP - (g || ' seconds')::interval
				FROM generate_series(1, $2) AS g`,
			url: "/api/audit", zeitFeld: "timestamp", tuer: srv.Protokoll(),
		},
		{
			name: "Verwaltungsprotokoll",
			fuelle: `
				INSERT INTO audit_logs (admin_id, aktion, details, zeitstempel)
				SELECT $1, 'BENUTZER_GEAENDERT', '{}',
				       CURRENT_TIMESTAMP - (g || ' seconds')::interval
				FROM generate_series(1, $2) AS g`,
			url: "/api/admin/auditlog", zeitFeld: "zeitstempel", tuer: srv.VerwaltungsProtokoll(),
		},
	}

	for _, f := range faelle {
		t.Run(f.name, func(t *testing.T) {
			if _, err := pool.Exec(ctx, f.fuelle, bearbeiterID, ueberschuss); err != nil {
				t.Fatalf("Zeilen anlegen: %v", err)
			}

			rec := httptest.NewRecorder()
			f.tuer(rec, httptest.NewRequest(http.MethodGet, f.url, nil))
			if rec.Code != http.StatusOK {
				t.Fatalf("Status %d, erwartet 200 — Antwort: %s", rec.Code, rec.Body.String())
			}

			var eintraege []map[string]any
			if err := json.Unmarshal(rec.Body.Bytes(), &eintraege); err != nil {
				t.Fatalf("Antwort lesen: %v", err)
			}
			if len(eintraege) != protokollMaxZeilen {
				t.Fatalf("%d Einträge geliefert, erwartet genau %d — ohne LIMIT wären es mindestens %d",
					len(eintraege), protokollMaxZeilen, ueberschuss)
			}

			// Der Kopf nennt die Zahl, bei der die Abfrage kappte. Nennte er eine andere, behauptete
			// die Oberfläche eine falsche Grenze oder schwiege über eine gekappte Liste.
			if kopf := rec.Header().Get(tuer.HeaderGrenze); kopf != strconv.Itoa(len(eintraege)) {
				t.Fatalf("Kopf %s = %q, die Antwort ist bei %d Einträgen gekappt",
					tuer.HeaderGrenze, kopf, len(eintraege))
			}

			// Und es sind die jüngsten. Eine Grenze, die die ältesten liefert, wäre schlimmer als
			// keine: Das Protokoll zeigte dauerhaft nur die Vergangenheit, und niemand bemerkte es.
			erster, zweiter := zeitstempelAus(t, eintraege[0], f.zeitFeld), zeitstempelAus(t, eintraege[1], f.zeitFeld)
			if erster.Before(zweiter) {
				t.Fatalf("Reihenfolge falsch: erster Eintrag (%s) ist älter als der zweite (%s)", erster, zweiter)
			}
		})
	}
}

// zeitstempelAus liest den Zeitstempel eines Eintrags aus dem Feld, das das jeweilige Protokoll
// dafür führt.
func zeitstempelAus(t *testing.T, eintrag map[string]any, feld string) time.Time {
	t.Helper()
	text, ok := eintrag[feld].(string)
	if !ok {
		t.Fatalf("Feld %q fehlt oder ist kein Text: %v", feld, eintrag[feld])
	}
	zeit, err := time.Parse(time.RFC3339Nano, text)
	if err != nil {
		t.Fatalf("Feld %q: %v", feld, err)
	}
	return zeit
}
