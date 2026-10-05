package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"slices"
	"testing"
	"time"

	"bibliothek/auth"
	"bibliothek/db"
	"bibliothek/repository"
	"bibliothek/sse"

	"github.com/jackc/pgx/v5/pgxpool"
)

// portalWelt baut den echten Router, ein Konto mit Sitzung und setzt der Rolle genau das
// Recht, über das „Mein Portal" sucht, reserviert und meldet.
func portalWelt(t *testing.T) (*pgxpool.Pool, http.Handler, string) {
	t.Helper()
	pool := pgTestPool(t)
	t.Setenv("RATE_LIMIT", "100000")
	resetBestandsdaten(t, pool)
	ctx := context.Background()
	if _, err := pool.Exec(ctx, `TRUNCATE schlagworte CASCADE`); err != nil {
		t.Fatal(err)
	}
	if err := (&db.Database{Pool: pool}).InitPermissions(ctx); err != nil {
		t.Fatalf("InitPermissions: %v", err)
	}
	authenticator, err := auth.NewAuthenticator("kollegium-katalog-testgeheimnis-32-bytes-lang!", pool, time.Hour)
	if err != nil {
		t.Fatalf("Authenticator: %v", err)
	}
	router := NewServer(&db.Database{Pool: pool}, authenticator, sse.NewBroker(), false).Routes()

	var kontoID string
	if err := pool.QueryRow(ctx, `
		INSERT INTO benutzer (vorname, nachname, email, rolle, aktiv)
		VALUES ('Karla', 'Katalog', 'katalog-ma@example.org', 'mitarbeiter', true)
		RETURNING id`).Scan(&kontoID); err != nil {
		t.Fatalf("Konto anlegen: %v", err)
	}
	token, err := authenticator.GenerateToken(kontoID, "KATALOG-MA-1", auth.RoleMitarbeiter, "")
	if err != nil {
		t.Fatalf("Sitzung: %v", err)
	}
	setzeGenauEinRecht(t, pool, "create_reservations")
	return pool, router, token
}

// katalogAnfrage ruft eine Tür über den echten Router; ohne token ohne Sitzung.
func katalogAnfrage(router http.Handler, token, pfad string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, pfad, nil)
	if token != "" {
		req.AddCookie(&http.Cookie{Name: "session_token", Value: token})
	}
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	return rec
}

