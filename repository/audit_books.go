package repository

import (
	"bibliothek/db"
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

// ErrTitelHatAktiveAusleihen meldet, dass ein Titel nicht gelöscht werden kann, weil
// noch Exemplare verliehen sind. Das ist ein Bedienfehler (HTTP 400), keine Störung.
//
// Als Sentinel und nicht als Textvergleich: Der Handler prüfte die Meldung früher per
// err.Error()[:22] gegen "Löschen fehlgeschlagen:" — und traf nie. Der Text hier begann
// klein, das Literal groß, und 22 Bytes können ein 24-Byte-Literal ohnehin nicht
// treffen (die Umlaute zählen doppelt). Jeder blockierte Löschversuch wurde damit zu
// einem HTTP 500, und die Liste der noch verliehenen Barcodes — genau die Auskunft, die
// weiterhilft — verschwand hinter „Serverfehler".
//
//nolint:staticcheck // ST1005: bewusst großgeschrieben, nutzer-sichtbare Meldung
var ErrTitelHatAktiveAusleihen = errors.New("Löschen fehlgeschlagen: folgende Exemplare sind noch verliehen")

// ErrExemplarNochVerliehen / ErrExemplarNichtGefunden: dieselbe Sentinel-Bauart für das
// EINZELNE Exemplar. Der Handler entschied hier bis zum 31.08.2026 per Textvergleich —
// „Exemplar ist aktuell noch verliehen!" gegen „exemplar ist aktuell noch verliehen"
// (Großschreibung + Ausrufezeichen): Der Vergleich traf nie, ein verliehenes Buch endete
// als 500 mit Sanitizer-Text statt als 400 mit Auskunft.
var (
	ErrExemplarNochVerliehen = errors.New("exemplar ist aktuell noch verliehen")
	ErrExemplarNichtGefunden = errors.New("exemplar nicht gefunden oder bereits ausgebucht")
	// ErrTitelNichtGefunden: unbekannte Titel-ID beim Löschen (Phantom-Erfolg-Sweep 31.08.2026).
	ErrTitelNichtGefunden = errors.New("titel nicht gefunden")
)

// DeleteTitle entfernt einen Buchtitel vollständig aus dem Katalog und erstellt einen revisionssicheren Audit-Eintrag.
// Vor dem Löschen wird geprüft, ob noch Exemplare dieses Titels verliehen sind (was das Löschen blockiert).
// Historische Ausleihen und ALLE Schadensfälle des Titels werden bereinigt, um
// Fremdschlüssel-Fehler zu vermeiden — unbezahlte Forderungen stehen vorher im
// Protokoll (protokolliereOffeneForderungen).
func (r *pgAuditRepository) DeleteTitle(ctx context.Context, titleID string, bearbeiterID string) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer db.SafeRollback(ctx, tx)

	// Gehört der Titel zu einem Buch in mehreren Auflagen, muss das Werk danach noch stimmen
	// (docs/OFFEN.md 4.18). Als erstes: Die Sperre der Auflagen kommt vor jeder Zeilensperre.
	werke, err := WerkeDerTitel(ctx, tx, []string{titleID})
	if err != nil {
		return err
	}

	vorher, err := leseTitelVorDemLoeschen(ctx, tx, titleID)
	if err != nil {
		return err
	}
	if err = protokolliereWasMitDemTitelFaellt(ctx, tx, titleID); err != nil {
		return err
	}
	// Das lokal gespeicherte Cover fällt mit dem Titel, wie in der Massenaktion der
	// Bestandstabelle (inventur.DeleteBooks).
	cover, err := LokaleCoverNurDieserTitel(ctx, tx, []string{titleID})
	if err != nil {
		return err
	}
	// Verknüpfte Einträge (Schadensfälle, alte Rückgaben) löschen, um ON DELETE RESTRICT Fehler zu vermeiden
	if _, err = tx.Exec(ctx, "DELETE FROM schadensfaelle WHERE exemplar_id IN (SELECT id FROM buecher_exemplare WHERE titel_id = $1)", titleID); err != nil {
		return fmt.Errorf("failed to delete damage records for title: %w", err)
	}
	if _, err = tx.Exec(ctx, "DELETE FROM ausleihen WHERE exemplar_id IN (SELECT id FROM buecher_exemplare WHERE titel_id = $1) AND rueckgabe_am IS NOT NULL", titleID); err != nil {
		return fmt.Errorf("failed to delete past loans for title: %w", err)
	}

	// Alle zugehörigen Exemplare löschen
	if _, err = tx.Exec(ctx, "DELETE FROM buecher_exemplare WHERE titel_id = $1", titleID); err != nil {
		return fmt.Errorf("failed to delete associated copies: %w", err)
	}

	// Eigentlichen Titel-Datensatz löschen. 0 Zeilen heißt unbekannte Kennung: Der Snapshot
	// toleriert ErrNoRows und es gibt keine aktiven Ausleihen, ohne die Prüfung entstünde
	// ein Protokolleintrag über einen Titel, den es nie gab.
	tag, err := tx.Exec(ctx, "DELETE FROM buecher_titel WHERE id = $1", titleID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrTitelNichtGefunden
	}
	if err = RaeumeWerkeAuf(ctx, tx, werke); err != nil {
		return err
	}

	if err = r.protokolliereTitelLoeschung(ctx, tx, vorher, bearbeiterID); err != nil {
		return err
	}

	if err = tx.Commit(ctx); err != nil {
		return err
	}
	LoescheCoverDateien(cover)
	return nil
}

