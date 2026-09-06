package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sort"
	"strings"
	"testing"
	"time"

	"bibliothek/auth"
	"bibliothek/db"
	"bibliothek/sse"
)

// Die Schreibwege am echten Router: Wer das Recht seiner Matrix-Zeile NICHT hat,
// bekommt 403 — und zwar wegen des Rechts, nicht wegen etwas anderem.
//
// Die Lücke, die dieses Gate schließt (Bestands-Durchgang 06.09.2026): Das
// Matrix-Gate (pii_matrix_test.go) vergleicht Text mit Text — es prüft, dass die
// Registrierung ein `RequirePermission(...)` mit dem dokumentierten Namen trägt. Das
// Antwort-Gate (pii_antwort_gate_pg_test.go) fährt den echten Router, aber nur über
// GET-Routen und zwei lesende POSTs; es sagt das selbst: „Die übrigen Nicht-GET-Zeilen
// bleiben außerhalb." Damit war die HÄLFTE, die die Welt verändert — 91 von 186 Routen,
// darunter jedes Löschen — nur lexikalisch geprüft.
//
// Was lexikalisch unsichtbar bleibt: ein Recht, das im Seed JEDER Rolle gehört. Der
// Wrapper steht da, das Dokument stimmt, und trotzdem darf jeder Angemeldete schreiben
// (Bugklasse „Seed erreicht bestehende DB nie"). Genau das misst dieses Gate.
//
// Bewusst nur die NEGATIVE Richtung: Der Request wird abgewiesen, bevor er den Handler
// erreicht — es wird also nichts geschrieben, gemailt oder importiert. Eine
// Positiv-Probe „mit Recht geht es durch" wäre hier gefährlich (POST /api/mahnwesen/…
// verschickt Post), und sie ist auch nicht nötig: Dass die Rechte-Prüfung überhaupt
// greift und nicht etwa CSRF oder die Sitzung das 403 auslöst, belegt der GRUND in der
// Antwort — „keine Berechtigung für diese Aktion" gegen „CSRF-Validierung
// fehlgeschlagen". Ohne diese Unterscheidung wäre das Gate grün aus dem falschen Grund.

// Routen, die kein Fachrecht tragen (können) — mit Begründung, wie es sich für eine
// Ausnahmeliste gehört.
var schreibwegeOhneFachrecht = map[string]string{
	"POST /api/public/bestellung/{token}/bestaetigen": "Bestätigungs-Link des Lieferanten: der 256-Bit-Token IST der Ausweis",
	"POST /api/auth/refresh":                          "Auth-Endpunkt, validiert das Token selbst",
	"POST /api/auth/logout":                           "Auth-Endpunkt",
	"POST /login":                                     "Login — es existiert noch keine Sitzung",
}

// füllwert ersetzt die Platzhalter einer Route. UUIDs sind Pflicht: Die
// UUID-Prüfung sitzt VOR der Rechte-Prüfung (ValidateUUIDParamsMiddleware), ein
// „x" im Pfad ergäbe 400 statt 403 und das Gate liefe ins Leere.
func fuellwert(platzhalter string) string {
	switch platzhalter {
	case "{art}":
		return "rueckgabe"
	case "{klasse}":
		return "05A"
	case "{token}":
		return "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	default:
		return "11111111-1111-1111-1111-111111111111"
	}
}

func schreibURL(route string) (methode, url string) {
	teile := strings.SplitN(route, " ", 2)
	if len(teile) != 2 {
		return "", ""
	}
	pfad := teile[1]
	for {
		auf := strings.Index(pfad, "{")
		if auf < 0 {
			break
		}
		zu := strings.Index(pfad[auf:], "}")
		if zu < 0 {
			break
		}
		platzhalter := pfad[auf : auf+zu+1]
		pfad = strings.Replace(pfad, platzhalter, fuellwert(platzhalter), 1)
	}
	return teile[0], pfad
}

func TestSchreibwegeWeisenOhneDasRechtAb(t *testing.T) {
	pool := pgTestPool(t)
	t.Setenv("RATE_LIMIT", "100000")

	if err := (&db.Database{Pool: pool}).InitPermissions(context.Background()); err != nil {
		t.Fatalf("InitPermissions: %v", err)
	}
	authenticator, err := auth.NewAuthenticator(
		"rechte-schreibwege-gate-testgeheimnis-32-bytes!", pool, time.Hour)
	if err != nil {
		t.Fatalf("Authenticator: %v", err)
	}
	srv := NewServer(&db.Database{Pool: pool}, authenticator, sse.NewBroker(), false)
	router := srv.Routes()
	welt := baueKanarienWelt(t, pool, authenticator)

	// Ein Füllrecht, das die Rolle STATT des geforderten hält. Zwei genügen: Für Zeilen,
	// die selbst view_books verlangen, muss ein anderes her — sonst prüfte das Gate eine
	// Rolle, die das geforderte Recht doch hat.
	const fuell, fuellAlternativ = "view_books", "view_orders"

	var geprueft, uebersprungen int
	var fehler []string
	routen := make([]string, 0)
	matrix := leseMatrix(t)
	for route := range matrix {
		if strings.HasPrefix(route, "GET ") || !strings.Contains(route, " ") {
			continue
		}
		routen = append(routen, route)
	}
	sort.Strings(routen)

	if len(routen) < 50 {
		t.Fatalf("nur %d Schreibrouten in der Matrix — die Suchform passt nicht mehr, "+
			"und dieses Gate wäre still grün", len(routen))
	}

	for _, route := range routen {
		if grund, ok := schreibwegeOhneFachrecht[route]; ok {
			_ = grund
			uebersprungen++
			continue
		}
		recht, mitSitzung := rechtFuerZeile(matrix[route].Recht)
		if recht == "" || !mitSitzung {
			uebersprungen++
			continue
		}
		methode, url := schreibURL(route)
		if methode == "" {
			t.Errorf("Route %q lässt sich nicht in einen Aufruf übersetzen", route)
			continue
		}

		anderes := fuell
		if recht == fuell {
			anderes = fuellAlternativ
		}
		setzeGenauEinRecht(t, pool, anderes)

		req := mitCSRF(httptest.NewRequest(methode, url, strings.NewReader("{}")))
		req.Header.Set("Content-Type", "application/json")
		req.AddCookie(&http.Cookie{Name: "session_token", Value: welt.sessionToken})
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		geprueft++

		rumpf := rec.Body.String()
		switch {
		case rec.Code != http.StatusForbidden:
			fehler = append(fehler, route+": HTTP "+http.StatusText(rec.Code)+
				" statt 403, obwohl die Rolle "+recht+" nicht hat — "+rumpf)
		case strings.Contains(rumpf, "CSRF"):
			fehler = append(fehler, route+": das 403 kam von der CSRF-Prüfung, nicht vom Recht "+
				"— dieser Fall würde jedes fehlende Recht verdecken")
		case !strings.Contains(rumpf, "keine Berechtigung"):
			fehler = append(fehler, route+": 403 ohne die Begründung des Rechte-Wächters — "+rumpf)
		}
	}

	if geprueft < 50 {
		t.Fatalf("nur %d Schreibrouten wirklich gefahren (%d übersprungen) — zu wenig für eine Aussage",
			geprueft, uebersprungen)
	}
	if len(fehler) > 0 {
		t.Errorf("Schreibrouten ohne wirksamen Rechte-Schutz (%d von %d gefahren):\n  %s",
			len(fehler), geprueft, strings.Join(fehler, "\n  "))
	}
	t.Logf("%d Schreibrouten am echten Router geprüft, %d begründet übersprungen", geprueft, uebersprungen)
}
