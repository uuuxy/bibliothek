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

// fristAnfrage ist die Frist, die von Hand gesetzt wird.
type fristAnfrage struct {
	FaelligAm string `json:"faellig_am" validate:"required"` // Zeitstempel nach RFC 3339 oder nur der Tag nach ISO 8601
}

// AusleiheVerlaengern verlängert eine Ausleihe um die Frist für Bücher aus den Einstellungen
// (ohne Einstellung 28 Tage).
func (t *Tueren) AusleiheVerlaengern(einstellungsAbfragen repository.EinstellungsAbfragen) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		t.verlaengereAusleihe(w, r, einstellungsAbfragen)
	}
}

// verlaengereAusleihe verlängert eine einzelne Ausleihe. Als eigene Methode ausgelagert,
// damit die frühen Rücksprünge nicht zusätzlich als Verschachtelung in der Closure zählen
// (Sonar S3776).
func (t *Tueren) verlaengereAusleihe(w http.ResponseWriter, r *http.Request, einstellungsAbfragen repository.EinstellungsAbfragen) {
	ausleiheID := r.PathValue("ausleihe_id")
	if ausleiheID == "" {
		apierrors.AntworteFehler(w, http.StatusBadRequest, errors.New("fehlende ausleihe_id"))
		return
	}

	ctx := r.Context()

	// Sanktions-Konsistenz: Ist das Buch an einen gesperrten Schüler verliehen, darf die
	// Frist nicht verlängert werden — die Sperre soll zur Rückgabe zwingen, nicht durch
	// eine Verlängerung ausgehebelt werden. Welche Sperre zählt, sagt die Regel der Theke
	// (pruefeAusleihSperre): Beim Schulbuch nur die von Hand, beim Kollegen keine.
	gesperrt, grund, geprueft := t.sperreDerAusleihe(ctx, w, ausleiheID, "Einzel-Verlängerung")
	if !geprueft {
		return
	}
	if gesperrt {
		meldung := "Verlängerung nicht möglich: Ausleihe gesperrt"
		// Der Sperr-Freitext ist Verwaltungsinformation (PII-Matrix Stufe 2) —
		// dieselbe view_students-Grenze wie bei /api/action (ohneSperrgrund).
		if grund != "" && t.HatRecht(r, "view_students") {
			meldung += " (" + grund + ")"
		}
		apierrors.AntworteFehler(w, http.StatusForbidden, errors.New(meldung))
		return
	}

	tage, sommerferien := verlaengerungsRegel(ctx, einstellungsAbfragen)

	// Gerechnet wird ab dem späteren von altem Fristende und heute: Wer vor Fristablauf
	// verlängert, verliert die verbleibenden Tage nicht, und eine lange überfällige Ausleihe
	// bekommt keine Frist in der Vergangenheit. Die neue Frist ist eine Tagesfrist wie die der
	// Ausleihe (lmfplan.Ferientabelle.Tagesfrist), und die Mahnfolge beginnt neu — sonst
	// überspränge dasselbe Buch beim nächsten Überziehen die erste Mahnstufe. Die Zeile bleibt
	// gesperrt, bis die neue Frist steht, damit zwei Verlängerungen zugleich nacheinander rechnen.
	tx, err := t.DB.Pool.Begin(ctx)
	if err != nil {
		log.Printf("Fehler beim Starten der Transaktion (Einzel-Verlaengerung): %v", err)
		apierrors.AntworteFehler(w, http.StatusInternalServerError, errors.New("interner Serverfehler"))
		return
	}
	defer db.Verwerfe(ctx, tx)

	alteFrist, err := repository.SperreOffeneAusleiheMitFrist(ctx, tx, ausleiheID)
	if errors.Is(err, pgx.ErrNoRows) {
		apierrors.AntworteFehler(w, http.StatusNotFound, errors.New("ausleihe nicht gefunden oder bereits zurückgegeben"))
		return
	}
	if err != nil {
		log.Printf("Fehler bei Einzel-Verlaengerung: %v", err)
		apierrors.AntworteFehler(w, http.StatusInternalServerError, errors.New("interner Serverfehler"))
		return
	}
	basis := t.Jetzt()
	if alteFrist.After(basis) {
		basis = alteFrist
	}
	neueFrist := lmfplan.FerientabelleAus(sommerferien).Tagesfrist(basis, tage)

	_, gesetzteFrist, err := repository.VerlaengereAusleihe(ctx, tx, ausleiheID, neueFrist)
	if err != nil {
		log.Printf("Fehler bei Einzel-Verlaengerung: %v", err)
		apierrors.AntworteFehler(w, http.StatusInternalServerError, errors.New("interner Serverfehler"))
		return
	}
	if err := tx.Commit(ctx); err != nil {
		log.Printf("Fehler beim Abschluss der Einzel-Verlaengerung: %v", err)
		apierrors.AntworteFehler(w, http.StatusInternalServerError, errors.New("interner Serverfehler"))
		return
	}

	tuer.AntworteJSON(w, http.StatusOK, map[string]interface{}{
		"success":               true,
		"neues_rueckgabe_datum": gesetzteFrist,
	})
}

