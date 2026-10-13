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

// GetSettings liefert die gesetzten Einstellungen oder den gesetzten Fehler.
func (m *EinstellungenAttrappe) GetSettings(ctx context.Context) (*repository.SystemEinstellungen, error) {
	return m.Einstellungen, m.Fehler
}

// SaveSettings nimmt eine Änderung an und verwirft sie.
func (m *EinstellungenAttrappe) SaveSettings(ctx context.Context, req *repository.EinstellungenPatch) error {
	return nil
}
