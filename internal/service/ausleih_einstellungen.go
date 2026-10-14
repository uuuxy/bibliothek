package service

import (
	"context"

	"bibliothek/repository"
)

// ausleihEinstellungen sind die Einstellungen, die Ausleihe und Sperren brauchen. Die Felder
// heißen wie in repository.SystemEinstellungen; TestAusleihEinstellungenAus hält beide gleich.
type ausleihEinstellungen struct {
	// FristBuchTage ist die Frist gewöhnlicher Bücher in Tagen.
	FristBuchTage int
	// FristMedienTage ist die Frist für CDs, DVDs und Hörbücher in Tagen.
	FristMedienTage int
	// MaxAusleihenSchueler begrenzt die gleichzeitigen Ausleihen eines Schülers außer den
	// Lernmitteln.
	MaxAusleihenSchueler int
	// LMFStichtag ist der jährliche Rückgabetag der Lernmittel als „MM-TT".
	LMFStichtag string
	// FerienLeseclubAktiv sagt, ob der Ferien-Leseclub läuft.
	FerienLeseclubAktiv bool
	// FerienLeseclubZieldatum ist das gemeinsame Rückgabedatum der Ausleihen im
	// Ferien-Leseclub.
	FerienLeseclubZieldatum *string
	// UeberfaelligKarenzTage sind die Tage, die ein Medium überfällig sein darf, bevor es für
	// die Sperr-Automatik zählt.
	UeberfaelligKarenzTage int
	// UeberfaelligSperreAbMedien ist die Zahl überfälliger Medien, ab der die Sperr-Automatik
	// greift.
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