// „Mein Portal" sucht hinter der Anmeldung und zeigt, was der öffentliche Katalog
// verbirgt: Lernmittel und Titel, deren Exemplare bestellt und noch nicht eingetroffen
// sind. Dort werden die Klassensätze reserviert; ein Schulbuch, das im Bestand steht,
// meldete die Suche sonst als nicht gefunden. Titel ohne ein Exemplar, das nicht
// ausgesondert ist, zeigt keiner der beiden Kataloge.
func TestKollegiumKatalog_ZeigtLernmittelUndBestellteTitel(t *testing.T) {
	pool, router, token := portalWelt(t)
	ctx := context.Background()

	buecherei := titelMitSignatur(t, pool, "Probe Mondflug", "", 0)
	exemplar(t, pool, buecherei, "B-KK-1", true, "")
	lernmittel := titelMitSignatur(t, pool, "LMF-Probe Physik 9", "", 0)
	exemplar(t, pool, lernmittel, "B-KK-2", true, "")
	bestellt := titelMitSignatur(t, pool, "Probe Sternkarte", "", 0)
	for _, barcode := range []string{"B-KK-3", "B-KK-4"} {
		if _, err := pool.Exec(ctx, `
			INSERT INTO buecher_exemplare (titel_id, barcode_id, ist_ausleihbar, bestellstatus)
			VALUES ($1, $2, false, 'bestellt')`, bestellt, barcode); err != nil {
			t.Fatalf("bestelltes Exemplar: %v", err)
		}
	}
	ausgesondert := titelMitSignatur(t, pool, "Probe Altband", "", 0)
	if _, err := pool.Exec(ctx, `
		INSERT INTO buecher_exemplare (titel_id, barcode_id, ist_ausleihbar, ist_ausgesondert, aussonderung_grund)
		VALUES ($1, 'B-KK-5', false, true, 'AUSSORTIERT')`, ausgesondert); err != nil {
		t.Fatalf("ausgesondertes Exemplar: %v", err)
	}
	titelMitSignatur(t, pool, "Probe Leertitel", "", 0)

	// Das Kollegium: Bücherei, Lernmittel und der bestellte Titel, alphabetisch.
	rec := katalogAnfrage(router, token, "/api/reservierungen/klassensatz/katalog?q=Probe")
	var treffer []KollegiumTitel
	if err := json.Unmarshal(rec.Body.Bytes(), &treffer); err != nil || rec.Code != http.StatusOK {
		t.Fatalf("Katalog des Kollegiums: %d %s (%v)", rec.Code, rec.Body.String(), err)
	}
	var namen []string
	for _, b := range treffer {
		namen = append(namen, b.Titel)
	}
	if will := []string{"LMF-Probe Physik 9", "Probe Mondflug", "Probe Sternkarte"}; !slices.Equal(namen, will) {
		t.Fatalf("Katalog des Kollegiums zeigt %q, erwartet %q", namen, will)
	}
	if gesamt := rec.Header().Get("X-Treffer-Gesamt"); gesamt != "3" {
		t.Errorf("X-Treffer-Gesamt %q, erwartet 3", gesamt)
	}
	if b := treffer[2]; b.Gesamt != 0 || b.Verfuegbar != 0 || b.ImZulauf != 2 {
		t.Errorf("bestellter Titel: gesamt %d, verfügbar %d, bestellt %d — erwartet 0, 0, 2", b.Gesamt, b.Verfuegbar, b.ImZulauf)
	}
	if b := treffer[0]; b.Gesamt != 1 || b.Verfuegbar != 1 || b.ImZulauf != 0 {
		t.Errorf("Lernmittel im Regal: gesamt %d, verfügbar %d, bestellt %d — erwartet 1, 1, 0", b.Gesamt, b.Verfuegbar, b.ImZulauf)
	}

	// Ohne Anmeldung bleibt es bei der Bücherei.
	rec = katalogAnfrage(router, "", "/api/public/opac/suche?q=Probe")
	var oeffentlich []OpacTitel
	if err := json.Unmarshal(rec.Body.Bytes(), &oeffentlich); err != nil || rec.Code != http.StatusOK {
		t.Fatalf("öffentlicher Katalog: %d %s (%v)", rec.Code, rec.Body.String(), err)
	}
	if len(oeffentlich) != 1 || oeffentlich[0].Titel != "Probe Mondflug" {
		t.Errorf("öffentlicher Katalog zeigt %+v, erwartet nur „Probe Mondflug“", oeffentlich)
	}

	// Die Tür des Kollegiums geht nur mit Sitzung und mit dem Recht auf.
	if rec := katalogAnfrage(router, "", "/api/reservierungen/klassensatz/katalog?q=Probe"); rec.Code != http.StatusUnauthorized {
		t.Errorf("ohne Sitzung: HTTP %d statt 401", rec.Code)
	}
	setzeGenauEinRecht(t, pool, "view_books")
	for _, pfad := range []string{"/api/reservierungen/klassensatz/katalog?q=Probe", "/api/reservierungen/klassensatz/katalog/filter"} {
		if rec := katalogAnfrage(router, token, pfad); rec.Code != http.StatusForbidden {
			t.Errorf("%s ohne create_reservations: HTTP %d statt 403", pfad, rec.Code)
		}
	}
}

