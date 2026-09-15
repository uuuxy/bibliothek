package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"bibliothek/db"
	"bibliothek/repository"

	"github.com/jackc/pgx/v5/pgconn"
)

// Die Nachbuch-Tür (Stufe 2 des Offline-Baus, Commit 11, OFFEN.md 2.2).
//
// Ein offline gescannter Vorgang kommt später an. Gebucht wird die WIRKLICHKEIT, nicht der
// Scan (Entscheidung Peter, 13.09.2026, c): Lag das Buch inzwischen bei jemand anderem, wird
// dort zurückgenommen und neu ausgeliehen; beim selben Kind wird nichts umgekehrt; ist der
// Scan älter als die letzte Bewegung des Exemplars, wird er abgewiesen. Was vom Scan
// abweicht, steht als Meldung (Migration 117), nichts verschwindet still.
//
// Je Eintrag EINE Transaktion, Sperren in der Reihenfolge Schüler → Ausleihe → Exemplar
// (docs/invarianten.md, Abschnitt 1) — dieselbe wie der Online-Scan, sonst verklemmen
// sie sich gegeneinander. Die Rücknahme beim Vorbesitzer steht VOR dem Savepoint und
// bleibt, wenn die neue Ausleihe an Sperre, Limit oder Vormerkung scheitert; nur sie
// meldet dann nicht_gebucht. Zeit: der Scan-Zeitpunkt, höchstens Serverzeit — Ausleih- und
// Rückgabedatum, Frist und Bewegungsstempel rechnen ab dem Scan.

// NachbuchService ist die Tür des Nachbuchens; erfüllt vom Ausleih-Service.
type NachbuchService interface {
	Nachbuchen(ctx context.Context, e NachbuchEintrag) (*NachbuchErgebnis, error)
}

// NewNachbuchService baut die Tür auf demselben Rumpf wie der Ausleih-Service — dieselben
// Schranken, dieselben Schreiber. userRepo löst Lehrerausweise auf, die offline gescannt
// wurden.
func NewNachbuchService(pool db.PgxPoolIface, studentRepo repository.StudentRepository, bookRepo repository.BookRepository, userRepo repository.UserRepository, loanRepo repository.LoanRepository, auditRepo repository.AuditRepository) NachbuchService {
	return &defaultLoanService{pool: pool, studentRepo: studentRepo, bookRepo: bookRepo, userRepo: userRepo, loanRepo: loanRepo, auditRepo: auditRepo}
}

// Absichten eines Eintrags: was der Theken-Rechner beim Scan meinte.
const (
	NachbuchAbsichtAusleihe  = "ausleihe"
	NachbuchAbsichtRueckgabe = "rueckgabe"
)

// NachbuchEintrag ist ein Eintrag der Warteschlange eines Theken-Rechners.
type NachbuchEintrag struct {
	Schluessel     string // Idempotenz-Schlüssel des Eintrags (UUID)
	Absicht        string // NachbuchAbsichtAusleihe | NachbuchAbsichtRueckgabe
	Barcode        string // Buch-Barcode wie gescannt (B-…, Ziffern, LMF-…)
	GescanntAm     time.Time
	SchuelerID     *string // Person, wenn der Rechner sie beim Scan noch auflösen konnte
	LehrerID       *string
	AusweisBarcode *string // sonst der offline gescannte Ausweis
	// SchluesselBekannt: Der Server hat diesen Schlüssel schon einmal gesehen — ein
	// abgebrochener Online-Versand, der die Bewegung schon gebucht hat. Der Wächter lässt
	// den Eintrag dann durch, obwohl der Scan älter ist als die letzte Bewegung: Die
	// fehlende Hälfte wird nachgeholt.
	SchluesselBekannt bool
	StaffID           string
}

// NachbuchErgebnis ist die Antwort je Eintrag.
type NachbuchErgebnis struct {
	Ergebnis        string // repository.Nachbuch*
	Grund           string
	Result          *LoanResult // bei ausgeliehen, umgebucht, bereits_ausgeliehen, zurueckgegeben
	AufsichtHinweis string      // das Buch stand auf einem Bescheid bei der Aufsicht
}

// nachbuchLage ist der gesperrte Zustand eines Eintrags in seiner Transaktion.
type nachbuchLage struct {
	e           NachbuchEintrag
	gescannt    time.Time
	copy        *repository.BookCopy
	student     *repository.Student
	teacher     *repository.User
	ausweis     *string // nicht auflösbarer Ausweis-Barcode, als Text für die Meldung
	activeLoan  *repository.Loan
	befund      repository.RueckkehrBefund
	reaktiviert bool
}

