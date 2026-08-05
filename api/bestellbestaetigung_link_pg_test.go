package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"bibliothek/db"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Der Bestätigungs-Link ist der einzige schreibende Zugang ohne Anmeldung. Diese Tests
// gehen ihn am echten Postgres durch — vom Token bis zum zweiten Klick.

// bestellungMitToken legt eine Bestellung samt Link-Token an und liefert den KLARTEXT.
func bestellungMitToken(t *testing.T, pool *pgxpool.Pool, lieferantID string, gueltigTage int) (bestellungID, token string) {
	t.Helper()
	token, hash, err := neuerBestaetigungsToken()
	if err != nil {
		t.Fatalf("Token erzeugen: %v", err)
	}
	err = pool.QueryRow(context.Background(), `
		INSERT INTO bestellungen_verlauf
			(lieferant_id, lieferant_name, lieferant_email, kundennummer, anzahl_exemplare,
			 bestaetigungs_token_hash, token_gueltig_bis)
		VALUES ($1, 'Naacher', 'naacher@example.invalid', 'K-1', 2, $2, now() + make_interval(days => $3))
		RETURNING id
	`, lieferantID, hash, gueltigTage).Scan(&bestellungID)
	if err != nil {
		t.Fatalf("Bestellung mit Token anlegen: %v", err)
	}
	return bestellungID, token
}

func getOeffentlicheBestellung(srv *Server, token string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, "/api/public/bestellung/"+token, nil)
	req.SetPathValue("token", token)
	rec := httptest.NewRecorder()
	srv.OeffentlicheBestellungHandler().ServeHTTP(rec, req)
	return rec
}

func postOeffentlichBestaetigen(srv *Server, token, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, "/api/public/bestellung/"+token+"/bestaetigen", strings.NewReader(body))
	req.SetPathValue("token", token)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	srv.OeffentlichBestaetigenHandler().ServeHTTP(rec, req)
	return rec
}

// Der Regelweg: Lieferant öffnet den Link, sieht seine Bestellung, bestätigt einmal —
// und die Bibliothek sieht in der Historie, dass ER es war und nicht jemand im Haus.
func TestOeffentlicheBestaetigung_Regelweg(t *testing.T) {
	pool := pgTestPool(t)
	resetBestandsdaten(t, pool)
	ctx := context.Background()
	srv := &Server{DB: &db.Database{Pool: pool}}

	lieferant := haendlerMitBestaetigung(t, pool, "Naacher", true)
	bestellungID, token := bestellungMitToken(t, pool, lieferant, TokenGueltigkeitTage)

	rec := getOeffentlicheBestellung(srv, token)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET Status = %d, want 200, body: %s", rec.Code, rec.Body.String())
	}
	var ansicht OeffentlicheBestellung
	if err := json.Unmarshal(rec.Body.Bytes(), &ansicht); err != nil {
		t.Fatalf("Antwort lesen: %v", err)
	}
	if ansicht.LieferantName != "Naacher" || ansicht.BestaetigtAm != nil {
		t.Fatalf("Ansicht = %+v, want Naacher und unbestätigt", ansicht)
	}

	if rec := postOeffentlichBestaetigen(srv, token, `{"etiketten_groesse":"gross"}`); rec.Code != http.StatusOK {
		t.Fatalf("POST Status = %d, want 200, body: %s", rec.Code, rec.Body.String())
	}

	var durch, groesse string
	var bestaetigt bool
	err := pool.QueryRow(ctx, `
		SELECT bestaetigt_am IS NOT NULL, coalesce(bestaetigt_durch, ''), coalesce(etiketten_groesse, '')
		FROM bestellungen_verlauf WHERE id = $1
	`, bestellungID).Scan(&bestaetigt, &durch, &groesse)
	if err != nil {
		t.Fatalf("Zustand lesen: %v", err)
	}
	if !bestaetigt || durch != "lieferant" || groesse != "gross" {
		t.Fatalf("bestaetigt=%v durch=%q groesse=%q — want true/lieferant/gross", bestaetigt, durch, groesse)
	}
}

