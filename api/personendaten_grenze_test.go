package api

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"bibliothek/api/tuer"
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
// Quelle (dieselbe Technik wie geschuetzte_routen_test.go): Diese Antworten
// verknüpfen Schülernamen mit Titeln und dürfen nie hinter ein Recht rutschen,
// das die Helfer-Rolle ab Werk besitzt (view_books, perform_actions).
//
// Zwei Schutzstufen: Ausleiher/Historie zeigen das volle Ausleihgeschehen und
// bleiben hinter view_students. Die Vormerkungen hängen am ENGEN Theken-Recht
// manage_vormerkungen (Kiosk-Felder: Name+Klasse; Betreiber-Entscheidung
// 18.08.2026: für Helfer optional per Rechte-Matrix zuschaltbar, Standard AUS —
// db/seed.go RechteVorgabe + RechteOptional).
func TestPersonendatenRoutenHinterViewStudents(t *testing.T) {
	quelle, err := os.ReadFile("routen_katalog.go")
	if err != nil {
		t.Fatalf("routen_katalog.go lesen: %v", err)
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
			t.Errorf("Route %q nicht in routen_katalog.go gefunden — bei Umzug diesen Test mitnehmen", route)
			continue
		}
		if !strings.Contains(zeile, `verlangeRecht("`+recht+`")`) {
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

// sperreUebergehenAttrappe merkt sich, mit welchem OverrideBlock der Ausleih-Kern
// aufgerufen wurde — genau der Wert, der über eine Sperre entscheidet.
type sperreUebergehenAttrappe struct {
	gesehenesUebergehen bool
}

func (a *sperreUebergehenAttrappe) VerarbeiteEingabe(_ context.Context, q service.OmniboxEingabe) (*service.OmniboxErgebnis, error) {
	a.gesehenesUebergehen = q.SperreUebergehen
	return &service.OmniboxErgebnis{Art: "info", Meldung: "ok"}, nil
}

func eingabeSiehtUebergehen(rolle string) bool {
	spy := &sperreUebergehenAttrappe{}
	s := &Server{}
	// Echte UUID: Eine Schüler-ID, die keine ist, weist EingabeVerarbeiten seit dem 13.09.2026
	// vor der Rechteprüfung mit 400 ab (uuid_eingaben_test.go).
	body := `{"query":"B-123","active_leser_id":"22222222-2222-2222-2222-222222222222","override_block":true}`
	req := httptest.NewRequest("POST", "/api/action", strings.NewReader(body))
	req = req.WithContext(context.WithValue(req.Context(), auth.ClaimsContextKey, &auth.Claims{Rolle: auth.Role(rolle), UserID: "u1"}))
	w := httptest.NewRecorder()
	s.theke().EingabeVerarbeiten(spy)(w, req)
	return spy.gesehenesUebergehen
}

// TestSperreUebergehenNurMitEditStudents: Eine Sperre aufheben ist ein Verwaltungs-
// akt. override_block darf nur wirken, wenn der Aufrufer edit_students besitzt —
// dasselbe Recht, das auch das Sperren/Entsperren erlaubt. Am 18.08.2026 live mit
// Helfer-Konto belegt: perform_actions allein reichte, die Sperre fiel per
// Request-Feld, obwohl die UI den Schalter nie anbot (Schutz nur als Konvention).
func TestSperreUebergehenNurMitEditStudents(t *testing.T) {
	tuer.LeereRechteCache()
	t.Cleanup(tuer.LeereRechteCache)
	tuer.MerkeRechtFuerTest("helfer", "edit_students", false)
	tuer.MerkeRechtFuerTest("mitarbeiter", "edit_students", true)

	if eingabeSiehtUebergehen("helfer") {
		t.Error("Helfer (ohne edit_students) konnte override_block durchsetzen — die Sperre ist aushebelbar")
	}
	if !eingabeSiehtUebergehen("mitarbeiter") {
		t.Error("Mitarbeiter mit edit_students muss override_block nutzen können — sonst ist keine berechtigte Sperr-Aufhebung mehr möglich")
	}
	if !eingabeSiehtUebergehen("admin") {
		t.Error("Admin muss override_block nutzen können")
	}
}

type geraeteListenAttrappe struct {
	repository.GeraeteAbfragen
	zeilen []repository.GeraetMitStatus
}

func (a *geraeteListenAttrappe) LadeAlle(_ context.Context) ([]repository.GeraetMitStatus, error) {
	kopie := make([]repository.GeraetMitStatus, len(a.zeilen))
	copy(kopie, a.zeilen)
	return kopie, nil
}

func geraeteAnfrage(rolle string) *httptest.ResponseRecorder {
	name := "Nio Test (6c)"
	stub := &geraeteListenAttrappe{zeilen: []repository.GeraetMitStatus{{
		Geraet:        repository.Geraet{ID: "g1", Modellname: "iPad", BarcodeID: "G-1"},
		AusgeliehenAn: &name,
	}}}
	s := &Server{}
	req := httptest.NewRequest("GET", "/api/geraete", nil)
	req = req.WithContext(context.WithValue(req.Context(), auth.ClaimsContextKey, &auth.Claims{Rolle: auth.Role(rolle)}))
	w := httptest.NewRecorder()
	s.katalog().Geraete(stub)(w, req)
	return w
}

// TestGeraetelisteBlendetAusleiherOhneViewStudentsAus: Der Helfer sieht, DASS
// ein Gerät verliehen ist (ausgeliehen_an = ""), aber nicht an WEN. Ein Admin
// sieht den Namen. Der Cache wird gezielt vorbefüllt, damit der Test ohne
// Datenbank läuft und trotzdem den echten Rechte-Pfad nimmt.
func TestGeraetelisteBlendetAusleiherOhneViewStudentsAus(t *testing.T) {
	tuer.LeereRechteCache()
	t.Cleanup(tuer.LeereRechteCache)
	tuer.MerkeRechtFuerTest("helfer", "view_students", false)

	helferAntwort := geraeteAnfrage("helfer").Body.String()
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

	adminAntwort := geraeteAnfrage("admin").Body.String()
	if !strings.Contains(adminAntwort, "Nio Test") {
		t.Errorf("Admin muss den Ausleihernamen weiterhin sehen: %s", adminAntwort)
	}
}
