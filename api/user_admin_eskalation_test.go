package api

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"bibliothek/auth"
	"bibliothek/repository"
)

// eskalationsRepo ist ein UserRepository, das nur die für die Rechtetrennung nötigen
// Fragen beantwortet. Das eingebettete Interface hält den Fake klein — jede nicht
// überschriebene Methode würde beim Aufruf panisch werden und damit anzeigen, dass der
// Handler mehr tut als hier abgebildet.
type eskalationsRepo struct {
	repository.UserRepository
	rollen    map[string]string // Benutzer-ID → aktuelle Rolle
	geaendert bool
	angelegt  bool
}

func (r *eskalationsRepo) GetRolleByID(_ context.Context, id string) (string, error) {
	return r.rollen[id], nil
}

func (r *eskalationsRepo) CheckEmailExists(_ context.Context, _, _ string) (bool, error) {
	return false, nil
}

func (r *eskalationsRepo) CheckBarcodeExists(_ context.Context, _, _ string) (bool, error) {
	return false, nil
}

func (r *eskalationsRepo) CreateUser(_ context.Context, _ *string, _, _, _, _ string) (string, error) {
	r.angelegt = true
	return "neu-1", nil
}

func (r *eskalationsRepo) UpdateUser(_ context.Context, _ repository.UpdateUserParams) error {
	r.geaendert = true
	return nil
}

// anfrageAls baut eine Anfrage mit Sitzungs-Claims. rolle ist die Rolle des AUFRUFERS,
// so wie auth.VerifyToken sie bei jeder Anfrage frisch aus der Datenbank setzt.
func anfrageAls(t *testing.T, methode, pfad, rumpf string, userID string, rolle auth.Role) *http.Request {
	t.Helper()
	req := httptest.NewRequest(methode, pfad, bytes.NewBufferString(rumpf))
	req.Header.Set("Content-Type", "application/json")
	claims := &auth.Claims{UserID: userID, Rolle: rolle}
	return req.WithContext(context.WithValue(req.Context(), auth.ClaimsContextKey, claims))
}

const rumpfAdmin = `{"vorname":"Eva","nachname":"Neu","email":"eva@schule.de","rolle":"admin","aktiv":true}`

// Weg 1: Selbstbeförderung. Ein MITARBEITER mit manage_users bearbeitet den EIGENEN
// Datensatz und trägt sich als Admin ein.
func TestSelbstbefoerderungZumAdminWirdAbgelehnt(t *testing.T) {
	repo := &eskalationsRepo{rollen: map[string]string{"mitarbeiter-1": "MITARBEITER"}}
	srv := &Server{}

	req := anfrageAls(t, http.MethodPut, "/api/benutzer/mitarbeiter-1", rumpfAdmin, "mitarbeiter-1", auth.RoleMitarbeiter)
	req.SetPathValue("id", "mitarbeiter-1")
	w := httptest.NewRecorder()

	srv.UpdateUserHandler(repo).ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("Selbstbeförderung zum Admin: Status %d, erwartet 403", w.Code)
	}
	if repo.geaendert {
		t.Error("es wurde trotz Ablehnung geschrieben — die Prüfung sitzt hinter dem Schreibzugriff")
	}
}

// Der Gegenbeweis zu Weg 1: Derselbe Benutzer muss seinen eigenen Datensatz mit
// UNVERÄNDERTER Rolle speichern können. Die Vorgängerfassung lehnte genau das ab
// („eigene Admin-Rolle kann nicht herabgestuft werden") und ließ nur die Beförderung zu.
func TestEigenerDatensatzMitGleicherRolleBleibtSpeicherbar(t *testing.T) {
	repo := &eskalationsRepo{rollen: map[string]string{"mitarbeiter-1": "MITARBEITER"}}
	srv := &Server{}

	rumpf := `{"vorname":"Max","nachname":"Neuername","email":"max@schule.de","rolle":"mitarbeiter","aktiv":true}`
	req := anfrageAls(t, http.MethodPut, "/api/benutzer/mitarbeiter-1", rumpf, "mitarbeiter-1", auth.RoleMitarbeiter)
	req.SetPathValue("id", "mitarbeiter-1")
	w := httptest.NewRecorder()

	srv.UpdateUserHandler(repo).ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("eigener Datensatz mit gleicher Rolle: Status %d, erwartet 200 (Rumpf: %s)", w.Code, w.Body.String())
	}
	if !repo.geaendert {
		t.Error("es wurde nichts geschrieben")
	}
}

