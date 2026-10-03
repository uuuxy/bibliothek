package inventur

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"bibliothek/internal/pgtest"
)

// Die Türen der Buchmaske an Titeln ohne ISBN. Rund ein Drittel der Titel aus Littera trägt
// keine (Zeitschriften, DVDs, Spiele, alte Bücher), oder die Übernahme hat sie weggelassen,
// weil die Prüfziffer nicht stimmt oder ein anderer Titel die Nummer trägt. Die Tests gehen
// über die Routen der Maske (PUT /api/books/{id}, POST /api/books) und lesen danach die
// Datenbank.

const ohneISBNVorsatz = "Ohne-ISBN-Probe "

// Die ISBN der Probe. Abgeräumt wird auch über sie: Ein Titel, der ohne den Vorsatz im Namen
// entsteht, bliebe sonst in der Test-Datenbank liegen.
const (
	ohneISBNVergeben  = "9783000471100"
	ohneISBNFrei      = "9783000471117"
	ohneISBNGeleert   = "9783000471124"
	ohneISBNMitNummer = "9783000471131"
	ohneISBNOhneTitel = "9783000471148"
)

type ohneISBNProbe struct {
	t       *testing.T
	handler *APIHandler
	sql     func(abfrage string, werte ...any) string
}

// neueOhneISBNProbe stellt die Tür der Maske vor die Test-Datenbank und räumt die Titel der
// Probe vorher und nachher ab.
func neueOhneISBNProbe(t *testing.T) *ohneISBNProbe {
	t.Helper()
	pool := pgtest.Pool(t)
	ctx := context.Background()
	loesche := func() {
		if _, err := pool.Exec(ctx, `DELETE FROM buecher_titel WHERE titel LIKE $1 OR isbn = ANY($2)`, ohneISBNVorsatz+"%",
			[]string{ohneISBNVergeben, ohneISBNFrei, ohneISBNGeleert, ohneISBNMitNummer, ohneISBNOhneTitel}); err != nil {
			t.Logf("Aufräumen: %v", err)
		}
	}
	loesche()
	t.Cleanup(loesche)

	durchlass := func(next http.Handler) http.Handler { return next }
	return &ohneISBNProbe{
		t: t,
		handler: NewAPIHandler(APIHandlerConfig{
			Repo:                 NewBookRepository(pool),
			Metadaten:            offlineMetadatenClient(),
			RequireViewBooks:     durchlass,
			RequireEditBooks:     durchlass,
			RequireDeleteBooks:   durchlass,
			RequireAuthenticated: durchlass,
		}),
		sql: func(abfrage string, werte ...any) string {
			t.Helper()
			var wert string
			if err := pool.QueryRow(ctx, abfrage, werte...).Scan(&wert); err != nil {
				t.Fatalf("%s: %v", abfrage, err)
			}
			return wert
		},
	}
}

type ohneISBNAntwort struct {
	Status int
	Rumpf  string
	Error  string `json:"error"`
	Data   struct {
		ID string `json:"id"`
	} `json:"data"`
	Vorhanden *struct {
		ID           string `json:"id"`
		Title        string `json:"title"`
		OhneExemplar bool   `json:"ohneExemplar"`
	} `json:"vorhanden"`
	GleicherTitel bool `json:"gleicherTitel"`
}

// schicke ruft die Tür wie die Maske: ohne Kennung POST, mit Kennung PUT.
func (p *ohneISBNProbe) schicke(id string, rumpf map[string]any) ohneISBNAntwort {
	p.t.Helper()
	daten, err := json.Marshal(rumpf)
	if err != nil {
		p.t.Fatal(err)
	}
	methode, pfad := http.MethodPost, "/api/books"
	if id != "" {
		methode, pfad = http.MethodPut, "/api/books/"+id
	}
	rec := httptest.NewRecorder()
	p.handler.ServeHTTP(rec, httptest.NewRequest(methode, pfad, bytes.NewReader(daten)))
	antwort := ohneISBNAntwort{Status: rec.Code, Rumpf: rec.Body.String()}
	if err := json.Unmarshal(rec.Body.Bytes(), &antwort); err != nil {
		p.t.Fatalf("Antwort ist kein JSON: %v — %s", err, rec.Body.String())
	}
	return antwort
}

