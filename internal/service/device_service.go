package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"bibliothek/db"
	"bibliothek/pkg/lmfplan"
	"bibliothek/repository"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// DeviceResult beschreibt das Ergebnis einer Geräte-Aktion wie Ausleihe, Rückgabe oder Checklisten-Anforderung.
type DeviceResult struct {
	// Type definiert den Typ des Ergebnisses (z. B. "ausleihe", "rueckgabe", "geraet_check").
	Type string
	// Geraet enthält die Stammdaten des betroffenen Geräts.
	Geraet *repository.Geraet
	// Student ist der LESER, für den die Aktion durchgeführt wurde.
	Student *repository.Student
	// DueDate gibt die berechnete Rückgabefrist für das Gerät an (nur bei Ausleihe).
	DueDate *time.Time
	// LoanID ist die ID des verknüpften Ausleihdatensatzes.
	LoanID *string
	// Fremdrueckgabe ist wahr, wenn das Gerät von einer anderen Person als dem Ausleiher zurückgegeben wurde.
	Fremdrueckgabe bool
	// Vorbesitzer speichert den LESER, der das Gerät zuletzt ausgeliehen hatte (bei Fremdrückgabe).
	Vorbesitzer *repository.Student
}

// DeviceService definiert die Schnittstelle für alle Aktionen rund um Hardware-Geräte (z. B. Laptops, Tablets).
type DeviceService interface {
	// HandleDeviceAction verarbeitet das Scannen eines Geräts. Je nach Zustand wird das Gerät
	// entweder ausgeliehen (falls frei) oder zurückgegeben (falls aktuell ausgeliehen).
	// Zudem wird geprüft, ob eine Zubehör-Checkliste vor der Ausleihe bestätigt werden muss.
	HandleDeviceAction(ctx context.Context, query string, activeLeserID *string, confirmedChecklist bool, staffID string) (*DeviceResult, error)
}

// defaultDeviceService ist die Standard-Implementierung des DeviceService.
type defaultDeviceService struct {
	pool        db.PgxPoolIface
	studentRepo repository.StudentRepository
	loanRepo    repository.LoanRepository
	auditRepo   repository.AuditRepository
}

// NewDeviceService erstellt eine neue Instanz des Standard-Geräteservice.
func NewDeviceService(pool db.PgxPoolIface, studentRepo repository.StudentRepository, loanRepo repository.LoanRepository, auditRepo repository.AuditRepository) DeviceService {
	return &defaultDeviceService{
		pool:        pool,
		studentRepo: studentRepo,
		loanRepo:    loanRepo,
		auditRepo:   auditRepo,
	}
}

// ladeGeraet lädt das Gerät anhand der Barcode-ID. Ob es verliehen werden darf, prüft
// pruefeGeraetAusleihbar, und zwar erst im Ausleih-Zweig: Bis zum 13.09.2026 stand die
// Prüfung hier, und ein verliehenes Gerät, das danach als defekt gemeldet wurde, ließ sich
// nicht mehr zurückgeben (geraet_rueckgabe_sperre_pg_test.go).
func (s *defaultDeviceService) ladeGeraet(ctx context.Context, query string) (repository.Geraet, error) {
	var g repository.Geraet
	err := s.pool.QueryRow(ctx, `
		SELECT id, modellname, seriennummer, barcode_id, zubehoer, ist_ausleihbar, ist_ausgesondert, zustand_notiz
		FROM geraete
		WHERE barcode_id = $1
	`, query).Scan(&g.ID, &g.Modellname, &g.Seriennummer, &g.BarcodeID, &g.Zubehoer, &g.IstAusleihbar, &g.IstAusgesondert, &g.ZustandNotiz)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return g, fmt.Errorf("%w: Gerät mit Barcode %s nicht gefunden", ErrNotFound, query)
		}
		return g, err
	}
	return g, nil
}

// pruefeGeraetAusleihbar hält ausgesonderte und gesperrte Geräte (z. B. defekt) von der
// AUSLEIHE fern. Die Rückgabe prüft sie nicht: Eine offene Ausleihe zu schließen ist immer
// richtig, auch für ein Gerät, das inzwischen defekt oder ausgesondert ist.
func pruefeGeraetAusleihbar(g repository.Geraet) error {
	if g.IstAusgesondert {
		return fmt.Errorf("%w: Gerät ist ausgesondert", ErrInvalidState)
	}
	if !g.IstAusleihbar {
		return fmt.Errorf("%w: Gerät ist aktuell gesperrt", ErrBlocked)
	}
	return nil
}

