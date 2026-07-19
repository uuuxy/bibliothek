package service

import (
	"context"
	"fmt"
	"log"
	"time"

	"bibliothek/db"
	"bibliothek/plugins"
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
func (s *defaultLoanService) processReturnVormerkungTx(ctx context.Context, tx pgx.Tx, copy *repository.BookCopy, resp *LoanResult, returningSchuelerID *string) {
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

	if err == nil {
		schuelerName := sVorname + " " + sNachname
		if sKlasse != "" {
			schuelerName += ", " + sKlasse
		}

		// Status der Vormerkung auf 'abholbereit' setzen. Das Buch wird für 3 Tage für diesen Schüler reserviert.
		if _, err := tx.Exec(ctx, "UPDATE vormerkungen SET status = 'abholbereit', bereitgestellt_exemplar_id = $1, bereitgestellt_bis = CURRENT_TIMESTAMP + INTERVAL '3 days' WHERE id = $2", copy.ID, vID); err != nil {
			log.Printf("rückgabe: Vormerkung %s konnte nicht auf 'abholbereit' gesetzt werden: %v", vID, err)
		}

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
	defer db.SafeRollback(ctx, tx)

	// Aktive Ausleihe für das Buchexemplar laden
	activeLoan, err := s.loanRepo.GetActiveLoanByCopyIDTx(ctx, tx, copy.ID)
	if err != nil {
		return nil, err
	}

	// Falls das Buch aktuell nicht ausgeliehen ist:
	if activeLoan == nil {
		// Spezialfall: Scan durch eine Lehrkraft -> Direktes Ausleihen als Handapparat (Dauerleihe, 1 Jahr Frist)
		if staffRole == "LEHRER" {
			return s.handleLehrerHandapparat(ctx, tx, copy, staffID, resp)
		}
		// Für normale Mitarbeiter/Schüler ist dies ein Fehler (Buch ist bereits im Bestand)
		return nil, fmt.Errorf("%w: Dieses Buchexemplar ist aktuell nicht ausgeliehen", ErrInvalidState)
	}

	// Falls das Buch auf denselben Mitarbeiter ausgeliehen ist, der es gerade scannt:
	if activeLoan.AusleiherBenutzerID != nil && *activeLoan.AusleiherBenutzerID == staffID {
		return s.handleEigenrueckgabeMitarbeiter(ctx, tx, copy, activeLoan, staffID, resp)
	}

	// Regulärer Fall: Rückgabe eines von einem Schüler ausgeliehenen Buchs
	return s.handleSchuelerRueckgabe(ctx, tx, copy, activeLoan, staffID, resp)
}

// handleLehrerHandapparat verbucht ein nicht ausgeliehenes Buch beim Lehrer-Scan als
// Handapparat-Dauerleihe (1 Jahr Frist) und committet die Transaktion.
func (s *defaultLoanService) handleLehrerHandapparat(ctx context.Context, tx pgx.Tx, copy *repository.BookCopy, staffID string, resp *LoanResult) (*LoanResult, error) {
	// Auch der Handapparat-Schnellpfad muss dieselben Schranken achten wie der reguläre
	// Checkout — sonst bucht ein versehentlicher Scan (LMF-Personal sortiert Rückläufer) ein
	// defektes, ausgesondertes oder für einen Schüler reserviertes Exemplar kommentarlos auf
	// die Lehrkraft, statt zu warnen ("das ist defekt" / "das gehört ins Reservierungsfach").
	if !copy.IstAusleihbar {
		return nil, fmt.Errorf("%w: dieses Buchexemplar ist nicht ausleihbar", ErrInvalidState)
	}
	if copy.IstAusgesondert {
		return nil, fmt.Errorf("%w: dieses Buchexemplar ist ausgesondert", ErrInvalidState)
	}
	// Die Lehrkraft ist kein Schüler-Ausleiher → jede aktive Reservierung (abholbereit) auf
	// dieses Exemplar ist ein Konflikt und blockiert die Handapparat-Buchung.
	if err := s.pruefeVormerkungKonflikt(ctx, tx, copy.ID,
		&checkoutContext{borrowerID: staffID, borrowerType: "teacher"}, false); err != nil {
		return nil, err
	}

	// 1 Jahr Dauerleihe, auf das Tagesende in der Schul-Zeitzone normalisiert — einheitlich
	// mit allen anderen Fristen (kein rohes AddDate mehr).
	dueTime := tagesEndeInSchulzeitzone(time.Now().In(schoolLocation()).AddDate(1, 0, 0))
	loan, err := s.loanRepo.CreateUserLoanTx(ctx, tx, copy.ID, staffID, staffID, dueTime, true)
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	logAuditErr("ausleihe", s.auditRepo.LogAusleihe(ctx, copy.ID, "", staffID, staffID))

	resp.Type = "ausleihe"
	resp.Book = copy
	if loan != nil {
		resp.DueDate = &loan.RueckgabeFrist
	}
	return resp, nil
}

// handleEigenrueckgabeMitarbeiter verbucht die Rückgabe eines Buchs, das auf denselben
// scannenden Mitarbeiter ausgeliehen ist, inkl. Vormerkungsaktivierung und Plugin-Event.
func (s *defaultLoanService) handleEigenrueckgabeMitarbeiter(ctx context.Context, tx pgx.Tx, copy *repository.BookCopy, activeLoan *repository.Loan, staffID string, resp *LoanResult) (*LoanResult, error) {
	if err := s.loanRepo.ReturnLoanTx(ctx, tx, activeLoan.ID, staffID, false); err != nil {
		return nil, err
	}

	// Handapparat-Rückgabe: kein Schüler-Ausleiher (activeLoan.SchuelerID == nil) → nichts
	// auszuschließen; die Zuteilung läuft normal über die Warteschlange.
	s.processReturnVormerkungTx(ctx, tx, copy, resp, activeLoan.SchuelerID)

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	logAuditErr(actionReturn, s.auditRepo.LogRueckgabe(ctx, copy.ID, "", *activeLoan.AusleiherBenutzerID, staffID))

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

// handleSchuelerRueckgabe verbucht die reguläre Rückgabe eines von einem Schüler (oder
// einer anderen Person) ausgeliehenen Buchs inkl. Vormerkungsaktivierung und Plugin-Event.
func (s *defaultLoanService) handleSchuelerRueckgabe(ctx context.Context, tx pgx.Tx, copy *repository.BookCopy, activeLoan *repository.Loan, staffID string, resp *LoanResult) (*LoanResult, error) {
	var borrowerStudent *repository.Student
	if activeLoan.SchuelerID != nil {
		var err error
		borrowerStudent, err = s.studentRepo.GetByID(ctx, *activeLoan.SchuelerID)
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
	s.processReturnVormerkungTx(ctx, tx, copy, resp, activeLoan.SchuelerID)

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	// Revisionssicheres Audit-Log schreiben
	if activeLoan.SchuelerID != nil {
		logAuditErr(actionReturn, s.auditRepo.LogRueckgabe(ctx, copy.ID, *activeLoan.SchuelerID, "", staffID))
	} else if activeLoan.AusleiherBenutzerID != nil {
		logAuditErr(actionReturn, s.auditRepo.LogRueckgabe(ctx, copy.ID, "", *activeLoan.AusleiherBenutzerID, staffID))
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
