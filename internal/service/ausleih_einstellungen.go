package service

import (
	"context"

	"bibliothek/repository"
)

// ausleihEinstellungen repräsentiert die Konfigurationsparameter des Ausleihsystems,
// die in der Datenbanktabelle `system_einstellungen` verwaltet werden.
type ausleihEinstellungen struct {
	// FristBuchTage definiert die Standardleihfrist für normale Bücher in Tagen.
	FristBuchTage int
	// FristMedienTage definiert die Leihfrist für Sonder-Medien (CDs, DVDs, Hörbücher) in Tagen.
	FristMedienTage int
	// MaxAusleihenSchueler begrenzt die Anzahl der gleichzeitig ausgeliehenen regulären Bücher pro Schüler.
	MaxAusleihenSchueler int
	// LMFStichtag bestimmt den jährlichen Rückgabetermin für Schulbücher der Lernmittelfreiheit (z. B. "07-31").
	LMFStichtag string
	// FerienLeseclubAktiv ist wahr, wenn die verlängerte Ausleihe für den Ferien-Leseclub aktiv ist.
	FerienLeseclubAktiv bool
	// FerienLeseclubZieldatum definiert das feste Rückgabedatum für alle Leseclub-Ausleihen.
	FerienLeseclubZieldatum *string
	// UeberfaelligKarenzTage: Anzahl der Toleranztage bevor ein Schüler blockiert wird.
	UeberfaelligKarenzTage int
	// UeberfaelligSperreAbMedien: Anzahl der überfälligen Medien ab denen blockiert wird.
	UeberfaelligSperreAbMedien int
	// Sommerferien ist der Text der Einstellung „sommerferien" — die Jahre, die die Schule
	// neben der Programmtabelle eingetragen hat. Leer heißt: nur die Programmtabelle.
	Sommerferien string
}

// ladeEinstellungen liest über den Pool, was die Ausleihe aus den Einstellungen braucht.
func (d *ausleihDienst) ladeEinstellungen(ctx context.Context) (*ausleihEinstellungen, error) {
	return ladeAusleihEinstellungen(ctx, d.pool)
}

// ladeAusleihEinstellungen liest über q, den Pool oder eine Transaktion, was die Ausleihe aus
// den Einstellungen braucht: Fristen und Sperr-Schwellen. Gelesen und abgebildet werden die
// Zeilen an einer Stelle (repository.LadeEinstellungen); ein fehlender oder unlesbarer Wert
// hat so an der Theke dieselbe Vorgabe wie in der Maske der Einstellungen.
func ladeAusleihEinstellungen(ctx context.Context, q repository.Datenbank) (*ausleihEinstellungen, error) {
	einstellungen, err := repository.LadeEinstellungen(ctx, q)
	if err != nil {
		return nil, err
	}
	return ausleihEinstellungenAus(einstellungen), nil
}

// ausleihEinstellungenAus wählt aus den Einstellungen die der Ausleihe.
func ausleihEinstellungenAus(e *repository.SystemEinstellungen) *ausleihEinstellungen {
	return &ausleihEinstellungen{
		FristBuchTage:              e.FristBuchTage,
		FristMedienTage:            e.FristMedienTage,
		MaxAusleihenSchueler:       e.MaxAusleihenSchueler,
		LMFStichtag:                e.LMFStichtag,
		FerienLeseclubAktiv:        e.FerienLeseclubAktiv,
		FerienLeseclubZieldatum:    e.FerienLeseclubZieldatum,
		UeberfaelligKarenzTage:     e.UeberfaelligKarenzTage,
		UeberfaelligSperreAbMedien: e.UeberfaelligSperreAbMedien,
		Sommerferien:               e.Sommerferien,
	}
}
