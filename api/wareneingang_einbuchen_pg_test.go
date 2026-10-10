package api

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

// POST /api/bestellungen/bulk-receive bucht markierte Exemplare aus dem Zulauf in den
// Bestand: ausleihbar, ohne Bestellstatus, ohne die Notiz „Im Zulauf". Gemessen am
// 07.10.2026 führte kein Go-Test mehr als 4,3 % dieses Wegs aus (OFFEN.md 5.10).
//
// Über den ganzen Router mit Sitzung. Was die Tür ablehnt, darf nichts gebucht haben.

// zulaufExemplarMitNotiz legt ein bestelltes Exemplar an, wie der Bestellweg es tut.
func zulaufExemplarMitNotiz(t *testing.T, pool *pgxpool.Pool, titelID, barcode string) string {
	t.Helper()
	var id string
	if err := pool.QueryRow(t.Context(), `
		INSERT INTO buecher_exemplare (titel_id, barcode_id, ist_ausleihbar, zustand_notiz, bestellstatus)
		VALUES ($1, $2, false, 'Im Zulauf - Testhändler', 'im_zulauf') RETURNING id`,
		titelID, barcode).Scan(&id); err != nil {
		t.Fatalf("Zulauf-Exemplar %s: %v", barcode, err)
	}
	return id
}

// exemplarStand liest, was das Einbuchen an einem Exemplar ändert.
type exemplarStand struct {
	ausleihbar, ausgesondert bool
	bestellstatus, notiz     string
}

func leseExemplarStand(t *testing.T, pool *pgxpool.Pool, id string) exemplarStand {
	t.Helper()
	var s exemplarStand
	if err := pool.QueryRow(t.Context(), `
		SELECT ist_ausleihbar, ist_ausgesondert, coalesce(bestellstatus, ''), coalesce(zustand_notiz, '')
		FROM buecher_exemplare WHERE id = $1`, id).
		Scan(&s.ausleihbar, &s.ausgesondert, &s.bestellstatus, &s.notiz); err != nil {
		t.Fatalf("Exemplar %s lesen: %v", id, err)
	}
	return s
}

