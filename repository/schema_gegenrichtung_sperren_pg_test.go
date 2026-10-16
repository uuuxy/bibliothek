package repository_test

import "testing"

// Frage 12 „Gegenrichtung Schema", zweite Hälfte: was die Datenbank ablehnt und was sie beim
// Umbenennen von selbst weiterträgt (docs/invarianten.md).
//
// schema_gegenrichtung_pg_test.go friert ein, was beim Löschen von selbst geschieht (CASCADE,
// SET NULL), dazu Trigger und CHECK-Bedingungen. Drei Arten von Regeln stehen dort nicht:
//
//   - Fremdschlüssel, an denen ein Löschen scheitert (RESTRICT). Für den Menschen an der Maske
//     endet das in einer Ablehnung (23503); ob daraus ein Satz wird, der sagt, was im Weg
//     steht, oder der neutrale Satz von apierrors, steht in keiner DDL.
//   - Fremdschlüssel, die ein Umbenennen weitertragen (ON UPDATE CASCADE). Der neue Name steht
//     danach in Zeilen, die kein Go-Code angefasst hat.
//   - Eindeutigkeit: UNIQUE-Bedingungen und eindeutige Indizes, auch die mit Bedingung
//     (Teilindex). Der zweite gleiche Wert wird abgelehnt (23505).
//
// Wie dort ein Änderungs-Melder, kein Urteil: Der Bestand ist eine Arbeitsliste, befragte
// Einträge tragen ihre Antwort als Kommentar.
//
// Sieht nicht: Primärschlüssel und Teilindizes ohne Eindeutigkeit (sie beschleunigen nur,
// über Daten entscheiden sie nicht). Ob ein Schreiber die Ablehnung behandelt, liest ein
// Mensch; halten muss es ein eigener Test. Den Wortlaut einer Bedingung schreibt Postgres in
// seiner eigenen Form aus — nach einem Wechsel der Hauptversion kann der Bestand neu
// einzutragen sein, ohne dass sich eine Regel geändert hat.

// Fremdschlüssel, an denen ein Löschen scheitert.
var fkSperrenBestand = []string{
	// Befragt: Zwei Wege löschen Exemplare und räumen vorher ab. Das Löschen eines Titels lehnt
	// verliehene Exemplare mit Satz ab (400 mit den Barcodes, repository.LoescheTitel); die
	// Massenaktion der Bestandstabelle löscht auch die offenen Ausleihen, mit Spur im Protokoll
	// (inventur.LoescheTitel). Das Löschen von Verlusten weist offene Vorgänge mit Satz ab (409,
	// ErrVerlustNochGebunden). Danach fallen Schadensfälle und zurückgegebene Ausleihen in
	// derselben Transaktion. Die Sperre trifft nur eine Ausleihe oder Forderung, die zwischen
	// Abräumen und Löschen entsteht: Dann bricht das Löschen mit 500 ab, mit dem neutralen Satz
	// von apierrors, an der Massenaktion mit deren festem Satz.
	"RESTRICT  ausleihen.exemplar_id -> buecher_exemplare",
	// Befragt: Kein Schreiber löscht ein Gerät, es gibt keine Route dafür; die Oberfläche führt
	// es höchstens als nicht ausleihbar (repository/geraete.go). Die Sperre trifft nur ein
	// Löschen von Hand.
	"RESTRICT  ausleihen.geraet_id -> geraete",
	// Befragt: Drei Wege löschen Leser und räumen vorher ab. Das endgültige Löschen aus dem
	// Papierkorb und das der Abgänger lehnen offene Ausleihen und unbezahlte Forderungen ab
	// (LoeschenBlockiertFehler, an der Tür 409 mit Satz), nehmen den übrigen Ausleihen den Leser
	// und löschen die erledigten Schadensfälle (tilgeSpurenUndLoescheLeser). Die Zusammenführung
	// sperrt beide Zeilen und verschiebt die Vorgänge vorher (FuehreLeserZusammen). Das Löschen
	// eines Kontos entfernt die Leserzeile nur, wenn kein Fremdschlüssel auf sie zeigt
	// (loescheUnberuehrteLeserzeile). Die Sperre trifft nur eine Ausleihe, die zwischen Prüfung
	// und Löschen entsteht: Dann bricht das Löschen ab, an der Tür mit 500 und dem neutralen
	// Satz von apierrors.
	"RESTRICT  ausleihen.schueler_id -> leser",
	// Befragt: Das Löschen einer Sachgruppe zählt vorher die Titel am Fach und antwortet mit
	// 409 und einem Satz; hängt sich zwischen Zählung und Löschen ein Titel an, fängt der
	// Handler die Ablehnung (23503) mit demselben Satz (api/katalog/systematik.go).
	"RESTRICT  buecher_titel.subject -> systematik_kategorien",
	// Befragt, für die fünf Bezüge auf klassen: Kein Schreiber im Programm löscht eine Klasse
	// oder benennt sie um; umbenannt hat nur Migration 087, einmal. Das Vokabular füllen die
	// Trigger der Tabellen (Migration 079), die Versetzung hängt Schüler an andere Namen. Die
	// Sperre trifft nur ein Löschen von Hand.
	"RESTRICT  class_books.class_name -> klassen",
	"RESTRICT  klassen_lehrer_mapping.klasse -> klassen",
	"RESTRICT  leser.klasse -> klassen",
	"RESTRICT  lmf_plan_ausgelassen.klasse -> klassen",
	"RESTRICT  lmf_termin_klassen.klasse -> klassen",
	// Befragt: wie ausleihen.exemplar_id. Beide Wege löschen die Schadensfälle vor den
	// Exemplaren, das Löschen eines Titels auch die unbezahlten, mit Spur im Protokoll.
	"RESTRICT  schadensfaelle.exemplar_id -> buecher_exemplare",
	// Befragt: wie ausleihen.geraet_id.
	"RESTRICT  schadensfaelle.geraet_id -> geraete",
	// Befragt: wie ausleihen.schueler_id. Die Zusammenführung verschiebt die Schadensfälle, das
	// endgültige Löschen löscht die erledigten.
	"RESTRICT  schadensfaelle.schueler_id -> leser",
}

