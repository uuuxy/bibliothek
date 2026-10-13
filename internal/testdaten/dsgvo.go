package testdaten

// DsgvoSchuelerQuellen nennt jede Tabelle, die auf einen Leser verweist. Auskunft und Tilgung
// sind ein Paar über dieser einen Liste: Eine Auskunft, die eine Tabelle auslässt, sieht
// trotzdem vollständig aus, und eine Tilgung, die eine auslässt, lässt die Person dort stehen.
// Die Prüfungen halten den Quelltext der Auskunft, Tilgung und Zusammenführen an der Datenbank
// und die Fremdschlüssel des Schemas gegen die Liste.
var DsgvoSchuelerQuellen = []struct {
	Tabelle string
	Bezug   string // wie die Tabelle auf den Schüler verweist (Doku + Fehlermeldung)
	MitFK   bool   // taucht im information_schema-Scan der FK-Ratsche auf
}{
	// Die Tabelle leser, nicht die Sicht schueler: Die Auskunft gibt es seit dem 24.09.2026
	// für jeden Leser (OFFEN.md 5.19).
	{"leser", "id", false},
	{"schueler_fotos", "schueler_id (FK, ON DELETE CASCADE)", true},
	{"ausleihen", "schueler_id (FK, bei Tilgung -> NULL)", true},
	{"schadensfaelle", "schueler_id (FK, bei Tilgung geloescht)", true},
	{"vormerkungen", "schueler_id (FK, bei Tilgung geloescht)", true},
	// Migration 110: Der Bescheid bleibt als Beleg (die Referenznummer ordnet Zahlungen
	// zu), sein Empfänger-Snapshot wird bei der Tilgung geleert.
	{"schadensersatz_bescheide", "schueler_id (FK, bei Tilgung wird empfaenger_snapshot geleert)", true},
	// Migration 117: Die Meldung bleibt als Vorgang (Barcode, Ergebnis), beide Personen-
	// spalten werden bei der Tilgung auf NULL gesetzt.
	{"nachbuch_meldungen", "ausleiher_schueler_id / vorbesitzer_schueler_id (FK, bei Tilgung -> NULL)", true},
	// Migration 123: Die Tabelle fuehrt alle Leser, und ein Konto zeigt auf seine
	// Leserzeile. Die Auskunft nennt das Konto samt Anmeldedaten (seit dem 24.09.2026,
	// repository/dsgvo_konto.go); die Tilgung loest die Verknuepfung. Der Fremdschluessel
	// steht auf ON DELETE SET NULL, greift also erst, wenn die Leserzeile ganz verschwindet.
	{"benutzer", "leser_id (FK, ON DELETE SET NULL; bei Tilgung -> NULL)", true},
	{"audit_log", "datensatz_id (tabelle='schueler') bzw. details->>'schueler_id'", false},
	{"audit_logs", "details->>'schueler_id'", false},
}
