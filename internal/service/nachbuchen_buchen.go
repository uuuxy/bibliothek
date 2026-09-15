package service

import (
	"context"
	"errors"
	"fmt"
	"log"

	"bibliothek/repository"

	"github.com/jackc/pgx/v5"
)

// nachbuchenRueckgabe: Absicht „Rückgabe". Lag das Buch bei jemandem, endet die Ausleihe
// zum Scan-Zeitpunkt — auch wenn es nicht die Person des Eintrags war (Fremdrückgabe).
// War es nicht verliehen, gibt es nichts zu buchen: nur_reaktiviert, wenn es abgeschrieben
// war (dann ist die Forderung beendet), sonst nicht_gebucht.
func (s *defaultLoanService) nachbuchenRueckgabe(ctx context.Context, tx pgx.Tx, l *nachbuchLage) (*NachbuchErgebnis, error) {
	if l.activeLoan == nil {
		if !l.reaktiviert {
			rollbackStill(ctx, tx)
			return s.meldeAbweisung(ctx, l, repository.NachbuchNichtGebucht, "Buch war nicht ausgeliehen")
		}
		hinweis := l.befund.AufsichtHinweis()
		grund := "Buch war abgeschrieben und ist wieder im Umlauf"
		if l.befund.StornierteForderungen > 0 {
			grund = fmt.Sprintf("Buch war abgeschrieben; Forderung über %.2f € storniert", l.befund.StornierterBetrag)
		}
		if hinweis != "" {
			grund = hinweis
		}
		if err := repository.SchreibeNachbuchMeldung(ctx, tx, s.meldung(l, repository.NachbuchNurReaktiviert, grund)); err != nil {
			return nil, err
		}
		if err := tx.Commit(ctx); err != nil {
			return nil, err
		}
		return &NachbuchErgebnis{Ergebnis: repository.NachbuchNurReaktiviert, Grund: grund, AufsichtHinweis: hinweis}, nil
	}

	resp := &LoanResult{}
	fremd := (l.student != nil || l.teacher != nil) && !gehoert(l)
	vorbesitzer, vorbesitzerUser, err := s.nimmZurueck(ctx, tx, l, fremd, resp)
	if errors.Is(err, errScanVeraltet) {
		rollbackStill(ctx, tx)
		return s.meldeAbweisung(ctx, l, repository.NachbuchVeraltet, "Rückgabe liegt vor der Ausleihe")
	}
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	resp.Type = "rueckgabe"
	resp.Book = l.copy
	resp.LoanID = &l.activeLoan.ID
	resp.Fremdrueckgabe = fremd
	resp.Vorbesitzer, resp.VorbesitzerUser = vorbesitzer, vorbesitzerUser
	if !fremd {
		resp.Student, resp.Teacher = l.student, l.teacher
	}
	return &NachbuchErgebnis{Ergebnis: repository.NachbuchZurueckgegeben, Result: resp}, nil
}

