package api

import (
	"context"
	"os"
	"testing"

	"bibliothek/db"
	"bibliothek/repository"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Der Ausweis-Generator steht seit Migration 136 in der Datenbank (ausweis_nummer_start),
// weil der Trigger konto_hat_leserzeile ihn braucht und ein Nummernkreis EINEN Zähler hat.
// Die Regeln sind die, die bis dahin GetNextSequence in Go absicherte — hier an der
// Funktion, die jetzt alle rufen.

// naechsteAusweisnummer zieht die nächste Nummer in einer eigenen, zurückgerollten Transaktion.
func naechsteAusweisnummer(t *testing.T, pool *pgxpool.Pool) int {
	t.Helper()
	ctx := context.Background()
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer db.SafeRollback(ctx, tx)
	n, err := repository.NewSequenceRepository(tx).NaechsteAusweisnummer(ctx)
	if err != nil {
		t.Fatalf("NaechsteAusweisnummer: %v", err)
	}
	return n
}

// Numerisch, nicht lexikografisch: Lexikografisch gilt 'A-99999' > 'A-100000' (die '9'
// schlägt die '1'); eine Sortierung als Text lieferte dauerhaft 99999 als Maximum und
// danach endlos 'A-100000' in den eindeutigen Index.
func TestAusweisnummer_NumerischNichtLexikografisch(t *testing.T) {
	pool := pgTestPool(t)
	resetBestandsdaten(t, pool)
	schueler(t, pool, "A-99999")
	schueler(t, pool, "A-100000")
	if got := naechsteAusweisnummer(t, pool); got != 100001 {
		t.Errorf("nächste Nummer %d, erwartet 100001 (Text-Sortierung hätte 100000 geliefert)", got)
	}
}

// Ohne A-Nummer im Bestand beginnt der Kreis bei 10001 — auch bei leerer Tabelle. Nummern
// anderer Vorsilben (die alten S-Ausweise) zählen nicht mit.
func TestAusweisnummer_LeererBestandFallback(t *testing.T) {
	pool := pgTestPool(t)
	resetBestandsdaten(t, pool)
	if got := naechsteAusweisnummer(t, pool); got != 10001 {
		t.Errorf("leere Tabelle: %d, erwartet 10001", got)
	}
	schueler(t, pool, "S-20000")
	if got := naechsteAusweisnummer(t, pool); got != 10001 {
		t.Errorf("nur eine S-Nummer im Bestand: %d, erwartet 10001", got)
	}
}

// Ein verrutschter Scan legte eine Nummer an, die keine bigint mehr ist — und ließ danach
// JEDE automatische Vergabe mit "value out of range for type bigint" scheitern. Zu lange
// Nummern werden übergangen.
func TestAusweisnummer_UeberlangeNummerBlockiertNicht(t *testing.T) {
	pool := pgTestPool(t)
	resetBestandsdaten(t, pool)
	schueler(t, pool, "A-10005")
	schueler(t, pool, "A-999999999999999999999") // 21 Ziffern, sprengt bigint
	if got := naechsteAusweisnummer(t, pool); got != 10006 {
		t.Errorf("nächste Nummer %d, erwartet 10006", got)
	}
}

// Go und Datenbank setzen dieselbe gedruckte Form: Der Trigger schreibt mit
// ausweisnummer(n), die Handanlage und der LUSD-Import mit AusweisNummer(n).
func TestAusweisnummer_FormGleichInGoUndSQL(t *testing.T) {
	pool := pgTestPool(t)
	for _, n := range []int{7, 10001, 99999, 100000, 1234567} {
		var sql string
		if err := pool.QueryRow(context.Background(), `SELECT ausweisnummer($1)`, n).Scan(&sql); err != nil {
			t.Fatal(err)
		}
		if sql != AusweisNummer(n) {
			t.Errorf("n=%d: Datenbank %q, Go %q", n, sql, AusweisNummer(n))
		}
	}
}

// Ein Konto bekommt beim Anlegen eine Ausweisnummer — aus demselben Kreis wie ein Schüler
// (docs/OFFEN.md 5.16). Bis Migration 136 legte konto_hat_leserzeile die Leserzeile ohne
// Nummer an, und der Ausweisdruck lieferte eine leere Zeile.
func TestAusweisnummer_KontoBekommtEineBeimAnlegen(t *testing.T) {
	pool := pgTestPool(t)
	resetBestandsdaten(t, pool)
	ctx := context.Background()
	schueler(t, pool, "A-10041")

	// Zwei Anweisungen: Die Leserzeile, die der Trigger anlegt, sähe eine äußere Abfrage
	// derselben Anweisung nicht (schreibende CTE, gleicher Schnappschuss).
	var leserID string
	if err := pool.QueryRow(ctx, `
		INSERT INTO benutzer (email, vorname, nachname, rolle)
		VALUES ('ausweis-konto@example.org', 'Konto', 'Probe', 'kollegium') RETURNING leser_id`).Scan(&leserID); err != nil {
		t.Fatalf("Konto anlegen: %v", err)
	}
	var nummer *string
	if err := pool.QueryRow(ctx, `SELECT barcode_id FROM leser WHERE id = $1`, leserID).Scan(&nummer); err != nil {
		t.Fatal(err)
	}
	if nummer == nil || *nummer != "A-10042" {
		t.Errorf("Ausweis des neuen Kontos %s, erwartet A-10042", nummerOderNull(nummer))
	}
}

// Eine offene Zugangsanfrage (inaktiv) bekommt keine Nummer — sonst bliebe ihre Leserzeile
// nach einer Ablehnung als Waise stehen (repository.loescheUnberuehrteLeserzeile). Die
// Freischaltung vergibt sie, auch wenn derselbe Vorgang danach die Leserzeile mit leerem
// Ausweisfeld schreibt (repository.UpdateUser): Der Trigger wartet bis zum Commit.
func TestAusweisnummer_AnfrageKeineFreischaltungEine(t *testing.T) {
	pool := pgTestPool(t)
	resetBestandsdaten(t, pool)
	ctx := context.Background()

	var kontoID, leserID string
	if err := pool.QueryRow(ctx, `
		INSERT INTO benutzer (email, vorname, nachname, rolle, aktiv, zugang_beantragt_am)
		VALUES ('ausweis-anfrage@example.org', 'Anfrage', 'Probe', 'kollegium', false, now())
		RETURNING id, leser_id`).Scan(&kontoID, &leserID); err != nil {
		t.Fatalf("Anfrage anlegen: %v", err)
	}
	ausweis := func() *string {
		t.Helper()
		var n *string
		if err := pool.QueryRow(ctx, `SELECT barcode_id FROM leser WHERE id = $1`, leserID).Scan(&n); err != nil {
			t.Fatal(err)
		}
		return n
	}
	if n := ausweis(); n != nil {
		t.Fatalf("offene Anfrage trägt Ausweis %s, erwartet keinen", *n)
	}

	// Freischaltung wie UpdateUser: erst das Konto, dann die Leserzeile mit leerem Feld.
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer db.SafeRollback(ctx, tx)
	if _, err := tx.Exec(ctx, `UPDATE benutzer SET aktiv = true, zugang_beantragt_am = NULL WHERE id = $1`, kontoID); err != nil {
		t.Fatal(err)
	}
	if _, err := tx.Exec(ctx, `UPDATE leser SET barcode_id = NULLIF('', '') WHERE id = $1`, leserID); err != nil {
		t.Fatal(err)
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatalf("Freischaltung: %v", err)
	}
	if n := ausweis(); n == nil || *n != "A-10001" {
		t.Fatalf("nach der Freischaltung Ausweis %s, erwartet A-10001", nummerOderNull(n))
	}
}

// Ein aktives Konto ist nie ohne Nummer (Migration 145, docs/OFFEN.md 5.16): Leert die
// Verwaltung sie, zieht die Datenbank eine neue — roh und über die Benutzerverwaltung; die
// Akte prüft TestAusweisnummerLeeren, das Zusammenführen TestZusammenfuehrenKollegium. Die
// geleerte kommt dabei nicht zurück. Ein inaktives Konto bleibt leer, wie eine offene
// Zugangsanfrage (Waisen-Regel, Migration 136).
func TestAusweisnummer_LeerenZiehtNeue(t *testing.T) {
	pool := pgTestPool(t)
	resetBestandsdaten(t, pool)
	ctx := context.Background()
	users := repository.NewUserRepository(pool)

	var kontoID, leserID string
	if err := pool.QueryRow(ctx, `
		INSERT INTO benutzer (email, vorname, nachname, rolle)
		VALUES ('ausweis-leeren@example.org', 'Leeren', 'Probe', 'kollegium')
		RETURNING id, leser_id`).Scan(&kontoID, &leserID); err != nil {
		t.Fatalf("Konto anlegen: %v", err)
	}
	ausweis := func() *string {
		t.Helper()
		var n *string
		if err := pool.QueryRow(ctx, `SELECT barcode_id FROM leser WHERE id = $1`, leserID).Scan(&n); err != nil {
			t.Fatal(err)
		}
		return n
	}
	pruefe := func(wo, erwartet string) {
		t.Helper()
		if n := ausweis(); nummerOderNull(n) != erwartet {
			t.Errorf("%s: Ausweis %s, erwartet %s", wo, nummerOderNull(n), erwartet)
		}
	}
	aendere := func(aktiv bool) error {
		return users.UpdateUser(ctx, repository.UpdateUserParams{
			ID: kontoID, Vorname: "Leeren", Nachname: "Probe", Email: "ausweis-leeren@example.org",
			Rolle: "kollegium", Aktiv: aktiv,
		})
	}
	pruefe("nach dem Anlegen", "A-10001")

	if _, err := pool.Exec(ctx, `UPDATE leser SET barcode_id = NULL WHERE id = $1`, leserID); err != nil {
		t.Fatal(err)
	}
	pruefe("roh geleert", "A-10002")

	// Die Benutzerverwaltung schickt ein leeres Feld; UpdateUser schreibt es als NULL.
	if err := aendere(true); err != nil {
		t.Fatalf("Benutzerverwaltung: %v", err)
	}
	pruefe("in der Benutzerverwaltung geleert", "A-10003")

	// Deaktivieren mit leerem Feld: kein aktives Konto, keine Nummer.
	if err := aendere(false); err != nil {
		t.Fatalf("deaktivieren: %v", err)
	}
	pruefe("deaktiviert und geleert", "NULL")

	// Wieder aktiv: Die Leserzeile bekommt NULL über NULL, das lässt der Trigger an leser
	// liegen; der aufgeschobene aus 136 zieht beim Commit. WELCHE Nummer, prüft der Test
	// nicht: Der Generator rechnet über die höchste Nummer in der Tabelle, und die beim
	// Deaktivieren geleerte A-10003 steht dort nicht mehr (docs/OFFEN.md 5.23).
	if err := aendere(true); err != nil {
		t.Fatalf("wieder freischalten: %v", err)
	}
	vorher := ausweis()
	if vorher == nil {
		t.Fatal("wieder aktiv: keine Ausweisnummer")
	}

	// Die gezogene Nummer geht durch die Buch-Prüfung (trg_leser_nummer_ist_kein_buch
	// feuert nach diesem Trigger): Trägt ein Buch die nächste Nummer, scheitert das Leeren
	// laut, statt dem Kollegen die Nummer eines Buchs zu geben.
	naechste := AusweisNummer(naechsteAusweisnummer(t, pool))
	if _, err := pool.Exec(ctx, `INSERT INTO buecher_titel (titel) VALUES ('Leeren: Nummernprobe')`); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO buecher_exemplare (titel_id, barcode_id)
		VALUES ((SELECT id FROM buecher_titel WHERE titel = 'Leeren: Nummernprobe'), $1)`, naechste); err != nil {
		t.Fatal(err)
	}
	_, err := pool.Exec(ctx, `UPDATE leser SET barcode_id = NULL WHERE id = $1`, leserID)
	if !repository.IstNummerBuchOderAusweisKollision(err) {
		t.Errorf("Leeren, während ein Buch %s trägt: %v, erwartet die Buch-Kollision", naechste, err)
	}
	pruefe("nach der Buch-Kollision", *vorher)
}

// Der Nachtrag der Migration: Jeder aktive Leser ohne Nummer bekommt eine, fortlaufend in
// der Reihenfolge seiner Anlage; ein gelöschter nicht, und ein zweiter Lauf ändert nichts.
// Ausgeführt wird die Migrationsdatei selbst — sie ist idempotent geschrieben.
//
// In einer Transaktion, die zurückgerollt wird: Die Datei fasst Funktion und Trigger
// aktives_konto_hat_ausweis in ihrer Form von 136, und Migration 145 hat beide neu gefasst.
// Liefe sie festgeschrieben, prüften alle späteren Tests des Pakets gegen den alten Trigger.
func TestAusweisnummer_MigrationTraegtFehlendeNach(t *testing.T) {
	pool := pgTestPool(t)
	resetBestandsdaten(t, pool)
	ctx := context.Background()
	schueler(t, pool, "A-10100")
	for _, sql := range []string{
		`INSERT INTO leser (vorname, nachname, art, erstellt_am) VALUES ('Erste', 'Lehrkraft', 'lehrkraft', '2026-01-01')`,
		`INSERT INTO leser (vorname, nachname, art, erstellt_am) VALUES ('Zweite', 'Lehrkraft', 'lehrkraft', '2026-02-01')`,
		`INSERT INTO leser (vorname, nachname, art, deleted_at) VALUES ('Geloescht', 'Lehrkraft', 'lehrkraft', now())`,
		// Die Leserzeile einer offenen Zugangsanfrage legt der Trigger an; sie bleibt ohne.
		`INSERT INTO benutzer (email, vorname, nachname, rolle, aktiv, zugang_beantragt_am)
		 VALUES ('nachtrag-anfrage@example.org', 'Offen', 'Anfrage', 'kollegium', false, now())`,
	} {
		if _, err := pool.Exec(ctx, sql); err != nil {
			t.Fatal(err)
		}
	}
	pruefeMigrationslauf(t, pool, "136_ausweisnummer_beim_konto.sql",
		map[string]*string{"Erste": ptr("A-10101"), "Zweite": ptr("A-10102"), "Geloescht": nil, "Offen": nil})
}

// Der Nachtrag von Migration 145: Ein aktives Konto, dessen Leserzeile vor 145 ohne Nummer
// dastand, bekommt eine. Ein inaktives Konto und ein Leser ohne Konto bleiben ohne, und ein
// zweiter Lauf ändert nichts.
func TestAusweisnummer_Migration145TraegtNach(t *testing.T) {
	pool := pgTestPool(t)
	resetBestandsdaten(t, pool)
	ctx := context.Background()
	schueler(t, pool, "A-10200")
	for _, sql := range []string{
		`INSERT INTO leser (vorname, nachname, art, erstellt_am) VALUES ('Aktiv', 'Altstand', 'lehrkraft', '2026-01-01')`,
		`INSERT INTO leser (vorname, nachname, art, erstellt_am) VALUES ('Inaktiv', 'Altstand', 'lehrkraft', '2026-02-01')`,
		`INSERT INTO leser (vorname, nachname, art) VALUES ('Kontolos', 'Altstand', 'lehrkraft')`,
	} {
		if _, err := pool.Exec(ctx, sql); err != nil {
			t.Fatal(err)
		}
	}
	// Der Zustand vor 145: Konten an Zeilen ohne Nummer, an den Triggern vorbei geschrieben
	// (Rolle „replica" schaltet sie für diese Transaktion ab).
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer db.SafeRollback(ctx, tx)
	for _, sql := range []string{
		`SET LOCAL session_replication_role = replica`,
		`INSERT INTO benutzer (email, vorname, nachname, rolle, aktiv, leser_id)
		 SELECT 'altstand-aktiv@example.org', 'Aktiv', 'Altstand', 'kollegium', true, id FROM leser WHERE vorname = 'Aktiv'`,
		`INSERT INTO benutzer (email, vorname, nachname, rolle, aktiv, leser_id)
		 SELECT 'altstand-inaktiv@example.org', 'Inaktiv', 'Altstand', 'kollegium', false, id FROM leser WHERE vorname = 'Inaktiv'`,
	} {
		if _, err := tx.Exec(ctx, sql); err != nil {
			t.Fatal(err)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatal(err)
	}
	pruefeMigrationslauf(t, pool, "145_geleerte_ausweisnummer_zieht_neue.sql",
		map[string]*string{"Aktiv": ptr("A-10201"), "Inaktiv": nil, "Kontolos": nil})
}

// pruefeMigrationslauf führt eine Migrationsdatei zweimal aus und liest nach jedem Lauf die
// Ausweisnummern nach Vorname — in EINER Transaktion, die danach zurückgerollt wird, damit
// die Funktionen und Trigger der Datei nicht die des Endstands (schema.sql) ersetzen.
func pruefeMigrationslauf(t *testing.T, pool *pgxpool.Pool, datei string, soll map[string]*string) {
	t.Helper()
	ctx := context.Background()
	migration, err := os.ReadFile("../migrations/" + datei)
	if err != nil {
		t.Fatal(err)
	}
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer db.SafeRollback(ctx, tx)
	for lauf := 1; lauf <= 2; lauf++ {
		if _, err := tx.Exec(ctx, string(migration)); err != nil {
			t.Fatalf("%s, Lauf %d: %v", datei, lauf, err)
		}
		for vorname, erwartet := range soll {
			var ist *string
			if err := tx.QueryRow(ctx, `SELECT barcode_id FROM leser WHERE vorname = $1`, vorname).Scan(&ist); err != nil {
				t.Fatal(err)
			}
			if (ist == nil) != (erwartet == nil) || (ist != nil && *ist != *erwartet) {
				t.Errorf("%s, Lauf %d, %s: Ausweis %v, erwartet %v", datei, lauf, vorname, nummerOderNull(ist), nummerOderNull(erwartet))
			}
		}
	}
}

// NULL über NULL zieht nicht sofort. So schreibt die Littera-Übernahme jede Lehrkraft ohne
// Ausweis: erst das Konto, dann die Leserzeile mit ihrer Nummer oder NULL — in EINER
// Transaktion über den ganzen Bestand. Nähme der Trigger an leser hier den Lock des
// Generators, hielte der Lauf ihn bis zu seinem Ende, und jede Neuanlage an der Theke
// wartete so lange. Die Nummer kommt beim Commit (aufgeschobener Trigger aus 136).
func TestAusweisnummer_NullUeberNullZiehtErstBeimCommit(t *testing.T) {
	pool := pgTestPool(t)
	resetBestandsdaten(t, pool)
	ctx := context.Background()

	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer db.SafeRollback(ctx, tx)
	var leserID string
	if err := tx.QueryRow(ctx, `
		INSERT INTO benutzer (email, vorname, nachname, rolle, aktiv)
		VALUES ('null-ueber-null@example.org', 'Null', 'Probe', 'kollegium', true)
		RETURNING leser_id`).Scan(&leserID); err != nil {
		t.Fatal(err)
	}
	if _, err := tx.Exec(ctx, `UPDATE leser SET barcode_id = NULL, art = 'lehrkraft' WHERE id = $1`, leserID); err != nil {
		t.Fatal(err)
	}
	var locks int
	if err := tx.QueryRow(ctx, `
		SELECT count(*) FROM pg_locks WHERE locktype = 'advisory' AND pid = pg_backend_pid()`).Scan(&locks); err != nil {
		t.Fatal(err)
	}
	if locks != 0 {
		t.Errorf("vor dem Commit hält die Transaktion %d Advisory-Lock(s) — der Generator wurde mitten im Lauf gerufen", locks)
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatal(err)
	}
	var nummer *string
	if err := pool.QueryRow(ctx, `SELECT barcode_id FROM leser WHERE id = $1`, leserID).Scan(&nummer); err != nil {
		t.Fatal(err)
	}
	if nummerOderNull(nummer) != "A-10001" {
		t.Errorf("nach dem Commit Ausweis %s, erwartet A-10001", nummerOderNull(nummer))
	}
}

func nummerOderNull(s *string) string {
	if s == nil {
		return "NULL"
	}
	return *s
}

// schueler legt einen aktiven Schüler mit gegebenem Ausweis-Barcode an.
func schueler(t *testing.T, pool *pgxpool.Pool, barcode string) {
	t.Helper()
	_, err := pool.Exec(context.Background(),
		`INSERT INTO schueler (barcode_id, vorname, nachname, klasse, abgaenger_jahr)
		 VALUES ($1, 'Test', 'Schueler', '5a', 2030)`, barcode)
	if err != nil {
		t.Fatalf("Schüler %q anlegen: %v", barcode, err)
	}
}
