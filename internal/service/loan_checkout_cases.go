package service

import (
	"context"
	"errors"
	"fmt"
	"log"

	"bibliothek/repository"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// mapLoanCreateErr übersetzt eine Unique-Verletzung (Migration 033: höchstens eine aktive
// Ausleihe je Exemplar/Gerät) in einen sauberen Konflikt. Das tritt auf, wenn ein zweiter
// zeitgleicher Checkout dasselbe Exemplar greifen will — dann ist 409 (statt 500) korrekt.
// erzeugeAusleihe schreibt die Ausleihe und behandelt den einen Fall, den der
// INSERT nicht selbst entscheiden kann: Er prallt am eindeutigen Index ab, weil
// zwischen Freiprüfung und Schreiben ein anderer Vorgang das Exemplar verbucht hat.
func (s *defaultLoanService) erzeugeAusleihe(
	ctx context.Context,
	tx pgx.Tx,
	copy *repository.BookCopy,
	chkCtx *checkoutContext,
	staffID string,
) (*repository.Loan, error) {
	// Ein Schreiber für jeden Leser (Migration 125). Die Dauerleihe bleibt an die Art
	// geknüpft — sie war vorher an die Tabelle geknüpft, was dasselbe war: Wer kein
	// Schüler ist, bekommt das Buch fürs Schuljahr und zählt nicht in die
	// Überfällig-Automatik (zaehleUeberfaelligeMedien).
	loan, err := s.loanRepo.CreateLoanTx(ctx, tx, copy.ID, chkCtx.borrowerID, staffID, chkCtx.dueTime, !chkCtx.istSchueler())
	if err == nil {
		return loan, nil
	}
	if !errors.Is(err, repository.ErrAusleiheKonflikt) {
		return nil, mapLoanCreateErr(err)
	}

	// Doppelter Scan desselben Ausleihers: gewünschter Zustand besteht schon.
	deutung := s.deuteAusleiheKonflikt(ctx, tx, copy, chkCtx)
	var bereits errAusleiheBereitsVorhanden
	if errors.As(deutung, &bereits) {
		return bereits.loan, nil
	}
	return nil, deutung
}

// deuteAusleiheKonflikt beantwortet die Frage, die nach einem abgeprallten INSERT
// wirklich zählt: WER hat das Exemplar jetzt?
//
// Zwischen der Freiprüfung und dem Schreiben hat ein anderer Vorgang es verbucht.
// Zwei sehr verschiedene Fälle:
//
//   - Derselbe Ausleiher (Handscanner feuert doppelt, ungeduldiges zweites Enter):
//     Der gewünschte Zustand ist bereits hergestellt. Die bestehende Ausleihe wird
//     zurückgemeldet — der Scan war wirkungslos, aber nicht falsch. Genau dafür war
//     das ON CONFLICT gedacht.
//   - Ein anderer Ausleiher: Hier MUSS der Arbeitsplatz einen Fehler sehen. Vorher
//     bekam er "ausgeliehen an <eigenen Schüler>" gemeldet, während das Buch auf
//     jemand anderem stand — grüner Blitz, Erfolgston, und die Kraft gibt das Buch
//     heraus. Auffällig wird das erst bei der Inventur.
//
// Read Committed: Die fremde Transaktion ist committet, ein neues SELECT sieht sie.
func (s *defaultLoanService) deuteAusleiheKonflikt(
	ctx context.Context,
	tx pgx.Tx,
	copy *repository.BookCopy,
	chkCtx *checkoutContext,
) error {
	aktiv, err := s.loanRepo.GetActiveLoanByCopyIDTx(ctx, tx, copy.ID)
	if err != nil {
		return err
	}
	if aktiv == nil {
		// Weder geschrieben noch auffindbar — nichts beschönigen.
		return fmt.Errorf("%w: dieses Exemplar konnte nicht verbucht werden, bitte erneut scannen", ErrConflict)
	}

	if istSelberAusleiher(aktiv, chkCtx) {
		return errAusleiheBereitsVorhanden{loan: aktiv}
	}

	return fmt.Errorf("%w: dieses Exemplar wurde soeben an einem anderen Arbeitsplatz verbucht", ErrConflict)
}

// istSelberAusleiher prüft, ob die bestehende Ausleihe demselben Leser gehört wie der
// aktuelle Vorgang.
func istSelberAusleiher(aktiv *repository.Loan, chkCtx *checkoutContext) bool {
	return aktiv.SchuelerID != nil && *aktiv.SchuelerID == chkCtx.borrowerID
}

// errAusleiheBereitsVorhanden ist kein Fehler nach außen, sondern das Signal an
// handleNewLoan, mit der bestehenden Ausleihe weiterzumachen (doppelter Scan).
type errAusleiheBereitsVorhanden struct{ loan *repository.Loan }

func (e errAusleiheBereitsVorhanden) Error() string {
	return "ausleihe besteht bereits für denselben Ausleiher"
}

func mapLoanCreateErr(err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return fmt.Errorf("%w: dieses Exemplar wurde soeben bereits ausgeliehen", ErrConflict)
	}
	return err
}

