package repository_test

import (
	"context"
	"sort"
	"strings"
	"testing"

	"bibliothek/internal/pgtest"
)

// Frage 12 „Gegenrichtung Schema" als Ratsche (06.09.2026, docs/invarianten.md).
//
// Die anderen elf Fragen sehen vom Code aus auf die Daten. Diese sieht umgekehrt: Was
// TUT die Datenbank, das im Code nirgends steht? Der Anlass sind zwei Funde desselben
// Abends, beide von der DDL und nicht vom Lesen des Go-Codes:
//
//   - `CONSTRAINT check_damage_item` verlangt genau eines von `exemplar_id`/`geraet_id`.
//     Der Geräteschaden ist also vorgesehen — die Rechnung an die Eltern verband per
//     INNER JOIN und hätte ihn verloren (ce875654).
//   - An `buecher_titel` hängen vier Kinder mit ON DELETE CASCADE; drei davon erwähnte
//     kein Löschpfad, und dahinter standen wartende Schüler und Lehrkräfte (77bbf931).
//
// Beides war jahrelang lesbar und wurde nie gelesen, weil niemand die DDL befragt hat.
//
// Diese Ratsche ist bewusst ein ÄNDERUNGS-Melder, kein Urteil: Sie friert ein, was die
// Datenbank heute tut. Kommt eine Zeile dazu — ein neuer CASCADE, eine neue Bedingung,
// ein neuer Trigger —, wird sie rot und verlangt die Frage: Wer behandelt die Folge?
// Der Bestand ist damit eine Arbeitsliste; die Einträge, die schon befragt sind, tragen
// ihre Antwort als Kommentar.
//
// Bewusst NICHT eingefroren: die 96 nullbaren Spalten. Ihre Gefahr (nullbare Spalte in
// nicht-nullbaren Go-Typ) ist die Bugklasse „NULL-Scan", und die hat ihre eigene
// Antwort — echte Postgres-Tests je Lesepfad. Eine Liste von 96 Namen wäre der
// Dateibaum und würde nichts aussagen (dieselbe Begründung wie bei der Farb-Ratsche).

