package inventur

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sort"
	"strings"
	"testing"

	"bibliothek/internal/pgtest"

	"github.com/jackc/pgx/v5/pgxpool"
)

// PUT /api/books/{id} schreibt die Felder, die der Rumpf nennt (docs/OFFEN.md 5.5). Die Maske
// schickt die Felder, die sie geändert hat: Zwei Plätze mit demselben Titel überschreiben sich
// nicht, was nur einer von beiden angefasst hat.

const genanntVorsatz = "Genannte-Felder-Probe "

type genanntProbe struct {
	t       *testing.T
	pool    *pgxpool.Pool
	handler *APIHandler
	nr      int
}

// neueGenanntProbe stellt die Tür der Maske vor die Test-Datenbank und räumt Titel und Fächer
// der Probe vorher und nachher ab.
func neueGenanntProbe(t *testing.T) *genanntProbe {
	t.Helper()
	pool := pgtest.Pool(t)
	ctx := context.Background()
	loesche := func() {
		for _, sql := range []string{
			`DELETE FROM buecher_titel WHERE titel LIKE $1`,
			`DELETE FROM systematik_kategorien WHERE bezeichnung LIKE $1`,
		} {
			if _, err := pool.Exec(ctx, sql, genanntVorsatz+"%"); err != nil {
				t.Errorf("Aufräumen: %v", err)
			}
		}
	}
	loesche()
	t.Cleanup(loesche)

	durchlass := func(next http.Handler) http.Handler { return next }
	return &genanntProbe{t: t, pool: pool, handler: NewAPIHandler(APIHandlerConfig{
		Repo:                 NewBookRepository(pool),
		Metadaten:            offlineMetadatenClient(),
		RequireViewBooks:     durchlass,
		RequireEditBooks:     durchlass,
		RequireDeleteBooks:   durchlass,
		RequireAuthenticated: durchlass,
	})}
}

// lege legt einen Titel an, der in jeder Spalte der Tür einen eigenen Wert trägt: Schriebe die
// Tür ein Feld, das der Rumpf nicht nennt, stünde dort danach etwas anderes.
func (p *genanntProbe) lege() string {
	p.t.Helper()
	ctx := context.Background()
	p.nr++
	fach := genanntVorsatz + "Fach alt"
	if _, err := StelleFaecherSicher(ctx, p.pool, []string{fach}); err != nil {
		p.t.Fatalf("Fach anlegen: %v", err)
	}
	var id string
	if err := p.pool.QueryRow(ctx, `
		INSERT INTO buecher_titel (titel, untertitel, autor, isbn, verlag, erscheinungsjahr, cover_url,
			signatur, subject, track, ist_lernmittel, last_counted, medientyp,
			erweiterte_eigenschaften, auflage, listenpreis, jahrgang_von, jahrgang_bis, mehrjahresband)
		VALUES ($1, 'Alter Untertitel', 'Alter Autor', $2, 'Alter Verlag', 2019, '/covers/alt.webp',
			'Bio 7', $3, 'G', true, DATE '2026-01-15', 'Buch',
			'{"probe": "alt"}', '1. Aufl.', 12.50, 5, 10, false)
		RETURNING id::text`,
		fmt.Sprintf("%sTitel %d", genanntVorsatz, p.nr), fmt.Sprintf("97899977%05d", p.nr), fach).
		Scan(&id); err != nil {
		p.t.Fatalf("Titel anlegen: %v", err)
	}
	return id
}

// aendere schickt einen Rumpf an PUT /api/books/{id} und liefert Status und Antwort.
func (p *genanntProbe) aendere(id string, rumpf any) (int, string) {
	p.t.Helper()
	daten, err := json.Marshal(rumpf)
	if err != nil {
		p.t.Fatal(err)
	}
	rec := httptest.NewRecorder()
	p.handler.ServeHTTP(rec, httptest.NewRequest(http.MethodPut, "/api/books/"+id, bytes.NewReader(daten)))
	return rec.Code, rec.Body.String()
}

// zeile liest den Titel, wie er in der Tabelle steht, ohne den Zeitstempel des Speicherns und
// ohne die Suchspalte, die Postgres aus den Texten ableitet.
func (p *genanntProbe) zeile(id string) map[string]string {
	p.t.Helper()
	var roh []byte
	if err := p.pool.QueryRow(context.Background(), `
		SELECT (to_jsonb(t) - 'aktualisiert_am' - 'search_vector')::text FROM buecher_titel t WHERE id = $1`, id).
		Scan(&roh); err != nil {
		p.t.Fatalf("Titel lesen: %v", err)
	}
	var spalten map[string]json.RawMessage
	if err := json.Unmarshal(roh, &spalten); err != nil {
		p.t.Fatalf("Titel lesen: %v", err)
	}
	zeile := make(map[string]string, len(spalten))
	for name, wert := range spalten {
		zeile[name] = string(wert)
	}
	return zeile
}

