package benutzer

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"bibliothek/api/tuer"
	"bibliothek/auth"
	"bibliothek/db"
	"bibliothek/internal/pgtest"
	"bibliothek/internal/testdaten"
	"bibliothek/internal/tuertest"
	"bibliothek/repository"
)

// Was das Löschen eines Zugangskontos ablehnt, über die Tür: das eigene Konto, das Konto
// eines Administrators durch jemanden ohne Adminrechte, ein Konto mit offenen Ausleihen und
// eine unbekannte Kennung. Jede Ablehnung lässt das Konto stehen; erst der letzte Fall löscht.
func TestKontoLoeschen_AblehnungenUeberDieTuer(t *testing.T) {
	pool := pgtest.Pool(t)
	testdaten.LeereBestandsdaten(t, pool)
	ctx := context.Background()
	srv := &Tueren{Basis: &tuer.Basis{DB: &db.Verbindung{Pool: pool}}}
	userRepo := repository.NewKontoAbfragen(pool)

	konto := func(vorname, rolle string) string {
		t.Helper()
		var id string
		if err := pool.QueryRow(ctx, `
			INSERT INTO benutzer (vorname, nachname, email, rolle, aktiv)
			VALUES ($1, 'Loeschtuer', $2, $3::benutzer_rolle, true)
			RETURNING id::text`, vorname, strings.ToLower(vorname)+"@loeschtuer.invalid", rolle).Scan(&id); err != nil {
			t.Fatalf("Konto %s anlegen: %v", vorname, err)
		}
		return id
	}
	admin, zweiterAdmin := konto("Ada", "admin"), konto("Alex", "admin")
	mitarbeiter := konto("Mia", "mitarbeiter")
	mitBuch, ohneBuch := konto("Bert", "kollegium"), konto("Kai", "kollegium")

	var leserMitBuch string
	if err := pool.QueryRow(ctx, `SELECT leser_id::text FROM benutzer WHERE id = $1`, mitBuch).Scan(&leserMitBuch); err != nil {
		t.Fatalf("Leserzeile des Kontos lesen: %v", err)
	}
	testdaten.LegeAusleiheMitFristAn(t, pool, leserMitBuch, "Handapparat Loeschtuer", time.Now().Add(30*24*time.Hour))

	loesche := func(ziel string, claims *auth.Claims) *httptest.ResponseRecorder {
		t.Helper()
		req := httptest.NewRequest(http.MethodDelete, "/api/benutzer/"+ziel, nil)
		req.SetPathValue("id", ziel)
		if claims != nil {
			req = req.WithContext(context.WithValue(req.Context(), auth.ClaimsSchluessel, claims))
		}
		rec := httptest.NewRecorder()
		srv.KontoLoeschen(userRepo)(rec, req)
		return rec
	}
	alsAdmin := &auth.Claims{KontoID: admin, Rolle: auth.RolleAdmin}
	alsMitarbeiter := &auth.Claims{KontoID: mitarbeiter, Rolle: auth.RolleMitarbeiter}

	for _, f := range []struct {
		name   string
		ziel   string
		claims *auth.Claims
		status int
		stueck string
	}{
		{"ohne Sitzung", ohneBuch, nil, http.StatusUnauthorized, "Nicht angemeldet. Bitte neu anmelden."},
		{"das eigene Konto", admin, alsAdmin, http.StatusForbidden, "Das eigene Konto lässt sich nicht löschen"},
		{"ein Administrator, gelöscht ohne Adminrechte", zweiterAdmin, alsMitarbeiter, http.StatusForbidden, "Administrator"},
		{"ein Konto mit offener Ausleihe", mitBuch, alsAdmin, http.StatusConflict,
			"Das Konto hat noch ausgeliehene Exemplare (1 offen). Bitte zuerst zurückbuchen."},
		{"eine unbekannte Kennung", "00000000-0000-0000-0000-00000000dead", alsAdmin, http.StatusNotFound, "error"},
	} {
		rec := loesche(f.ziel, f.claims)
		if rec.Code != f.status {
			t.Errorf("%s: Status %d, erwartet %d: %s", f.name, rec.Code, f.status, rec.Body.String())
		}
		if !strings.Contains(rec.Body.String(), f.stueck) {
			t.Errorf("%s: die Antwort nennt %q nicht: %s", f.name, f.stueck, rec.Body.String())
		}
		// Den Satz des Konflikts liest die Verwaltung der Konten; er steht ganz da.
		if f.status == http.StatusConflict {
			if got := tuertest.Fehlermeldung(t, rec); got != f.stueck {
				t.Errorf("%s: Meldung %q, erwartet %q", f.name, got, f.stueck)
			}
		}
	}
	if n := pgtest.ZaehleZeilen(t, pool, `SELECT count(*) FROM benutzer WHERE email LIKE '%@loeschtuer.invalid'`); n != 5 {
		t.Fatalf("nach fünf Ablehnungen stehen %d von 5 Konten", n)
	}

	if rec := loesche(ohneBuch, alsMitarbeiter); rec.Code != http.StatusOK {
		t.Fatalf("ein Kollegiumskonto ohne Ausleihen: Status %d, erwartet 200: %s", rec.Code, rec.Body.String())
	}
	if n := pgtest.ZaehleZeilen(t, pool, `SELECT count(*) FROM benutzer WHERE id = $1`, ohneBuch); n != 0 {
		t.Errorf("das gelöschte Konto steht noch da")
	}
	// Das Protokoll nennt, wer gelöscht hat; ohne Bearbeiter stünde die Löschung als SYSTEM da.
	if n := pgtest.ZaehleZeilen(t, pool, `
		SELECT count(*) FROM audit_log
		 WHERE tabelle = 'benutzer' AND aktion = 'DELETE' AND datensatz_id = $1
		   AND bearbeiter_id = $2 AND akteur = 'USER'`, ohneBuch, mitarbeiter); n != 1 {
		t.Errorf("%d Einträge der Löschung mit der löschenden Person als Bearbeiter, erwartet 1", n)
	}
}

