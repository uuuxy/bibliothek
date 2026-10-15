package tuer

import (
	"bibliothek/pdf"
	"bibliothek/repository"
)

// SchuleAus liefert Name und Anschrift der Schule für den Kopf eines Briefs. Jede Tür, die
// einen Brief druckt oder verschickt, nimmt sie von hier, damit alle Briefe denselben Kopf
// tragen.
func SchuleAus(e *repository.SystemEinstellungen) pdf.SchuleInfo {
	return pdf.SchuleInfo{Name: e.SchuleName, Strasse: e.SchuleStrasse, PLZ: e.SchulePLZ, Ort: e.SchuleOrt}
}
