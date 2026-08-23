package repository

// loeschbedingung.go — die Begriffe, mit denen die Löschroutinen rechnen: Fristen,
// Kulanz, und die Bauform, in der eine Bedingung ihre Zahlen mit sich führt.
//
// Getrennt von loeschfristen.go, wo die Bedingungen selbst stehen: Dort ändert sich
// etwas, wenn eine fachliche Regel wechselt — hier, wenn eine Frist wechselt. Zwei
// Anlässe, zwei Dateien.

const (
	// KulanzJob ist die Kulanz des Löschers selbst: keine. Er ist die Instanz, die die
	// Frist definiert.
	KulanzJob = 0
	// KulanzWaechter ist der eine Tag Luft, den die Selbstprüfung dem letzten
	// nächtlichen Lauf zugesteht.
	KulanzWaechter = 1

	// StandardAnonymisierungSoftDeleteTage sind die Tage nach dem Soft-Delete
	// (Papierkorb), nach denen ein Schüler-Datensatz anonymisiert wird.
	StandardAnonymisierungSoftDeleteTage = 180
	// StandardAnonymisierungAbgaengerTage sind die Tage nach der letzten Änderung, nach
	// denen ein als Abgänger markierter Schüler anonymisiert wird.
	StandardAnonymisierungAbgaengerTage = 360

	// StandardAuditAufbewahrungMonate ist die Vorgabe-Aufbewahrung der beiden
	// Protokolltabellen.
	StandardAuditAufbewahrungMonate = 24
	// MindestAuditAufbewahrungMonate ist die Untergrenze: Eine versehentliche 0 in den
	// Einstellungen darf nicht das komplette Protokoll wegräumen — Revisionsfähigkeit
	// (wer hat die Gebühr storniert?) ist der Zweck der Tabellen.
	MindestAuditAufbewahrungMonate = 6
	// AuditAufbewahrungSchluessel ist die Zeile in system_einstellungen, aus der Job
	// UND Wächter die Frist lesen.
	AuditAufbewahrungSchluessel = "audit_aufbewahrung_monate"
)

// Loeschbedingung ist eine WHERE-Bedingung MIT ihren Parametern.
//
// Warum als Paar und nicht als String: Vorher lieferte jedes Prädikat nur den Text, die
// Zahlen setzte der Aufrufer daneben ein. Das ist eine Konvention, die nichts durchsetzt
// — und bei `make_interval(months => $1, days => $2)` kostet ein Vertauschen alles: Aus
// 24 Monaten Aufbewahrung werden 24 Tage, die Grenze springt um zwei Jahre nach vorn und
// das Protokoll ist bis auf den letzten Monat weg. Nächtlich, unumkehrbar, und es
// kompiliert fehlerfrei. Die fünf anderen Prädikate überstehen ein Vertauschen nur, weil
// Addition kommutativ ist — zufällig sicher ist nicht sicher.
//
// Jetzt kann die Zahl nur noch dorthin, wo ihr Name steht.
type Loeschbedingung struct {
	Where string
	Args  []any
}