// Fremdschlüssel, die beim Löschen etwas TUN. Jede Zeile ist eine Folge, die jemand
// behandeln muss — oder eine bewusste Entscheidung, sie nicht zu behandeln.
var fkAktionenBestand = []string{
	// Befragt am 06.09.2026: Diese drei fielen lautlos; jetzt protokolliert sie
	// repository/titel_loeschen_wartende.go, bevor der CASCADE sie nimmt (77bbf931).
	"CASCADE  class_books.book_id -> buecher_titel",
	"CASCADE  klassensatz_reservierungen.titel_id -> buecher_titel",
	"CASCADE  vormerkungen.titel_id -> buecher_titel",
	// Befragt: Der Löschpfad räumt die abhängigen Zeilen selbst und in Reihenfolge ab;
	// der CASCADE auf die Exemplare ist der letzte Schritt.
	"CASCADE  buecher_exemplare.titel_id -> buecher_titel",
	// Befragt (Register 06.09.2026): „Plan verwerfen" löscht den ganzen Plan samt
	// Zeilen, freien Tagen und Auslassungen — bekannt und als Produktfrage notiert.
	"CASCADE  lmf_plan_ausgelassen.plan_id -> lmf_plaene",
	"CASCADE  lmf_plan_freie_tage.plan_id -> lmf_plaene",
	"CASCADE  lmf_termin_klassen.termin_id -> lmf_termine",
	"CASCADE  lmf_termine.plan_id -> lmf_plaene",
	// Befragt am 06.09.2026: Die Erfassungen fielen mit dem Exemplar und senkten damit
	// rückwirkend das Ergebnis abgeschlossener Durchgänge — neben einem
	// verloren_gemeldet, das feststand. Migration 103 friert die Zahl beim Abschluss ein.
	"CASCADE  inventur_erfassungen.exemplar_id -> buecher_exemplare",
	// Noch nicht befragt — beim nächsten Anfassen des jeweiligen Pfades.
	"CASCADE  bestellungen_positionen.bestellung_id -> bestellungen_verlauf",
	"CASCADE  inventur_erfassungen.session_id -> inventur_sessions",
	"CASCADE  inventur_verluste.session_id -> inventur_sessions",
	// Befragt am 06.09.2026: Das Foto ist PII und MUSS mit dem Kind fallen; beim
	// Zusammenführen wandert es vorher (das jüngere gewinnt, schueler_zusammenfuehren.go).
	"CASCADE  schueler_fotos.schueler_id -> schueler",
	// Befragt am 06.09.2026: Hier ist der CASCADE nur das Netz — die Spuren-Tilgung
	// löscht die Vormerkungen selbst (Freitext-Notiz ist PII). Ihr fehlte das
	// Nachrücken: Ein bereits abholbereit gelegtes Exemplar blieb liegen, statt an den
	// nächsten Wartenden zu gehen (vormerkung_nachruecken.go).
	"CASCADE  vormerkungen.schueler_id -> schueler",
	// Befragt: Ein gelöschter Benutzer soll seine Spuren behalten, nur ohne Person —
	// deshalb SET NULL statt RESTRICT. Die Lesepfade zeigen dann „unbekannt".
	"SET NULL  audit_log.bearbeiter_id -> benutzer",
	"SET NULL  audit_logs.admin_id -> benutzer",
	"SET NULL  ausleihen.ausleiher_benutzer_id -> benutzer",
	"SET NULL  ausleihen.bearbeiter_id -> benutzer",
	"SET NULL  ausleihen.rueckgabe_bearbeiter_id -> benutzer",
	"SET NULL  inventur_sessions.gestartet_von -> benutzer",
	"SET NULL  klassensatz_reservierungen.angefordert_von -> benutzer",
	"SET NULL  lehrer_anliegen.angefordert_von -> benutzer",
	"SET NULL  schadensfaelle.benutzer_id -> benutzer",
	"SET NULL  schadensfaelle.storniert_von -> benutzer",
	// Befragt am 06.09.2026: LEFT JOIN mit ausdrücklicher Begründung im Code
	// (bestelldetail_repo.go), beide Geschwister-Pfade halten es genauso.
	"SET NULL  bestellungen_positionen.titel_id -> buecher_titel",
	// Befragt: Der Fehlbestandsbericht hält eine Abschrift, gerade damit er das Löschen
	// des Exemplars überlebt (inventur_verlust_aktionen.go).
	"SET NULL  inventur_verluste.exemplar_id -> buecher_exemplare",
	// Befragt: Die Vormerkung fällt zurück auf „wartend", wenn ihr bereitgestelltes
	// Exemplar verschwindet (repository/damage.go).
	"SET NULL  vormerkungen.bereitgestellt_exemplar_id -> buecher_exemplare",
	// Befragt am 07.09.2026: Die harmlose Hälfte — die Bestellung hält Name und E-Mail
	// des Händlers als eigene Abschrift, die Historie überlebt das Löschen unbeschadet
	// (bestellbestaetigung_handler.go COALESCEt ausdrücklich dagegen). Der LÖSCHWEG
	// daneben war der Fund: Er nahm auch den Hauptlieferanten, an dem Bestellmail,
	// Bestätigungs-Link und Etiketten-Verhalten hängen — ohne Rückfrage und ohne Meldung.
	// Seit 07.09. weist der Handler das ab (api/supplier_handler.go).
	"SET NULL  bestellungen_verlauf.lieferant_id -> lieferanten",
	"SET NULL  buecher_exemplare.bestellung_id -> bestellungen_verlauf",
	"SET NULL  lehrer_anliegen.titel_id -> buecher_titel",
	// Befragt am 06.09.2026: Genau dieser SET NULL machte die Forderung in der Rechnung
	// unsichtbar, solange dort INNER JOIN stand (ce875654).
	"SET NULL  schadensfaelle.ausleihe_id -> ausleihen",
}

