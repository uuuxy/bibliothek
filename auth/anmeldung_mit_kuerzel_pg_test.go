package auth

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"bibliothek/internal/adressabgleich"
	"bibliothek/repository"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Die Anmeldung mit dem Kürzel am ganzen Weg: Tür, Adressbuch, Verzeichnisdienst über die
// Leitung (miniVerzeichnis) und die Datenbank aus internal/pgtest.

const kuerzelTestDomain = "kuerzeltest.invalid"

// testAdressbuch ist die Quelle des Adressbuchs und zählt die Abrufe.
type testAdressbuch struct {
	mu        sync.Mutex
	eintraege []adressabgleich.Eintrag
	fehler    error
	abrufe    int
}

func (b *testAdressbuch) hole(context.Context) ([]adressabgleich.Eintrag, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.abrufe++
	return b.eintraege, b.fehler
}

func (b *testAdressbuch) anzahlAbrufe() int {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.abrufe
}

// eintragMit ist ein Eintrag des Adressbuchs mit Kürzel und einer Adresse der Schule.
func eintragMit(vorname, nachname, kuerzel string) adressabgleich.Eintrag {
	return adressabgleich.Eintrag{
		Vorname: vorname, Nachname: nachname, Kuerzel: strings.ToUpper(kuerzel),
		Adressen: []string{strings.ToLower(vorname + "." + nachname + "@" + kuerzelTestDomain)},
	}
}

// verzeichnisFuer startet einen Verzeichnisdienst, in dem jede Kennung einen Eintrag mit dem
// Passwort testKontoPasswort hat.
func verzeichnisFuer(t *testing.T, kennungen ...string) *miniVerzeichnis {
	t.Helper()
	dns, passwoerter := map[string]string{}, map[string]string{}
	for _, k := range kennungen {
		dn := "cn=" + k + ",ou=lehrkraefte,dc=schule,dc=test"
		dns["(sAMAccountName="+k+")"] = dn
		passwoerter[dn] = testKontoPasswort
	}
	v := &miniVerzeichnis{
		treffer: func(_, filter string) []string {
			if dn, ok := dns[filter]; ok {
				return []string{dn}
			}
			return nil
		},
		passwoerter: passwoerter,
	}
	starteMiniVerzeichnis(t, v)
	return v
}

// kuerzelAnmeldung baut die Anmeldung mit dem Adressbuch, ohne Wartezeit bis zur Absage. Der
// Mailserver ist abgeschaltet: Was ohne Kürzel durchfällt, endet als gewöhnliche Absage.
func kuerzelAnmeldung(t *testing.T, buch *testAdressbuch) (*Anmeldung, *pgxpool.Pool) {
	t.Helper()
	t.Setenv(selbstanmeldeDomainVariable, kuerzelTestDomain)
	t.Setenv("IMAP_HOST", "")
	pool := pgPoolFuerSelbstanmeldung(t)
	a, err := NewAnmeldung("test-secret-mit-mindestens-32-zeichen!!", pool, time.Hour)
	if err != nil {
		t.Fatalf("NewAnmeldung: %v", err)
	}
	a.absageFrist = 0
	a.SetzeAdressbuch(buch.hole)
	return a, pool
}