// geaenderteSpalten nennt die Spalten, die sich zwischen zwei Lesungen unterscheiden.
func geaenderteSpalten(vorher, nachher map[string]string) []string {
	var spalten []string
	for name, wert := range nachher {
		if vorher[name] != wert {
			spalten = append(spalten, name)
		}
	}
	sort.Strings(spalten)
	return spalten
}

// Der Ablauf aus OFFEN 5.5: Platz 1 hat den Titel offen, Platz 2 berichtigt den Listenpreis,
// danach speichert Platz 1 einen anderen Verlag. Beide Eingaben bleiben stehen.
func TestTitelAendern_ZweiPlaetzeBehaltenIhreFelder(t *testing.T) {
	p := neueGenanntProbe(t)
	id := p.lege()

	if status, rumpf := p.aendere(id, map[string]any{"listenpreis": 21.5}); status != http.StatusOK {
		t.Fatalf("Platz 2: Status %d: %s", status, rumpf)
	}
	if status, rumpf := p.aendere(id, map[string]any{"verlag": "Neuer Verlag"}); status != http.StatusOK {
		t.Fatalf("Platz 1: Status %d: %s", status, rumpf)
	}
	zeile := p.zeile(id)
	if zeile["listenpreis"] != "21.50" {
		t.Errorf("Listenpreis von Platz 2: %s, erwartet 21.50", zeile["listenpreis"])
	}
	if zeile["verlag"] != `"Neuer Verlag"` {
		t.Errorf("Verlag von Platz 1: %s", zeile["verlag"])
	}
}

// Jedes Feld der Tür einzeln: Der Rumpf nennt eines, und in der Tabelle ändert sich genau
// seine Spalte. Ein Feld ohne Beispiel fällt hier auf, bevor es ungeprüft bleibt.
func TestTitelAendern_SchreibtNurDasGenannteFeld(t *testing.T) {
	p := neueGenanntProbe(t)
	beispiele := map[string]struct {
		wert   any
		spalte string
		soll   string
	}{
		"isbn":                    {"9783161484100", "isbn", `"9783161484100"`},
		"title":                   {genanntVorsatz + "neu", "titel", `"` + genanntVorsatz + `neu"`},
		"author":                  {"Neue Autorin", "autor", `"Neue Autorin"`},
		"coverUrl":                {"/covers/neu.webp", "cover_url", `"/covers/neu.webp"`},
		"subject":                 {genanntVorsatz + "Fach neu", "subject", `"` + genanntVorsatz + `Fach neu"`},
		"track":                   {"R", "track", `"R"`},
		"lastCounted":             {"2026-09-30", "last_counted", `"2026-09-30"`},
		"medientyp":               {"DVD", "medientyp", `"DVD"`},
		"erweiterteEigenschaften": {map[string]any{"probe": "neu"}, "erweiterte_eigenschaften", `{"probe": "neu"}`},
		"jahrgangVon":             {6, "jahrgang_von", "6"},
		"jahrgangBis":             {9, "jahrgang_bis", "9"},
		"untertitel":              {"Neuer Untertitel", "untertitel", `"Neuer Untertitel"`},
		"verlag":                  {"Neuer Verlag", "verlag", `"Neuer Verlag"`},
		"erscheinungsjahr":        {2021, "erscheinungsjahr", "2021"},
		"signatur":                {"Bio 8", "signatur", `"Bio 8"`},
		"istLernmittel":           {false, "ist_lernmittel", "false"},
		"auflage":                 {"2. Aufl.", "auflage", `"2. Aufl."`},
		"listenpreis":             {19.9, "listenpreis", "19.90"},
		"mehrjahresband":          {true, "mehrjahresband", "true"},
	}
	if len(beispiele) != len(titelFelder) {
		t.Errorf("%d Beispiele für %d Felder der Tür", len(beispiele), len(titelFelder))
	}

	for _, feld := range titelFelder {
		beispiel, ok := beispiele[feld.name]
		if !ok {
			t.Errorf("Feld %q hat kein Beispiel", feld.name)
			continue
		}
		t.Run(feld.name, func(t *testing.T) {
			id := p.lege()
			vorher := p.zeile(id)

			if status, rumpf := p.aendere(id, map[string]any{feld.name: beispiel.wert}); status != http.StatusOK {
				t.Fatalf("Status %d: %s", status, rumpf)
			}

			nachher := p.zeile(id)
			if nachher[beispiel.spalte] != beispiel.soll {
				t.Errorf("Spalte %s: %s, erwartet %s", beispiel.spalte, nachher[beispiel.spalte], beispiel.soll)
			}
			if spalten := geaenderteSpalten(vorher, nachher); len(spalten) != 1 || spalten[0] != beispiel.spalte {
				t.Errorf("geändert: %v, erwartet nur %s", spalten, beispiel.spalte)
			}
		})
	}
}