// AusleihFristSetzen setzt die Frist einer laufenden Ausleihe von Hand
// (PATCH /api/admin/ausleihen/{id}/faelligkeit).
func (t *Tueren) AusleihFristSetzen(protokollAbfragen repository.ProtokollAbfragen) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		t.setzeAusleihFrist(w, r, protokollAbfragen)
	}
}

// setzeAusleihFrist setzt die Frist einer laufenden Ausleihe von Hand.
func (t *Tueren) setzeAusleihFrist(w http.ResponseWriter, r *http.Request, protokollAbfragen repository.ProtokollAbfragen) {
	claims, ok := auth.ClaimsAusKontext(r.Context())
	if !ok {
		apierrors.AntworteFehlerMitMeldung(w, http.StatusUnauthorized, auth.MeldungNichtAngemeldet, nil)
		return
	}

	ausleiheID := r.PathValue("id")
	if ausleiheID == "" {
		apierrors.AntworteFehler(w, http.StatusBadRequest, errors.New("fehlende Ausleihe-ID"))
		return
	}

	var req fristAnfrage
	if !tuer.LeseAnfrage(w, r, &req) {
		return
	}

	neuesDatum, err := leseFaelligAm(req.FaelligAm)
	if err != nil {
		apierrors.AntworteFehler(w, http.StatusBadRequest, err)
		return
	}

	ctx := r.Context()

	// Wie bei den beiden Verlängerungen: Eine Frist in der Zukunft machte die Ausleihe eines
	// gesperrten Lesers wieder „nicht überfällig", setzte die Mahnfolge zurück und hebelte so
	// die Sperre aus, die aus überfälligen Ausleihen berechnet wird. Ein vorgezogenes Datum
	// (Rückruf) bleibt auch bei Sperre erlaubt: Es hebt die Sanktion nicht auf.
	if neuesDatum.After(time.Now()) {
		gesperrt, _, geprueft := t.sperreDerAusleihe(ctx, w, ausleiheID, "Frist-Überschreibung")
		if !geprueft {
			return
		}
		if gesperrt {
			apierrors.AntworteFehler(w, http.StatusForbidden,
				errors.New("gesperrte Ausleihe: die Frist kann nicht in die Zukunft verschoben werden — die Sperre soll zur Rückgabe zwingen"))
			return
		}
	}

	id, gesetzteFrist, err := repository.SetzeAusleihFrist(ctx, t.DB.Pool, ausleiheID, neuesDatum)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			apierrors.AntworteFehler(w, http.StatusNotFound, errors.New("ausleihe nicht gefunden oder bereits zurückgegeben"))
			return
		}
		log.Printf("Fehler bei manueller Frist-Überschreibung: %v", err)
		apierrors.AntworteFehler(w, http.StatusInternalServerError, errors.New("interner Serverfehler"))
		return
	}

	// Eine von Hand gesetzte Frist ist ein Eingriff in eine Sanktion (Mahnfristen, Sperre durch
	// das Programm); sie gehört ins Protokoll wie das Übergehen beim Ausleihen (OVERRIDE_BLOCK).
	// Der Eingriff gilt auch, wenn das Protokoll klemmt; der Fehlversuch steht dann im Log.
	if protokollFehler := protokollAbfragen.ProtokolliereAdminAktion(ctx, claims.KontoID, "FRIST_OVERRIDE", "", map[string]any{
		"ausleihe_id": id,
		"neue_frist":  gesetzteFrist.Format(time.RFC3339),
	}); protokollFehler != nil {
		log.Printf("Audit für Frist-Überschreibung fehlgeschlagen: %v", protokollFehler)
	}

	tuer.AntworteJSON(w, http.StatusOK, map[string]interface{}{
		"success":    true,
		"faellig_am": gesetzteFrist,
	})
}

