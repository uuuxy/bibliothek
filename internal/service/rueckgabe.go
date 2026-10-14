package service

import (
	"context"
	"fmt"

	"bibliothek/db"
	"bibliothek/repository"

	"github.com/jackc/pgx/v5"
)

// bedieneVormerkung stellt das zurückgegebene Buch in der Transaktion der Rückgabe dem
// nächsten Wartenden bereit, der abholen darf, und trägt ihn in die Antwort ein.
//
// returningSchuelerID ist, wer das Buch gerade zurückgibt (nil ohne Ausleiher). Seine eigene
// Vormerkung wird übergangen: Sonst stellte er sich das Buch beim Zurückgeben selbst wieder
// bereit und hielte die Warteschlange an.
//
// Ein Fehler kommt zurück, und der Aufrufer rollt die Rückgabe zurück: Meldete die Theke „ins
// Abholfach legen", während die Vormerkung wartend bliebe, läge das Buch im Fach und wäre
// zugleich frei ausleihbar. Die Theke wiederholt den Scan.
func (d *ausleihDienst) bedieneVormerkung(ctx context.Context, tx pgx.Tx, exemplar *repository.Exemplar, erg *AusleihErgebnis, rueckgeberID *string) error {
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

	// Status der Vormerkung auf 'abholbereit' setzen. Das Buch liegt für diesen Schüler bis zum
	// Ende der Abholfrist bereit — ab der Uhr des Dienstes, dieselbe Rechnung wie beim
	// Nachrücken in der Warteschlange (repository.BestimmeAbholfrist).
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

// NimmZurueck wickelt die einfache Rückgabe eines Buchexemplars ab, wenn kein neuer
// Ausleiher aktiv ist.
func (d *ausleihDienst) NimmZurueck(
	ctx context.Context,
	exemplar *repository.Exemplar,
	bearbeiterID string,
) (*AusleihErgebnis, error) {
	erg := &AusleihErgebnis{}

	// Transaktion starten, um Datenkonsistenz bei Rückgabe und Vormerkungsverarbeitung zu garantieren
	tx, err := d.ausleihAbfragen.BeginneTx(ctx)
	if err != nil {
		return nil, err
	}
	defer db.SafeRollback(ctx, tx)

	// Aktive Ausleihe für das Buchexemplar laden
	offeneAusleihe, err := d.ausleihAbfragen.SperreOffene(ctx, tx, exemplar.ID)
	if err != nil {
		return nil, err
	}

	// Ein freies Buch ohne offene Sitzung ist nichts, was man zurückgeben kann.
	//
	// Bis zum 16.09.2026 stand hier ein Sonderweg: Scannte ein angemeldetes
	// Kollegiumskonto ein freies Buch, buchte das System es SOFORT auf diese Person —
	// eine Ausleihe ohne Ausweis und ohne dass jemand sie ausgelöst hätte. Wer
	// Rückläufer sortiert, sammelte damit still Bücher auf seinem eigenen Namen. Wer
	// ausleihen will, legt seinen Ausweis vor wie alle anderen auch.
	if offeneAusleihe == nil {
		return nil, meldung(ErrUngueltigerZustand, "Dieses Buchexemplar ist aktuell nicht ausgeliehen")
	}

	return d.verarbeiteRueckgabe(ctx, tx, exemplar, offeneAusleihe, bearbeiterID, erg)
}

// verarbeiteRueckgabe verbucht die Rückgabe eines ausgeliehenen Buchs inkl.
// Vormerkungsaktivierung und Plugin-Event — für JEDEN Ausleiher.
//
// Bis Migration 125 gab es zwei Fassungen davon: eine für Schüler und eine für den
// Mitarbeiter, der sein eigenes Buch scannt. Sie unterschieden sich nicht in der Sache,
// sondern nur in der Spalte, in der der Ausleiher stand — und die Mitarbeiter-Fassung
// vergaß dabei, den Ausleiher in die Antwort zu legen.
func (d *ausleihDienst) verarbeiteRueckgabe(ctx context.Context, tx pgx.Tx, exemplar *repository.Exemplar, offeneAusleihe *repository.Ausleihe, bearbeiterID string, erg *AusleihErgebnis) (*AusleihErgebnis, error) {
	var ausleiher *repository.Leser
	if offeneAusleihe.SchuelerID != nil {
		var err error
		// LeserAbfragen.Finde: Der Ausleiher kann ein Kollege sein; die Sicht `schueler` zeigte
		// ihn nicht und die Rückgabe meldete dann einen Ausleiher, den es nicht gibt.
		ausleiher, err = d.leserAbfragen.Finde(ctx, *offeneAusleihe.SchuelerID)
		if err != nil {
			return nil, err
		}
	}

	// Rückgabe buchen
	if err := d.ausleihAbfragen.BucheRueckgabe(ctx, tx, offeneAusleihe.ID, bearbeiterID, false); err != nil {
		return nil, err
	}

	// Eventuelle Vormerkungen aktivieren — die eigene Vormerkung des zurückgebenden
	// Schülers wird dabei übersprungen (Monopolisierungs-Schutz).
	if err := d.bedieneVormerkung(ctx, tx, exemplar, erg, offeneAusleihe.SchuelerID); err != nil {
		return nil, err
	}

	// Revisionssicheres Audit-Log schreiben
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
