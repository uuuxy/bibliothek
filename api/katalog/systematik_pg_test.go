package katalog

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"bibliothek/api/tuer"
	"bibliothek/db"
	"bibliothek/internal/pgtest"
)

// systematikAnlegen ruft die Tür SachgruppeAnlegen und liefert den Recorder und die neue Kennung.
func systematikAnlegen(t *testing.T, srv *Tueren, kuerzel, bezeichnung string) (*httptest.ResponseRecorder, string) {
	t.Helper()
	koerper, err := json.Marshal(map[string]string{"kuerzel": kuerzel, "bezeichnung": bezeichnung})
	if err != nil {
		t.Fatalf("Anfrage kodieren: %v", err)
	}
	rec := httptest.NewRecorder()
	srv.SachgruppeAnlegen().ServeHTTP(rec,
		httptest.NewRequest(http.MethodPost, "/api/systematics", bytes.NewReader(koerper)))

	// Fehlerantworten tragen kein id-Feld; das ist erwartbar und kein Testfehler.
	var antwort map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &antwort); err != nil {
		return rec, ""
	}
	return rec, antwort["id"]
}

// TestSachgruppeAnlegen: Das Vokabular, aus dem das Buchformular die Signatur vorschlägt, lässt
// sich über die Tür füllen; der Vorschlag hängt davon ab.
func TestSachgruppeAnlegen(t *testing.T) {
	pool := pgtest.Pool(t)
	ctx := context.Background()
	srv := &Tueren{Basis: &tuer.Basis{DB: &db.Database{Pool: pool}}}

	// Bezeichnung mit Anhang: Sie ist eindeutig, auch ohne Rücksicht auf Groß- und
	// Kleinschreibung (Migration 078), und andere Testpakete legen in derselben Datenbank schon
	// „Deutsch" an.
	suffix := fmt.Sprintf("%d", time.Now().UnixNano()%100000)
	kuerzel := "Deu" + suffix
	bezeichnung := "Deutsch" + suffix
	t.Cleanup(func() {
		pgtest.RaeumeAuf(t, pool, `DELETE FROM systematik_kategorien WHERE kuerzel LIKE 'Deu%'`)
	})

	rec, id := systematikAnlegen(t, srv, "  "+kuerzel+"  ", "  "+bezeichnung+"  ")
	if rec.Code != http.StatusCreated {
		t.Fatalf("Anlegen: Status %d, Koerper %s", rec.Code, rec.Body.String())
	}
	if id == "" {
		t.Fatal("keine ID zurueckgegeben")
	}

	// Ohne Leerzeichen am Rand gespeichert: Ein Kürzel mit Leerzeichen am Rand landete sonst als
	// Anfang der Signatur am Buchrücken und passte zu keinem Regalbereich mehr.
	var gespeichert string
	if err := pool.QueryRow(ctx,
		`SELECT kuerzel FROM systematik_kategorien WHERE id = $1::uuid`, id).Scan(&gespeichert); err != nil {
		t.Fatalf("Kuerzel lesen: %v", err)
	}
	if gespeichert != kuerzel {
		t.Errorf("Kuerzel nicht getrimmt gespeichert: %q", gespeichert)
	}

	// Doppeltes Kürzel: 409, nicht 500.
	rec, _ = systematikAnlegen(t, srv, kuerzel, "Deutsch nochmal")
	if rec.Code != http.StatusConflict {
		t.Errorf("doppeltes Kuerzel: erwartet 409, war %d", rec.Code)
	}

	// Kürzel mit Leerzeichen: 400. Die Signatur wird am Leerzeichen in Regalbereiche
	// geschnitten; ein Kürzel mit Leerzeichen zerlegte genau diese Grenze.
	rec, _ = systematikAnlegen(t, srv, "Deu Neu", "Kaputt")
	if rec.Code != http.StatusBadRequest {
		t.Errorf("Kuerzel mit Leerzeichen: erwartet 400, war %d", rec.Code)
	}
}

// TestSachgruppeLoeschenGeschuetzt sichert die Prüfung der Tür ab: 409 mit verständlicher
// Meldung, solange Titel auf dem Fach stehen. Die Datenbank hält zusätzlich dagegen
// (fk_titel_subject_systematik, ON DELETE RESTRICT, Migration 078); die Tür bleibt für die
// fachliche Meldung zuständig, der Fremdschlüssel ist die Rückfallebene für jeden Weg an der
// Tür vorbei.
func TestSachgruppeLoeschenGeschuetzt(t *testing.T) {
	pool := pgtest.Pool(t)
	ctx := context.Background()
	srv := &Tueren{Basis: &tuer.Basis{DB: &db.Database{Pool: pool}}}

	suffix := fmt.Sprintf("%d", time.Now().UnixNano()%100000)
	bezeichnung := "Erdkunde" + suffix
	_, id := systematikAnlegen(t, srv, "Ek"+suffix, bezeichnung)
	if id == "" {
		t.Fatal("Sachgruppe konnte nicht angelegt werden")
	}
	t.Cleanup(func() {
		pgtest.RaeumeAuf(t, pool, `DELETE FROM systematik_kategorien WHERE id = $1::uuid`, id)
		pgtest.RaeumeAuf(t, pool, `DELETE FROM buecher_titel WHERE subject = $1`, bezeichnung)
	})

	loeschen := func() int {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodDelete, "/api/systematics/"+id, nil)
		req.SetPathValue("id", id)
		srv.SachgruppeLoeschen().ServeHTTP(rec, req)
		return rec.Code
	}

	// Erst mit einem Buch belegen, dann den Schutz prüfen.
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
	// Sie ist jetzt fort: Ein zweites Löschen findet sie nicht, und das ist keine Störung.
	if code := loeschen(); code != http.StatusNotFound {
		t.Errorf("Loeschen einer unbekannten Sachgruppe: erwartet 404, war %d", code)
	}
}

