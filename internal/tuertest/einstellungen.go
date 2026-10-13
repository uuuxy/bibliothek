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
func (m *EinstellungenAttrappe) Lade(ctx context.Context) (*repository.SystemEinstellungen, error) {
	return m.Einstellungen, m.Fehler
}

// Schreibe nimmt eine Änderung an und verwirft sie.
func (m *EinstellungenAttrappe) Schreibe(ctx context.Context, req *repository.EinstellungenPatch) error {
	return nil
}
