package tuertest

import (
	"context"

	"bibliothek/repository"
)

// EinstellungenAttrappe ist ein Ablageort der Einstellungen für Tests: Sie liefert die gesetzten
// Werte oder den gesetzten Fehler und speichert nichts.
type EinstellungenAttrappe struct {
	Einstellungen *repository.SystemEinstellungen
	Fehler        error
}

// Lade liefert die gesetzten Einstellungen oder den gesetzten Fehler.
func (e *EinstellungenAttrappe) Lade(ctx context.Context) (*repository.SystemEinstellungen, error) {
	return e.Einstellungen, e.Fehler
}

// Schreibe nimmt eine Änderung an und verwirft sie.
func (e *EinstellungenAttrappe) Schreibe(ctx context.Context, aenderung *repository.EinstellungenPatch) error {
	return nil
}