// meldeMitKuerzelAn schickt eine Anmeldung durch die Tür und liefert Status und Meldung.
func meldeMitKuerzelAn(t *testing.T, a *Anmeldung, pool *pgxpool.Pool, kennung, passwort string) (int, string) {
	t.Helper()
	rec := httptest.NewRecorder()
	anfrage := httptest.NewRequest(http.MethodPost, "/login",
		strings.NewReader(`{"email":"`+kennung+`","password":"`+passwort+`"}`))
	anfrage.Header.Set("Content-Type", "application/json")
	Anmelden(pool, a)(rec, anfrage)
	var antwort struct {
		Error string `json:"error"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &antwort); err != nil {
		t.Logf("Antwort war kein JSON (Status %d): %s", rec.Code, rec.Body.String())
	}
	return rec.Code, antwort.Error
}

// raeumeKuerzelKontoAb räumt wie raeumeKontoAb auf und nimmt dazu die Leserzeile des Kontos mit:
// Bliebe sie stehen, stünde der Name beim nächsten Lauf ohne Konto in der Leserdatei, und die
// Anmeldung mit dem Kürzel legte zu Recht keins an.
func raeumeKuerzelKontoAb(t *testing.T, pool *pgxpool.Pool, email string) {
	t.Helper()
	t.Cleanup(func() {
		ctx := context.Background()
		if _, err := pool.Exec(ctx, `
			DELETE FROM audit_logs WHERE admin_id IN (SELECT id FROM benutzer WHERE LOWER(email) = $1)`, email); err != nil {
			t.Errorf("Aufräumen audit_logs: %v", err)
		}
		if _, err := pool.Exec(ctx, `
			WITH weg AS (DELETE FROM benutzer WHERE LOWER(email) = $1 RETURNING leser_id)
			DELETE FROM leser WHERE id IN (SELECT leser_id FROM weg)`, email); err != nil {
			t.Errorf("Aufräumen benutzer und leser: %v", err)
		}
	})
}

// kontoStand ist, was die Tests an einem Konto prüfen.
type kontoStand struct {
	id, vorname, nachname, rolle, verfahren, kennung, art string
	aktiv, beantragt                                      bool
}

func leseKontoStand(t *testing.T, pool *pgxpool.Pool, email string) (kontoStand, int) {
	t.Helper()
	ctx := context.Background()
	var anzahl int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM benutzer WHERE LOWER(email) = $1`, email).Scan(&anzahl); err != nil {
		t.Fatalf("Konten zählen: %v", err)
	}
	if anzahl != 1 {
		return kontoStand{}, anzahl
	}
	var k kontoStand
	if err := pool.QueryRow(ctx, `
		SELECT b.id::text, b.vorname, b.nachname, b.rolle::text, b.anmeldeverfahren, coalesce(b.ldap_kennung, ''),
		       coalesce(l.art, ''), b.aktiv, b.zugang_beantragt_am IS NOT NULL
		FROM benutzer b LEFT JOIN leser l ON l.id = b.leser_id
		WHERE LOWER(b.email) = $1`, email).Scan(
		&k.id, &k.vorname, &k.nachname, &k.rolle, &k.verfahren, &k.kennung, &k.art, &k.aktiv, &k.beantragt); err != nil {
		t.Fatalf("Konto lesen: %v", err)
	}
	return k, anzahl
}

func zaehleProtokoll(t *testing.T, pool *pgxpool.Pool, kontoID, aktion string) int {
	t.Helper()
	var n int
	if err := pool.QueryRow(context.Background(), `
		SELECT count(*) FROM audit_logs
		WHERE aktion = $2 AND admin_id::text = $1 AND details->>'ziel_id' = $1`, kontoID, aktion).Scan(&n); err != nil {
		t.Fatalf("Protokoll lesen: %v", err)
	}
	return n
}

