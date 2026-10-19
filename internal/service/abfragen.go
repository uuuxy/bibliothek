package service

import (
	"context"
	"time"

	"bibliothek/repository"

	"github.com/jackc/pgx/v5"
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

// exemplarErkennung liefert das Exemplar zu einem Barcode (repository.KatalogAbfragen).
type exemplarErkennung interface {
	// FindeExemplar liefert das Exemplar zu einem Barcode mit den Angaben seines Titels; nil,
	// wenn keines die Nummer trägt.
	FindeExemplar(ctx context.Context, barcode string) (*repository.Exemplar, error)
}

// katalogErmittlung findet zu einer Eingabe der Theke das Exemplar oder die Titel.
type katalogErmittlung interface {
	exemplarErkennung
	// FindeTitel sucht über Titel, Autor und ISBN und ordnet die Treffer nach ihrem Rang;
	// höchstens 50, nur Titel mit einem Exemplar.
	FindeTitel(ctx context.Context, suchtext string) ([]repository.Titel, error)
}

// titelImport legt die Titel eines Imports an oder ändert sie (repository.KatalogAbfragen).
type titelImport interface {
	// SchreibeTitel legt viele Titel in einem Sammelauftrag an oder ändert sie und liefert die
	// Zahl der verarbeiteten; einen Titel ohne ISBN ordnet es über seinen Titel zu.
	SchreibeTitel(ctx context.Context, titel []repository.Titel) (int, error)
}

// ausleihPflege legt Ausleihen an und beendet sie, in einer Transaktion mit gesperrter Zeile
// (repository.AusleihAbfragen).
type ausleihPflege interface {
	// BeginneTx beginnt eine Transaktion (Read Committed); der Rufer rollt sie mit defer zurück
	// und schreibt sie bei Erfolg fest.
	BeginneTx(ctx context.Context) (pgx.Tx, error)
	// SperreOffene liefert die offene Ausleihe eines Exemplars und sperrt ihre Zeile bis zum
	// Ende der Transaktion; nil, wenn das Exemplar nicht verliehen ist.
	SperreOffene(ctx context.Context, tx pgx.Tx, exemplarID string) (*repository.Ausleihe, error)
	// LegeAn legt die Ausleihe eines Lesers in der Transaktion an, zum jetzigen Zeitpunkt;
	// istDauerleihe kennzeichnet die Ausleihe für das Schuljahr, sie wird nicht überfällig.
	LegeAn(ctx context.Context, tx pgx.Tx, exemplarID, leserID, bearbeiterID string, rueckgabeFrist time.Time, istDauerleihe bool) (*repository.Ausleihe, error)
	// BucheRueckgabe beendet eine offene Ausleihe in der Transaktion, zum jetzigen Zeitpunkt.
	BucheRueckgabe(ctx context.Context, tx pgx.Tx, ausleiheID, bearbeiterID string, istFremdrueckgabe bool) error
}
