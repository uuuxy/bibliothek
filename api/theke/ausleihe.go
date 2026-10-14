package theke

import (
	"bibliothek/apierrors"
	"bibliothek/auth"
	"bibliothek/db"
	"errors"

	"context"
	"log"
	"net/http"
	"time"

	"bibliothek/api/tuer"
	"bibliothek/internal/service"
	"bibliothek/pkg/lmfplan"
	"bibliothek/pkg/schulzeit"
	"bibliothek/repository"

	"github.com/jackc/pgx/v5"
)

// OverrideDueDateRequest holds the JSON payload for manually overriding a due date.
type OverrideDueDateRequest struct {
	FaelligAm string `json:"faellig_am" validate:"required"` // ISO 8601 or YYYY-MM-DD
}

// ExtendLoanHandler extends the due date of a single loan by the standard book interval (e.g. 28 days).
func (s *Tueren) ExtendLoanHandler(settingsRepo repository.EinstellungsAbfragen) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		s.handleExtendLoan(w, r, settingsRepo)
	}
}

// handleExtendLoan verlängert eine einzelne Ausleihe. Als Top-Level-Methode ausgelagert,
// damit die Frühabbrüche nicht zusätzlich als Closure-Verschachtelung zählen (S3776).
func (s *Tueren) handleExtendLoan(w http.ResponseWriter, r *http.Request, settingsRepo repository.EinstellungsAbfragen) {
	ausleiheID := r.PathValue("ausleihe_id")
	if ausleiheID == "" {
		apierrors.SendHTTPError(w, http.StatusBadRequest, errors.New("fehlende ausleihe_id"))
		return
	}

	ctx := r.Context()

	// Sanktions-Konsistenz: Ist das Buch an einen gesperrten Schüler verliehen, darf die
	// Frist nicht verlängert werden — die Sperre soll zur Rückgabe zwingen, nicht durch
	// eine Verlängerung ausgehebelt werden. Welche Sperre zählt, sagt die Regel der Theke
	// (checkAusleiheGesperrt): Beim Schulbuch nur die von Hand, beim Kollegen keine.
	gesperrt, blockReason, geprueft := s.sperreDerAusleihe(ctx, w, ausleiheID, "Einzel-Verlängerung")
	if !geprueft {
		return
	}
	if gesperrt {
		msg := "Verlängerung nicht möglich: Ausleihe gesperrt"
		// Der Sperr-Freitext ist Verwaltungsinformation (PII-Matrix Stufe 2) —
		// dieselbe view_students-Grenze wie bei /api/action (ohneSperrgrund).
		if blockReason != "" && s.BesitztRecht(r, "view_students") {
			msg += " (" + blockReason + ")"
		}
		apierrors.SendHTTPError(w, http.StatusForbidden, errors.New(msg))
		return
	}

	extensionDays, sommerferien := verlaengerungsRegel(ctx, settingsRepo)

	// Gerechnet wird ab dem späteren von altem Fristende und heute: Wer vor Fristablauf
	// verlängert, verliert die verbleibenden Tage nicht, und eine lange überfällige Ausleihe
	// bekommt keine Frist in der Vergangenheit. Die neue Frist ist eine Tagesfrist wie die der
	// Ausleihe (lmfplan.Ferientabelle.Tagesfrist), und die Mahnfolge beginnt neu — sonst
	// überspränge dasselbe Buch beim nächsten Überziehen die erste Mahnstufe. Die Zeile bleibt
	// gesperrt, bis die neue Frist steht, damit zwei Verlängerungen zugleich nacheinander rechnen.
	tx, err := s.DB.Pool.Begin(ctx)
	if err != nil {
		log.Printf("Fehler beim Starten der Transaktion (Einzel-Verlaengerung): %v", err)
		apierrors.SendHTTPError(w, http.StatusInternalServerError, errors.New("interner Serverfehler"))
		return
	}
	defer db.SafeRollback(ctx, tx)

	alteFrist, err := repository.SperreOffeneAusleiheMitFrist(ctx, tx, ausleiheID)
	if errors.Is(err, pgx.ErrNoRows) {
		apierrors.SendHTTPError(w, http.StatusNotFound, errors.New("ausleihe nicht gefunden oder bereits zurückgegeben"))
		return
	}
	if err != nil {
		log.Printf("Fehler bei Einzel-Verlaengerung: %v", err)
		apierrors.SendHTTPError(w, http.StatusInternalServerError, errors.New("interner Serverfehler"))
		return
	}
	basis := s.Jetzt()
	if alteFrist.After(basis) {
		basis = alteFrist
	}
	neueFrist := lmfplan.FerientabelleAus(sommerferien).Tagesfrist(basis, extensionDays)

	_, newFrist, err := repository.VerlaengereAusleihe(ctx, tx, ausleiheID, neueFrist)
	if err != nil {
		log.Printf("Fehler bei Einzel-Verlaengerung: %v", err)
		apierrors.SendHTTPError(w, http.StatusInternalServerError, errors.New("interner Serverfehler"))
		return
	}
	if err := tx.Commit(ctx); err != nil {
		log.Printf("Fehler beim Abschluss der Einzel-Verlaengerung: %v", err)
		apierrors.SendHTTPError(w, http.StatusInternalServerError, errors.New("interner Serverfehler"))
		return
	}

	tuer.RespondJSON(w, http.StatusOK, map[string]interface{}{
		"success":               true,
		"neues_rueckgabe_datum": newFrist,
	})
}

