package jobs

import (
	"context"
	"os"
	"testing"

	"bibliothek/repository"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Die Lesehistorie-Befristung am echten Postgres. Geprüft wird die PAARUNG aus Frist
// und Medienklasse — genau das, was ein Mock nur nachspielen würde: LMF-Erkennung per
// Regex auf der Signatur, make_interval, der Schadensfall-Wächter, die Geräte ohne
// Exemplar, die Lehrer-Ausleihe und der Aus-Schalter (0 Tage).
//
// Erwartung je Ausleihe nach dem Lauf mit den Vorgaben (90 / 730 Tage):
//
//	F-ALT   Freihand, vor 100 Tagen zurück            → getrennt
//	F-JUNG  Freihand, vor 10 Tagen zurück             → bleibt
//	F-SCHAD Freihand, vor 100 Tagen, offener Schaden  → bleibt
//	F-OFFEN Freihand, noch ausgeliehen                → bleibt
//	F-LEHR  Freihand, Lehrer, vor 800 Tagen zurück    → bleibt (dienstlich)
//	G-ALT   Gerät, vor 100 Tagen zurück               → getrennt (kurze Frist)
//	L-MITTEL Lernmittel, vor 100 Tagen zurück         → bleibt (lange Frist)
//	L-ALT   Lernmittel, vor 800 Tagen zurück          → getrennt
func TestLesehistorieBefristung_TrenntNachFristUndKlasse(t *testing.T) {
	adminDSN := os.Getenv(drillEnvVar)
	if adminDSN == "" {
		t.Skipf("%s nicht gesetzt — Test übersprungen", drillEnvVar)
	}
	_, dsn := legeProbeDatenbankAn(t, adminDSN, "lesehist")
	befuelleQuelle(t, dsn)

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatalf("Pool: %v", err)
	}
	t.Cleanup(pool.Close)

	must := func(sql string, args ...any) {
		t.Helper()
		if _, err := pool.Exec(ctx, sql, args...); err != nil {
			t.Fatalf("%s: %v", sql, err)
		}
	}
	var schuelerID, lehrerID, freihandTitel, lmfTitel, geraetID string
	if err := pool.QueryRow(ctx, `SELECT id FROM schueler WHERE barcode_id = 'S-DRILL-1'`).Scan(&schuelerID); err != nil {
		t.Fatalf("Probeschüler: %v", err)
	}
	if err := pool.QueryRow(ctx, `INSERT INTO benutzer (barcode_id, vorname, nachname, email, rolle, aktiv)
		VALUES ('L-HIST', 'Hanna', 'Lehr', 'lehr@example.org', 'kollegium', true) RETURNING id`).Scan(&lehrerID); err != nil {
		t.Fatalf("Lehrer: %v", err)
	}
	if err := pool.QueryRow(ctx, `INSERT INTO buecher_titel (titel, signatur) VALUES ('Der Roman', 'Ro Mus') RETURNING id`).Scan(&freihandTitel); err != nil {
		t.Fatalf("Freihand-Titel: %v", err)
	}
	// Lernmittel-Kennung in der SIGNATUR (so schreibt die Admin-UI), nicht im Titel.
	if err := pool.QueryRow(ctx, `INSERT INTO buecher_titel (titel, signatur) VALUES ('Deutschbuch 7', 'LMF-Deutsch 7') RETURNING id`).Scan(&lmfTitel); err != nil {
		t.Fatalf("LMF-Titel: %v", err)
	}
	if err := pool.QueryRow(ctx, `INSERT INTO geraete (modellname, barcode_id) VALUES ('Laptop', 'G-HIST') RETURNING id`).Scan(&geraetID); err != nil {
		t.Fatalf("Gerät: %v", err)
	}

	exemplar := func(titel, barcode string) string {
		t.Helper()
		var id string
		if err := pool.QueryRow(ctx, `INSERT INTO buecher_exemplare (titel_id, barcode_id) VALUES ($1, $2) RETURNING id`, titel, barcode).Scan(&id); err != nil {
			t.Fatalf("Exemplar %s: %v", barcode, err)
		}
		return id
	}
	// leihe legt eine Schüler-Ausleihe an; tageZurueck < 0 = noch offen.
	leihe := func(name, exemplarID string, tageZurueck int) string {
		t.Helper()
		var id string
		if err := pool.QueryRow(ctx, `
			INSERT INTO ausleihen (exemplar_id, schueler_id, ausgeliehen_am, rueckgabe_frist, rueckgabe_am)
			VALUES ($1, $2, NOW() - make_interval(days => $3 + 30), NOW() - make_interval(days => $3 + 9),
			        CASE WHEN $3 < 0 THEN NULL ELSE NOW() - make_interval(days => $3) END)
			RETURNING id`, exemplarID, schuelerID, tageZurueck).Scan(&id); err != nil {
			t.Fatalf("Ausleihe %s: %v", name, err)
		}
		return id
	}
	ids := map[string]string{}
	ids["F-ALT"] = leihe("F-ALT", exemplar(freihandTitel, "B-F1"), 100)
	ids["F-JUNG"] = leihe("F-JUNG", exemplar(freihandTitel, "B-F2"), 10)
	ids["F-SCHAD"] = leihe("F-SCHAD", exemplar(freihandTitel, "B-F3"), 100)
	ids["F-OFFEN"] = leihe("F-OFFEN", exemplar(freihandTitel, "B-F4"), -1)
	ids["L-MITTEL"] = leihe("L-MITTEL", exemplar(lmfTitel, "B-L1"), 100)
	ids["L-ALT"] = leihe("L-ALT", exemplar(lmfTitel, "B-L2"), 800)
	must(`INSERT INTO schadensfaelle (exemplar_id, ausleihe_id, schueler_id, beschreibung, betrag, ist_bezahlt)
	      SELECT exemplar_id, id, schueler_id, 'Einband gerissen', 12.50, false FROM ausleihen WHERE id = $1`, ids["F-SCHAD"])
	if err := pool.QueryRow(ctx, `
		INSERT INTO ausleihen (exemplar_id, ausleiher_benutzer_id, ausgeliehen_am, rueckgabe_frist, rueckgabe_am, ist_handapparat)
		VALUES ($1, $2, NOW() - interval '830 days', NOW() - interval '809 days', NOW() - interval '800 days', true) RETURNING id`,
		exemplar(freihandTitel, "B-F5"), lehrerID).Scan(new(string)); err != nil {
		t.Fatalf("Lehrer-Ausleihe: %v", err)
	}
	if err := pool.QueryRow(ctx, `
		INSERT INTO ausleihen (geraet_id, schueler_id, ausgeliehen_am, rueckgabe_frist, rueckgabe_am)
		VALUES ($1, $2, NOW() - interval '130 days', NOW() - interval '109 days', NOW() - interval '100 days') RETURNING id`,
		geraetID, schuelerID).Scan(new(string)); err != nil {
		t.Fatalf("Geräte-Ausleihe: %v", err)
	}
	var geraetAusleihe string
	if err := pool.QueryRow(ctx, `SELECT id FROM ausleihen WHERE geraet_id = $1`, geraetID).Scan(&geraetAusleihe); err != nil {
		t.Fatalf("Geräte-Ausleihe lesen: %v", err)
	}
	ids["G-ALT"] = geraetAusleihe

	// Prüfung 22.08.2026, A5: Das Ausleih-Protokoll (audit_log CHECKOUT/RETURN, Details
	// mit schueler_id) trug die Lesehistorie weiter, nachdem die Ausleihe längst getrennt
	// war — bis zur Audit-Aufbewahrung (24 Monate). Dieselbe Frist gilt jetzt auch dort.
	exemplarVon := func(ausleiheID string) string {
		t.Helper()
		var e string
		if err := pool.QueryRow(ctx, `SELECT exemplar_id FROM ausleihen WHERE id = $1`, ausleiheID).Scan(&e); err != nil {
			t.Fatalf("Exemplar lesen: %v", err)
		}
		return e
	}
	for name, tage := range map[string]int{"F-ALT": 100, "F-JUNG": 10, "L-MITTEL": 100, "L-ALT": 800} {
		must(`INSERT INTO audit_log (tabelle, aktion, datensatz_id, akteur, details, timestamp)
		      VALUES ('ausleihen', 'RETURN', $1::uuid, 'USER',
		              jsonb_build_object('exemplar_id', $1::text, 'schueler_id', $2::text),
		              NOW() - make_interval(days => $3))`, exemplarVon(ids[name]), schuelerID, tage)
	}

	s := NewScheduler(pool, repository.NewAuditRepository(pool))
	s.RunLesehistorieBefristung()

	auditTraegtSchueler := func(name string) bool {
		t.Helper()
		var n int
		if err := pool.QueryRow(ctx, `SELECT count(*) FROM audit_log WHERE tabelle='ausleihen' AND datensatz_id = $1 AND details ? 'schueler_id'`, exemplarVon(ids[name])).Scan(&n); err != nil {
			t.Fatalf("audit_log lesen: %v", err)
		}
		return n > 0
	}
	for name, bleibt := range map[string]bool{"F-ALT": false, "F-JUNG": true, "L-MITTEL": true, "L-ALT": false} {
		if got := auditTraegtSchueler(name); got != bleibt {
			t.Errorf("audit_log %s: schueler_id vorhanden = %v, erwartet %v", name, got, bleibt)
		}
	}

	hatSchueler := func(id string) bool {
		t.Helper()
		var hat bool
		if err := pool.QueryRow(ctx, `SELECT schueler_id IS NOT NULL FROM ausleihen WHERE id = $1`, id).Scan(&hat); err != nil {
			t.Fatalf("Ausleihe %s lesen: %v", id, err)
		}
		return hat
	}
	erwartung := map[string]bool{ // true = Schüler bleibt zugeordnet
		"F-ALT": false, "F-JUNG": true, "F-SCHAD": true, "F-OFFEN": true,
		"G-ALT": false, "L-MITTEL": true, "L-ALT": false,
	}
	for name, bleibt := range erwartung {
		if got := hatSchueler(ids[name]); got != bleibt {
			t.Errorf("%s: Schüler zugeordnet = %v, erwartet %v", name, got, bleibt)
		}
	}
	var lehrerBleibt bool
	if err := pool.QueryRow(ctx, `SELECT ausleiher_benutzer_id IS NOT NULL FROM ausleihen WHERE ausleiher_benutzer_id = $1`, lehrerID).Scan(&lehrerBleibt); err != nil || !lehrerBleibt {
		t.Errorf("Lehrer-Ausleihe wurde angefasst (err=%v, bleibt=%v)", err, lehrerBleibt)
	}

	// Der Lauf protokolliert sich als Systemaktion — mit den Zahlen beider Klassen.
	var auditZeilen int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM audit_log WHERE tabelle = 'ausleihen' AND aktion = 'ANONYMIZE'
		AND details->>'schuelerbuecherei_getrennt' = '2' AND details->>'lernmittel_getrennt' = '1'`).Scan(&auditZeilen); err != nil {
		t.Fatalf("audit_log: %v", err)
	}
	if auditZeilen != 1 {
		t.Errorf("erwartet genau 1 Audit-Eintrag mit 2/1 Trennungen, gefunden %d", auditZeilen)
	}

	// Aus-Schalter: 0 Tage Schülerbücherei → nichts mehr trennen, Lernmittel läuft weiter.
	must(`INSERT INTO system_einstellungen (schluessel, wert) VALUES ('lesehistorie_tage', '0')
	      ON CONFLICT (schluessel) DO UPDATE SET wert = EXCLUDED.wert`)
	fAus := leihe("F-AUS", exemplar(freihandTitel, "B-F6"), 100)
	lAus := leihe("L-AUS", exemplar(lmfTitel, "B-L3"), 800)
	s.RunLesehistorieBefristung()
	if !hatSchueler(fAus) {
		t.Errorf("F-AUS: bei lesehistorie_tage=0 darf die Schülerbücherei NICHT getrennt werden")
	}
	if hatSchueler(lAus) {
		t.Errorf("L-AUS: Lernmittel-Frist muss unabhängig vom Schülerbücherei-Schalter weiterlaufen")
	}
}