// Weg 2: Fremdbeförderung — einen zweiten Benutzer zum Admin machen.
func TestFremdbefoerderungZumAdminWirdAbgelehnt(t *testing.T) {
	repo := &eskalationsRepo{rollen: map[string]string{"kollege-1": "HELFER"}}
	srv := &Server{}

	req := anfrageAls(t, http.MethodPut, "/api/benutzer/kollege-1", rumpfAdmin, "mitarbeiter-1", auth.RoleMitarbeiter)
	req.SetPathValue("id", "kollege-1")
	w := httptest.NewRecorder()

	srv.UpdateUserHandler(repo).ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("Fremdbeförderung: Status %d, erwartet 403", w.Code)
	}
	if repo.geaendert {
		t.Error("es wurde trotz Ablehnung geschrieben")
	}
}

// Weg 2b: Denselben Weg über die Neuanlage.
func TestAdminNeuanlageOhneAdminrechteWirdAbgelehnt(t *testing.T) {
	repo := &eskalationsRepo{rollen: map[string]string{}}
	srv := &Server{}

	req := anfrageAls(t, http.MethodPost, "/api/benutzer", rumpfAdmin, "mitarbeiter-1", auth.RoleMitarbeiter)
	w := httptest.NewRecorder()

	srv.CreateUserHandler(repo).ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("Admin-Neuanlage: Status %d, erwartet 403", w.Code)
	}
	if repo.angelegt {
		t.Error("der Admin wurde trotz Ablehnung angelegt")
	}
}

// Weg 3: Übernahme eines Admin-Kontos OHNE jede Rollenänderung.
//
// Die Anmeldung prüft per IMAP und sucht den Benutzer danach über seine E-Mail
// (auth/handlers.go). Wer die E-Mail des Admin-Datensatzes auf die eigene Schuladresse
// umschreibt, meldet sich anschließend mit den EIGENEN Zugangsdaten an und erhält die
// Admin-Sitzung. Die Rolle im Rumpf bleibt dabei unverdächtig "admin".
func TestEmailUebernahmeEinesAdminKontosWirdAbgelehnt(t *testing.T) {
	repo := &eskalationsRepo{rollen: map[string]string{"admin-1": "ADMIN"}}
	srv := &Server{}

	rumpf := `{"vorname":"Chef","nachname":"Chefin","email":"angreifer@schule.de","rolle":"admin","aktiv":true}`
	req := anfrageAls(t, http.MethodPut, "/api/benutzer/admin-1", rumpf, "mitarbeiter-1", auth.RoleMitarbeiter)
	req.SetPathValue("id", "admin-1")
	w := httptest.NewRecorder()

	srv.UpdateUserHandler(repo).ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("E-Mail-Übernahme eines Admin-Kontos: Status %d, erwartet 403", w.Code)
	}
	if repo.geaendert {
		t.Error("die E-Mail des Admins wurde trotz Ablehnung geschrieben")
	}
}

// Weg 3b: derselbe Angriff, aber mit HERABSTUFUNG statt Beibehaltung der Admin-Rolle.
//
// Dieser Fall ist der eigentliche Prüfstein für pruefeAdminZiel: Weil im Rumpf nicht
// "admin" steht, greift pruefeAdminVergabe hier nicht. Hängen bleibt der Versuch allein
// daran, dass das ZIEL heute ein Administrator ist. Ohne diesen Test könnte man
// pruefeAdminZiel entfernen, ohne dass ein Gate rot wird — und mit ihm den Schutz gegen
// „Admin herabstufen, E-Mail übernehmen, danach in Ruhe wieder hochstufen".
func TestAdminHerabstufenDurchNichtAdminWirdAbgelehnt(t *testing.T) {
	repo := &eskalationsRepo{rollen: map[string]string{"admin-1": "ADMIN"}}
	srv := &Server{}

	rumpf := `{"vorname":"Chef","nachname":"Chefin","email":"angreifer@schule.de","rolle":"mitarbeiter","aktiv":true}`
	req := anfrageAls(t, http.MethodPut, "/api/benutzer/admin-1", rumpf, "mitarbeiter-1", auth.RoleMitarbeiter)
	req.SetPathValue("id", "admin-1")
	w := httptest.NewRecorder()

	srv.UpdateUserHandler(repo).ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("Herabstufung eines Admins durch Nicht-Admin: Status %d, erwartet 403", w.Code)
	}
	if repo.geaendert {
		t.Error("der Admin-Datensatz wurde trotz Ablehnung geschrieben")
	}
}