// entferneErfuellteVormerkung löscht die (erfüllte) Vormerkung des Schülers für diesen
// Titel und erkennt dabei den "Geisterbuch"-Fall: War für ihn bereits ein ANDERES
// Exemplar im Reservierungsfach bereitgestellt, er nimmt sich aber ein Freihand-
// Exemplar, muss das reservierte zurück ins Regal. Der Barcode dieses Exemplars wandert
// als Regal-Hinweis in die Antwort. Fehler hier sind nicht ausleihe-blockierend
// (die Ausleihe selbst ist bereits verbucht) — sie werden nur protokolliert.
func entferneErfuellteVormerkung(ctx context.Context, tx pgx.Tx, copy *repository.BookCopy, schuelerID string, resp *LoanResult) {
	var bereitgestellt *string
	err := tx.QueryRow(ctx,
		`DELETE FROM vormerkungen WHERE titel_id = $1 AND schueler_id = $2
		 RETURNING bereitgestellt_exemplar_id`,
		copy.TitelID, schuelerID).Scan(&bereitgestellt)
	if errors.Is(err, pgx.ErrNoRows) {
		return // keine Vormerkung — Normalfall
	}
	if err != nil {
		log.Printf("ausleihe: Vormerkung für Titel %s konnte nicht entfernt werden: %v", copy.TitelID, err)
		return
	}
	if bereitgestellt == nil || *bereitgestellt == copy.ID {
		return // nichts reserviert, oder genau dieses Exemplar wurde genommen
	}

	var barcode string
	if err := tx.QueryRow(ctx,
		`SELECT barcode_id FROM buecher_exemplare WHERE id = $1`, *bereitgestellt).Scan(&barcode); err != nil {
		log.Printf("ausleihe: Barcode des reservierten Exemplars %s nicht ladbar: %v", *bereitgestellt, err)
		return
	}
	resp.RegalfreigabeBarcode = barcode
}

// handleNewLoan handles the case where the book is currently available (not checked out).
func (s *defaultLoanService) handleNewLoan(
	ctx context.Context,
	tx pgx.Tx,
	copy *repository.BookCopy,
	chkCtx *checkoutContext,
	staffID string,
	resp *LoanResult,
) (*LoanResult, error) {
	loan, err := s.erzeugeAusleihe(ctx, tx, copy, chkCtx, staffID)
	if err != nil {
		return nil, err
	}

	if chkCtx.istSchueler() {
		entferneErfuellteVormerkung(ctx, tx, copy, chkCtx.borrowerID, resp)
	}

	if err := s.auditRepo.LogAusleihe(ctx, tx, copy.ID, chkCtx.borrowerID, "", staffID); err != nil {
		return nil, err
	}
	resp.Student = chkCtx.leser

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	protokolliereUebergangen(ctx, s.auditRepo, staffID, chkCtx.borrowerID, chkCtx.uebergangen)

	resp.Type = "ausleihe"
	resp.Book = copy
	if loan != nil {
		resp.DueDate = &loan.RueckgabeFrist
	}
	return resp, nil
}