// ladeAkteur ermittelt den aktiven Schüler bzw. Lehrer und wendet die Sperren an — für
// die AUSLEIHE. Die Rückgabe nimmt ladeRueckgeber: Ein gesperrter Schüler muss sein Gerät
// zurückgeben können. Die Einstellungen, die die Sperr-Prüfung geladen hat, gehen mit
// zurück — aus ihnen rechnet leiheGeraetAus die Frist, ohne sie ein zweites Mal zu lesen.
func (s *defaultDeviceService) ladeAkteur(ctx context.Context, activeLeserID *string) (*repository.Student, *SystemEinstellungen, error) {
	leser, err := s.ladeRueckgeber(ctx, activeLeserID)
	if err != nil || leser == nil {
		return nil, nil, err
	}
	// BEIDE Sperr-Flags prüfen — wie der Buch-Pfad (pruefeGesperrt +
	// pruefeManuellGesperrt). Zuvor blockierte nur die System-Sperre (ist_gesperrt);
	// ein von der Bibliothek MANUELL gesperrter Leser (is_manually_blocked, etwa
	// wegen unbezahlter Schäden) konnte trotzdem ein Gerät ausleihen — und Geräte
	// sind wertvoller als Bücher. Geräte kennen bewusst KEIN override_block.
	if leser.IstGesperrt || leser.IsManuallyBlocked {
		return nil, nil, fmt.Errorf("%w: Diese Person ist gesperrt", ErrBlocked)
	}
	// Dieselben AUTOMATIK-Sperren wie der Buch-Pfad (Betreiber-Entscheidung
	// 19.08.2026): unbezahlte Schäden und die Überfällig-Automatik. Wer kein Buch
	// bekäme, bekommt auch kein iPad. Geräte kennen kein Override.
	//
	// Seit Migration 125 gelten sie für JEDEN Leser. Vorher lief eine Lehrkraft an
	// diesen Prüfungen vorbei — nicht als Entscheidung, sondern weil ihre Ausleihen in
	// einer anderen Spalte standen und der Zähler sie nicht sah.
	einst, err := pruefeGeraetAutomatikSperren(ctx, s.pool, leser.ID)
	if err != nil {
		return nil, nil, err
	}
	return leser, einst, nil
}

// ladeRueckgeber ermittelt den aktiven Leser aus dem gescannten Ausweis, ohne
// Sperrprüfung. Die Rückgabe braucht ihn nur, um eine Fremdrückgabe zu erkennen.
func (s *defaultDeviceService) ladeRueckgeber(ctx context.Context, activeLeserID *string) (*repository.Student, error) {
	if activeLeserID == nil || *activeLeserID == "" {
		return nil, nil
	}
	leser, err := s.studentRepo.GetLeserByID(ctx, *activeLeserID)
	if err != nil {
		return nil, err
	}
	if leser == nil {
		return nil, fmt.Errorf("%w: Aktiver Leser nicht gefunden", ErrNotFound)
	}
	return leser, nil
}

// ladeAktiveAusleihe sperrt und lädt die offene Ausleihe des Geräts (Row-Level-Lock via
// FOR UPDATE). hasActiveLoan ist false, wenn keine offene Ausleihe existiert.
func ladeAktiveAusleihe(ctx context.Context, tx pgx.Tx, geraetID string) (repository.Loan, bool, error) {
	var activeLoan repository.Loan
	err := tx.QueryRow(ctx, `
		SELECT id, geraet_id, schueler_id, ausgeliehen_am, rueckgabe_frist, rueckgabe_am, bearbeiter_id, ist_fremdrueckgabe, ist_handapparat
		FROM ausleihen
		WHERE geraet_id = $1 AND rueckgabe_am IS NULL
		FOR UPDATE
	`, geraetID).Scan(
		&activeLoan.ID, &activeLoan.GeraetID, &activeLoan.SchuelerID,
		&activeLoan.AusgeliehenAm, &activeLoan.RueckgabeFrist, &activeLoan.RueckgabeAm,
		&activeLoan.BearbeiterID, &activeLoan.IstFremdrueckgabe, &activeLoan.IstHandapparat,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return activeLoan, false, nil
		}
		return activeLoan, false, err
	}
	return activeLoan, true, nil
}

// leiheGeraetAus behandelt Fall A: das freie Gerät wird an den aktiven Leser ausgeliehen.
// geraeteLeihfristTage ist die Standard-Leihfrist für Hardware (2 Wochen).
const geraeteLeihfristTage = 14

