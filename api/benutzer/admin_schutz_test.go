package benutzer

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"bibliothek/api/tuer"
	"bibliothek/auth"
	"bibliothek/repository"
)

// eskalationsAttrappe erfüllt KontoAbfragen und beantwortet nur die Fragen, die die Trennung der
// Rechte braucht. Die eingebettete Schnittstelle hält die Attrappe klein: Jede Methode, die
// sie nicht überschreibt, bricht beim Aufruf ab und zeigt so, dass die Tür mehr tut als hier
// abgebildet.
type eskalationsAttrappe struct {
	repository.KontoAbfragen
	rollen    map[string]string // Kennung des Kontos → aktuelle Rolle
	geaendert bool
	angelegt  bool
}

func (a *eskalationsAttrappe) LadeRolle(_ context.Context, id string) (string, error) {
	return a.rollen[id], nil
}

func (a *eskalationsAttrappe) IstEmailVergeben(_ context.Context, _, _ string) (bool, error) {
	return false, nil
}

func (a *eskalationsAttrappe) IstAusweisnummerVergeben(_ context.Context, _, _ string) (bool, error) {
	return false, nil
}

func (a *eskalationsAttrappe) HatLeserzeileOhneKonto(_ context.Context, _, _ string) (bool, error) {
	return false, nil
}

func (a *eskalationsAttrappe) LegeAn(_ context.Context, _ *string, _, _, _, _ string) (string, error) {
	a.angelegt = true
	return "neu-1", nil
}

func (a *eskalationsAttrappe) Aendere(_ context.Context, _ repository.KontoAenderung) (repository.KontoStand, error) {
	a.geaendert = true
	return repository.KontoStand{}, nil
}

// anfrageAls baut eine Anfrage mit Sitzungs-Claims. rolle ist die Rolle des Aufrufers,
// so wie auth.VerifyToken sie bei jeder Anfrage frisch aus der Datenbank setzt.
func anfrageAls(t *testing.T, methode, pfad, rumpf string, userID string, rolle auth.Role) *http.Request {
	t.Helper()
	req := httptest.NewRequest(methode, pfad, bytes.NewBufferString(rumpf))
	req.Header.Set("Content-Type", "application/json")
	claims := &auth.Claims{UserID: userID, Rolle: rolle}
	return req.WithContext(context.WithValue(req.Context(), auth.ClaimsContextKey, claims))
}

const rumpfAdmin = `{"vorname":"Eva","nachname":"Neu","email":"eva@schule.de","rolle":"admin","aktiv":true}`

// Weg 1: Selbstbeförderung. Ein Mitarbeiter mit manage_users bearbeitet den eigenen
// Datensatz und trägt sich als Admin ein.
func TestSelbstbefoerderungZumAdminWirdAbgelehnt(t *testing.T) {
	repo := &eskalationsAttrappe{rollen: map[string]string{"mitarbeiter-1": "MITARBEITER"}}
	srv := &Tueren{Basis: &tuer.Basis{}}

	req := anfrageAls(t, http.MethodPut, "/api/benutzer/mitarbeiter-1", rumpfAdmin, "mitarbeiter-1", auth.RoleMitarbeiter)
	req.SetPathValue("id", "mitarbeiter-1")
	w := httptest.NewRecorder()

	srv.KontoAendern(repo).ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("Selbstbeförderung zum Admin: Status %d, erwartet 403", w.Code)
	}
	if repo.geaendert {
		t.Error("es wurde trotz Ablehnung geschrieben — die Prüfung sitzt hinter dem Schreibzugriff")
	}
}

// Die Gegenprobe zu Weg 1: Dasselbe Konto muss seinen eigenen Datensatz mit unveränderter
// Rolle speichern können. Ein Selbstschutz, der jeden, der sich selbst bearbeitet, für einen
// Admin hält, lehnte genau das ab.
func TestEigenerDatensatzMitGleicherRolleBleibtSpeicherbar(t *testing.T) {
	repo := &eskalationsAttrappe{rollen: map[string]string{"mitarbeiter-1": "MITARBEITER"}}
	srv := &Tueren{Basis: &tuer.Basis{}}

	rumpf := `{"vorname":"Max","nachname":"Neuername","email":"max@schule.de","rolle":"mitarbeiter","aktiv":true}`
	req := anfrageAls(t, http.MethodPut, "/api/benutzer/mitarbeiter-1", rumpf, "mitarbeiter-1", auth.RoleMitarbeiter)
	req.SetPathValue("id", "mitarbeiter-1")
	w := httptest.NewRecorder()

	srv.KontoAendern(repo).ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("eigener Datensatz mit gleicher Rolle: Status %d, erwartet 200 (Rumpf: %s)", w.Code, w.Body.String())
	}
	if !repo.geaendert {
		t.Error("es wurde nichts geschrieben")
	}
}

