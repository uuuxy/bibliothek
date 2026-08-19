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

	// Bezeichnung mit Suffix: Sie ist seit Migration 078 (case-insensitiv) eindeutig,
	// und andere Testpakete registrieren im selben Test-Postgres bereits "Deutsch".
	suffix := fmt.Sprintf("%d", time.Now().UnixNano()%100000)
	kuerzel := "Deu" + suffix
	bezeichnung := "Deutsch" + suffix
	t.Cleanup(func() {
		aufraeumen(t, pool, `DELETE FROM systematik_kategorien WHERE kuerzel LIKE 'Deu%'`)
	})

	rec, id := systematikAnlegen(t, srv, "  "+kuerzel+"  ", "  "+bezeichnung+"  ")
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

// TestSystematikLoeschenGeschuetzt sichert den Waechter des Handlers ab: 409 mit
// verstaendlicher Meldung, solange Titel auf dem Fach stehen. Seit Migration 078
// haelt zusaetzlich die Datenbank dagegen (fk_titel_subject_systematik, ON DELETE
// RESTRICT) — der Handler bleibt fuer die fachliche Meldung zustaendig, der FK ist
// die Rueckfallebene fuer jeden Weg am Handler vorbei.
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

// TestSystematikRenameZiehtTitelMit belegt den F3-Fix: Wird eine Sachgruppe
// umbenannt, wandert die neue Bezeichnung auf die Titel (buecher_titel.subject) —
// sonst blieben sie lautlos auf dem alten Fachnamen und fielen aus der Fach-Auswahl.
// Die Signatur (Kürzel am Buchrücken) bleibt bewusst unberührt.
func TestSystematikRenameZiehtTitelMit(t *testing.T) {
	pool := pgTestPool(t)
	ctx := context.Background()
	srv := &Server{DB: &db.Database{Pool: pool}}

	suffix := fmt.Sprintf("%d", time.Now().UnixNano()%100000)
	alt := "Biologie" + suffix
	neu := "Naturwissenschaften" + suffix
	_, id := systematikAnlegen(t, srv, "Bio"+suffix, alt)
	if id == "" {
		t.Fatal("Sachgruppe konnte nicht angelegt werden")
	}
	t.Cleanup(func() {
		aufraeumen(t, pool, `DELETE FROM systematik_kategorien WHERE id = $1::uuid`, id)
		aufraeumen(t, pool, `DELETE FROM buecher_titel WHERE subject IN ($1,$2)`, alt, neu)
	})

	// Zwei Titel auf dem alten Fach, einer auf einem Fremdfach (darf NICHT mitgezogen
	// werden). Das Fremdfach muss seit Migration 078 registriert sein (subject ist FK).
	fremd := "Fremdfach" + suffix
	if _, err := pool.Exec(ctx,
		`INSERT INTO systematik_kategorien (kuerzel, bezeichnung) VALUES ($1, $2)`,
		"Frd"+suffix, fremd); err != nil {
		t.Fatalf("Fremdfach registrieren: %v", err)
	}
	t.Cleanup(func() {
		aufraeumen(t, pool, `DELETE FROM systematik_kategorien WHERE bezeichnung = $1`, fremd)
	})
	if _, err := pool.Exec(ctx, `
		INSERT INTO buecher_titel (titel, subject, signatur) VALUES
		($1,$2,'BIB Bio 1'), ($3,$2,'BIB Bio 2'), ($4,$5,'BIB X')`,
		"Biobuch-A-"+suffix, alt, "Biobuch-B-"+suffix, "Fremd-"+suffix, fremd); err != nil {
		t.Fatalf("Titel anlegen: %v", err)
	}

	// Umbenennen (Kürzel unverändert, nur Bezeichnung).
	koerper, err := json.Marshal(map[string]string{"kuerzel": "Bio" + suffix, "bezeichnung": neu})
	if err != nil {
		t.Fatalf("Anfrage kodieren: %v", err)
	}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/api/systematics/"+id, bytes.NewReader(koerper))
	req.SetPathValue("id", id)
	srv.UpdateSystematikHandler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("Umbenennen: erwartet 200, war %d: %s", rec.Code, rec.Body.String())
	}
	var antwort map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &antwort); err != nil {
		t.Fatalf("Antwort unlesbar: %v", err)
	}
	mit, ok := antwort["titel_mitgezogen"].(float64)
	if !ok || mit != 2 {
		t.Errorf("erwartet 2 mitgezogene Titel, waren %v", antwort["titel_mitgezogen"])
	}

	// Die zwei Titel tragen jetzt das neue Fach, der Fremdtitel nicht.
	var aufNeu, aufAlt, fremdCount int
	if err := pool.QueryRow(ctx, `SELECT
		count(*) FILTER (WHERE subject=$1),
		count(*) FILTER (WHERE subject=$2),
		count(*) FILTER (WHERE subject=$3)
		FROM buecher_titel WHERE titel LIKE $4`,
		neu, alt, fremd, "%"+suffix).Scan(&aufNeu, &aufAlt, &fremdCount); err != nil {
		t.Fatalf("Bestand prüfen: %v", err)
	}
	if aufNeu != 2 {
		t.Errorf("erwartet 2 Titel auf dem neuen Fach, waren %d", aufNeu)
	}
	if aufAlt != 0 {
		t.Errorf("kein Titel darf auf dem alten Fach zurückbleiben, waren %d", aufAlt)
	}
	if fremdCount != 1 {
		t.Errorf("Fremdfach-Titel darf nicht mitgezogen werden, war %d statt 1", fremdCount)
	}

	// Signatur bleibt unberührt (physisches Etikett).
	var sig string
	if err := pool.QueryRow(ctx,
		`SELECT signatur FROM buecher_titel WHERE titel = $1`, "Biobuch-A-"+suffix).Scan(&sig); err != nil {
		t.Fatalf("Signatur lesen: %v", err)
	}
	if sig != "BIB Bio 1" {
		t.Errorf("Signatur darf sich nicht ändern, war %q", sig)
	}
}
