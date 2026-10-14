package tuertest

import (
	"testing"
	"time"

	"bibliothek/auth"
	"bibliothek/internal/testdaten"

	"github.com/jackc/pgx/v5/pgxpool"
)

// AdminSitzung legt ein Admin-Konto an und stellt ihm eine Sitzung aus. Ein Test, der eine Tür
// über den ganzen Router prüft, baut mit der Anmeldung den Server und schickt die Sitzung als
// Cookie session_token mit.
func AdminSitzung(t *testing.T, pool *pgxpool.Pool, email, vorname, nachname string) (adminID, sitzung string, anmeldung *auth.Authenticator) {
	t.Helper()
	anmeldung, err := auth.NewAuthenticator("pg-test-sitzungsgeheimnis-32-zeichen!!", pool, time.Hour)
	if err != nil {
		t.Fatalf("Authenticator: %v", err)
	}
	adminID = testdaten.LegeAdminKontoAn(t, pool, email, vorname, nachname)
	sitzung, err = anmeldung.GenerateToken(adminID, "PG-TEST-1", auth.RoleAdmin, "")
	if err != nil {
		t.Fatalf("Sitzung: %v", err)
	}
	return adminID, sitzung, anmeldung
}