// titelVorDemLoeschen hält fest, was nach dem Löschen nicht mehr zu lesen ist: die Angaben
// des Titels und die Barcodes seiner Exemplare, für das Protokoll.
type titelVorDemLoeschen struct {
	id, titel, autor, isbn string
	exemplare              []titelExemplar
}

type titelExemplar struct{ id, barcode string }

// leseTitelVorDemLoeschen liest die Angaben fürs Protokoll und lehnt ab, solange ein Exemplar
// verliehen ist. Einen unbekannten Titel lässt sie durch; den meldet das DELETE.
func leseTitelVorDemLoeschen(ctx context.Context, tx pgx.Tx, titleID string) (titelVorDemLoeschen, error) {
	vorher := titelVorDemLoeschen{id: titleID}
	err := tx.QueryRow(ctx,
		`SELECT coalesce(titel,''), coalesce(autor,''), coalesce(isbn,'') FROM buecher_titel WHERE id = $1`,
		titleID,
	).Scan(&vorher.titel, &vorher.autor, &vorher.isbn)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return vorher, fmt.Errorf("failed to snapshot title for audit: %w", err)
	}
	if err := pruefeKeineAktivenAusleihen(ctx, tx, titleID); err != nil {
		return vorher, err
	}
	vorher.exemplare, err = leseTitelExemplare(ctx, tx, titleID)
	return vorher, err
}

