package repository

// Die Begriffe, mit denen die Löschroutinen rechnen: Fristen, Kulanz und die Bauform, in der
// eine Bedingung ihre Zahlen mit sich führt.
//
// Getrennt von loeschfristen.go, wo die Bedingungen selbst stehen: Dort ändert sich etwas, wenn
// eine fachliche Regel wechselt, hier, wenn eine Frist wechselt.

const (
	// KulanzJob ist die Kulanz des Löschers selbst: keine. Er ist die Instanz, die die
	// Frist definiert.
	KulanzJob = 0
	// KulanzWaechter ist der eine Tag Luft, den die Selbstprüfung dem letzten
	// nächtlichen Lauf zugesteht.
	KulanzWaechter = 1

	// AnonymisierungPapierkorbTage sind die Tage nach dem Soft-Delete
	// (Papierkorb), nach denen ein Schüler-Datensatz anonymisiert wird.
	AnonymisierungPapierkorbTage = 180
	// AbgaengerKarenzTageVorgabe ist die Vorgabe der Karenzzeit nach dem Abgang: So lange
	// bleibt ein Abgänger ohne offene Vorgänge nur gesperrt, bevor der nächtliche Job ihn
	// anonymisiert. Eine sofortige Anonymisierung beim LUSD-Import nähme jede Reparatur einer
	// falschen Zuordnung mit (Namensänderung ohne Schüler-ID = Abgänger + Neuanlage). 90 Tage
	// decken das erste Quartal nach dem Schuljahreswechsel, in dem solche Fälle an der Theke
	// auffallen. Der Wert ist einstellbar (AbgaengerKarenzSchluessel); 0 heißt „sofort". Die
	// endgültige Löschung (AbgaengerStichjahr, 30. Januar) trifft nur anonymisierte Zeilen: Sie
	// wartet die Karenz ab, statt sie abzuschneiden (praedikatAbgaengerLoeschung).
	AbgaengerKarenzTageVorgabe = 90
	// AbgaengerKarenzSchluessel ist die Zeile in system_einstellungen, aus der Import,
	// Job und Wächter die Karenz lesen — ein Schlüssel für drei Leser.
	AbgaengerKarenzSchluessel = "abgaenger_karenz_tage"

	// BearbeiterKennungTage sind die Tage nach der Rückgabe, nach denen eine Ausleihe nicht
	// mehr nennt, wer sie an der Theke gebucht hat. Die Frist ist fest und nicht einstellbar.
	BearbeiterKennungTage = 14

	// ProtokollAufbewahrungMonateVorgabe ist die Vorgabe-Aufbewahrung der beiden
	// Protokolltabellen.
	ProtokollAufbewahrungMonateVorgabe = 24
	// protokollAufbewahrungMonateMinimum ist die Untergrenze: Eine versehentliche 0 in den
	// Einstellungen darf nicht das komplette Protokoll wegräumen — Revisionsfähigkeit
	// (wer hat die Gebühr storniert?) ist der Zweck der Tabellen.
	protokollAufbewahrungMonateMinimum = 6
	// ProtokollAufbewahrungSchluessel ist die Zeile in system_einstellungen, aus der Job
	// und Wächter die Frist lesen.
	ProtokollAufbewahrungSchluessel = "audit_aufbewahrung_monate"
)

// Loeschbedingung ist eine WHERE-Bedingung mit ihren Parametern.
//
// Ein Paar und kein String: Lieferte ein Prädikat nur den Text und setzte der Rufer die Zahlen
// daneben ein, wäre das eine Verabredung, die nichts durchsetzt. Bei `make_interval(months =>
// $1, days => $2)` kostet ein Vertauschen alles: Aus 24 Monaten Aufbewahrung werden 24 Tage,
// die Grenze springt um zwei Jahre nach vorn, und das Protokoll ist bis auf den letzten Monat
// weg, nächtlich, unumkehrbar und ohne Fehler beim Bauen. So kann die Zahl nur dorthin, wo ihr
// Name steht.
type Loeschbedingung struct {
	Where string
	Args  []any
}
