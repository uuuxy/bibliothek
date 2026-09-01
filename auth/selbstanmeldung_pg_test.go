package auth

import (
	"bibliothek/internal/pgtest"
	"bibliothek/repository"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Der Live-Pfad der Selbstanmeldung, gegen echtes Postgres und durch den ECHTEN
// LoginHandler — nicht gegen die Hilfsfunktionen daneben.
//
// Der Kern ist eine Sicherheitszusage: Ein selbst angelegtes Konto darf sich NICHT
// anmelden, bevor es jemand freigeschaltet hat. Diese Zusage lässt sich nur am ganzen
// Weg prüfen. Eine isoliert getestete Funktion, die aktiv=false schreibt, beweist nichts
// darüber, was der Handler daraus macht — genau diese Lücke hat uns hier schon einmal
// eine Runde gekostet (Ghost-Block, Runde 8).

// Seit 01.09.2026 über internal/pgtest statt eines eigenen Harness: Der frühere
// Aufbau hier hielt den Advisory-Lock 0x42DB0001 SELBST und machte je Test einen
// eigenen DROP SCHEMA. Sobald ein zweiter Test im selben Binary den Lock über
// pgtest genommen hätte — der ihn bewusst bis Prozessende hält —, hätte dieser
// Harness für immer auf sein eigenes Binary gewartet; genau dieser Selbst-Deadlock
// hat am 01.09. über order_search die ganze Suite in 10-Minuten-Timeouts gezogen
// (de42b820). pgtest spielt schema.sql als die EINE Quelle ein — die Lektion vom
// 30.08. (abgeschriebene benutzer-DDL ohne Migration 086 = eine Woche CI-Münzwurf,
// b8daed0a) bleibt damit gewahrt. Das Schema wird je Binary EINMAL gebaut und mit
// allen Tests geteilt: Jeder Test räumt sein Konto über raeumeKontoAb selbst ab
// und prüft nur an seiner eigenen E-Mail-Adresse.
func pgPoolFuerSelbstanmeldung(t *testing.T) *pgxpool.Pool {
	t.Helper()
	pool := pgtest.Pool(t)
	ctx := context.Background()

	// role_permissions steht NICHT in schema.sql — die Tabelle legt db.InitPermissions
	// beim Serverstart an. Ohne sie scheitert der Login nach der Freischaltung am Laden
	// der Rechte, und zwar als 500: Der Fehlertext wird unterwegs durch „interner
	// Datenbankfehler" ersetzt, die Ursache stünde also nirgends. Hier nachgebaut, damit
	// der Test den echten Betriebszustand prüft und nicht einen halben.
	if _, err := pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS role_permissions (
			role VARCHAR(50) NOT NULL,
			permission VARCHAR(100) NOT NULL,
			allowed BOOLEAN NOT NULL DEFAULT false,
			PRIMARY KEY (role, permission)
		)`); err != nil {
		t.Fatalf("role_permissions anlegen: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO role_permissions (role, permission, allowed) VALUES ('KOLLEGIUM', 'view_books', true)
		ON CONFLICT DO NOTHING`); err != nil {
		t.Fatalf("Rechte seeden: %v", err)
	}
	return pool
}

// raeumeKontoAb entfernt am Testende das Konto zu dieser Adresse samt seiner
// Audit-Zeilen — pgtest teilt das Schema mit allen Tests des Binaries. Erst die
// Audit-Zeilen: audit_logs.admin_id steht auf ON DELETE SET NULL, ein nacktes
// DELETE auf benutzer ließe sie also verwaist zurück.
func raeumeKontoAb(t *testing.T, pool *pgxpool.Pool, email string) {
	t.Helper()
	t.Cleanup(func() {
		ctx := context.Background()
		if _, err := pool.Exec(ctx, `
			DELETE FROM audit_logs
			WHERE admin_id IN (SELECT id FROM benutzer WHERE LOWER(email) = $1)`, email); err != nil {
			t.Errorf("Aufräumen audit_logs: %v", err)
		}
		if _, err := pool.Exec(ctx, `DELETE FROM benutzer WHERE LOWER(email) = $1`, email); err != nil {
			t.Errorf("Aufräumen benutzer: %v", err)
		}
	})
}

