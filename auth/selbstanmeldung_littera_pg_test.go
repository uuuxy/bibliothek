package auth

import (
	"context"
	"net/http"
	"strings"
	"testing"
)

// Eine aus Littera übernommene Lehrkraft trägt eine Platzhalter-Adresse
// (littera-<id>@littera.invalid, internal/littera/schreiber_personen.go), ihr Ausweis und ihre
// Ausleihen hängen an diesem Eintrag. Meldet sie sich mit ihrem Schulpostfach selbst an, findet
// die Anmeldung sie über die E-Mail nicht — es entsteht eine zweite, inaktive Zugangsanfrage
// (nachgestellt am 16.09.2026, OFFEN.md 5.16, Stufe 1).
//
// Das ist gewollt, soweit es die Sicherheit betrifft: Die Anmeldung verbindet NIE über den
// Namen. Der Name einer Selbstanmeldung ist aus der Adresse geraten, zwei Lehrkräfte können
// gleich heißen, und wer den Eintrag übernähme, bekäme fremde Ausleihen und einen fremden
// Ausweis. Die Verbindung stellt die Bibliothek beim Freischalten her; die Anfragen-Zeile der
// Benutzerverwaltung zeigt dafür den gleichnamigen Littera-Eintrag
// (UserManagementZugangsanfragen.svelte).
func TestSelbstanmeldung_UebernimmtKeinenLitteraEintragUeberDenNamen(t *testing.T) {
	t.Setenv("APP_ENV", "test")
	t.Setenv("IMAP_HOST", "mock")
	t.Setenv(selbstanmeldeDomainEnv, "selbsttest.invalid")

	pool := pgPoolFuerSelbstanmeldung(t)
	ctx := context.Background()
	const (
		schulAdresse   = "hanne.littera@selbsttest.invalid"
		platzhalter    = "littera-990011@littera.invalid"
		litteraAusweis = "L-990011"
	)
	raeumeKontoAb(t, pool, schulAdresse)
	raeumeKontoAb(t, pool, platzhalter)

	var litteraID, litteraLeserID string
	if err := pool.QueryRow(ctx, `
		INSERT INTO benutzer (vorname, nachname, email, rolle, aktiv)
		VALUES ('Hanne', 'Littera', $1, 'kollegium', true)
		RETURNING id, leser_id::text`, platzhalter).Scan(&litteraID, &litteraLeserID); err != nil {
		t.Fatalf("Littera-Eintrag anlegen: %v", err)
	}
	// Der Ausweis gehört zur Leserzeile (Migration 125).
	if _, err := pool.Exec(ctx, `UPDATE leser SET barcode_id = $1 WHERE id = $2`,
		litteraAusweis, litteraLeserID); err != nil {
		t.Fatalf("Ausweis eintragen: %v", err)
	}

	code, meldung := anmelden(t, pool, schulAdresse)
	if code != http.StatusForbidden || !strings.Contains(meldung, "beantragt") {
		t.Fatalf("Selbstanmeldung: Status %d (%q), erwartet 403 mit „beantragt“", code, meldung)
	}

	// Der Littera-Eintrag bleibt, wie er war: Adresse, Ausweis, aktiv.
	var email, ausweis string
	var aktiv bool
	if err := pool.QueryRow(ctx, `
		SELECT b.email, coalesce(l.barcode_id, ''), b.aktiv
		FROM benutzer b LEFT JOIN leser l ON l.id = b.leser_id WHERE b.id = $1`, litteraID).
		Scan(&email, &ausweis, &aktiv); err != nil {
		t.Fatalf("Littera-Eintrag lesen: %v", err)
	}
	if email != platzhalter || ausweis != litteraAusweis || !aktiv {
		t.Errorf("der Littera-Eintrag wurde verändert: E-Mail %q, Ausweis %q, aktiv %v", email, ausweis, aktiv)
	}

	// Die Anfrage ist ein eigener Eintrag, ohne Ausweis und inaktiv.
	var anfrageID, anfrageAusweis string
	var anfrageAktiv bool
	if err := pool.QueryRow(ctx, `
		SELECT b.id, coalesce(l.barcode_id, ''), b.aktiv
		FROM benutzer b LEFT JOIN leser l ON l.id = b.leser_id WHERE LOWER(b.email) = $1`, schulAdresse).
		Scan(&anfrageID, &anfrageAusweis, &anfrageAktiv); err != nil {
		t.Fatalf("Anfrage lesen: %v", err)
	}
	if anfrageID == litteraID {
		t.Fatal("die Anmeldung hat den Littera-Eintrag übernommen")
	}
	if anfrageAusweis != "" || anfrageAktiv {
		t.Errorf("die Anfrage trägt Ausweis %q, aktiv %v — erwartet ohne Ausweis und inaktiv", anfrageAusweis, anfrageAktiv)
	}
}
