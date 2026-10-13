package repository

import (
	"context"
	"errors"
	"fmt"

	"bibliothek/db"

	"github.com/jackc/pgx/v5"
)

// ErrKontoHatAusleihen meldet, dass ein Konto nicht gelöscht werden kann, weil auf seiner
// Leserzeile noch Ausleihen offen sind. Der Text steht so vor dem Menschen (409) und nennt
// deshalb kein Wort aus dem Code (konto_loeschen_meldung_test.go).
//
//nolint:staticcheck // ST1005: bewusst großgeschrieben, Endnutzer-Meldung
var ErrKontoHatAusleihen = errors.New("Das Konto hat noch ausgeliehene Bücher — bitte zuerst zurückbuchen")

// LoescheKonto löscht ein Konto und schreibt den Eintrag dazu. Offene Ausleihen auf seiner
// Leserzeile verhindern es; eine unberührte Leserzeile geht mit.
func (a *pgProtokollAbfragen) LoescheKonto(ctx context.Context, kontoID string, bearbeiterID string) error {
	tx, err := a.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer db.SafeRollback(ctx, tx)

	// Was der Eintrag nennt, wird vor dem Löschen gelesen. Die leser_id auch: Nach dem DELETE
	// ist sie nicht mehr zu finden.
	var vorname, nachname, email, rolle string
	var leserID *string
	err = tx.QueryRow(ctx,
		`SELECT coalesce(vorname,''), coalesce(nachname,''), coalesce(email,''), coalesce(rolle::text,''), leser_id
		 FROM benutzer WHERE id = $1`,
		kontoID,
	).Scan(&vorname, &nachname, &email, &rolle, &leserID)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return fmt.Errorf("failed to snapshot user for audit: %w", err)
	}

	// Die Bücher hängen an der Leserzeile des Kontos und blieben nach seinem Löschen als
	// ausgeliehen stehen. Deshalb kein Löschen, solange Ausleihen offen sind.
	var aktiveAusleihen int
	if err = tx.QueryRow(ctx,
		`SELECT count(*) FROM ausleihen a
		  JOIN benutzer b ON b.leser_id = a.schueler_id
		 WHERE b.id = $1 AND a.rueckgabe_am IS NULL`,
		kontoID,
	).Scan(&aktiveAusleihen); err != nil {
		return fmt.Errorf("failed to check active loans for user: %w", err)
	}
	if aktiveAusleihen > 0 {
		return fmt.Errorf("%w (%d offen)", ErrKontoHatAusleihen, aktiveAusleihen)
	}

	// Keine getroffene Zeile heißt unbekannte ID: Die Aufnahme oben lässt ein fehlendes Konto
	// durch, und ohne diese Prüfung entstünde ein Eintrag über ein Konto, das es nie gab.
	tag, err := tx.Exec(ctx, "DELETE FROM benutzer WHERE id = $1", kontoID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrKontoNichtGefunden
	}

	// Die Leserzeile des Kontos geht mit, wenn sie unberührt ist. Der Fall ist die abgelehnte
	// Zugangsanfrage: Die Selbstanmeldung legt ein Konto an, der Trigger
	// trg_benutzer_hat_leserzeile hängt eine frische Leserzeile daran, und nach dem Löschen des
	// Kontos bliebe sie als Waise stehen, ohne Ausweis und mit dem aus der Adresse geratenen
	// Namen.
	//
	// „Unberührt" heißt: kein Ausweis und nichts, was an ihr hängt. Hängt etwas daran (Bücher,
	// ein Schadensfall, ein Foto, eine Vormerkung), bleibt die Zeile; sie gehört dann einem
	// Menschen in der Lesertabelle und nicht dem Konto.
	geloescht, grund, err := loescheUnberuehrteLeserzeile(ctx, tx, leserID)
	if err != nil {
		return err
	}

	// Die Leserkennung kommt nur in den Eintrag, wenn die Leserzeile stehen bleibt: Über sie
	// findet die Auskunft des Lesers das frühere Konto. Ging die Zeile mit, gibt es keinen
	// Leser mehr, dem die Kennung etwas sagte (protokolliereKontoLoeschung).
	var bleibenderLeser *string
	if !geloescht {
		bleibenderLeser = leserID
	}
	if err = a.protokolliereKontoLoeschung(ctx, tx, kontoLoeschung{
		kontoID: kontoID, vorname: vorname, nachname: nachname, email: email, rolle: rolle,
		leserID: bleibenderLeser, bearbeiterID: bearbeiterID,
		ueberBenutzerverwaltung: true, leserzeileGeloescht: geloescht, leserzeileBleibtWegen: grund,
	}); err != nil {
		return err
	}

	return tx.Commit(ctx)
}

