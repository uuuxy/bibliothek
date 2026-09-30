package repository

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Schlagworte am Titel (Migration 138) gegen echtes Postgres: Die Regeln — vorhandene
// Schreibweise gewinnt, die Menge wird ersetzt, leer heißt leer — leben im SQL
// (lower()-Index, ON CONFLICT, DELETE … USING). pgxmock sähe davon nichts.

func resetSchlagworte(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	resetInventurDaten(t, pool)
	if _, err := pool.Exec(context.Background(), `TRUNCATE schlagworte CASCADE`); err != nil {
		t.Fatalf("Schlagworte leeren: %v", err)
	}
}

func seedSchlagwortTitel(t *testing.T, pool *pgxpool.Pool, titel string) string {
	t.Helper()
	var id string
	if err := pool.QueryRow(context.Background(),
		`INSERT INTO buecher_titel (titel) VALUES ($1) RETURNING id`, titel).Scan(&id); err != nil {
		t.Fatalf("Titel %q anlegen: %v", titel, err)
	}
	return id
}

func TestSetzeSchlagworte_VorhandeneSchreibweiseGewinnt(t *testing.T) {
	pool := pgTestPool(t)
	resetSchlagworte(t, pool)
	ctx := context.Background()

	erster := seedSchlagwortTitel(t, pool, "Tintenherz")
	zweiter := seedSchlagwortTitel(t, pool, "Woodwalkers")

	if _, err := SetzeSchlagworte(ctx, pool, erster, []string{"Fantasy"}); err != nil {
		t.Fatalf("erster Titel: %v", err)
	}
	gespeichert, err := SetzeSchlagworte(ctx, pool, zweiter, []string{"fantasy", "  Magische \t Tiere  "})
	if err != nil {
		t.Fatalf("zweiter Titel: %v", err)
	}

	if want := []string{"Fantasy", "Magische Tiere"}; !slices.Equal(gespeichert, want) {
		t.Errorf("gespeichert = %q, erwartet %q — „fantasy“ muss das vorhandene „Fantasy“ treffen", gespeichert, want)
	}
	var anzahl int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM schlagworte WHERE lower(wort) = 'fantasy'`).Scan(&anzahl); err != nil {
		t.Fatal(err)
	}
	if anzahl != 1 {
		t.Errorf("%d Zeilen für „fantasy“, erwartet 1 — eine zweite Schreibweise ist entstanden", anzahl)
	}
}

func TestSetzeSchlagworte_ErsetztDieMengeUndLeerEntferntAlle(t *testing.T) {
	pool := pgTestPool(t)
	resetSchlagworte(t, pool)
	ctx := context.Background()
	id := seedSchlagwortTitel(t, pool, "Die drei ???")

	schritte := []struct {
		eingabe []string
		want    []string
	}{
		{[]string{"Krimi", "Pferde"}, []string{"Krimi", "Pferde"}},
		{[]string{"Pferde", "Freundschaft"}, []string{"Freundschaft", "Pferde"}},
		{[]string{}, []string{}},
	}
	for i, s := range schritte {
		gespeichert, err := SetzeSchlagworte(ctx, pool, id, s.eingabe)
		if err != nil {
			t.Fatalf("Schritt %d: %v", i+1, err)
		}
		gelesen, err := SchlagworteDesTitels(ctx, pool, id)
		if err != nil {
			t.Fatalf("Schritt %d lesen: %v", i+1, err)
		}
		if !slices.Equal(gespeichert, s.want) || !slices.Equal(gelesen, s.want) {
			t.Errorf("Schritt %d: gespeichert %q, gelesen %q, erwartet %q", i+1, gespeichert, gelesen, s.want)
		}
		if gelesen == nil {
			t.Errorf("Schritt %d: nil statt leerer Liste — das Formular schickte „nichts gesagt“ zurück", i+1)
		}
	}
}

func TestSetzeSchlagworte_UnbekannterTitel(t *testing.T) {
	pool := pgTestPool(t)
	resetSchlagworte(t, pool)
	ctx := context.Background()
	const unbekannt = "00000000-0000-0000-0000-000000000138"

	for _, eingabe := range [][]string{{"Fantasy"}, {}} {
		if _, err := SetzeSchlagworte(ctx, pool, unbekannt, eingabe); !errors.Is(err, ErrTitelNichtGefunden) {
			t.Errorf("Eingabe %q: Fehler %v, erwartet ErrTitelNichtGefunden", eingabe, err)
		}
	}
	// Lesen ebenso: „keine Schlagworte" wäre für eine falsche Kennung ein Scheinerfolg.
	if woerter, err := SchlagworteDesTitels(ctx, pool, unbekannt); !errors.Is(err, ErrTitelNichtGefunden) {
		t.Errorf("Lesen: %q, Fehler %v, erwartet ErrTitelNichtGefunden", woerter, err)
	}
	var angelegt int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM schlagworte`).Scan(&angelegt); err != nil {
		t.Fatal(err)
	}
	if angelegt != 0 {
		t.Errorf("%d Wörter angelegt, obwohl der Titel fehlt", angelegt)
	}
}

