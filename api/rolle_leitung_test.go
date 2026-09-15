package api

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"bibliothek/auth"
)

// Die Rolle Leitung muss vergebbar sein. normalisiereBenutzerRolle ist die Stelle, an
// der eine unbekannte Rolle auf die harmloseste zurückfällt — stünde „leitung" nicht in
// der Liste, würde jede Vergabe still zu „mitarbeiter", und die Auswahl in der
// Benutzerverwaltung sähe nur so aus, als hätte sie gewirkt.
func TestNormalisiereBenutzerRolleKenntLeitung(t *testing.T) {
	faelle := map[string]string{
		"leitung":   "leitung",
		"LEITUNG":   "leitung",
		" Leitung ": "leitung",
		// Die Rückfallebene bleibt die harmloseste Rolle, nicht die nächstliegende:
		// Ein Tippfehler darf niemals nach oben führen.
		"leitungg": "mitarbeiter",
	}
	for eingabe, erwartet := range faelle {
		if got := normalisiereBenutzerRolle(eingabe); got != erwartet {
			t.Errorf("normalisiereBenutzerRolle(%q) = %q, erwartet %q", eingabe, got, erwartet)
		}
	}
}

// Die Leitung ist KEIN Administrator. Ab Werk hat sie kein manage_users und erreicht die
// Benutzerverwaltung gar nicht — aber der Admin kann ihr das Recht erteilen (es ist
// delegierbar, siehe user_admin_eskalation.go). Dann gilt dieselbe Grenze wie für einen
// Mitarbeiter: Wer einen Administrator anlegen oder verändern will, muss selbst einer
// sein. Diese Regel bestand schon; der Test hält sie für die neue Rolle fest, damit sie
// nicht beim nächsten Umbau an eine Rollenliste gerät, die „leitung" mitzählt.
func TestLeitungDarfKeinenAdminVergeben(t *testing.T) {
	repo := &eskalationsRepo{rollen: map[string]string{"leitung-1": "LEITUNG"}}
	srv := &Server{}

	req := anfrageAls(t, http.MethodPost, "/api/benutzer", rumpfAdmin, "leitung-1", auth.RoleLeitung)
	w := httptest.NewRecorder()

	srv.CreateUserHandler(repo).ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("Status = %d, erwartet 403 — eine Leitung darf keinen Administrator anlegen", w.Code)
	}
	if repo.angelegt {
		t.Error("es wurde ein Konto angelegt — die Prüfung lief nach dem Schreiben oder gar nicht")
	}
}

// Dieselbe Grenze in der anderen Richtung: Das KONTO eines Admins ist für eine Leitung
// tabu, gleich welches Feld sie ändern will. Der Grund ist nicht das Rollenfeld, sondern
// die E-Mail: Wer sie auf die eigene Adresse setzt, bekommt beim nächsten Login die
// Admin-Sitzung (auth/handlers.go sucht den Benutzer über seine E-Mail).
func TestLeitungDarfAdminKontoNichtBearbeiten(t *testing.T) {
	repo := &eskalationsRepo{rollen: map[string]string{
		"leitung-1": "LEITUNG",
		"admin-1":   "ADMIN",
	}}
	srv := &Server{}

	const rumpf = `{"vorname":"Eva","nachname":"Neu","email":"eva@schule.de","rolle":"admin","aktiv":true}`
	req := anfrageAls(t, http.MethodPut, "/api/benutzer/admin-1", rumpf, "leitung-1", auth.RoleLeitung)
	req.SetPathValue("id", "admin-1")
	w := httptest.NewRecorder()

	srv.UpdateUserHandler(repo).ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("Status = %d, erwartet 403 — eine Leitung darf ein Admin-Konto nicht bearbeiten", w.Code)
	}
	if repo.geaendert {
		t.Error("das Admin-Konto wurde geändert — die Prüfung lief nach dem Schreiben oder gar nicht")
	}
}
