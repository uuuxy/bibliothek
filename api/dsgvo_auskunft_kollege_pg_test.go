package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"bibliothek/auth"
	"bibliothek/db"
	"bibliothek/internal/pdftest"
	"bibliothek/repository"
)

// Die Auskunft für einen Kollegen (entschieden am 24.09.2026, OFFEN.md 5.19).
//
// Bis dahin las sie die Stammdaten aus der Sicht `schueler` und antwortete bei einem
// Kollegen mit 404. Der Test stellt den Fall am echten Postgres nach: ein Konto mit
// Leserzeile, eine Klassenleitung, ein Wunsch, eine Reservierung und Einträge des
// Verwaltungsprotokolls über das Konto. Daneben stehen zwei Gegenproben, die NICHT in die
// Auskunft gehören: die Daten eines zweiten Kollegen und die Anlage eines früheren Kontos
// mit derselben Adresse (vor der Anlage dieses Kontos).
func TestDsgvoAuskunft_Kollege(t *testing.T) {
	pool := pgTestPool(t)
	resetBestandsdaten(t, pool)
	ctx := context.Background()
	exec := func(sql string, args ...any) {
		t.Helper()
		if _, err := pool.Exec(ctx, sql, args...); err != nil {
			t.Fatalf("%s: %v", strings.TrimSpace(sql)[:40], err)
		}
	}
	t.Cleanup(func() {
		for _, sql := range []string{
			`DELETE FROM klassen_lehrer_mapping WHERE klasse = 'KA-07B'`,
			`DELETE FROM audit_logs WHERE details->>'email' ILIKE '%@auskunft-kollege.invalid'
			    OR details->>'ziel_id' IN (SELECT id::text FROM benutzer WHERE email LIKE '%@auskunft-kollege.invalid')
			    OR admin_id IN (SELECT id FROM benutzer WHERE email LIKE '%@auskunft-kollege.invalid')`,
			`DELETE FROM lehrer_anliegen WHERE angefordert_von IN (SELECT id FROM benutzer WHERE email LIKE '%@auskunft-kollege.invalid')`,
			`DELETE FROM klassensatz_reservierungen WHERE angefordert_von IN (SELECT id FROM benutzer WHERE email LIKE '%@auskunft-kollege.invalid')`,
			`DELETE FROM benutzer WHERE email LIKE '%@auskunft-kollege.invalid'`,
		} {
			if _, err := pool.Exec(context.Background(), sql); err != nil {
				t.Errorf("aufräumen: %v", err)
			}
		}
	})

	// Das Konto: Der Trigger trg_benutzer_hat_leserzeile hängt die Leserzeile an.
	legeKontoAn := func(vorname, email string) (kontoID, leserID string) {
		t.Helper()
		if err := pool.QueryRow(ctx, `
			INSERT INTO benutzer (vorname, nachname, email, rolle, aktiv)
			VALUES ($1, 'Kollegin', $2, 'mitarbeiter', true)
			RETURNING id, leser_id`, vorname, email).Scan(&kontoID, &leserID); err != nil {
			t.Fatalf("Konto %s anlegen: %v", email, err)
		}
		return kontoID, leserID
	}
	konto, leser := legeKontoAn("Kora", "kora@auskunft-kollege.invalid")
	fremdKonto, _ := legeKontoAn("Fremda", "fremda@auskunft-kollege.invalid")
	titelID := seedMonitorTitel(t, pool, "Kanari-Klassensatz", "Aut Or", false, 0)

	exec(`INSERT INTO klassen_lehrer_mapping (klasse, lehrer_email) VALUES ('KA-07B', 'KORA@auskunft-kollege.invalid')`)
	exec(`INSERT INTO lehrer_anliegen (art, titel_text, klasse, kommentar, angefordert_von)
		VALUES ('wunsch', 'Kanari-Wunschtitel', 'KA-07B', 'bitte zwei', $1),
		       ('wunsch', 'Fremder-Wunschtitel', '', '', $2)`, konto, fremdKonto)
	exec(`INSERT INTO klassensatz_reservierungen (titel_id, klasse, anzahl, notiz, angefordert_von)
		VALUES ($1, 'KA-07B', 28, 'für Montag', $2)`, titelID, konto)
	// Das Verwaltungsprotokoll über die Konten. admin_id bleibt leer: Die Person, die einen
	// Eintrag auslöst, ist nicht die betroffene. Die IP-Adresse gehört ihr ebenso wenig
	// und darf deshalb nirgends in der Antwort auftauchen.
	exec(`INSERT INTO audit_logs (aktion, details, ip_adresse, zeitstempel) VALUES
		('USER_UPDATE', jsonb_build_object('ziel_id', $1::text, 'rolle', 'mitarbeiter'), '10.9.8.7', NOW()),
		('USER_CREATE', jsonb_build_object('email', 'Kora@Auskunft-Kollege.invalid', 'vorname', 'Kora'), '10.9.8.7',
		    (SELECT erstellt_am FROM benutzer WHERE id = $1::uuid) + interval '1 second'),
		('USER_CREATE', jsonb_build_object('email', 'kora@auskunft-kollege.invalid', 'vorname', 'Vorgaengerin'), NULL,
		    NOW() - interval '400 days'),
		('USER_UPDATE', jsonb_build_object('ziel_id', $2::text, 'rolle', 'helfer'), NULL, NOW())`,
		konto, fremdKonto)
	exec(`INSERT INTO audit_logs (admin_id, aktion, details) VALUES ($1, 'SELBSTANMELDUNG', '{"rolle":"kollegium"}')`, konto)

	srv := &Server{DB: &db.Database{Pool: pool}}
	rufe := func(t *testing.T, h http.HandlerFunc, pfad string) *httptest.ResponseRecorder {
		t.Helper()
		req := httptest.NewRequest(http.MethodGet, pfad, nil)
		req.SetPathValue("id", leser)
		rec := httptest.NewRecorder()
		h(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("%s: Status %d, erwartet 200 — %s", pfad, rec.Code, rec.Body.String())
		}
		return rec
	}

	rec := rufe(t, srv.DsgvoAuskunftHandler(), "/api/schueler/"+leser+"/dsgvo-auskunft")
	var a DsgvoAuskunftResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &a); err != nil {
		t.Fatalf("Antwort lesen: %v", err)
	}
	if a.Stammdaten.Art != "lehrkraft" || a.Stammdaten.Vorname != "Kora" {
		t.Errorf("Stammdaten: Art %q, Vorname %q", a.Stammdaten.Art, a.Stammdaten.Vorname)
	}
	k := a.Zugangskonto
	if k == nil {
		t.Fatal("die Auskunft eines Kollegen nennt sein Zugangskonto nicht")
	}
	if k.Email != "kora@auskunft-kollege.invalid" || k.Rolle != "mitarbeiter" || !k.Aktiv {
		t.Errorf("Konto: %+v", k)
	}
	if len(k.Klassenleitungen) != 1 || k.Klassenleitungen[0] != "KA-07B" {
		t.Errorf("Klassenleitungen: %v (erwartet KA-07B — die Adresse im Mapping ist anders geschrieben)", k.Klassenleitungen)
	}
	titel := map[string]string{}
	for _, an := range k.Anfragen {
		titel[an.Titel] = an.Art
	}
	if titel["Kanari-Wunschtitel"] != "wunsch" || titel["Kanari-Klassensatz"] != "klassensatz" || len(titel) != 2 {
		t.Errorf("Anfragen: %v — erwartet genau den eigenen Wunsch und die eigene Reservierung", titel)
	}
	aktionen := map[string]int{}
	for _, e := range k.Ereignisse {
		aktionen[e.Aktion]++
	}
	if aktionen["USER_UPDATE"] != 1 || aktionen["USER_CREATE"] != 1 || aktionen["SELBSTANMELDUNG"] != 1 {
		t.Errorf("Kontoereignisse: %v — erwartet je einen Eintrag (Änderung, Anlage, Selbstanmeldung)", aktionen)
	}

	roh := rec.Body.String()
	for _, fremd := range []string{"Fremder-Wunschtitel", "Vorgaengerin", `"helfer"`, "10.9.8.7"} {
		if strings.Contains(roh, fremd) {
			t.Errorf("die Auskunft enthält %q — das gehört zu einer anderen Person", fremd)
		}
	}
	va := a.Verarbeitungsangaben
	if !strings.Contains(va.Rechtsgrundlage, "§ 23 HDSIG") {
		t.Errorf("Rechtsgrundlage nennt das Beschäftigungsverhältnis nicht: %q", va.Rechtsgrundlage)
	}
	for _, schuelerwort := range []string{"LUSD", "Karenz", "Erziehungsberechtigt", "Abgang"} {
		if strings.Contains(va.Speicherdauer+va.Herkunft+va.Rechtsgrundlage+strings.Join(va.Zwecke, " "), schuelerwort) {
			t.Errorf("die Pflichtangaben eines Kollegen sprechen von %q", schuelerwort)
		}
	}

	pdfRec := rufe(t, srv.DsgvoAuskunftPDFHandler(), "/api/schueler/"+leser+"/dsgvo-auskunft/pdf")
	blatt := strings.Join(pdftest.Texte(t, pdfRec.Body.Bytes()), "\n")
	for _, soll := range []string{"(Lehrkraft)", "kora@auskunft-kollege.invalid", "Kanari-Wunschtitel", "Konto angelegt"} {
		if !strings.Contains(blatt, soll) {
			t.Errorf("das PDF enthält %q nicht", soll)
		}
	}
}