func TestSetzeSchlagworte_Grenzen(t *testing.T) {
	pool := pgTestPool(t)
	resetSchlagworte(t, pool)
	ctx := context.Background()
	id := seedSchlagwortTitel(t, pool, "Grenzfall")

	zuViele := make([]string, SchlagworteJeTitelMax+1)
	for i := range zuViele {
		zuViele[i] = fmt.Sprintf("Wort %d", i)
	}
	if _, err := SetzeSchlagworte(ctx, pool, id, zuViele); !errors.Is(err, ErrSchlagwortUngueltig) {
		t.Errorf("%d Wörter: Fehler %v, erwartet ErrSchlagwortUngueltig", len(zuViele), err)
	}
	zuLang := strings.Repeat("ä", SchlagwortMaxZeichen+1)
	if _, err := SetzeSchlagworte(ctx, pool, id, []string{zuLang}); !errors.Is(err, ErrSchlagwortUngueltig) {
		t.Errorf("%d Zeichen: Fehler %v, erwartet ErrSchlagwortUngueltig", SchlagwortMaxZeichen+1, err)
	}
	// Genau an der Grenze: Die Datenbank zählt Zeichen, nicht Bytes — ein Umlaut ist eins.
	if _, err := SetzeSchlagworte(ctx, pool, id, []string{strings.Repeat("ä", SchlagwortMaxZeichen)}); err != nil {
		t.Errorf("%d Umlaute (an der Grenze): %v", SchlagwortMaxZeichen, err)
	}
	// So viele, wie Littera an einem Titel führt: 282 im Katalogisat vom Juni 2026 (bis zum
	// 30.09.2026 lag die Grenze bei 30, docs/OFFEN.md 4.20).
	littera := make([]string, 282)
	for i := range littera {
		littera[i] = fmt.Sprintf("Pflanze %d", i)
	}
	if got, err := SetzeSchlagworte(ctx, pool, id, littera); err != nil || len(got) != 282 {
		t.Errorf("282 Wörter wie in Littera: %d gespeichert, Fehler %v", len(got), err)
	}
}

