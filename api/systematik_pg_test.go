package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"bibliothek/db"

	"github.com/jackc/pgx/v5/pgxpool"
)

// systematikAnlegen ruft den Anlege-Endpunkt auf und liefert Recorder + neue ID.
func systematikAnlegen(t *testing.T, srv *Server, kuerzel, bezeichnung string) (*httptest.ResponseRecorder, string) {
	t.Helper()
	koerper, err := json.Marshal(map[string]string{"kuerzel": kuerzel, "bezeichnung": bezeichnung})
	if err != nil {
		t.Fatalf("Anfrage kodieren: %v", err)
	}
	rec := httptest.NewRecorder()
	srv.CreateSystematikHandler().ServeHTTP(rec,
		httptest.NewRequest(http.MethodPost, "/api/systematics", bytes.NewReader(koerper)))

	// Fehlerantworten tragen kein id-Feld — das ist erwartbar und kein Testfehler.
	var antwort map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &antwort); err != nil {
		return rec, ""
	}
	return rec, antwort["id"]
}

// aufraeumen fuehrt eine Cleanup-Anweisung aus und meldet Fehler, statt sie zu schlucken.
func aufraeumen(t *testing.T, pool *pgxpool.Pool, sql string, args ...any) {
	t.Helper()
	if _, err := pool.Exec(context.Background(), sql, args...); err != nil {
		t.Logf("Aufraeumen fehlgeschlagen (%s): %v", sql, err)
	}
}

// TestSystematikAnlegen belegt die Luecke, die diese Aenderung schliesst: Das Vokabular,
// aus dem das Buchformular die Signatur vorschlaegt, war bisher nur lesbar — es gab
// keinen Weg, es zu fuellen, obwohl der Vorschlag davon abhaengt.
func TestSystematikAnlegen(t *testing.T) {
	pool := pgTestPool(t)
	ctx := context.Background()
	srv := &Server{DB: &db.Database{Pool: pool}}

	kuerzel := fmt.Sprintf("Deu%d", time.Now().UnixNano()%100000)
	t.Cleanup(func() {
		aufraeumen(t, pool, `DELETE FROM systematik_kategorien WHERE kuerzel LIKE 'Deu%'`)
	})

	rec, id := systematikAnlegen(t, srv, "  "+kuerzel+"  ", "  Deutsch  ")
	if rec.Code != http.StatusCreated {
		t.Fatalf("Anlegen: Status %d, Koerper %s", rec.Code, rec.Body.String())
	}
	if id == "" {
		t.Fatal("keine ID zurueckgegeben")
	}

	// Getrimmt gespeichert: Ein Kuerzel mit Rand-Leerzeichen landete sonst als
	// Signatur-Praefix am Buchruecken und passte zu keinem Regalbereich mehr.
	var gespeichert string
	if err := pool.QueryRow(ctx,
		`SELECT kuerzel FROM systematik_kategorien WHERE id = $1::uuid`, id).Scan(&gespeichert); err != nil {
		t.Fatalf("Kuerzel lesen: %v", err)
	}
	if gespeichert != kuerzel {
		t.Errorf("Kuerzel nicht getrimmt gespeichert: %q", gespeichert)
	}

	// Doppeltes Kuerzel -> 409, nicht 500.
	rec, _ = systematikAnlegen(t, srv, kuerzel, "Deutsch nochmal")
	if rec.Code != http.StatusConflict {
		t.Errorf("doppeltes Kuerzel: erwartet 409, war %d", rec.Code)
	}

	// Kuerzel mit Leerzeichen -> 400. Die Signatur wird am Leerzeichen in Regalbereiche
	// geschnitten; ein Kuerzel mit Leerzeichen zerlegte genau diese Grenze.
	rec, _ = systematikAnlegen(t, srv, "Deu Neu", "Kaputt")
	if rec.Code != http.StatusBadRequest {
		t.Errorf("Kuerzel mit Leerzeichen: erwartet 400, war %d", rec.Code)
	}
}

// TestSystematikLoeschenGeschuetzt sichert den Waechter ab, den die Datenbank NICHT
// stellt: buecher_titel.subject haelt die Bezeichnung als blossen Text, ohne
// Fremdschluessel. Ein Loeschen bliebe deshalb unbemerkt und liesse die betroffenen
// Titel auf ein Fach zeigen, das es nicht mehr gibt — sie verschwaenden aus der
// Fach-Auswahl, ohne dass irgendwo etwas fehlschlaegt.
func TestSystematikLoeschenGeschuetzt(t *testing.T) {
	pool := pgTestPool(t)
	ctx := context.Background()
	srv := &Server{DB: &db.Database{Pool: pool}}

	suffix := fmt.Sprintf("%d", time.Now().UnixNano()%100000)
	bezeichnung := "Erdkunde" + suffix
	_, id := systematikAnlegen(t, srv, "Ek"+suffix, bezeichnung)
	if id == "" {
		t.Fatal("Sachgruppe konnte nicht angelegt werden")
	}
	t.Cleanup(func() {
		aufraeumen(t, pool, `DELETE FROM systematik_kategorien WHERE id = $1::uuid`, id)
		aufraeumen(t, pool, `DELETE FROM buecher_titel WHERE subject = $1`, bezeichnung)
	})

	loeschen := func() int {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodDelete, "/api/systematics/"+id, nil)
		req.SetPathValue("id", id)
		srv.DeleteSystematikHandler().ServeHTTP(rec, req)
		return rec.Code
	}

	// Ohne Buecher: loeschbar? Erst mit Buch belegen, dann Schutz pruefen.
	if _, err := pool.Exec(ctx,
		`INSERT INTO buecher_titel (titel, subject) VALUES ($1, $2)`,
		"Erdkundebuch-"+suffix, bezeichnung); err != nil {
		t.Fatalf("Titel anlegen: %v", err)
	}

	if code := loeschen(); code != http.StatusConflict {
		t.Errorf("Loeschen einer benutzten Sachgruppe: erwartet 409, war %d", code)
	}

	var nochDa int
	if err := pool.QueryRow(ctx,
		`SELECT count(*) FROM systematik_kategorien WHERE id = $1::uuid`, id).Scan(&nochDa); err != nil {
		t.Fatalf("Bestand pruefen: %v", err)
	}
	if nochDa != 1 {
		t.Error("Sachgruppe wurde trotz 409 geloescht")
	}

	// Nach dem Entfernen des Buches ist sie frei.
	if _, err := pool.Exec(ctx, `DELETE FROM buecher_titel WHERE subject = $1`, bezeichnung); err != nil {
		t.Fatalf("Titel entfernen: %v", err)
	}
	if code := loeschen(); code != http.StatusNoContent {
		t.Errorf("Loeschen einer freien Sachgruppe: erwartet 204, war %d", code)
	}
}
