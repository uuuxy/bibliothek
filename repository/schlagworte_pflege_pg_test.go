package repository

import (
	"context"
	"errors"
	"slices"
	"testing"
	"time"

	"bibliothek/db"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Pflege der Schlagworte (Migration 143, docs/OFFEN.md 4.20 Stufe 1) gegen echtes Postgres:
// Die Regeln der Verweise stehen doppelt — in repository/schlagworte_pflege.go und als
// Trigger —, und beides sieht nur ein echter Lauf.

// pflegeStand legt Titel mit Schlagworten an und liefert die Kennungen der Wörter.
func pflegeStand(t *testing.T, pool *pgxpool.Pool, titelWoerter map[string][]string) map[string]string {
	t.Helper()
	ctx := context.Background()
	for titel, woerter := range titelWoerter {
		if _, err := SetzeSchlagworte(ctx, pool, seedSchlagwortTitel(t, pool, titel), woerter); err != nil {
			t.Fatalf("%s: %v", titel, err)
		}
	}
	ids := map[string]string{}
	rows, err := pool.Query(ctx, `SELECT wort, id::text FROM schlagworte`)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	for rows.Next() {
		var w, id string
		if err := rows.Scan(&w, &id); err != nil {
			t.Fatal(err)
		}
		ids[w] = id
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	return ids
}

func woerterAm(t *testing.T, pool *pgxpool.Pool, titel string) []string {
	t.Helper()
	var id string
	if err := pool.QueryRow(context.Background(), `SELECT id FROM buecher_titel WHERE titel = $1`, titel).Scan(&id); err != nil {
		t.Fatalf("Titel %q: %v", titel, err)
	}
	w, err := SchlagworteDesTitels(context.Background(), pool, id)
	if err != nil {
		t.Fatal(err)
	}
	return w
}

func pflegeZeile(t *testing.T, pool *pgxpool.Pool, wort string) SchlagwortPflegeZeile {
	t.Helper()
	liste, err := SchlagworteZurPflege(context.Background(), pool)
	if err != nil {
		t.Fatal(err)
	}
	for _, z := range liste.Zeilen {
		if z.Wort == wort {
			return z
		}
	}
	t.Fatalf("„%s“ fehlt in der Pflegeliste", wort)
	return SchlagwortPflegeZeile{}
}

func TestSchlagwortPflege_ZusammenfuehrenMachtDasAlteWortZumVerweis(t *testing.T) {
	pool := pgTestPool(t)
	resetSchlagworte(t, pool)
	ctx := context.Background()
	ids := pflegeStand(t, pool, map[string][]string{
		"Tintenherz":    {"Tierfantasy", "Abenteuer"},
		"Woodwalkers":   {"Tierfantasy", "Fantasy"},
		"Drachenreiter": {"Fantasy"},
	})
	if err := SetzeSchlagwortFilter(ctx, pool, ids["Tierfantasy"], true); err != nil {
		t.Fatal(err)
	}

	titel, err := FuehreSchlagworteZusammen(ctx, pool, ids["Tierfantasy"], ids["Fantasy"], true)
	if err != nil {
		t.Fatalf("zusammenführen: %v", err)
	}
	if titel != 2 {
		t.Errorf("gemeldet %d Titel, „Tierfantasy“ trug 2", titel)
	}
	// Woodwalkers trug beide — danach genau einmal „Fantasy".
	for titelName, want := range map[string][]string{
		"Tintenherz":  {"Abenteuer", "Fantasy"},
		"Woodwalkers": {"Fantasy"},
	} {
		if got := woerterAm(t, pool, titelName); !slices.Equal(got, want) {
			t.Errorf("%s: %q, want %q", titelName, got, want)
		}
	}
	alt := pflegeZeile(t, pool, "Tierfantasy")
	if alt.VerweisAuf != "Fantasy" || alt.Titel != 0 || alt.IstFilter {
		t.Errorf("„Tierfantasy“ nach dem Zusammenführen: %+v — erwartet Verweis auf Fantasy, 0 Titel, kein Filter", alt)
	}
	ziel := pflegeZeile(t, pool, "Fantasy")
	if ziel.Titel != 3 || !ziel.IstFilter || !slices.Equal(ziel.Verweise, []string{"Tierfantasy"}) {
		t.Errorf("„Fantasy“ nach dem Zusammenführen: %+v — erwartet 3 Titel, Filter übernommen, Verweis Tierfantasy", ziel)
	}

	// Wer den Verweis tippt, bekommt das Ziel.
	neu := seedSchlagwortTitel(t, pool, "Warrior Cats")
	got, err := SetzeSchlagworte(ctx, pool, neu, []string{"tierfantasy", "Fantasy"})
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(got, []string{"Fantasy"}) {
		t.Errorf("getippter Verweis: %q, want [Fantasy]", got)
	}

	if _, err := FuehreSchlagworteZusammen(ctx, pool, ids["Fantasy"], ids["Tierfantasy"], true); !errors.Is(err, ErrSchlagwortRegel) {
		t.Errorf("mit dem eigenen Verweis zusammenführen: %v, want ErrSchlagwortRegel", err)
	}
}

func TestSchlagwortPflege_UmbenennenLoeschenVerweisFilter(t *testing.T) {
	pool := pgTestPool(t)
	resetSchlagworte(t, pool)
	ctx := context.Background()
	ids := pflegeStand(t, pool, map[string][]string{
		"Die Welle": {"gewalt", "Schule"},
		"Krabat":    {"Magie"},
	})

	if neu, verweis, err := BenenneSchlagwortUm(ctx, pool, ids["gewalt"], "  Gewalt ", true); err != nil || neu != "Gewalt" || verweis {
		t.Errorf("Schreibweise ändern: %q, verweis=%v, %v — want Gewalt ohne Verweis (dasselbe Wort)", neu, verweis, err)
	}
	if _, _, err := BenenneSchlagwortUm(ctx, pool, ids["Schule"], "magie", true); !errors.Is(err, ErrSchlagwortGibtEs) {
		t.Errorf("umbenennen auf ein vorhandenes Wort: %v, want ErrSchlagwortGibtEs", err)
	}
	if _, _, err := BenenneSchlagwortUm(ctx, pool, "00000000-0000-0000-0000-000000000000", "X", true); !errors.Is(err, ErrSchlagwortNichtGefunden) {
		t.Errorf("unbekanntes Wort: %v, want ErrSchlagwortNichtGefunden", err)
	}

	// Ein neuer Verweis entsteht ohne Titel; als Filter lässt er sich nicht markieren.
	if err := SetzeSchlagwortVerweis(ctx, pool, "Zauberei", ids["Magie"]); err != nil {
		t.Fatalf("Verweis anlegen: %v", err)
	}
	verweis := pflegeZeile(t, pool, "Zauberei")
	if verweis.VerweisAuf != "Magie" || verweis.Titel != 0 {
		t.Errorf("Verweis: %+v", verweis)
	}
	if err := SetzeSchlagwortFilter(ctx, pool, verweis.ID, true); !errors.Is(err, ErrSchlagwortRegel) {
		t.Errorf("Verweis als Filter: %v, want ErrSchlagwortRegel", err)
	}

	// Verweis auf eine Schreibweise, die Titel trägt: das ist ein Zusammenführen.
	if err := SetzeSchlagwortVerweis(ctx, pool, "Schule", ids["gewalt"]); err != nil {
		t.Fatalf("Verweis auf vorhandenes Wort: %v", err)
	}
	if got := woerterAm(t, pool, "Die Welle"); !slices.Equal(got, []string{"Gewalt"}) {
		t.Errorf("Die Welle nach dem Verweis: %q, want [Gewalt]", got)
	}

	// Löschen: Titel verlieren das Wort, Verweise darauf fallen mit.
	geloescht, err := LoescheSchlagworte(ctx, pool, []string{ids["Magie"]})
	if err != nil || geloescht != (SchlagwortLoeschung{Woerter: 1, Titel: 1, Verweise: 1}) {
		t.Errorf("löschen: %+v err=%v, want 1 Wort, 1 Titel, 1 Verweis", geloescht, err)
	}
	if got := woerterAm(t, pool, "Krabat"); len(got) != 0 {
		t.Errorf("Krabat nach dem Löschen: %q", got)
	}
	var rest int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM schlagworte WHERE wort IN ('Magie', 'Zauberei')`).Scan(&rest); err != nil {
		t.Fatal(err)
	}
	if rest != 0 {
		t.Errorf("%d Zeilen von Magie/Zauberei übrig, want 0", rest)
	}
}

// Die Regeln stehen auch in der Datenbank — eine Tür, die an der Pflege vorbei schreibt,
// kommt nicht durch.
func TestSchlagwortPflege_DatenbankHaeltDieRegeln(t *testing.T) {
	pool := pgTestPool(t)
	resetSchlagworte(t, pool)
	ctx := context.Background()
	ids := pflegeStand(t, pool, map[string][]string{"Momo": {"Zeit", "Freundschaft"}})
	if err := SetzeSchlagwortVerweis(ctx, pool, "Uhren", ids["Zeit"]); err != nil {
		t.Fatal(err)
	}
	uhren := pflegeZeile(t, pool, "Uhren").ID
	var momo string
	if err := pool.QueryRow(ctx, `SELECT id FROM buecher_titel WHERE titel = 'Momo'`).Scan(&momo); err != nil {
		t.Fatal(err)
	}

	for name, sql := range map[string]string{
		"Titel am Verweis":             `INSERT INTO titel_schlagworte (titel_id, schlagwort_id) VALUES ('` + momo + `', '` + uhren + `')`,
		"Kette":                        `UPDATE schlagworte SET verweis_auf = '` + uhren + `' WHERE id = '` + ids["Freundschaft"] + `'`,
		"Wort mit Titeln wird Verweis": `UPDATE schlagworte SET verweis_auf = '` + ids["Zeit"] + `' WHERE id = '` + ids["Freundschaft"] + `'`,
		"Verweis auf sich selbst":      `UPDATE schlagworte SET verweis_auf = id WHERE id = '` + ids["Zeit"] + `'`,
		"Verweis als Filter":           `UPDATE schlagworte SET ist_filter = true WHERE id = '` + uhren + `'`,
	} {
		if _, err := pool.Exec(ctx, sql); err == nil {
			t.Errorf("%s: die Datenbank hat es angenommen", name)
		}
	}
}

// titelAnVerweisen zählt Titel, die an einem Verweis hängen — das verbietet Migration 143.
func titelAnVerweisen(t *testing.T, pool *pgxpool.Pool) int {
	t.Helper()
	var n int
	if err := pool.QueryRow(context.Background(), `
		SELECT count(*)::int FROM titel_schlagworte ts
		JOIN schlagworte s ON s.id = ts.schlagwort_id
		WHERE s.verweis_auf IS NOT NULL`).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

// ueberschneidung wartet, bis der nebenläufige Schreiber entweder fertig ist oder von der
// Transaktion `sperrer` blockiert wird — erst danach darf der Test sie committen. Beides kann
// richtig sein: Vor Migration 144 kam ein Schreiber teils sofort durch (die Fremdschlüssel-
// Prüfung verträgt sich mit einer offenen Änderung), seit 144 wartet er. Gefragt wird nach
// genau diesem Sperrer (pg_blocking_pids), nicht nach irgendeiner wartenden Sitzung:
// internal/pgtest hält eine Sitzungssperre, auf die in der vollen Suite andere Testpakete
// warten. `fertig` muss gepuffert sein; sein Wert bleibt für den Test im Kanal.
func ueberschneidung(t *testing.T, pool *pgxpool.Pool, sperrer pgx.Tx, fertig chan error) {
	t.Helper()
	ctx := context.Background()
	var pid int
	if err := sperrer.QueryRow(ctx, `SELECT pg_backend_pid()`).Scan(&pid); err != nil {
		t.Fatal(err)
	}
	for range 100 {
		if len(fertig) > 0 {
			return
		}
		var n int
		if err := pool.QueryRow(ctx, `
			SELECT count(*)::int FROM pg_stat_activity WHERE $1 = ANY (pg_blocking_pids(pid))`, pid).Scan(&n); err != nil {
			t.Fatal(err)
		}
		if n > 0 {
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatal("der nebenläufige Schreiber ist weder fertig, noch wartet er auf die Transaktion des Tests")
}

// Rasterdurchgang 23.09.2026, V1: Das Zusammenführen hat „Detektiv" gesperrt, aber noch nicht
// zum Verweis gemacht, als das Buchformular einen Titel daran hängt. Vor Migration 144 kamen
// beide Trigger durch — jeder sah die offene Änderung des anderen nicht —, und der Titel hing
// danach an einem Verweis. Seit 144 wartet der Trigger am Titel, bis das Zusammenführen fertig
// ist, und sieht dann den Verweis: Das Speichern bekommt die Ausnahme, ein zweites Speichern
// löst den Verweis auf. Die Pflege läuft Schritt für Schritt in einer Transaktion des Tests,
// damit die Überschneidung sicher entsteht.
func TestSchlagwortPflege_GleichzeitigKeinTitelAmVerweis(t *testing.T) {
	pool := pgTestPool(t)
	resetSchlagworte(t, pool)
	ctx := context.Background()
	ids := pflegeStand(t, pool, map[string][]string{"Emil": {"Krimi"}, "Kalle": {"Detektiv"}})
	neu := seedSchlagwortTitel(t, pool, "Die drei ???")

	pflege, err := pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer db.SafeRollback(ctx, pflege)
	if _, err := sperreSchlagwort(ctx, pflege, ids["Detektiv"]); err != nil {
		t.Fatal(err)
	}
	fertig := make(chan error, 1)
	go func() {
		_, err := SetzeSchlagworte(ctx, pool, neu, []string{"Detektiv"})
		fertig <- err
	}()
	ueberschneidung(t, pool, pflege, fertig)
	if _, err := fuehreZusammenIn(ctx, pflege, ids["Detektiv"], ids["Krimi"], true); err != nil {
		t.Fatalf("zusammenführen: %v", err)
	}
	if err := pflege.Commit(ctx); err != nil {
		t.Fatal(err)
	}
	if err := <-fertig; err == nil {
		t.Error("das Speichern kam durch — erwartet war die Ausnahme des Triggers")
	}
	if n := titelAnVerweisen(t, pool); n != 0 {
		t.Errorf("%d Titel hängen an einem Verweis; „Die drei ???“ trägt %v", n, woerterAm(t, pool, "Die drei ???"))
	}

	if _, err := SetzeSchlagworte(ctx, pool, neu, []string{"Detektiv"}); err != nil {
		t.Fatalf("zweites Speichern: %v", err)
	}
	if got := woerterAm(t, pool, "Die drei ???"); !slices.Equal(got, []string{"Krimi"}) {
		t.Errorf("nach dem zweiten Speichern trägt „Die drei ???“ %v, erwartet [Krimi]", got)
	}
}

// Rasterdurchgang 23.09.2026, V3: Ein Verweis „Tierfantasy" → „Fantasy" entsteht, während
// „Fantasy" selbst zum Verweis wird. Vor Migration 144 kamen beide durch — eine Kette. Rohes
// SQL, weil die Go-Tür ihr Ziel ohnehin sperrt: geprüft wird die Zusage der Datenbank.
func TestSchlagwortPflege_GleichzeitigKeineKette(t *testing.T) {
	pool := pgTestPool(t)
	resetSchlagworte(t, pool)
	ctx := context.Background()
	ids := pflegeStand(t, pool, map[string][]string{"Emil": {"Fantasy", "Abenteuer"}})
	if _, err := pool.Exec(ctx, `DELETE FROM titel_schlagworte`); err != nil {
		t.Fatal(err)
	}
	b, err := pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer db.SafeRollback(ctx, b)
	if _, err := b.Exec(ctx, `UPDATE schlagworte SET verweis_auf = $2 WHERE id = $1`,
		ids["Fantasy"], ids["Abenteuer"]); err != nil {
		t.Fatal(err)
	}
	fertig := make(chan error, 1)
	go func() {
		_, err := pool.Exec(ctx, `INSERT INTO schlagworte (wort, verweis_auf) VALUES ('Tierfantasy', $1)`, ids["Fantasy"])
		fertig <- err
	}()
	ueberschneidung(t, pool, b, fertig)
	if err := b.Commit(ctx); err != nil {
		t.Fatal(err)
	}
	if err := <-fertig; err == nil {
		t.Error("der Verweis auf einen Verweis kam durch — erwartet war die Ausnahme des Triggers")
	}
	var ketten int
	if err := pool.QueryRow(ctx, `SELECT count(*)::int FROM schlagworte v
		JOIN schlagworte z ON z.id = v.verweis_auf WHERE z.verweis_auf IS NOT NULL`).Scan(&ketten); err != nil {
		t.Fatal(err)
	}
	if ketten != 0 {
		t.Errorf("%d Verweis(e) zeigen auf einen Verweis", ketten)
	}
}
