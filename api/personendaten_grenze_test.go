package api

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"bibliothek/auth"
	"bibliothek/db"
	"bibliothek/repository"
)

// Die Personendaten-Grenze der Katalog-Rechte (Nachgang zur unabhängigen
// Prüfung, bewertung/sicherheitsbefund-*.md): view_books wurde der Helfer-Rolle
// mit der Zusage "öffnet keine Personendaten" gegeben. Diese Tests halten die
// Zusage an beiden Sorten von Türen fest — Routen, deren Zweck Personendaten
// sind, hängen an view_students; die Geräteliste blendet den Ausleihernamen aus.

// TestPersonendatenRoutenHinterViewStudents liest die Routen-Registrierung als
// Quelle (dieselbe Technik wie routes_authz_coverage_test.go): Diese Antworten
// verknüpfen Schülernamen mit Titeln und dürfen nie hinter ein Recht rutschen,
// das die Helfer-Rolle ab Werk besitzt (view_books, perform_actions).
//
// Zwei Schutzstufen: Ausleiher/Historie zeigen das volle Ausleihgeschehen und
// bleiben hinter view_students. Die Vormerkungen hängen am ENGEN Theken-Recht
// manage_vormerkungen (Kiosk-Felder: Name+Klasse; Betreiber-Entscheidung
// 18.08.2026: für Helfer optional per Rechte-Matrix zuschaltbar, Standard AUS —
// db/seed.go RechteVorgabe + RechteOptional).
func TestPersonendatenRoutenHinterViewStudents(t *testing.T) {
	quelle, err := os.ReadFile("routes_books.go")
	if err != nil {
		t.Fatalf("routes_books.go lesen: %v", err)
	}
	for route, recht := range map[string]string{
		"GET /api/vormerkungen":                 "manage_vormerkungen",
		"POST /api/vormerkungen":                "manage_vormerkungen",
		"DELETE /api/vormerkungen/{id}":         "manage_vormerkungen",
		"GET /api/buecher/titel/{id}/ausleiher": "view_students",
		"GET /api/buecher/titel/{id}/historie":  "view_students",
	} {
		zeile := ""
		for _, l := range strings.Split(string(quelle), "\n") {
			if strings.Contains(l, "\""+route+"\"") {
				zeile = l
				break
			}
		}
		if zeile == "" {
			t.Errorf("Route %q nicht in routes_books.go gefunden — bei Umzug diesen Test mitnehmen", route)
			continue
		}
		if !strings.Contains(zeile, `RequirePermission("`+recht+`")`) {
			t.Errorf("Route %q gibt Schülernamen heraus und muss hinter %s liegen, gefunden: %s", route, recht, strings.TrimSpace(zeile))
		}
	}
}

// TestVormerkungsRechtIstFuerHelferOptional hält die Entscheidung als Code fest:
// Das Paar HELFER/manage_vormerkungen existiert in der Vorgabe, startet AUS und
// ist als optional markiert — nur dann schweigt die Selbstprüfung, wenn ein
// Admin es einschaltet. Fällt eine der drei Bedingungen, ist entweder das Recht
// verschwunden, plötzlich Standard-AN oder jede nutzende Anlage dauerhaft gelb.
func TestVormerkungsRechtIstFuerHelferOptional(t *testing.T) {
	if !db.RechteOptional["HELFER/manage_vormerkungen"] {
		t.Error("HELFER/manage_vormerkungen fehlt in db.RechteOptional — die Selbstprüfung würde jeden gesetzten Haken als Drift melden")
	}
	gefunden := false
	for _, v := range db.RechteVorgabe {
		if v.Role == "HELFER" && v.Permission == "manage_vormerkungen" {
			gefunden = true
			if v.Allowed {
				t.Error("HELFER/manage_vormerkungen muss ab Werk AUS sein (optionales Recht, Betreiber-Entscheidung)")
			}
		}
	}
	if !gefunden {
		t.Error("HELFER/manage_vormerkungen fehlt in db.RechteVorgabe")
	}
}

type geraeteListenStub struct {
	repository.GeraeteRepository
	zeilen []repository.GeraetMitStatus
}

func (g *geraeteListenStub) ListGeraete(_ context.Context) ([]repository.GeraetMitStatus, error) {
	kopie := make([]repository.GeraetMitStatus, len(g.zeilen))
	copy(kopie, g.zeilen)
	return kopie, nil
}

func geraeteRequest(rolle string) *httptest.ResponseRecorder {
	name := "Nio Test (6c)"
	stub := &geraeteListenStub{zeilen: []repository.GeraetMitStatus{{
		Geraet:        repository.Geraet{ID: "g1", Modellname: "iPad", BarcodeID: "G-1"},
		AusgeliehenAn: &name,
	}}}
	s := &Server{}
	req := httptest.NewRequest("GET", "/api/geraete", nil)
	req = req.WithContext(context.WithValue(req.Context(), auth.ClaimsContextKey, &auth.Claims{Rolle: auth.Role(rolle)}))
	w := httptest.NewRecorder()
	s.ListGeraeteHandler(stub)(w, req)
	return w
}

// TestGeraetelisteBlendetAusleiherOhneViewStudentsAus: Der Helfer sieht, DASS
// ein Gerät verliehen ist (ausgeliehen_an = ""), aber nicht an WEN. Ein Admin
// sieht den Namen. Der Cache wird gezielt vorbefüllt, damit der Test ohne
// Datenbank läuft und trotzdem den echten Rechte-Pfad nimmt.
func TestGeraetelisteBlendetAusleiherOhneViewStudentsAus(t *testing.T) {
	InvalidatePermissionCache()
	t.Cleanup(InvalidatePermissionCache)
	permCacheMu.Lock()
	permCache["helfer:view_students"] = cacheEntry{Allowed: false, ExpiresAt: time.Now().Add(time.Minute)}
	permCacheMu.Unlock()

	helferAntwort := geraeteRequest("helfer").Body.String()
	if strings.Contains(helferAntwort, "Nio Test") {
		t.Errorf("Helfer sieht den Ausleihernamen: %s", helferAntwort)
	}
	var helferDaten struct {
		Data []map[string]any `json:"data"`
	}
	if err := json.Unmarshal([]byte(helferAntwort), &helferDaten); err != nil || len(helferDaten.Data) != 1 {
		t.Fatalf("Geräteliste unlesbar: %v / %s", err, helferAntwort)
	}
	if wert, ok := helferDaten.Data[0]["ausgeliehen_an"]; !ok || wert != "" {
		t.Errorf("verliehen-Status muss als leerer Name sichtbar bleiben, got %v", helferDaten.Data[0]["ausgeliehen_an"])
	}

	adminAntwort := geraeteRequest("admin").Body.String()
	if !strings.Contains(adminAntwort, "Nio Test") {
		t.Errorf("Admin muss den Ausleihernamen weiterhin sehen: %s", adminAntwort)
	}
}
