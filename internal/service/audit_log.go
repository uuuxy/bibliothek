package service

import "log"

// logAuditErr protokolliert einen fehlgeschlagenen Audit-Log-Schreibvorgang, ohne den
// aufrufenden Geschäftsvorgang abzubrechen. Audit-Einträge laufen in einer eigenen
// Transaktion; ihr Scheitern darf eine bereits committete Ausleihe/Rückgabe nicht
// rückgängig machen, muss aber für die Revisionssicherheit sichtbar sein.
func logAuditErr(action string, err error) {
	if err != nil {
		log.Printf("audit: %s konnte nicht protokolliert werden: %v", action, err)
	}
}