// Fremdschlüssel, die eine Änderung des Schlüssels weitertragen.
var fkAenderungenBestand = []string{
	// Befragt: Umbenannt wird über SachgruppeAendern (PUT /api/systematics/{id}); die Antwort
	// nennt die mitgezogenen Titel (titel_mitgezogen). Einen Protokolleintrag gibt es nicht, wie
	// bei jeder Änderung am Katalog; dort stehen nur Löschungen. Die einzige Abschrift des Fachs
	// als Text, den Bereich offener Inventuren (scope_subject), stellt die Tür in derselben
	// Transaktion um (BenenneFachOffenerInventurenUm); abgeschlossene Läufe und die Signaturen
	// behalten den alten Wert mit Absicht.
	"CASCADE  buecher_titel.subject -> systematik_kategorien",
	// Befragt, für die fünf Bezüge auf klassen: Kein Schreiber im Programm benennt eine Klasse
	// um; umbenannt hat nur Migration 087, einmal.
	"CASCADE  class_books.class_name -> klassen",
	"CASCADE  klassen_lehrer_mapping.klasse -> klassen",
	"CASCADE  leser.klasse -> klassen",
	"CASCADE  lmf_plan_ausgelassen.klasse -> klassen",
	"CASCADE  lmf_termin_klassen.klasse -> klassen",
}

// Eindeutigkeit: Name des Index, Tabelle, Spalten oder Ausdrücke und die Bedingung eines
// Teilindex.
var eindeutigkeitBestand = []string{
	"benutzer_email_key @ benutzer (email)",
	"buecher_exemplare_barcode_id_key @ buecher_exemplare (barcode_id)",
	"buecher_titel_isbn_key @ buecher_titel (isbn)",
	"geraete_barcode_id_key @ geraete (barcode_id)",
	"geraete_seriennummer_key @ geraete (seriennummer)",
	"idx_bestellungen_token_hash @ bestellungen_verlauf (bestaetigungs_token_hash) WHERE (bestaetigungs_token_hash IS NOT NULL)",
	"idx_inv_session_offen_filter @ inventur_sessions (COALESCE(scope_subject, ''::text), COALESCE((scope_grade)::integer, '-1'::integer)) WHERE ((abgeschlossen_am IS NULL) AND ((scope_type)::text = 'filter'::text))",
	"idx_inv_session_offen_global @ inventur_sessions ((true)) WHERE ((abgeschlossen_am IS NULL) AND ((scope_type)::text = 'global'::text))",
	"idx_inv_session_offen_signature @ inventur_sessions (btrim(scope_signatur)) WHERE ((abgeschlossen_am IS NULL) AND ((scope_type)::text = 'signature'::text))",
	"idx_inventur_verluste_einmalig @ inventur_verluste (session_id, exemplar_id) WHERE (exemplar_id IS NOT NULL)",
	"idx_lieferanten_ein_hauptlieferant @ lieferanten (ist_hauptlieferant) WHERE ist_hauptlieferant",
	"klassen_name_key @ klassen (name)",
	"mail_vorlagen_typ_key @ mail_vorlagen (typ)",
	"schadensersatz_bescheide_referenznummer_key @ schadensersatz_bescheide (referenznummer)",
	"subjects_name_key @ subjects (name)",
	"systematik_kategorien_kuerzel_key @ systematik_kategorien (kuerzel)",
	"uniq_ausleihen_aktiv_exemplar @ ausleihen (exemplar_id) WHERE ((rueckgabe_am IS NULL) AND (exemplar_id IS NOT NULL))",
	"uniq_ausleihen_aktiv_geraet @ ausleihen (geraet_id) WHERE ((rueckgabe_am IS NULL) AND (geraet_id IS NOT NULL))",
	"uniq_benutzer_email_lower @ benutzer (lower((email)::text))",
	"uniq_benutzer_leser @ benutzer (leser_id) WHERE (leser_id IS NOT NULL)",
	"uniq_bescheid_nummer @ schadensersatz_bescheide (mittel, kassenjahr, laufende_nr)",
	"uniq_bestellung_idempotenz @ bestellungen_verlauf (idempotenz_schluessel) WHERE (idempotenz_schluessel IS NOT NULL)",
	"uniq_klassen_normkey @ klassen (klassen_normkey((name)::text))",
	"uniq_ksr_idempotenz @ klassensatz_reservierungen (idempotenz_schluessel) WHERE (idempotenz_schluessel IS NOT NULL)",
	"uniq_lmf_plaene_art_schuljahr @ lmf_plaene (art, schuljahr_beginn)",
	"uniq_lmf_plaene_id_art @ lmf_plaene (id, art)",
	"uniq_lmf_termine_plan_position @ lmf_termine (plan_id, \"position\")",
	"uniq_nachbuch_meldungen_schluessel @ nachbuch_meldungen (idempotency_key)",
	"uniq_schlagworte_wort @ schlagworte (lower(wort))",
	"uniq_schueler_barcode_active @ leser (barcode_id) WHERE (deleted_at IS NULL)",
	"uniq_schueler_lusd_id_active @ leser (lusd_id) WHERE ((deleted_at IS NULL) AND (lusd_id IS NOT NULL))",
	"uniq_systematik_bezeichnung @ systematik_kategorien (bezeichnung)",
	"uniq_systematik_bezeichnung_ci @ systematik_kategorien (lower((bezeichnung)::text))",
	"unique_schueler_name_gebdatum @ leser (suchnorm((vorname)::text), suchnorm((nachname)::text), geburtsdatum) WHERE ((geburtsdatum IS NOT NULL) AND (deleted_at IS NULL) AND (lusd_id IS NULL))",
	"vormerkungen_titel_id_schueler_id_key @ vormerkungen (titel_id, schueler_id)",
}

