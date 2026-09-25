package repository

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Auflagen eines Schulbuchs (Migration 148, docs/OFFEN.md 4.18) gegen echtes Postgres: Die
// Regeln — nur Lernmittel, zwei Gruppen werden eine, ein Werk mit weniger als zwei Titeln
// fällt — stehen in repository/auflagen.go und wirken über mehrere Zeilen und zwei
// Tabellen. pgxmock sähe davon nichts.

func resetAuflagen(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	resetInventurDaten(t, pool)
	// CASCADE nimmt buecher_titel mit (werk_id verweist auf werke) — nach dem Reset oben
	// ist dort ohnehin nichts mehr.
	if _, err := pool.Exec(context.Background(), `TRUNCATE werke CASCADE`); err != nil {
		t.Fatalf("Werke leeren: %v", err)
	}
}

// seedAuflage legt einen Titel an; jahr 0 heißt ohne Erscheinungsjahr.
func seedAuflage(t *testing.T, pool *pgxpool.Pool, titel string, jahr int, lernmittel bool) string {
	t.Helper()
	var id string
	if err := pool.QueryRow(context.Background(),
		`INSERT INTO buecher_titel (titel, erscheinungsjahr, ist_lernmittel) VALUES ($1, NULLIF($2, 0), $3) RETURNING id`,
		titel, jahr, lernmittel).Scan(&id); err != nil {
		t.Fatalf("Titel %q anlegen: %v", titel, err)
	}
	return id
}

// werkVon liefert die werk_id eines Titels, "" für keine.
func werkVon(t *testing.T, pool *pgxpool.Pool, titelID string) string {
	t.Helper()
	var werk *string
	if err := pool.QueryRow(context.Background(),
		`SELECT werk_id::text FROM buecher_titel WHERE id = $1`, titelID).Scan(&werk); err != nil {
		t.Fatalf("werk_id lesen: %v", err)
	}
	if werk == nil {
		return ""
	}
	return *werk
}

func anzahlWerke(t *testing.T, pool *pgxpool.Pool) int {
	t.Helper()
	var n int
	if err := pool.QueryRow(context.Background(), `SELECT count(*) FROM werke`).Scan(&n); err != nil {
		t.Fatalf("werke zählen: %v", err)
	}
	return n
}

func ids(auflagen []Auflage) []string {
	out := make([]string, 0, len(auflagen))
	for _, a := range auflagen {
		out = append(out, a.ID)
	}
	return out
}

func TestFasseAuflagenZusammen_ZweiTitelWerdenEinBuch(t *testing.T) {
	pool := pgTestPool(t)
	resetAuflagen(t, pool)
	ctx := context.Background()
	alt := seedAuflage(t, pool, "Lambacher Schweizer 7", 2019, true)
	neu := seedAuflage(t, pool, "Lambacher Schweizer 7", 2023, true)

	auflagen, err := FasseAuflagenZusammen(ctx, pool, alt, neu)
	if err != nil {
		t.Fatalf("zusammenfassen: %v", err)
	}
	if got := ids(auflagen); len(got) != 2 || got[0] != neu || got[1] != alt {
		t.Errorf("Auflagen %v, erwartet [neu alt] = [%s %s] — die neueste zuerst", got, neu, alt)
	}
	if w := werkVon(t, pool, alt); w == "" || w != werkVon(t, pool, neu) {
		t.Errorf("werk_id alt %q, neu %q — beide müssen am selben Werk hängen", w, werkVon(t, pool, neu))
	}
	if n := anzahlWerke(t, pool); n != 1 {
		t.Errorf("%d Werke, erwartet 1", n)
	}

	// Ein zweites Mal ist kein Fehler und schreibt nichts.
	if _, err := FasseAuflagenZusammen(ctx, pool, neu, alt); err != nil {
		t.Fatalf("zweites Mal: %v", err)
	}
	if n := anzahlWerke(t, pool); n != 1 {
		t.Errorf("nach dem zweiten Mal %d Werke, erwartet 1", n)
	}
}

func TestFasseAuflagenZusammen_DritteAuflageKommtDazu(t *testing.T) {
	pool := pgTestPool(t)
	resetAuflagen(t, pool)
	ctx := context.Background()
	a := seedAuflage(t, pool, "Deutschbuch 7", 2015, true)
	b := seedAuflage(t, pool, "Deutschbuch 7", 2019, true)
	c := seedAuflage(t, pool, "Deutschbuch 7", 2024, true)

	if _, err := FasseAuflagenZusammen(ctx, pool, a, b); err != nil {
		t.Fatal(err)
	}
	// Von der neuen Auflage aus: Sie hat kein Werk, die alte schon.
	if _, err := FasseAuflagenZusammen(ctx, pool, c, a); err != nil {
		t.Fatalf("dritte Auflage: %v", err)
	}
	auflagen, err := AuflagenDesTitels(ctx, pool, b)
	if err != nil {
		t.Fatal(err)
	}
	if got := ids(auflagen); len(got) != 3 || got[0] != c || got[1] != b || got[2] != a {
		t.Errorf("Auflagen von b: %v, erwartet [c b a]", got)
	}
	if n := anzahlWerke(t, pool); n != 1 {
		t.Errorf("%d Werke, erwartet 1", n)
	}
}

