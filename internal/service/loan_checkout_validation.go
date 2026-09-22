package service

import (
	"context"
	"fmt"
	"time"

	"bibliothek/repository"
)

// checkoutContext holds the resolved borrower information and the due time.
// checkoutContext ist der Ausleiher eines laufenden Vorgangs: EIN Leser (Migration 125).
// Bis dahin standen hier zwei Paare — borrowerType/borrowerID und student/teacher — und
// jede Fallunterscheidung im Ausleihpfad musste beide richtig bedienen.
type checkoutContext struct {
	borrowerID string
	// leser ist der Ausleiher. Über die Theke immer gesetzt; er trägt seine Art.
	leser   *repository.Student
	dueTime time.Time
}

// istSchueler sagt, ob die Schülerregeln greifen: Ausleihlimit, Vormerkungen und die
// Frist aus der Klasse. Sie hängen an der ART des Lesers, nicht mehr daran, aus welcher
// Tabelle er kam — das war vorher dasselbe und ist es seit Migration 125 nicht mehr.
func (c *checkoutContext) istSchueler() bool {
	return c.leser != nil && c.leser.Art == "schueler"
}

// logOverride schreibt einen Audit-Eintrag, wenn eine Ausleihsperre manuell
// ignoriert wurde.
func (s *defaultLoanService) logOverride(ctx context.Context, staffID, borrowerID, reason string) {
	logAuditErr(overrideBlockAction, s.auditRepo.LogAdminAktion(ctx, staffID, "OVERRIDE_BLOCK", "", map[string]any{
		"schueler_id": borrowerID,
		"reason":      reason,
	}))
}

// pruefeSchuelerAusleihbar prüft die Sperrgründe (Sperr-Flags, offene Schadensrechnungen
// und die Überfällig-Automatik). Ist overrideBlock gesetzt, wird die Sperre statt
// eines Fehlers nur revisionssicher protokolliert.
// pruefeSchuelerAusleihbar führt die vier Sperr-Checks der Reihe nach aus. Jeder Check ist
// in einen eigenen Helfer ausgelagert (reine Extract-Method, keine Logikänderung) — die
// Reihenfolge und das overrideBlock-/Audit-Verhalten sind identisch zum vorherigen Monolithen.
//
// lernmittel sagt, ob das Buch, um das es geht, ein Lernmittel ist — dann entfallen die
// zwei Automatiken (siehe pruefeSchuelerAusleihbarMit).
func (s *defaultLoanService) pruefeSchuelerAusleihbar(ctx context.Context, sObj *repository.Student, borrowerID, staffID string, overrideBlock, lernmittel bool) error {
	return s.pruefeSchuelerAusleihbarMit(ctx, s.pool, sObj, borrowerID, staffID, overrideBlock, lernmittel)
}

// pruefeSchuelerAusleihbarMit prüft dieselben vier Sperren über q — beim Nachbuchen die
// Transaktion des Eintrags (sonst zwei Verbindungen je Eintrag, OFFEN.md 6.1), am
// Online-Scan der Pool wie bisher.
//
// Lernmittel (Antwort der Schule vom 22.09.2026, docs/OFFEN.md 9.3 c): Die
// Lernmittelfreiheit in Hessen lässt keine automatische Sperre zu — auch keine, die jemand
// übergehen kann. Die zwei Automatiken (offene Forderung, Überfällig-Automatik) laufen
// deshalb nur für ein Buch der Schülerbücherei. Die zwei Schalter am Leser (gesperrt, von
// Hand gesperrt) gelten weiter: Sie sind die Entscheidung eines Menschen, keine Automatik.
func (s *defaultLoanService) pruefeSchuelerAusleihbarMit(ctx context.Context, q repository.DBQueryer, sObj *repository.Student, borrowerID, staffID string, overrideBlock, lernmittel bool) error {
	if err := s.pruefeGesperrt(ctx, sObj, borrowerID, staffID, overrideBlock); err != nil {
		return err
	}
	if err := s.pruefeManuellGesperrt(ctx, sObj, borrowerID, staffID, overrideBlock); err != nil {
		return err
	}
	if lernmittel {
		return nil
	}
	if err := s.pruefeOffeneSchaeden(ctx, q, borrowerID, staffID, overrideBlock); err != nil {
		return err
	}
	return s.pruefeUeberfaellig(ctx, q, borrowerID, staffID, overrideBlock)
}

