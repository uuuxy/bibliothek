package benutzer

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"bibliothek/api/tuer"
	"bibliothek/auth"
)

// Die Rolle Leitung muss vergebbar sein. normalisiereRolle ist die Stelle, an
// der eine unbekannte Rolle auf die harmloseste zurückfällt — stünde „leitung" nicht in
// der Liste, würde jede Vergabe still zu „mitarbeiter", und die Auswahl in der
// Benutzerverwaltung sähe nur so aus, als hätte sie gewirkt.
func TestNormalisiereRolle_KenntLeitung(t *testing.T) {
	faelle := map[string]string{
		"leitung":   "leitung",
		"LEITUNG":   "leitung",
		" Leitung ": "leitung",
		// Die Rückfallebene bleibt die harmloseste Rolle, nicht die nächstliegende:
		// Ein Tippfehler darf niemals nach oben führen.
		"leitungg": "mitarbeiter",
	}
	for eingabe, erwartet := range faelle {
		if got := normalisiereRolle(eingabe); got != erwartet {
			t.Errorf("normalisiereRolle(%q) = %q, erwartet %q", eingabe, got, erwartet)
		}
	}
}

// Die Leitung ist kein Administrator. Ab Werk hat sie kein manage_users und erreicht die
// Benutzerverwaltung gar nicht — aber der Admin kann ihr das Recht erteilen (es ist
// delegierbar, siehe admin_schutz.go). Dann gilt dieselbe Grenze wie für einen
// Mitarbeiter: Wer einen Administrator anlegen oder verändern will, muss selbst einer sein.
// Der Test hält die Regel für diese Rolle fest, damit sie nicht an eine Rollenliste gerät,
// die „leitung" mitzählt.
func TestLeitungDarfKeinenAdminVergeben(t *testing.T) {
	repo := &eskalationsAttrappe{rollen: map[string]string{"leitung-1": "LEITUNG"}}
	srv := &Tueren{Basis: &tuer.Basis{}}

	req := anfrageAls(t, http.MethodPost, "/api/benutzer", rumpfAdmin, "leitung-1", auth.RolleLeitung)
	w := httptest.NewRecorder()

	srv.KontoAnlegen(repo).ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("Status = %d, erwartet 403 — eine Leitung darf keinen Administrator anlegen", w.Code)
	}
	if repo.angelegt {
		t.Error("es wurde ein Konto angelegt — die Prüfung lief nach dem Schreiben oder gar nicht")
	}
}

// Dieselbe Grenze in der anderen Richtung: Das Konto eines Admins ist für eine Leitung
// tabu, gleich welches Feld sie ändern will. Der Grund ist nicht das Rollenfeld, sondern
// die E-Mail: Wer sie auf die eigene Adresse setzt, bekommt beim nächsten Login die
// Admin-Sitzung (auth/anmelden.go sucht den Benutzer über seine E-Mail).
func TestLeitungDarfAdminKontoNichtBearbeiten(t *testing.T) {
	repo := &eskalationsAttrappe{rollen: map[string]string{
		"leitung-1": "LEITUNG",
		"admin-1":   "ADMIN",
	}}
	srv := &Tueren{Basis: &tuer.Basis{}}

	const rumpf = `{"vorname":"Eva","nachname":"Neu","email":"eva@schule.de","rolle":"admin","aktiv":true}`
	req := anfrageAls(t, http.MethodPut, "/api/benutzer/admin-1", rumpf, "leitung-1", auth.RolleLeitung)
	req.SetPathValue("id", "admin-1")
	w := httptest.NewRecorder()

	srv.KontoAendern(repo).ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("Status = %d, erwartet 403 — eine Leitung darf ein Admin-Konto nicht bearbeiten", w.Code)
	}
	if repo.geaendert {
		t.Error("das Admin-Konto wurde geändert — die Prüfung lief nach dem Schreiben oder gar nicht")
	}
}