// „gradeLevel" schickt nur noch ein Browser-Fenster, das vor dem Wegfall der Klasse geladen
// wurde. Die Tür kennt das Feld nicht mehr: Sie ändert dafür nichts, scheitert nicht daran und
// speichert, was der Rumpf daneben nennt (docs/ARCHITEKTUR.md 11.5).
func TestTitelAendern_FruehereKlasseWirdNichtGelesen(t *testing.T) {
	p := neueGenanntProbe(t)
	id := p.lege()
	vorher := p.zeile(id)

	status, antwort := p.aendere(id, map[string]any{"gradeLevel": 8, "verlag": "Neuer Verlag"})
	if status != http.StatusOK {
		t.Fatalf("Status %d, erwartet 200: %s", status, antwort)
	}
	if spalten := geaenderteSpalten(vorher, p.zeile(id)); len(spalten) != 1 || spalten[0] != "verlag" {
		t.Errorf("geändert wurden %v, erwartet nur verlag", spalten)
	}
}

// Was der Rumpf nicht nennt, wird weder geschrieben noch geprüft; was er nennt, wird am Stand
// nach der Änderung geprüft.
func TestTitelAendern_PrueftWasSichAendert(t *testing.T) {
	p := neueGenanntProbe(t)
	id := p.lege()
	start := p.zeile(id)

	// Ein Rumpf ohne Felder ändert nichts.
	if status, rumpf := p.aendere(id, map[string]any{}); status != http.StatusOK {
		t.Fatalf("leerer Rumpf: Status %d: %s", status, rumpf)
	}
	if spalten := geaenderteSpalten(start, p.zeile(id)); len(spalten) != 0 {
		t.Errorf("leerer Rumpf änderte %v", spalten)
	}

	// Der Name eines Felds zählt ohne Groß- und Kleinschreibung, wie beim Dekodieren.
	if status, rumpf := p.aendere(id, map[string]any{"Verlag": "Großer Verlag"}); status != http.StatusOK {
		t.Fatalf("Verlag: Status %d: %s", status, rumpf)
	}
	if verlag := p.zeile(id)["verlag"]; verlag != `"Großer Verlag"` {
		t.Errorf("Verlag unter dem Namen „Verlag“: %s", verlag)
	}

	if status, rumpf := p.aendere(id, map[string]any{"mehrjahresband": true}); status != http.StatusOK {
		t.Fatalf("Mehrjahresband an: Status %d: %s", status, rumpf)
	}
	band := p.zeile(id)
	abgelehnt := []struct {
		name  string
		rumpf map[string]any
		text  string
	}{
		{"ein genannter Titel ist nicht leer", map[string]any{"title": "  "}, "titel darf nicht leer sein"},
		{"ein genannter Autor wird nicht geleert", map[string]any{"author": ""}, "autor darf nicht leer sein"},
		{"Mehrjahresband nur am Lernmittel", map[string]any{"istLernmittel": false}, "mehrjahresband"},
		{"Mehrjahresband nur über mehr als einen Jahrgang", map[string]any{"jahrgangBis": 5}, "mehrjahresband"},
	}
	for _, f := range abgelehnt {
		status, rumpf := p.aendere(id, f.rumpf)
		if status != http.StatusBadRequest || !strings.Contains(rumpf, f.text) {
			t.Errorf("%s: Status %d: %s — erwartet 400 mit %q", f.name, status, rumpf, f.text)
		}
	}
	if spalten := geaenderteSpalten(band, p.zeile(id)); len(spalten) != 0 {
		t.Errorf("abgelehnte Änderungen schrieben %v", spalten)
	}

	// Ein anderes Feld bleibt speicherbar, und beide Felder des Mehrjahresbands zusammen auch.
	if status, rumpf := p.aendere(id, map[string]any{"untertitel": "Band 7 bis 10"}); status != http.StatusOK {
		t.Errorf("Untertitel am Mehrjahresband: Status %d: %s", status, rumpf)
	}
	if status, rumpf := p.aendere(id, map[string]any{"istLernmittel": false, "mehrjahresband": false}); status != http.StatusOK {
		t.Errorf("Lernmittel und Mehrjahresband aus: Status %d: %s", status, rumpf)
	}
}
