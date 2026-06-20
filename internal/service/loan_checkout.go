package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"bibliothek/plugins"
	"bibliothek/repository"

	"github.com/jackc/pgx/v5"
)

// HandleUnifiedCheckout wickelt die Ausleihe eines Buchexemplars an einen aktiven Schüler oder Lehrer ab.
// Wenn das Buch bereits ausgeliehen ist, entscheidet die Methode, ob es sich um eine reguläre Rückgabe handelt
// (Ausleiher scannt sein eigenes Buch) oder um eine Fremdrückgabe mit anschließender Neuausleihe.
func (s *defaultLoanService) HandleUnifiedCheckout(
	ctx context.Context,
	copy *repository.BookCopy,
	activeStudentID *string,
	activeTeacherID *string,
	staffID string,
) (*LoanResult, error) {
	resp := &LoanResult{}

	// Sicherheitsschranke: Nur ausleihbare Exemplare dürfen verarbeitet werden
	if !copy.IstAusleihbar {
		return nil, fmt.Errorf("%w: dieses Buchexemplar ist nicht ausleihbar", ErrInvalidState)
	}

	var borrowerID string
	var borrowerType string
	var student *repository.Student
	var teacher *repository.User
	var dueTime time.Time

	// 1. Ermitteln, wer das Buch ausleihen möchte (Schüler oder Lehrkraft)
	if activeStudentID != nil && *activeStudentID != "" {
		borrowerType = "student"
		borrowerID = *activeStudentID
		sObj, err := s.studentRepo.GetByID(ctx, borrowerID)
		if err != nil {
			return nil, err
		}
		if sObj == nil {
			return nil, fmt.Errorf("%w: Aktives Schülerprofil nicht gefunden", ErrNotFound)
		}
		// Wenn der Schüler gesperrt ist (z. B. wegen Mahnungen), darf er nichts ausleihen
		if sObj.IstGesperrt {
			return nil, fmt.Errorf("%w: Die Ausleihe für diese/n Schüler/in ist gesperrt", ErrBlocked)
		}
		student = sObj

		// Fälligkeitsdatum anhand der Ausleihregeln ermitteln (berücksichtigt Medienart, LMF und Leseclub)
		dt, err := s.resolveCheckoutDueDate(ctx, copy)
		if err != nil {
			return nil, err
		}
		dueTime = dt
	} else if activeTeacherID != nil && *activeTeacherID != "" {
		borrowerType = "teacher"
		borrowerID = *activeTeacherID
		teacher = &repository.User{}
		// Verifizieren, ob die Lehrkraft existiert und aktiv ist
		err := s.pool.QueryRow(ctx, `
			SELECT b.id, b.barcode_id, b.vorname, b.nachname, br.rolle 
			FROM benutzer b JOIN benutzer_rollen br ON b.id = br.benutzer_id
			WHERE b.id = $1 AND br.rolle = 'LEHRER' AND b.aktiv = true LIMIT 1
		`, borrowerID).Scan(&teacher.ID, &teacher.BarcodeID, &teacher.Vorname, &teacher.Nachname, &teacher.Rolle)
		if err != nil {
			return nil, fmt.Errorf("%w: Aktives Lehrerprofil nicht gefunden", ErrNotFound)
		}
		// Für Lehrkräfte gilt standardmäßig eine Leihfrist von 1 Jahr (Dauerleihgabe / Handapparat)
		dueTime = time.Now().AddDate(1, 0, 0)
	} else {
		return nil, fmt.Errorf("%w: Weder Schüler noch Lehrer aktiv", ErrInvalidState)
	}

	// 2. Datenbanktransaktion starten, um Nebenläufigkeitsfehler (Race Conditions) zu verhindern
	tx, err := s.loanRepo.BeginTx(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	var activeLoansCount int
	if borrowerType == "student" {
		// Row-Level Lock auf den Schülereintrag setzen (`FOR UPDATE`), um zeitgleiche Ausleihen
		// auf denselben Schüler im parallelen Request zu synchronisieren.
		if _, err = tx.Exec(ctx, "SELECT id FROM schueler WHERE id = $1 FOR UPDATE", borrowerID); err != nil {
			return nil, err
		}
		// Ermitteln, wie viele reguläre Bücher (ausgenommen Lernmittelfreiheit 'lmf-') der Schüler aktuell besitzt
		err = tx.QueryRow(ctx, `
			SELECT COUNT(*) 
			FROM ausleihen a
			JOIN buecher_exemplare be ON a.exemplar_id = be.id
			JOIN buecher_titel bt ON be.titel_id = bt.id
			WHERE a.schueler_id = $1 
			  AND a.rueckgabe_am IS NULL
			  AND LOWER(bt.titel) NOT LIKE 'lmf-%'
		`, borrowerID).Scan(&activeLoansCount)
		if err != nil {
			return nil, err
		}
	}

	// Aktuellen Ausleihstatus dieses Buchexemplars in der Transaktion prüfen
	activeLoan, err := s.loanRepo.GetActiveLoanByCopyIDTx(ctx, tx, copy.ID)
	if err != nil {
		return nil, err
	}

	// Prüfen, ob der aktive Ausleiher das Buch bereits ausgeliehen hat (dann ist es eine Rückgabe)
	isReturningThis := false
	if activeLoan != nil {
		if borrowerType == "student" && activeLoan.SchuelerID != nil && *activeLoan.SchuelerID == borrowerID {
			isReturningThis = true
		} else if borrowerType == "teacher" && activeLoan.AusleiherBenutzerID != nil && *activeLoan.AusleiherBenutzerID == borrowerID {
			isReturningThis = true
		}
	}

	// 3. Ausleihlimit für Schüler prüfen
	if borrowerType == "student" {
		settings, err := s.querySettings(ctx)
		if err != nil {
			return nil, err
		}

		isLMF := strings.HasPrefix(strings.ToLower(copy.Titel), "lmf-")

		// Ausleihlimit (meist max 5 Bücher) greift nur bei regulärem Bestand (kein LMF-Schulbuch)
		// und nur, wenn der Schüler das Buch nicht gerade zurückgibt.
		if !isLMF && activeLoansCount >= settings.MaxAusleihenSchueler {
			if !isReturningThis {
				return nil, fmt.Errorf("%w: Ausleihlimit von %d Bibliotheks-Büchern überschritten (aktuell: %d)", ErrBlocked, settings.MaxAusleihenSchueler, activeLoansCount)
			}
		}
	}

	// 4. Reservierungsprüfung (Vormerkung)
	// Falls das Buch für einen anderen Schüler zur Abholung bereitgestellt wurde und die Frist
	// noch läuft, blockieren wir die Ausleihe für Dritte.
	if !isReturningThis {
		var reservedSchuelerID string
		var resVorname, resNachname string
		err = tx.QueryRow(ctx, `
			SELECT v.schueler_id, s.vorname, s.nachname
			FROM vormerkungen v
			JOIN schueler s ON v.schueler_id = s.id
			WHERE v.bereitgestellt_exemplar_id = $1 
			  AND v.status = 'abholbereit' 
			  AND v.bereitgestellt_bis > CURRENT_TIMESTAMP
		`, copy.ID).Scan(&reservedSchuelerID, &resVorname, &resNachname)

		if err == nil {
			if borrowerType != "student" || borrowerID != reservedSchuelerID {
				return nil, fmt.Errorf("%w: Achtung: Dieses Exemplar ist noch für %s %s reserviert!", ErrConflict, resVorname, resNachname)
			}
		} else if !errors.Is(err, pgx.ErrNoRows) {
			return nil, err
		}
	}

	// Fall A: Buch ist frei -> Neue Ausleihe anlegen
	if activeLoan == nil {
		var loan *repository.Loan
		if borrowerType == "student" {
			loan, err = s.loanRepo.CreateLoanTx(ctx, tx, copy.ID, borrowerID, staffID, dueTime)
		} else {
			// Für Lehrer wird die Ausleihe als Handapparat (ist_handapparat = true) markiert
			loan, err = s.loanRepo.CreateUserLoanTx(ctx, tx, copy.ID, borrowerID, staffID, dueTime, true)
		}
		if err != nil {
			return nil, err
		}

		// Falls der Schüler eine Vormerkung auf diesen Titel hatte, löschen wir diese nun
		if borrowerType == "student" {
			_, _ = tx.Exec(ctx, "DELETE FROM vormerkungen WHERE titel_id = $1 AND schueler_id = $2", copy.TitelID, borrowerID)
		}

		if err := tx.Commit(ctx); err != nil {
			return nil, err
		}

		// Revisionssicheres Audit-Log schreiben
		if borrowerType == "student" {
			_ = s.auditRepo.LogAusleihe(ctx, copy.ID, borrowerID, "", staffID)
			resp.Student = student
		} else {
			_ = s.auditRepo.LogAusleihe(ctx, copy.ID, "", borrowerID, staffID)
			resp.Teacher = teacher
		}

		resp.Type = "ausleihe"
		resp.Book = copy
		if loan != nil {
			resp.DueDate = &loan.RueckgabeFrist
		}
		return resp, nil
	}

	// Fall B: Der aktuelle Ausleiher scannt das Buch erneut -> Rückgabe durchführen
	if isReturningThis {
		if err = s.loanRepo.ReturnLoanTx(ctx, tx, activeLoan.ID, staffID, false); err != nil {
			return nil, err
		}

		// Prüfen, ob eine Vormerkung für dieses Buch vorliegt und diese ggf. aktivieren
		s.processReturnVormerkungTx(ctx, tx, copy, resp)

		if err := tx.Commit(ctx); err != nil {
			return nil, err
		}

		// Revisionssicheres Audit-Log schreiben
		if borrowerType == "student" {
			_ = s.auditRepo.LogRueckgabe(ctx, copy.ID, borrowerID, "", staffID)
			resp.Student = student
		} else {
			_ = s.auditRepo.LogRueckgabe(ctx, copy.ID, "", borrowerID, staffID)
			resp.Teacher = teacher
		}

		// Event für Plugins triggern
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

	// Fall C: Fremdrückgabe (Buch ist bei Person A ausgeliehen, wird aber für Person B gescannt).
	// Zuerst wird das Buch für Person A zurückgebucht (ist_fremdrueckgabe = true),
	// danach wird es direkt für Person B ausgeliehen.
	var prevStudent *repository.Student
	var prevTeacher *repository.User

	// Infos über den Vorbesitzer (Person A) für die Rückmeldung ermitteln
	if activeLoan.SchuelerID != nil {
		prevStudent, _ = s.studentRepo.GetByID(ctx, *activeLoan.SchuelerID)
	} else if activeLoan.AusleiherBenutzerID != nil {
		prevTeacher = &repository.User{}
		err = tx.QueryRow(ctx, "SELECT b.id, b.vorname, b.nachname, COALESCE(br.rolle, 'HELFER') FROM benutzer b LEFT JOIN benutzer_rollen br ON b.id = br.benutzer_id WHERE b.id = $1 LIMIT 1", *activeLoan.AusleiherBenutzerID).Scan(&prevTeacher.ID, &prevTeacher.Vorname, &prevTeacher.Nachname, &prevTeacher.Rolle)
		if errors.Is(err, pgx.ErrNoRows) {
			prevTeacher = nil
		}
	}

	// Buch für den Vorbesitzer zurückbuchen (ist_fremdrueckgabe = true)
	if err = s.loanRepo.ReturnLoanTx(ctx, tx, activeLoan.ID, staffID, true); err != nil {
		return nil, err
	}

	// Direkt neue Ausleihe für den neuen aktiven Benutzer (Person B) anlegen
	var loan *repository.Loan
	if borrowerType == "student" {
		loan, err = s.loanRepo.CreateLoanTx(ctx, tx, copy.ID, borrowerID, staffID, dueTime)
	} else {
		loan, err = s.loanRepo.CreateUserLoanTx(ctx, tx, copy.ID, borrowerID, staffID, dueTime, true)
	}
	if err != nil {
		return nil, err
	}

	// Vormerkungen für den neuen Ausleiher löschen
	if borrowerType == "student" {
		_, _ = tx.Exec(ctx, "DELETE FROM vormerkungen WHERE titel_id = $1 AND schueler_id = $2", copy.TitelID, borrowerID)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	// Audit-Log für die automatische Rückgabe schreiben
	if activeLoan.SchuelerID != nil {
		_ = s.auditRepo.LogRueckgabe(ctx, copy.ID, *activeLoan.SchuelerID, "", staffID)
	} else if activeLoan.AusleiherBenutzerID != nil {
		_ = s.auditRepo.LogRueckgabe(ctx, copy.ID, "", *activeLoan.AusleiherBenutzerID, staffID)
	}

	// Audit-Log für die neue Ausleihe schreiben
	if borrowerType == "student" {
		_ = s.auditRepo.LogAusleihe(ctx, copy.ID, borrowerID, "", staffID)
		resp.Student = student
	} else {
		_ = s.auditRepo.LogAusleihe(ctx, copy.ID, "", borrowerID, staffID)
		resp.Teacher = teacher
	}

	// Event für Plugins triggern (Buch zurückgegeben)
	plugins.DispatchEvent(ctx, plugins.EventBookReturned, plugins.BookReturnedPayload{
		CopyID:       copy.ID,
		BarcodeID:    copy.BarcodeID,
		Titel:        copy.Titel,
		SchuelerID:   activeLoan.SchuelerID,
		BearbeiterID: staffID,
	})

	resp.Type = "ausleihe"
	resp.Book = copy
	if loan != nil {
		resp.DueDate = &loan.RueckgabeFrist
	}
	resp.Fremdrueckgabe = true
	resp.Vorbesitzer = prevStudent
	resp.VorbesitzerUser = prevTeacher
	return resp, nil
}
