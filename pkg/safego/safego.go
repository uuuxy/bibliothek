// Package safego schützt unbeaufsichtigte Goroutinen vor dem Prozess-Aus.
//
// In Go reisst ein Panic, das aus einer Goroutine nicht gefangen wird, den GESAMTEN
// Prozess mit — nicht nur die eine Goroutine. Die globale PanicRecoveryMiddleware
// deckt ausschliesslich HTTP-Handler ab; manuell gestartete Hintergrund-Goroutinen
// (Cover-Sync, Metadaten-Abgleich, Fire-and-forget-Jobs) sind ihr entzogen. Ein Panic
// dort — etwa aus einem künftigen Parser für eine unerwartete externe API-Antwort —
// wäre ein Totalausfall für alle Nutzer, ausgelöst von einer Nebentätigkeit.
//
// Das Muster stammt aus plugins/hooks.go, wo es bereits einzeln angewandt wurde; dieses
// Paket macht es wiederverwendbar, damit nicht jede Goroutine ihren eigenen recover
// mitbringen (oder vergessen) muss.
package safego

import "log"

// Guard fängt ein Panic der laufenden Goroutine ab und protokolliert es unter name.
// Als ERSTE deferte Zeile einer Hintergrund-Goroutine gedacht:
//
//	go func() {
//	    defer safego.Guard("cover-sync")
//	    ...
//	}()
//
// Es ersetzt kein sauberes Fehler-Handling — ein erwarteter Fehler gehört als Rückgabe
// behandelt, nicht als Panic. Guard ist das Sicherheitsnetz für das Unerwartete: Der
// Job stirbt, der Server lebt weiter.
func Guard(name string) {
	if r := recover(); r != nil {
		log.Printf("safego: Panic in Goroutine %q gefangen, Prozess bleibt am Leben: %v", name, r)
	}
}