// pruefeGesperrt blockt einen gesperrten Schüler; overrideBlock hebt die Sperre
// revisionssicher protokolliert auf.
func (s *defaultLoanService) pruefeGesperrt(ctx context.Context, sObj *repository.Student, borrowerID, staffID string, overrideBlock bool) error {
	if !sObj.IstGesperrt {
		return nil
	}
	// Grund mit ausgeben — ein gesperrter Schüler ohne sichtbaren Grund zwingt das
	// Personal sonst, in der Historie zu wühlen. block_reason ist bei Sperre garantiert
	// gefüllt (chk_schueler_block_reason); der Fallback greift nur bei Altbeständen.
	reason := "Grund nicht erfasst"
	if sObj.BlockReason != nil && *sObj.BlockReason != "" {
		reason = *sObj.BlockReason
	}
	if !overrideBlock {
		return &SperrGrundFehler{Kern: UebergehbareSperre(fmt.Errorf("%w: Ausleihe gesperrt", ErrBlocked)), Grund: reason}
	}
	s.logOverride(ctx, staffID, borrowerID, "Ausleihsperre manuell ignoriert (gesperrt: "+reason+")")
	return nil
}

// pruefeManuellGesperrt behandelt die separate manuelle Sperre (is_manually_blocked).
func (s *defaultLoanService) pruefeManuellGesperrt(ctx context.Context, sObj *repository.Student, borrowerID, staffID string, overrideBlock bool) error {
	if !sObj.IsManuallyBlocked {
		return nil
	}
	reason := "ohne Grund"
	if sObj.BlockReason != nil && *sObj.BlockReason != "" {
		reason = *sObj.BlockReason
	}
	if !overrideBlock {
		return &SperrGrundFehler{Kern: UebergehbareSperre(fmt.Errorf("%w: Manuelle Sperre", ErrBlocked)), Grund: reason}
	}
	s.logOverride(ctx, staffID, borrowerID, "Ausleihsperre manuell ignoriert (Manuelle Sperre: "+reason+")")
	return nil
}

// pruefeOffeneSchaeden blockt bei unbezahlten, nicht stornierten Schadensrechnungen: Wer
// einen unbezahlten Schadensfall hat, darf nichts Neues ausleihen, bis die Schule entschädigt
// ist — sonst könnte man Bücher zerstören, die Rechnung ignorieren und sich am nächsten Tag
// neu eindecken. storniert_am setzt ist_bezahlt = true (repository/audit_system.go), daher
// genügt ist_bezahlt = false. overrideBlock möglich, wird dann revisionssicher protokolliert.
func (s *defaultLoanService) pruefeOffeneSchaeden(ctx context.Context, q repository.DBQueryer, borrowerID, staffID string, overrideBlock bool) error {
	offeneSchaeden, err := zaehleOffeneSchaeden(ctx, q, borrowerID)
	if err != nil {
		return err
	}
	if offeneSchaeden > 0 {
		if !overrideBlock {
			return UebergehbareSperre(fmt.Errorf("%w: %d unbezahlte(r) Schadensfall/-fälle offen", ErrBlocked, offeneSchaeden))
		}
		s.logOverride(ctx, staffID, borrowerID, fmt.Sprintf("Ausleihsperre manuell ignoriert (unbezahlte Schäden: %d)", offeneSchaeden))
	}
	return nil
}

// pruefeUeberfaellig setzt die Overdue-Sperr-Automatik um: ab MaxOverdueItems überfälligen
// Medien (älter als MaxOverdueDays) ist die Ausleihe gesperrt.
func (s *defaultLoanService) pruefeUeberfaellig(ctx context.Context, q repository.DBQueryer, borrowerID, staffID string, overrideBlock bool) error {
	settings, err := s.querySettings(ctx)
	if err != nil {
		return err
	}

	overdueCount, errOverdue := zaehleUeberfaelligeMedien(ctx, q, borrowerID, settings.MaxOverdueDays)
	if errOverdue != nil {
		return errOverdue
	}
	if overdueCount >= settings.MaxOverdueItems {
		if !overrideBlock {
			return UebergehbareSperre(fmt.Errorf("%w: %d überfällige Medien vorhanden (Sperr-Automatik)", ErrBlocked, overdueCount))
		}
		s.logOverride(ctx, staffID, borrowerID, fmt.Sprintf("Ausleihsperre manuell ignoriert (überfällig: %d Medien)", overdueCount))
	}
	return nil
}

