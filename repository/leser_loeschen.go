package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"bibliothek/db"
	"bibliothek/pkg/leserart"

	"github.com/jackc/pgx/v5"
)

// LegeLeserInPapierkorb legt einen Leser in den Papierkorb: deleted_at wird gesetzt und der
// Leser gesperrt. Seine Daten bleiben, damit sich ein versehentliches Löschen zurücknehmen
// lässt; endgültig löscht LoescheLeserEndgueltig.
//
// Die Tür ruft es mit der ID des Bearbeiters, der nächtliche Lauf mit leerer ID; der Eintrag
// nennt dann SYSTEM.
func (a *LeserAbfragen) LegeLeserInPapierkorb(ctx context.Context, leserID string, bearbeiterID string, grund string) error {
	tx, err := a.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer db.Verwerfe(ctx, tx)

	// Was der Eintrag nennt, wird vorher gelesen, aus der Tabelle leser und nicht aus der Sicht
	// schueler: Die zeigt nur Schüler, und für einen Kollegen liefe der Weg ins Leere.
	var vorname, nachname, klasse, barcodeID, art string
	var abgaengerJahr int
	err = tx.QueryRow(ctx,
		`SELECT coalesce(vorname,''), coalesce(nachname,''), coalesce(klasse,''),
		        coalesce(barcode_id,''), coalesce(abgaenger_jahr, 0), art
		 FROM leser WHERE id = $1`,
		leserID,
	).Scan(&vorname, &nachname, &klasse, &barcodeID, &abgaengerJahr, &art)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return fmt.Errorf("failed to snapshot student for audit: %w", err)
	}

	// Ein bestehender Grund der Sperre bleibt stehen (COALESCE über NULLIF), wie in
	// SperreAbgaenger: Überschriebe das Löschen ihn, hielte das Zurückholen die Zeile für bloß
	// durch das Löschen gesperrt, nähme die Sperre zurück und verletzte
	// chk_schueler_block_reason; ein von Hand gesperrter Leser ließe sich nicht zurückholen,
	// und sein Grund wäre weg.
	//
	// COALESCE auch auf deleted_at: Liegt die Zeile schon im Papierkorb, bleibt der erste
	// Zeitpunkt. Er ist die Uhr der Anonymisierung (PraedikatAnonymisierung); ein zweites
	// Löschen schöbe sie sonst nach hinten.
	tag, err := tx.Exec(ctx, `UPDATE leser
		SET deleted_at = COALESCE(deleted_at, CURRENT_TIMESTAMP),
		    ist_gesperrt = true,
		    block_reason = COALESCE(NULLIF(btrim(block_reason), ''), 'Systematisch gelöscht')
		WHERE id = $1`, leserID)
	if err != nil {
		return fmt.Errorf("soft-deleting student: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("%w: %s", ErrLeserNichtGefunden, leserID)
	}

	// Beim Kollegium geht das Konto mit. Es bleibt nicht abgeschaltet stehen, weil die Adresse
	// der Schlüssel der Anmeldung ist: Ein Konto im Papierkorb hielte sie besetzt, und wer die
	// Person neu anlegt, bekäme „Unter … steht bereits ein Zugang" über einen Eintrag, der
	// nirgends zu sehen ist. Die Spuren bleiben: Jeder Fremdschlüssel auf benutzer steht auf ON
	// DELETE SET NULL, die Einträge verlieren nur den Verweis.
	//
	// Aus dem Papierkorb kommt die Leserzeile ohne Zugang zurück. Den legt man an, indem man in
	// der Akte die Schul-E-Mail nachträgt (api/leser/leser_schul_email.go).
	var kontenGeloescht int64
	if !leserart.IstSchueler(art) {
		if kontenGeloescht, err = loescheKontenDerLeserzeile(ctx, tx, leserID, bearbeiterID); err != nil {
			return err
		}
	}

	// Ohne Bearbeiter war es der nächtliche Lauf.
	var akteur string
	var bearbeiterPtr *string
	if bearbeiterID != "" {
		akteur = "USER"
		bearbeiterPtr = &bearbeiterID
	} else {
		akteur = "SYSTEM"
	}

	kontext := "Soft-Delete Routine"

	if err = schreibeProtokollEintrag(ctx, tx, protokollEintrag{
		Tabelle: "schueler", Aktion: "UPDATE", DatensatzID: leserID,
		BearbeiterID: bearbeiterPtr, Akteur: akteur, Kontext: &kontext,
		Details: map[string]any{
			"vorname":        vorname,
			"nachname":       nachname,
			"klasse":         klasse,
			"barcode_id":     barcodeID,
			"abgaenger_jahr": abgaengerJahr,
			"grund":          grund,
			"geloescht_am":   time.Now().UTC().Format(time.RFC3339),
			"action":         "soft_delete",
			"art":            art,
			// Wie viele Zugänge dabei erloschen sind: Ohne diese Zahl ließe sich später nicht
			// mehr sagen, ob die Person je einen hatte.
			"konten_geloescht": kontenGeloescht,
		},
	}); err != nil {
		return fmt.Errorf("writing audit log: %w", err)
	}

	return tx.Commit(ctx)
}