// handleReturn handles the case where the active user returns their own book.
func (s *defaultLoanService) handleReturn(
	ctx context.Context,
	tx pgx.Tx,
	copy *repository.BookCopy,
	chkCtx *checkoutContext,
	activeLoan *repository.Loan,
	staffID string,
	resp *LoanResult,
) (*LoanResult, error) {
	if err := s.loanRepo.ReturnLoanTx(ctx, tx, activeLoan.ID, staffID, false); err != nil {
		return nil, err
	}

	// Eigene Vormerkung des zurückgebenden Schülers überspringen (Monopolisierungs-Schutz).
	if err := s.processReturnVormerkungTx(ctx, tx, copy, resp, activeLoan.SchuelerID); err != nil {
		return nil, err
	}

	if err := s.auditRepo.LogRueckgabe(ctx, tx, copy.ID, chkCtx.borrowerID, "", staffID); err != nil {
		return nil, err
	}
	resp.Student = chkCtx.leser

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	resp.Type = "rueckgabe"
	resp.Book = copy
	resp.LoanID = &activeLoan.ID
	return resp, nil
}

// handleForeignReturn: In einer aktiven Sitzung wird ein Buch gescannt, das auf
// jemand ANDEREN verbucht ist. Bewusst NUR eine Rückgabe beim Vorbesitzer —
// kein automatisches Umbuchen auf die aktive Sitzung (Produktentscheidung
// 10.07.: Freund-Rückgaben landeten sonst still auf dem falschen Konto).
// Soll das Buch an die aktive Sitzung: einfach erneut scannen — das Buch ist
// jetzt frei und der zweite Scan läuft als normale Ausleihe (handleNewLoan).
func (s *defaultLoanService) handleForeignReturn(
	ctx context.Context,
	tx pgx.Tx,
	copy *repository.BookCopy,
	activeLoan *repository.Loan,
	staffID string,
	resp *LoanResult,
) (*LoanResult, error) {
	// Der Vorbesitzer ist ein Leser — Schüler oder Kollege. Deshalb GetLeserByID und
	// nicht GetByID: Letzteres liest die Sicht `schueler` und meldete für das Buch eines
	// Kollegen „Vorbesitzer unbekannt".
	var prevLeser *repository.Student
	var err error

	if activeLoan.SchuelerID != nil {
		prevLeser, err = s.studentRepo.GetLeserByID(ctx, *activeLoan.SchuelerID)
		if err != nil {
			log.Printf("fremdrückgabe: Vorbesitzer konnte nicht geladen werden: %v", err)
		}
	}

	if err = s.loanRepo.ReturnLoanTx(ctx, tx, activeLoan.ID, staffID, true); err != nil {
		return nil, err
	}

	// Vormerkungs-Hinweis wie bei jeder Rückgabe: das Buch wird gerade frei. Die eigene
	// Vormerkung des Vorbesitzers wird übersprungen (Monopolisierungs-Schutz).
	if err := s.processReturnVormerkungTx(ctx, tx, copy, resp, activeLoan.SchuelerID); err != nil {
		return nil, err
	}

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
	resp.LoanID = &activeLoan.ID
	// Student = Vorbesitzer: SSE-Livesync zielt auf das Konto, das sich
	// geändert hat; die aktive Sitzung bleibt clientseitig unangetastet.
	resp.Student = prevLeser
	resp.Fremdrueckgabe = true
	resp.Vorbesitzer = prevLeser
	return resp, nil
}