// TestSachgruppeAendern_ZiehtTitelMit: Wird eine Sachgruppe umbenannt, wandert die neue
// Bezeichnung auf die Titel (buecher_titel.subject); sonst blieben sie still auf dem alten
// Fachnamen und fielen aus der Auswahl der Fächer. Die Signatur (Kürzel am Buchrücken) bleibt
// unberührt.
func TestSachgruppeAendern_ZiehtTitelMit(t *testing.T) {
	pool := pgtest.Pool(t)
	ctx := context.Background()
	srv := &Tueren{Basis: &tuer.Basis{DB: &db.Database{Pool: pool}}}

	suffix := fmt.Sprintf("%d", time.Now().UnixNano()%100000)
	alt := "Biologie" + suffix
	neu := "Naturwissenschaften" + suffix
	_, id := systematikAnlegen(t, srv, "Bio"+suffix, alt)
	if id == "" {
		t.Fatal("Sachgruppe konnte nicht angelegt werden")
	}
	t.Cleanup(func() {
		pgtest.RaeumeAuf(t, pool, `DELETE FROM systematik_kategorien WHERE id = $1::uuid`, id)
		pgtest.RaeumeAuf(t, pool, `DELETE FROM buecher_titel WHERE subject IN ($1,$2)`, alt, neu)
	})

	// Zwei Titel auf dem alten Fach, einer auf einem Fremdfach (darf nicht mitgezogen
	// werden). Das Fremdfach muss angelegt sein, weil subject ein Fremdschlüssel ist
	// (Migration 078).
	fremd := "Fremdfach" + suffix
	if _, err := pool.Exec(ctx,
		`INSERT INTO systematik_kategorien (kuerzel, bezeichnung) VALUES ($1, $2)`,
		"Frd"+suffix, fremd); err != nil {
		t.Fatalf("Fremdfach registrieren: %v", err)
	}
	t.Cleanup(func() {
		pgtest.RaeumeAuf(t, pool, `DELETE FROM systematik_kategorien WHERE bezeichnung = $1`, fremd)
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
	srv.SachgruppeAendern().ServeHTTP(rec, req)
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

// Eine unbekannte Sachgruppe zu ändern ist ein 404; eine Bezeichnung, die eine andere schon
// trägt, ein 409, und die Sachgruppe behält ihre.
func TestSachgruppeAendern_UnbekanntUndDoppelt(t *testing.T) {
	pool := pgtest.Pool(t)
	ctx := context.Background()
	srv := &Tueren{Basis: &tuer.Basis{DB: &db.Database{Pool: pool}}}

	suffix := fmt.Sprintf("%d", time.Now().UnixNano()%100000)
	t.Cleanup(func() {
		pgtest.RaeumeAuf(t, pool, `DELETE FROM systematik_kategorien WHERE kuerzel LIKE 'Dop%'`)
	})
	if rec, _ := systematikAnlegen(t, srv, "DopA"+suffix, "Doppelt A "+suffix); rec.Code != http.StatusCreated {
		t.Fatalf("erste Sachgruppe: Status %d, %s", rec.Code, rec.Body.String())
	}
	rec, zweite := systematikAnlegen(t, srv, "DopB"+suffix, "Doppelt B "+suffix)
	if rec.Code != http.StatusCreated {
		t.Fatalf("zweite Sachgruppe: Status %d, %s", rec.Code, rec.Body.String())
	}
	aendere := func(id, kuerzel, bezeichnung string) *httptest.ResponseRecorder {
		t.Helper()
		koerper, err := json.Marshal(map[string]string{"kuerzel": kuerzel, "bezeichnung": bezeichnung})
		if err != nil {
			t.Fatalf("Anfrage kodieren: %v", err)
		}
		req := httptest.NewRequest(http.MethodPut, "/api/systematics/"+id, bytes.NewReader(koerper))
		req.SetPathValue("id", id)
		rec := httptest.NewRecorder()
		srv.SachgruppeAendern().ServeHTTP(rec, req)
		return rec
	}

	if rec := aendere("00000000-0000-0000-0000-00000000dead", "DopC"+suffix, "Doppelt C "+suffix); rec.Code != http.StatusNotFound {
		t.Errorf("unbekannte Sachgruppe: Status %d, erwartet 404: %s", rec.Code, rec.Body.String())
	}
	if rec := aendere(zweite, "DopB"+suffix, "Doppelt A "+suffix); rec.Code != http.StatusConflict {
		t.Errorf("schon vergebene Bezeichnung: Status %d, erwartet 409: %s", rec.Code, rec.Body.String())
	}
	var bezeichnung string
	if err := pool.QueryRow(ctx,
		`SELECT bezeichnung FROM systematik_kategorien WHERE id = $1::uuid`, zweite).Scan(&bezeichnung); err != nil {
		t.Fatalf("Bezeichnung lesen: %v", err)
	}
	if bezeichnung != "Doppelt B "+suffix {
		t.Errorf("die abgelehnte Änderung hat die Bezeichnung geändert: %q", bezeichnung)
	}
}
