package service

import (
	"context"
	"errors"
	"fmt"

	"bibliothek/db"
	"bibliothek/repository"

	"github.com/jackc/pgx/v5"
)

// schuelerAbholberechtigt filtert wartende Vormerkungen auf Schüler, die das Buch auch
// tatsächlich abholen dürfen: nicht soft-gelöscht (deleted_at) und nicht gesperrt
// (ist_gesperrt / is_manually_blocked). Ohne diesen Filter würde ein zurückgegebenes Buch
// für einen gelöschten oder gesperrten "Geister"-Schüler abholbereit blockiert, während
// aktive Schüler leer ausgehen. Der Alias "s" muss dabei die schueler-Tabelle sein.
const schuelerAbholberechtigt = `s.deleted_at IS NULL AND s.ist_gesperrt = false AND COALESCE(s.is_manually_blocked, false) = false`

// processReturnVormerkungTx prüft innerhalb einer laufenden Transaktion, ob eine Vormerkung (Reservierung)
// für das zurückgegebene Buch vorliegt. Wenn ja, wird diese Vormerkung aktiviert (Status auf 'abholbereit' gesetzt)
// und dem nächsten wartenden, abholberechtigten Schüler zugeteilt.
//
// returningSchuelerID ist der Schüler, der das Buch GERADE zurückgibt (nil bei Mitarbeiter-/
// Handapparat-Rückgaben). Seine eigene Vormerkung wird bei der Zuteilung übersprungen: Sonst
// könnte er das Buch beim Zurückgeben sofort wieder für sich selbst abholbereit stellen und die
// Warteschlange dauerhaft monopolisieren (Vormerkungs-Monopolisierung).
// Fehler kommen seit dem 31.08.2026 ZURÜCK statt nur ins Log: Vorher wurden die
// Antwortfelder trotz gescheitertem UPDATE gesetzt und der Aufrufer committete — der
// Arbeitsplatz meldete „ins Abholfach legen", die Vormerkung blieb 'wartend', das Buch
// lag im Fach UND war frei ausleihbar, und der Verfall-Cron griff mangels
// bereitgestellt_bis nie. Auch ein Transportfehler des SELECTs galt still als „keine
// Vormerkung". Der Aufrufer rollt jetzt zurück; die Theke wiederholt den Scan.
func (s *defaultLoanService) processReturnVormerkungTx(ctx context.Context, tx pgx.Tx, copy *repository.BookCopy, resp *LoanResult, returningSchuelerID *string) error {
	var vID, sVorname, sNachname, sKlasse string
	// Die älteste wartende Vormerkung eines abholberechtigten Schülers ermitteln und sperren.
	// Die Vormerkung des gerade zurückgebenden Schülers wird ausgeschlossen ($2).
	//
	// FOR UPDATE OF v: NUR die vormerkungen-Zeile sperren, nicht die (nur gelesene)
	// schueler-Zeile. Ein pauschales FOR UPDATE lockt beim JOIN auch schueler — das kann mit
	// der Ausleih-Logik (loan_checkout.go: SELECT ... FROM schueler ... FOR UPDATE) in einen
	// Deadlock laufen (Sperren in umgekehrter Reihenfolge). SKIP LOCKED lässt zwei zeitgleiche
	// Rückgaben desselben Titels je die nächste freie Vormerkung greifen, statt zu blockieren.
	err := tx.QueryRow(ctx, `
		SELECT v.id, s.vorname, s.nachname, COALESCE(s.klasse, '')
		FROM vormerkungen v
		JOIN schueler s ON v.schueler_id = s.id
		WHERE v.titel_id = $1 AND v.status = 'wartend'
		  AND `+schuelerAbholberechtigt+`
		  AND ($2::uuid IS NULL OR v.schueler_id <> $2::uuid)
		ORDER BY v.erstellt_am ASC LIMIT 1
		FOR UPDATE OF v SKIP LOCKED
	`, copy.TitelID, returningSchuelerID).Scan(&vID, &sVorname, &sNachname, &sKlasse)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil // niemand wartet — der Normalfall
	}
	if err != nil {
		return fmt.Errorf("wartende Vormerkung ermitteln: %w", err)
	}

	schuelerName := sVorname + " " + sNachname
	if sKlasse != "" {
		schuelerName += ", " + sKlasse
	}

	// Status der Vormerkung auf 'abholbereit' setzen. Das Buch wird für 3 Tage für diesen Schüler reserviert.
	if _, err := tx.Exec(ctx, "UPDATE vormerkungen SET status = 'abholbereit', bereitgestellt_exemplar_id = $1, bereitgestellt_bis = CURRENT_TIMESTAMP + INTERVAL '3 days' WHERE id = $2", copy.ID, vID); err != nil {
		return fmt.Errorf("vormerkung %s auf 'abholbereit' setzen: %w", vID, err)
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
		return nil, fmt.Errorf("%w: Dieses Buchexemplar ist aktuell nicht ausgeliehen", ErrInvalidState)
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