// pruefeKeineAktivenAusleihen ist die Sicherheitsschranke vor dem Löschen: Ist ein Exemplar
// des Titels verliehen, nennt der Fehler die Barcodes.
func pruefeKeineAktivenAusleihen(ctx context.Context, tx pgx.Tx, titleID string) error {
	var activeLoans []string
	rows, err := tx.Query(ctx, `
		SELECT e.barcode_id 
		FROM ausleihen a 
		JOIN buecher_exemplare e ON a.exemplar_id = e.id 
		WHERE e.titel_id = $1 AND a.rueckgabe_am IS NULL
	`, titleID)
	if err != nil {
		return fmt.Errorf("failed to check active loans for title: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var barcode string
		if err := rows.Scan(&barcode); err == nil {
			activeLoans = append(activeLoans, barcode)
		}
	}
	// Bricht die Iteration durch einen Verbindungsfehler vorzeitig ab, heißt das nicht
	// „keine aktiven Ausleihen"; der Titel darf dann nicht gelöscht werden.
	if err := rows.Err(); err != nil {
		return fmt.Errorf("failed to read active loans for title: %w", err)
	}
	if len(activeLoans) > 0 {
		return fmt.Errorf("%w: %v", ErrTitelHatAktiveAusleihen, activeLoans)
	}
	return nil
}

// leseTitelExemplare liest Kennung und Barcode jedes Exemplars vor dem Löschen. Die
// Tresen-Auskunft findet gelöschte Exemplare nur über Protokollzeilen mit
// tabelle='buecher_exemplare' und details->>'barcode_id' (SucheTresenExemplare); ohne sie
// hieße ein später gescanntes Buch „nie gesehen" statt „gelöscht am …".
func leseTitelExemplare(ctx context.Context, tx pgx.Tx, titleID string) ([]titelExemplar, error) {
	var exemplare []titelExemplar
	exRows, err := tx.Query(ctx,
		`SELECT id::text, barcode_id FROM buecher_exemplare WHERE titel_id = $1`, titleID)
	if err != nil {
		return nil, fmt.Errorf("failed to snapshot copies for audit: %w", err)
	}
	for exRows.Next() {
		var s titelExemplar
		if err := exRows.Scan(&s.id, &s.barcode); err != nil {
			exRows.Close()
			return nil, fmt.Errorf("failed to scan copy snapshot: %w", err)
		}
		exemplare = append(exemplare, s)
	}
	exRows.Close()
	// Bricht die Iteration vorzeitig ab, fehlten Einträge still.
	if err := exRows.Err(); err != nil {
		return nil, fmt.Errorf("failed to read copy snapshots: %w", err)
	}
	return exemplare, nil
}

// protokolliereWasMitDemTitelFaellt hält vor dem Löschen fest, was mit dem Titel verschwindet:
// unbezahlte Forderungen (wer wie viel wofür schuldete) und, per ON DELETE CASCADE, wer auf
// den Titel wartet. Dieselbe Regel wie in der Massenaktion der Bestandstabelle
// (titel_loeschen_wartende.go).
func protokolliereWasMitDemTitelFaellt(ctx context.Context, tx pgx.Tx, titleID string) error {
	if err := protokolliereOffeneForderungen(ctx, tx, titleID); err != nil {
		return err
	}
	wartende, err := LeseWartendeBezuege(ctx, tx, []string{titleID})
	if err != nil {
		return err
	}
	return ProtokolliereWartendeBezuege(ctx, tx, wartende)
}

// protokolliereTitelLoeschung schreibt den Eintrag zum Titel und je Exemplar den
// Barcode-Eintrag in der Form der Geschwister-Pfade (DeleteCopy, Verlust-Löschen), in der
// Transaktion des Löschens: entweder Löschung und Spur oder keins von beiden.
func (r *pgAuditRepository) protokolliereTitelLoeschung(ctx context.Context, tx pgx.Tx, vorher titelVorDemLoeschen, bearbeiterID string) error {
	if err := r.insertAuditLog(ctx, tx, auditEntry{
		Tabelle: "buecher_titel", Aktion: "DELETE", DatensatzID: vorher.id,
		BearbeiterID: &bearbeiterID, Akteur: "USER",
		Details: map[string]any{"titel": vorher.titel, "autor": vorher.autor, "isbn": vorher.isbn},
	}); err != nil {
		return err
	}
	kontext := "Titel gelöscht — Exemplar mit entfernt"
	for _, ex := range vorher.exemplare {
		if err := r.insertAuditLog(ctx, tx, auditEntry{
			Tabelle: "buecher_exemplare", Aktion: "DELETE", DatensatzID: ex.id,
			BearbeiterID: &bearbeiterID, Akteur: "USER", Kontext: &kontext,
			Details: map[string]any{
				"barcode_id": ex.barcode, "titel": vorher.titel, "titel_id": vorher.id,
				"action": AuditAktionTitelGeloescht,
			},
		}); err != nil {
			return err
		}
	}
	return nil
}

// DeleteCopy bucht ein physisches Exemplar aus dem System aus (Soft-Delete) und protokolliert dies im Audit-Log.
// Da historische Ausleihdaten und Schadensfälle für statistische Zwecke erhalten bleiben müssen, wird das Exemplar
// nicht physisch aus der Tabelle gelöscht, sondern als ausgesondert markiert.
// Die Protokoll-Marker der drei Türen, die ein Exemplar KÖRPERLICH entfernen.
//
// Warum als Konstante und nicht als getippter Text an drei Stellen: Seit dem 17.09.2026
// liest das Abgangsbuch sie (abgangsbuch.go). Ein Exemplar, das gelöscht statt
// ausgesondert wird, fällt aus jeder Abfrage über `buecher_exemplare` — der Nachweis kann
// es nur über diese Spur zählen. Wer den Text an einer Tür umbenennt, ohne die Abfrage zu
// kennen, senkt die Zahl still auf 0, und ein Nachweis, der schweigt, sieht aus wie einer,
// der vollständig ist. So ist die Umbenennung eine Änderung an EINEM Wort.
const (
	// AuditAktionTitelGeloescht: Das Exemplar ging mit seinem Titel (beide Lösch-Türen).
	AuditAktionTitelGeloescht = "titel_geloescht"
	// AuditAktionVerlustEndgueltigGeloescht: Ein als Verlust gebuchtes Exemplar wurde in
	// der Inventur endgültig entfernt (inventur_verlust_aktionen.go).
	AuditAktionVerlustEndgueltigGeloescht = "verlust_endgueltig_geloescht"
)

func (r *pgAuditRepository) DeleteCopy(ctx context.Context, copyID string, bearbeiterID string) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer db.SafeRollback(ctx, tx)

	// Snapshot erstellen: Exemplardaten vor dem Aussondern für das Audit-Log sichern.
	// ErrNoRows ist hier KEIN tolerierbarer Fall (bis 31.08.2026 war er es): Eine
	// unbekannte ID endete sonst als Phantom-Erfolg — 200 samt Audit-Eintrag über eine
	// Löschung, die nie stattfand.
	var barcode, zustandNotiz, titel string
	var titelID string
	err = tx.QueryRow(ctx,
		`SELECT e.barcode_id, coalesce(e.zustand_notiz,''), e.titel_id, t.titel
		 FROM buecher_exemplare e
		 JOIN buecher_titel t ON e.titel_id = t.id
		 WHERE e.id = $1`,
		copyID,
	).Scan(&barcode, &zustandNotiz, &titelID, &titel)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrExemplarNichtGefunden
	}
	if err != nil {
		return fmt.Errorf("failed to snapshot copy for audit: %w", err)
	}

	// Sicherheitsschranke: Ist das Exemplar aktuell noch verliehen?
	var activeLoanCount int
	err = tx.QueryRow(ctx, "SELECT count(*) FROM ausleihen WHERE exemplar_id = $1 AND rueckgabe_am IS NULL", copyID).Scan(&activeLoanCount)
	if err != nil {
		return fmt.Errorf("failed to check active loans for copy: %w", err)
	}
	if activeLoanCount > 0 {
		return ErrExemplarNochVerliehen
	}

	// Soft-Delete durchführen: Exemplar sperren und Zustand auf "Systematisch gelöscht"
	// setzen. Der Guard ist_ausgesondert = false macht den Doppelklick zum 404 statt zum
	// zweiten „Erfolg" — und verhindert, dass ein Wiederholungsklick die Zustandsnotiz
	// eines längst ausgebuchten Exemplars erneut überschreibt.
	tag, err := tx.Exec(ctx, "UPDATE buecher_exemplare SET ist_ausgesondert = true, ist_ausleihbar = false, aussonderung_grund = 'AUSSORTIERT', zustand_notiz = 'Systematisch gelöscht', bestellstatus = NULL, letzte_bewegung_am = "+sqlStempelJetzt+" WHERE id = $1 AND ist_ausgesondert = false", copyID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrExemplarNichtGefunden
	}

	kontext := "Buch ausgebuchen (Soft-Delete)"
	if err = r.insertAuditLog(ctx, tx, auditEntry{
		Tabelle: "buecher_exemplare", Aktion: "UPDATE", DatensatzID: copyID,
		BearbeiterID: &bearbeiterID, Akteur: "USER", Kontext: &kontext,
		Details: map[string]any{"barcode_id": barcode, "zustand_notiz": zustandNotiz, "titel_id": titelID, "titel": titel, "action": "soft_delete"},
	}); err != nil {
		return err
	}

	return tx.Commit(ctx)
}

