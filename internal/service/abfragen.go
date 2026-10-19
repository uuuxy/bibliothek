package service

import (
	"context"

	"bibliothek/repository"
)

// Die Abfragen, die die Dienste dieses Pakets brauchen, je nach dem Vorgang geschnitten
// (docs/STIL.md 14). Umgesetzt sind sie von den Typen in repository; die Tests setzen an ihrer
// Stelle Attrappen ein.

// leserAbruf liefert den Leser zu seiner Kennung (repository.LeserAbfragen).
type leserAbruf interface {
	// Finde liefert den Leser zu seiner Kennung; nil, wenn es ihn nicht gibt.
	Finde(ctx context.Context, id string) (*repository.Leser, error)
}

// ausweisErkennung liefert den Leser zu einer Ausweisnummer (repository.LeserAbfragen).
type ausweisErkennung interface {
	// FindeZuBarcode liefert den Leser zu einer Ausweisnummer, Schüler wie Kollegium; nil, wenn
	// die Nummer zu niemandem gehört.
	FindeZuBarcode(ctx context.Context, barcode string) (*repository.Leser, error)
}

// leserErmittlung findet den Leser zu einer Kennung oder einem Ausweis: für den Leser an der
// Theke, für einen gescannten Ausweis und beim Nachbuchen.
type leserErmittlung interface {
	leserAbruf
	ausweisErkennung
}