// anmelden schickt eine echte Login-Anfrage durch den Handler.
func anmelden(t *testing.T, pool *pgxpool.Pool, email string) (int, string) {
	t.Helper()
	auth, err := NewAuthenticator("test-secret-mit-mindestens-32-zeichen!!", pool, time.Hour)
	if err != nil {
		t.Fatalf("Authenticator: %v", err)
	}
	rec := httptest.NewRecorder()
	body := strings.NewReader(`{"email":"` + email + `","password":"beliebig"}`)
	req := httptest.NewRequest(http.MethodPost, "/login", body)
	req.Header.Set("Content-Type", "application/json")

	LoginHandler(pool, auth, false)(rec, req)

	// Erfolgsantworten tragen kein error-Feld — ein Unmarshal-Fehler ist deshalb kein
	// Testfehler, der Statuscode bleibt aussagekräftig. Nur schweigend verwerfen wollen
	// wir ihn nicht.
	var antwort map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &antwort); err != nil {
		t.Logf("Antwort war kein JSON (Status %d): %s", rec.Code, rec.Body.String())
	}
	// errcheck läuft hier mit check-blank und check-type-assertions: Auch das verworfene
	// „ok" der Typzusicherung zählt als ignorierter Fehler. Erfolgsantworten tragen kein
	// error-Feld — dann bleibt der Text leer, und das ist kein Fehler.
	text, vorhanden := antwort["error"].(string)
	if !vorhanden {
		text = ""
	}
	return rec.Code, text
}

func TestSelbstanmeldung_LegtAnAberLaesstNichtRein(t *testing.T) {
	// IMAP-Mock: jedes Passwort gilt. Damit prüft der Test genau das, was NACH der
	// erfolgreichen Authentifizierung passiert — und nichts anderes.
	t.Setenv("APP_ENV", "test")
	t.Setenv("IMAP_HOST", "mock")
	t.Setenv(selbstanmeldeDomainEnv, "selbsttest.invalid")

	pool := pgPoolFuerSelbstanmeldung(t)
	ctx := context.Background()
	const email = "erika.musterfrau@selbsttest.invalid"
	raeumeKontoAb(t, pool, email)

	// 1. Erster Anmeldeversuch: Es gibt noch kein Konto.
	code, meldung := anmelden(t, pool, email)
	if code != http.StatusForbidden {
		t.Fatalf("erster Versuch: Status %d, erwartet 403 — ein neues Konto darf NICHT hineinkommen", code)
	}
	if !strings.Contains(meldung, "beantragt") {
		t.Errorf("die Meldung muss sagen, dass der Zugang beantragt ist, war: %q", meldung)
	}

	// 2. Der Eintrag existiert — inaktiv, Rolle kollegium, Name aus der Adresse geraten.
	var aktiv bool
	var rolle, vorname, nachname string
	if err := pool.QueryRow(ctx, `
		SELECT aktiv, rolle::text, vorname, nachname FROM benutzer WHERE LOWER(email) = $1
	`, email).Scan(&aktiv, &rolle, &vorname, &nachname); err != nil {
		t.Fatalf("angelegtes Konto lesen: %v", err)
	}
	if aktiv {
		t.Error("das angelegte Konto ist AKTIV — damit wäre die Freischaltung wirkungslos")
	}
	if rolle != "kollegium" {
		t.Errorf("Rolle = %q, erwartet kollegium", rolle)
	}
	if vorname != "Erika" || nachname != "Musterfrau" {
		t.Errorf("Name aus der Adresse = %q %q, erwartet Erika Musterfrau", vorname, nachname)
	}
	// Der Antrag ist als solcher markiert (Migration 086) — sonst wäre er in der
	// Benutzerverwaltung von einem bewusst deaktivierten Konto nicht zu unterscheiden.
	var beantragt bool
	if err := pool.QueryRow(ctx, `
		SELECT zugang_beantragt_am IS NOT NULL FROM benutzer WHERE LOWER(email) = $1
	`, email).Scan(&beantragt); err != nil {
		t.Fatalf("zugang_beantragt_am lesen: %v", err)
	}
	if !beantragt {
		t.Error("zugang_beantragt_am ist NULL — die Selbstanmeldung muss den Antrag markieren")
	}
	// Und eine Audit-Spur: Wer hat dieses Konto wann angelegt? Niemand — es hat sich
	// selbst angemeldet, und genau das muss nachlesbar sein.
	var auditZeilen int
	if err := pool.QueryRow(ctx, `
		SELECT count(*) FROM audit_logs a JOIN benutzer b ON b.id = a.admin_id
		WHERE a.aktion = 'SELBSTANMELDUNG' AND LOWER(b.email) = $1
	`, email).Scan(&auditZeilen); err != nil {
		t.Fatalf("Audit lesen: %v", err)
	}
	if auditZeilen != 1 {
		t.Errorf("%d Audit-Zeilen SELBSTANMELDUNG für das Konto, erwartet genau 1", auditZeilen)
	}

	// 3. Zweiter Versuch, immer noch nicht freigeschaltet: weiterhin kein Zugang, und
	//    es entsteht KEIN zweiter Eintrag.
	if code, _ := anmelden(t, pool, email); code != http.StatusForbidden {
		t.Errorf("zweiter Versuch vor der Freischaltung: Status %d, erwartet 403", code)
	}
	var anzahl int
	if err := pool.QueryRow(ctx,
		`SELECT count(*) FROM benutzer WHERE LOWER(email) = $1`, email).Scan(&anzahl); err != nil {
		t.Fatal(err)
	}
	if anzahl != 1 {
		t.Errorf("nach zwei Versuchen stehen %d Einträge in der Tabelle, erwartet genau 1", anzahl)
	}

	// 4. Freischalten — über DENSELBEN Repository-Pfad wie die Benutzerverwaltung,
	//    nicht über ein rohes UPDATE: Der Pfad muss den Antrag auch erledigen.
	var id string
	if err := pool.QueryRow(ctx, `SELECT id FROM benutzer WHERE LOWER(email) = $1`, email).Scan(&id); err != nil {
		t.Fatal(err)
	}
	if err := repository.NewUserRepository(pool).UpdateUser(ctx, repository.UpdateUserParams{
		ID: id, Vorname: vorname, Nachname: nachname, Email: email, Rolle: rolle, Aktiv: true,
	}); err != nil {
		t.Fatalf("freischalten: %v", err)
	}
	if code, meldung := anmelden(t, pool, email); code != http.StatusOK {
		t.Errorf("nach der Freischaltung: Status %d (%s), erwartet 200", code, meldung)
	}
	if err := pool.QueryRow(ctx, `
		SELECT zugang_beantragt_am IS NOT NULL FROM benutzer WHERE LOWER(email) = $1
	`, email).Scan(&beantragt); err != nil {
		t.Fatal(err)
	}
	if beantragt {
		t.Error("nach der Freischaltung steht zugang_beantragt_am noch — ein späteres Deaktivieren sähe wieder wie ein Antrag aus")
	}
}

