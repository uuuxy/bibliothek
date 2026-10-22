package repository

import (
	"context"
	"errors"
	"regexp"
	"strconv"
	"testing"
	"time"

	"bibliothek/db"

	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	lmfNummerEins = "LMF-9783060158423-001"
	lmfNummerZwei = "LMF-9783060158423-002"
)

// legeNummernprobeAn legt einen Bearbeiter, einen Titel und Exemplare mit den Nummern an, alle
// mit gedrucktem Etikett wie nach dem Bestands-Import, und liefert den Bearbeiter und die
// Kennungen der Exemplare je Nummer.
func legeNummernprobeAn(t *testing.T, pool *pgxpool.Pool, nummern ...string) (string, map[string]string) {
	t.Helper()
	ctx := context.Background()
	resetInventurDaten(t, pool)
	var bearbeiter, titel string
	if err := pool.QueryRow(ctx, `
		INSERT INTO benutzer (vorname, nachname, email, rolle, aktiv)
		VALUES ('Nora', 'Nummer', 'nora.nummer@test.invalid', 'mitarbeiter', true) RETURNING id`).
		Scan(&bearbeiter); err != nil {
		t.Fatalf("Bearbeiter: %v", err)
	}
	if err := pool.QueryRow(ctx,
		`INSERT INTO buecher_titel (titel) VALUES ('Natur und Technik - Biologie 7/10') RETURNING id`).
		Scan(&titel); err != nil {
		t.Fatalf("Titel: %v", err)
	}
	ids := map[string]string{}
	for _, n := range nummern {
		var id string
		if err := pool.QueryRow(ctx, `
			INSERT INTO buecher_exemplare (titel_id, barcode_id, etikett_gedruckt)
			VALUES ($1, $2, true) RETURNING id`, titel, n).Scan(&id); err != nil {
			t.Fatalf("Exemplar %s: %v", n, err)
		}
		ids[n] = id
	}
	return bearbeiter, ids
}

// exemplarStand liest Nummer und Etikett-Vermerk eines Exemplars.
func exemplarStand(t *testing.T, pool *pgxpool.Pool, id string) (nummer string, gedruckt bool) {
	t.Helper()
	if err := pool.QueryRow(context.Background(),
		`SELECT barcode_id, etikett_gedruckt FROM buecher_exemplare WHERE id = $1`, id).
		Scan(&nummer, &gedruckt); err != nil {
		t.Fatalf("Exemplar %s lesen: %v", id, err)
	}
	return nummer, gedruckt
}

// protokollZuNummern zählt die Einträge zu einem Exemplar, die eine neue Nummer protokollieren.
func protokollZuNummern(t *testing.T, pool *pgxpool.Pool, id string) int {
	t.Helper()
	var n int
	if err := pool.QueryRow(context.Background(), `
		SELECT count(*) FROM audit_log
		WHERE datensatz_id = $1 AND details->>'action' = $2`, id, protokollAktionNeueNummer).
		Scan(&n); err != nil {
		t.Fatalf("Protokoll lesen: %v", err)
	}
	return n
}

// Eine LMF-Nummer weicht einer kurzen aus barcode_seq. Das Etikett steht danach aus, damit die
// Liste „Fehlende Etiketten" zu dem Exemplar führt, wenn der Druck abbricht, und das Protokoll
// nennt beide Nummern und die Person. Eine doppelt genannte Nummer bekommt eine neue Nummer.
func TestVergebeNeueNummern_KurzeNummerOffenesEtikettUndSpur(t *testing.T) {
	pool := pgTestPool(t)
	ctx := context.Background()
	bearbeiter, ids := legeNummernprobeAn(t, pool, lmfNummerEins, lmfNummerZwei, "B-10500")

	zuordnung, err := VergebeNeueNummern(ctx, pool, []string{lmfNummerEins, lmfNummerZwei, lmfNummerEins}, bearbeiter)
	if err != nil {
		t.Fatalf("VergebeNeueNummern: %v", err)
	}
	if len(zuordnung) != 2 {
		t.Fatalf("Zuordnung %v, erwartet zwei Nummern", zuordnung)
	}
	var stand int64
	if err := pool.QueryRow(ctx, `SELECT last_value FROM barcode_seq`).Scan(&stand); err != nil {
		t.Fatalf("barcode_seq lesen: %v", err)
	}
	kurz := regexp.MustCompile(`^B-([0-9]+)$`)
	gesehen := map[string]bool{"B-10500": true}
	for _, alt := range []string{lmfNummerEins, lmfNummerZwei} {
		neu := zuordnung[alt]
		teile := kurz.FindStringSubmatch(neu)
		if teile == nil {
			t.Fatalf("%s bekam %q, erwartet eine Nummer B-…", alt, neu)
		}
		// Aus der Sequenz und nicht neben ihr: Sonst zöge die nächste Bestellung dieselbe Nummer.
		zahl, err := strconv.ParseInt(teile[1], 10, 64)
		if err != nil {
			t.Fatalf("%s: Ziffern nicht lesbar: %v", neu, err)
		}
		if zahl > stand {
			t.Errorf("%s liegt über dem Stand von barcode_seq (%d)", neu, stand)
		}
		if gesehen[neu] {
			t.Errorf("%s ist doppelt vergeben", neu)
		}
		gesehen[neu] = true

		nummer, gedruckt := exemplarStand(t, pool, ids[alt])
		if nummer != neu || gedruckt {
			t.Errorf("Exemplar %s: Nummer %q, Etikett gedruckt %v — erwartet %q und offen", alt, nummer, gedruckt, neu)
		}

		var person, altImProtokoll, neuImProtokoll, kontext string
		if err := pool.QueryRow(ctx, `
			SELECT bearbeiter_id::text, details->>'barcode_id', details->>'neue_nummer', kontext
			FROM audit_log WHERE datensatz_id = $1 AND tabelle = 'buecher_exemplare' AND aktion = 'UPDATE'
			  AND details->>'action' = $2`, ids[alt], protokollAktionNeueNummer).
			Scan(&person, &altImProtokoll, &neuImProtokoll, &kontext); err != nil {
			t.Fatalf("Protokoll zu %s: %v", alt, err)
		}
		if person != bearbeiter || altImProtokoll != alt || neuImProtokoll != neu || kontext != kontextNeueNummer {
			t.Errorf("Protokoll zu %s: Person %q, alt %q, neu %q, Kontext %q", alt, person, altImProtokoll, neuImProtokoll, kontext)
		}
	}
	if nummer, gedruckt := exemplarStand(t, pool, ids["B-10500"]); nummer != "B-10500" || !gedruckt {
		t.Errorf("das nicht genannte Exemplar änderte sich: %q, gedruckt %v", nummer, gedruckt)
	}
}

