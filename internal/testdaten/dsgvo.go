package testdaten

// DSGVOLeserQuellen nennt jede Tabelle, die auf einen Leser verweist. Auskunft und Tilgung
// sind ein Paar über dieser einen Liste: Eine Auskunft, die eine Tabelle auslässt, sieht
// trotzdem vollständig aus, und eine Tilgung, die eine auslässt, lässt die Person dort stehen.
// Die Prüfungen halten den Quelltext der Auskunft, Tilgung und Zusammenführen an der Datenbank
// und die Fremdschlüssel des Schemas gegen die Liste.
var DSGVOLeserQuellen = []struct {
	Tabelle            string
	Bezug              string // wie die Tabelle auf den Leser verweist, für die Meldungen
	MitFremdschluessel bool   // die Prüfung über die Fremdschlüssel des Schemas findet die Tabelle
}{
	// Die Tabelle leser, nicht die Sicht schueler: Die Auskunft gilt für jeden Leser.
	{"leser", "id", false},
	{"schueler_fotos", "schueler_id (FK, ON DELETE CASCADE)", true},
	{"ausleihen", "schueler_id (FK, bei Tilgung -> NULL)", true},
	{"schadensfaelle", "schueler_id (FK, bei Tilgung geloescht)", true},
	{"vormerkungen", "schueler_id (FK, bei Tilgung geloescht)", true},
	// Der Bescheid bleibt als Beleg (die Referenznummer ordnet Zahlungen zu); die Tilgung leert
	// seinen Empfänger-Snapshot.
	{"schadensersatz_bescheide", "schueler_id (FK, bei Tilgung wird empfaenger_snapshot geleert)", true},
	// Die Meldung bleibt als Vorgang (Barcode, Ergebnis); die Tilgung setzt beide Spalten mit
	// Personenbezug auf NULL.
	{"nachbuch_meldungen", "ausleiher_schueler_id / vorbesitzer_schueler_id (FK, bei Tilgung -> NULL)", true},
	// Ein Konto zeigt auf seine Leserzeile. Die Auskunft nennt das Konto samt Anmeldedaten
	// (repository/dsgvo_konto.go), die Tilgung löst die Verknüpfung. Der Fremdschlüssel steht
	// auf ON DELETE SET NULL und greift erst, wenn die Leserzeile ganz verschwindet.
	{"benutzer", "leser_id (FK, ON DELETE SET NULL; bei Tilgung -> NULL)", true},
	{"audit_log", "datensatz_id (tabelle='schueler') bzw. details->>'schueler_id'", false},
	{"audit_logs", "details->>'schueler_id'", false},
}
