package benutzer

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"bibliothek/internal/api/tuer"
	"bibliothek/internal/auth"
)

// Ein Name aus Leerzeichen besteht `validate:"required"`. Die Neuanlage zählt ihn wie die
// Änderung (pruefeGenanntePflichtfelder) als leer und legt kein Konto an.
func TestKontoAnlegen_NameAusLeerzeichenAbgewiesen(t *testing.T) {
	for _, rumpf := range []string{
		`{"vorname":"   ","nachname":"Neu","email":"eva@schule.de","rolle":"mitarbeiter"}`,
		`{"vorname":"Eva","nachname":" \t ","email":"eva@schule.de","rolle":"mitarbeiter"}`,
	} {
		repo := &eskalationsAttrappe{rollen: map[string]string{}}
		srv := &Tueren{Basis: &tuer.Basis{}}
		w := httptest.NewRecorder()
		srv.KontoAnlegen(repo).ServeHTTP(w, anfrageAls(t, http.MethodPost, "/api/benutzer", rumpf, "admin-1", auth.RolleAdmin))
		if w.Code != http.StatusBadRequest {
			t.Errorf("%s: Status %d, erwartet 400", rumpf, w.Code)
		}
		if repo.angelegt {
			t.Errorf("%s: Konto ohne Namen angelegt", rumpf)
		}
	}
}

// Eine Rolle, die es nicht gibt, legt kein Konto an. Vorher wurde sie still „mitarbeiter“.
func TestKontoAnlegen_UnbekannteRolleAbgewiesen(t *testing.T) {
	for _, rolle := range []string{"lehrer", "kolleguim", "helferin"} {
		repo := &eskalationsAttrappe{rollen: map[string]string{}}
		srv := &Tueren{Basis: &tuer.Basis{}}
		w := httptest.NewRecorder()
		rumpf := `{"vorname":"Eva","nachname":"Neu","email":"eva@schule.de","rolle":"` + rolle + `"}`
		srv.KontoAnlegen(repo).ServeHTTP(w, anfrageAls(t, http.MethodPost, "/api/benutzer", rumpf, "admin-1", auth.RolleAdmin))
		if w.Code != http.StatusBadRequest {
			t.Errorf("Rolle %q: Status %d, erwartet 400", rolle, w.Code)
		}
		if repo.angelegt {
			t.Errorf("Rolle %q: Konto angelegt", rolle)
		}
	}
}
