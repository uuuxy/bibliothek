package jobs

import (
	"context"
	"os"
	"testing"

	"bibliothek/repository"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Wird ein Titel gelöscht, an dem noch eine Vormerkung hängt, bleibt im Protokoll eine Zeile
// mit Kennung und Name des Lesers und dem Titel des Buchs. Sie verliert den Leser nach der
// Lesehistorie-Frist der Schülerbücherei, wie die Ausleihe: Länger als die Lesehistorie bindet
// nichts einen Leser an ein Buch. Die Zeile einer gelöschten Forderung behält ihn, sie belegt,
// wessen Forderung mit dem Titel gelöscht wurde.
//
// Die Zeilen entstehen über die Tür der Buchakte (DeleteTitle); der Wächter stellt dieselbe
// Frage wie der Lauf: vorher meldet er die fällige Zeile, danach nichts.
func TestLesehistorieBefristung_VormerkSpurVerliertDenLeser(t *testing.T) {
	adminDSN := os.Getenv(drillEnvVar)
	if adminDSN == "" {
		t.Skipf("%s nicht gesetzt — Test übersprungen", drillEnvVar)
	}
	_, dsn := legeProbeDatenbankAn(t, adminDSN, "vormerkspur")
	befuelleQuelle(t, dsn)

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatalf("Pool: %v", err)
	}
	t.Cleanup(pool.Close)

	zeile := func(ziel any, sql string, args ...any) {
		t.Helper()
		if err := pool.QueryRow(ctx, sql, args...).Scan(ziel); err != nil {
			t.Fatalf("%s: %v", sql, err)
		}
	}
	must := func(sql string, args ...any) {
		t.Helper()
		if _, err := pool.Exec(ctx, sql, args...); err != nil {
			t.Fatalf("%s: %v", sql, err)
		}
	}
	var schuelerID, bearbeiterID string
	zeile(&schuelerID, `SELECT id FROM schueler WHERE barcode_id = 'S-DRILL-1'`)
	zeile(&bearbeiterID, `INSERT INTO benutzer (vorname, nachname, email, rolle, aktiv)
		VALUES ('Vera', 'Vormerkspur', 'vormerkspur@example.org', 'admin', true) RETURNING id`)

	// titelMitVormerkungUndForderung legt einen Titel an, an dem eine offene Vormerkung und
	// eine offene Forderung des Lesers hängen, und löscht ihn über die Tür der Buchakte.
	audit := repository.NewAuditRepository(pool)
	titelMitVormerkungUndForderung := func(name, barcode string) (titelID, exemplarID string) {
		t.Helper()
		zeile(&titelID, `INSERT INTO buecher_titel (titel, signatur) VALUES ($1, 'JF') RETURNING id`, name)
		zeile(&exemplarID, `INSERT INTO buecher_exemplare (titel_id, barcode_id) VALUES ($1, $2) RETURNING id`, titelID, barcode)
		must(`INSERT INTO vormerkungen (titel_id, schueler_id) VALUES ($1, $2)`, titelID, schuelerID)
		must(`INSERT INTO schadensfaelle (exemplar_id, schueler_id, beschreibung, betrag)
			VALUES ($1, $2, 'Einband gerissen', 9.50)`, exemplarID, schuelerID)
		if err := audit.DeleteTitle(ctx, titelID, bearbeiterID); err != nil {
			t.Fatalf("Titel %q löschen: %v", name, err)
		}
		return titelID, exemplarID
	}
	alt, altExemplar := titelMitVormerkungUndForderung("Vorgemerkt und alt", "B-VSPUR-ALT")
	frisch, _ := titelMitVormerkungUndForderung("Vorgemerkt und frisch", "B-VSPUR-NEU")
	// Die Zeilen des ersten Titels sind älter als Frist und Kulanz des Wächters. Die Zeile der
	// Vormerkung steht unter der Kennung des Titels, die der Forderung unter der des Exemplars.
	must(`UPDATE audit_log SET timestamp = NOW() - make_interval(days => $3 + 5)
		WHERE (tabelle = 'vormerkungen' AND datensatz_id = $1)
		   OR (tabelle = 'schadensfaelle' AND datensatz_id = $2)`,
		alt, altExemplar, repository.StandardLesehistorieTage)

	// traegt sagt, ob die eine Zeile zu dieser Kennung in dieser Tabelle den Schlüssel trägt.
	traegt := func(kennung, tabelle, schluessel string) bool {
		t.Helper()
		var zeilen, mit int
		zeile(&zeilen, `SELECT count(*) FROM audit_log WHERE datensatz_id = $1 AND tabelle = $2`, kennung, tabelle)
		if zeilen != 1 {
			t.Fatalf("%s: %d Zeilen für %s im Protokoll, erwartet 1", kennung, zeilen, tabelle)
		}
		zeile(&mit, `SELECT count(*) FROM audit_log
			WHERE datensatz_id = $1 AND tabelle = $2 AND details ? $3`, kennung, tabelle, schluessel)
		return mit == 1
	}
	for _, schluessel := range []string{"schueler_id", "betrifft", "titel"} {
		if !traegt(alt, "vormerkungen", schluessel) {
			t.Fatalf("Die Zeile der Vormerkung trägt %q schon vor dem Lauf nicht", schluessel)
		}
	}

	const routine = "Lesehistorie Schülerbücherei"
	zustand := repository.NewBetriebszustandRepository(pool)
	if n := rueckstandAlsMap(ctx, t, zustand)[routine]; n != 1 {
		t.Errorf("Wächter vor dem Lauf: %d fällige Zeilen, erwartet 1 (die Zeile der alten Vormerkung)", n)
	}

	NewScheduler(pool, audit).RunLesehistorieBefristung()

	for _, schluessel := range []string{"schueler_id", "betrifft"} {
		if traegt(alt, "vormerkungen", schluessel) {
			t.Errorf("Die Zeile der alten Vormerkung trägt %q nach dem Lauf noch", schluessel)
		}
		if !traegt(frisch, "vormerkungen", schluessel) {
			t.Errorf("Die Zeile der frischen Vormerkung hat %q vor der Frist verloren", schluessel)
		}
	}
	if !traegt(alt, "vormerkungen", "titel") {
		t.Error("Die Zeile der alten Vormerkung hat den Titel verloren; nur der Leser fällt")
	}
	for _, schluessel := range []string{"schueler_id", "schuldner"} {
		if !traegt(altExemplar, "schadensfaelle", schluessel) {
			t.Errorf("Die Zeile der gelöschten Forderung hat %q verloren; sie bleibt bis zur Tilgung des Lesers", schluessel)
		}
	}
	if n := rueckstandAlsMap(ctx, t, zustand)[routine]; n != 0 {
		t.Errorf("Wächter nach dem Lauf: %d fällige Zeilen, erwartet 0", n)
	}
}