// Bedingungen, die die Datenbank durchsetzt. Der Code muss sie kennen — sonst schreibt
// er Daten, die abgewiesen werden, oder er verlässt sich auf eine Form, die die
// Datenbank ausdrücklich zulässt (check_damage_item war genau das).
var checkBedingungenBestand = []string{
	"audit_log_akteur_check", "bestellungen_verlauf_bestaetigt_durch_check",
	"bestellungen_verlauf_etiketten_groesse_check", "check_damage_item",
	"check_damage_responsible", "check_loan_borrower", "check_loan_item",
	"check_positive_amount", "check_return_date", "chk_anliegen_art",
	"chk_aussonderung_grund", "chk_cover_status", "chk_einkaufspreis_nonneg",
	"chk_exemplar_bestellstatus", "chk_grade_level_bereich", "chk_inv_session_scope",
	"chk_ksr_anzahl_positiv", "chk_lmf_plaene_anker", "chk_lmf_plaene_art",
	"chk_lmf_plaene_letzte_stunde", "chk_lmf_plaene_startstunde", "chk_lmf_plaene_stunden",
	"chk_lmf_termine_art", "chk_lmf_termine_stunde", "chk_meldebestand_nonneg",
	"chk_pos_einzelpreis_nonneg", "chk_pos_menge_positiv", "chk_schueler_block_reason",
	"chk_verlauf_anzahl_nonneg", "chk_verlauf_gesamtbetrag_nonneg", "chk_vormerkung_status",
	"mail_settings_config_single_row_chk",
}

// Trigger ändern Daten, ohne dass eine Zeile Go-Code davon weiß. Die neunzehn hier sind
// zwei Sorten: `aktualisiert_am`-Stempel und die Klassen-Normalisierung (Migration 087,
// „05F1" statt „5f1"). Ein NEUER Trigger ist immer eine Frage.
var triggerBestand = []string{
	"trg_benutzer_aktualisiert_am @ benutzer",
	"trg_buecher_exemplare_aktualisiert_am @ buecher_exemplare",
	"trg_buecher_titel_aktualisiert_am @ buecher_titel",
	"trg_class_books_vokabular @ class_books",
	"trg_klassen_aktualisiert_am @ klassen",
	"trg_klassen_anzeigeform @ klassen",
	"trg_klm_klasse_vokabular @ klassen_lehrer_mapping",
	"trg_ksr_klasse_vokabular @ klassensatz_reservierungen",
	"trg_lesergruppen_aktualisiert_am @ lesergruppen",
	"trg_lmf_plaene_aktualisiert_am @ lmf_plaene",
	"trg_lmf_plan_ausgelassen_vokabular @ lmf_plan_ausgelassen",
	"trg_lmf_termin_klassen_vokabular @ lmf_termin_klassen",
	"trg_lmf_termine_aktualisiert_am @ lmf_termine",
	"trg_mail_vorlagen_updated_at @ mail_vorlagen",
	"trg_schadensfaelle_aktualisiert_am @ schadensfaelle",
	"trg_schueler_aktualisiert_am @ schueler",
	"trg_schueler_fotos_aktualisiert_am @ schueler_fotos",
	"trg_schueler_klasse_vokabular @ schueler",
	"trg_systematik_kategorien_aktualisiert_am @ systematik_kategorien",
}

func leseSchemaFakten(t *testing.T, sql string) []string {
	t.Helper()
	pool := pgtest.Pool(t)
	rows, err := pool.Query(context.Background(), sql)
	if err != nil {
		t.Fatalf("Schema lesen: %v", err)
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var s string
		if err := rows.Scan(&s); err != nil {
			t.Fatalf("Schema lesen: %v", err)
		}
		out = append(out, s)
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("Schema lesen: %v", err)
	}
	sort.Strings(out)
	return out
}