func TestFasseAuflagenZusammen_ZweiGruppenWerdenEine(t *testing.T) {
	pool := pgTestPool(t)
	resetAuflagen(t, pool)
	ctx := context.Background()
	a := seedAuflage(t, pool, "Mathe live 5", 2012, true)
	b := seedAuflage(t, pool, "Mathe live 5", 2014, true)
	c := seedAuflage(t, pool, "Mathe live 5", 2018, true)
	d := seedAuflage(t, pool, "Mathe live 5", 2022, true)
	if _, err := FasseAuflagenZusammen(ctx, pool, a, b); err != nil {
		t.Fatal(err)
	}
	if _, err := FasseAuflagenZusammen(ctx, pool, c, d); err != nil {
		t.Fatal(err)
	}
	if n := anzahlWerke(t, pool); n != 2 {
		t.Fatalf("vorher %d Werke, erwartet 2", n)
	}

	auflagen, err := FasseAuflagenZusammen(ctx, pool, b, c)
	if err != nil {
		t.Fatalf("zwei Gruppen vereinen: %v", err)
	}
	if got := ids(auflagen); len(got) != 4 || got[0] != d || got[3] != a {
		t.Errorf("Auflagen %v, erwartet alle vier, d zuerst und a zuletzt", got)
	}
	werk := werkVon(t, pool, a)
	for _, id := range []string{b, c, d} {
		if w := werkVon(t, pool, id); w != werk {
			t.Errorf("Titel %s hängt an %q, erwartet %q — alle vier gehören zu einem Werk", id, w, werk)
		}
	}
	if n := anzahlWerke(t, pool); n != 1 {
		t.Errorf("%d Werke, erwartet 1 — das leere Werk muss gelöscht sein", n)
	}
}

func TestFasseAuflagenZusammen_NurLernmittel(t *testing.T) {
	pool := pgTestPool(t)
	resetAuflagen(t, pool)
	ctx := context.Background()
	schulbuch := seedAuflage(t, pool, "Tintenherz", 2003, true)
	roman := seedAuflage(t, pool, "Tintenherz", 2020, false)

	_, err := FasseAuflagenZusammen(ctx, pool, schulbuch, roman)
	if !errors.Is(err, ErrAuflageUngueltig) || !strings.Contains(err.Error(), "kein Lernmittel") {
		t.Fatalf("Fehler %v, erwartet ErrAuflageUngueltig mit „kein Lernmittel“", err)
	}
	if werkVon(t, pool, schulbuch) != "" || werkVon(t, pool, roman) != "" || anzahlWerke(t, pool) != 0 {
		t.Error("nach der Abweisung darf kein Werk entstanden und keine werk_id gesetzt sein")
	}
}

func TestFasseAuflagenZusammen_NichtMitSichSelbst(t *testing.T) {
	pool := pgTestPool(t)
	resetAuflagen(t, pool)
	ctx := context.Background()
	id := seedAuflage(t, pool, "Green Line 1", 2014, true)

	// Auch in Großbuchstaben ist es derselbe Titel — sonst liefe die Anweisung auf eine
	// Zeile statt zwei und endete als 500.
	for _, andere := range []string{id, strings.ToUpper(id)} {
		if _, err := FasseAuflagenZusammen(ctx, pool, id, andere); !errors.Is(err, ErrAuflageUngueltig) {
			t.Errorf("mit %s: Fehler %v, erwartet ErrAuflageUngueltig", andere, err)
		}
	}
	if anzahlWerke(t, pool) != 0 {
		t.Error("es darf kein Werk entstanden sein")
	}
}

func TestFasseAuflagenZusammen_UnbekannterTitel(t *testing.T) {
	pool := pgTestPool(t)
	resetAuflagen(t, pool)
	ctx := context.Background()
	id := seedAuflage(t, pool, "Green Line 2", 2015, true)
	const unbekannt = "00000000-0000-0000-0000-000000000148"

	if _, err := FasseAuflagenZusammen(ctx, pool, id, unbekannt); !errors.Is(err, ErrTitelNichtGefunden) {
		t.Errorf("Fehler %v, erwartet ErrTitelNichtGefunden", err)
	}
	if _, err := LoeseAuflage(ctx, pool, unbekannt); !errors.Is(err, ErrTitelNichtGefunden) {
		t.Errorf("lösen: Fehler %v, erwartet ErrTitelNichtGefunden", err)
	}
	if _, err := AuflagenDesTitels(ctx, pool, unbekannt); !errors.Is(err, ErrTitelNichtGefunden) {
		t.Errorf("lesen: Fehler %v, erwartet ErrTitelNichtGefunden", err)
	}
}