// Nachbuchen bucht einen Eintrag. Fachliche Ausgänge kommen als Ergebnis zurück; ein
// Fehler ist ein Serverfehler (Datenbank), und der Eintrag bleibt auf dem Rechner liegen.
func (s *defaultLoanService) Nachbuchen(ctx context.Context, e NachbuchEintrag) (*NachbuchErgebnis, error) {
	if e.Absicht != NachbuchAbsichtAusleihe && e.Absicht != NachbuchAbsichtRueckgabe {
		return nil, fmt.Errorf("%w: unbekannte Absicht %q", ErrInvalidState, e.Absicht)
	}
	jetzt := s.heute()
	gescannt := e.GescanntAm.In(schoolLocation())
	if gescannt.After(jetzt) {
		gescannt = jetzt // höchstens Serverzeit — eine falsch gehende Theken-Uhr datiert nichts vor
	}
	l := &nachbuchLage{e: e, gescannt: gescannt}

	copy, err := s.loeseExemplar(ctx, e.Barcode)
	if err != nil {
		return nil, err
	}
	if copy == nil {
		return s.meldeAbweisung(ctx, l, repository.NachbuchNichtGebucht, "Buch unbekannt: "+e.Barcode)
	}
	l.copy = copy
	if err := s.loesePerson(ctx, l); err != nil {
		return nil, err
	}
	if e.Absicht == NachbuchAbsichtAusleihe && l.student == nil && l.teacher == nil {
		grund := "Ausweis unbekannt"
		if l.ausweis != nil {
			grund += ": " + *l.ausweis
		}
		return s.meldeAbweisung(ctx, l, repository.NachbuchNichtGebucht, grund)
	}

	tx, err := s.loanRepo.BeginTx(ctx)
	if err != nil {
		return nil, err
	}
	defer db.SafeRollback(ctx, tx)

	// Sperren: Schüler, dann die Ausleihe des Exemplars, dann das Exemplar.
	if l.student != nil {
		if _, err := tx.Exec(ctx, "SELECT id FROM schueler WHERE id = $1 FOR UPDATE", l.student.ID); err != nil {
			return nil, err
		}
	}
	if l.activeLoan, err = s.loanRepo.GetActiveLoanByCopyIDTx(ctx, tx, copy.ID); err != nil {
		return nil, err
	}
	var ausleihbar, ausgesondert bool
	var letzteBewegung *time.Time
	if err := tx.QueryRow(ctx, `SELECT ist_ausleihbar, ist_ausgesondert, letzte_bewegung_am
		FROM buecher_exemplare WHERE id = $1 FOR UPDATE`, copy.ID).Scan(&ausleihbar, &ausgesondert, &letzteBewegung); err != nil {
		return nil, err
	}

	// Der Wächter: Ein Scan, der älter ist als die letzte Bewegung, beschreibt eine
	// Wirklichkeit, die es nicht mehr gibt — außer der Server hat den Schlüssel schon
	// gesehen (abgebrochener Online-Versand).
	if letzteBewegung != nil && gescannt.Before(*letzteBewegung) && !e.SchluesselBekannt {
		db.SafeRollback(ctx, tx)
		return s.meldeAbweisung(ctx, l, repository.NachbuchVeraltet,
			fmt.Sprintf("Scan von %s liegt vor der letzten Bewegung des Exemplars (%s)",
				gescannt.Format("02.01.2006 15:04"), letzteBewegung.In(schoolLocation()).Format("02.01.2006 15:04")))
	}

	// Abgeschrieben oder gesperrt, aber nicht verliehen: zurückholen — Umlauf und
	// Forderung in dieser Transaktion (Baustein aus Commit 8).
	if (ausgesondert || !ausleihbar) && l.activeLoan == nil {
		if l.befund, err = repository.HoleExemplarZurueck(ctx, tx, copy.ID, e.StaffID, &gescannt); err != nil {
			return nil, err
		}
		l.reaktiviert = true
		copy.IstAusleihbar, copy.IstAusgesondert, copy.ZustandNotiz = true, false, ""
	}

	if e.Absicht == NachbuchAbsichtRueckgabe {
		return s.nachbuchenRueckgabe(ctx, tx, l)
	}
	return s.nachbuchenAusleihe(ctx, tx, l)
}