// OverrideDueDateHandler manually overrides the due date of an active loan.
func (s *Tueren) OverrideDueDateHandler(auditRepo repository.ProtokollAbfragen) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		s.handleOverrideDueDate(w, r, auditRepo)
	}
}

// handleOverrideDueDate setzt die Frist einer laufenden Ausleihe von Hand.
func (s *Tueren) handleOverrideDueDate(w http.ResponseWriter, r *http.Request, auditRepo repository.ProtokollAbfragen) {
	claims, ok := auth.GetClaims(r.Context())
	if !ok {
		apierrors.SendHTTPError(w, http.StatusUnauthorized, errors.New("Sitzungs-Information fehlt oder ist abgelaufen"))
		return
	}

	ausleiheID := r.PathValue("id")
	if ausleiheID == "" {
		apierrors.SendHTTPError(w, http.StatusBadRequest, errors.New("fehlende Ausleihe-ID"))
		return
	}

	var req OverrideDueDateRequest
	if !tuer.DecodeAndValidate(w, r, &req) {
		return
	}

	newDate, err := leseFaelligAm(req.FaelligAm)
	if err != nil {
		apierrors.SendHTTPError(w, http.StatusBadRequest, err)
		return
	}

	ctx := r.Context()

	// Wie bei den beiden Verlängerungen: Eine Frist in der Zukunft machte die Ausleihe eines
	// gesperrten Lesers wieder „nicht überfällig", setzte die Mahnfolge zurück und hebelte so
	// die Sperre aus, die aus überfälligen Ausleihen berechnet wird. Ein vorgezogenes Datum
	// (Rückruf) bleibt auch bei Sperre erlaubt: Es hebt die Sanktion nicht auf.
	if newDate.After(time.Now()) {
		gesperrt, _, geprueft := s.sperreDerAusleihe(ctx, w, ausleiheID, "Frist-Überschreibung")
		if !geprueft {
			return
		}
		if gesperrt {
			apierrors.SendHTTPError(w, http.StatusForbidden,
				errors.New("gesperrte Ausleihe: die Frist kann nicht in die Zukunft verschoben werden — die Sperre soll zur Rückgabe zwingen"))
			return
		}
	}

	id, newFrist, err := repository.SetzeAusleihFrist(ctx, s.DB.Pool, ausleiheID, newDate)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			apierrors.SendHTTPError(w, http.StatusNotFound, errors.New("ausleihe nicht gefunden oder bereits zurückgegeben"))
			return
		}
		log.Printf("Fehler bei manueller Frist-Überschreibung: %v", err)
		apierrors.SendHTTPError(w, http.StatusInternalServerError, errors.New("interner Serverfehler"))
		return
	}

	// Ein manuell überschriebenes Fälligkeitsdatum ist ein Eingriff in eine Sanktion
	// (Mahnfristen, Auto-Sperre) — er gehört revisionssicher protokolliert, genau wie
	// der Checkout-Override (OVERRIDE_BLOCK). Best effort: Der Eingriff gilt, auch wenn
	// das Log klemmt, aber der Fehlversuch steht wenigstens im Server-Log.
	if logErr := auditRepo.ProtokolliereAdminAktion(ctx, claims.UserID, "FRIST_OVERRIDE", "", map[string]any{
		"ausleihe_id": id,
		"neue_frist":  newFrist.Format(time.RFC3339),
	}); logErr != nil {
		log.Printf("Audit für Frist-Überschreibung fehlgeschlagen: %v", logErr)
	}

	tuer.RespondJSON(w, http.StatusOK, map[string]interface{}{
		"success":    true,
		"faellig_am": newFrist,
	})
}