// Eine Lehrkraft ohne Konto meldet sich mit Kürzel und PaedNet-Passwort an und ist sofort drin,
// mit Name und Adresse aus dem Adressbuch. Die zweite Anmeldung findet das Konto über die
// Kennung und fragt das Adressbuch nicht mehr.
func TestAnmeldungMitKuerzel_NewColleagueLogsInAtOnce(t *testing.T) {
	buch := &testAdressbuch{eintraege: []adressabgleich.Eintrag{eintragMit("Nina", "Neumann", "kneu")}}
	a, pool := kuerzelAnmeldung(t, buch)
	verzeichnisFuer(t, "kneu")
	const email = "nina.neumann@" + kuerzelTestDomain
	raeumeKuerzelKontoAb(t, pool, email)

	if code, meldung := meldeMitKuerzelAn(t, a, pool, "kneu", testKontoPasswort); code != http.StatusOK {
		t.Fatalf("erste Anmeldung: Status %d (%q), erwartet 200", code, meldung)
	}
	k, anzahl := leseKontoStand(t, pool, email)
	if anzahl != 1 {
		t.Fatalf("%d Konten mit der Adresse aus dem Adressbuch, erwartet 1", anzahl)
	}
	want := kontoStand{id: k.id, vorname: "Nina", nachname: "Neumann", rolle: "kollegium",
		verfahren: repository.AnmeldeverfahrenLDAP, kennung: "kneu", art: "lehrkraft", aktiv: true}
	if k != want {
		t.Errorf("neues Konto = %+v, erwartet %+v", k, want)
	}
	if n := zaehleProtokoll(t, pool, k.id, "USER_CREATE"); n != 1 {
		t.Errorf("%d Einträge USER_CREATE über das Konto, erwartet 1", n)
	}

	if code, meldung := meldeMitKuerzelAn(t, a, pool, "kneu", testKontoPasswort); code != http.StatusOK {
		t.Fatalf("zweite Anmeldung: Status %d (%q), erwartet 200", code, meldung)
	}
	if n := buch.anzahlAbrufe(); n != 1 {
		t.Errorf("Adressbuch %d-mal abgerufen, erwartet einmal: Die zweite Anmeldung findet das Konto über die Kennung", n)
	}
	if _, anzahl := leseKontoStand(t, pool, email); anzahl != 1 {
		t.Errorf("nach zwei Anmeldungen %d Konten, erwartet 1", anzahl)
	}
}

// Ein Konto des Kollegiums mit der Adresse aus dem Adressbuch wird umgestellt, kein zweites
// angelegt: ein aktives, und ein offener Antrag, der damit erledigt ist. Ein deaktiviertes bleibt,
// wie es ist, und die Anmeldung damit wie bisher.
func TestAnmeldungMitKuerzel_MatchingAddressIsSwitched(t *testing.T) {
	buch := &testAdressbuch{eintraege: []adressabgleich.Eintrag{
		eintragMit("Anna", "Aktiv", "kakt"), eintragMit("Paula", "Wartend", "kwart"), eintragMit("Dora", "Deaktiviert", "kdea"),
	}}
	a, pool := kuerzelAnmeldung(t, buch)
	verzeichnisFuer(t, "kakt", "kwart", "kdea")
	ctx := context.Background()
	konten := repository.NewKontoAbfragen(pool)

	const aktiv = "anna.aktiv@" + kuerzelTestDomain
	raeumeKuerzelKontoAb(t, pool, aktiv)
	if _, err := konten.LegeAn(ctx, repository.KontoNeu{Vorname: "Anna", Nachname: "Aktiv", Email: aktiv, Rolle: "kollegium"}); err != nil {
		t.Fatalf("Konto anlegen: %v", err)
	}
	const wartend = "paula.wartend@" + kuerzelTestDomain
	raeumeKuerzelKontoAb(t, pool, wartend)
	if _, err := repository.LegeZugangsanfrageAn(ctx, pool, "Paula", "Wartend", wartend); err != nil {
		t.Fatalf("Antrag anlegen: %v", err)
	}
	const deaktiviert = "dora.deaktiviert@" + kuerzelTestDomain
	raeumeKuerzelKontoAb(t, pool, deaktiviert)
	doraID, err := konten.LegeAn(ctx, repository.KontoNeu{Vorname: "Dora", Nachname: "Deaktiviert", Email: deaktiviert, Rolle: "kollegium"})
	if err != nil {
		t.Fatalf("Konto anlegen: %v", err)
	}
	if _, err := pool.Exec(ctx, `UPDATE benutzer SET aktiv = false WHERE id = $1`, doraID); err != nil {
		t.Fatalf("Konto deaktivieren: %v", err)
	}

	for _, f := range []struct{ kuerzel, email string }{{"kakt", aktiv}, {"kwart", wartend}} {
		vorher, _ := leseKontoStand(t, pool, f.email)
		if code, meldung := meldeMitKuerzelAn(t, a, pool, f.kuerzel, testKontoPasswort); code != http.StatusOK {
			t.Errorf("%s: Status %d (%q), erwartet 200", f.kuerzel, code, meldung)
			continue
		}
		k, anzahl := leseKontoStand(t, pool, f.email)
		if anzahl != 1 || k.id != vorher.id {
			t.Errorf("%s: %d Konten, Kennung %s statt %s, erwartet das vorhandene Konto", f.kuerzel, anzahl, k.id, vorher.id)
		}
		if k.verfahren != repository.AnmeldeverfahrenLDAP || k.kennung != f.kuerzel || !k.aktiv || k.beantragt {
			t.Errorf("%s: Konto = %+v, erwartet LDAP mit Kennung %s, aktiv, ohne Antrag", f.kuerzel, k, f.kuerzel)
		}
		if n := zaehleProtokoll(t, pool, k.id, "USER_UPDATE"); n != 1 {
			t.Errorf("%s: %d Einträge USER_UPDATE über das Konto, erwartet 1", f.kuerzel, n)
		}
	}

	if code, _ := meldeMitKuerzelAn(t, a, pool, "kdea", testKontoPasswort); code != http.StatusUnauthorized {
		t.Errorf("deaktiviertes Konto: Status %d, erwartet 401 wie ohne Anmeldung mit dem Kürzel", code)
	}
	if k, _ := leseKontoStand(t, pool, deaktiviert); k.verfahren != repository.AnmeldeverfahrenIMAP || k.aktiv {
		t.Errorf("deaktiviertes Konto = %+v, erwartet unverändert IMAP und inaktiv", k)
	}
}