// leseFaelligAm liest die Frist als Zeitstempel (ISO 8601) oder als Datum. Ein Datum gilt bis
// zum Tagesende in der Schulzeitzone, nach derselben Definition wie die berechnete Frist
// (schulzeit.TagesEnde): Roh in UTC wäre die gesetzte Frist ein bis zwei Stunden später fällig
// als eine berechnete zum selben Datum.
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
func verlaengerungsRegel(ctx context.Context, einstellungsAbfragen repository.EinstellungsAbfragen) (tage int, sommerferien string) {
	einstellungen, err := einstellungsAbfragen.Lade(ctx)
	if err != nil {
		return 28, ""
	}
	tage = 28
	if einstellungen.FristBuchTage > 0 {
		tage = einstellungen.FristBuchTage
	}
	return tage, einstellungen.Sommerferien
}

// sperreDerAusleihe fragt die Regel der Theke (pruefeAusleihSperre) für eine Verlängerung oder
// eine Frist von Hand. Fehlt die Ausleihe oder scheitert die Prüfung, ist die Anfrage hier
// beantwortet und geprueft false.
func (t *Tueren) sperreDerAusleihe(ctx context.Context, w http.ResponseWriter, ausleiheID, vorgang string) (gesperrt bool, grund string, geprueft bool) {
	gesperrt, grund, err := t.pruefeAusleihSperre(ctx, ausleiheID)
	if errors.Is(err, pgx.ErrNoRows) {
		apierrors.AntworteFehler(w, http.StatusNotFound, errors.New("ausleihe nicht gefunden oder bereits zurückgegeben"))
		return false, "", false
	}
	if err != nil {
		log.Printf("Fehler bei Sperr-Prüfung (%s): %v", vorgang, err)
		apierrors.AntworteFehler(w, http.StatusInternalServerError, errors.New("interner Serverfehler"))
		return false, "", false
	}
	return gesperrt, grund, true
}

// pruefeAusleihSperre prüft, ob eine Sperre am Leser die Verlängerung dieser Ausleihe anhält,
// und liefert den Grund. Es entscheidet die Regel der Theke (service.PruefeSperreAmLeser):
// beim Schulbuch nur die Sperre von Hand, beim Kollegen keine; ein anonymisierter Datensatz
// und der Papierkorb bekommen nichts. Eine eigene Abfrage, die jede Sperre zählte, hielte
// auch die Ehemaligen beim Schulbuch an, das die Theke demselben Kind ausgibt.
func (t *Tueren) pruefeAusleihSperre(ctx context.Context, ausleiheID string) (bool, string, error) {
	leserID, lernmittel, err := repository.LadeLeserUndLernmittelDerAusleihe(ctx, t.DB.Pool, ausleiheID)
	if err != nil || leserID == nil {
		return false, "", err
	}
	leser, err := repository.NewLeserAbfragen(t.DB.Pool).Finde(ctx, *leserID)
	if err != nil {
		return false, "", err
	}
	if leser == nil {
		// Im Papierkorb: Die Theke findet ihn nicht und gibt ihm nichts aus.
		return true, "", nil
	}
	sperre := service.PruefeSperreAmLeser(leser, lernmittel)
	if sperre == nil {
		return false, "", nil
	}
	var mitGrund *service.SperrgrundFehler
	if errors.As(sperre, &mitGrund) {
		return true, mitGrund.Grund, nil
	}
	return true, "", nil
}
