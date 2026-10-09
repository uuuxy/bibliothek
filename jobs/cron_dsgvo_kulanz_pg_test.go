package jobs

import (
	"context"
	"os"
	"testing"

	"bibliothek/repository"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Die Kulanz gehört dem Wächter, nicht dem Lauf. Ein Datensatz, dessen Frist seit zwölf
// Stunden abgelaufen ist, zählt für den Wächter noch nicht als Rückstand, und der Lauf dieser
// Nacht nimmt ihn trotzdem. Rechnete eine Routine mit der Kulanz des Wächters, bliebe jeder
// Datensatz einen Tag länger stehen, und kein Rückstand zeigte es.
func TestNachtlauf_NimmtWasSeitZwoelfStundenFaelligIst(t *testing.T) {
	adminDSN := os.Getenv(drillEnvVar)
	if adminDSN == "" {
		t.Skipf("%s nicht gesetzt — Test übersprungen", drillEnvVar)
	}
	_, dsn := legeProbeDatenbankAn(t, adminDSN, "laufkulanz")
	befuelleQuelle(t, dsn)

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatalf("Pool: %v", err)
	}
	t.Cleanup(pool.Close)

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
	anzahl := func(sql string, args ...any) int {
		t.Helper()
		var n int
		if err := pool.QueryRow(ctx, sql, args...).Scan(&n); err != nil {
			t.Fatalf("%s: %v", sql, err)
		}
		return n
	}

	const (
		freihandTage   = repository.StandardLesehistorieTage
		lernmittelTage = repository.StandardLesehistorieLernmittelTage
		anliegenTage   = repository.StandardAnliegenTage
		papierkorbTage = repository.StandardAnonymisierungSoftDeleteTage
		// Länger als die Vorgabe von 24 Monaten: Mit ihr fiele die Protokollzeile der
		// Lernmittel-Ausleihe (730 Tage) in derselben Nacht unter beide Fristen, und ihr Fall
		// mäße die Lesehistorie nicht.
		auditMonate = 36
	)
	must("Aufbewahrung der Protokolle", `INSERT INTO system_einstellungen (schluessel, wert) VALUES ($1, $2)`,
		repository.AuditAufbewahrungSchluessel, "36")
	// Ohne gespeicherte Einstellungen folgt die Frist der Nachbuch-Meldungen der Vorgabe der
	// Lesehistorie.
	nachbuchTage := repository.NachbuchMeldungenTage(&repository.SystemEinstellungen{})

	schuelerID := eins("Probeschüler", `SELECT id FROM schueler WHERE barcode_id = 'S-DRILL-1'`)
	freihand := eins("Freihand-Titel", `INSERT INTO buecher_titel (titel, signatur) VALUES ('Der Roman', 'Ro Mus') RETURNING id`)
	lmf := eins("LMF-Titel", `INSERT INTO buecher_titel (titel, signatur, ist_lernmittel) VALUES ('Deutschbuch 7', 'LMF-Deutsch 7', true) RETURNING id`)

	must("Schüler im Papierkorb", `
		INSERT INTO schueler (barcode_id, vorname, nachname, klasse, abgaenger_jahr, deleted_at)
		VALUES ('S-KULANZ', 'Paula', 'Papierkorb', '9c', 2030, NOW() - make_interval(days => $1, hours => 12))`, papierkorbTage)
	kollege := eins("Kollege im Papierkorb", `
		INSERT INTO leser (art, vorname, nachname, deleted_at)
		VALUES ('lehrkraft', 'Konrad', 'Papierkorb', NOW() - make_interval(days => $1, hours => 12)) RETURNING id`, papierkorbTage)

	// leihe legt eine zurückgegebene Ausleihe an und die Protokollzeile, die denselben Leser trägt.
	leihe := func(was, titelID, barcode string, tage int) (ausleihe, exemplar string) {
		t.Helper()
		exemplar = eins("Exemplar "+barcode,
			`INSERT INTO buecher_exemplare (titel_id, barcode_id) VALUES ($1, $2) RETURNING id`, titelID, barcode)
		ausleihe = eins(was, `
			INSERT INTO ausleihen (exemplar_id, schueler_id, ausgeliehen_am, rueckgabe_frist, rueckgabe_am)
			VALUES ($1, $2, NOW() - make_interval(days => $3 + 30), NOW() - make_interval(days => $3 + 9),
			        NOW() - make_interval(days => $3, hours => 12)) RETURNING id`, exemplar, schuelerID, tage)
		must(was+" im Protokoll", `
			INSERT INTO audit_log (tabelle, aktion, datensatz_id, timestamp, details)
			VALUES ('ausleihen', 'RETURN', $1, NOW() - make_interval(days => $2, hours => 12),
			        jsonb_build_object('schueler_id', $3::text))`, exemplar, tage, schuelerID)
		return ausleihe, exemplar
	}
	freihandLeihe, freihandExemplar := leihe("Freihand-Ausleihe", freihand, "B-KULANZ-F", freihandTage)
	lmfLeihe, lmfExemplar := leihe("Lernmittel-Ausleihe", lmf, "B-KULANZ-L", lernmittelTage)

	vormerkSpur := eins("Spur einer gelöschten Vormerkung", `
		INSERT INTO audit_log (tabelle, aktion, datensatz_id, timestamp, details)
		VALUES ('vormerkungen', 'DELETE', gen_random_uuid(), NOW() - make_interval(days => $1, hours => 12),
		        jsonb_build_object('schueler_id', $2::text, 'betrifft', 'Probe Schüler', 'titel', 'Der Roman'))
		RETURNING id`, freihandTage, schuelerID)

	must("Anliegen", `
		INSERT INTO lehrer_anliegen (art, titel_text, kommentar, erstellt_am, erledigt_am)
		VALUES ('wunsch', 'GERADE-FAELLIG', 'x', NOW() - make_interval(days => $1 + 30),
		        NOW() - make_interval(days => $1, hours => 12))`, anliegenTage)
	must("Klassensatz-Reservierung", `
		INSERT INTO klassensatz_reservierungen (titel_id, klasse, anzahl, erledigt, erstellt_am, erledigt_am)
		VALUES ($1, '7a', 30, true, NOW() - make_interval(days => $2 + 30),
		        NOW() - make_interval(days => $2, hours => 12))`, freihand, anliegenTage)
	must("Nachbuch-Meldung", `
		INSERT INTO nachbuch_meldungen (idempotency_key, barcode, ergebnis, gescannt_am, erstellt_am, quittiert_am)
		VALUES (gen_random_uuid(), 'B-KULANZ-NB', 'umgebucht', NOW() - make_interval(days => $1 + 5),
		        NOW() - make_interval(days => $1 + 5), NOW() - make_interval(days => $1, hours => 12))`, nachbuchTage)
	altesProtokoll := eins("altes Protokoll der Datensätze", `
		INSERT INTO audit_log (tabelle, aktion, datensatz_id, timestamp)
		VALUES ('buecher_titel', 'UPDATE', $1, NOW() - make_interval(months => $2, hours => 12)) RETURNING id`, freihand, auditMonate)
	alteVerwaltung := eins("altes Protokoll der Verwaltung", `
		INSERT INTO audit_logs (aktion, details, zeitstempel)
		VALUES ('EINSTELLUNG_GEAENDERT', '{}'::jsonb, NOW() - make_interval(months => $1, hours => 12)) RETURNING id`, auditMonate)

	// Vor dem Lauf: Der Wächter mahnt nichts davon an.
	for routine, zeilen := range rueckstandAlsMap(ctx, t, repository.NewBetriebszustandRepository(pool)) {
		if zeilen != 0 {
			t.Errorf("Wächter vor dem Lauf: %q meldet %d Zeilen, obwohl die Frist erst seit zwölf Stunden abgelaufen ist", routine, zeilen)
		}
	}

	s := NewScheduler(pool, repository.NewAuditRepository(pool))
	s.RunNaechtlicheDSGVO()
	s.RunAuditAufbewahrung()

	// Nach dem Lauf: Jede Routine hat ihren Datensatz genommen.
	faelle := []struct {
		routine string
		rest    int
	}{
		{"Schüler-Anonymisierung", anzahl(`SELECT count(*) FROM schueler WHERE barcode_id = 'S-KULANZ' AND anonymized_at IS NULL`)},
		{"Gelöschte Kollegen endgültig löschen", anzahl(`SELECT count(*) FROM leser WHERE id = $1`, kollege)},
		{"Lesehistorie Schülerbücherei, Ausleihe", anzahl(`SELECT count(*) FROM ausleihen WHERE id = $1 AND schueler_id IS NOT NULL`, freihandLeihe)},
		{"Lesehistorie Lernmittel, Ausleihe", anzahl(`SELECT count(*) FROM ausleihen WHERE id = $1 AND schueler_id IS NOT NULL`, lmfLeihe)},
		{"Lesehistorie Schülerbücherei, Protokoll", anzahl(`SELECT count(*) FROM audit_log WHERE tabelle = 'ausleihen' AND datensatz_id = $1 AND details ? 'schueler_id'`, freihandExemplar)},
		{"Lesehistorie Lernmittel, Protokoll", anzahl(`SELECT count(*) FROM audit_log WHERE tabelle = 'ausleihen' AND datensatz_id = $1 AND details ? 'schueler_id'`, lmfExemplar)},
		{"Lesehistorie Schülerbücherei, Spur einer Vormerkung", anzahl(`SELECT count(*) FROM audit_log WHERE id = $1 AND details ?| ARRAY['schueler_id', 'betrifft']`, vormerkSpur)},
		{"Erledigte Anliegen", anzahl(`SELECT count(*) FROM lehrer_anliegen WHERE titel_text = 'GERADE-FAELLIG'`)},
		{"Erledigte Klassensatz-Reservierungen", anzahl(`SELECT count(*) FROM klassensatz_reservierungen WHERE titel_id = $1`, freihand)},
		{"Quittierte Nachbuch-Meldungen", anzahl(`SELECT count(*) FROM nachbuch_meldungen WHERE barcode = 'B-KULANZ-NB'`)},
		{"Audit-Aufbewahrung, Datensätze", anzahl(`SELECT count(*) FROM audit_log WHERE id = $1`, altesProtokoll)},
		{"Audit-Aufbewahrung, Verwaltung", anzahl(`SELECT count(*) FROM audit_logs WHERE id = $1`, alteVerwaltung)},
	}
	for _, f := range faelle {
		if f.rest != 0 {
			t.Errorf("%s: Der Lauf hat den Datensatz stehen lassen, dessen Frist seit zwölf Stunden abgelaufen ist", f.routine)
		}
	}
	// Die Spur der Vormerkung verliert nur den Leser, die Zeile bleibt.
	if n := anzahl(`SELECT count(*) FROM audit_log WHERE id = $1 AND details ? 'titel'`, vormerkSpur); n != 1 {
		t.Errorf("Spur der Vormerkung: %d Zeilen mit Titel nach dem Lauf, erwartet 1", n)
	}
}