// Ein Name, der nicht als Kürzel im Adressbuch steht, erreicht den Verzeichnisdienst nicht: Wer
// an der Bibliothek Namen durchprobiert, sperrt dort niemanden. Ein falsches Passwort legt nichts
// an.
func TestAnmeldungMitKuerzel_OnlyListedNamesReachDirectory(t *testing.T) {
	buch := &testAdressbuch{eintraege: []adressabgleich.Eintrag{eintragMit("Fritz", "Falsch", "kfal")}}
	a, pool := kuerzelAnmeldung(t, buch)
	v := verzeichnisFuer(t, "kfal", "schueler1")
	raeumeKuerzelKontoAb(t, pool, "fritz.falsch@"+kuerzelTestDomain)

	if code, _ := meldeMitKuerzelAn(t, a, pool, "schueler1", testKontoPasswort); code != http.StatusUnauthorized {
		t.Errorf("Name außerhalb des Adressbuchs: Status %d, erwartet 401", code)
	}
	if binds, _, _ := v.gesehen(); len(binds) != 0 {
		t.Errorf("Verzeichnisdienst bekam Binds %q für einen Namen außerhalb des Adressbuchs, erwartet keine", binds)
	}

	if code, _ := meldeMitKuerzelAn(t, a, pool, "kfal", "falsch"); code != http.StatusUnauthorized {
		t.Errorf("falsches Passwort: Status %d, erwartet 401", code)
	}
	if _, anzahl := leseKontoStand(t, pool, "fritz.falsch@"+kuerzelTestDomain); anzahl != 0 {
		t.Errorf("nach falschem Passwort %d Konten, erwartet keins", anzahl)
	}
}

// Ist das Webmail nicht erreichbar oder der Zugang nicht eingerichtet, greift die Anmeldung mit
// dem Kürzel nicht: Sie bleibt wie bisher, und der Verzeichnisdienst wird nicht gefragt.
func TestAnmeldungMitKuerzel_WebmailFailure(t *testing.T) {
	faelle := []struct {
		name   string
		fehler error
		status int
	}{
		{"Webmail gestört", errors.New("webmail antwortet nicht"), http.StatusUnauthorized},
		{"nicht eingerichtet", adressabgleich.ErrNichtEingerichtet, http.StatusUnauthorized},
	}
	for _, f := range faelle {
		t.Run(f.name, func(t *testing.T) {
			buch := &testAdressbuch{fehler: f.fehler}
			a, pool := kuerzelAnmeldung(t, buch)
			v := verzeichnisFuer(t, "kweg")
			if code, _ := meldeMitKuerzelAn(t, a, pool, "kweg", testKontoPasswort); code != f.status {
				t.Errorf("Status %d, erwartet %d", code, f.status)
			}
			if binds, _, _ := v.gesehen(); len(binds) != 0 {
				t.Errorf("Verzeichnisdienst bekam Binds %q ohne Adressbuch, erwartet keine", binds)
			}
		})
	}
}

