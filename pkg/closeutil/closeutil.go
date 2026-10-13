// Package closeutil schließt Ressourcen, deren Fehler beim Schließen sich nicht beheben
// lässt, und schreibt ihn ins Log, damit er nicht verloren geht.
package closeutil

import (
	"io"
	"log"
)

// Schliesse schließt c und schreibt einen Fehler mit der genannten Stelle ins Log. Gedacht
// für defer an Lesern, Antwortrümpfen und nur lesend geöffneten Dateien.
func Schliesse(c io.Closer, stelle string) {
	if err := c.Close(); err != nil {
		log.Printf("%s: close failed: %v", stelle, err)
	}
}
