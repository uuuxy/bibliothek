// Package safego fängt ein Panic in einer Goroutine, die niemand beaufsichtigt.
//
// Ein Panic, das eine Goroutine nicht fängt, beendet den ganzen Prozess. Die Middleware des
// Servers fängt es nur für Anfragen; die Läufe im Hintergrund (Cover, Metadaten, Zeitplan,
// Wächter) starten eigene Goroutinen, und ein Panic dort wäre ein Ausfall für alle.
//
// Eine Goroutine, auf deren Ergebnis jemand wartet (eine Leitung, ein Kanal), fängt kein Panic:
// Der Wartende bliebe hängen. Dann endet der Prozess, Docker startet ihn neu, und der nächste
// Start meldet den Abbruch (jobs.legeFehlerstandAb).
package safego

import (
	"log/slog"
	"runtime/debug"
)

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
		slog.Error("safego: Panic in Goroutine gefangen, Prozess bleibt am Leben", "goroutine", name, "fehler", r, "stack", string(debug.Stack()))
	}
}

// FuehreAus führt lauf aus und fängt sein Panic wie FangePanic. So endet in einer Schleife nur
// der eine Lauf, und der nächste kommt im Takt.
func FuehreAus(name string, lauf func()) {
	defer FangePanic(name)
	lauf()
}
