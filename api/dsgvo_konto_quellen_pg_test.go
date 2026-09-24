package api

import (
	"context"
	"encoding/json"
	"slices"
	"strings"
	"testing"

	"bibliothek/repository"
)

// Die Wachstums-Ratsche des Zugangskontos (24.09.2026) — das Gegenstück zu
// TestSchuelerFremdschluessel_WachsenNurMitDemPaar für die Tabelle benutzer. Jede Spalte, die
// auf ein Konto zeigt, steht in der Auskunft: als Anfrage oder als selbst bearbeiteter
// Vorgang (entschieden am 24.09.2026, OFFEN.md 5.19).
//
// Zwei Prüfungen:
//  1. dsgvoKontoQuellen deckt sich mit den Fremdschlüsseln auf benutzer im Schema — eine
//     neue Spalte macht den Test rot, bis die Auskunft sie liest.
//  2. Je Quelle ein Eintrag mit dem Konto; die Auskunft zeigt aus jeder etwas und nichts
//     über die betroffene Schülerin. Eine neue Quelle ohne Fall im switch ist rot.
//
// BLINDHEIT: Verweise ohne Fremdschlüssel sieht der Scan nicht — die Klassenleitung
// (klassen_lehrer_mapping, über die Adresse) und die Kontoereignisse (audit_logs.details
// mit ziel_id). Beide prüft TestDsgvoAuskunft_Kollege.
var dsgvoKontoQuellen = []string{
	"audit_log.bearbeiter_id",
	"audit_logs.admin_id",
	"ausleihen.bearbeiter_id",
	"ausleihen.rueckgabe_bearbeiter_id",
	"inventur_sessions.gestartet_von",
	"klassensatz_reservierungen.angefordert_von",
	"lehrer_anliegen.angefordert_von",
	"nachbuch_meldungen.quittiert_von",
	"schadensersatz_bescheide.erstellt_von",
	"schadensfaelle.storniert_von",
}

