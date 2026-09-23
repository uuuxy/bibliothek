package api

import (
	"bibliothek/pdf"
	"bibliothek/repository"
)

// bescheidAbsenderAus: die Angaben der Schule, wie sie auf dem Bescheid stehen, als Karte
// für den Schnappschuss (Migration 141). EINE Zuordnung für beide Wege: Beim Erstellen
// speichert der Bescheid diese Karte; beim Nachdruck liest der Brief sie zurück — und fehlt
// sie (Bescheid von vor Migration 141), entsteht dieselbe Karte aus den Einstellungen von
// heute. Zahlstelle und Bankverbindung tragen dabei schon ihre Vorgaben
// (BescheidAngabenAus), also genau das, was im Brief stand.
func bescheidAbsenderAus(angaben repository.BescheidAngaben, schule repository.SchuleAngaben) map[string]string {
	return map[string]string{
		"schule_name":       schule.Name,
		"schule_strasse":    schule.Strasse,
		"schule_plz":        schule.PLZ,
		"schule_ort":        schule.Ort,
		"geschaeftszeichen": angaben.Geschaeftszeichen,
		"bearbeiter":        angaben.Bearbeiter,
		"durchwahl":         angaben.Durchwahl,
		"zahlstelle":        angaben.Zahlstelle,
		"bankverbindung":    angaben.Bankverbindung,
		"aufsicht":          angaben.Aufsicht,
		"schulleitung":      angaben.Schulleitung,
	}
}

// bescheidSchuleAus: die Schulanschrift des Briefkopfs aus der Karte.
func bescheidSchuleAus(absender map[string]string) pdf.SchuleInfo {
	return pdf.SchuleInfo{
		Name:    absender["schule_name"],
		Strasse: absender["schule_strasse"],
		PLZ:     absender["schule_plz"],
		Ort:     absender["schule_ort"],
	}
}