// Ohne freigegebene Domain darf gar nichts entstehen — auch dann nicht, wenn der
// Mailserver die Zugangsdaten bestätigt. Sonst legte jedes fremde Postfach eine Zeile an.
func TestSelbstanmeldung_FremdeDomainLegtNichtsAn(t *testing.T) {
	t.Setenv("APP_ENV", "test")
	t.Setenv("IMAP_HOST", "mock")
	t.Setenv(selbstanmeldeDomainEnv, "selbsttest.invalid")

	pool := pgPoolFuerSelbstanmeldung(t)
	ctx := context.Background()
	const fremd = "wer.auch.immer@ganz-andere-domain.invalid"
	raeumeKontoAb(t, pool, fremd)

	code, _ := anmelden(t, pool, fremd)
	if code != http.StatusUnauthorized {
		t.Errorf("fremde Domain: Status %d, erwartet 401", code)
	}

	var anzahl int
	if err := pool.QueryRow(ctx,
		`SELECT count(*) FROM benutzer WHERE LOWER(email) = $1`, fremd).Scan(&anzahl); err != nil {
		t.Fatal(err)
	}
	if anzahl != 0 {
		t.Errorf("es wurde ein Eintrag für eine fremde Domain angelegt (%d)", anzahl)
	}
}

// Abgeschaltet heißt abgeschaltet: Ohne SELBSTANMELDUNG_DOMAIN bleibt es beim alten
// Verhalten, auch für die Adresse, die sonst durchkäme.
func TestSelbstanmeldung_AbgeschaltetLegtNichtsAn(t *testing.T) {
	t.Setenv("APP_ENV", "test")
	t.Setenv("IMAP_HOST", "mock")
	t.Setenv(selbstanmeldeDomainEnv, "")

	pool := pgPoolFuerSelbstanmeldung(t)
	ctx := context.Background()
	const email = "niemand@selbsttest.invalid"
	raeumeKontoAb(t, pool, email)

	if code, _ := anmelden(t, pool, email); code != http.StatusUnauthorized {
		t.Errorf("abgeschaltet: Status %d, erwartet 401", code)
	}

	var anzahl int
	if err := pool.QueryRow(ctx,
		`SELECT count(*) FROM benutzer WHERE LOWER(email) = $1`, email).Scan(&anzahl); err != nil {
		t.Fatal(err)
	}
	if anzahl != 0 {
		t.Error("bei abgeschalteter Selbstanmeldung darf kein Eintrag entstehen")
	}
}