func TestSchlagwortVorschlaege_HaeufigsteZuerstGekapptOhneWaisen(t *testing.T) {
	pool := pgTestPool(t)
	resetSchlagworte(t, pool)
	ctx := context.Background()

	haeufig := []string{
		seedSchlagwortTitel(t, pool, "A"), seedSchlagwortTitel(t, pool, "B"), seedSchlagwortTitel(t, pool, "C"),
	}
	for _, id := range haeufig {
		if _, err := SetzeSchlagworte(ctx, pool, id, []string{"Zauberei"}); err != nil {
			t.Fatal(err)
		}
	}
	// Mehr Wörter als die Kappung, je an einem Titel — direkt geschrieben, weil ein
	// Titel über den Schreibpfad höchstens SchlagworteJeTitelMax trägt.
	sammel := seedSchlagwortTitel(t, pool, "Sammelband")
	if _, err := pool.Exec(ctx, `
		WITH neu AS (
			INSERT INTO schlagworte (wort)
			SELECT 'Wort ' || lpad(g::text, 4, '0') FROM generate_series(1, $1) g
			RETURNING id)
		INSERT INTO titel_schlagworte (titel_id, schlagwort_id) SELECT $2, id FROM neu`,
		schlagwortVorschlaegeMax+10, sammel); err != nil {
		t.Fatal(err)
	}
	// Eine Waise: angelegt, von keinem Titel getragen.
	if _, err := pool.Exec(ctx, `INSERT INTO schlagworte (wort) VALUES ('Tippfehlr')`); err != nil {
		t.Fatal(err)
	}

	vorschlaege, err := SchlagwortVorschlaege(ctx, pool, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(vorschlaege) != schlagwortVorschlaegeMax {
		t.Fatalf("%d Vorschläge, erwartet die Kappung %d", len(vorschlaege), schlagwortVorschlaegeMax)
	}
	if erster := vorschlaege[0]; erster.Wort != "Zauberei" || erster.Titel != 3 {
		t.Errorf("erster Vorschlag %+v, erwartet Zauberei mit 3 Titeln — die Häufigsten zuerst", erster)
	}
	if zweiter := vorschlaege[1].Wort; zweiter != "Wort 0001" {
		t.Errorf("zweiter Vorschlag %q, erwartet „Wort 0001“ — bei gleicher Zahl alphabetisch", zweiter)
	}
	for _, v := range vorschlaege {
		if v.Wort == "Tippfehlr" {
			t.Error("ein Wort ohne Titel steht in den Vorschlägen")
		}
	}
}

// Wer tippt, bekommt Treffer über ALLE Wörter (Rasterdurchgang vom 30.09.2026). Bis dahin
// lud das Feld nur die 500 Häufigsten; mit den Littera-Wörtern (13.224 im Katalogisat vom
// Juni 2026) blieben 12.724 ungezeigt, und wer „Pilz" tippte, sah das vorhandene „Zwergpilze"
// nicht. Der Fall hier: ein seltenes Wort jenseits der Kappung, Anfangstreffer zuerst, dann
// die häufigsten; Verweise und Wörter ohne Titel nie; „%" ist Text; zerlegte Umlaute im
// Suchtext treffen; die Treffer einer Suche sind gekappt.
func TestSchlagwortVorschlaege_SucheUeberAlleWoerter(t *testing.T) {
	pool := pgTestPool(t)
	resetSchlagworte(t, pool)
	ctx := context.Background()

	sammel := seedSchlagwortTitel(t, pool, "Sammelband")
	if _, err := pool.Exec(ctx, `
		WITH neu AS (
			INSERT INTO schlagworte (wort)
			SELECT 'Wort ' || lpad(g::text, 4, '0') FROM generate_series(1, $1) g
			RETURNING id)
		INSERT INTO titel_schlagworte (titel_id, schlagwort_id) SELECT $2, id FROM neu`,
		schlagwortVorschlaegeMax+10, sammel); err != nil {
		t.Fatal(err)
	}
	for titel, woerter := range map[string][]string{
		"Pilzbuch":   {"Pilze", "Speisepilze", "Zwergpilze", "100% Wald", "V\u00f6gel"},
		"Kochbuch":   {"Speisepilze"},
		"Wanderbuch": {"Wald"},
	} {
		if _, err := SetzeSchlagworte(ctx, pool, seedSchlagwortTitel(t, pool, titel), woerter); err != nil {
			t.Fatal(err)
		}
	}
	// Ein Verweis auf „Pilze" und eine Waise: beide ohne Titel.
	if _, err := pool.Exec(ctx, `
		INSERT INTO schlagworte (wort, verweis_auf) SELECT 'Pilzkunde', id FROM schlagworte WHERE wort = 'Pilze';
		INSERT INTO schlagworte (wort) VALUES ('Pilzbefall')`); err != nil {
		t.Fatal(err)
	}

	woerter := func(suche string) []string {
		t.Helper()
		vorschlaege, err := SchlagwortVorschlaege(ctx, pool, suche)
		if err != nil {
			t.Fatalf("Suche %+q: %v", suche, err)
		}
		var aus []string
		for _, v := range vorschlaege {
			aus = append(aus, v.Wort)
		}
		return aus
	}

	if slices.Contains(woerter(""), "Zwergpilze") {
		t.Fatal("Vorbedingung: „Zwergpilze“ steht schon unter den 500 Häufigsten — der Fall prüft dann nichts")
	}
	for _, fall := range []struct {
		suche string
		want  []string
	}{
		{"pilz", []string{"Pilze", "Speisepilze", "Zwergpilze"}},
		{"  PILZE ", []string{"Pilze", "Speisepilze", "Zwergpilze"}},
		{"%", []string{"100% Wald"}},
		{"Vo\u0308g", []string{"V\u00f6gel"}},
		{"gibt es nicht", nil},
	} {
		if got := woerter(fall.suche); !slices.Equal(got, fall.want) {
			t.Errorf("Suche %+q: %+q, erwartet %+q", fall.suche, got, fall.want)
		}
	}
	if n := len(woerter("wort")); n != schlagwortVorschlaegeSucheMax {
		t.Errorf("Suche „wort“: %d Treffer, erwartet die Kappung %d", n, schlagwortVorschlaegeSucheMax)
	}
}