// Weg 2: Fremdbeförderung, ein zweites Konto zum Admin machen.
func TestFremdbefoerderungZumAdminWirdAbgelehnt(t *testing.T) {
	repo := &eskalationsAttrappe{rollen: map[string]string{"kollege-1": "HELFER"}}
	srv := &Tueren{Basis: &tuer.Basis{}}

	req := anfrageAls(t, http.MethodPut, "/api/benutzer/kollege-1", rumpfAdmin, "mitarbeiter-1", auth.RoleMitarbeiter)
	req.SetPathValue("id", "kollege-1")
	w := httptest.NewRecorder()

	srv.KontoAendern(repo).ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("Fremdbeförderung: Status %d, erwartet 403", w.Code)
	}
	if repo.geaendert {
		t.Error("es wurde trotz Ablehnung geschrieben")
	}
}

// Weg 2b: Denselben Weg über die Neuanlage.
func TestAdminNeuanlageOhneAdminrechteWirdAbgelehnt(t *testing.T) {
	repo := &eskalationsAttrappe{rollen: map[string]string{}}
	srv := &Tueren{Basis: &tuer.Basis{}}

	req := anfrageAls(t, http.MethodPost, "/api/benutzer", rumpfAdmin, "mitarbeiter-1", auth.RoleMitarbeiter)
	w := httptest.NewRecorder()

	srv.KontoAnlegen(repo).ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("Admin-Neuanlage: Status %d, erwartet 403", w.Code)
	}
	if repo.angelegt {
		t.Error("der Admin wurde trotz Ablehnung angelegt")
	}
}

// Weg 3: Übernahme eines Admin-Kontos ohne jede Änderung der Rolle.
//
// Die Anmeldung prüft per IMAP und sucht das Konto danach über seine E-Mail
// (auth/handlers.go). Wer die E-Mail des Admin-Datensatzes auf die eigene Schuladresse
// umschreibt, meldet sich anschließend mit den eigenen Zugangsdaten an und erhält die
// Admin-Sitzung. Die Rolle im Rumpf bleibt dabei unverdächtig "admin".
func TestEmailUebernahmeEinesAdminKontosWirdAbgelehnt(t *testing.T) {
	repo := &eskalationsAttrappe{rollen: map[string]string{"admin-1": "ADMIN"}}
	srv := &Tueren{Basis: &tuer.Basis{}}

	rumpf := `{"vorname":"Chef","nachname":"Chefin","email":"angreifer@schule.de","rolle":"admin","aktiv":true}`
	req := anfrageAls(t, http.MethodPut, "/api/benutzer/admin-1", rumpf, "mitarbeiter-1", auth.RoleMitarbeiter)
	req.SetPathValue("id", "admin-1")
	w := httptest.NewRecorder()

	srv.KontoAendern(repo).ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("E-Mail-Übernahme eines Admin-Kontos: Status %d, erwartet 403", w.Code)
	}
	if repo.geaendert {
		t.Error("die E-Mail des Admins wurde trotz Ablehnung geschrieben")
	}
}

// Weg 3b: derselbe Angriff, aber mit Herabstufung statt Beibehaltung der Admin-Rolle.
//
// Dieser Fall ist der eigentliche Prüfstein für pruefeAdminZiel: Weil im Rumpf nicht
// "admin" steht, greift pruefeAdminVergabe hier nicht. Hängen bleibt der Versuch allein
// daran, dass das Ziel heute ein Administrator ist. Ohne diesen Test könnte man
// pruefeAdminZiel entfernen, ohne dass eine Prüfung rot wird — und mit ihm den Schutz gegen
// „Admin herabstufen, E-Mail übernehmen, danach in Ruhe wieder hochstufen".
func TestAdminHerabstufenDurchNichtAdminWirdAbgelehnt(t *testing.T) {
	repo := &eskalationsAttrappe{rollen: map[string]string{"admin-1": "ADMIN"}}
	srv := &Tueren{Basis: &tuer.Basis{}}

	rumpf := `{"vorname":"Chef","nachname":"Chefin","email":"angreifer@schule.de","rolle":"mitarbeiter","aktiv":true}`
	req := anfrageAls(t, http.MethodPut, "/api/benutzer/admin-1", rumpf, "mitarbeiter-1", auth.RoleMitarbeiter)
	req.SetPathValue("id", "admin-1")
	w := httptest.NewRecorder()

	srv.KontoAendern(repo).ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("Herabstufung eines Admins durch Nicht-Admin: Status %d, erwartet 403", w.Code)
	}
	if repo.geaendert {
		t.Error("der Admin-Datensatz wurde trotz Ablehnung geschrieben")
	}
}