// Alle oder keins: Trägt eine der Nummern kein Exemplar, ändert sich auch an den anderen nichts.
func TestVergebeNeueNummern_AllesOderNichts(t *testing.T) {
	pool := pgTestPool(t)
	ctx := context.Background()
	bearbeiter, ids := legeNummernprobeAn(t, pool, lmfNummerEins)

	_, err := VergebeNeueNummern(ctx, pool, []string{lmfNummerEins, "LMF-9783060158423-999"}, bearbeiter)
	if !errors.Is(err, ErrExemplarNichtGefunden) {
		t.Fatalf("Fehler %v, erwartet ErrExemplarNichtGefunden", err)
	}
	if nummer, gedruckt := exemplarStand(t, pool, ids[lmfNummerEins]); nummer != lmfNummerEins || !gedruckt {
		t.Errorf("Exemplar geändert: %q, gedruckt %v", nummer, gedruckt)
	}
	if n := protokollZuNummern(t, pool, ids[lmfNummerEins]); n != 0 {
		t.Errorf("%d Protokolleinträge, erwartet keinen", n)
	}
}

// Ohne Person keine neue Nummer: Der Eintrag im Protokoll nennt, wer sie vergab.
func TestVergebeNeueNummern_OhneBearbeiter(t *testing.T) {
	pool := pgTestPool(t)
	_, ids := legeNummernprobeAn(t, pool, lmfNummerEins)

	if _, err := VergebeNeueNummern(context.Background(), pool, []string{lmfNummerEins}, ""); !errors.Is(err, errNeueNummerOhneBearbeiter) {
		t.Fatalf("Fehler %v, erwartet errNeueNummerOhneBearbeiter", err)
	}
	if nummer, _ := exemplarStand(t, pool, ids[lmfNummerEins]); nummer != lmfNummerEins {
		t.Errorf("Exemplar geändert: %q", nummer)
	}
}

// Zwei Arbeitsplätze, eine Nummer: Der zweite Auftrag wartet an der Zeilensperre und findet
// danach kein Exemplar mehr mit der Nummer, statt die Nummer des ersten zu überschreiben. Der
// erste Auftrag ist hier eine Transaktion des Tests, die die Nummer ersetzt.
func TestVergebeNeueNummern_ZweiterAuftragFindetDieNummerNichtMehr(t *testing.T) {
	pool := pgTestPool(t)
	ctx := context.Background()
	bearbeiter, ids := legeNummernprobeAn(t, pool, lmfNummerEins)

	erster, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("Transaktion: %v", err)
	}
	defer db.Verwerfe(ctx, erster)
	if _, err := erster.Exec(ctx,
		`UPDATE buecher_exemplare SET barcode_id = 'B-10900' WHERE barcode_id = $1`, lmfNummerEins); err != nil {
		t.Fatalf("erster Auftrag: %v", err)
	}

	ergebnis := make(chan error, 1)
	go func() {
		_, err := VergebeNeueNummern(ctx, pool, []string{lmfNummerEins}, bearbeiter)
		ergebnis <- err
	}()
	wartetAnSperre(t, pool)
	if err := erster.Commit(ctx); err != nil {
		t.Fatalf("erster Auftrag: %v", err)
	}

	if err := <-ergebnis; !errors.Is(err, ErrExemplarNichtGefunden) {
		t.Fatalf("zweiter Auftrag: Fehler %v, erwartet ErrExemplarNichtGefunden", err)
	}
	if nummer, _ := exemplarStand(t, pool, ids[lmfNummerEins]); nummer != "B-10900" {
		t.Errorf("Nummer %q, erwartet die des ersten Auftrags B-10900", nummer)
	}
	if n := protokollZuNummern(t, pool, ids[lmfNummerEins]); n != 0 {
		t.Errorf("%d Protokolleinträge des zweiten Auftrags, erwartet keinen", n)
	}
}

// wartetAnSperre kehrt zurück, sobald eine Sitzung der Datenbank auf eine Sperre wartet.
func wartetAnSperre(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	ende := time.Now().Add(10 * time.Second)
	for time.Now().Before(ende) {
		var wartend int
		if err := pool.QueryRow(context.Background(), `
			SELECT count(*) FROM pg_stat_activity
			WHERE datname = current_database() AND wait_event_type = 'Lock'`).Scan(&wartend); err != nil {
			t.Fatalf("pg_stat_activity: %v", err)
		}
		if wartend > 0 {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatal("kein Auftrag wartet an der Zeilensperre")
}