// loeseExemplar findet das Exemplar zum gescannten Barcode — direkt oder über die
// EAN-13 eines Littera-Etiketts (dieselben Formen wie der Online-Scan).
func (s *defaultLoanService) loeseExemplar(ctx context.Context, barcode string) (*repository.BookCopy, error) {
	copy, err := s.bookRepo.GetCopyByBarcode(ctx, barcode)
	if err != nil || copy != nil {
		return copy, err
	}
	if nummer, istEtikett := dekodiereLitteraEtikett(barcode); istEtikett {
		return s.bookRepo.GetCopyByBarcode(ctx, nummer)
	}
	return nil, nil
}

// loesePerson bestimmt, wer ausleiht: die schon aufgelöste Person des Eintrags, sonst der
// offline gescannte Ausweis (Schüler vor Lehrkraft). Bleibt der Ausweis unbekannt, steht
// sein Text in der Meldung.
func (s *defaultLoanService) loesePerson(ctx context.Context, l *nachbuchLage) error {
	e := l.e
	if e.SchuelerID != nil && *e.SchuelerID != "" {
		st, err := s.studentRepo.GetByID(ctx, *e.SchuelerID)
		if err != nil {
			return err
		}
		l.student = st
		return nil
	}
	if e.LehrerID != nil && *e.LehrerID != "" {
		lk, err := ladeAktiveLehrkraft(ctx, s.pool, *e.LehrerID)
		if err != nil && !errors.Is(err, ErrNotFound) {
			return err
		}
		l.teacher = lk
		return nil
	}
	if e.AusweisBarcode == nil || *e.AusweisBarcode == "" {
		return nil
	}
	st, err := s.studentRepo.GetByBarcode(ctx, *e.AusweisBarcode)
	if err != nil {
		return err
	}
	if st != nil {
		l.student = st
		return nil
	}
	if s.userRepo != nil {
		lk, err := s.userRepo.GetLehrerByBarcode(ctx, *e.AusweisBarcode)
		if err != nil {
			return err
		}
		if lk != nil {
			l.teacher = lk
			return nil
		}
	}
	l.ausweis = e.AusweisBarcode
	return nil
}

// meldeAbweisung schreibt nicht_gebucht/veraltet in eigener Transaktion — nach dem
// Rollback der Buchung, damit die Meldung steht, obwohl nichts gebucht wurde.
func (s *defaultLoanService) meldeAbweisung(ctx context.Context, l *nachbuchLage, ergebnis, grund string) (*NachbuchErgebnis, error) {
	if err := repository.SchreibeNachbuchMeldung(ctx, s.pool, s.meldung(l, ergebnis, grund)); err != nil {
		return nil, err
	}
	return &NachbuchErgebnis{Ergebnis: ergebnis, Grund: grund}, nil
}

// meldung baut die Meldung aus der Lage; Ausleiher ist die Person des Eintrags.
func (s *defaultLoanService) meldung(l *nachbuchLage, ergebnis, grund string) repository.NachbuchMeldungEingabe {
	m := repository.NachbuchMeldungEingabe{
		IdempotencyKey: l.e.Schluessel, Barcode: l.e.Barcode, Ergebnis: ergebnis, Grund: grund,
		AusweisText: l.ausweis, GescanntAm: l.gescannt,
	}
	if l.copy != nil {
		m.ExemplarID = &l.copy.ID
	}
	if l.student != nil {
		m.AusleiherSchuelerID = &l.student.ID
	}
	if l.teacher != nil {
		m.AusleiherBenutzerID = &l.teacher.ID
	}
	return m
}

// istPruefungVerletzt erkennt eine verletzte CHECK-Bedingung (23514) — beim Nachbuchen
// heißt das: Die Rückgabe läge vor der Ausleihe (check_return_date), der Scan ist veraltet.
func istPruefungVerletzt(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23514"
}

// gehoert sagt, ob die aktive Ausleihe der Person des Eintrags gehört.
func gehoert(l *nachbuchLage) bool {
	if l.activeLoan == nil {
		return false
	}
	if l.student != nil {
		return l.activeLoan.SchuelerID != nil && *l.activeLoan.SchuelerID == l.student.ID
	}
	if l.teacher != nil {
		return l.activeLoan.AusleiherBenutzerID != nil && *l.activeLoan.AusleiherBenutzerID == l.teacher.ID
	}
	return false
}
