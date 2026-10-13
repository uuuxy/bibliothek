// Package safego fängt ein Panic in einer Goroutine, die niemand beaufsichtigt.
//
// Ein Panic, das eine Goroutine nicht fängt, beendet den ganzen Prozess. Die Middleware des
// Servers fängt es nur für Anfragen; die Läufe im Hintergrund (Cover, Metadaten) starten
// eigene Goroutinen, und ein Panic dort wäre ein Ausfall für alle.
package safego

import "log"

// FangePanic fängt ein Panic der laufenden Goroutine und schreibt es mit name ins Log. Es
// steht als erste Zeile mit defer in der Goroutine:
//
//	go func() {
//	    defer safego.FangePanic("cover-sync")
//	    ...
//	}()
//
// Ein erwarteter Fehler bleibt eine Rückgabe. FangePanic ist für das Unerwartete: Der Lauf
// endet, der Server läuft weiter.
func FangePanic(name string) {
	if r := recover(); r != nil {
		log.Printf("safego: Panic in Goroutine %q gefangen, Prozess bleibt am Leben: %v", name, r)
	}
}
