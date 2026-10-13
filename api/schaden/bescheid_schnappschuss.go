package schaden

import (
	"context"
	"strings"

	"bibliothek/pdf"
	"bibliothek/repository"
)

// Was ein Bescheid einfriert, damit ein Nachdruck denselben Brief ergibt: die Angaben der
// Schule und den Empfänger, je als Karte. Schreiben und Zurücklesen einer Karte stehen
// nebeneinander, weil beide dieselben Schlüssel nennen.

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

// bescheidSnapshotAus friert Anrede, Name und Anschrift zum Briefdatum ein.
//
// Volljährig oder nicht entscheidet über Anrede und die Zeile über dem Namen: Bei
// minderjährigen Schülern geht der Brief an die Erziehungsberechtigten, bei volljährigen
// an die Person selbst. Ohne bekanntes Geschlecht ist bei Volljährigen die neutrale
// Anrede die richtige; den Namen trägt das Anschriftfeld.
func bescheidSnapshotAus(ctx context.Context, repo repository.BescheidAbfragen, schuelerID string) (map[string]string, error) {
	d, err := repo.LadeEmpfaenger(ctx, schuelerID)
	if err != nil {
		return nil, err
	}
	anrede := "Sehr geehrte Erziehungsberechtigte,"
	anZeile := pdf.BescheidAnAnErzieher
	if d.Volljaehrig {
		anrede = "Sehr geehrte Damen und Herren,"
		anZeile = ""
	}
	return map[string]string{
		"anrede":   anrede,
		"an_zeile": anZeile,
		"name":     strings.TrimSpace(d.Vorname + " " + d.Nachname),
		"strasse":  strings.TrimSpace(d.Strasse + " " + d.Hausnummer),
		"plz":      d.PLZ,
		"ort":      d.Ort,
	}, nil
}

// bescheidEmpfaengerAus baut den Empfänger aus dem Snapshot. Ist er leer (Anonymisierung),
// sagt der Brief das — ein Nachdruck mit erfundener Anschrift wäre schlimmer als keiner.
func bescheidEmpfaengerAus(snapshot map[string]string) pdf.BescheidEmpfaenger {
	if len(snapshot) == 0 {
		return pdf.BescheidEmpfaenger{
			Anrede: "Sehr geehrte Damen und Herren,",
			Name:   "(Empfängerangaben nach Löschfrist getilgt)",
		}
	}
	return pdf.BescheidEmpfaenger{
		Anrede:  snapshot["anrede"],
		AnZeile: snapshot["an_zeile"],
		Name:    snapshot["name"],
		Strasse: snapshot["strasse"],
		PLZ:     snapshot["plz"],
		Ort:     snapshot["ort"],
	}
}
