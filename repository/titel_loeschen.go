package repository

import (
	"bibliothek/db"
	"context"
	"errors"
	"fmt"

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

// ErrTitelNichtGefunden meldet eine unbekannte ID beim Löschen eines Titels.
var ErrTitelNichtGefunden = errors.New("titel nicht gefunden")

// LoescheTitel entfernt einen Buchtitel vollständig aus dem Katalog und erstellt einen revisionssicheren Audit-Eintrag.
// Vor dem Löschen wird geprüft, ob noch Exemplare dieses Titels verliehen sind (was das Löschen blockiert).
// Historische Ausleihen und ALLE Schadensfälle des Titels werden bereinigt, um
// Fremdschlüssel-Fehler zu vermeiden — unbezahlte Forderungen stehen vorher im
// Protokoll (protokolliereOffeneForderungen).
func (a *pgProtokollAbfragen) LoescheTitel(ctx context.Context, titelID string, bearbeiterID string) error {
	tx, err := a.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer db.SafeRollback(ctx, tx)

	// Gehört der Titel zu einem Buch in mehreren Auflagen, muss das Werk danach noch stimmen
	// (docs/OFFEN.md 4.18). Als erstes: Die Sperre der Auflagen kommt vor jeder Zeilensperre.
	werke, err := WerkeDerTitel(ctx, tx, []string{titelID})
	if err != nil {
		return err
	}

	vorher, err := leseTitelVorDemLoeschen(ctx, tx, titelID)
	if err != nil {
		return err
	}
	if err = protokolliereWasMitDemTitelFaellt(ctx, tx, titelID); err != nil {
		return err
	}
	// Das lokal gespeicherte Cover fällt mit dem Titel, wie in der Massenaktion der
	// Bestandstabelle (inventur.DeleteBooks).
	cover, err := LokaleCoverNurDieserTitel(ctx, tx, []string{titelID})
	if err != nil {
		return err
	}
	// Verknüpfte Einträge (Schadensfälle, alte Rückgaben) löschen, um ON DELETE RESTRICT Fehler zu vermeiden
	if _, err = tx.Exec(ctx, "DELETE FROM schadensfaelle WHERE exemplar_id IN (SELECT id FROM buecher_exemplare WHERE titel_id = $1)", titelID); err != nil {
		return fmt.Errorf("failed to delete damage records for title: %w", err)
	}
	if _, err = tx.Exec(ctx, "DELETE FROM ausleihen WHERE exemplar_id IN (SELECT id FROM buecher_exemplare WHERE titel_id = $1) AND rueckgabe_am IS NOT NULL", titelID); err != nil {
		return fmt.Errorf("failed to delete past loans for title: %w", err)
	}

	// Alle zugehörigen Exemplare löschen
	if _, err = tx.Exec(ctx, "DELETE FROM buecher_exemplare WHERE titel_id = $1", titelID); err != nil {
		return fmt.Errorf("failed to delete associated copies: %w", err)
	}

	// Eigentlichen Titel-Datensatz löschen. 0 Zeilen heißt unbekannte Kennung: Der Snapshot
	// toleriert ErrNoRows und es gibt keine aktiven Ausleihen, ohne die Prüfung entstünde
	// ein Protokolleintrag über einen Titel, den es nie gab.
	tag, err := tx.Exec(ctx, "DELETE FROM buecher_titel WHERE id = $1", titelID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrTitelNichtGefunden
	}
	if err = RaeumeWerkeAuf(ctx, tx, werke); err != nil {
		return err
	}

	if err = a.protokolliereTitelLoeschung(ctx, tx, vorher, bearbeiterID); err != nil {
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

type titelExemplar struct {
	id, barcode  string
	warImBestand bool
}

// leseTitelVorDemLoeschen liest die Angaben fürs Protokoll und lehnt ab, solange ein Exemplar
// verliehen ist. Einen unbekannten Titel lässt sie durch; den meldet das DELETE.
func leseTitelVorDemLoeschen(ctx context.Context, tx pgx.Tx, titelID string) (titelVorDemLoeschen, error) {
	vorher := titelVorDemLoeschen{id: titelID}
	err := tx.QueryRow(ctx,
		`SELECT coalesce(titel,''), coalesce(autor,''), coalesce(isbn,'') FROM buecher_titel WHERE id = $1`,
		titelID,
	).Scan(&vorher.titel, &vorher.autor, &vorher.isbn)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return vorher, fmt.Errorf("failed to snapshot title for audit: %w", err)
	}
	if err := pruefeKeineAktivenAusleihen(ctx, tx, titelID); err != nil {
		return vorher, err
	}
	vorher.exemplare, err = leseTitelExemplare(ctx, tx, titelID)
	return vorher, err
}

// pruefeKeineAktivenAusleihen ist die Sicherheitsschranke vor dem Löschen: Ist ein Exemplar
// des Titels verliehen, nennt der Fehler die Barcodes.
func pruefeKeineAktivenAusleihen(ctx context.Context, tx pgx.Tx, titelID string) error {
	var verliehen []string
	rows, err := tx.Query(ctx, `
		SELECT e.barcode_id 
		FROM ausleihen a 
		JOIN buecher_exemplare e ON a.exemplar_id = e.id 
		WHERE e.titel_id = $1 AND a.rueckgabe_am IS NULL
	`, titelID)
	if err != nil {
		return fmt.Errorf("failed to check active loans for title: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var barcode string
		if err := rows.Scan(&barcode); err == nil {
			verliehen = append(verliehen, barcode)
		}
	}
	// Bricht die Iteration durch einen Verbindungsfehler vorzeitig ab, heißt das nicht
	// „keine aktiven Ausleihen"; der Titel darf dann nicht gelöscht werden.
	if err := rows.Err(); err != nil {
		return fmt.Errorf("failed to read active loans for title: %w", err)
	}
	if len(verliehen) > 0 {
		return fmt.Errorf("%w: %v", ErrTitelHatAktiveAusleihen, verliehen)
	}
	return nil
}

// leseTitelExemplare liest Kennung und Barcode jedes Exemplars vor dem Löschen. Die
// Tresen-Auskunft findet gelöschte Exemplare nur über Protokollzeilen mit
// tabelle='buecher_exemplare' und details->>'barcode_id' (FindeTresenExemplare); ohne sie
// hieße ein später gescanntes Buch „nie gesehen" statt „gelöscht am …".
func leseTitelExemplare(ctx context.Context, tx pgx.Tx, titelID string) ([]titelExemplar, error) {
	var exemplare []titelExemplar
	rows, err := tx.Query(ctx,
		`SELECT e.id::text, e.barcode_id, `+SQLWarImBestand+` FROM buecher_exemplare e WHERE e.titel_id = $1`, titelID)
	if err != nil {
		return nil, fmt.Errorf("failed to snapshot copies for audit: %w", err)
	}
	for rows.Next() {
		var s titelExemplar
		if err := rows.Scan(&s.id, &s.barcode, &s.warImBestand); err != nil {
			rows.Close()
			return nil, fmt.Errorf("failed to scan copy snapshot: %w", err)
		}
		exemplare = append(exemplare, s)
	}
	rows.Close()
	// Bricht die Iteration vorzeitig ab, fehlten Einträge still.
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to read copy snapshots: %w", err)
	}
	return exemplare, nil
}

// protokolliereWasMitDemTitelFaellt hält vor dem Löschen fest, was mit dem Titel verschwindet:
// unbezahlte Forderungen (wer wie viel wofür schuldete) und, per ON DELETE CASCADE, wer auf
// den Titel wartet. Dieselbe Regel wie in der Massenaktion der Bestandstabelle
// (titel_loeschen_wartende.go).
func protokolliereWasMitDemTitelFaellt(ctx context.Context, tx pgx.Tx, titelID string) error {
	if err := protokolliereOffeneForderungen(ctx, tx, titelID); err != nil {
		return err
	}
	wartende, err := LeseWartendeBezuege(ctx, tx, []string{titelID})
	if err != nil {
		return err
	}
	return ProtokolliereWartendeBezuege(ctx, tx, wartende)
}

// protokolliereTitelLoeschung schreibt den Eintrag zum Titel und je Exemplar den
// Barcode-Eintrag in der Form der Geschwister-Pfade (SondereExemplarAus, Verlust-Löschen), in der
// Transaktion des Löschens: entweder Löschung und Spur oder keins von beiden.
func (a *pgProtokollAbfragen) protokolliereTitelLoeschung(ctx context.Context, tx pgx.Tx, vorher titelVorDemLoeschen, bearbeiterID string) error {
	if err := a.schreibeEintrag(ctx, tx, protokollEintrag{
		Tabelle: "buecher_titel", Aktion: "DELETE", DatensatzID: vorher.id,
		BearbeiterID: &bearbeiterID, Akteur: "USER",
		Details: map[string]any{"titel": vorher.titel, "autor": vorher.autor, "isbn": vorher.isbn},
	}); err != nil {
		return err
	}
	kontext := "Titel gelöscht — Exemplar mit entfernt"
	for _, ex := range vorher.exemplare {
		if err := a.schreibeEintrag(ctx, tx, protokollEintrag{
			Tabelle: "buecher_exemplare", Aktion: "DELETE", DatensatzID: ex.id,
			BearbeiterID: &bearbeiterID, Akteur: "USER", Kontext: &kontext,
			Details: map[string]any{
				"barcode_id": ex.barcode, "titel": vorher.titel, "titel_id": vorher.id,
				"action": ProtokollAktionTitelGeloescht, ProtokollDetailWarImBestand: ex.warImBestand,
			},
		}); err != nil {
			return err
		}
	}
	return nil
}

// SondereExemplarAus bucht ein physisches Exemplar aus dem System aus (Soft-Delete) und protokolliert dies im Audit-Log.
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
	// ProtokollAktionTitelGeloescht: Das Exemplar ging mit seinem Titel (beide Lösch-Türen).
	ProtokollAktionTitelGeloescht = "titel_geloescht"
	// ProtokollAktionVerlustEndgueltigGeloescht: Ein als Verlust gebuchtes Exemplar wurde in
	// der Inventur endgültig entfernt (api/inventurlauf/inventur_verlust_aktionen.go).
	ProtokollAktionVerlustEndgueltigGeloescht = "verlust_endgueltig_geloescht"
	// ProtokollDetailWarImBestand steht an jeder dieser Spuren: ob das Exemplar ein Zugangsdatum
	// trug. Ein bestelltes, nie eingetroffenes Exemplar zählt das Abgangsbuch damit nicht.
	ProtokollDetailWarImBestand = "war_im_bestand"
)
