package jobs

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"bibliothek/repository"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Die Paarung aus Löschlauf und Wächter, an Postgres.
//
// Eine Löschung mit langer Frist läuft im Betrieb erst nach Monaten zum ersten Mal wirklich;
// Tests, die das Datum fälschen, belegen sie nur am Modell. Ein stummer Fehlschlag der ersten
// echten Löschung fiele erst auf, wenn jemand fragt, warum Daten von vor zwei Jahren noch da
// sind.
//
// Der Test prüft nicht, ob der Lauf löscht (das tun die Tests daneben je Lauf), sondern ob
// Wächter und Lauf dieselbe Frage stellen:
//
//	vorher:   jeder Lauf meldet seinen überfälligen Datensatz       → sonst ist der Wächter blind
//	Lauf:     die echten Läufe, ohne gefälschtes Datum
//	nachher:  kein Lauf meldet mehr etwas                          → sonst fragt er etwas anderes als der Lauf
//
// Beide Richtungen zählen. Ein Wächter, der nach dem Lauf noch meldet, schlüge jede Nacht
// falschen Alarm, und ein falscher Alarm wird abgeschaltet, nicht gelesen.
func TestLoeschRueckstand_WaechterUndLaufStellenDieselbeFrage(t *testing.T) {
	adminDSN := os.Getenv(datenbankVariable)
	if adminDSN == "" {
		t.Skipf("%s nicht gesetzt — Test übersprungen", datenbankVariable)
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

	zustand := repository.NewBetriebszustandAbfragen(pool)
	vorher := rueckstandAlsMap(ctx, t, zustand)

	// Richtung 1: Sieht der Wächter überhaupt etwas? Ein Wächter, der nie ausschlägt,
	// ist von einem funktionierenden System nicht zu unterscheiden.
	for lauf, zeilen := range vorher {
		if zeilen == 0 {
			t.Errorf("Wächter blind: %q meldet 0, obwohl ein überfälliger Datensatz existiert", lauf)
		}
	}
	if len(vorher) == 0 {
		t.Fatal("Wächter lieferte keine einzige Routine")
	}

	// Der Lauf, ohne gefälschtes Datum und ohne gesetzte Frist: die Vorgaben. Gefahren wird, was
	// der Cron plant (Starte in cron.go): die Folge der Nacht und die Aufbewahrung des Protokolls um
	// 03:00. Riefe der Test jeden Lauf einzeln auf, bliebe ein Lauf, der in der Folge der Nacht
	// fehlt, grün und fiele erst im Betrieb als Rückstand auf.
	z := NewZeitplan(pool, repository.NewProtokollAbfragen(pool))
	z.fuehreNaechtlicheDSGVOAus(ctx)
	z.fuehreProtokollAufbewahrungAus(ctx)

	// Richtung 2: Ist danach Ruhe? Jede verbleibende Zeile heißt, dass der Wächter eine andere
	// Frage stellt als der Lauf, und genau gegen diese Klasse von Fehlern steht die Datei.
	for lauf, zeilen := range rueckstandAlsMap(ctx, t, zustand) {
		if zeilen > 0 {
			t.Errorf("nach dem Lauf meldet %q noch %d Zeilen — Wächter und Job fragen Verschiedenes", lauf, zeilen)
		}
	}
}

// Ein Datensatz, der heute fällig wird, darf den Wächter nicht auslösen: Der Lauf der Nacht
// hatte ihn noch nicht in der Hand. Ohne diese Kulanz stünde die Selbstprüfung jeden Tag
// zwischen Fälligkeit und Mitternacht auf kritisch, und die tägliche Alarm-Mail ginge mit ihr
// los.
func TestLoeschRueckstand_KulanzDecktDieLetzteNacht(t *testing.T) {
	adminDSN := os.Getenv(datenbankVariable)
	if adminDSN == "" {
		t.Skipf("%s nicht gesetzt — Test übersprungen", datenbankVariable)
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
		repository.AnliegenTageVorgabe); err != nil {
		t.Fatalf("Anliegen: %v", err)
	}

	if n := rueckstandAlsMap(ctx, t, repository.NewBetriebszustandAbfragen(pool))["Erledigte Anliegen"]; n != 0 {
		t.Errorf("gerade fällig gewordenes Anliegen wird schon angemahnt (%d) — die Kulanz greift nicht", n)
	}
}

// rueckstandAlsMap liest den Wächter und wirft abgeschaltete Fristen heraus (die sind
// eine Entscheidung der Schule, kein Rückstand).
func rueckstandAlsMap(ctx context.Context, t *testing.T, r *repository.BetriebszustandAbfragen) map[string]int {
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

// legeUeberfaelligeDatenAn baut für jeden Löschlauf genau einen klar überfälligen
// Datensatz. „Klar" heißt: deutlich jenseits von Frist plus Kulanz, damit der Test nicht
// an der Tagesgrenze wackelt.
func legeUeberfaelligeDatenAn(ctx context.Context, t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	muss := func(was, sql string, argumente ...any) {
		t.Helper()
		if _, err := pool.Exec(ctx, sql, argumente...); err != nil {
			t.Fatalf("%s: %v", was, err)
		}
	}
	eins := func(was, sql string, argumente ...any) string {
		t.Helper()
		var id string
		if err := pool.QueryRow(ctx, sql, argumente...).Scan(&id); err != nil {
			t.Fatalf("%s: %v", was, err)
		}
		return id
	}

	// 1. Anonymisierung: seit 200 Tagen im Papierkorb (Frist 180).
	muss("Papierkorb-Schüler", `
		INSERT INTO schueler (barcode_id, vorname, nachname, klasse, abgaenger_jahr, deleted_at)
		VALUES ('S-PAPIERKORB', 'Paula', 'Papierkorb', '9c', 2030, NOW() - interval '200 days')`)

	// 2. Abgänger endgültig löschen: Abgangsjahr vor dem Stichjahr und schon anonymisiert (der
	//	Lauf löscht nur, was die Karenz durchlaufen hat). Damit betrifft ihn nur die Löschung und
	//	nicht auch die Anonymisierung; der Test soll je Lauf eine Aussage machen.
	muss("Abgänger", `
		INSERT INTO schueler (barcode_id, vorname, nachname, klasse, ist_abgaenger, abgaenger_jahr, aktualisiert_am, anonymized_at)
		VALUES ('S-ABGANG', 'Aaron', 'Abgang', '10b', true, $1, NOW(), NOW() - interval '1 day')`,
		repository.AbgaengerStichjahr(time.Now())-1)

	schuelerID := eins("Probeschüler", `SELECT id FROM schueler WHERE barcode_id = 'S-DRILL-1'`)

	// 2a. Gelöschter Kollege seit 200 Tagen im Papierkorb (Frist 180). Dazu einer, den eine
	//	offene Ausleihe hält: Weder Wächter noch Lauf dürfen ihn anfassen; zählte der Wächter
	//     ihn, meldete er nach dem Lauf jede Nacht einen Rückstand, den niemand abbauen kann.
	muss("Kollege im Papierkorb", `
		INSERT INTO leser (art, vorname, nachname, deleted_at)
		VALUES ('lehrkraft', 'Konrad', 'Papierkorb', NOW() - interval '200 days')`)
	gehalten := eins("gehaltener Kollege", `
		INSERT INTO leser (art, vorname, nachname, deleted_at)
		VALUES ('lehrkraft', 'Greta', 'Gehalten', NOW() - interval '200 days') RETURNING id`)
	schuldner := eins("Kollege mit unbezahlter Forderung", `
		INSERT INTO leser (art, vorname, nachname, deleted_at)
		VALUES ('lehrkraft', 'Fritz', 'Forderung', NOW() - interval '200 days') RETURNING id`)
	freihand := eins("Freihand-Titel", `INSERT INTO buecher_titel (titel, signatur) VALUES ('Der Roman', 'Ro Mus') RETURNING id`)
	lmf := eins("LMF-Titel", `INSERT INTO buecher_titel (titel, signatur, ist_lernmittel) VALUES ('Deutschbuch 7', 'LMF-Deutsch 7', true) RETURNING id`)

	// 3./4. Lesehistorie, beide Klassen: je die Ausleihe und die Zeile im Protokoll, die
	//	dieselbe Zuordnung trägt.
	leihe := func(was, titelID, barcode string, tageZurueck int) {
		t.Helper()
		exemplarID := eins("Exemplar "+barcode,
			`INSERT INTO buecher_exemplare (titel_id, barcode_id) VALUES ($1, $2) RETURNING id`, titelID, barcode)
		muss(was, `
			INSERT INTO ausleihen (exemplar_id, schueler_id, ausgeliehen_am, rueckgabe_frist, rueckgabe_am)
			VALUES ($1, $2, NOW() - make_interval(days => $3 + 30), NOW() - make_interval(days => $3 + 9),
			        NOW() - make_interval(days => $3))`, exemplarID, schuelerID, tageZurueck)
		muss(was+"-Protokoll", `
			INSERT INTO audit_log (tabelle, aktion, datensatz_id, timestamp, details)
			VALUES ('ausleihen', 'RETURN', $1, NOW() - make_interval(days => $2),
			        jsonb_build_object('schueler_id', $3::text))`, exemplarID, tageZurueck, schuelerID)
	}
	leihe("Freihand-Ausleihe", freihand, "B-RUECK-F", repository.LesehistorieTageVorgabe+40)
	offenesExemplar := eins("Exemplar des gehaltenen Kollegen",
		`INSERT INTO buecher_exemplare (titel_id, barcode_id) VALUES ($1, 'B-KOLL-OFFEN') RETURNING id`, freihand)
	muss("offene Ausleihe des gehaltenen Kollegen", `
		INSERT INTO ausleihen (exemplar_id, schueler_id, ausgeliehen_am, rueckgabe_frist)
		VALUES ($1, $2, NOW() - interval '210 days', NOW() + interval '30 days')`, offenesExemplar, gehalten)
	muss("unbezahlte Forderung des Kollegen", `
		INSERT INTO schadensfaelle (exemplar_id, schueler_id, beschreibung, betrag)
		VALUES ($1, $2, 'Wasserschaden', 12.50)`,
		eins("Exemplar der Forderung", `INSERT INTO buecher_exemplare (titel_id, barcode_id) VALUES ($1, 'B-KOLL-SCHADEN') RETURNING id`, freihand),
		schuldner)
	leihe("Lernmittel-Ausleihe", lmf, "B-RUECK-L", repository.LesehistorieLernmittelTageVorgabe+70)

	// 4a. Bearbeiter an einer Ausleihe, die vor 20 Tagen zurückkam (Frist 14). Die Ausleihe
	//     gehört niemandem mehr und ist für die Lesehistorie kein Fall.
	bearbeiter := eins("Bearbeiter", `
		INSERT INTO benutzer (vorname, nachname, email, rolle, aktiv)
		VALUES ('Theo', 'Theke', 'theke-paarung@example.org', 'kollegium', true) RETURNING id`)
	muss("Ausleihe mit Bearbeiter", `
		INSERT INTO ausleihen (exemplar_id, ausgeliehen_am, rueckgabe_frist, rueckgabe_am, bearbeiter_id, rueckgabe_bearbeiter_id)
		VALUES ($1, NOW() - interval '50 days', NOW() - interval '29 days', NOW() - interval '20 days', $2, $2)`,
		eins("Exemplar der Ausleihe mit Bearbeiter", `INSERT INTO buecher_exemplare (titel_id, barcode_id) VALUES ($1, 'B-BEARBEITER') RETURNING id`, freihand),
		bearbeiter)

	// 5. Erledigte Anliegen: vor 400 Tagen erledigt (Frist 365).
	muss("Anliegen", `
		INSERT INTO lehrer_anliegen (art, titel_text, kommentar, erstellt_am, erledigt_am)
		VALUES ('wunsch', 'ALT-ERLEDIGT', 'Bitte anschaffen',
		        NOW() - interval '430 days', NOW() - interval '400 days')`)

	// 5a. Erledigte Reservierung eines Klassensatzes: vor 400 Tagen erledigt (dieselbe Frist, 365).
	muss("Klassensatz-Reservierung", `
		INSERT INTO klassensatz_reservierungen (titel_id, klasse, anzahl, erledigt, erstellt_am, erledigt_am)
		VALUES ($1, '7a', 30, true, NOW() - interval '430 days', NOW() - interval '400 days')`, freihand)
	// Dazu eine offene mit altem erledigt_am: Weder Wächter noch Lauf dürfen sie zählen. Fragte
	// der Wächter nur nach erledigt_am, meldete er sie nach dem Lauf weiter.
	muss("wieder geöffnete Reservierung", `
		INSERT INTO klassensatz_reservierungen (titel_id, klasse, anzahl, erledigt, erstellt_am, erledigt_am)
		VALUES ($1, '7a', 30, false, NOW() - interval '430 days', NOW() - interval '400 days')`, freihand)

	// 6. Aufbewahrung des Protokolls: beide Tabellen, jenseits der 24 Monate.
	muss("altes audit_log", `
		INSERT INTO audit_log (tabelle, aktion, datensatz_id, timestamp)
		VALUES ('buecher_titel', 'UPDATE', $1, NOW() - interval '800 days')`, freihand)
	muss("altes audit_logs", `
		INSERT INTO audit_logs (aktion, details, zeitstempel)
		VALUES ('EINSTELLUNG_GEAENDERT', '{}'::jsonb, NOW() - interval '800 days')`)

	// 7. Quittierte Nachbuch-Meldungen: vor 40 Tagen quittiert (Frist höchstens 30).
	muss("Nachbuch-Meldung", `
		INSERT INTO nachbuch_meldungen (idempotency_key, barcode, ergebnis, gescannt_am, erstellt_am, quittiert_am)
		VALUES (gen_random_uuid(), 'B-ALT-NB', 'umgebucht', NOW() - interval '45 days',
		        NOW() - interval '45 days', NOW() - interval '40 days')`)
}

// Wird ein Titel gelöscht, während ein Exemplar verliehen ist, hinterlässt das eine Zeile im
// Protokoll mit dem Namen des Entleihers (inventur/titel_loeschen_spur.go). Die Zeile ist eine
// Kopie von Personendaten wie jede andere Spur einer Ausleihe und muss derselben Frist
// unterliegen.
//
// Trägt die Zeile den Klarnamen, aber keine schueler_id, greift die Befristung nicht, denn ihre
// Bedingung verlangt genau `details ? 'schueler_id'`; der Name stünde dann 24 Monate (bis zum
// Ende der Aufbewahrung des Protokolls) statt der Frist der Lesehistorie. Der Wächter der
// Selbstprüfung ist dafür blind, weil er dieselbe Frage stellt wie der Lauf: Eine geteilte
// Quelle schützt vor Auseinanderlaufen, nicht vor einer Lücke, die beide haben.
func TestLesehistorie_ErreichtDieSpurGeloeschterTitel(t *testing.T) {
	adminDSN := os.Getenv(datenbankVariable)
	if adminDSN == "" {
		t.Skipf("%s nicht gesetzt — Test übersprungen", datenbankVariable)
	}
	_, dsn := legeProbeDatenbankAn(t, adminDSN, "spurfrist")
	befuelleQuelle(t, dsn)

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatalf("Pool: %v", err)
	}
	t.Cleanup(pool.Close)

	var schuelerID string
	if err := pool.QueryRow(ctx, `SELECT id FROM schueler WHERE barcode_id = 'S-DRILL-1'`).Scan(&schuelerID); err != nil {
		t.Fatalf("Probeschüler: %v", err)
	}

	// spur legt die Zeile im Protokoll an, wie titel_loeschen_spur.go sie schreibt. Exemplar und
	// Titel sind zu diesem Zeitpunkt schon gelöscht, deshalb eine freie UUID als datensatz_id: genau
	// die Lage, in der die Frist greifen muss.
	spur := func(barcode string, mitSchuelerID bool) {
		t.Helper()
		if _, err := pool.Exec(ctx, `
			INSERT INTO audit_log (tabelle, aktion, datensatz_id, akteur, timestamp, details)
			VALUES ('ausleihen', 'DELETE', gen_random_uuid(), 'SYSTEM',
			        NOW() - make_interval(days => $1 + 40),
			        jsonb_build_object('barcode_id',$3::text,'titel','Der Zauberberg',
			                           'entleiher','Hans Castorp','schueler_id',$2::text,
			                           'action','titel_geloescht_mit_offener_ausleihe'))`,
			repository.LesehistorieTageVorgabe, schuelerID, barcode); err != nil {
			t.Fatalf("Spur %s: %v", barcode, err)
		}
		if !mitSchuelerID {
			if _, err := pool.Exec(ctx,
				`UPDATE audit_log SET details = details - 'schueler_id' WHERE details->>'barcode_id' = $1`,
				barcode); err != nil {
				t.Fatalf("Spur %s ohne ID: %v", barcode, err)
			}
		}
	}

	spur("B-MIT-ID", true)
	// Die Gegenprobe am Detektor: dieselbe Zeile ohne schueler_id. Sie muss den Namen behalten,
	// sonst bewiese der Test nichts über den Schlüssel, an dem die Frist hängt.
	spur("B-OHNE-ID", false)

	NewZeitplan(pool, repository.NewProtokollAbfragen(pool)).fuehreLesehistorieBefristungAus(ctx)

	lies := func(barcode string) string {
		t.Helper()
		var details string
		if err := pool.QueryRow(ctx,
			`SELECT details::text FROM audit_log WHERE details->>'barcode_id' = $1`, barcode).Scan(&details); err != nil {
			t.Fatalf("Spur %s nach dem Lauf: %v", barcode, err)
		}
		return details
	}

	mitID := lies("B-MIT-ID")
	for _, darfNicht := range []string{"schueler_id", "entleiher", "Hans Castorp"} {
		if strings.Contains(mitID, darfNicht) {
			t.Errorf("Personenbezug %q überlebte die Frist: %s", darfNicht, mitID)
		}
	}
	// Der sachliche Teil muss bleiben, sonst wäre die Spur wertlos: Sie besteht, damit sich ein
	// zurückgebrachtes Buch zuordnen lässt.
	for _, muss := range []string{"B-MIT-ID", "Zauberberg"} {
		if !strings.Contains(mitID, muss) {
			t.Errorf("die Befristung hat auch den Sachbezug %q getilgt: %s", muss, mitID)
		}
	}

	if ohneID := lies("B-OHNE-ID"); !strings.Contains(ohneID, "Hans Castorp") {
		t.Errorf("Gegenprobe am Detektor gescheitert: auch ohne schueler_id wurde getilgt — "+
			"der Test würde einen Rückbau nicht bemerken: %s", ohneID)
	}
}