// resolveBorrowerAndDueTime lädt den aktiven LESER und bestimmt seine Leihfrist.
//
// Die Sperrgründe prüft HandleUnifiedCheckout erst, wenn feststeht, ob es eine eigene
// Rückgabe ist (pruefeSchuelerAusleihbar) — wer sein Buch ZURÜCKgibt, darf gesperrt sein.
//
// Die Frist: Für einen Schüler aus der Klasse (Schuljahresende, Lernmittel abweichend),
// für alle anderen ein Jahr. Das ist unverändert die Regel von vor Migration 125 — dort
// hieß sie „Lehrerausleihe = Dauerleihgabe". Ob ein Kollege, der sich einen Roman
// mitnimmt, den wirklich ein Jahr behalten soll, ist eine Frage an den Betrieb und keine,
// die dieser Umbau nebenbei beantwortet (docs/OFFEN.md 5.16).
func (s *defaultLoanService) resolveBorrowerAndDueTime(ctx context.Context, copy *repository.BookCopy, leserID *string) (*checkoutContext, error) {
	if leserID == nil || *leserID == "" {
		return nil, fmt.Errorf("%w: Kein Leser aktiv", ErrInvalidState)
	}

	leser, err := s.studentRepo.GetLeserByID(ctx, *leserID)
	if err != nil {
		return nil, err
	}
	if leser == nil {
		return nil, fmt.Errorf("%w: Aktiver Leser nicht gefunden", ErrNotFound)
	}

	result := &checkoutContext{borrowerID: *leserID, leser: leser}
	if !result.istSchueler() {
		// Tagesende in der Schul-Zeitzone — dieselbe Normalisierung wie alle Fristen
		// (TagesEndeInSchulzeitzone).
		result.dueTime = TagesEndeInSchulzeitzone(s.heute().AddDate(1, 0, 0))
		return result, nil
	}

	dt, err := s.resolveCheckoutDueDate(ctx, copy, leser.Klasse)
	if err != nil {
		return nil, err
	}
	result.dueTime = dt
	return result, nil
}

// zaehleOffeneSchaeden zählt die unbezahlten, nicht stornierten Schadensfälle eines
// Schülers. Geteilte Quelle für Buch- (pruefeOffeneSchaeden) und Geräte-Pfad
// (pruefeGeraetAutomatikSperren) — storniert_am setzt ist_bezahlt=true, daher genügt
// die Flag-Prüfung.
func zaehleOffeneSchaeden(ctx context.Context, q repository.DBQueryer, schuelerID string) (int, error) {
	var n int
	err := q.QueryRow(ctx,
		`SELECT COUNT(*) FROM schadensfaelle WHERE schueler_id = $1 AND ist_bezahlt = false`,
		schuelerID).Scan(&n)
	return n, err
}

// zaehleUeberfaelligeMedien zählt die überfälligen (älter als maxOverdueDays), noch nicht
// zurückgegebenen BUCH-Medien eines Schülers (Handapparate und Geräte ausgenommen).
// Geteilte Quelle für die Überfällig-Sperr-Automatik in Buch- und Geräte-Pfad.
func zaehleUeberfaelligeMedien(ctx context.Context, q repository.DBQueryer, schuelerID string, maxOverdueDays int) (int, error) {
	var n int
	err := q.QueryRow(ctx, `
		SELECT COUNT(*)
		FROM ausleihen
		WHERE schueler_id = $1
		  AND rueckgabe_am IS NULL
		  AND rueckgabe_frist < CURRENT_TIMESTAMP - (INTERVAL '1 day' * $2)
		  AND ist_handapparat = false
		  AND geraet_id IS NULL
	`, schuelerID, maxOverdueDays).Scan(&n)
	return n, err
}
