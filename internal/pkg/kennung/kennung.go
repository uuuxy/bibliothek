// Package kennung prüft, ob eine Eingabe eine UUID in der Schreibweise ist, die die
// Datenbank annimmt.
//
// uuid.Validate und uuid.Parse aus google/uuid nehmen auch `urn:uuid:7c9e6679-…` an. Postgres
// weist diese Form ab (`invalid input syntax for type uuid`, 22P02), und der Rufer bekäme 500
// statt 400. Geprüft wird deshalb die eine Form, die Oberfläche und Server selbst erzeugen:
// 8-4-4-4-12 Hex-Ziffern. Postgres nähme auch geschweifte Klammern und die Form ohne
// Bindestriche an; beide weist die Prüfung ab, damit nichts durchgeht, was die Datenbank
// danach ablehnt.
package kennung

import "regexp"

var uuidForm = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

// IstUUID sagt, ob s eine UUID in der Form 8-4-4-4-12 ist. Leer ist keine UUID.
func IstUUID(s string) bool {
	return uuidForm.MatchString(s)
}
