package repository

// InventurAbfragen liest und schreibt, was eine Inventur braucht: die Läufe mit ihren
// Erfassungen (inventurlauf.go, inventurlauf_scan.go, inventurlauf_abschluss.go), den Bereich
// eines Laufs (inventur_bereich.go) und den Fehlbestand (inventur_verluste.go).
type InventurAbfragen struct {
	db Datenbank
}

// NewInventurAbfragen liefert die Abfragen der Inventur. db ist der Pool oder eine Transaktion:
// Abschluss und die Handlungen am Fehlbestand laufen in der Transaktion der Tür.
func NewInventurAbfragen(db Datenbank) *InventurAbfragen {
	return &InventurAbfragen{db: db}
}