// vergleiche meldet in BEIDE Richtungen: Neues, das niemand befragt hat, und Einträge,
// die es nicht mehr gibt (eine Liste, die Erledigtes weiterführt, verliert ihre Aussage).
func vergleiche(t *testing.T, was string, ist, bestand []string, hinweis string) {
	t.Helper()
	if len(ist) == 0 {
		t.Fatalf("%s: die Abfrage liefert nichts — dieses Gate wäre still grün", was)
	}
	drin := map[string]bool{}
	for _, b := range bestand {
		drin[b] = true
	}
	var neu []string
	for _, i := range ist {
		if !drin[i] {
			neu = append(neu, i)
		}
	}
	hat := map[string]bool{}
	for _, i := range ist {
		hat[i] = true
	}
	var weg []string
	for _, b := range bestand {
		if !hat[b] {
			weg = append(weg, b)
		}
	}
	if len(neu) > 0 {
		t.Errorf("%s — neu in der Datenbank:\n  %s\n\n%s", was, strings.Join(neu, "\n  "), hinweis)
	}
	if len(weg) > 0 {
		t.Errorf("%s — steht im Bestand, gibt es aber nicht (mehr):\n  %s\n\nEintrag austragen.",
			was, strings.Join(weg, "\n  "))
	}
}

func TestSchemaGegenrichtung_FremdschluesselAktionen(t *testing.T) {
	ist := leseSchemaFakten(t, `
		SELECT rc.delete_rule||'  '||tc.table_name||'.'||kcu.column_name||' -> '||ccu.table_name
		FROM information_schema.referential_constraints rc
		JOIN information_schema.table_constraints tc ON tc.constraint_name = rc.constraint_name
		JOIN information_schema.key_column_usage kcu ON kcu.constraint_name = rc.constraint_name
		JOIN information_schema.constraint_column_usage ccu ON ccu.constraint_name = rc.constraint_name
		WHERE rc.delete_rule IN ('CASCADE','SET NULL')`)
	vergleiche(t, "Fremdschlüssel mit Löschwirkung", ist, fkAktionenBestand,
		"Diese Zeilen verschwinden oder werden geleert, wenn ihr Bezug fällt — ohne dass "+
			"eine Zeile Go-Code davon weiß. Frage 12: Wer behandelt die Folge? Wer merkt es? "+
			"Steht sie im Protokoll? Dann hier mit der Antwort eintragen.")
}

func TestSchemaGegenrichtung_CheckBedingungen(t *testing.T) {
	ist := leseSchemaFakten(t, `
		SELECT conname FROM pg_constraint
		WHERE contype = 'c' AND connamespace = 'public'::regnamespace`)
	vergleiche(t, "CHECK-Bedingungen", ist, checkBedingungenBestand,
		"Die Datenbank setzt hier eine Regel durch. Frage 12: Kennt der Code sie — und "+
			"verlässt er sich umgekehrt auf etwas, das sie gar nicht verbietet? "+
			"(check_damage_item erlaubt den Geräteschaden ausdrücklich; die Rechnung an die "+
			"Eltern hat ihn trotzdem verloren.)")
}

func TestSchemaGegenrichtung_Trigger(t *testing.T) {
	ist := leseSchemaFakten(t, `
		SELECT DISTINCT trigger_name||' @ '||event_object_table
		FROM information_schema.triggers WHERE trigger_schema = 'public'`)
	vergleiche(t, "Trigger", ist, triggerBestand,
		"Ein Trigger ändert Daten, ohne dass eine Zeile Go-Code davon weiß. Frage 12: "+
			"Was schreibt er um, und rechnet ein Lesepfad noch mit dem Wert von vorher?")
}