// LogAusleihe schreibt einen neuen Ausleiheintrag (CHECKOUT) in das Audit-Log.
func (r *pgAuditRepository) LogAusleihe(ctx context.Context, tx pgx.Tx, exemplarID string, schuelerID string, benutzerID string, bearbeiterID string) error {
	return r.logLoanEvent(ctx, tx, "CHECKOUT", exemplarID, schuelerID, benutzerID, bearbeiterID)
}

// LogRueckgabe schreibt einen neuen Rückgabeeintrag (RETURN) in das Audit-Log.
func (r *pgAuditRepository) LogRueckgabe(ctx context.Context, tx pgx.Tx, exemplarID string, schuelerID string, benutzerID string, bearbeiterID string) error {
	return r.logLoanEvent(ctx, tx, "RETURN", exemplarID, schuelerID, benutzerID, bearbeiterID)
}

// logLoanEvent schreibt den Ausleih-/Rückgabe-Eintrag in die ÜBERGEBENE Transaktion —
// keine eigene: Eintrag und Buchung stehen oder fallen zusammen.
func (r *pgAuditRepository) logLoanEvent(ctx context.Context, tx pgx.Tx, aktion, exemplarID, schuelerID, benutzerID, bearbeiterID string) error {
	var bearbeiterPtr *string
	if bearbeiterID != "" {
		bearbeiterPtr = &bearbeiterID
	}

	details := map[string]any{
		"exemplar_id": exemplarID,
		"zeitpunkt":   time.Now().UTC().Format(time.RFC3339),
	}
	if schuelerID != "" {
		details["schueler_id"] = schuelerID
	}
	if benutzerID != "" {
		details["benutzer_id"] = benutzerID
	}

	return r.insertAuditLog(ctx, tx, auditEntry{
		Tabelle: "ausleihen", Aktion: aktion, DatensatzID: exemplarID,
		BearbeiterID: bearbeiterPtr, Akteur: "USER",
		Details: details,
	})
}