func TestLoeseAuflage_DieAnderenBleibenZusammen(t *testing.T) {
	pool := pgTestPool(t)
	resetAuflagen(t, pool)
	ctx := context.Background()
	a := seedAuflage(t, pool, "Biologie heute 7", 2010, true)
	b := seedAuflage(t, pool, "Biologie heute 7", 2016, true)
	c := seedAuflage(t, pool, "Biologie heute 7", 2021, true)
	if _, err := FasseAuflagenZusammen(ctx, pool, a, b); err != nil {
		t.Fatal(err)
	}
	if _, err := FasseAuflagenZusammen(ctx, pool, a, c); err != nil {
		t.Fatal(err)
	}

	auflagen, err := LoeseAuflage(ctx, pool, a)
	if err != nil {
		t.Fatalf("lösen: %v", err)
	}
	if got := ids(auflagen); len(got) != 1 || got[0] != a {
		t.Errorf("nach dem Lösen: %v, erwartet nur a", got)
	}
	if werkVon(t, pool, a) != "" {
		t.Error("a hängt noch an einem Werk")
	}
	if w := werkVon(t, pool, b); w == "" || w != werkVon(t, pool, c) {
		t.Errorf("b an %q, c an %q — die beiden anderen müssen zusammenbleiben", w, werkVon(t, pool, c))
	}
	if n := anzahlWerke(t, pool); n != 1 {
		t.Errorf("%d Werke, erwartet 1", n)
	}
}

func TestLoeseAuflage_DerLetzteWirdMitGeloest(t *testing.T) {
	pool := pgTestPool(t)
	resetAuflagen(t, pool)
	ctx := context.Background()
	a := seedAuflage(t, pool, "Physik 8", 2011, true)
	b := seedAuflage(t, pool, "Physik 8", 2020, true)
	if _, err := FasseAuflagenZusammen(ctx, pool, a, b); err != nil {
		t.Fatal(err)
	}

	if _, err := LoeseAuflage(ctx, pool, b); err != nil {
		t.Fatalf("lösen: %v", err)
	}
	if werkVon(t, pool, a) != "" || werkVon(t, pool, b) != "" {
		t.Errorf("werk_id a %q, b %q — ein Buch mit einer Auflage ist keine Gruppe", werkVon(t, pool, a), werkVon(t, pool, b))
	}
	if n := anzahlWerke(t, pool); n != 0 {
		t.Errorf("%d Werke, erwartet 0", n)
	}

	// Ein Titel ohne Werk bleibt, wie er ist.
	auflagen, err := LoeseAuflage(ctx, pool, a)
	if err != nil || len(auflagen) != 1 || auflagen[0].ID != a {
		t.Errorf("ohne Werk lösen: %v, %v — erwartet nur a und keinen Fehler", ids(auflagen), err)
	}
}

func TestAuflagenDesTitels_BestandWieImKatalog(t *testing.T) {
	pool := pgTestPool(t)
	resetAuflagen(t, pool)
	ctx := context.Background()
	id := seedAuflage(t, pool, "Geschichte und Geschehen 7", 2018, true)
	// Im Regal, gesperrt, bestellt und ausgesondert: gesamt 2, verfügbar 1, im Zulauf 1.
	if _, err := pool.Exec(ctx, `
		INSERT INTO buecher_exemplare (titel_id, barcode_id, ist_ausleihbar, ist_ausgesondert, aussonderung_grund, bestellstatus)
		VALUES ($1, 'B-148001', true,  false, NULL,      NULL),
		       ($1, 'B-148002', false, false, NULL,      NULL),
		       ($1, 'B-148003', false, false, NULL,      'bestellt'),
		       ($1, 'B-148004', true,  true,  'VERLUST', NULL)`, id); err != nil {
		t.Fatalf("Exemplare anlegen: %v", err)
	}

	auflagen, err := AuflagenDesTitels(ctx, pool, id)
	if err != nil {
		t.Fatal(err)
	}
	if len(auflagen) != 1 {
		t.Fatalf("%d Auflagen, erwartet 1", len(auflagen))
	}
	a := auflagen[0]
	if a.Gesamt != 2 || a.Verfuegbar != 1 || a.ImZulauf != 1 {
		t.Errorf("gesamt %d, verfügbar %d, im Zulauf %d — erwartet 2, 1, 1", a.Gesamt, a.Verfuegbar, a.ImZulauf)
	}
	if a.Titel != "Geschichte und Geschehen 7" || a.Erscheinungsjahr != 2018 || !a.IstLernmittel {
		t.Errorf("Kopf %+v", a)
	}
}
