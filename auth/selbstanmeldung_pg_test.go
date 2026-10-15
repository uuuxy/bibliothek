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

// Die Selbstanmeldung am ganzen Weg, gegen Postgres und durch die echte Tür der Anmeldung,
// nicht gegen die Hilfsfunktionen daneben.
//
// Der Kern ist eine Zusage der Sicherheit: Ein selbst angelegtes Konto darf sich nicht anmelden,
// bevor es jemand freigeschaltet hat. Das lässt sich nur am ganzen Weg prüfen; eine einzeln
// getestete Funktion, die aktiv=false schreibt, sagt nichts darüber, was die Tür daraus macht.

// pgPoolFuerSelbstanmeldung liefert die Datenbank aus internal/pgtest, eingespielt aus
// schema.sql als der einen Quelle des Schemas. Das Schema wird je Testprogramm einmal gebaut und
// mit allen Tests geteilt: Jeder Test räumt sein Konto über raeumeKontoAb selbst ab und prüft
// nur an seiner eigenen E-Mail-Adresse. Eine eigene Sperre der Datenbank nimmt der Test nicht:
// pgtest hält seine bis zum Ende des Prozesses, und ein zweiter Halter wartete ewig.
func pgPoolFuerSelbstanmeldung(t *testing.T) *pgxpool.Pool {
	t.Helper()
	pool := pgtest.Pool(t)
	ctx := context.Background()

	// Die Zeilen von role_permissions schreibt im Betrieb db.SchreibeRechteVorgabe beim Start.
	// Ohne sie scheitert die Anmeldung nach der Freischaltung am Laden der Rechte, und zwar als
	// 500: Der Fehlertext wird unterwegs durch „interner Datenbankfehler" ersetzt, die Ursache
	// stünde also nirgends. Der Test legt das eine Recht deshalb selbst an.
	if _, err := pool.Exec(ctx, `
		INSERT INTO role_permissions (role, permission, allowed) VALUES ('KOLLEGIUM', 'view_books', true)
		ON CONFLICT DO NOTHING`); err != nil {
		t.Fatalf("Rechte seeden: %v", err)
	}
	return pool
}

// raeumeKontoAb entfernt am Ende des Tests das Konto zu dieser Adresse samt seinen Zeilen im
// Protokoll; pgtest teilt das Schema mit allen Tests des Testprogramms. Die Zeilen im Protokoll
// zuerst: audit_logs.admin_id steht auf ON DELETE SET NULL, ein bloßes DELETE auf benutzer ließe
// sie verwaist zurück.
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

// anmelden schickt eine echte Anmeldung durch die Tür.
func anmelden(t *testing.T, pool *pgxpool.Pool, email string) (int, string) {
	t.Helper()
	auth, err := NewAnmeldung("test-secret-mit-mindestens-32-zeichen!!", pool, time.Hour)
	if err != nil {
		t.Fatalf("NewAnmeldung: %v", err)
	}
	rec := httptest.NewRecorder()
	rumpf := strings.NewReader(`{"email":"` + email + `","password":"beliebig"}`)
	anfrage := httptest.NewRequest(http.MethodPost, "/login", rumpf)
	anfrage.Header.Set("Content-Type", "application/json")

	Anmelden(pool, auth, false)(rec, anfrage)

	// Eine gelungene Antwort trägt kein Feld error; ein Fehler beim Lesen ist deshalb kein Fehler
	// des Tests, der Statuscode bleibt aussagekräftig. Stillschweigend verworfen wird er trotzdem
	// nicht.
	var antwort map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &antwort); err != nil {
		t.Logf("Antwort war kein JSON (Status %d): %s", rec.Code, rec.Body.String())
	}
	// errcheck läuft hier mit check-blank und check-type-assertions: Auch das verworfene „ok" der
	// Typzusicherung zählt als übergangener Fehler. Eine gelungene Antwort trägt kein Feld error;
	// dann bleibt der Text leer, und das ist kein Fehler.
	text, vorhanden := antwort["error"].(string)
	if !vorhanden {
		text = ""
	}
	return rec.Code, text
}

