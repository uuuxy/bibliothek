// Package kennung prüft, ob eine Eingabe eine UUID in der Schreibweise ist, die die
// Datenbank annimmt.
//
// Nicht uuid.Validate oder uuid.Parse (google/uuid): Beide nehmen auch
// `urn:uuid:7c9e6679-…` an, Postgres weist diese Form ab (`invalid input syntax for type
// uuid`, 22P02). Die Prüfung war dann bestanden, der Rohtext ging weiter, und der Aufrufer
// bekam 500 statt 400 — am 13.09.2026 am Stack nachgestellt (Schüler zusammenführen).
//
// Geprüft wird deshalb die eine Form, die Frontend und Server selbst erzeugen:
// 8-4-4-4-12 Hex-Ziffern. Postgres nähme zusätzlich geschweifte Klammern und die Form ohne
// Bindestriche an; beide weist diese Prüfung ab. Das ist gewollt: Eine engere Tür lässt
// nichts durch, was die Datenbank danach ablehnt.
package kennung

import "regexp"

var uuidForm = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

// IstUUID sagt, ob s eine UUID in der Form 8-4-4-4-12 ist. Leer ist keine UUID.
func IstUUID(s string) bool {
	return uuidForm.MatchString(s)
}