// leseFaelligAm liest die Frist als Zeitstempel (ISO 8601) oder als Datum. Ein Datum gilt bis
// zum Tagesende in der Schulzeitzone, nach derselben Definition wie service.CalculateDueDate:
// Roh in UTC wäre die überschriebene Frist ein bis zwei Stunden später fällig als eine
// berechnete zum selben Datum.
func leseFaelligAm(text string) (time.Time, error) {
	if zeitstempel, err := time.Parse(time.RFC3339, text); err == nil {
		return zeitstempel, nil
	}
	datum, err := time.Parse("2006-01-02", text)
	if err != nil {
		return time.Time{}, errors.New("ungültiges Datumsformat (erwartet ISO 8601 oder YYYY-MM-DD)")
	}
	return schulzeit.TagesEnde(datum), nil
}

// verlaengerungsRegel liefert die Tage einer Verlängerung und die Sommerferien aus den
// Einstellungen. Sind sie nicht lesbar, gelten 28 Tage und die Ferientabelle des Programms.
func verlaengerungsRegel(ctx context.Context, settingsRepo repository.EinstellungsAbfragen) (tage int, sommerferien string) {
	settings, err := settingsRepo.Lade(ctx)
	if err != nil {
		return 28, ""
	}
	tage = 28
	if settings.FristBuchTage > 0 {
		tage = settings.FristBuchTage
	}
	return tage, settings.Sommerferien
}

// sperreDerAusleihe fragt die Regel der Theke (checkAusleiheGesperrt) für eine Verlängerung oder
// eine Frist von Hand. Fehlt die Ausleihe oder scheitert die Prüfung, ist die Anfrage hier
// beantwortet und geprueft false.
func (s *Tueren) sperreDerAusleihe(ctx context.Context, w http.ResponseWriter, ausleiheID, vorgang string) (gesperrt bool, grund string, geprueft bool) {
	gesperrt, grund, err := s.checkAusleiheGesperrt(ctx, ausleiheID)
	if errors.Is(err, pgx.ErrNoRows) {
		apierrors.SendHTTPError(w, http.StatusNotFound, errors.New("ausleihe nicht gefunden oder bereits zurückgegeben"))
		return false, "", false
	}
	if err != nil {
		log.Printf("Fehler bei Sperr-Prüfung (%s): %v", vorgang, err)
		apierrors.SendHTTPError(w, http.StatusInternalServerError, errors.New("interner Serverfehler"))
		return false, "", false
	}
	return gesperrt, grund, true
}

// checkAusleiheGesperrt prüft, ob eine Sperre am Leser die Verlängerung dieser Ausleihe
// anhält, und liefert den Grund. Es entscheidet die Regel der Theke
// (service.SperreAmLeserHaeltAn): beim Schulbuch nur die Sperre von Hand, beim Kollegen keine;
// ein anonymisierter Datensatz und der Papierkorb bekommen nichts.
// Bis zum 29.09.2026 stand hier eine eigene Abfrage, die jede Sperre zählte — auch die der
// Ehemaligen beim Schulbuch, das die Theke demselben Kind ausgibt.
func (s *Tueren) checkAusleiheGesperrt(ctx context.Context, ausleiheID string) (bool, string, error) {
	leserID, lernmittel, err := repository.LadeLeserUndLernmittelDerAusleihe(ctx, s.DB.Pool, ausleiheID)
	if err != nil || leserID == nil {
		return false, "", err
	}
	leser, err := repository.NewLeserAbfragen(s.DB.Pool).Finde(ctx, *leserID)
	if err != nil {
		return false, "", err
	}
	if leser == nil {
		// Im Papierkorb: Die Theke findet ihn nicht und gibt ihm nichts aus.
		return true, "", nil
	}
	sperre := service.SperreAmLeserHaeltAn(leser, lernmittel)
	if sperre == nil {
		return false, "", nil
	}
	var mitGrund *service.SperrGrundFehler
	if errors.As(sperre, &mitGrund) {
		return true, mitGrund.Grund, nil
	}
	return true, "", nil
}
