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
// Wie dort ein Änderungs-Melder, kein Urteil: Jeder Eintrag trägt seine Antwort als Kommentar,
// ein neuer braucht sie ebenso.
//
// Sieht nicht: Primärschlüssel und Teilindizes ohne Eindeutigkeit (sie beschleunigen nur,
// über Daten entscheiden sie nicht). Ob ein Schreiber die Ablehnung behandelt, liest ein
// Mensch; halten muss es ein eigener Test. Den Wortlaut einer Bedingung schreibt Postgres in
// seiner eigenen Form aus — nach einem Wechsel der Hauptversion kann der Bestand neu
// einzutragen sein, ohne dass sich eine Regel geändert hat.

// Fremdschlüssel, an denen ein Löschen scheitert.
var fkSperrenBestand = []string{
	// Befragt: Drei Wege löschen Exemplare und räumen vorher ab. Das Löschen eines Titels lehnt
	// verliehene Exemplare mit Satz ab (400 mit den Barcodes, repository.LoescheTitel); die
	// Massenaktion der Bestandstabelle löscht auch die offenen Ausleihen, mit Spur im Protokoll
	// (TitelAbfragen.LoescheTitelSamtAusleihen). Das Löschen von Verlusten weist offene Vorgänge mit Satz ab (409,
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
	// Befragt: wie ausleihen.exemplar_id. Alle drei Wege löschen die Schadensfälle vor den
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
	// Befragt, mit uniq_benutzer_email_lower: Die Benutzerverwaltung prüft die Adresse vorher ohne
	// Rücksicht auf die Schreibweise (pruefeEmailEindeutig, 400 mit Satz), die Leserdatei fängt
	// eine belegte Adresse beim Anlegen eines Kollegen ab (antworteAufKontoFehler, 409). Der Antrag
	// auf Zugang schreibt die Adresse klein und übergeht eine belegte (LegeZugangsanfrageAn, ON
	// CONFLICT DO NOTHING); die Übernahme aus Littera vergibt je Person eine Platzhalter-Adresse.
	// Legen zwei Plätze zugleich dieselbe Adresse an, bekommt der zweite 500 mit dem neutralen Satz
	// von apierrors.
	"benutzer_email_key @ benutzer (email)",
	// Befragt: Neue Exemplare ziehen ihre Nummer aus barcode_seq, abgeglichen gegen den Bestand
	// (zieheFreieExemplarBarcodes). Das Ändern einer Nummer antwortet auf eine vergebene mit 409
	// (IstExemplarBarcodeVergeben, api/katalog/exemplar_barcode.go), der Bestands-Import übergeht
	// sie (ON CONFLICT DO NOTHING), die Übernahme aus Littera gibt doppelten Nummern vorher eine
	// neue (klaereBarcodes).
	"buecher_exemplare_barcode_id_key @ buecher_exemplare (barcode_id)",
	// Befragt: Ein altes LMF-Etikett führt zu genau einem Exemplar; zwei mit derselben Nummer
	// machten den Scan mehrdeutig. Der Bestands-Import prüft vorher, welche LMF-Nummern schon ein
	// Exemplar trägt, und übergeht sie; eine doppelte Zeile derselben Datei fängt ON CONFLICT
	// (lmf_barcode) DO NOTHING (import_zeilen_lmf_pg_test.go). Die Migrationen 167 und 168
	// nehmen den Wert aus barcode_id, das eindeutig ist. Gegen barcode_id braucht es keine
	// Prüfung: chk_exemplar_barcode_keine_lmf_nummer hält LMF-Nummern dort heraus.
	"buecher_exemplare_lmf_barcode_key @ buecher_exemplare (lmf_barcode)",
	// Befragt: Das Anlegen über die ISBN schreibt mit ON CONFLICT (isbn) DO UPDATE
	// (repository/titel_isbn.go). Anlegen und Ändern in der Bestandstabelle übersetzen die
	// Verletzung in ErrISBNDoppelt, die Türen antworten mit 409 und Satz
	// (api/katalog/titel_anlegen.go, endpunkte_buecher_aktualisieren.go). Die Importe
	// gleichen vorher gegen den Bestand und die Datei ab (titelSchreibLauf), die Übernahme aus
	// Littera klärt doppelte ISBN vorher.
	"buecher_titel_isbn_key @ buecher_titel (isbn)",
	// Befragt, mit geraete_seriennummer_key: Anlegen und Ändern übersetzen die Verletzung in einen
	// Fehler, der das Feld nennt (geraetEindeutigkeit); die Tür antwortet mit 409 und Satz
	// (geraetKonflikt, api/katalog/geraete.go).
	"geraete_barcode_id_key @ geraete (barcode_id)",
	// Befragt: wie geraete_barcode_id_key.
	"geraete_seriennummer_key @ geraete (seriennummer)",
	// Befragt: Der Wert ist der SHA-256 eines Geheimnisses aus 256 Bit Zufall
	// (bestelllink.ErzeugeToken). Ein zweiter gleicher Wert ist nicht zu erwarten; die Regel hält
	// die Suche über den Link eindeutig.
	"idx_bestellungen_token_hash @ bestellungen_verlauf (bestaetigungs_token_hash) WHERE (bestaetigungs_token_hash IS NOT NULL)",
	// Befragt, für die drei Indizes offener Läufe: Der Start übersetzt die Verletzung in
	// ErrInventurLaeuftBereits (LegeLaufAn), die Tür antwortet mit 409 und Satz
	// (antworteAufLaufkollision). Das Umbenennen einer Sachgruppe stellt offene Läufe auf den neuen
	// Namen um; trifft es einen offenen Lauf auf dem neuen Namen mit demselben Jahrgang, bricht es
	// mit 500 und dem neutralen Satz ab.
	"idx_inv_session_offen_filter @ inventur_sessions (COALESCE(scope_subject, ''::text), COALESCE((scope_grade)::integer, '-1'::integer)) WHERE ((abgeschlossen_am IS NULL) AND ((scope_type)::text = 'filter'::text))",
	"idx_inv_session_offen_global @ inventur_sessions ((true)) WHERE ((abgeschlossen_am IS NULL) AND ((scope_type)::text = 'global'::text))",
	"idx_inv_session_offen_signature @ inventur_sessions (btrim(scope_signatur)) WHERE ((abgeschlossen_am IS NULL) AND ((scope_type)::text = 'signature'::text))",
	// Befragt: Einziger Schreiber ist der Abschluss eines Laufs; er schreibt mit ON CONFLICT DO
	// NOTHING und zählt die Verluste an der Aussonderung, nicht an der Mitschrift
	// (repository/inventurlauf_abschluss.go).
	"idx_inventur_verluste_einmalig @ inventur_verluste (session_id, exemplar_id) WHERE (exemplar_id IS NOT NULL)",
	// Befragt: Das Merkmal setzt nur SetzeHauptlieferant, erst räumen, dann setzen, in einer
	// Transaktion; das Anlegen setzt es nie. Zwei Wechsel im selben Augenblick können sich treffen,
	// weil das Räumen des zweiten den eben gesetzten nicht sieht; dann bricht der zweite mit 500
	// und dem neutralen Satz ab.
	"idx_lieferanten_ein_hauptlieferant @ lieferanten (ist_hauptlieferant) WHERE ist_hauptlieferant",
	// Befragt, mit uniq_klassen_normkey: Einzige Schreiber sind die Trigger klasse_kanonisieren und
	// class_name_kanonisieren. Sie suchen die eingetragene Schreibweise über klassen_normkey, legen
	// Unbekanntes mit ON CONFLICT DO NOTHING an und lesen danach noch einmal; auch zwei
	// gleichzeitige Schreibweisen derselben neuen Klasse enden auf einem Namen.
	"klassen_name_key @ klassen (name)",
	// Befragt: Kein Schreiber im Programm legt eine Vorlage an oder ändert ihre Art; die Tür ändert
	// nur Betreff und Text (repository/mail_vorlagen.go). Die Vorlagen legen die Migrationen an.
	"mail_vorlagen_typ_key @ mail_vorlagen (typ)",
	// Befragt, mit uniq_bescheid_nummer: Einziger Schreiber ist das Anlegen eines Bescheids. Die
	// laufende Nummer zieht es je Topf und Kassenjahr aus der gesperrten Zeile von
	// schadensersatz_nummern (zieheLaufendeNummer). Die Referenznummer im Format des Landes nennt
	// den Topf nicht; für den Schulträger sieht docs/mittel_konzept.md ein eigenes Format vor, bis
	// dahin lehnt die Tür jeden anderen Topf ab (api/schaden/bescheid_erstellen.go).
	"schadensersatz_bescheide_referenznummer_key @ schadensersatz_bescheide (referenznummer)",
	// Befragt, mit den beiden Regeln der Bezeichnung: Anlegen und Ändern einer Sachgruppe antworten
	// mit 409 und einem Satz, der das Feld nennt (meldungDoppelteSachgruppe). Die Importe legen
	// fehlende Fächer an (registriereFach): Ein besetztes Kürzel bekommt eine Endung aus einem
	// Hash, eine besetzte Bezeichnung übergehen sie und lesen die vorhandene.
	"systematik_kategorien_kuerzel_key @ systematik_kategorien (kuerzel)",
	// Befragt: Die Ausleihe an der Theke und das Nachbuchen schreiben mit ON CONFLICT DO NOTHING
	// (sqlAusleihe). Die Theke gibt bei demselben Leser die bestehende Ausleihe zurück und weist
	// sonst mit 409 und Satz ab (deuteAusleiheKonflikt); das Nachbuchen weist die Zeile ab. Die
	// Übernahme aus Littera siebt eine zweite offene Ausleihe vorher aus (Doppelbelegung).
	"uniq_ausleihen_aktiv_exemplar @ ausleihen (exemplar_id) WHERE ((rueckgabe_am IS NULL) AND (exemplar_id IS NOT NULL))",
	// Befragt: Die Ausleihe eines Geräts übersetzt die Verletzung in einen Konflikt mit Satz, an
	// der Tür 409 (internal/service/geraet.go).
	"uniq_ausleihen_aktiv_geraet @ ausleihen (geraet_id) WHERE ((rueckgabe_am IS NULL) AND (geraet_id IS NOT NULL))",
	// Befragt: wie benutzer_email_key.
	"uniq_benutzer_email_lower @ benutzer (lower((email)::text))",
	// Befragt: Jedes Konto bekommt seine eigene Leserzeile (Trigger trg_benutzer_hat_leserzeile,
	// LegeKollegiumskontoAn mit der neuen Zeile). Die Zusammenführung bricht ab, wenn beide Seiten
	// ein Konto haben, mit 409 und Satz (ErrZusammenfuehrenZweiKonten); ein Konto unter der
	// Platzhalter-Domäne räumt sie vorher.
	"uniq_benutzer_leser @ benutzer (leser_id) WHERE (leser_id IS NOT NULL)",
	// Befragt: wie schadensersatz_bescheide_referenznummer_key.
	"uniq_bescheid_nummer @ schadensersatz_bescheide (mittel, kassenjahr, laufende_nr)",
	// Befragt: Der Schlüssel kommt von der Oberfläche, einer je Absenden. Der Bestellkopf schreibt
	// mit ON CONFLICT DO NOTHING, danach liefert der Dienst die schon angelegte Bestellung
	// (LegeBestellkopfAn, LadeBestellungZuIdempotenzSchluessel).
	"uniq_bestellung_idempotenz @ bestellungen_verlauf (idempotenz_schluessel) WHERE (idempotenz_schluessel IS NOT NULL)",
	// Befragt: wie klassen_name_key.
	"uniq_klassen_normkey @ klassen (klassen_normkey((name)::text))",
	// Befragt: wie uniq_bestellung_idempotenz; die Reservierung liefert die schon angelegte zurück
	// und verschickt keine zweite Mail (ReservierungsAbfragen.LegeAn).
	"uniq_ksr_idempotenz @ klassensatz_reservierungen (idempotenz_schluessel) WHERE (idempotenz_schluessel IS NOT NULL)",
	// Befragt: Das Speichern schreibt mit ON CONFLICT (art, schuljahr_beginn) DO UPDATE: je Art und
	// Schuljahr ein Plan, ein zweites Speichern ersetzt ihn (LMFAbfragen.SchreibePlan).
	"uniq_lmf_plaene_art_schuljahr @ lmf_plaene (art, schuljahr_beginn)",
	// Befragt: Bezugspunkt des zusammengesetzten Fremdschlüssels fk_lmf_termine_plan_art (Migration
	// 115). id ist der Primärschlüssel, ein zweiter gleicher Wert kann nicht entstehen.
	"uniq_lmf_plaene_id_art @ lmf_plaene (id, art)",
	// Befragt: Das Speichern löscht die Zeilen des Plans und schreibt sie mit den Positionen 1 bis
	// n neu, in der Transaktion, in der das Upsert die Zeile des Plans sperrt; zwei
	// Speichervorgänge desselben Plans warten aufeinander (LMFAbfragen.SchreibePlan).
	"uniq_lmf_termine_plan_position @ lmf_termine (plan_id, \"position\")",
	// Befragt: Der Schlüssel kommt vom Theken-Rechner; eine Wiederholung schreibt mit ON CONFLICT
	// DO NOTHING keine zweite Meldung (SchreibeNachbuchMeldung).
	"uniq_nachbuch_meldungen_schluessel @ nachbuch_meldungen (idempotency_key)",
	// Befragt: Das Speichern der Schlagworte eines Titels legt neue Wörter mit ON CONFLICT DO
	// NOTHING an (repository/schlagworte.go). Die Pflegeseite prüft vor dem Umbenennen, ob die
	// Schreibweise frei ist (409, ErrSchlagwortGibtEs), und führt bei einem neuen Verweis ein
	// vorhandenes Wort zusammen. Läuft zwischen Prüfung und Schreiben ein zweiter Platz dagegen,
	// bricht sie mit 500 und dem neutralen Satz ab.
	"uniq_schlagworte_wort @ schlagworte (lower(wort))",
	// Befragt: Das Anlegen eines Lesers und die Benutzerverwaltung prüfen vorher
	// (IstAusweisnummerVergeben). Greift die Datenbank, antworten Anlegen und Ändern von Leser und
	// Konto mit 409 und Satz (IstAusweisKollision); das Wiederherstellen aus dem Papierkorb nennt
	// den aktiven Zwilling (wiederherstellungsKollision).
	"uniq_schueler_barcode_active @ leser (barcode_id) WHERE (deleted_at IS NULL)",
	// Befragt: Das Ändern prüft vorher und antwortet mit 409 und Satz (api/leser/leser_aendern.go);
	// der LUSD-Abgleich sucht vor dem Anlegen die Zeile, die die ID schon hält (belegteLUSDID); das
	// Wiederherstellen nennt den Zwilling (wiederherstellungsKollision).
	"uniq_schueler_lusd_id_active @ leser (lusd_id) WHERE ((deleted_at IS NULL) AND (lusd_id IS NOT NULL))",
	// Befragt: wie systematik_kategorien_kuerzel_key.
	"uniq_systematik_bezeichnung @ systematik_kategorien (bezeichnung)",
	// Befragt: wie systematik_kategorien_kuerzel_key.
	"uniq_systematik_bezeichnung_ci @ systematik_kategorien (lower((bezeichnung)::text))",
	// Befragt: Anlegen und Ändern eines Schülers antworten mit 409 und Satz
	// (meldungSchuelerDuplikat), das Wiederherstellen nennt den Zwilling. Der LUSD-Abgleich und die
	// Übernahme aus Littera ordnen über Name und Geburtsdatum zu, bevor sie schreiben
	// (internal/lusd/klassifizierung.go, internal/littera/schreiber_personen.go).
	"unique_schueler_name_gebdatum @ leser (suchnorm((vorname)::text), suchnorm((nachname)::text), geburtsdatum) WHERE ((geburtsdatum IS NOT NULL) AND (deleted_at IS NULL) AND (lusd_id IS NULL))",
	// Befragt: Das Vormerken erkennt die zweite Vormerkung am Constraint und antwortet mit 409 und
	// Satz (ErrVormerkungBereitsVorhanden); die Zusammenführung räumt doppelte Vormerkungen vor dem
	// Umhängen ab.
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