// lege schreibt einen Titel an der Maske vorbei, wie ihn die Übernahme und die Importe
// hinterlassen. isbn nil ist „ohne ISBN".
func (p *ohneISBNProbe) lege(name, autor string, isbn any) string {
	p.t.Helper()
	return p.sql(`INSERT INTO buecher_titel (titel, autor, isbn) VALUES ($1, NULLIF($2, ''), $3) RETURNING id::text`,
		ohneISBNVorsatz+name, autor, isbn)
}

func (p *ohneISBNProbe) isbnVon(id string) string {
	p.t.Helper()
	return p.sql(`SELECT COALESCE(isbn, '(keine)') FROM buecher_titel WHERE id = $1`, id)
}

func (p *ohneISBNProbe) signaturVon(id string) string {
	p.t.Helper()
	return p.sql(`SELECT COALESCE(signatur, '') FROM buecher_titel WHERE id = $1`, id)
}

func (p *ohneISBNProbe) zaehle(name string) string {
	p.t.Helper()
	return p.sql(`SELECT count(*)::text FROM buecher_titel WHERE titel = $1`, ohneISBNVorsatz+name)
}

// Beim Ändern prüft die Tür, was sich ändert. Was unverändert am Titel steht, hindert das
// Speichern eines anderen Felds nicht.
func TestTitelOhneISBN_Aendern(t *testing.T) {
	p := neueOhneISBNProbe(t)

	t.Run("ein Titel ohne ISBN nimmt eine Signatur an und bleibt ohne ISBN", func(t *testing.T) {
		id := p.lege("allein", "Autorin", nil)
		a := p.schicke(id, map[string]any{"isbn": "", "title": ohneISBNVorsatz + "allein", "author": "Autorin", "signatur": "Pro 1"})
		if a.Status != http.StatusOK || p.signaturVon(id) != "Pro 1" || p.isbnVon(id) != "(keine)" {
			t.Errorf("Status %d %s, Signatur %q, ISBN %q", a.Status, a.Rumpf, p.signaturVon(id), p.isbnVon(id))
		}
	})

	t.Run("zwei Hefte mit gleichem Titel und Autor bleiben beide speicherbar", func(t *testing.T) {
		heft := p.lege("Zeitschrift", "Redaktion", nil)
		p.lege("Zeitschrift", "Redaktion", nil)
		a := p.schicke(heft, map[string]any{"isbn": "", "title": ohneISBNVorsatz + "Zeitschrift", "author": "Redaktion", "signatur": "Pro 2"})
		if a.Status != http.StatusOK || p.signaturVon(heft) != "Pro 2" {
			t.Errorf("Status %d %s, Signatur %q", a.Status, a.Rumpf, p.signaturVon(heft))
		}
	})

	t.Run("eine Nummer, die keine ISBN ist, hindert nicht, solange sie stehen bleibt", func(t *testing.T) {
		id := p.lege("neunstellig", "Autorin", "312674058")
		a := p.schicke(id, map[string]any{"isbn": "312674058", "title": ohneISBNVorsatz + "neunstellig", "author": "Autorin", "signatur": "Pro 3"})
		if a.Status != http.StatusOK || p.signaturVon(id) != "Pro 3" || p.isbnVon(id) != "312674058" {
			t.Errorf("Status %d %s, Signatur %q, ISBN %q", a.Status, a.Rumpf, p.signaturVon(id), p.isbnVon(id))
		}
	})

	t.Run("eine geänderte ISBN wird geprüft", func(t *testing.T) {
		p.lege("traegt die Nummer", "Autorin", ohneISBNVergeben)
		id := p.lege("bekommt eine Nummer", "Autorin", nil)
		rumpf := func(isbn string) map[string]any {
			return map[string]any{"isbn": isbn, "title": ohneISBNVorsatz + "bekommt eine Nummer", "author": "Autorin"}
		}

		if a := p.schicke(id, rumpf("12345")); a.Status != http.StatusBadRequest || a.Error != "ungültiges ISBN-Format" {
			t.Errorf("Nummer ohne ISBN-Form: Status %d %s", a.Status, a.Rumpf)
		}
		a := p.schicke(id, rumpf("978-3-00-047110-0"))
		if a.Status != http.StatusConflict || a.Vorhanden == nil || a.Vorhanden.Title != ohneISBNVorsatz+"traegt die Nummer" {
			t.Errorf("vergebene ISBN: Status %d %s", a.Status, a.Rumpf)
		}
		if got := p.isbnVon(id); got != "(keine)" {
			t.Errorf("nach zwei Ablehnungen trägt der Titel die ISBN %q", got)
		}

		if a := p.schicke(id, rumpf("978-3-00-047111-7")); a.Status != http.StatusOK || p.isbnVon(id) != ohneISBNFrei {
			t.Errorf("freie ISBN: Status %d %s, ISBN %q", a.Status, a.Rumpf, p.isbnVon(id))
		}
		// Dieselbe ISBN in anderer Schreibweise ist keine Änderung.
		if a := p.schicke(id, rumpf("978 3 00 047111 7")); a.Status != http.StatusOK || p.isbnVon(id) != ohneISBNFrei {
			t.Errorf("gleiche ISBN, andere Schreibweise: Status %d %s, ISBN %q", a.Status, a.Rumpf, p.isbnVon(id))
		}
	})

	t.Run("die ISBN lässt sich leeren", func(t *testing.T) {
		id := p.lege("verliert die Nummer", "Autorin", ohneISBNGeleert)
		a := p.schicke(id, map[string]any{"isbn": "", "title": ohneISBNVorsatz + "verliert die Nummer", "author": "Autorin"})
		if a.Status != http.StatusOK || p.isbnVon(id) != "(keine)" {
			t.Errorf("Status %d %s, ISBN %q", a.Status, a.Rumpf, p.isbnVon(id))
		}
	})
}