// Ohne Verzeichnisdienst wird das Adressbuch gar nicht erst geholt.
func TestAnmeldungMitKuerzel_WithoutLDAPNoFetch(t *testing.T) {
	buch := &testAdressbuch{eintraege: []adressabgleich.Eintrag{eintragMit("Olga", "Ohne", "kohn")}}
	a, pool := kuerzelAnmeldung(t, buch)
	t.Setenv("LDAP_URL", "")
	if code, _ := meldeMitKuerzelAn(t, a, pool, "kohn", testKontoPasswort); code != http.StatusUnauthorized {
		t.Errorf("Status %d, erwartet 401", code)
	}
	if n := buch.anzahlAbrufe(); n != 0 {
		t.Errorf("Adressbuch %d-mal abgerufen, erwartet keinmal ohne LDAP", n)
	}
}

// Steht der Name schon in der Leserdatei, ohne Konto oder mit Konto unter einer anderen Adresse,
// entsteht kein zweiter Eintrag derselben Person; die Anmeldung bleibt wie bisher, bis die
// Bibliothek zuordnet.
func TestAnmeldungMitKuerzel_ListedNameBlocksDuplicate(t *testing.T) {
	buch := &testAdressbuch{eintraege: []adressabgleich.Eintrag{
		eintragMit("Lea", "Leserzeile", "klez"), eintragMit("Vera", "Vertippt", "kver"),
	}}
	a, pool := kuerzelAnmeldung(t, buch)
	v := verzeichnisFuer(t, "klez", "kver")
	ctx := context.Background()
	const lea = "lea.leserzeile@" + kuerzelTestDomain
	raeumeKuerzelKontoAb(t, pool, lea)
	var leserID string
	if err := pool.QueryRow(ctx,
		`INSERT INTO leser (vorname, nachname, art) VALUES ('Lea', 'Leserzeile', 'lehrkraft') RETURNING id::text`).Scan(&leserID); err != nil {
		t.Fatalf("Leserzeile anlegen: %v", err)
	}
	t.Cleanup(func() {
		if _, err := pool.Exec(context.Background(), `DELETE FROM leser WHERE id = $1`, leserID); err != nil {
			t.Errorf("Aufräumen leser: %v", err)
		}
	})
	// Vera hat ein Konto, an der Theke unter einer vertippten Adresse angelegt.
	const vertippt = "v.vertipt@" + kuerzelTestDomain
	raeumeKuerzelKontoAb(t, pool, vertippt)
	raeumeKuerzelKontoAb(t, pool, "vera.vertippt@"+kuerzelTestDomain)
	if _, err := repository.NewKontoAbfragen(pool).LegeAn(ctx, repository.KontoNeu{
		Vorname: "Vera", Nachname: "Vertippt", Email: vertippt, Rolle: "kollegium"}); err != nil {
		t.Fatalf("Konto anlegen: %v", err)
	}

	for _, f := range []struct{ kuerzel, email string }{{"klez", lea}, {"kver", "vera.vertippt@" + kuerzelTestDomain}} {
		if code, _ := meldeMitKuerzelAn(t, a, pool, f.kuerzel, testKontoPasswort); code != http.StatusUnauthorized {
			t.Errorf("%s: Status %d, erwartet 401 wie ohne Anmeldung mit dem Kürzel", f.kuerzel, code)
		}
		if _, anzahl := leseKontoStand(t, pool, f.email); anzahl != 0 {
			t.Errorf("%s: %d Konten angelegt, erwartet keins neben dem vorhandenen Eintrag", f.kuerzel, anzahl)
		}
	}
	if k, _ := leseKontoStand(t, pool, vertippt); k.verfahren != repository.AnmeldeverfahrenIMAP || !k.aktiv {
		t.Errorf("vorhandenes Konto = %+v, erwartet unverändert", k)
	}
	if binds, _, _ := v.gesehen(); len(binds) != 0 {
		t.Errorf("Verzeichnisdienst bekam Binds %q, erwartet keine", binds)
	}
}