// loescheUnberuehrteLeserzeile entfernt die Leserzeile eines gelöschten Kontos, sofern nichts
// an ihr hängt, und liefert, ob gelöscht wurde, sonst den Grund.
//
// Welche Tabellen an leser hängen, liest sie aus dem Katalog der Datenbank und nicht aus einer
// Liste im Code: Die Fremdschlüssel wirken verschieden (RESTRICT bei Ausleihen und
// Schadensfällen, CASCADE bei Fotos und Vormerkungen, SET NULL bei Bescheiden,
// Nachbuch-Meldungen und Konten). Mit einer Liste verschwänden die Zeilen der nächsten
// angehängten Tabelle bei CASCADE mit oder verlören bei SET NULL ihren Bezug.
//
// Den Ausweis sieht keine Abfrage über Fremdschlüssel: Er steht als Spalte in der Zeile. Eine
// Nummer wird nie neu vergeben; wer eine hat, bleibt in der Lesertabelle.
func loescheUnberuehrteLeserzeile(ctx context.Context, tx pgx.Tx, leserID *string) (bool, string, error) {
	if leserID == nil || *leserID == "" {
		return false, "", nil
	}

	var hatAusweis bool
	if err := tx.QueryRow(ctx,
		`SELECT barcode_id IS NOT NULL AND barcode_id <> '' FROM leser WHERE id = $1`, *leserID,
	).Scan(&hatAusweis); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, "", nil // schon weg (Papierkorb endgültig geleert)
		}
		return false, "", fmt.Errorf("leserzeile lesen: %w", err)
	}
	if hatAusweis {
		return false, "ausweis", nil
	}

	kinder, err := ladeFremdschluesselAufLeser(ctx, tx)
	if err != nil {
		return false, "", err
	}
	// Kein Fremdschlüssel gefunden heißt nicht „nichts hängt daran", sondern dass die
	// Abfrage nicht getan hat, was sie soll. Dann lieber die Zeile stehen lassen.
	if len(kinder) == 0 {
		return false, "katalog leer", nil
	}
	tabelle, err := findeErsteTabelleMitBezug(ctx, tx, kinder, *leserID)
	if err != nil {
		return false, "", err
	}
	if tabelle != "" {
		return false, tabelle, nil
	}

	tag, err := tx.Exec(ctx, `DELETE FROM leser WHERE id = $1`, *leserID)
	if err != nil {
		return false, "", fmt.Errorf("leserzeile löschen: %w", err)
	}
	// Null Zeilen heißt: Die Zeile war zwischen Prüfung und Löschung schon weg. Kein Fehler,
	// aber auch kein „gelöscht": Der Eintrag soll nicht behaupten, was nicht geschah.
	if tag.RowsAffected() == 0 {
		return false, "schon weg", nil
	}
	return true, "", nil
}

// leserKind ist eine Spalte einer anderen Tabelle, die per Fremdschlüssel auf leser zeigt.
type leserKind struct{ tabelle, spalte string }

// ladeFremdschluesselAufLeser liest aus dem Katalog der Datenbank, welche Spalten auf leser zeigen.
func ladeFremdschluesselAufLeser(ctx context.Context, tx pgx.Tx) ([]leserKind, error) {
	rows, err := tx.Query(ctx, `
		SELECT c.conrelid::regclass::text, a.attname
		  FROM pg_constraint c
		  JOIN unnest(c.conkey) AS k(attnum) ON true
		  JOIN pg_attribute a ON a.attrelid = c.conrelid AND a.attnum = k.attnum
		 WHERE c.contype = 'f' AND c.confrelid = 'leser'::regclass`)
	if err != nil {
		return nil, fmt.Errorf("fremdschlüssel auf leser lesen: %w", err)
	}
	var kinder []leserKind
	for rows.Next() {
		var k leserKind
		if err := rows.Scan(&k.tabelle, &k.spalte); err != nil {
			rows.Close()
			return nil, err
		}
		kinder = append(kinder, k)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return kinder, nil
}

// findeErsteTabelleMitBezug nennt die erste Tabelle, in der etwas an der Leserzeile hängt, sonst "".
func findeErsteTabelleMitBezug(ctx context.Context, tx pgx.Tx, kinder []leserKind, leserID string) (string, error) {
	for _, k := range kinder {
		// Die Namen stammen aus dem Katalog, nicht aus einer Eingabe; `regclass` liefert sie
		// bereits so, wie Postgres sie wieder liest.
		var haengt bool
		if err := tx.QueryRow(ctx,
			`SELECT EXISTS (SELECT 1 FROM `+k.tabelle+` WHERE `+k.spalte+` = $1)`, leserID,
		).Scan(&haengt); err != nil {
			return "", fmt.Errorf("%s prüfen: %w", k.tabelle, err)
		}
		if haengt {
			return k.tabelle, nil
		}
	}
	return "", nil
}
