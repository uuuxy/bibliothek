package service

import (
	"context"
	"errors"
	"fmt"
	"log"

	"bibliothek/pkg/betrag"
	"bibliothek/repository"

	"github.com/jackc/pgx/v5"
)

// nachbuchenRueckgabe: Absicht „Rückgabe". Lag das Buch bei jemandem, endet die Ausleihe
// zum Scan-Zeitpunkt — auch wenn es nicht die Person des Eintrags war (Fremdrückgabe).
// War es nicht verliehen, gibt es nichts zu buchen: nur_reaktiviert, wenn es abgeschrieben
// war (dann ist die Forderung beendet), sonst nicht_gebucht.
func (s *defaultLoanService) nachbuchenRueckgabe(ctx context.Context, tx pgx.Tx, l *nachbuchLage) (*NachbuchErgebnis, error) {
	if l.activeLoan == nil {
		return s.rueckgabeOhneAusleihe(ctx, tx, l)
	}

	resp := &LoanResult{}
	fremd := l.leser != nil && !gehoert(l)
	vorbesitzer, err := s.nimmZurueck(ctx, tx, l, fremd, resp)
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
	resp.Vorbesitzer = vorbesitzer
	if !fremd {
		resp.Student = l.leser
	}
	return &NachbuchErgebnis{Ergebnis: repository.NachbuchZurueckgegeben, Result: resp}, nil
}

