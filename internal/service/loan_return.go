package service

import (
	"context"
	"fmt"

	"bibliothek/db"
	"bibliothek/repository"

	"github.com/jackc/pgx/v5"
)

// processReturnVormerkungTx stellt das zurückgegebene Buch in der Transaktion der Rückgabe dem
// nächsten Wartenden bereit, der abholen darf, und trägt ihn in die Antwort ein.
//
// returningSchuelerID ist, wer das Buch gerade zurückgibt (nil ohne Ausleiher). Seine eigene
// Vormerkung wird übergangen: Sonst stellte er sich das Buch beim Zurückgeben selbst wieder
// bereit und hielte die Warteschlange an.
//
// Ein Fehler kommt zurück, und der Aufrufer rollt die Rückgabe zurück: Meldete die Theke „ins
// Abholfach legen", während die Vormerkung wartend bliebe, läge das Buch im Fach und wäre
// zugleich frei ausleihbar. Die Theke wiederholt den Scan.
func (s *defaultLoanService) processReturnVormerkungTx(ctx context.Context, tx pgx.Tx, copy *repository.BookCopy, resp *LoanResult, returningSchuelerID *string) error {
	// Die älteste wartende Vormerkung eines Schülers, der abholen darf; die des Schülers, der
	// gerade zurückgibt, ist ausgenommen.
	wartende, da, err := repository.SperreNaechsteWartendeVormerkung(ctx, tx, copy.TitelID, returningSchuelerID)
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
	// Nachrücken in der Warteschlange (repository.Abholfrist).
	frist, err := repository.Abholfrist(ctx, tx, s.heute())
	if err != nil {
		return fmt.Errorf("abholfrist für vormerkung %s: %w", wartende.ID, err)
	}
	if err := repository.StelleVormerkungBereit(ctx, tx, wartende.ID, copy.ID, frist); err != nil {
		return fmt.Errorf("vormerkung %s auf 'abholbereit' setzen: %w", wartende.ID, err)
	}

	resp.HasVormerkung = true
	resp.VormerkungTitel = copy.Titel
	resp.VormerkungUser = schuelerName
	return nil
}

// HandleSimpleReturn wickelt die einfache Rückgabe eines Buchexemplars ab, wenn kein neuer
// Ausleiher aktiv ist.
func (s *defaultLoanService) HandleSimpleReturn(
	ctx context.Context,
	copy *repository.BookCopy,
	staffID string,
) (*LoanResult, error) {
	resp := &LoanResult{}

	// Transaktion starten, um Datenkonsistenz bei Rückgabe und Vormerkungsverarbeitung zu garantieren
	tx, err := s.loanRepo.BeginTx(ctx)
	if err != nil {
		return nil, err
	}
	defer db.SafeRollback(ctx, tx)

	// Aktive Ausleihe für das Buchexemplar laden
	activeLoan, err := s.loanRepo.GetActiveLoanByCopyIDTx(ctx, tx, copy.ID)
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
	if activeLoan == nil {
		return nil, meldung(ErrInvalidState, "Dieses Buchexemplar ist aktuell nicht ausgeliehen")
	}

	return s.handleRueckgabe(ctx, tx, copy, activeLoan, staffID, resp)
}

// handleRueckgabe verbucht die Rückgabe eines ausgeliehenen Buchs inkl.
// Vormerkungsaktivierung und Plugin-Event — für JEDEN Ausleiher.
//
// Bis Migration 125 gab es zwei Fassungen davon: eine für Schüler und eine für den
// Mitarbeiter, der sein eigenes Buch scannt. Sie unterschieden sich nicht in der Sache,
// sondern nur in der Spalte, in der der Ausleiher stand — und die Mitarbeiter-Fassung
// vergaß dabei, den Ausleiher in die Antwort zu legen.
func (s *defaultLoanService) handleRueckgabe(ctx context.Context, tx pgx.Tx, copy *repository.BookCopy, activeLoan *repository.Loan, staffID string, resp *LoanResult) (*LoanResult, error) {
	var borrower *repository.Student
	if activeLoan.SchuelerID != nil {
		var err error
		// GetLeserByID: Der Ausleiher kann ein Kollege sein; die Sicht `schueler` zeigte
		// ihn nicht und die Rückgabe meldete dann einen Ausleiher, den es nicht gibt.
		borrower, err = s.studentRepo.GetLeserByID(ctx, *activeLoan.SchuelerID)
		if err != nil {
			return nil, err
		}
	}

	// Rückgabe buchen
	if err := s.loanRepo.ReturnLoanTx(ctx, tx, activeLoan.ID, staffID, false); err != nil {
		return nil, err
	}

	// Eventuelle Vormerkungen aktivieren — die eigene Vormerkung des zurückgebenden
	// Schülers wird dabei übersprungen (Monopolisierungs-Schutz).
	if err := s.processReturnVormerkungTx(ctx, tx, copy, resp, activeLoan.SchuelerID); err != nil {
		return nil, err
	}

	// Revisionssicheres Audit-Log schreiben
	if activeLoan.SchuelerID != nil {
		if err := s.auditRepo.LogRueckgabe(ctx, tx, copy.ID, *activeLoan.SchuelerID, "", staffID); err != nil {
			return nil, err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	resp.Type = "rueckgabe"
	resp.Book = copy
	resp.Student = borrower
	resp.LoanID = &activeLoan.ID
	return resp, nil
}
