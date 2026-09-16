package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"bibliothek/auth"
	"bibliothek/db"
	"bibliothek/repository"
)

// Die Leserdatei am LIVE-PFAD: über den Handler, nicht über das Repository.
//
// Das Repository allein beweist nur, dass die Abfrage es könnte. Dieser Test beweist,
// dass die Tür sie auch aufruft — und dass sie das NUR tut, wenn die Ansicht alle Leser
// verlangt. An derselben Tür hängen der Reiter „Ehemalige" und die Schülersuche des
// Vormerkungs-Reiters; für die wäre ein Kollege in der Liste falsch.
func TestLeserdatei_ListeUeberAlleLeser(t *testing.T) {
	pool := pgTestPool(t)
	ctx := context.Background()
	srv := &Server{DB: &db.Database{Pool: pool}}
	studentRepo := repository.NewStudentRepository(pool)
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

	liste := func(t *testing.T, adresse string) []repository.StudentListStat {
		t.Helper()
		req := httptest.NewRequest(http.MethodGet, adresse, nil)
		req = req.WithContext(context.WithValue(req.Context(), auth.ClaimsContextKey, claims))
		rec := httptest.NewRecorder()
		srv.ListStudentsHandler(studentRepo).ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("GET %s: Status %d, %s", adresse, rec.Code, rec.Body.String())
		}
		var out []repository.StudentListStat
		if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
			t.Fatalf("Antwort lesen: %v (%s)", err, rec.Body.String())
		}
		return out
	}
	// nurTest filtert die Zeilen dieses Tests heraus — die Datenbank trägt daneben, was
	// andere Tests und der Seed hinterlassen haben.
	nurTest := func(zeilen []repository.StudentListStat) map[string]repository.StudentListStat {
		out := map[string]repository.StudentListStat{}
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
// Bis zum 16.09.2026 las GET /api/schueler/{id} die Sicht `schueler` — bei einer
// Lehrkraft kam 404 zurück. An der Theke bedeutete das: Der Kollege ist geladen, aber
// niemand sieht, welche Bücher er hat. Genau deshalb zeigte die Theke ihm nur eine
// schmale Karte statt seiner Akte.
func TestLeserdatei_AkteEinerLehrkraft(t *testing.T) {
	pool := pgTestPool(t)
	ctx := context.Background()
	srv := &Server{DB: &db.Database{Pool: pool}}
	studentRepo := repository.NewStudentRepository(pool)
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
	srv.GetStudentProfileHandler(studentRepo).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("Akte einer Lehrkraft: Status %d, %s", rec.Code, rec.Body.String())
	}
	var akte StudentProfileResponse
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
	srv.GetStudentProfileHandler(studentRepo).ServeHTTP(rec2, req2)
	if rec2.Code != http.StatusOK {
		t.Fatalf("Akte eines Schülers: Status %d, %s", rec2.Code, rec2.Body.String())
	}
	var schuelerAkte StudentProfileResponse
	if err := json.Unmarshal(rec2.Body.Bytes(), &schuelerAkte); err != nil {
		t.Fatalf("Antwort lesen: %v", err)
	}
	if schuelerAkte.Art != "schueler" || schuelerAkte.Klasse != "07A" {
		t.Errorf("Akte des Schülers: Art %q, Klasse %q", schuelerAkte.Art, schuelerAkte.Klasse)
	}
}