// pruefeKeineOffenenVorgaenge lehnt das endgültige Löschen ab, solange ein Buch draußen oder
// eine Forderung unbezahlt ist: In beiden Fällen besteht noch ein Grund, die Daten zu behalten.
func pruefeKeineOffenenVorgaenge(ctx context.Context, tx pgx.Tx, leserID string) error {
	var offeneAusleihen int
	if err := tx.QueryRow(ctx, `SELECT count(*) FROM ausleihen WHERE schueler_id = $1 AND rueckgabe_am IS NULL`, leserID).Scan(&offeneAusleihen); err != nil {
		return fmt.Errorf("checking open loans: %w", err)
	}
	if offeneAusleihen > 0 {
		return &LoeschenBlockiertFehler{OffeneAusleihen: offeneAusleihen}
	}
	var offeneSchaeden int
	if err := tx.QueryRow(ctx, `SELECT count(*) FROM schadensfaelle WHERE schueler_id = $1 AND ist_bezahlt = false`, leserID).Scan(&offeneSchaeden); err != nil {
		return fmt.Errorf("checking unpaid damages: %w", err)
	}
	if offeneSchaeden > 0 {
		return &LoeschenBlockiertFehler{UnbezahlteSchadensfaelle: offeneSchaeden}
	}
	return nil
}

