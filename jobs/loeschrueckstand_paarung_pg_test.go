package jobs

import (
	"context"
	"os"
	"testing"
	"time"

	"bibliothek/repository"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Die PAARUNG aus Löschroutine und Wächter, am echten Postgres.
//
// Das Projekt war am 23.08.2026 86 Tage alt — keine der fünf Löschroutinen hatte je
// eine Zeile gelöscht, die kürzeste Frist lag bei 90 Tagen. Sie waren durch Tests
// belegt, die das Datum fälschen; in der Wirklichkeit war noch nie etwas passiert. Ein
// stiller Fehlschlag der ersten echten Löschung wäre erst aufgefallen, wenn jemand
// fragt, warum Daten von vor zwei Jahren noch da sind.
//
// Dieser Test prüft nicht, ob der Job löscht (das tun die Tests daneben je Routine),
// sondern ob WÄCHTER UND JOB DIESELBE FRAGE STELLEN:
//
//	vorher:   jede Routine meldet ihren überfälligen Datensatz     → sonst ist der Wächter blind
//	Lauf:     die echten Jobs, ohne gefälschtes Datum
//	nachher:  keine Routine meldet mehr etwas                      → sonst fragt er etwas anderes als der Job
//
// Beide Richtungen zählen. Ein Wächter, der nach dem Lauf noch meldet, hätte jede Nacht
// falschen Alarm geschlagen — und ein falscher Alarm wird abgeschaltet, nicht gelesen.
func TestLoeschRueckstand_WaechterUndJobStellenDieselbeFrage(t *testing.T) {
	adminDSN := os.Getenv(drillEnvVar)
	if adminDSN == "" {
		t.Skipf("%s nicht gesetzt — Test übersprungen", drillEnvVar)
	}
	_, dsn := legeProbeDatenbankAn(t, adminDSN, "rueckstand")
	befuelleQuelle(t, dsn)

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatalf("Pool: %v", err)
	}
	t.Cleanup(pool.Close)

	legeUeberfaelligeDatenAn(ctx, t, pool)

	zustand := repository.NewBetriebszustandRepository(pool)
	vorher := rueckstandAlsMap(ctx, t, zustand)

	// Richtung 1: Sieht der Wächter überhaupt etwas? Ein Wächter, der nie ausschlägt,
	// ist von einem funktionierenden System nicht zu unterscheiden.
	for routine, zeilen := range vorher {
		if zeilen == 0 {
			t.Errorf("Wächter blind: %q meldet 0, obwohl ein überfälliger Datensatz existiert", routine)
		}
	}
	if len(vorher) == 0 {
		t.Fatal("Wächter lieferte keine einzige Routine")
	}

	// Der Lauf. Kein gefälschtes Datum, keine gesetzte Frist — die Vorgaben.
	s := NewScheduler(pool, repository.NewAuditRepository(pool))
	s.RunGDPRAnonymizeOldData()
	s.RunGDPRDeleteAbgaenger()
	s.RunLesehistorieBefristung()
	s.RunAnliegenBefristung()
	s.RunAuditAufbewahrung()

	// Richtung 2: Ist danach Ruhe? Jede verbleibende Zeile heißt, dass der Wächter eine
	// ANDERE Frage stellt als der Job — die Bugklasse, gegen die diese Datei antritt.
	for routine, zeilen := range rueckstandAlsMap(ctx, t, zustand) {
		if zeilen > 0 {
			t.Errorf("nach dem Lauf meldet %q noch %d Zeilen — Wächter und Job fragen Verschiedenes", routine, zeilen)
		}
	}
}

// TestLoeschRueckstand_KulanzDecktDieLetzteNacht: Ein Datensatz, der HEUTE fällig wird,
// darf den Wächter nicht auslösen — der nächtliche Job hatte ihn noch nicht in der Hand.
// Ohne diese Kulanz stünde die Selbstprüfung jeden Tag zwischen Fälligkeit und Mitternacht
// auf Kritisch, und die tägliche Alarm-Mail ginge mit ihr los.
func TestLoeschRueckstand_KulanzDecktDieLetzteNacht(t *testing.T) {
	adminDSN := os.Getenv(drillEnvVar)
	if adminDSN == "" {
		t.Skipf("%s nicht gesetzt — Test übersprungen", drillEnvVar)
	}
	_, dsn := legeProbeDatenbankAn(t, adminDSN, "kulanz")
	befuelleQuelle(t, dsn)

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatalf("Pool: %v", err)
	}
	t.Cleanup(pool.Close)

	// Erledigt vor genau der Frist plus zwölf Stunden: fällig, aber noch keine Nacht alt.
	if _, err := pool.Exec(ctx, `
		INSERT INTO lehrer_anliegen (art, titel_text, kommentar, erstellt_am, erledigt_am)
		VALUES ('wunsch', 'GERADE-FAELLIG', 'x',
		        NOW() - make_interval(days => $1 + 30),
		        NOW() - make_interval(days => $1, hours => 12))`,
		repository.StandardAnliegenTage); err != nil {
		t.Fatalf("Anliegen: %v", err)
	}

	if n := rueckstandAlsMap(ctx, t, repository.NewBetriebszustandRepository(pool))["Erledigte Anliegen"]; n != 0 {
		t.Errorf("gerade fällig gewordenes Anliegen wird schon angemahnt (%d) — die Kulanz greift nicht", n)
	}
}

