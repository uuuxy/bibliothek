package auth

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
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

const testDBEnvVar = "TEST_DATABASE_URL"

// testDBLockKey serialisiert die Test-DB-Nutzung über die Paketgrenzen hinweg —
// derselbe Schlüssel wie in db/, repository/ und api/. `go test ./...` startet die
// Paket-Binaries PARALLEL, und die anderen drei machen DROP SCHEMA public CASCADE.
// Ohne diesen Lock zieht eines davon mitten im Lauf hier die Tabellen weg; der Test
// fällt dann scheinbar zufällig um. Genau so ist es beim ersten Suitenlauf passiert:
// einzeln grün, in der Suite rot.
const testDBLockKey int64 = 0x42DB0001

func pgPoolFuerSelbstanmeldung(t *testing.T) *pgxpool.Pool {
	t.Helper()
	dsn := os.Getenv(testDBEnvVar)
	if dsn == "" {
		t.Skipf("%s nicht gesetzt — DB-Integrationstest übersprungen", testDBEnvVar)
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatalf("Pool: %v", err)
	}
	t.Cleanup(pool.Close)

	// Über eine eigene, bis zum Testende offene Verbindung — der Lock hängt an der
	// Sitzung, nicht am Pool.
	lockConn, err := pool.Acquire(ctx)
	if err != nil {
		t.Fatalf("Lock-Verbindung: %v", err)
	}
	if _, err := lockConn.Exec(ctx, "SELECT pg_advisory_lock($1)", testDBLockKey); err != nil {
		t.Fatalf("Test-DB-Lock: %v", err)
	}
	t.Cleanup(func() {
		// Der Lock hängt an der Sitzung und fällt beim Release ohnehin weg — das
		// ausdrückliche Entsperren ist nur höflich. Fehler hier sollen den Testlauf nicht
		// umwerfen, aber auch nicht verschwinden.
		if _, err := lockConn.Exec(context.Background(),
			"SELECT pg_advisory_unlock($1)", testDBLockKey); err != nil {
			t.Logf("Test-DB-Lock freigeben: %v", err)
		}
		lockConn.Release()
	})

	var name string
	if err := pool.QueryRow(ctx, `SELECT current_database()`).Scan(&name); err != nil {
		t.Fatalf("Datenbanknamen lesen: %v", err)
	}
	if !strings.Contains(strings.ToLower(name), "test") {
		t.Fatalf("Sicherheitsabbruch: Datenbank %q enthält nicht \"test\"", name)
	}

	// Die eigenen Tabellen anlegen, falls sie fehlen — auth/ spielt kein schema.sql ein,
	// und sich darauf zu verlassen, dass ein anderes Paket das vorher getan hat, wäre
	// genau die Reihenfolgen-Abhängigkeit, die der Lock oben verhindern soll.
	if _, err := pool.Exec(ctx, `
		DO $$ BEGIN
			IF to_regtype('benutzer_rolle') IS NULL THEN
				CREATE TYPE benutzer_rolle AS ENUM ('admin','kollegium','mitarbeiter','helfer');
			END IF;
		END $$;
		CREATE TABLE IF NOT EXISTS benutzer (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			barcode_id VARCHAR(100) UNIQUE,
			vorname VARCHAR(100) NOT NULL,
			nachname VARCHAR(100) NOT NULL,
			email VARCHAR(255) UNIQUE NOT NULL,
			rolle benutzer_rolle NOT NULL,
			aktiv BOOLEAN NOT NULL DEFAULT true,
			erstellt_am TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
			aktualisiert_am TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`); err != nil {
		t.Fatalf("benutzer anlegen: %v", err)
	}

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

	if _, err := pool.Exec(ctx, `DELETE FROM benutzer WHERE email LIKE '%@selbsttest.invalid'`); err != nil {
		t.Fatalf("aufräumen: %v", err)
	}
	return pool
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

	// 4. Freischalten — genau das, was die Benutzerverwaltung tut.
	if _, err := pool.Exec(ctx,
		`UPDATE benutzer SET aktiv = true WHERE LOWER(email) = $1`, email); err != nil {
		t.Fatalf("freischalten: %v", err)
	}
	if code, meldung := anmelden(t, pool, email); code != http.StatusOK {
		t.Errorf("nach der Freischaltung: Status %d (%s), erwartet 200", code, meldung)
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