func TestSchemaGegenrichtung_FremdschluesselSperren(t *testing.T) {
	ist := leseSchemaFakten(t, `
		SELECT rc.delete_rule||'  '||tc.table_name||'.'||kcu.column_name||' -> '||ccu.table_name
		FROM information_schema.referential_constraints rc
		JOIN information_schema.table_constraints tc ON tc.constraint_name = rc.constraint_name
		JOIN information_schema.key_column_usage kcu ON kcu.constraint_name = rc.constraint_name
		JOIN information_schema.constraint_column_usage ccu ON ccu.constraint_name = rc.constraint_name
		WHERE rc.delete_rule IN ('RESTRICT','NO ACTION')`)
	vergleiche(t, "Fremdschlüssel, an denen ein Löschen scheitert", ist, fkSperrenBestand,
		"An diesem Bezug lehnt die Datenbank ein Löschen ab (23503). Frage 12: Welcher Löschweg "+
			"läuft dagegen, und was bekommt der Mensch — einen Satz, der sagt, was im Weg steht "+
			"(409), oder den neutralen Satz von apierrors? Dann hier mit der Antwort eintragen.")
}

func TestSchemaGegenrichtung_FremdschluesselAenderungen(t *testing.T) {
	ist := leseSchemaFakten(t, `
		SELECT rc.update_rule||'  '||tc.table_name||'.'||kcu.column_name||' -> '||ccu.table_name
		FROM information_schema.referential_constraints rc
		JOIN information_schema.table_constraints tc ON tc.constraint_name = rc.constraint_name
		JOIN information_schema.key_column_usage kcu ON kcu.constraint_name = rc.constraint_name
		JOIN information_schema.constraint_column_usage ccu ON ccu.constraint_name = rc.constraint_name
		WHERE rc.update_rule NOT IN ('RESTRICT','NO ACTION')`)
	vergleiche(t, "Fremdschlüssel, die eine Änderung weitertragen", ist, fkAenderungenBestand,
		"Ändert sich hier der Schlüssel, schreibt die Datenbank ihn in die abhängigen Zeilen — "+
			"ohne dass eine Zeile Go-Code davon weiß. Frage 12: Wer benennt um, steht das "+
			"Umbenennen im Protokoll, und rechnet ein Lesepfad noch mit dem alten Wert?")
}

func TestSchemaGegenrichtung_Eindeutigkeit(t *testing.T) {
	ist := leseSchemaFakten(t, `
		SELECT c.relname||' @ '||t.relname||' '||
		       regexp_replace(pg_get_indexdef(i.indexrelid), '^.* USING [a-z]+ ', '')
		FROM pg_index i
		JOIN pg_class c ON c.oid = i.indexrelid
		JOIN pg_class t ON t.oid = i.indrelid
		JOIN pg_namespace n ON n.oid = t.relnamespace
		WHERE n.nspname = 'public' AND (i.indisunique OR i.indisexclusion) AND NOT i.indisprimary`)
	vergleiche(t, "Eindeutigkeit", ist, eindeutigkeitBestand,
		"Die Datenbank lehnt hier den zweiten gleichen Wert ab (23505). Frage 12: Welche "+
			"Schreiber können dagegen laufen, und was bekommt jeder — einen Satz (409) oder einen "+
			"Abbruch? Bei einem Teilindex: Für welche Zeilen gilt die Regel nicht, und verlässt "+
			"sich ein Lesepfad trotzdem darauf?")
}
