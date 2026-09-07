package service

import "log"

// logAuditErr protokolliert einen fehlgeschlagenen Admin-Audit-Eintrag (LogAdminAktion:
// Sperr-Override, Wareneingang-Sammelbuchung), ohne den Vorgang abzubrechen — diese
// Einträge stehen NEBEN dem Geschäftsvorgang, nicht in seiner Transaktion.
//
// Für Ausleihe und Rückgabe gilt das seit dem 07.09.2026 nicht mehr: LogAusleihe und
// LogRueckgabe schreiben in die Transaktion der Buchung, vor deren Commit; scheitert die
// Spur, scheitert die Buchung laut (Register B, Persistenz-Audit).
func logAuditErr(action string, err error) {
	if err != nil {
		log.Printf("audit: %s konnte nicht protokolliert werden: %v", action, err)
	}
}