// geraeteRueckgabeFrist ist die Geräte-Leihfrist als Tagesfrist wie die der Bücher
// (loan_rules.go): Tagesende in der Schul-Zeitzone (Europe/Berlin), und fällt der Tag auf ein
// Wochenende, einen Feiertag oder in die Ferien, der nächste Schultag. Ohne die
// Normalisierung fiel die Frist auf die Sekunde genau N Tage später in der Server-Zeitzone
// (im Docker-Container UTC); ein um 10:00 MESZ geliehenes Gerät wäre 08:00 UTC fällig, was
// Mahnläufe und die "heute/morgen fällig"-Anzeige verschob.
func geraeteRueckgabeFrist(now time.Time, kalender lmfplan.Ferientabelle) time.Time {
	return Tagesfrist(now, geraeteLeihfristTage, kalender)
}

func (s *defaultDeviceService) leiheGeraetAus(ctx context.Context, tx pgx.Tx, g *repository.Geraet, leser *repository.Student, einst *SystemEinstellungen, staffID string) (*DeviceResult, error) {
	if leser == nil {
		return nil, fmt.Errorf("%w: Bitte scannen Sie zuerst einen Ausweis", ErrInvalidState)
	}

	// Standard-Hardware-Leihfrist beträgt 14 Tage (2 Wochen), auf das Tagesende in der
	// Schul-Zeitzone normalisiert (analog zu den Buch-Fristen).
	sommerferien := ""
	if einst != nil {
		sommerferien = einst.Sommerferien
	}
	dueTime := geraeteRueckgabeFrist(time.Now(), lmfplan.FerientabelleAus(sommerferien))
	resp := &DeviceResult{Student: leser}
	var newLoanID string
	// Ein Schreiber für jeden Leser. Nicht-Schüler bekommen das Gerät als Dauerleihe —
	// dieselbe Regel wie beim Buch (erzeugeAusleihe).
	err := tx.QueryRow(ctx, `
		INSERT INTO ausleihen (geraet_id, schueler_id, rueckgabe_frist, bearbeiter_id, ist_handapparat)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id
	`, g.ID, leser.ID, dueTime, staffID, leser.Art != "schueler").Scan(&newLoanID)
	if err != nil {
		// Zwei gleichzeitige Scans desselben Geräts: Der Verlierer verletzt
		// uniq_ausleihen_aktiv_geraet (23505). Die Daten sind sicher (nur EINE aktive
		// Ausleihe entsteht), aber ohne dieses Mapping käme ein 500 "interner
		// Datenbankfehler" statt einer klaren 409-Meldung — analog zum Buch-Pfad
		// (loan.go: ON CONFLICT → ErrAusleiheKonflikt).
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return nil, fmt.Errorf("%w: Gerät ist bereits ausgeliehen (woanders verbucht)", ErrConflict)
		}
		return nil, err
	}

	// Revisionssicheres Audit-Log schreiben.
	if err := s.auditRepo.LogAusleihe(ctx, tx, g.ID, leser.ID, "", staffID); err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	resp.Type = "ausleihe"
	resp.Geraet = g
	resp.DueDate = &dueTime
	resp.LoanID = &newLoanID
	return resp, nil
}

// ermittleVorbesitzer stellt fest, ob eine Fremdrückgabe vorliegt (Rückgeber ≠ Ausleiher)
// und lädt in diesem Fall die Stammdaten des ursprünglichen Ausleihers in resp.
func (s *defaultDeviceService) ermittleVorbesitzer(ctx context.Context, activeLoan *repository.Loan, leser *repository.Student, resp *DeviceResult) (bool, error) {
	if activeLoan.SchuelerID == nil {
		return false, nil
	}
	if leser != nil && *activeLoan.SchuelerID == leser.ID {
		return false, nil
	}
	// `leser` und nicht `schueler`: Sonst bliebe der Vorbesitzer namenlos, wenn ein
	// Kollege das Gerät hatte.
	var vorbesitzer repository.Student
	err := s.pool.QueryRow(ctx,
		"SELECT vorname, nachname, coalesce(klasse, '') FROM leser WHERE id = $1 AND deleted_at IS NULL",
		*activeLoan.SchuelerID).Scan(&vorbesitzer.Vorname, &vorbesitzer.Nachname, &vorbesitzer.Klasse)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return false, err
	}
	resp.Vorbesitzer = &vorbesitzer
	return true, nil
}