// Die Anmeldung mit dem Kürzel gilt nur für Mein Portal. Ein Konto mit einer anderen Rolle, ein
// offener Antrag mit einer anderen Rolle und ein Konto des Kollegiums mit einer anderen
// LDAP-Kennung bleiben, wie sie sind; die Anmeldung geht wie bisher, ohne den Verzeichnisdienst.
func TestAnmeldungMitKuerzel_PortalOnly(t *testing.T) {
	buch := &testAdressbuch{eintraege: []adressabgleich.Eintrag{
		eintragMit("Marta", "Mitarbeit", "kmit"), eintragMit("Hanna", "Helferantrag", "khel"),
		eintragMit("Konrad", "Kennung", "kken"),
	}}
	a, pool := kuerzelAnmeldung(t, buch)
	v := verzeichnisFuer(t, "kmit", "khel", "kken")
	ctx := context.Background()
	konten := repository.NewKontoAbfragen(pool)

	const mitarbeit = "marta.mitarbeit@" + kuerzelTestDomain
	raeumeKuerzelKontoAb(t, pool, mitarbeit)
	if _, err := konten.LegeAn(ctx, repository.KontoNeu{
		Vorname: "Marta", Nachname: "Mitarbeit", Email: mitarbeit, Rolle: "mitarbeiter"}); err != nil {
		t.Fatalf("Konto anlegen: %v", err)
	}
	const helferantrag = "hanna.helferantrag@" + kuerzelTestDomain
	raeumeKuerzelKontoAb(t, pool, helferantrag)
	if _, err := repository.LegeZugangsanfrageAn(ctx, pool, "Hanna", "Helferantrag", helferantrag); err != nil {
		t.Fatalf("Antrag anlegen: %v", err)
	}
	if _, err := pool.Exec(ctx, `UPDATE benutzer SET rolle = 'helfer' WHERE LOWER(email) = $1`, helferantrag); err != nil {
		t.Fatalf("Rolle setzen: %v", err)
	}
	const kennung = "konrad.kennung@" + kuerzelTestDomain
	raeumeKuerzelKontoAb(t, pool, kennung)
	if _, err := konten.LegeAn(ctx, repository.KontoNeu{
		Vorname: "Konrad", Nachname: "Kennung", Email: kennung, Rolle: "kollegium",
		Anmeldeverfahren: repository.AnmeldeverfahrenLDAP, LDAPKennung: "kanders"}); err != nil {
		t.Fatalf("Konto anlegen: %v", err)
	}

	for _, f := range []struct {
		kuerzel, email, verfahren, kennung string
		aktiv                              bool
	}{
		{"kmit", mitarbeit, repository.AnmeldeverfahrenIMAP, "", true},
		{"khel", helferantrag, repository.AnmeldeverfahrenIMAP, "", false},
		{"kken", kennung, repository.AnmeldeverfahrenLDAP, "kanders", true},
	} {
		if code, _ := meldeMitKuerzelAn(t, a, pool, f.kuerzel, testKontoPasswort); code != http.StatusUnauthorized {
			t.Errorf("%s: Status %d, erwartet 401 wie ohne Anmeldung mit dem Kürzel", f.kuerzel, code)
		}
		if k, _ := leseKontoStand(t, pool, f.email); k.verfahren != f.verfahren || k.kennung != f.kennung || k.aktiv != f.aktiv {
			t.Errorf("%s: Konto = %+v, erwartet unverändert %s mit Kennung %q, aktiv %v", f.kuerzel, k, f.verfahren, f.kennung, f.aktiv)
		}
	}
	if binds, _, _ := v.gesehen(); len(binds) != 0 {
		t.Errorf("Verzeichnisdienst bekam Binds %q, erwartet keine", binds)
	}
}
