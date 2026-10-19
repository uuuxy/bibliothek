package service

import (
	"context"
	"fmt"

	"bibliothek/db"
	"bibliothek/repository"

	"github.com/jackc/pgx/v5"
)

// bedieneVormerkung stellt das zurückgegebene Buch in der Transaktion der Rückgabe dem nächsten
// Wartenden bereit, der abholen darf, und trägt ihn in das Ergebnis ein.
//
// rueckgeberID ist, wer das Buch gerade zurückgibt (nil ohne Ausleiher). Seine eigene
// Vormerkung wird übergangen: Sonst stellte er sich das Buch beim Zurückgeben selbst wieder
// bereit und hielte die Warteschlange an.
//
// Ein Fehler kommt zurück, und der Aufrufer rollt die Rückgabe zurück: Meldete die Theke „ins
// Abholfach legen", während die Vormerkung wartend bliebe, läge das Buch im Fach und wäre
// zugleich frei ausleihbar. Die Theke wiederholt den Scan.
func (d *AusleihDienst) bedieneVormerkung(ctx context.Context, tx pgx.Tx, exemplar *repository.Exemplar, erg *AusleihErgebnis, rueckgeberID *string) error {
	// Die älteste wartende Vormerkung eines Schülers, der abholen darf; die des Schülers, der
	// gerade zurückgibt, ist ausgenommen.
	wartende, da, err := repository.SperreNaechsteWartendeVormerkung(ctx, tx, exemplar.TitelID, rueckgeberID)
	if err != nil {
		return fmt.Errorf("wartende Vormerkung ermitteln: %w", err)
	}
	if !da {
		return nil // niemand wartet — der Normalfall
	}

	schuelerName := wartende.Vorname + " " + wartende.Nachname
	if wartende.Klasse != "" {
		schuelerName += ", " + wartende.Klasse
	}

	// Das Buch liegt bis zum Ende der Abholfrist bereit, gerechnet ab der Uhr des Dienstes wie
	// beim Nachrücken in der Warteschlange (repository.BestimmeAbholfrist).
	frist, err := repository.BestimmeAbholfrist(ctx, tx, d.heute())
	if err != nil {
		return fmt.Errorf("abholfrist für vormerkung %s: %w", wartende.ID, err)
	}
	if err := repository.StelleVormerkungBereit(ctx, tx, wartende.ID, exemplar.ID, frist); err != nil {
		return fmt.Errorf("vormerkung %s auf 'abholbereit' setzen: %w", wartende.ID, err)
	}

	erg.HatVormerkung = true
	erg.VormerkungTitel = exemplar.Titel
	erg.VorgemerktFuer = schuelerName
	return nil
}

// NimmZurueck bucht die Rückgabe eines Exemplars, das ohne geöffneten Leser gescannt wurde. Ein
// freies Exemplar ist hier ein Fehler: Ausgeliehen wird über den Ausweis, nicht dadurch, dass ein
// angemeldetes Konto ein Buch scannt.
func (d *AusleihDienst) NimmZurueck(
	ctx context.Context,
	exemplar *repository.Exemplar,
	bearbeiterID string,
) (*AusleihErgebnis, error) {
	erg := &AusleihErgebnis{}

	tx, err := d.ausleihAbfragen.BeginneTx(ctx)
	if err != nil {
		return nil, err
	}
	defer db.Verwerfe(ctx, tx)

	offeneAusleihe, err := d.ausleihAbfragen.SperreOffene(ctx, tx, exemplar.ID)
	if err != nil {
		return nil, err
	}

	// Ein freies Buch lässt sich nicht zurückgeben. An das angemeldete Konto geht es auch
	// nicht: Wer Rückläufer sortiert, sammelte sonst Bücher auf seinem eigenen Namen.
	if offeneAusleihe == nil {
		return nil, meldung(ErrUngueltigerZustand, "Dieses Buchexemplar ist aktuell nicht ausgeliehen")
	}

	return d.verarbeiteRueckgabe(ctx, tx, exemplar, offeneAusleihe, bearbeiterID, erg)
}

// verarbeiteRueckgabe bucht die Rückgabe eines verliehenen Buchs samt Vormerkung, gleich wer es
// ausgeliehen hat.
func (d *AusleihDienst) verarbeiteRueckgabe(ctx context.Context, tx pgx.Tx, exemplar *repository.Exemplar, offeneAusleihe *repository.Ausleihe, bearbeiterID string, erg *AusleihErgebnis) (*AusleihErgebnis, error) {
	var ausleiher *repository.Leser
	if offeneAusleihe.SchuelerID != nil {
		var err error
		// Der Ausleiher kann ein Kollege sein: LeserAbfragen.Finde liest alle Leser, die Sicht
		// schueler nur die Schüler.
		ausleiher, err = d.leserAbfragen.Finde(ctx, *offeneAusleihe.SchuelerID)
		if err != nil {
			return nil, err
		}
	}

	if err := d.ausleihAbfragen.BucheRueckgabe(ctx, tx, offeneAusleihe.ID, bearbeiterID, false); err != nil {
		return nil, err
	}

	// Die eigene Vormerkung des Schülers zählt nicht, sonst stellte er sich das Buch selbst
	// wieder bereit.
	if err := d.bedieneVormerkung(ctx, tx, exemplar, erg, offeneAusleihe.SchuelerID); err != nil {
		return nil, err
	}

	// Der Eintrag ins Protokoll steht in der Transaktion der Rückgabe.
	if offeneAusleihe.SchuelerID != nil {
		if err := d.protokollAbfragen.ProtokolliereRueckgabe(ctx, tx, exemplar.ID, *offeneAusleihe.SchuelerID, "", bearbeiterID); err != nil {
			return nil, err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	erg.Art = "rueckgabe"
	erg.Exemplar = exemplar
	erg.Leser = ausleiher
	erg.AusleiheID = &offeneAusleihe.ID
	return erg, nil
}