// Zweimal bestätigen darf nicht zweimal gelten: Der Lieferant klickt doppelt, oder er
// bestätigt, während jemand im Haus dasselbe nachträgt. Genau einer gewinnt.
func TestOeffentlicheBestaetigung_ZweiterKlickIstKonflikt(t *testing.T) {
	pool := pgTestPool(t)
	resetBestandsdaten(t, pool)
	srv := &Server{DB: &db.Database{Pool: pool}}

	lieferant := haendlerMitBestaetigung(t, pool, "Naacher", true)
	_, token := bestellungMitToken(t, pool, lieferant, TokenGueltigkeitTage)

	if rec := postOeffentlichBestaetigen(srv, token, `{}`); rec.Code != http.StatusOK {
		t.Fatalf("erster POST = %d, want 200", rec.Code)
	}
	rec := postOeffentlichBestaetigen(srv, token, `{}`)
	if rec.Code != http.StatusConflict {
		t.Fatalf("zweiter POST = %d, want 409, body: %s", rec.Code, rec.Body.String())
	}
}

// Abgelaufen und erfunden sehen von außen gleich aus — sonst verriete die Antwort,
// dass ein geratener Token einmal echt war.
func TestOeffentlicheBestaetigung_AbgelaufenUndUnbekannt(t *testing.T) {
	pool := pgTestPool(t)
	resetBestandsdaten(t, pool)
	srv := &Server{DB: &db.Database{Pool: pool}}

	lieferant := haendlerMitBestaetigung(t, pool, "Naacher", true)
	_, abgelaufen := bestellungMitToken(t, pool, lieferant, -1)

	faelle := map[string]string{
		"abgelaufen": abgelaufen,
		"unbekannt":  "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA",
	}
	for name, token := range faelle {
		t.Run(name, func(t *testing.T) {
			if rec := getOeffentlicheBestellung(srv, token); rec.Code != http.StatusNotFound {
				t.Fatalf("GET = %d, want 404", rec.Code)
			}
			if rec := postOeffentlichBestaetigen(srv, token, `{}`); rec.Code != http.StatusNotFound {
				t.Fatalf("POST = %d, want 404", rec.Code)
			}
		})
	}
}

// Ein neuer Link tötet den alten. Das ist der Zweck des Knopfes in der Historie: Ging
// die Mail an die falsche Adresse, darf der verschickte Link nicht weiterleben.
func TestNeuerLinkEntwertetDenAlten(t *testing.T) {
	pool := pgTestPool(t)
	resetBestandsdaten(t, pool)
	ctx := context.Background()
	srv := &Server{DB: &db.Database{Pool: pool}}

	lieferant := haendlerMitBestaetigung(t, pool, "Naacher", true)
	bestellungID, alterToken := bestellungMitToken(t, pool, lieferant, TokenGueltigkeitTage)

	if _, err := pool.Exec(ctx,
		`INSERT INTO system_einstellungen (schluessel, wert) VALUES ('oeffentliche_adresse', 'https://bib.example.invalid')
		 ON CONFLICT (schluessel) DO UPDATE SET wert = EXCLUDED.wert`); err != nil {
		t.Fatalf("Einstellung setzen: %v", err)
	}

	req := httptest.NewRequest(http.MethodPut, "/api/bestellungen/"+bestellungID+"/bestaetigungs-link", nil)
	req.SetPathValue("id", bestellungID)
	rec := httptest.NewRecorder()
	srv.NeuerBestaetigungsLinkHandler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("Status = %d, want 200, body: %s", rec.Code, rec.Body.String())
	}

	var antwort struct {
		Link string `json:"link"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &antwort); err != nil {
		t.Fatalf("Antwort lesen: %v", err)
	}
	if !strings.HasPrefix(antwort.Link, "https://bib.example.invalid/bestellung/") {
		t.Fatalf("Link = %q — erwartet Adresse aus den Einstellungen", antwort.Link)
	}

	if rec := getOeffentlicheBestellung(srv, alterToken); rec.Code != http.StatusNotFound {
		t.Fatalf("alter Token = %d, want 404 (er muss tot sein)", rec.Code)
	}
	neuerToken := strings.TrimPrefix(antwort.Link, "https://bib.example.invalid/bestellung/")
	if rec := getOeffentlicheBestellung(srv, neuerToken); rec.Code != http.StatusOK {
		t.Fatalf("neuer Token = %d, want 200", rec.Code)
	}
}