// rueckstandAlsMap liest den Wächter und wirft abgeschaltete Fristen heraus (die sind
// eine Entscheidung der Schule, kein Rückstand).
func rueckstandAlsMap(ctx context.Context, t *testing.T, r *repository.BetriebszustandRepository) map[string]int {
	t.Helper()
	stand, err := r.ZaehleLoeschRueckstand(ctx)
	if err != nil {
		t.Fatalf("ZaehleLoeschRueckstand: %v", err)
	}
	m := map[string]int{}
	for _, z := range stand {
		if !z.Aus {
			m[z.Routine] = z.Zeilen
		}
	}
	return m
}

// legeUeberfaelligeDatenAn baut für JEDE Löschroutine genau einen klar überfälligen
// Datensatz. „Klar" heißt: deutlich jenseits von Frist plus Kulanz, damit der Test nicht
// an der Tagesgrenze wackelt.
func legeUeberfaelligeDatenAn(ctx context.Context, t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	must := func(was, sql string, args ...any) {
		t.Helper()
		if _, err := pool.Exec(ctx, sql, args...); err != nil {
			t.Fatalf("%s: %v", was, err)
		}
	}
	eins := func(was, sql string, args ...any) string {
		t.Helper()
		var id string
		if err := pool.QueryRow(ctx, sql, args...).Scan(&id); err != nil {
			t.Fatalf("%s: %v", was, err)
		}
		return id
	}

	// 1. Anonymisierung: seit 200 Tagen im Papierkorb (Frist 180).
	must("Papierkorb-Schüler", `
		INSERT INTO schueler (barcode_id, vorname, nachname, klasse, abgaenger_jahr, deleted_at)
		VALUES ('S-PAPIERKORB', 'Paula', 'Papierkorb', '9c', 2030, NOW() - interval '200 days')`)

	// 2. Abgänger endgültig löschen: Abgangsjahr vor dem Stichjahr. aktualisiert_am
	//    bleibt frisch, damit ihn NUR die Löschung betrifft und nicht auch die
	//    Anonymisierung — der Test soll je Routine eine Aussage machen.
	must("Abgänger", `
		INSERT INTO schueler (barcode_id, vorname, nachname, klasse, ist_abgaenger, abgaenger_jahr, aktualisiert_am)
		VALUES ('S-ABGANG', 'Aaron', 'Abgang', '10b', true, $1, NOW())`,
		repository.AbgaengerStichjahr(time.Now())-1)

	schuelerID := eins("Probeschüler", `SELECT id FROM schueler WHERE barcode_id = 'S-DRILL-1'`)
	freihand := eins("Freihand-Titel", `INSERT INTO buecher_titel (titel, signatur) VALUES ('Der Roman', 'Ro Mus') RETURNING id`)
	lmf := eins("LMF-Titel", `INSERT INTO buecher_titel (titel, signatur) VALUES ('Deutschbuch 7', 'LMF-Deutsch 7') RETURNING id`)

	// 3./4. Lesehistorie, beide Klassen — je die Ausleihe UND die Protokollzeile, die
	//       dieselbe Zuordnung trägt.
	leihe := func(was, titelID, barcode string, tageZurueck int) {
		t.Helper()
		exemplarID := eins("Exemplar "+barcode,
			`INSERT INTO buecher_exemplare (titel_id, barcode_id) VALUES ($1, $2) RETURNING id`, titelID, barcode)
		must(was, `
			INSERT INTO ausleihen (exemplar_id, schueler_id, ausgeliehen_am, rueckgabe_frist, rueckgabe_am)
			VALUES ($1, $2, NOW() - make_interval(days => $3 + 30), NOW() - make_interval(days => $3 + 9),
			        NOW() - make_interval(days => $3))`, exemplarID, schuelerID, tageZurueck)
		must(was+"-Protokoll", `
			INSERT INTO audit_log (tabelle, aktion, datensatz_id, timestamp, details)
			VALUES ('ausleihen', 'RETURN', $1, NOW() - make_interval(days => $2),
			        jsonb_build_object('schueler_id', $3::text))`, exemplarID, tageZurueck, schuelerID)
	}
	leihe("Freihand-Ausleihe", freihand, "B-RUECK-F", repository.StandardLesehistorieTage+40)
	leihe("Lernmittel-Ausleihe", lmf, "B-RUECK-L", repository.StandardLesehistorieLernmittelTage+70)

	// 5. Erledigte Anliegen: vor 400 Tagen erledigt (Frist 365).
	must("Anliegen", `
		INSERT INTO lehrer_anliegen (art, titel_text, kommentar, erstellt_am, erledigt_am)
		VALUES ('wunsch', 'ALT-ERLEDIGT', 'Bitte anschaffen',
		        NOW() - interval '430 days', NOW() - interval '400 days')`)

	// 6. Audit-Aufbewahrung: beide Protokolltabellen, jenseits der 24 Monate.
	must("altes audit_log", `
		INSERT INTO audit_log (tabelle, aktion, datensatz_id, timestamp)
		VALUES ('buecher_titel', 'UPDATE', $1, NOW() - interval '800 days')`, freihand)
	must("altes audit_logs", `
		INSERT INTO audit_logs (aktion, details, zeitstempel)
		VALUES ('EINSTELLUNG_GEAENDERT', '{}'::jsonb, NOW() - interval '800 days')`)
}