// tilgeSpurenUndLoescheLeser ist das endgültige Löschen für beide Wege, aus dem Papierkorb und
// für Abgänger: Die Ausleihen verlieren ihren Leser, die erledigten Schadensfälle werden
// gelöscht, die Spuren in den Nebentabellen getilgt (TilgeSchuelerSpuren, dabei fallen auch die
// Vormerkungen), die Zeile des Lesers entfernt und das Löschen ohne Personenbezug
// protokolliert.
func tilgeSpurenUndLoescheLeser(ctx context.Context, tx pgx.Tx, leserID, bearbeiterID, kontextText string) error {
	if _, err := tx.Exec(ctx, `UPDATE ausleihen SET schueler_id = NULL WHERE schueler_id = $1`, leserID); err != nil {
		return fmt.Errorf("anonymizing loans: %w", err)
	}
	if _, err := tx.Exec(ctx, `DELETE FROM schadensfaelle WHERE schueler_id = $1`, leserID); err != nil {
		return fmt.Errorf("deleting damages: %w", err)
	}
	if err := TilgeSchuelerSpuren(ctx, tx, leserID, "DSGVO-Löschung"); err != nil {
		return err
	}
	// Aus leser und nicht aus der Sicht schueler: Sonst bliebe die Zeile eines Kollegen stehen,
	// ohne Vorgänge, aber mit Namen.
	tag, err := tx.Exec(ctx, `DELETE FROM leser WHERE id = $1`, leserID)
	if err != nil {
		return fmt.Errorf("deleting student: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("%w: %s", ErrLeserNichtGefunden, leserID)
	}

	var bearbeiterPtr *string
	akteur := "SYSTEM"
	if bearbeiterID != "" {
		akteur = "USER"
		bearbeiterPtr = &bearbeiterID
	}
	if err := schreibeProtokollEintrag(ctx, tx, protokollEintrag{
		Tabelle: "schueler", Aktion: "DELETE", DatensatzID: leserID,
		BearbeiterID: bearbeiterPtr, Akteur: akteur, Kontext: &kontextText,
		Details: map[string]any{"action": "purge", "geloescht_am": time.Now().UTC().Format(time.RFC3339)},
	}); err != nil {
		return fmt.Errorf("writing purge audit log: %w", err)
	}
	return nil
}

// LoeschenBlockiertFehler meldet, dass das endgültige Löschen nicht möglich ist, weil noch etwas
// offen ist (eine Ausleihe, ein unbezahlter Schaden) oder die Zeile nicht im Papierkorb liegt.
// Genau einer der Gründe ist gesetzt. Die Tür braucht den Unterschied zu jedem anderen Fehler:
// Ein Abbruch der Verbindung, der sich als Konflikt ausgibt, schickt die Bibliothek los, ein
// Problem zu suchen, das es nicht gibt.
type LoeschenBlockiertFehler struct {
	NichtImPapierkorb        bool
	OffeneAusleihen          int
	UnbezahlteSchadensfaelle int
}

func (e *LoeschenBlockiertFehler) Error() string {
	switch {
	case e.NichtImPapierkorb:
		return "endgültiges löschen blockiert: nicht im papierkorb"
	case e.OffeneAusleihen > 0:
		return fmt.Sprintf("endgültiges löschen blockiert: %d offene ausleihen", e.OffeneAusleihen)
	}
	return fmt.Sprintf("endgültiges löschen blockiert: %d unbezahlte schadensfälle", e.UnbezahlteSchadensfaelle)
}

// LoescheLeserEndgueltig entfernt einen Leser, der im Papierkorb liegt, endgültig; was dabei
// geschieht, steht an tilgeSpurenUndLoescheLeser. Offene Ausleihen und unbezahlte Forderungen
// verhindern es.
func (a *LeserAbfragen) LoescheLeserEndgueltig(ctx context.Context, leserID string, bearbeiterID string) error {
	tx, err := a.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer db.Verwerfe(ctx, tx)

	// Endgültig gelöscht wird nur aus dem Papierkorb. Die Frage sperrt die Zeile bis zum Löschen:
	// Ein Wiederherstellen dazwischen sähe sie sonst nicht, und gelöscht würde der aktive Leser.
	var imPapierkorb bool
	err = tx.QueryRow(ctx, `SELECT deleted_at IS NOT NULL FROM leser WHERE id = $1 FOR UPDATE`, leserID).Scan(&imPapierkorb)
	if errors.Is(err, pgx.ErrNoRows) {
		return fmt.Errorf("%w: %s", ErrLeserNichtGefunden, leserID)
	}
	if err != nil {
		return fmt.Errorf("checking trash state: %w", err)
	}
	if !imPapierkorb {
		return &LoeschenBlockiertFehler{NichtImPapierkorb: true}
	}

	if err = pruefeKeineOffenenVorgaenge(ctx, tx, leserID); err != nil {
		return err
	}
	if err = tilgeSpurenUndLoescheLeser(ctx, tx, leserID, bearbeiterID, "DSGVO-Löschung (Purge)"); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// LoescheKollegenAusDemPapierkorb entfernt für den nächtlichen Lauf einen Kollegen, wenn er beim
// Löschen noch löschreif ist (praedikatKollegenPapierkorb, dieselbe Bedingung wie die Liste des
// Laufs). Der Lauf liest seine Liste vorher; wer dazwischen wiederhergestellt und neu gelöscht
// wurde, verlöre sonst seine 180 Tage. Die Zeile bleibt bis zum Löschen gesperrt. false heißt:
// nicht mehr löschreif, nichts geändert.
func (a *LeserAbfragen) LoescheKollegenAusDemPapierkorb(ctx context.Context, leserID string) (bool, error) {
	tx, err := a.db.Begin(ctx)
	if err != nil {
		return false, err
	}
	defer db.Verwerfe(ctx, tx)

	bedingung := praedikatKollegenPapierkorb(KulanzJob)
	argumente := append(bedingung.Args, leserID)
	var loeschreif int
	err = tx.QueryRow(ctx, fmt.Sprintf(`SELECT 1 FROM leser WHERE id = $%d AND %s FOR UPDATE`,
		len(argumente), bedingung.Where), argumente...).Scan(&loeschreif)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if err = tilgeSpurenUndLoescheLeser(ctx, tx, leserID, "", "DSGVO-Löschung (Purge)"); err != nil {
		return false, err
	}
	return true, tx.Commit(ctx)
}

// LoescheAbgaengerEndgueltig entfernt einen Abgänger endgültig, für den nächtlichen Lauf. Der
// Abgänger liegt nicht im Papierkorb; wen es trifft (Karenz, Merkmal Abgänger), wählt der
// Rufer. Offene Vorgänge prüft die Funktion trotzdem, weil zwischen Auswahl und Löschen ein
// Buch ausgeliehen sein kann.
func (a *LeserAbfragen) LoescheAbgaengerEndgueltig(ctx context.Context, leserID string, bearbeiterID string) error {
	tx, err := a.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer db.Verwerfe(ctx, tx)

	if err = pruefeKeineOffenenVorgaenge(ctx, tx, leserID); err != nil {
		return err
	}
	if err = tilgeSpurenUndLoescheLeser(ctx, tx, leserID, bearbeiterID, "DSGVO-Löschung (Abgänger-Cronjob)"); err != nil {
		return err
	}
	return tx.Commit(ctx)
}