// Ein Administrator muss all das weiterhin dürfen — sonst ist die Kontoverwaltung kaputt.
func TestAdminDarfWeiterhinAdminsAnlegenUndBearbeiten(t *testing.T) {
	repo := &eskalationsAttrappe{rollen: map[string]string{"admin-2": "ADMIN"}}
	srv := &Tueren{Basis: &tuer.Basis{}}

	neu := anfrageAls(t, http.MethodPost, "/api/benutzer", rumpfAdmin, "admin-1", auth.RoleAdmin)
	wNeu := httptest.NewRecorder()
	srv.KontoAnlegen(repo).ServeHTTP(wNeu, neu)
	if wNeu.Code != http.StatusOK {
		t.Errorf("Admin legt Admin an: Status %d, erwartet 200 (%s)", wNeu.Code, wNeu.Body.String())
	}

	aend := anfrageAls(t, http.MethodPut, "/api/benutzer/admin-2", rumpfAdmin, "admin-1", auth.RoleAdmin)
	aend.SetPathValue("id", "admin-2")
	wAend := httptest.NewRecorder()
	srv.KontoAendern(repo).ServeHTTP(wAend, aend)
	if wAend.Code != http.StatusOK {
		t.Errorf("Admin bearbeitet Admin: Status %d, erwartet 200 (%s)", wAend.Code, wAend.Body.String())
	}
}

// Und der Normalfall bleibt offen: manage_users verwaltet Konten unterhalb der
// Administratorebene weiterhin ohne Einschränkung.
func TestManageUsersDarfNichtAdminKontenWeiterhinBearbeiten(t *testing.T) {
	repo := &eskalationsAttrappe{rollen: map[string]string{"helfer-1": "HELFER"}}
	srv := &Tueren{Basis: &tuer.Basis{}}

	rumpf := `{"vorname":"Hilde","nachname":"Helfer","email":"hilde@schule.de","rolle":"helfer","aktiv":true}`
	req := anfrageAls(t, http.MethodPut, "/api/benutzer/helfer-1", rumpf, "mitarbeiter-1", auth.RoleMitarbeiter)
	req.SetPathValue("id", "helfer-1")
	w := httptest.NewRecorder()

	srv.KontoAendern(repo).ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Helfer bearbeiten: Status %d, erwartet 200 (%s)", w.Code, w.Body.String())
	}
	if !repo.geaendert {
		t.Error("die Änderung am Helfer-Konto kam nicht an")
	}
}

// Der Selbstschutz hält auch für Administratoren: Ein Administrator darf
// sich nicht selbst herabstufen und damit womöglich den letzten Admin-Zugang schließen.
//
// Nur pruefeSelbstschutz fängt diesen Fall: Die Admin-Rolle wird nicht vergeben (also
// kein Fall für pruefeAdminVergabe), und der Aufrufer ist Admin (also kein Fall für
// pruefeAdminZiel).
func TestAdminKannSichNichtSelbstHerabstufen(t *testing.T) {
	repo := &eskalationsAttrappe{rollen: map[string]string{"admin-1": "ADMIN"}}
	srv := &Tueren{Basis: &tuer.Basis{}}

	rumpf := `{"vorname":"Chef","nachname":"Chefin","email":"chef@schule.de","rolle":"mitarbeiter","aktiv":true}`
	req := anfrageAls(t, http.MethodPut, "/api/benutzer/admin-1", rumpf, "admin-1", auth.RoleAdmin)
	req.SetPathValue("id", "admin-1")
	w := httptest.NewRecorder()

	srv.KontoAendern(repo).ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("Selbstherabstufung eines Admins: Status %d, erwartet 403", w.Code)
	}
	if repo.geaendert {
		t.Error("die Herabstufung wurde trotz Ablehnung geschrieben")
	}
}

