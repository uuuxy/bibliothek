package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"bibliothek/auth"
	"bibliothek/db"
	"bibliothek/internal/service"
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

// TestSperrgrundBleibtOhneViewStudentsGeheim: Der Freitext aus schueler.block_reason
// kann Zahlungsrückstände oder Familieninterna nennen (PII-Matrix Stufe 2). Die Theke
// erfährt DASS gesperrt ist — das WARUM nur mit view_students. Am 18.08.2026 live mit
// Helfer-Konto belegt: vor dem Fix kam der volle Text („… Fall Jugendamt") durch.
func TestSperrgrundBleibtOhneViewStudentsGeheim(t *testing.T) {
	grund := "Eltern zahlen Schadensrechnung nicht - Fall Jugendamt"
	sperrFehler := &service.SperrGrundFehler{
		Kern:  fmt.Errorf("%w: Manuelle Sperre", service.ErrBlocked),
		Grund: grund,
	}

	ohne := ohneSperrgrund(sperrFehler, false)
	if strings.Contains(ohne.Error(), grund) {
		t.Errorf("Sperrgrund erreicht Aufrufer ohne view_students: %s", ohne)
	}
	if !errors.Is(ohne, service.ErrBlocked) {
		t.Error("gekürzter Fehler verliert ErrBlocked — der HTTP-Status würde von 403 auf 500 kippen")
	}

	mit := ohneSperrgrund(sperrFehler, true)
	if !strings.Contains(mit.Error(), grund) {
		t.Errorf("berechtigte Aufrufer müssen den Grund sehen (Theken-Arbeitsfähigkeit): %s", mit)
	}

	// Andere Fehler (auch ErrBlocked ohne Freitext, z. B. Ausleihlimit) unverändert.
	limit := fmt.Errorf("%w: Ausleihlimit von 5 Büchern überschritten", service.ErrBlocked)
	if ohneSperrgrund(limit, false) != limit {
		t.Error("Fehler ohne Sperr-Freitext dürfen nicht angefasst werden")
	}
	if ohneSperrgrund(nil, false) != nil {
		t.Error("nil bleibt nil")
	}
}

// overrideSpyOmnibox merkt sich, mit welchem OverrideBlock der Ausleih-Kern
// aufgerufen wurde — genau der Wert, der über eine Sperre entscheidet.
type overrideSpyOmnibox struct {
	gesehenerOverride bool
}

func (o *overrideSpyOmnibox) ProcessQuery(_ context.Context, q service.OmniboxQuery) (*service.OmniboxResult, error) {
	o.gesehenerOverride = q.OverrideBlock
	return &service.OmniboxResult{Type: "info", Message: "ok"}, nil
}

func actionSiehtOverride(rolle string) bool {
	spy := &overrideSpyOmnibox{}
	s := &Server{}
	body := `{"query":"B-123","active_student_id":"s1","override_block":true}`
	req := httptest.NewRequest("POST", "/api/action", strings.NewReader(body))
	req = req.WithContext(context.WithValue(req.Context(), auth.ClaimsContextKey, &auth.Claims{Rolle: auth.Role(rolle), UserID: "u1"}))
	w := httptest.NewRecorder()
	s.ActionHandler(spy)(w, req)
	return spy.gesehenerOverride
}

// TestOverrideBlockNurMitEditStudents: Eine Sperre aufheben ist ein Verwaltungs-
// akt. override_block darf nur wirken, wenn der Aufrufer edit_students besitzt —
// dasselbe Recht, das auch das Sperren/Entsperren erlaubt. Am 18.08.2026 live mit
// Helfer-Konto belegt: perform_actions allein reichte, die Sperre fiel per
// Request-Feld, obwohl die UI den Schalter nie anbot (Schutz nur als Konvention).
func TestOverrideBlockNurMitEditStudents(t *testing.T) {
	InvalidatePermissionCache()
	t.Cleanup(InvalidatePermissionCache)
	permCacheMu.Lock()
	permCache["helfer:edit_students"] = cacheEntry{Allowed: false, ExpiresAt: time.Now().Add(time.Minute)}
	permCache["mitarbeiter:edit_students"] = cacheEntry{Allowed: true, ExpiresAt: time.Now().Add(time.Minute)}
	permCacheMu.Unlock()

	if actionSiehtOverride("helfer") {
		t.Error("Helfer (ohne edit_students) konnte override_block durchsetzen — die Sperre ist aushebelbar")
	}
	if !actionSiehtOverride("mitarbeiter") {
		t.Error("Mitarbeiter mit edit_students muss override_block nutzen können — sonst ist keine berechtigte Sperr-Aufhebung mehr möglich")
	}
	if !actionSiehtOverride("admin") {
		t.Error("Admin muss override_block nutzen können")
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