// gibGeraetZurueck behandelt Fall B: das ausgeliehene Gerät wird zurückgegeben.
func (s *defaultDeviceService) gibGeraetZurueck(ctx context.Context, tx pgx.Tx, g *repository.Geraet, activeLoan *repository.Loan, leser *repository.Student, staffID string) (*DeviceResult, error) {
	resp := &DeviceResult{}

	isFremd, err := s.ermittleVorbesitzer(ctx, activeLoan, leser, resp)
	if err != nil {
		return nil, err
	}

	// Rückgabe in der Datenbank eintragen (rueckgabe_am und bearbeiter_id setzen).
	if _, err := tx.Exec(ctx, `
		UPDATE ausleihen
		SET rueckgabe_am = CURRENT_TIMESTAMP, rueckgabe_bearbeiter_id = $1, ist_fremdrueckgabe = $2
		WHERE id = $3
	`, staffID, isFremd, activeLoan.ID); err != nil {
		return nil, err
	}

	// Revisionssicheres Audit-Log für die Rückgabe schreiben.
	if activeLoan.SchuelerID != nil {
		if err := s.auditRepo.LogRueckgabe(ctx, tx, g.ID, *activeLoan.SchuelerID, "", staffID); err != nil {
			return nil, err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	resp.Type = "rueckgabe"
	resp.Geraet = g
	resp.LoanID = &activeLoan.ID
	resp.Fremdrueckgabe = isFremd

	return resp, nil
}

// HandleDeviceAction implementiert die Geschäftslogik für die Ausleihe und Rückgabe von Geräten.
func (s *defaultDeviceService) HandleDeviceAction(
	ctx context.Context,
	query string,
	activeLeserID *string,
	confirmedChecklist bool,
	staffID string,
) (*DeviceResult, error) {
	g, err := s.ladeGeraet(ctx, query)
	if err != nil {
		return nil, err
	}

	// Transaktion starten, um den Ausleihprozess atomar und thread-sicher zu gestalten.
	tx, err := s.loanRepo.BeginTx(ctx)
	if err != nil {
		return nil, err
	}
	defer db.SafeRollback(ctx, tx)

	activeLoan, hasActiveLoan, err := ladeAktiveAusleihe(ctx, tx, g.ID)
	if err != nil {
		return nil, err
	}

	// Die Sperren (Gerät defekt oder ausgesondert, Schüler gesperrt) gelten der AUSLEIHE.
	// Bis zum 13.09.2026 standen sie vor dieser Weiche und hielten auch die Rückgabe auf:
	// 403, die Ausleihe blieb offen (geraet_rueckgabe_sperre_pg_test.go).
	var leser *repository.Student
	var einst *SystemEinstellungen
	if hasActiveLoan {
		leser, err = s.ladeRueckgeber(ctx, activeLeserID)
	} else {
		if err = pruefeGeraetAusleihbar(g); err != nil {
			return nil, err
		}
		leser, einst, err = s.ladeAkteur(ctx, activeLeserID)
	}
	if err != nil {
		return nil, err
	}

	// Checklisten-Regel: Hat das Gerät Zubehör und der Benutzer hat es noch nicht
	// bestätigt, unterbrechen wir und fordern die Bestätigung an.
	if g.Zubehoer != "" && !confirmedChecklist {
		return &DeviceResult{Type: "geraet_check", Geraet: &g}, nil
	}

	if !hasActiveLoan {
		return s.leiheGeraetAus(ctx, tx, &g, leser, einst, staffID)
	}
	return s.gibGeraetZurueck(ctx, tx, &g, &activeLoan, leser, staffID)
}

// pruefeGeraetAutomatikSperren wendet die AUTOMATIK-Sperren des Buch-Pfads auch auf die
// Geräte-Ausleihe an: unbezahlte Schäden und die Überfällig-Automatik. Die expliziten
// Sperr-Flags prüft bereits ladeAkteur; Geräte kennen bewusst kein override_block.
// Nutzt dieselben Zähl-/Settings-Quellen wie der Buch-Pfad (loan_checkout_validation.go)
// und gibt die geladenen Einstellungen zurück (für die Frist in leiheGeraetAus).
func pruefeGeraetAutomatikSperren(ctx context.Context, pool db.PgxPoolIface, leserID string) (*SystemEinstellungen, error) {
	offeneSchaeden, err := zaehleOffeneSchaeden(ctx, pool, leserID)
	if err != nil {
		return nil, err
	}
	if offeneSchaeden > 0 {
		return nil, fmt.Errorf("%w: %d unbezahlte(r) Schadensfall/-fälle offen", ErrBlocked, offeneSchaeden)
	}

	settings, err := ladeSystemEinstellungen(ctx, pool)
	if err != nil {
		return nil, err
	}
	overdue, err := zaehleUeberfaelligeMedien(ctx, pool, leserID, settings.MaxOverdueDays)
	if err != nil {
		return nil, err
	}
	if overdue >= settings.MaxOverdueItems {
		return nil, fmt.Errorf("%w: %d überfällige Medien vorhanden (Sperr-Automatik)", ErrBlocked, overdue)
	}
	return settings, nil
}