// Ein Konto zu ändern, das es nicht gibt, ist ein 404 und hinterlässt keinen Eintrag im
// Protokoll: Der Eintrag beschriebe eine Änderung, die nie stattfand.
func TestKontoAendern_UnbekannteKennungOhneProtokoll(t *testing.T) {
	pool := pgtest.Pool(t)
	testdaten.LeereBestandsdaten(t, pool)
	ctx := context.Background()
	srv := &Tueren{Basis: &tuer.Basis{DB: &db.Verbindung{Pool: pool}}}

	var admin string
	if err := pool.QueryRow(ctx, `
		INSERT INTO benutzer (vorname, nachname, email, rolle, aktiv)
		VALUES ('Ada', 'Aendertuer', 'ada@aendertuer.invalid', 'admin', true) RETURNING id::text`).Scan(&admin); err != nil {
		t.Fatalf("Konto anlegen: %v", err)
	}
	const unbekannt = "00000000-0000-0000-0000-00000000dead"
	req := httptest.NewRequest(http.MethodPut, "/api/benutzer/"+unbekannt,
		strings.NewReader(`{"vorname":"Nie","nachname":"Da","email":"nie.da@aendertuer.invalid","rolle":"helfer","aktiv":true}`))
	req.Header.Set("Content-Type", "application/json")
	req.SetPathValue("id", unbekannt)
	req = req.WithContext(context.WithValue(req.Context(), auth.ClaimsSchluessel,
		&auth.Claims{KontoID: admin, Rolle: auth.RolleAdmin}))
	rec := httptest.NewRecorder()
	srv.KontoAendern(repository.NewKontoAbfragen(pool))(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("Status %d, erwartet 404: %s", rec.Code, rec.Body.String())
	}
	if n := pgtest.ZaehleZeilen(t, pool, `SELECT count(*) FROM audit_logs WHERE aktion = 'USER_UPDATE'`); n != 0 {
		t.Errorf("%d Einträge USER_UPDATE zu einer Änderung, die es nicht gab", n)
	}
}