func TestSelbstanmeldung_LegtAnAberLaesstNichtRein(t *testing.T) {
	// Attrappe des Mailservers: Jedes Passwort gilt. So prüft der Test genau das, was nach der
	// gelungenen Prüfung des Passworts geschieht, und nichts anderes.
	t.Setenv("APP_ENV", "test")
	t.Setenv("IMAP_HOST", "mock")
	t.Setenv(selbstanmeldeDomainVariable, "selbsttest.invalid")

	pool := pgPoolFuerSelbstanmeldung(t)
	ctx := context.Background()
	const email = "erika.musterfrau@selbsttest.invalid"
	raeumeKontoAb(t, pool, email)

	// 1. Erster Versuch: Es gibt noch kein Konto.
	code, meldung := anmelden(t, pool, email)
	if code != http.StatusForbidden {
		t.Fatalf("erster Versuch: Status %d, erwartet 403 — ein neues Konto darf NICHT hineinkommen", code)
	}
	if erwartet := "Zugang beantragt. Die Bibliothek muss ihn noch freischalten."; meldung != erwartet {
		t.Errorf("erster Versuch: Meldung %q, erwartet %q", meldung, erwartet)
	}

	// 2. Der Eintrag besteht: inaktiv, Rolle kollegium, Name aus der Adresse geraten.
	var aktiv bool
	var rolle, vorname, nachname, art string
	// Die Art steht an der Leserzeile des Kontos. Dass es überhaupt eine gibt, ist Teil der
	// Zusage: Ohne sie fände die Theke die neue Lehrkraft nicht.
	if err := pool.QueryRow(ctx, `
		SELECT b.aktiv, b.rolle::text, b.vorname, b.nachname, coalesce(l.art, '')
		FROM benutzer b LEFT JOIN leser l ON l.id = b.leser_id
		WHERE LOWER(b.email) = $1
	`, email).Scan(&aktiv, &rolle, &vorname, &nachname, &art); err != nil {
		t.Fatalf("angelegtes Konto lesen: %v", err)
	}
	if aktiv {
		t.Error("das angelegte Konto ist AKTIV — damit wäre die Freischaltung wirkungslos")
	}
	if rolle != "kollegium" {
		t.Errorf("Rolle = %q, erwartet kollegium", rolle)
	}
	// Wer sich über das Postfach der Schule selbst anmeldet, ist Lehrkraft; eine LiV trägt die
	// Leserdatei von Hand um. 'schueler' wäre hier gefährlich: Die Zeile käme in jede Klassenliste
	// und würde vom nächsten Abgleich mit der LUSD als Abgänger markiert.
	if art != "lehrkraft" {
		t.Errorf("Art der Leserzeile = %q, erwartet lehrkraft", art)
	}
	if vorname != "Erika" || nachname != "Musterfrau" {
		t.Errorf("Name aus der Adresse = %q %q, erwartet Erika Musterfrau", vorname, nachname)
	}
	// Der Antrag ist als solcher markiert, sonst wäre er in der Benutzerverwaltung von einem
	// absichtlich deaktivierten Konto nicht zu unterscheiden.
	var beantragt bool
	if err := pool.QueryRow(ctx, `
		SELECT zugang_beantragt_am IS NOT NULL FROM benutzer WHERE LOWER(email) = $1
	`, email).Scan(&beantragt); err != nil {
		t.Fatalf("zugang_beantragt_am lesen: %v", err)
	}
	if !beantragt {
		t.Error("zugang_beantragt_am ist NULL — die Selbstanmeldung muss den Antrag markieren")
	}
	// Und ein Eintrag im Protokoll: Wer hat das Konto wann angelegt? Niemand, es hat sich selbst
	// angemeldet, und genau das muss nachzulesen sein.
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
	// Der Eintrag nennt das Konto als Ziel, dazu Adresse, Rolle und dass es noch nicht
	// freigeschaltet ist: Daran findet ihn die Auskunft, auch nachdem das Konto gelöscht ist.
	// Eine IP-Adresse trägt er nicht.
	var zielIstKonto, ohneIP bool
	var mailImEintrag, rolleImEintrag, aktivImEintrag string
	if err := pool.QueryRow(ctx, `
		SELECT coalesce(a.details->>'ziel_id' = b.id::text, false), coalesce(a.details->>'email', ''),
		       coalesce(a.details->>'rolle', ''), coalesce(a.details->>'aktiv', ''), a.ip_adresse IS NULL
		FROM audit_logs a JOIN benutzer b ON b.id = a.admin_id
		WHERE a.aktion = 'SELBSTANMELDUNG' AND LOWER(b.email) = $1
	`, email).Scan(&zielIstKonto, &mailImEintrag, &rolleImEintrag, &aktivImEintrag, &ohneIP); err != nil {
		t.Fatalf("Inhalt des Audit-Eintrags lesen: %v", err)
	}
	if !zielIstKonto || mailImEintrag != email || rolleImEintrag != "kollegium" || aktivImEintrag != "false" || !ohneIP {
		t.Errorf("Audit-Eintrag: Ziel ist das Konto %v, Adresse %q, Rolle %q, aktiv %q, ohne IP %v",
			zielIstKonto, mailImEintrag, rolleImEintrag, aktivImEintrag, ohneIP)
	}

	// 3. Zweiter Versuch, noch nicht freigeschaltet: weiter kein Zugang, dieselbe Meldung wie beim
	//    ersten Mal („beantragt", nicht „deaktiviert"), und es entsteht kein zweiter Eintrag.
	if code, meldung := anmelden(t, pool, email); code != http.StatusForbidden || !strings.Contains(meldung, "beantragt") {
		t.Errorf("zweiter Versuch vor der Freischaltung: Status %d (%q), erwartet 403 mit „beantragt“", code, meldung)
	}
	var anzahl int
	if err := pool.QueryRow(ctx,
		`SELECT count(*) FROM benutzer WHERE LOWER(email) = $1`, email).Scan(&anzahl); err != nil {
		t.Fatal(err)
	}
	if anzahl != 1 {
		t.Errorf("nach zwei Versuchen stehen %d Einträge in der Tabelle, erwartet genau 1", anzahl)
	}

	// 4. Freischalten über denselben Weg in repository wie die Benutzerverwaltung, nicht über ein
	//    rohes UPDATE: Der Weg muss auch den Antrag erledigen.
	var id string
	if err := pool.QueryRow(ctx, `SELECT id FROM benutzer WHERE LOWER(email) = $1`, email).Scan(&id); err != nil {
		t.Fatal(err)
	}
	wahr, falsch := true, false
	if _, err := repository.NewKontoAbfragen(pool).Aendere(ctx, repository.KontoAenderung{
		ID: id, Aktiv: &wahr,
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

	// 5. Absichtlich deaktivieren: Das ist kein Antrag mehr, die Meldung sagt „deaktiviert",
	//    nicht „beantragt". Sonst wartete die Person auf eine Freischaltung, die nicht kommt.
	if _, err := repository.NewKontoAbfragen(pool).Aendere(ctx, repository.KontoAenderung{
		ID: id, Aktiv: &falsch,
	}); err != nil {
		t.Fatalf("deaktivieren: %v", err)
	}
	if code, meldung := anmelden(t, pool, email); code != http.StatusForbidden ||
		strings.Contains(meldung, "beantragt") || !strings.Contains(meldung, "deaktiviert") {
		t.Errorf("bewusst deaktiviert: Status %d (%q), erwartet 403 mit „deaktiviert“", code, meldung)
	}
}

// Ohne freigegebene Domain entsteht gar nichts, auch wenn der Mailserver die Zugangsdaten
// bestätigt. Sonst legte jedes fremde Postfach eine Zeile an.
func TestSelbstanmeldung_FremdeDomainLegtNichtsAn(t *testing.T) {
	t.Setenv("APP_ENV", "test")
	t.Setenv("IMAP_HOST", "mock")
	t.Setenv(selbstanmeldeDomainVariable, "selbsttest.invalid")

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

// Abgeschaltet heißt abgeschaltet: Ohne SELBSTANMELDUNG_DOMAIN entsteht kein Konto, auch nicht
// für die Adresse, die sonst durchkäme.
func TestSelbstanmeldung_AbgeschaltetLegtNichtsAn(t *testing.T) {
	t.Setenv("APP_ENV", "test")
	t.Setenv("IMAP_HOST", "mock")
	t.Setenv(selbstanmeldeDomainVariable, "")

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

// Ein zweiter Anmeldeversuch hinterlässt keine zweite Leserzeile.
//
// Warum das eine Zusage ist: Der Wächter trg_benutzer_hat_leserzeile läuft BEFORE INSERT, also
// bevor Postgres einen Konflikt bemerkt, und legt dabei eine frische Leserzeile an. Ein
// `INSERT ... ON CONFLICT DO NOTHING` auf `benutzer` verwürfe die Zeile des Kontos, die
// Leserzeile des Wächters bliebe stehen, ohne Konto und ohne Ausweisnummer.
//
// Auf dem Weg der Anmeldung geschieht das nicht, und zwar nicht durch Zufall:
// legeZugangsanfrageAn wird nur im Zweig `pgx.ErrNoRows` aufgerufen, also nur, wenn es gar kein
// Konto gibt. Beim zweiten Versuch findet die Abfrage das Konto, und der INSERT wird nie
// versucht.
//
// Der Test hält deshalb das Ergebnis fest, nicht eine einzelne Zeile: Der Schutz liegt im
// Zuschnitt des Aufrufers, und ein Umbau, der die Anlage aus diesem Zweig herauszieht, hinterließe
// unbemerkt verwaiste Leserzeilen. Mit einer kleinen Änderung lässt sich der Test nicht rot
// bekommen; er sichert eine Struktur, keinen Schalter.
func TestSelbstanmeldung_ZweiterVersuchLaesstKeineLeserzeileZurueck(t *testing.T) {
	// Attrappe des Mailservers wie im Test daneben: Ohne sie scheitert die Anmeldung schon am
	// Passwort und käme nie bis zur Anlage.
	t.Setenv("APP_ENV", "test")
	t.Setenv("IMAP_HOST", "mock")
	t.Setenv(selbstanmeldeDomainVariable, "schule-test.invalid")
	pool := pgPoolFuerSelbstanmeldung(t)
	const mail = "zweimal.klick@schule-test.invalid"
	raeumeKontoAb(t, pool, mail)
	ctx := context.Background()

	zaehleLeserzeilen := func() int {
		t.Helper()
		var n int
		if err := pool.QueryRow(ctx,
			`SELECT count(*) FROM leser WHERE nachname = 'Klick' AND deleted_at IS NULL`).Scan(&n); err != nil {
			t.Fatalf("Leserzeilen zählen: %v", err)
		}
		return n
	}
	t.Cleanup(func() {
		if _, err := pool.Exec(ctx, `DELETE FROM leser WHERE nachname = 'Klick'`); err != nil {
			t.Errorf("Aufräumen leser: %v", err)
		}
	})

	anmelden(t, pool, mail)
	nachErstem := zaehleLeserzeilen()
	if nachErstem != 1 {
		t.Fatalf("nach dem ersten Versuch stehen %d Leserzeilen da, erwartet 1", nachErstem)
	}

	anmelden(t, pool, mail)
	anmelden(t, pool, mail)
	if n := zaehleLeserzeilen(); n != 1 {
		t.Fatalf("nach drei Versuchen stehen %d Leserzeilen da, erwartet 1 — die Anlage "+
			"wird offenbar auch dann versucht, wenn das Konto schon existiert", n)
	}

	var konten int
	if err := pool.QueryRow(ctx,
		`SELECT count(*) FROM benutzer WHERE LOWER(email) = $1`, mail).Scan(&konten); err != nil {
		t.Fatal(err)
	}
	if konten != 1 {
		t.Fatalf("%d Konten unter derselben Adresse", konten)
	}
}

// Zwischen dem ersten Lesen der Anmeldung und dem Anlegen kann das Konto entstanden sein:
// von einem zweiten Gerät derselben Person oder in der Benutzerverwaltung. Die Anlage trifft
// dann die vorhandene Zeile, und die Anmeldung geht mit ihr weiter. Sie muss dasselbe Konto
// liefern wie das erste Lesen, mit der Adresse: Aus ihr entsteht die Antwort an den Browser.
func TestSelbstanmeldung_VorhandenesKontoKommtVollstaendigZurueck(t *testing.T) {
	t.Setenv(selbstanmeldeDomainVariable, "schule-test.invalid")
	pool := pgPoolFuerSelbstanmeldung(t)
	const mail = "wettlauf.konto@schule-test.invalid"
	raeumeKontoAb(t, pool, mail)
	ctx := context.Background()

	var id string
	if err := pool.QueryRow(ctx, `
		INSERT INTO benutzer (vorname, nachname, email, rolle, aktiv)
		VALUES ('Wanda', 'Wettlauf', $1, 'kollegium', true)
		RETURNING id::text`, mail).Scan(&id); err != nil {
		t.Fatalf("Konto anlegen: %v", err)
	}
	t.Cleanup(func() {
		if _, err := pool.Exec(context.Background(), `DELETE FROM leser WHERE nachname = 'Wettlauf'`); err != nil {
			t.Errorf("Aufräumen leser: %v", err)
		}
	})

	konto, err := legeZugangsanfrageAn(ctx, pool, mail)
	if err != nil {
		t.Fatalf("legeZugangsanfrageAn: %v", err)
	}
	if konto.neuAngelegt {
		t.Error("ein vorhandenes Konto gilt als neu angelegt")
	}
	if konto.id != id || !konto.aktiv || konto.vorname != "Wanda" || konto.nachname != "Wettlauf" {
		t.Errorf("Konto = %+v, erwartet das vorhandene, aktive Konto %s", konto, id)
	}
	if konto.email != mail {
		t.Errorf("Adresse des Kontos = %q, erwartet %q", konto.email, mail)
	}
}