// Ein Administrator muss all das weiterhin dürfen — sonst ist die Kontoverwaltung kaputt.
func TestAdminDarfWeiterhinAdminsAnlegenUndBearbeiten(t *testing.T) {
	repo := &eskalationsRepo{rollen: map[string]string{"admin-2": "ADMIN"}}
	srv := &Server{}

	neu := anfrageAls(t, http.MethodPost, "/api/benutzer", rumpfAdmin, "admin-1", auth.RoleAdmin)
	wNeu := httptest.NewRecorder()
	srv.CreateUserHandler(repo).ServeHTTP(wNeu, neu)
	if wNeu.Code != http.StatusOK {
		t.Errorf("Admin legt Admin an: Status %d, erwartet 200 (%s)", wNeu.Code, wNeu.Body.String())
	}

	aend := anfrageAls(t, http.MethodPut, "/api/benutzer/admin-2", rumpfAdmin, "admin-1", auth.RoleAdmin)
	aend.SetPathValue("id", "admin-2")
	wAend := httptest.NewRecorder()
	srv.UpdateUserHandler(repo).ServeHTTP(wAend, aend)
	if wAend.Code != http.StatusOK {
		t.Errorf("Admin bearbeitet Admin: Status %d, erwartet 200 (%s)", wAend.Code, wAend.Body.String())
	}
}

// Und der Normalfall bleibt offen: manage_users verwaltet Konten unterhalb der
// Administratorebene weiterhin ohne Einschränkung.
func TestManageUsersDarfNichtAdminKontenWeiterhinBearbeiten(t *testing.T) {
	repo := &eskalationsRepo{rollen: map[string]string{"helfer-1": "HELFER"}}
	srv := &Server{}

	rumpf := `{"vorname":"Hilde","nachname":"Helfer","email":"hilde@schule.de","rolle":"helfer","aktiv":true}`
	req := anfrageAls(t, http.MethodPut, "/api/benutzer/helfer-1", rumpf, "mitarbeiter-1", auth.RoleMitarbeiter)
	req.SetPathValue("id", "helfer-1")
	w := httptest.NewRecorder()

	srv.UpdateUserHandler(repo).ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Helfer bearbeiten: Status %d, erwartet 200 (%s)", w.Code, w.Body.String())
	}
	if !repo.geaendert {
		t.Error("die Änderung am Helfer-Konto kam nicht an")
	}
}

// Der ursprüngliche Zweck des Selbstschutzes bleibt erhalten: Ein Administrator darf
// sich nicht selbst herabstufen und damit womöglich den letzten Admin-Zugang schließen.
//
// Nur pruefeSelbstschutz fängt diesen Fall: Die Admin-Rolle wird nicht vergeben (also
// kein Fall für pruefeAdminVergabe), und der Aufrufer IST Admin (also kein Fall für
// pruefeAdminZiel).
func TestAdminKannSichNichtSelbstHerabstufen(t *testing.T) {
	repo := &eskalationsRepo{rollen: map[string]string{"admin-1": "ADMIN"}}
	srv := &Server{}

	rumpf := `{"vorname":"Chef","nachname":"Chefin","email":"chef@schule.de","rolle":"mitarbeiter","aktiv":true}`
	req := anfrageAls(t, http.MethodPut, "/api/benutzer/admin-1", rumpf, "admin-1", auth.RoleAdmin)
	req.SetPathValue("id", "admin-1")
	w := httptest.NewRecorder()

	srv.UpdateUserHandler(repo).ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("Selbstherabstufung eines Admins: Status %d, erwartet 403", w.Code)
	}
	if repo.geaendert {
		t.Error("die Herabstufung wurde trotz Ablehnung geschrieben")
	}
}

// Die eigene Abmeldung aus dem System bleibt ebenfalls gesperrt.
func TestEigeneDeaktivierungWirdAbgelehnt(t *testing.T) {
	repo := &eskalationsRepo{rollen: map[string]string{"admin-1": "ADMIN"}}
	srv := &Server{}

	rumpf := `{"vorname":"Chef","nachname":"Chefin","email":"chef@schule.de","rolle":"admin","aktiv":false}`
	req := anfrageAls(t, http.MethodPut, "/api/benutzer/admin-1", rumpf, "admin-1", auth.RoleAdmin)
	req.SetPathValue("id", "admin-1")
	w := httptest.NewRecorder()

	srv.UpdateUserHandler(repo).ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("Selbstdeaktivierung: Status %d, erwartet 403", w.Code)
	}
}