// Die Anlage eines Kontos trägt seit dem 24.09.2026 die ziel_id — wie die Änderung. Daran
// findet die Auskunft den Eintrag, auch wenn die Adresse später geändert wird.
func TestKontoAnlage_ProtokollTraegtZielID(t *testing.T) {
	pool := pgTestPool(t)
	resetBestandsdaten(t, pool)
	ctx := context.Background()
	t.Cleanup(func() {
		for _, sql := range []string{
			`DELETE FROM audit_logs WHERE details->>'email' LIKE '%@zielid.invalid'`,
			`DELETE FROM benutzer WHERE email LIKE '%@zielid.invalid'`,
		} {
			if _, err := pool.Exec(context.Background(), sql); err != nil {
				t.Errorf("aufräumen: %v", err)
			}
		}
	})

	// Ein echtes Administratorkonto: audit_logs.admin_id verweist auf benutzer.
	var adminID string
	if err := pool.QueryRow(ctx, `INSERT INTO benutzer (vorname, nachname, email, rolle, aktiv)
		VALUES ('Ada', 'Admin', 'ada@zielid.invalid', 'admin', true) RETURNING id`).Scan(&adminID); err != nil {
		t.Fatalf("Admin anlegen: %v", err)
	}
	srv := &Server{DB: &db.Database{Pool: pool}}
	req := httptest.NewRequest(http.MethodPost, "/api/benutzer", strings.NewReader(
		`{"vorname":"Neu","nachname":"Konto","email":"neu@zielid.invalid","rolle":"kollegium"}`))
	req.Header.Set("Content-Type", "application/json")
	req = req.WithContext(context.WithValue(req.Context(), auth.ClaimsContextKey,
		&auth.Claims{UserID: adminID, Rolle: auth.RoleAdmin}))
	rec := httptest.NewRecorder()
	srv.CreateUserHandler(repository.NewUserRepository(pool)).ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("Konto anlegen: %d %s", rec.Code, rec.Body.String())
	}

	var gleich bool
	if err := pool.QueryRow(ctx, `
		SELECT COALESCE(l.details->>'ziel_id' = b.id::text, false)
		FROM audit_logs l JOIN benutzer b ON b.email = 'neu@zielid.invalid'
		WHERE l.aktion = 'USER_CREATE' AND l.details->>'email' = 'neu@zielid.invalid'`).Scan(&gleich); err != nil {
		t.Fatalf("Protokolleintrag der Anlage lesen: %v", err)
	}
	if !gleich {
		t.Error("die Anlage protokolliert nicht die ID des neuen Kontos (ziel_id)")
	}
}