// Anlegen und Ändern lehnen eine Adresse, die in irgendeiner Schreibweise schon ein anderes Konto
// trägt, mit einem Satz für die Maske ab; ohne die Prüfung scheiterte erst der eindeutige Index, und
// die Maske zeigte einen Serverfehler. Die eigene Adresse in neuer Schreibweise bleibt speicherbar.
func TestKonto_VergebeneEmailWirdAbgelehnt(t *testing.T) {
	pool := pgtest.Pool(t)
	testdaten.LeereBestandsdaten(t, pool)
	ctx := context.Background()
	srv := &Tueren{Basis: &tuer.Basis{DB: &db.Verbindung{Pool: pool}}}
	kontoAbfragen := repository.NewKontoAbfragen(pool)

	konto := func(vorname, email, rolle string) string {
		t.Helper()
		var id string
		if err := pool.QueryRow(ctx, `
			INSERT INTO benutzer (vorname, nachname, email, rolle, aktiv)
			VALUES ($1, 'Emailtuer', $2, $3::benutzer_rolle, true) RETURNING id::text`, vorname, email, rolle).Scan(&id); err != nil {
			t.Fatalf("Konto %s anlegen: %v", vorname, err)
		}
		return id
	}
	admin := konto("Ada", "ada@emailtuer.invalid", "admin")
	ida := konto("Ida", "ida@emailtuer.invalid", "kollegium")
	bert := konto("Bert", "bert@emailtuer.invalid", "kollegium")
	alsAdmin := &auth.Claims{KontoID: admin, Rolle: auth.RolleAdmin}

	rufe := func(tuerAufruf http.HandlerFunc, methode, ziel, rumpf string) *httptest.ResponseRecorder {
		t.Helper()
		req := httptest.NewRequest(methode, "/api/benutzer/"+ziel, strings.NewReader(rumpf))
		req.Header.Set("Content-Type", "application/json")
		req.SetPathValue("id", ziel)
		req = req.WithContext(context.WithValue(req.Context(), auth.ClaimsSchluessel, alsAdmin))
		rec := httptest.NewRecorder()
		tuerAufruf(rec, req)
		return rec
	}
	emailVon := func(id string) string {
		t.Helper()
		var email string
		if err := pool.QueryRow(ctx, `SELECT email FROM benutzer WHERE id = $1`, id).Scan(&email); err != nil {
			t.Fatalf("Adresse lesen: %v", err)
		}
		return email
	}
	const meldung = "Ein Konto mit dieser E-Mail-Adresse gibt es bereits"

	rec := rufe(srv.KontoAnlegen(kontoAbfragen), http.MethodPost, "",
		`{"vorname":"Igor","nachname":"Neu","email":"IDA@Emailtuer.invalid","rolle":"helfer"}`)
	if rec.Code != http.StatusBadRequest || tuertest.Fehlermeldung(t, rec) != meldung {
		t.Errorf("Anlegen mit vergebener Adresse: Status %d, erwartet 400 mit %q: %s", rec.Code, meldung, rec.Body.String())
	}
	if n := pgtest.ZaehleZeilen(t, pool, `SELECT count(*) FROM benutzer WHERE lower(email) = 'ida@emailtuer.invalid'`); n != 1 {
		t.Errorf("nach dem abgelehnten Anlegen tragen %d Konten die Adresse, erwartet 1", n)
	}

	rec = rufe(srv.KontoAendern(kontoAbfragen), http.MethodPut, bert, `{"email":"Ida@emailtuer.invalid"}`)
	if rec.Code != http.StatusBadRequest || tuertest.Fehlermeldung(t, rec) != meldung {
		t.Errorf("Ändern auf eine vergebene Adresse: Status %d, erwartet 400 mit %q: %s", rec.Code, meldung, rec.Body.String())
	}
	if got := emailVon(bert); got != "bert@emailtuer.invalid" {
		t.Errorf("nach dem abgelehnten Ändern lautet die Adresse %q, erwartet die alte", got)
	}

	if rec := rufe(srv.KontoAendern(kontoAbfragen), http.MethodPut, ida, `{"email":"IDA@emailtuer.invalid"}`); rec.Code != http.StatusOK {
		t.Errorf("die eigene Adresse in neuer Schreibweise: Status %d, erwartet 200: %s", rec.Code, rec.Body.String())
	}
	if got := emailVon(ida); got != "IDA@emailtuer.invalid" {
		t.Errorf("Adresse nach dem Ändern %q, erwartet %q", got, "IDA@emailtuer.invalid")
	}
}