// nachbuchenAusleihe: Absicht „Ausleihe". Beim selben Kind wird nichts umgekehrt
// (bereits_ausgeliehen). Lag das Buch bei jemand anderem, wird dort zurückgenommen — VOR
// dem Savepoint, damit die Rücknahme bleibt, wenn die neue Ausleihe an Sperre, Limit oder
// Vormerkung scheitert (Entscheidung Peter, 13.09.2026, c).
func (s *defaultLoanService) nachbuchenAusleihe(ctx context.Context, tx pgx.Tx, l *nachbuchLage) (*NachbuchErgebnis, error) {
	if l.activeLoan != nil && gehoert(l) {
		rollbackStill(ctx, tx)
		resp := &LoanResult{Type: "ausleihe", Book: l.copy, Student: l.student, Teacher: l.teacher, LoanID: &l.activeLoan.ID}
		frist := l.activeLoan.RueckgabeFrist
		resp.DueDate = &frist
		return &NachbuchErgebnis{Ergebnis: repository.NachbuchBereitsAusgeliehen, Result: resp}, nil
	}

	resp := &LoanResult{}
	var vorbesitzer *repository.Student
	var vorbesitzerUser *repository.User
	if l.activeLoan != nil {
		var err error
		vorbesitzer, vorbesitzerUser, err = s.nimmZurueck(ctx, tx, l, true, resp)
		if errors.Is(err, errScanVeraltet) {
			rollbackStill(ctx, tx)
			return s.meldeAbweisung(ctx, l, repository.NachbuchVeraltet, "Scan liegt vor der Ausleihe des Vorbesitzers")
		}
		if err != nil {
			return nil, err
		}
	}

	// Savepoint: Was ab hier scheitert, nimmt die Rücknahme nicht mit.
	// Savepoint: Was ab hier scheitert, nimmt die Rücknahme nicht mit.
	sp, err := tx.Begin(ctx)
	if err != nil {
		return nil, err
	}
	chkCtx, err := s.nachbuchKontext(ctx, tx, l)
	if err != nil {
		return nil, err
	}
	if err := s.nachbuchSchranken(ctx, sp, l, chkCtx); err != nil {
		if !errors.Is(err, ErrBlocked) && !errors.Is(err, ErrConflict) {
			return nil, err
		}
		if err := sp.Rollback(ctx); err != nil {
			return nil, err
		}
		if err := tx.Commit(ctx); err != nil { // die Rücknahme beim Vorbesitzer bleibt
			return nil, err
		}
		return s.meldeAbweisung(ctx, l, repository.NachbuchNichtGebucht, err.Error())
	}

	var loan *repository.Loan
	if l.student != nil {
		loan, err = repository.CreateLoanZumTx(ctx, sp, l.copy.ID, l.student.ID, l.e.StaffID, chkCtx.dueTime, &l.gescannt)
	} else {
		loan, err = repository.CreateUserLoanZumTx(ctx, sp, l.copy.ID, l.teacher.ID, l.e.StaffID, chkCtx.dueTime, true, &l.gescannt)
	}
	if err != nil {
		if errors.Is(err, repository.ErrAusleiheKonflikt) {
			if rbErr := sp.Rollback(ctx); rbErr != nil {
				return nil, rbErr
			}
			if err := tx.Commit(ctx); err != nil {
				return nil, err
			}
			return s.meldeAbweisung(ctx, l, repository.NachbuchNichtGebucht, "Exemplar wurde soeben an einem anderen Arbeitsplatz verbucht")
		}
		return nil, err
	}
	if l.student != nil {
		entferneErfuellteVormerkung(ctx, sp, l.copy, l.student.ID, resp)
		if err := s.auditRepo.LogAusleihe(ctx, sp, l.copy.ID, l.student.ID, "", l.e.StaffID); err != nil {
			return nil, err
		}
	} else if err := s.auditRepo.LogAusleihe(ctx, sp, l.copy.ID, "", l.teacher.ID, l.e.StaffID); err != nil {
		return nil, err
	}
	if err := sp.Commit(ctx); err != nil {
		return nil, err
	}

	ergebnis := repository.NachbuchAusgeliehen
	if vorbesitzer != nil || vorbesitzerUser != nil {
		ergebnis = repository.NachbuchUmgebucht
		m := s.meldung(l, ergebnis, "lag bei jemand anderem — dort zurückgenommen, neu ausgeliehen")
		if vorbesitzer != nil {
			m.VorbesitzerSchuelerID = &vorbesitzer.ID
		}
		if vorbesitzerUser != nil {
			m.VorbesitzerBenutzerID = &vorbesitzerUser.ID
		}
		if err := repository.SchreibeNachbuchMeldung(ctx, tx, m); err != nil {
			return nil, err
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	resp.Type = "ausleihe"
	resp.Book = l.copy
	resp.Student, resp.Teacher = l.student, l.teacher
	resp.Vorbesitzer, resp.VorbesitzerUser = vorbesitzer, vorbesitzerUser
	resp.Fremdrueckgabe = vorbesitzer != nil || vorbesitzerUser != nil
	if loan != nil {
		resp.DueDate = &loan.RueckgabeFrist
		resp.LoanID = &loan.ID
	}
	return &NachbuchErgebnis{Ergebnis: ergebnis, Result: resp, AufsichtHinweis: l.befund.AufsichtHinweis()}, nil
}

// errScanVeraltet: die Rückgabe läge vor der Ausleihe (check_return_date).
var errScanVeraltet = errors.New("scan veraltet")

// nimmZurueck beendet die aktive Ausleihe zum Scan-Zeitpunkt, bedient die Vormerkung und
// schreibt das Protokoll; liefert den Vorbesitzer für Antwort und Meldung.
func (s *defaultLoanService) nimmZurueck(ctx context.Context, tx pgx.Tx, l *nachbuchLage, fremd bool, resp *LoanResult) (*repository.Student, *repository.User, error) {
	aktiv := l.activeLoan
	if err := repository.ReturnLoanZumTx(ctx, tx, aktiv.ID, l.e.StaffID, fremd, &l.gescannt); err != nil {
		if istPruefungVerletzt(err) {
			return nil, nil, errScanVeraltet
		}
		return nil, nil, err
	}
	if err := s.processReturnVormerkungTx(ctx, tx, l.copy, resp, aktiv.SchuelerID); err != nil {
		return nil, nil, err
	}
	var vorbesitzer *repository.Student
	var vorbesitzerUser *repository.User
	switch {
	case aktiv.SchuelerID != nil:
		if err := s.auditRepo.LogRueckgabe(ctx, tx, l.copy.ID, *aktiv.SchuelerID, "", l.e.StaffID); err != nil {
			return nil, nil, err
		}
		if fremd {
			st, err := s.studentRepo.GetByID(ctx, *aktiv.SchuelerID)
			if err != nil {
				return nil, nil, err
			}
			vorbesitzer = st
		}
	case aktiv.AusleiherBenutzerID != nil:
		if err := s.auditRepo.LogRueckgabe(ctx, tx, l.copy.ID, "", *aktiv.AusleiherBenutzerID, l.e.StaffID); err != nil {
			return nil, nil, err
		}
		if fremd {
			u := &repository.User{}
			err := tx.QueryRow(ctx, "SELECT b.id, b.vorname, b.nachname, b.rolle::text FROM benutzer b WHERE b.id = $1", *aktiv.AusleiherBenutzerID).
				Scan(&u.ID, &u.Vorname, &u.Nachname, &u.Rolle)
			if err != nil && !errors.Is(err, pgx.ErrNoRows) {
				return nil, nil, err
			}
			if err == nil {
				vorbesitzerUser = u
			}
		}
	}
	return vorbesitzer, vorbesitzerUser, nil
}

// nachbuchKontext baut den Ausleih-Kontext mit der Frist ab dem Scan-Zeitpunkt.
func (s *defaultLoanService) nachbuchKontext(ctx context.Context, tx pgx.Tx, l *nachbuchLage) (*checkoutContext, error) {
	if l.student != nil {
		frist, err := s.resolveCheckoutDueDateAm(ctx, l.copy, l.student.Klasse, l.gescannt)
		if err != nil {
			return nil, err
		}
		return &checkoutContext{borrowerType: "student", borrowerID: l.student.ID, student: l.student, dueTime: frist}, nil
	}
	return &checkoutContext{borrowerType: "teacher", borrowerID: l.teacher.ID, teacher: l.teacher,
		dueTime: TagesEndeInSchulzeitzone(l.gescannt.AddDate(1, 0, 0))}, nil
}

// nachbuchSchranken sind dieselben Schranken wie am Online-Scan — Sperren, Ausleihlimit,
// fremde Vormerkung —, die Zählungen laufen über die Transaktion des Eintrags. Ein
// Sperr-Override gibt es beim Nachbuchen nicht: Niemand steht daneben.
func (s *defaultLoanService) nachbuchSchranken(ctx context.Context, q pgx.Tx, l *nachbuchLage, chkCtx *checkoutContext) error {
	if l.student != nil {
		if err := s.pruefeSchuelerAusleihbarMit(ctx, q, l.student, l.student.ID, l.e.StaffID, false); err != nil {
			return err
		}
		anzahl, err := s.zaehleAktiveSchuelerAusleihen(ctx, q, chkCtx)
		if err != nil {
			return err
		}
		if err := s.pruefeSchuelerAusleihlimit(ctx, chkCtx, l.copy, anzahl, true); err != nil {
			return err
		}
	}
	return s.pruefeVormerkungKonflikt(ctx, q, l.copy.ID, chkCtx, true)
}

// rollbackStill verwirft die Transaktion, wenn fachlich nichts zu buchen ist. Ein Fehler
// beim Rollback wird protokolliert, nicht gemeldet: Gebucht wurde ohnehin nichts, und der
// Aufrufer soll seine fachliche Antwort behalten.
func rollbackStill(ctx context.Context, tx pgx.Tx) {
	if err := tx.Rollback(ctx); err != nil && !errors.Is(err, pgx.ErrTxClosed) {
		log.Printf("nachbuchen: Rollback fehlgeschlagen: %v", err)
	}
}

// LoanResultAlsOmnibox überträgt ein Ausleih-Ergebnis in die Omnibox-Form — dieselbe
// Abbildung, die der Online-Scan nutzt (mapLoanResult), damit die Nachbuch-Antwort für
// die Theke dieselbe Gestalt hat.
func LoanResultAlsOmnibox(lr *LoanResult) *OmniboxResult {
	resp := &OmniboxResult{}
	(&defaultOmniboxService{}).mapLoanResult(lr, resp)
	return resp
}
