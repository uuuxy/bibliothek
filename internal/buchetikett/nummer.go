package buchetikett

import "strings"

// IstDruckbar sagt, ob ein Etikett dieses Programms die Nummer tragen darf. Eine LMF-Nummer
// (LMF-, ISBN-13, laufende Nummer, etwa LMF-9783060158423-001) darf es nicht: Das Programm
// vergibt diese Form nicht, und ihr Strichcode hätte auf dem Etikett Striche von 0,16 mm. Braucht
// ein solches Exemplar ein neues Etikett, bekommt es eine kurze Nummer
// (repository.VergebeNeueNummern).
func IstDruckbar(nummer string) bool {
	return !strings.HasPrefix(nummer, "LMF-")
}

// NichtDruckbare liefert die Nummern, die IstDruckbar ablehnt, jede einmal und in der Reihenfolge
// ihres ersten Vorkommens.
func NichtDruckbare(nummern []string) []string {
	var abgelehnt []string
	gesehen := map[string]bool{}
	for _, n := range nummern {
		if !IstDruckbar(n) && !gesehen[n] {
			gesehen[n] = true
			abgelehnt = append(abgelehnt, n)
		}
	}
	return abgelehnt
}
