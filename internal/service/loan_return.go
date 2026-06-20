package service

import (
	"context"
	"fmt"
	"time"

	"bibliothek/plugins"
	"bibliothek/repository"

	"github.com/jackc/pgx/v5"
)

// processReturnVormerkungTx prüft innerhalb einer laufenden Transaktion, ob eine Vormerkung (Reservierung)
// für das zurückgegebene Buch vorliegt. Wenn ja, wird diese Vormerkung aktiviert (Status auf 'abholbereit' gesetzt)
// und dem nächsten wartenden Schüler zugeteilt.
func (s *defaultLoanService) processReturnVormerkungTx(ctx context.Context, tx pgx.Tx, copy *repository.BookCopy, resp *LoanResult) {
	var vID, sVorname, sNachname, sKlasse string
	// Die älteste wartende Vormerkung für diesen Buchtitel ermitteln und sperren (FOR UPDATE)
	err := tx.QueryRow(ctx, `
		SELECT v.id, s.vorname, s.nachname, COALESCE(s.klasse, '')
		FROM vormerkungen v
		JOIN schueler s ON v.schueler_id = s.id
		WHERE v.titel_id = $1 AND v.status = 'wartend'
		ORDER BY v.erstellt_am ASC LIMIT 1
		FOR UPDATE
	`, copy.TitelID).Scan(&vID, &sVorname, &sNachname, &sKlasse)

	if err == nil {
		schuelerName := sVorname + " " + sNachname
		if sKlasse != "" {
			schuelerName += ", " + sKlasse
		}

		// Status der Vormerkung auf 'abholbereit' setzen. Das Buch wird für 3 Tage für diesen Schüler reserviert.
		_, _ = tx.Exec(ctx, "UPDATE vormerkungen SET status = 'abholbereit', bereitgestellt_exemplar_id = $1, bereitgestellt_bis = CURRENT_TIMESTAMP + INTERVAL '3 days' WHERE id = $2", copy.ID, vID)

		resp.HasVormerkung = true
		resp.VormerkungTitel = copy.Titel
		resp.VormerkungUser = schuelerName
	}
}

// HandleSimpleReturn wickelt die einfache Rückgabe eines Buchexemplars ab, wenn kein neuer Ausleiher aktiv ist.
// Sonderfall: Wenn eine Lehrkraft ein aktuell nicht ausgeliehenes Buch scannt, wird dieses als Handapparat-Ausleihe
// für diese Lehrkraft (1 Jahr Frist) verbucht.
func (s *defaultLoanService) HandleSimpleReturn(
	ctx context.Context,
	copy *repository.BookCopy,
	staffID string,
	staffRole string,
) (*LoanResult, error) {
	resp := &LoanResult{}

	// Transaktion starten, um Datenkonsistenz bei Rückgabe und Vormerkungsverarbeitung zu garantieren
	tx, err := s.loanRepo.BeginTx(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	// Aktive Ausleihe für das Buchexemplar laden
	activeLoan, err := s.loanRepo.GetActiveLoanByCopyIDTx(ctx, tx, copy.ID)
	if err != nil {
		return nil, err
	}

	// Falls das Buch aktuell nicht ausgeliehen ist:
	if activeLoan == nil {
		// Spezialfall: Scan durch eine Lehrkraft -> Direktes Ausleihen als Handapparat (Dauerleihe, 1 Jahr Frist)
		if staffRole == "LEHRER" {
			dueTime := time.Now().AddDate(1, 0, 0) // 1 Jahr Leihfrist
			loan, err := s.loanRepo.CreateUserLoanTx(ctx, tx, copy.ID, staffID, staffID, dueTime, true)
			if err != nil {
				return nil, err
			}
			if err := tx.Commit(ctx); err != nil {
				return nil, err
			}
			_ = s.auditRepo.LogAusleihe(ctx, copy.ID, "", staffID, staffID)

			resp.Type = "ausleihe"
			resp.Book = copy
			if loan != nil {
				resp.DueDate = &loan.RueckgabeFrist
			}
			return resp, nil
		}
		// Für normale Mitarbeiter/Schüler ist dies ein Fehler (Buch ist bereits im Bestand)
		return nil, fmt.Errorf("%w: Dieses Buchexemplar ist aktuell nicht ausgeliehen", ErrInvalidState)
	}

	// Falls das Buch auf denselben Mitarbeiter ausgeliehen ist, der es gerade scannt:
	if activeLoan.AusleiherBenutzerID != nil && *activeLoan.AusleiherBenutzerID == staffID {
		if err = s.loanRepo.ReturnLoanTx(ctx, tx, activeLoan.ID, staffID, false); err != nil {
			return nil, err
		}

		s.processReturnVormerkungTx(ctx, tx, copy, resp)

		if err := tx.Commit(ctx); err != nil {
			return nil, err
		}
		_ = s.auditRepo.LogRueckgabe(ctx, copy.ID, "", *activeLoan.AusleiherBenutzerID, staffID)

		// Event für Plugins triggern (Rückgabe registriert)
		plugins.DispatchEvent(ctx, plugins.EventBookReturned, plugins.BookReturnedPayload{
			CopyID:       copy.ID,
			BarcodeID:    copy.BarcodeID,
			Titel:        copy.Titel,
			SchuelerID:   activeLoan.SchuelerID,
			BearbeiterID: staffID,
		})

		resp.Type = "rueckgabe"
		resp.Book = copy
		resp.LoanID = &activeLoan.ID
		return resp, nil
	}

	// Regulärer Fall: Rückgabe eines von einem Schüler ausgeliehenen Buchs
	var borrowerStudent *repository.Student
	if activeLoan.SchuelerID != nil {
		borrowerStudent, err = s.studentRepo.GetByID(ctx, *activeLoan.SchuelerID)
		if err != nil {
			return nil, err
		}
	}

	// Rückgabe buchen
	if err = s.loanRepo.ReturnLoanTx(ctx, tx, activeLoan.ID, staffID, false); err != nil {
		return nil, err
	}

	// Eventuelle Vormerkungen aktivieren
	s.processReturnVormerkungTx(ctx, tx, copy, resp)

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	// Revisionssicheres Audit-Log schreiben
	if activeLoan.SchuelerID != nil {
		_ = s.auditRepo.LogRueckgabe(ctx, copy.ID, *activeLoan.SchuelerID, "", staffID)
	} else if activeLoan.AusleiherBenutzerID != nil {
		_ = s.auditRepo.LogRueckgabe(ctx, copy.ID, "", *activeLoan.AusleiherBenutzerID, staffID)
	}

	// Event für Plugins triggern (Rückgabe registriert)
	plugins.DispatchEvent(ctx, plugins.EventBookReturned, plugins.BookReturnedPayload{
		CopyID:       copy.ID,
		BarcodeID:    copy.BarcodeID,
		Titel:        copy.Titel,
		SchuelerID:   activeLoan.SchuelerID,
		BearbeiterID: staffID,
	})

	resp.Type = "rueckgabe"
	resp.Book = copy
	resp.Student = borrowerStudent
	resp.LoanID = &activeLoan.ID
	return resp, nil
}