func TestWareneingang_EinbuchenUeberDieTuer(t *testing.T) {
	pool := pgTestPool(t)
	resetBestandsdaten(t, pool)
	adminID, rufe := protokollWelt(t, pool)
	t.Cleanup(func() {
		aufraeumen(t, pool, `DELETE FROM audit_logs WHERE aktion = 'BULK_RECEIVE_ITEMS'`)
	})
	titelID := titelMitSignatur(t, pool, "Einbuch-Titel", "Ein 1", 0)

	// einbuchen liefert Status, die gemeldete Zahl und den Rumpf für die Fehlermeldung.
	einbuchen := func(t *testing.T, rumpf string) (int, int, string) {
		t.Helper()
		rec := rufe(t, http.MethodPost, "/api/bestellungen/bulk-receive", rumpf)
		var antwort struct {
			ReceivedCount int `json:"received_count"`
		}
		if rec.Code == http.StatusOK {
			if err := json.Unmarshal(rec.Body.Bytes(), &antwort); err != nil {
				t.Fatalf("Antwort unlesbar (%s): %v", rec.Body.String(), err)
			}
		}
		return rec.Code, antwort.ReceivedCount, rec.Body.String()
	}

	t.Run("zwei Exemplare aus dem Zulauf kommen in den Bestand", func(t *testing.T) {
		a := zulaufExemplarMitNotiz(t, pool, titelID, "EIN-A1")
		b := zulaufExemplarMitNotiz(t, pool, titelID, "EIN-A2")
		code, anzahl, rumpf := einbuchen(t, `{"exemplar_ids":["`+a+`","`+b+`"]}`)
		if code != http.StatusOK {
			t.Fatalf("Status %d, erwartet 200: %s", code, rumpf)
		}
		if anzahl != 2 {
			t.Errorf("gemeldet %d eingebucht, erwartet 2", anzahl)
		}
		for _, id := range []string{a, b} {
			s := leseExemplarStand(t, pool, id)
			if !s.ausleihbar || s.bestellstatus != "" || s.notiz != "" {
				t.Errorf("Exemplar %s nach dem Einbuchen: %+v — erwartet ausleihbar, ohne Bestellstatus und Notiz", id, s)
			}
		}
		if n := zaehleZeilen(t, pool,
			`SELECT count(*) FROM audit_logs WHERE aktion = 'BULK_RECEIVE_ITEMS' AND admin_id = $1
			   AND details->>'received_count' = '2'`,
			adminID); n != 1 {
			t.Errorf("%d Protokolleinträge mit der Person und der Zahl 2, erwartet 1", n)
		}
	})

	// Ein ausgesondertes Exemplar (der Händler liefert nicht) kann noch in einer offenen
	// Liste stehen, wenn ein anderer Platz es inzwischen ausgesondert hat. Es darf beim
	// Einbuchen nicht wieder aufleben und seine Notiz nicht verlieren. Aussondern nimmt
	// den Bestellstatus weg (chk_exemplar_bestellstatus_nur_im_zulauf) — so steht es auch hier.
	t.Run("ein ausgesondertes Exemplar in der Auswahl bleibt ausgesondert", func(t *testing.T) {
		zulauf := zulaufExemplarMitNotiz(t, pool, titelID, "EIN-B1")
		weg := zulaufExemplarMitNotiz(t, pool, titelID, "EIN-B2")
		if _, err := pool.Exec(t.Context(), `
			UPDATE buecher_exemplare SET ist_ausgesondert = true, aussonderung_grund = 'AUSSORTIERT',
			       bestellstatus = NULL, zustand_notiz = 'Händler liefert nicht'
			WHERE id = $1`, weg); err != nil {
			t.Fatal(err)
		}
		code, anzahl, rumpf := einbuchen(t, `{"exemplar_ids":["`+zulauf+`","`+weg+`"]}`)
		if code != http.StatusOK {
			t.Fatalf("Status %d, erwartet 200: %s", code, rumpf)
		}
		if anzahl != 1 {
			t.Errorf("gemeldet %d eingebucht, erwartet 1", anzahl)
		}
		s := leseExemplarStand(t, pool, weg)
		if !s.ausgesondert || s.ausleihbar || s.notiz != "Händler liefert nicht" {
			t.Errorf("das ausgesonderte Exemplar wurde angefasst: %+v", s)
		}
	})

	// Nichts mehr zu buchen (ein zweiter Platz war schneller, oder die Liste ist veraltet)
	// ist eine Auskunft, kein Serverfehler.
	t.Run("schon eingebucht: 404, nichts geändert", func(t *testing.T) {
		a := zulaufExemplarMitNotiz(t, pool, titelID, "EIN-C1")
		if code, _, _ := einbuchen(t, `{"exemplar_ids":["`+a+`"]}`); code != http.StatusOK {
			t.Fatalf("erstes Einbuchen: Status %d", code)
		}
		vorher := zaehleZeilen(t, pool,
			`SELECT count(*) FROM audit_logs WHERE aktion = 'BULK_RECEIVE_ITEMS'`)
		code, _, rumpf := einbuchen(t, `{"exemplar_ids":["`+a+`"]}`)
		if code != http.StatusNotFound {
			t.Fatalf("zweites Einbuchen: Status %d, erwartet 404: %s", code, rumpf)
		}
		if nachher := zaehleZeilen(t, pool,
			`SELECT count(*) FROM audit_logs WHERE aktion = 'BULK_RECEIVE_ITEMS'`); nachher != vorher {
			t.Errorf("ein Einbuchen ohne Wirkung schrieb einen Protokolleintrag (%d → %d)", vorher, nachher)
		}
	})

	t.Run("leere Auswahl: 400", func(t *testing.T) {
		if code, _, rumpf := einbuchen(t, `{"exemplar_ids":[]}`); code != http.StatusBadRequest {
			t.Errorf("Status %d, erwartet 400: %s", code, rumpf)
		}
	})

	// Die Prüfung des Rumpfs lässt je Kennung „UUID oder leer" zu. Eine leere Kennung in der
	// Liste ist eine Eingabe, die nicht stimmt — eine Auskunft, kein Serverfehler.
	t.Run("leere Kennung in der Auswahl: 400", func(t *testing.T) {
		if code, _, rumpf := einbuchen(t, `{"exemplar_ids":[""]}`); code != http.StatusBadRequest {
			t.Errorf("Status %d, erwartet 400: %s", code, rumpf)
		}
	})
}
