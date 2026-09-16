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
// Scan (entschieden am 13.09.2026, c): Lag das Buch inzwischen bei jemand anderem, wird
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
	Schluessel string // Idempotenz-Schlüssel des Eintrags (UUID)
	Absicht    string // NachbuchAbsichtAusleihe | NachbuchAbsichtRueckgabe
	Barcode    string // Buch-Barcode wie gescannt (B-…, Ziffern, LMF-…)
	GescanntAm time.Time
	// UhrVersatz ist der gemessene Versatz der Uhr des Theken-Rechners: Serverzeit beim Empfang
	// der Portion minus Sendezeit des Rechners. Er wird auf GescanntAm addiert, bevor der Wächter
	// vergleicht — Bewegungsstempel sind Serverzeit. Ohne ihn beendete eine offline gebuchte
	// Rückgabe von einem vorgehenden Rechner die jüngere Online-Ausleihe eines anderen Kindes
	// (Rasterdurchgang 15.09.2026, OFFEN.md 5.15).
	UhrVersatz     time.Duration
	LeserID        *string // Person, wenn der Rechner sie beim Scan noch auflösen konnte
	AusweisBarcode *string // sonst der offline gescannte Ausweis
	// NachFremdrueckgabeVon: Der Online-Versand unter demselben Schlüssel hat nur die
	// Fremdrückgabe dieser Ausleihe gebucht (das Buch stand auf jemand anderem), seine Antwort
	// kam nicht an. Die Ausleihe an die Person des Eintrags fehlt noch. Der Wächter lässt den
	// Eintrag durch, wenn sich das Exemplar seit dieser Rückgabe nicht bewegt hat; gebucht wird
	// frühestens ab der Rückgabe. Bis zum 15.09.2026 hieß das SchluesselBekannt und galt für
	// JEDE gespeicherte Antwort — auch für eine vollständige Ausleihe, nach der das Buch
	// zurückgegeben worden war: Das Nachbuchen lieh es erneut aus (OFFEN.md 5.15).
	NachFremdrueckgabeVon *string
	StaffID               string
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
	leser       *repository.Student
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
	gescannt := e.GescanntAm.Add(e.UhrVersatz).In(schoolLocation())
	if gescannt.After(jetzt) {
		gescannt = jetzt // höchstens Serverzeit — auch nach der Umrechnung datiert nichts vor
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
	if e.Absicht == NachbuchAbsichtAusleihe && l.leser == nil {
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

	// Sperren: Leser, dann die Ausleihe des Exemplars, dann das Exemplar.
	if l.leser != nil {
		if _, err := tx.Exec(ctx, "SELECT id FROM leser WHERE id = $1 FOR UPDATE", l.leser.ID); err != nil {
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

	// Die Ausnahme: Der Online-Versand hat nur die Fremdrückgabe gebucht, die Ausleihe fehlt.
	// Der Scan wird auf den Zeitpunkt dieser Rückgabe gelegt. Die Ausleihe überlappt die
	// beendete dann nicht, und der Wächter darunter entscheidet wie immer: Ist die Rückgabe
	// noch die letzte Bewegung, kommt der Eintrag durch; hat sich das Exemplar danach bewegt
	// (Rückgabe, Aussonderung), liegt der Stempel später, und er meldet „veraltet". Das trägt
	// nur, weil der Stempel nie rückwärts läuft (repository.sqlStempelVor) und jede Bewegung
	// stempelt (docs/invarianten.md).
	// „Nicht verliehen" ist trotzdem nötig: Eine Ausleihe, die selbst so nachgeholt wurde, liegt
	// auf demselben Rückgabezeitpunkt und schiebt den Stempel nicht darüber hinaus — ein zweiter
	// Eintrag mit Verweis auf dieselbe Fremdrückgabe käme sonst durch und buchte sie um
	// (TestNachbuchen_WaechterUndFremdrueckgabe).
	if e.NachFremdrueckgabeVon != nil && e.Absicht == NachbuchAbsichtAusleihe && l.activeLoan == nil {
		rueckgabe, err := repository.LiesFremdrueckgabeZeitpunkt(ctx, tx, *e.NachFremdrueckgabeVon, copy.ID)
		if err != nil {
			return nil, err
		}
		if rueckgabe != nil && gescannt.Before(*rueckgabe) {
			gescannt = *rueckgabe
			l.gescannt = gescannt
		}
	}

	// Der Wächter: Ein Scan, der älter ist als die letzte Bewegung, beschreibt eine
	// Wirklichkeit, die es nicht mehr gibt.
	if letzteBewegung != nil && gescannt.Before(*letzteBewegung) {
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
// offline gescannte Ausweis. Bleibt der Ausweis unbekannt, steht sein Text in der Meldung.
//
// Bis Migration 125 stand hier zweimal dasselbe — einmal für Schüler, einmal für
// Lehrkräfte, mit der Reihenfolge „Schüler vor Lehrkraft" als stiller Entscheidung für
// den Fall, dass eine Nummer in beiden Tabellen steht. Diesen Fall gibt es nicht mehr.
func (s *defaultLoanService) loesePerson(ctx context.Context, l *nachbuchLage) error {
	e := l.e
	if e.LeserID != nil && *e.LeserID != "" {
		leser, err := s.studentRepo.GetLeserByID(ctx, *e.LeserID)
		if err != nil {
			return err
		}
		l.leser = leser
		return nil
	}
	if e.AusweisBarcode == nil || *e.AusweisBarcode == "" {
		return nil
	}
	leser, err := s.studentRepo.GetLeserByBarcode(ctx, *e.AusweisBarcode)
	if err != nil {
		return err
	}
	if leser != nil {
		l.leser = leser
		return nil
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
	if l.leser != nil {
		m.AusleiherSchuelerID = &l.leser.ID
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
	if l.activeLoan == nil || l.leser == nil {
		return false
	}
	return l.activeLoan.SchuelerID != nil && *l.activeLoan.SchuelerID == l.leser.ID
}