// Die Aufnahme verlangt den Titel, die ISBN nicht. Ohne ISBN fragt sie nach dem Titel, der
// ohne Nummer gleich heißt, und legt erst nach der Antwort „anderes Medium" an.
func TestTitelOhneISBN_Anlegen(t *testing.T) {
	p := neueOhneISBNProbe(t)

	t.Run("ein Spiel ohne ISBN und ohne Autor", func(t *testing.T) {
		rumpf := map[string]any{"isbn": "", "title": ohneISBNVorsatz + "Spiel", "author": "", "medientyp": "Spiel", "signatur": "Pro 4"}
		a := p.schicke("", rumpf)
		if a.Status != http.StatusCreated || a.Data.ID == "" {
			t.Fatalf("Status %d %s", a.Status, a.Rumpf)
		}
		if autor := p.sql(`SELECT COALESCE(autor, '') FROM buecher_titel WHERE id = $1`, a.Data.ID); autor != "" {
			t.Errorf("der Titel trägt den Autor %q, den niemand eingetragen hat", autor)
		}
		if got := p.isbnVon(a.Data.ID); got != "(keine)" {
			t.Errorf("ISBN %q, erwartet keine", got)
		}
		rumpf["signatur"] = "Pro 4b"
		if b := p.schicke(a.Data.ID, rumpf); b.Status != http.StatusOK || p.signaturVon(a.Data.ID) != "Pro 4b" {
			t.Errorf("danach ändern: Status %d %s", b.Status, b.Rumpf)
		}
	})

	t.Run("ohne Titel wird nichts angelegt", func(t *testing.T) {
		for _, isbn := range []string{"", ohneISBNOhneTitel} {
			a := p.schicke("", map[string]any{"isbn": isbn, "title": "  ", "author": "Autorin"})
			if a.Status != http.StatusBadRequest || !strings.Contains(a.Error, "titel") {
				t.Errorf("ISBN %q ohne Titel: Status %d %s", isbn, a.Status, a.Rumpf)
			}
		}
		if n := p.sql(`SELECT count(*)::text FROM buecher_titel WHERE titel = 'Unbekannter Titel' OR isbn = $1`, ohneISBNOhneTitel); n != "0" {
			t.Errorf("%s Titel sind trotzdem entstanden", n)
		}
	})

	t.Run("gleicher Titel und Autor ohne ISBN: die Tür nennt den vorhandenen und fragt", func(t *testing.T) {
		rumpf := map[string]any{"isbn": "", "title": ohneISBNVorsatz + "Heft", "author": "Redaktion", "signatur": "Pro 5"}
		erster := p.schicke("", rumpf)
		if erster.Status != http.StatusCreated {
			t.Fatalf("erstes Heft: Status %d %s", erster.Status, erster.Rumpf)
		}

		zweiter := p.schicke("", rumpf)
		if zweiter.Status != http.StatusConflict || !zweiter.GleicherTitel || zweiter.Vorhanden == nil ||
			zweiter.Vorhanden.ID != erster.Data.ID || !zweiter.Vorhanden.OhneExemplar {
			t.Fatalf("zweites Heft: Status %d %s", zweiter.Status, zweiter.Rumpf)
		}
		if !strings.Contains(zweiter.Error, ohneISBNVorsatz+"Heft") {
			t.Errorf("die Meldung nennt den vorhandenen Titel nicht: %q", zweiter.Error)
		}
		if n := p.zaehle("Heft"); n != "1" {
			t.Errorf("vor der Antwort stehen %s Titel im Katalog", n)
		}

		rumpf["anderesMedium"] = true
		if a := p.schicke("", rumpf); a.Status != http.StatusCreated || p.zaehle("Heft") != "2" {
			t.Errorf("anderes Medium: Status %d %s, Titel %s", a.Status, a.Rumpf, p.zaehle("Heft"))
		}
	})

	t.Run("eine andere Auflage ist ein anderes Buch und wird ohne Frage angelegt", func(t *testing.T) {
		rumpf := map[string]any{"isbn": "", "title": ohneISBNVorsatz + "Faust", "author": "Goethe"}
		if a := p.schicke("", rumpf); a.Status != http.StatusCreated {
			t.Fatalf("erste Ausgabe: Status %d %s", a.Status, a.Rumpf)
		}
		rumpf["auflage"] = "2. Aufl."
		if a := p.schicke("", rumpf); a.Status != http.StatusCreated {
			t.Errorf("zweite Auflage: Status %d %s", a.Status, a.Rumpf)
		}
	})

	t.Run("von zwei gleichnamigen nennt die Tür den mit Exemplar", func(t *testing.T) {
		rumpf := map[string]any{"isbn": "", "title": ohneISBNVorsatz + "Band", "author": "Verlag", "stock": 0}
		if a := p.schicke("", rumpf); a.Status != http.StatusCreated {
			t.Fatalf("Band ohne Exemplar: Status %d %s", a.Status, a.Rumpf)
		}
		rumpf["anderesMedium"], rumpf["stock"] = true, 1
		mitExemplar := p.schicke("", rumpf)
		if mitExemplar.Status != http.StatusCreated {
			t.Fatalf("Band mit Exemplar: Status %d %s", mitExemplar.Status, mitExemplar.Rumpf)
		}
		delete(rumpf, "anderesMedium")
		a := p.schicke("", rumpf)
		if a.Status != http.StatusConflict || a.Vorhanden == nil || a.Vorhanden.ID != mitExemplar.Data.ID || a.Vorhanden.OhneExemplar {
			t.Errorf("Status %d %s, erwartet den Titel %s mit Exemplar", a.Status, a.Rumpf, mitExemplar.Data.ID)
		}
	})

	t.Run("mit ISBN bleibt die Aufnahme, wie sie war", func(t *testing.T) {
		rumpf := map[string]any{"isbn": ohneISBNMitNummer, "title": ohneISBNVorsatz + "mit Nummer", "author": "Autorin"}
		if a := p.schicke("", map[string]any{"isbn": "12345", "title": ohneISBNVorsatz + "mit Nummer"}); a.Status != http.StatusBadRequest {
			t.Errorf("Nummer ohne ISBN-Form: Status %d %s", a.Status, a.Rumpf)
		}
		if a := p.schicke("", rumpf); a.Status != http.StatusCreated {
			t.Fatalf("Status %d %s", a.Status, a.Rumpf)
		}
		rumpf["anderesMedium"] = true
		a := p.schicke("", rumpf)
		if a.Status != http.StatusConflict || a.GleicherTitel || a.Vorhanden == nil {
			t.Errorf("vergebene ISBN, auch mit „anderes Medium“: Status %d %s", a.Status, a.Rumpf)
		}
	})
}