// Die eigene Abmeldung aus dem System bleibt ebenfalls gesperrt.
func TestEigeneDeaktivierungWirdAbgelehnt(t *testing.T) {
	repo := &eskalationsAttrappe{rollen: map[string]string{"admin-1": "ADMIN"}}
	srv := &Tueren{Basis: &tuer.Basis{}}

	rumpf := `{"vorname":"Chef","nachname":"Chefin","email":"chef@schule.de","rolle":"admin","aktiv":false}`
	req := anfrageAls(t, http.MethodPut, "/api/benutzer/admin-1", rumpf, "admin-1", auth.RoleAdmin)
	req.SetPathValue("id", "admin-1")
	w := httptest.NewRecorder()

	srv.KontoAendern(repo).ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("Selbstdeaktivierung: Status %d, erwartet 403", w.Code)
	}
}

// Die Schutzregeln gelten auch, wenn der Rumpf nur einzelne Felder nennt: Ein fehlendes Feld
// ändert nichts und gibt nichts frei; ein genanntes wird geprüft.
func TestSchutzregelnBeiEinzelnenFeldern(t *testing.T) {
	faelle := []struct {
		name          string
		rollen        map[string]string
		ziel, rumpf   string
		aufrufer      string
		aufruferRolle auth.Role
		status        int
		geschrieben   bool
	}{
		{"das eigene Konto abschalten", map[string]string{"mitarbeiter-1": "MITARBEITER"},
			"mitarbeiter-1", `{"aktiv":false}`, "mitarbeiter-1", auth.RoleMitarbeiter, http.StatusForbidden, false},
		{"sich selbst zum Administrator machen", map[string]string{"mitarbeiter-1": "MITARBEITER"},
			"mitarbeiter-1", `{"rolle":"admin"}`, "mitarbeiter-1", auth.RoleMitarbeiter, http.StatusForbidden, false},
		{"den eigenen Namen ändern", map[string]string{"mitarbeiter-1": "MITARBEITER"},
			"mitarbeiter-1", `{"vorname":"Max"}`, "mitarbeiter-1", auth.RoleMitarbeiter, http.StatusOK, true},
		{"den Namen eines Administrators ändern, ohne einer zu sein", map[string]string{"chef-1": "ADMIN"},
			"chef-1", `{"vorname":"Max"}`, "mitarbeiter-1", auth.RoleMitarbeiter, http.StatusForbidden, false},
		{"die Adresse eines Administrators ändern, ohne einer zu sein", map[string]string{"chef-1": "ADMIN"},
			"chef-1", `{"email":"ich@schule.de"}`, "mitarbeiter-1", auth.RoleMitarbeiter, http.StatusForbidden, false},
		{"einen anderen zum Administrator machen, ohne einer zu sein", map[string]string{"kollege-1": "HELFER"},
			"kollege-1", `{"rolle":"admin"}`, "mitarbeiter-1", auth.RoleMitarbeiter, http.StatusForbidden, false},
		{"als Administrator eine Rolle vergeben", map[string]string{"kollege-1": "HELFER"},
			"kollege-1", `{"rolle":"admin"}`, "chef-1", auth.RoleAdmin, http.StatusOK, true},
		{"ein Rumpf ohne Feld schreibt nichts", map[string]string{"kollege-1": "HELFER"},
			"kollege-1", `{}`, "chef-1", auth.RoleAdmin, http.StatusOK, false},
		{"ein Rumpf ohne Feld an ein unbekanntes Konto", map[string]string{},
			"niemand-1", `{}`, "chef-1", auth.RoleAdmin, http.StatusNotFound, false},
	}
	for _, f := range faelle {
		t.Run(f.name, func(t *testing.T) {
			repo := &eskalationsAttrappe{rollen: f.rollen}
			req := anfrageAls(t, http.MethodPut, "/api/benutzer/"+f.ziel, f.rumpf, f.aufrufer, f.aufruferRolle)
			req.SetPathValue("id", f.ziel)
			w := httptest.NewRecorder()

			(&Tueren{Basis: &tuer.Basis{}}).KontoAendern(repo).ServeHTTP(w, req)

			if w.Code != f.status {
				t.Errorf("Status %d, erwartet %d (%s)", w.Code, f.status, w.Body.String())
			}
			if repo.geaendert != f.geschrieben {
				t.Errorf("geschrieben = %v, erwartet %v", repo.geaendert, f.geschrieben)
			}
		})
	}
}