// rueckgabeOhneAusleihe: Das Buch war nicht verliehen, es gibt nichts zurückzunehmen. War es
// abgeschrieben, ist es jetzt wieder im Umlauf (nur_reaktiviert), und die Meldung nennt, was mit
// der Forderung geschah; sonst nicht_gebucht.
func (s *defaultLoanService) rueckgabeOhneAusleihe(ctx context.Context, tx pgx.Tx, l *nachbuchLage) (*NachbuchErgebnis, error) {
	if !l.reaktiviert {
		rollbackStill(ctx, tx)
		return s.meldeAbweisung(ctx, l, repository.NachbuchNichtGebucht, "Buch war nicht ausgeliehen")
	}
	hinweis := l.befund.AufsichtHinweis()
	grund := "Buch war abgeschrieben und ist wieder im Umlauf"
	if l.befund.StornierteForderungen > 0 {
		grund = fmt.Sprintf("Buch war abgeschrieben; Forderung über %s storniert", betrag.Euro(l.befund.StornierterBetrag))
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

// nachbuchenAusleihe: Absicht „Ausleihe". Beim selben Kind wird nichts umgekehrt
// (bereits_ausgeliehen). Lag das Buch bei jemand anderem, wird dort zurückgenommen — vor
// dem Savepoint, damit die Rücknahme bleibt, wenn die neue Ausleihe an Sperre, Limit oder
// Vormerkung scheitert.
func (s *defaultLoanService) nachbuchenAusleihe(ctx context.Context, tx pgx.Tx, l *nachbuchLage) (*NachbuchErgebnis, error) {
	if l.activeLoan != nil && gehoert(l) {
		rollbackStill(ctx, tx)
		resp := &LoanResult{Type: "ausleihe", Book: l.copy, Student: l.leser, LoanID: &l.activeLoan.ID}
		frist := l.activeLoan.RueckgabeFrist
		resp.DueDate = &frist
		return &NachbuchErgebnis{Ergebnis: repository.NachbuchBereitsAusgeliehen, Result: resp}, nil
	}

	resp := &LoanResult{}
	var vorbesitzer *repository.Student
	if l.activeLoan != nil {
		var err error
		vorbesitzer, err = s.nimmZurueck(ctx, tx, l, true, resp)
		if errors.Is(err, errScanVeraltet) {
			rollbackStill(ctx, tx)
			return s.meldeAbweisung(ctx, l, repository.NachbuchVeraltet, "Scan liegt vor der Ausleihe des Vorbesitzers")
		}
		if err != nil {
			return nil, err
		}
	}

	// Savepoint: Was ab hier scheitert, nimmt die Rücknahme nicht mit.
	loan, abweisung, err := s.leiheImSavepointAus(ctx, tx, l, resp)
	if abweisung != nil || err != nil {
		return abweisung, err
	}

	ergebnis := repository.NachbuchAusgeliehen
	if vorbesitzer != nil {
		ergebnis = repository.NachbuchUmgebucht
		m := s.meldung(l, ergebnis, "lag bei jemand anderem — dort zurückgenommen, neu ausgeliehen")
		m.VorbesitzerSchuelerID = &vorbesitzer.ID
		if err := repository.SchreibeNachbuchMeldung(ctx, tx, m); err != nil {
			return nil, err
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	resp.Type = "ausleihe"
	resp.Book = l.copy
	resp.Student = l.leser
	resp.Vorbesitzer = vorbesitzer
	resp.Fremdrueckgabe = vorbesitzer != nil
	if loan != nil {
		resp.DueDate = &loan.RueckgabeFrist
		resp.LoanID = &loan.ID
	}
	return &NachbuchErgebnis{Ergebnis: ergebnis, Result: resp, AufsichtHinweis: l.befund.AufsichtHinweis()}, nil
}

// leiheImSavepointAus prüft die Schranken und legt die Ausleihe an, beides im Savepoint.
// Scheitert es an Sperre, Limit, Vormerkung oder an einer gleichzeitigen Buchung, ist der
// Eintrag abgewiesen: Die Abweisung kommt als zweiter Wert zurück, und die Transaktion ist dann
// abgeschlossen (weiseImSavepointAb).
func (s *defaultLoanService) leiheImSavepointAus(ctx context.Context, tx pgx.Tx, l *nachbuchLage, resp *LoanResult) (*repository.Loan, *NachbuchErgebnis, error) {
	sp, err := tx.Begin(ctx)
	if err != nil {
		return nil, nil, err
	}
	chkCtx, err := s.nachbuchKontext(ctx, tx, l)
	if err != nil {
		return nil, nil, err
	}
	if err := s.nachbuchSchranken(ctx, sp, l, chkCtx); err != nil {
		if !errors.Is(err, ErrBlocked) && !errors.Is(err, ErrConflict) {
			return nil, nil, err
		}
		abweisung, abwErr := s.weiseImSavepointAb(ctx, tx, sp, l, err.Error(), err)
		return nil, abweisung, abwErr
	}

	loan, err := repository.CreateLoanZumTx(ctx, sp, repository.CreateLoanParams{
		ExemplarID:     l.copy.ID,
		LeserID:        l.leser.ID,
		BearbeiterID:   l.e.StaffID,
		RueckgabeFrist: chkCtx.dueTime,
		IstDauerleihe:  !chkCtx.istSchueler(),
		Zeitpunkt:      &l.gescannt,
	})
	if errors.Is(err, repository.ErrAusleiheKonflikt) {
		abweisung, abwErr := s.weiseImSavepointAb(ctx, tx, sp, l, "Exemplar wurde soeben an einem anderen Arbeitsplatz verbucht", nil)
		return nil, abweisung, abwErr
	}
	if err != nil {
		return nil, nil, err
	}
	if chkCtx.istSchueler() {
		entferneErfuellteVormerkung(ctx, sp, l.copy, l.leser.ID, resp)
	}
	if err := s.auditRepo.LogAusleihe(ctx, sp, l.copy.ID, l.leser.ID, "", l.e.StaffID); err != nil {
		return nil, nil, err
	}
	if err := sp.Commit(ctx); err != nil {
		return nil, nil, err
	}
	return loan, nil, nil
}

// weiseImSavepointAb nimmt den Savepoint zurück und schließt die Transaktion ab — die
// Rücknahme beim Vorbesitzer bleibt —, dann schreibt es die Abweisung als Meldung. schranke
// ist der Fehler der Schranke, wenn es an einer lag; er reist mit dem Ergebnis zur Tür.
func (s *defaultLoanService) weiseImSavepointAb(ctx context.Context, tx, sp pgx.Tx, l *nachbuchLage, grund string, schranke error) (*NachbuchErgebnis, error) {
	if err := sp.Rollback(ctx); err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	abweisung, err := s.meldeAbweisung(ctx, l, repository.NachbuchNichtGebucht, grund)
	if err != nil {
		return nil, err
	}
	abweisung.Schranke = schranke
	return abweisung, nil
}

// errScanVeraltet: die Rückgabe läge vor der Ausleihe (check_return_date).
var errScanVeraltet = errors.New("scan veraltet")

// nimmZurueck beendet die aktive Ausleihe zum Scan-Zeitpunkt, bedient die Vormerkung und
// schreibt das Protokoll; liefert den Vorbesitzer für Antwort und Meldung.
func (s *defaultLoanService) nimmZurueck(ctx context.Context, tx pgx.Tx, l *nachbuchLage, fremd bool, resp *LoanResult) (*repository.Student, error) {
	aktiv := l.activeLoan
	if err := repository.ReturnLoanZumTx(ctx, tx, aktiv.ID, l.e.StaffID, fremd, &l.gescannt); err != nil {
		if istPruefungVerletzt(err) {
			return nil, errScanVeraltet
		}
		return nil, err
	}
	if err := s.processReturnVormerkungTx(ctx, tx, l.copy, resp, aktiv.SchuelerID); err != nil {
		return nil, err
	}
	if aktiv.SchuelerID == nil {
		return nil, nil
	}
	if err := s.auditRepo.LogRueckgabe(ctx, tx, l.copy.ID, *aktiv.SchuelerID, "", l.e.StaffID); err != nil {
		return nil, err
	}
	if !fremd {
		return nil, nil
	}
	return s.studentRepo.GetLeserByID(ctx, *aktiv.SchuelerID)
}

// nachbuchKontext baut den Ausleih-Kontext mit der Frist ab dem Scan-Zeitpunkt.
func (s *defaultLoanService) nachbuchKontext(ctx context.Context, tx pgx.Tx, l *nachbuchLage) (*checkoutContext, error) {
	chkCtx := &checkoutContext{borrowerID: l.leser.ID, leser: l.leser}
	if !chkCtx.istSchueler() {
		chkCtx.dueTime = TagesEndeInSchulzeitzone(l.gescannt.AddDate(1, 0, 0))
		return chkCtx, nil
	}
	frist, err := s.resolveCheckoutDueDateAm(ctx, l.copy, l.leser.Klasse, l.gescannt)
	if err != nil {
		return nil, err
	}
	chkCtx.dueTime = frist
	return chkCtx, nil
}

// nachbuchSchranken sind dieselben Schranken wie am Online-Scan — Sperren, Ausleihlimit,
// fremde Vormerkung —, die Zählungen laufen über die Transaktion des Eintrags. Übergehen
// gibt es beim Nachbuchen nicht: Niemand steht daneben.
func (s *defaultLoanService) nachbuchSchranken(ctx context.Context, q pgx.Tx, l *nachbuchLage, chkCtx *checkoutContext) error {
	if _, err := pruefeAusleihSperren(ctx, q, l.leser, l.copy.IstLernmittel, false); err != nil {
		return err
	}
	if chkCtx.istSchueler() {
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