func TestDsgvoKontoQuellen_DeckenSichMitDemSchema(t *testing.T) {
	pool := pgTestPool(t)
	rows, err := pool.Query(context.Background(), `
		SELECT c.conrelid::regclass::text || '.' || a.attname
		FROM pg_constraint c
		JOIN pg_attribute a ON a.attrelid = c.conrelid AND a.attnum = ANY(c.conkey)
		WHERE c.contype = 'f' AND c.confrelid = 'benutzer'::regclass
		ORDER BY 1`)
	if err != nil {
		t.Fatalf("FK-Scan: %v", err)
	}
	defer rows.Close()
	var imSchema []string
	for rows.Next() {
		var fk string
		if err := rows.Scan(&fk); err != nil {
			t.Fatal(err)
		}
		imSchema = append(imSchema, fk)
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	if len(imSchema) < 5 {
		t.Fatalf("Liveness: nur %d Fremdschlüssel auf benutzer gefunden — der Scan misst nichts mehr", len(imSchema))
	}
	for _, fk := range imSchema {
		if !slices.Contains(dsgvoKontoQuellen, fk) {
			t.Errorf("%s zeigt auf ein Zugangskonto, die Auskunft liest es nicht — in "+
				"repository/dsgvo_konto_vorgaenge.go (oder als Anfrage) aufnehmen und hier eintragen", fk)
		}
	}
	for _, fk := range dsgvoKontoQuellen {
		if !slices.Contains(imSchema, fk) {
			t.Errorf("dsgvoKontoQuellen nennt %s, das Schema kennt diesen Fremdschlüssel nicht (mehr)", fk)
		}
	}
}

func TestDsgvoAuskunft_ZeigtJedeKontoQuelle(t *testing.T) {
	pool := pgTestPool(t)
	resetBestandsdaten(t, pool)
	ctx := context.Background()
	exec := func(sql string, args ...any) {
		t.Helper()
		if _, err := pool.Exec(ctx, sql, args...); err != nil {
			t.Fatalf("%s: %v", strings.TrimSpace(sql)[:40], err)
		}
	}

	var konto, leser string
	if err := pool.QueryRow(ctx, `
		INSERT INTO benutzer (vorname, nachname, email, rolle, aktiv)
		VALUES ('Bea', 'Bibliothek', 'bea@konto-quellen.invalid', 'mitarbeiter', true)
		RETURNING id, leser_id`).Scan(&konto, &leser); err != nil {
		t.Fatalf("Konto anlegen: %v", err)
	}
	// Die Betroffene der bearbeiteten Vorgänge — nichts von ihr darf in Beas Auskunft stehen.
	const schuelerBarcode = "S-QUELLEN-KANARI"
	schuelerin := seedSchueler(t, pool, schuelerBarcode, "Kanarina", "5a")
	titelID := seedMonitorTitel(t, pool, "Kanari-Titel", "Aut Or", false, 0)
	ex1 := exemplar(t, pool, titelID, "EX-KANARI-1", true, "")
	ex2 := exemplar(t, pool, titelID, "EX-KANARI-2", true, "")

	// Je Quelle: Eintrag anlegen und sagen, woran die Auskunft ihn zeigt.
	erwartet := map[string]func(k *repository.DsgvoZugangskonto) bool{}
	vorgang := func(handlung string) func(k *repository.DsgvoZugangskonto) bool {
		return func(k *repository.DsgvoZugangskonto) bool {
			return slices.ContainsFunc(k.EigeneVorgaenge, func(v repository.DsgvoEigenerVorgang) bool {
				return v.Handlung == handlung && v.Zeitpunkt != nil
			})
		}
	}
	anfrage := func(art string) func(k *repository.DsgvoZugangskonto) bool {
		return func(k *repository.DsgvoZugangskonto) bool {
			return slices.ContainsFunc(k.Anfragen, func(a repository.DsgvoAnfrage) bool { return a.Art == art })
		}
	}
	for _, quelle := range dsgvoKontoQuellen {
		switch quelle {
		case "ausleihen.bearbeiter_id":
			exec(`INSERT INTO ausleihen (exemplar_id, schueler_id, ausgeliehen_am, rueckgabe_frist, bearbeiter_id)
				VALUES ($1, $2, NOW() - interval '2 days', NOW() + interval '12 days', $3)`, ex1, schuelerin, konto)
			erwartet[quelle] = vorgang("Ausleihe gebucht")
		case "ausleihen.rueckgabe_bearbeiter_id":
			exec(`INSERT INTO ausleihen (exemplar_id, schueler_id, ausgeliehen_am, rueckgabe_frist, rueckgabe_am, rueckgabe_bearbeiter_id)
				VALUES ($1, $2, NOW() - interval '9 days', NOW() + interval '5 days', NOW() - interval '1 day', $3)`, ex2, schuelerin, konto)
			erwartet[quelle] = vorgang("Rückgabe gebucht")
		case "schadensfaelle.storniert_von":
			exec(`INSERT INTO schadensfaelle (exemplar_id, schueler_id, beschreibung, betrag, ist_bezahlt,
					storniert_am, storniert_von, stornierungsgrund)
				VALUES ($1, $2, 'Kanari-Schaden', 7.77, false, NOW(), $3, 'Kanari-Stornogrund')`, ex1, schuelerin, konto)
			erwartet[quelle] = vorgang("Schadensfall storniert")
		case "schadensersatz_bescheide.erstellt_von":
			exec(`INSERT INTO schadensersatz_bescheide (schueler_id, mittel, kassenjahr, laufende_nr, referenznummer,
					frist_bis, gesamtbetrag, empfaenger_snapshot, erstellt_von)
				VALUES ($1, 'land', 2026, 1, '5830 2026 9999 0001', CURRENT_DATE + 28, 7.77,
					jsonb_build_object('name', 'Kanarina Eltern'), $2)`, schuelerin, konto)
			erwartet[quelle] = vorgang("Schadensersatz-Bescheid erstellt")
		case "inventur_sessions.gestartet_von":
			exec(`INSERT INTO inventur_sessions (scope_type, scope_label, gestartet_von)
				VALUES ('global', 'Gesamtbestand', $1)`, konto)
			erwartet[quelle] = vorgang("Inventur begonnen: Gesamtbestand")
		case "nachbuch_meldungen.quittiert_von":
			exec(`INSERT INTO nachbuch_meldungen (idempotency_key, barcode, ergebnis, grund,
					ausleiher_schueler_id, gescannt_am, quittiert_am, quittiert_von)
				VALUES (gen_random_uuid(), 'EX-KANARI-1', 'umgebucht', 'Kanari-Grund', $1, NOW(), NOW(), $2)`, schuelerin, konto)
			erwartet[quelle] = vorgang("Meldung nach Netzausfall bearbeitet")
		case "audit_log.bearbeiter_id":
			exec(`INSERT INTO audit_log (tabelle, aktion, datensatz_id, bearbeiter_id, akteur, details)
				VALUES ('ausleihen', 'CHECKOUT', $1, $2, 'USER',
					jsonb_build_object('schueler_id', $3::text, 'entleiher', 'Kanarina Kanari'))`, ex1, konto, schuelerin)
			erwartet[quelle] = vorgang("Protokolleintrag: CHECKOUT (ausleihen)")
		case "audit_logs.admin_id":
			exec(`INSERT INTO audit_logs (admin_id, aktion, details, ip_adresse)
				VALUES ($1, 'OVERRIDE_BLOCK', jsonb_build_object('schueler_id', $2::text, 'reason', 'Kanari-Sperrgrund'), '10.1.2.3')`,
				konto, schuelerin)
			erwartet[quelle] = func(k *repository.DsgvoZugangskonto) bool {
				return slices.ContainsFunc(k.EigeneVorgaenge, func(v repository.DsgvoEigenerVorgang) bool {
					return v.Handlung == "Verwaltungseingriff: OVERRIDE_BLOCK" && v.IPAdresse != nil && *v.IPAdresse == "10.1.2.3"
				})
			}
		case "lehrer_anliegen.angefordert_von":
			exec(`INSERT INTO lehrer_anliegen (art, titel_text, angefordert_von) VALUES ('meldung', 'Kanari-Meldung', $1)`, konto)
			erwartet[quelle] = anfrage("meldung")
		case "klassensatz_reservierungen.angefordert_von":
			// Ein eigener Titel: Der Titel der eigenen Reservierung gehört in Beas Auskunft,
			// der Titel, den die Schülerin ausgeliehen hat, nicht.
			eigenerTitel := seedMonitorTitel(t, pool, "Beas Klassensatz-Titel", "Aut Or", false, 0)
			exec(`INSERT INTO klassensatz_reservierungen (titel_id, klasse, anzahl, angefordert_von)
				VALUES ($1, '5a', 25, $2)`, eigenerTitel, konto)
			erwartet[quelle] = anfrage("klassensatz")
		default:
			t.Fatalf("keine Probe für die Kontoquelle %s — Test und Auskunft nachziehen", quelle)
		}
	}

	k, err := repository.LeseDsgvoZugangskonto(ctx, pool, leser)
	if err != nil || k == nil {
		t.Fatalf("Auskunft lesen: %v (Konto %v)", err, k)
	}
	for _, quelle := range dsgvoKontoQuellen {
		if !erwartet[quelle](k) {
			t.Errorf("die Auskunft zeigt nichts aus %s, obwohl dort ein Eintrag mit dem Konto liegt", quelle)
		}
	}

	roh, err := json.Marshal(k)
	if err != nil {
		t.Fatal(err)
	}
	for _, dritte := range []string{schuelerin, schuelerBarcode, "Kanarina", "EX-KANARI", "Kanari-Titel",
		"Kanari-Schaden", "Kanari-Stornogrund", "5830 2026 9999", "Kanari-Grund", "Kanari-Sperrgrund", "7.77"} {
		if strings.Contains(string(roh), dritte) {
			t.Errorf("Beas Auskunft enthält %q — das sind Daten der betroffenen Schülerin", dritte)
		}
	}
}
