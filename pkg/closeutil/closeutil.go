// Package closeutil bietet Helfer zum Schließen von Ressourcen, deren Close-Fehler
// nicht behebbar ist, aber dennoch beobachtet (protokolliert) werden soll.
package closeutil

import (
	"io"
	"log"
)

// LogClose schließt c und protokolliert einen etwaigen Fehler unter dem angegebenen
// Kontext-Label. Gedacht für deferte Best-Effort-Closes von Readern, Response-Bodies
// und nur lesend geöffneten Dateien, z. B. `defer closeutil.LogClose(file, "upload")`.
func LogClose(c io.Closer, context string) {
	if err := c.Close(); err != nil {
		log.Printf("%s: close failed: %v", context, err)
	}
}