// Der Filter unter der Suche: Die Liste nennt nur markierte Wörter, zu denen der Katalog
// des Kollegiums einen Titel zeigt — ein Filter, der nichts findet, wäre eine Sackgasse.
// Die gefilterte Suche geht ohne und mit Suchtext, und der Kopf X-Treffer-Gesamt sagt, wie
// viele Titel es sind, wenn die Antwort bei 50 abschneidet: Beim Stöbern über ein Thema
// sind mehr als 50 Titel der Normalfall. Der öffentliche Katalog kennt den Filter nicht.
func TestKollegiumKatalog_FilterlisteUndGefilterteSuche(t *testing.T) {
	pool, router, token := portalWelt(t)
	ctx := context.Background()

	titel := func(name string, woerter ...string) string {
		t.Helper()
		id := titelMitSignatur(t, pool, name, "", 0)
		exemplar(t, pool, id, "B-FILTER-"+name, true, "")
		if _, err := repository.SetzeSchlagworte(ctx, pool, id, woerter); err != nil {
			t.Fatal(err)
		}
		return id
	}
	titel("Mondflug", "Weltraum", "Abenteuer")
	titel("Krabat", "Sage")         // ohne „Weltraum": zählt beim Filter nicht mit
	titel("LMF-Physik 9", "Physik") // Lernmittel: nur im Katalog des Kollegiums
	for i := range 51 {
		titel(fmt.Sprintf("Sternfahrt %02d", i), "Weltraum")
	}
	// Ein markiertes Wort, das nur ein Titel ohne Exemplar trägt, findet nichts.
	ohneExemplar := titelMitSignatur(t, pool, "Leerband", "", 0)
	if _, err := repository.SetzeSchlagworte(ctx, pool, ohneExemplar, []string{"Vergriffen"}); err != nil {
		t.Fatal(err)
	}
	wortID := func(wort string) string {
		t.Helper()
		var id string
		if err := pool.QueryRow(ctx, `SELECT id FROM schlagworte WHERE wort = $1`, wort).Scan(&id); err != nil {
			t.Fatal(err)
		}
		return id
	}
	for _, wort := range []string{"Weltraum", "Physik", "Vergriffen"} { // „Abenteuer" bleibt unmarkiert
		if err := repository.SetzeSchlagwortFilter(ctx, pool, wortID(wort), true); err != nil {
			t.Fatal(err)
		}
	}

	// Die Liste: markiert und mit einem Titel in diesem Katalog, alphabetisch.
	rec := katalogAnfrage(router, token, "/api/reservierungen/klassensatz/katalog/filter")
	var filter []repository.SchlagwortFilter
	if err := json.Unmarshal(rec.Body.Bytes(), &filter); err != nil || rec.Code != http.StatusOK {
		t.Fatalf("Filterliste: %d %s (%v)", rec.Code, rec.Body.String(), err)
	}
	var woerter []string
	for _, f := range filter {
		woerter = append(woerter, f.Wort)
	}
	if will := []string{"Physik", "Weltraum"}; !slices.Equal(woerter, will) {
		t.Errorf("Filterliste %q, erwartet %q — „Abenteuer“ ist nicht markiert, „Vergriffen“ trägt nur ein Titel ohne Exemplar", woerter, will)
	}

	suche := func(query string) (int, []string, string) {
		t.Helper()
		rec := katalogAnfrage(router, token, "/api/reservierungen/klassensatz/katalog?"+query)
		var treffer []KollegiumTitel
		if rec.Code == http.StatusOK {
			if err := json.Unmarshal(rec.Body.Bytes(), &treffer); err != nil {
				t.Fatal(err)
			}
		}
		var namen []string
		for _, b := range treffer {
			namen = append(namen, b.Titel)
		}
		return rec.Code, namen, rec.Header().Get("X-Treffer-Gesamt")
	}

	weltraum := url.QueryEscape(wortID("Weltraum"))
	if code, namen, gesamt := suche("schlagwort_id=" + weltraum); code != http.StatusOK || len(namen) != 50 || gesamt != "52" {
		t.Errorf("nur Filter: HTTP %d, %d Titel, X-Treffer-Gesamt %q — erwartet 50 gezeigt von 52", code, len(namen), gesamt)
	}
	if code, namen, gesamt := suche("q=mond&schlagwort_id=" + weltraum); code != http.StatusOK || !slices.Equal(namen, []string{"Mondflug"}) || gesamt != "1" {
		t.Errorf("Filter und Suchtext: HTTP %d, %q, X-Treffer-Gesamt %q", code, namen, gesamt)
	}
	if code, namen, _ := suche("schlagwort_id=" + url.QueryEscape(wortID("Physik"))); code != http.StatusOK || !slices.Equal(namen, []string{"LMF-Physik 9"}) {
		t.Errorf("Filter über das Wort eines Lernmittels: HTTP %d, %q", code, namen)
	}
	if code, namen, _ := suche("q=mond&schlagwort_id=" + url.QueryEscape(wortID("Abenteuer"))); code != http.StatusOK || !slices.Equal(namen, []string{"Mondflug"}) {
		t.Errorf("Filter über ein unmarkiertes Wort: HTTP %d, %q", code, namen)
	}
	if code, namen, _ := suche("q=Sternfahrt&schlagwort_id=" + url.QueryEscape(wortID("Physik"))); code != http.StatusOK || len(namen) != 0 {
		t.Errorf("Filter ohne gemeinsamen Titel: HTTP %d, %q — erwartet nichts", code, namen)
	}
	if code, _, _ := suche("schlagwort_id=kein-wort"); code != http.StatusBadRequest {
		t.Errorf("kaputte Kennung: HTTP %d statt 400", code)
	}

	// Der öffentliche Katalog liest schlagwort_id nicht: ohne Suchtext eine leere Liste.
	rec = katalogAnfrage(router, "", "/api/public/opac/suche?schlagwort_id="+weltraum)
	if rumpf := rec.Body.String(); rec.Code != http.StatusOK || rumpf != "[]" {
		t.Errorf("öffentlicher Katalog mit schlagwort_id: HTTP %d, %q — erwartet 200 und eine leere Liste", rec.Code, rumpf)
	}
}
