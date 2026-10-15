package leser

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"bibliothek/api/tuer"
	"bibliothek/auth"
	"bibliothek/db"
	"bibliothek/internal/pgtest"
	"bibliothek/repository"
)

// Die Leserdatei am Live-Pfad: über die Tür, nicht über das Repository.
//
// Das Repository allein beweist nur, dass die Abfrage es könnte. Dieser Test beweist,
// dass die Tür sie auch aufruft — und dass sie das nur tut, wenn die Ansicht alle Leser
// verlangt. An derselben Tür hängen der Reiter „Ehemalige" und die Schülersuche des
// Vormerkungs-Reiters; für die wäre ein Kollege in der Liste falsch.
func TestLeserdatei_ListeUeberAlleLeser(t *testing.T) {
	pool := pgtest.Pool(t)
	ctx := context.Background()
	srv := &Tueren{Basis: &tuer.Basis{DB: &db.Verbindung{Pool: pool}}}
	studentRepo := repository.NewLeserAbfragen(pool)
	claims := &auth.Claims{UserID: "00000000-0000-0000-0000-00000000a126", Rolle: auth.RoleAdmin}

	t.Cleanup(func() {
		if _, err := pool.Exec(ctx, `DELETE FROM leser WHERE nachname = 'Leserdateitest'`); err != nil {
			t.Errorf("aufräumen: %v", err)
		}
	})

	var lehrkraftID string
	if err := pool.QueryRow(ctx, `
		INSERT INTO leser (vorname, nachname, art) VALUES ('Katrin', 'Leserdateitest', 'lehrkraft')
		RETURNING id`).Scan(&lehrkraftID); err != nil {
		t.Fatalf("Lehrkraft anlegen: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO leser (barcode_id, vorname, nachname, klasse, abgaenger_jahr, art)
		VALUES ('LDT-1', 'Lena', 'Leserdateitest', '7a', 2030, 'schueler')`); err != nil {
		t.Fatalf("Schüler anlegen: %v", err)
	}

	liste := func(t *testing.T, adresse string) []repository.LeserListenZeile {
		t.Helper()
		req := httptest.NewRequest(http.MethodGet, adresse, nil)
		req = req.WithContext(context.WithValue(req.Context(), auth.ClaimsContextKey, claims))
		rec := httptest.NewRecorder()
		srv.Leserliste(studentRepo, repository.NewLMFAbfragen(pool)).ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("GET %s: Status %d, %s", adresse, rec.Code, rec.Body.String())
		}
		var out []repository.LeserListenZeile
		if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
			t.Fatalf("Antwort lesen: %v (%s)", err, rec.Body.String())
		}
		return out
	}
	// nurTest filtert die Zeilen dieses Tests heraus — die Datenbank trägt daneben, was
	// andere Tests und der Seed hinterlassen haben.
	nurTest := func(zeilen []repository.LeserListenZeile) map[string]repository.LeserListenZeile {
		out := map[string]repository.LeserListenZeile{}
		for _, z := range zeilen {
			if z.Nachname == "Leserdateitest" {
				out[z.Vorname] = z
			}
		}
		return out
	}

	ohneAngabe := nurTest(liste(t, "/api/schueler"))
	if _, da := ohneAngabe["Katrin"]; da {
		t.Error("ohne art=alle darf kein Kollege in der Liste stehen — an dieser Tür hängen auch Vormerkung und Ehemalige")
	}
	if _, da := ohneAngabe["Lena"]; !da {
		t.Error("die Schülerliste hat den Schüler verloren")
	}

	alle := nurTest(liste(t, "/api/schueler?art=alle"))
	kollegin, da := alle["Katrin"]
	if !da {
		t.Fatal("die Leserdatei zeigt keinen Kollegen — dann sieht niemand, welche Bücher er hat")
	}
	if kollegin.Art != "lehrkraft" {
		t.Errorf("Art der Kollegin: %q, erwartet \"lehrkraft\"", kollegin.Art)
	}
	if kollegin.ID != lehrkraftID {
		t.Errorf("ID der Kollegin: %q, erwartet %q", kollegin.ID, lehrkraftID)
	}
	// „07A": Das Klassen-Vokabular schreibt die eingetragene Form auf die registrierte
	// um (Migration 087) — hier nur als Beleg, dass die Klasse eines Schülers durchkommt.
	if schueler := alle["Lena"]; schueler.Art != "schueler" || schueler.Klasse != "07A" {
		t.Errorf("der Schüler in der Leserdatei: Art %q, Klasse %q", schueler.Art, schueler.Klasse)
	}

	// Eine Suche über alle findet die Kollegin ebenfalls — „eine Suche über alle Leser".
	if _, da := nurTest(liste(t, "/api/schueler?art=alle&q=Leserdateitest"))["Katrin"]; !da {
		t.Error("die Suche der Leserdatei findet den Kollegen nicht")
	}
}

// TestLeserdatei_AkteEinerLehrkraft: Die Akte muss jeden Leser öffnen.
//
// Läse GET /api/schueler/{id} die Sicht `schueler`, käme bei einer Lehrkraft 404 zurück. An
// der Theke hieße das: Der Kollege ist geladen, aber niemand sieht, welche Bücher er hat.
func TestLeserdatei_AkteEinerLehrkraft(t *testing.T) {
	pool := pgtest.Pool(t)
	ctx := context.Background()
	srv := &Tueren{Basis: &tuer.Basis{DB: &db.Verbindung{Pool: pool}}}
	studentRepo := repository.NewLeserAbfragen(pool)
	claims := &auth.Claims{UserID: "00000000-0000-0000-0000-00000000a127", Rolle: auth.RoleAdmin}

	t.Cleanup(func() {
		if _, err := pool.Exec(ctx, `DELETE FROM leser WHERE nachname = 'Aktetest'`); err != nil {
			t.Errorf("aufräumen: %v", err)
		}
	})

	var lehrkraftID string
	if err := pool.QueryRow(ctx, `
		INSERT INTO leser (barcode_id, vorname, nachname, art)
		VALUES ('AKTE-1', 'Katrin', 'Aktetest', 'lehrkraft') RETURNING id`).Scan(&lehrkraftID); err != nil {
		t.Fatalf("Lehrkraft anlegen: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/schueler/"+lehrkraftID, nil)
	req.SetPathValue("id", lehrkraftID)
	req = req.WithContext(context.WithValue(req.Context(), auth.ClaimsContextKey, claims))
	rec := httptest.NewRecorder()
	srv.Leserakte(studentRepo).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("Akte einer Lehrkraft: Status %d, %s", rec.Code, rec.Body.String())
	}
	var akte leserakteAntwort
	if err := json.Unmarshal(rec.Body.Bytes(), &akte); err != nil {
		t.Fatalf("Antwort lesen: %v (%s)", err, rec.Body.String())
	}
	if akte.ID != lehrkraftID || akte.Nachname != "Aktetest" {
		t.Fatalf("falsche Akte: %+v", akte)
	}
	if akte.Art != "lehrkraft" {
		t.Errorf("Art in der Akte: %q, erwartet \"lehrkraft\" — ohne sie zeigt die Akte die Felder eines Schülers", akte.Art)
	}
	if akte.Klasse != "" || akte.AbgaengerJahr != 0 {
		t.Errorf("eine Lehrkraft hat keine Klasse und kein Abgangsjahr: Klasse %q, Abgang %d", akte.Klasse, akte.AbgaengerJahr)
	}
	if akte.EntlieheneBuecher == nil {
		t.Error("entliehene_buecher muss ein Array sein, auch ein leeres — sonst bricht die Ausleihliste")
	}

	// Die Akte eines Schülers bleibt, was sie war.
	var schuelerID string
	if err := pool.QueryRow(ctx, `
		INSERT INTO leser (barcode_id, vorname, nachname, klasse, abgaenger_jahr, art)
		VALUES ('AKTE-2', 'Lena', 'Aktetest', '7a', 2030, 'schueler') RETURNING id`).Scan(&schuelerID); err != nil {
		t.Fatalf("Schüler anlegen: %v", err)
	}
	req2 := httptest.NewRequest(http.MethodGet, "/api/schueler/"+schuelerID, nil)
	req2.SetPathValue("id", schuelerID)
	req2 = req2.WithContext(context.WithValue(req2.Context(), auth.ClaimsContextKey, claims))
	rec2 := httptest.NewRecorder()
	srv.Leserakte(studentRepo).ServeHTTP(rec2, req2)
	if rec2.Code != http.StatusOK {
		t.Fatalf("Akte eines Schülers: Status %d, %s", rec2.Code, rec2.Body.String())
	}
	var schuelerAkte leserakteAntwort
	if err := json.Unmarshal(rec2.Body.Bytes(), &schuelerAkte); err != nil {
		t.Fatalf("Antwort lesen: %v", err)
	}
	if schuelerAkte.Art != "schueler" || schuelerAkte.Klasse != "07A" {
		t.Errorf("Akte des Schülers: Art %q, Klasse %q", schuelerAkte.Art, schuelerAkte.Klasse)
	}
}

// Zwei Türen, die über die Leserdatei erreichbar sind und über die Sicht `schueler` nichts
// fänden. Beide stehen in der Akte eines Kollegen sichtbar da:
//
//   - Der Sperr-Endpunkt (PATCH /api/admin/students/{id}/lock) antwortete mit „Schüler
//     nicht gefunden". Ein Knopf, der nur scheitern kann.
//   - Das Passbild würde gespeichert, aber nie ausgeliefert, wenn die Auslieferung das
//     Foto über die Sicht mit seiner Person verbände. Die Akte zeigte weiter die
//     Initialen, und niemand könnte sagen, warum.
//
// Ein Kollege wird nicht gesperrt: Die Tür lehnt es mit einem Satz ab, und eine alte Sperre
// an seinem Konto lässt sie aufheben, über `leser`, sonst wäre auch das „nicht gefunden".
// Rot gesehen am Rückbau: ohne den Fall für Kollegen sperrt die Tür ihn (200).
func TestLeserdatei_SperreUndFotoEinesKollegen(t *testing.T) {
	pool := pgtest.Pool(t)
	ctx := context.Background()
	srv := &Tueren{Basis: &tuer.Basis{DB: &db.Verbindung{Pool: pool}}}
	claims := &auth.Claims{UserID: "00000000-0000-0000-0000-00000000a129", Rolle: auth.RoleAdmin}

	t.Cleanup(func() {
		if _, err := pool.Exec(ctx, `DELETE FROM leser WHERE nachname = 'Sperrtest'`); err != nil {
			t.Errorf("aufräumen: %v", err)
		}
	})

	var id string
	if err := pool.QueryRow(ctx, `
		INSERT INTO leser (barcode_id, vorname, nachname, art)
		VALUES ('SPERR-1', 'Katrin', 'Sperrtest', 'lehrkraft') RETURNING id`).Scan(&id); err != nil {
		t.Fatalf("Lehrkraft anlegen: %v", err)
	}

	sperrTuer := func(body string) *httptest.ResponseRecorder {
		t.Helper()
		req := httptest.NewRequest(http.MethodPatch, "/api/admin/students/"+id+"/lock", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.SetPathValue("id", id)
		req = req.WithContext(context.WithValue(req.Context(), auth.ClaimsContextKey, claims))
		rec := httptest.NewRecorder()
		srv.LeserSperren(repository.NewProtokollAbfragen(pool)).ServeHTTP(rec, req)
		return rec
	}
	gesperrt := func() bool {
		t.Helper()
		var b bool
		if err := pool.QueryRow(ctx, `SELECT is_manually_blocked FROM leser WHERE id = $1`, id).Scan(&b); err != nil {
			t.Fatalf("Sperrstatus lesen: %v", err)
		}
		return b
	}

	// 1. Sperren lehnt die Tür ab — mit dem Satz, den auch die Akte zeigt.
	rec := sperrTuer(`{"is_locked":true,"reason":"Buch seit zwei Jahren überfällig"}`)
	if rec.Code != http.StatusBadRequest || !strings.Contains(rec.Body.String(), "Kollegen werden nicht gesperrt") {
		t.Fatalf("Kollegen sperren: Status %d, %s — erwartet 400 „Kollegen werden nicht gesperrt.“", rec.Code, rec.Body.String())
	}
	if gesperrt() {
		t.Error("abgelehnt, aber trotzdem gesperrt")
	}

	// 1b. Eine Sperre von früher lässt sich aufheben (die Tür schreibt `leser`).
	if _, err := pool.Exec(ctx, `UPDATE leser SET is_manually_blocked = true, block_reason = 'alt' WHERE id = $1`, id); err != nil {
		t.Fatalf("alte Sperre anlegen: %v", err)
	}
	if rec := sperrTuer(`{"is_locked":false}`); rec.Code != http.StatusOK {
		t.Fatalf("alte Sperre eines Kollegen aufheben: Status %d, %s", rec.Code, rec.Body.String())
	}
	if gesperrt() {
		t.Error("die alte Sperre steht noch")
	}

	// 2. Passbild ausliefern. Ein Klartextbild genügt nicht — die Spalte ist
	// verschlüsselt —, aber der Verbund Foto→Person ist genau die Stelle, die brach:
	// Steht die Zeile nicht, antwortet die Tür mit 404 „kein foto gefunden".
	if _, err := pool.Exec(ctx,
		`INSERT INTO schueler_fotos (schueler_id, foto_encrypted) VALUES ($1, $2)`, id, []byte("x")); err != nil {
		t.Fatalf("Foto anlegen: %v", err)
	}
	var zugeordnet bool
	if err := pool.QueryRow(ctx, `
		SELECT EXISTS(SELECT 1 FROM schueler_fotos sf JOIN leser l ON l.id = sf.schueler_id
		               WHERE l.barcode_id = 'SPERR-1')`).Scan(&zugeordnet); err != nil {
		t.Fatalf("Foto-Zuordnung lesen: %v", err)
	}
	if !zugeordnet {
		t.Fatal("der Test misst nichts — das Foto hängt gar nicht am Kollegen")
	}
	fotoReq := httptest.NewRequest(http.MethodGet, "/api/schueler/SPERR-1/photo", nil)
	fotoReq.SetPathValue("barcode_id", "SPERR-1")
	fotoReq = fotoReq.WithContext(context.WithValue(fotoReq.Context(), auth.ClaimsContextKey, claims))
	fotoRec := httptest.NewRecorder()
	srv.Foto().ServeHTTP(fotoRec, fotoReq)
	if fotoRec.Code == http.StatusNotFound {
		t.Error("das Passbild eines Kollegen wird gespeichert, aber nicht ausgeliefert — die Akte zeigt weiter Initialen")
	}
}
