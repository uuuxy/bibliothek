package repository

import (
	"bibliothek/db"
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
)

// ErrKontoHatAusleihen signalisiert, dass ein Benutzer nicht gelöscht werden kann, weil er
// noch Bücher ausgeliehen hat (offene Ausleihen auf seiner Leserzeile).
// Nutzer-sichtbar (409) — deshalb ohne Wörter aus dem Code (konto_loeschen_meldung_test.go).
//
//nolint:staticcheck // ST1005: bewusst großgeschrieben, Endnutzer-Meldung
var ErrKontoHatAusleihen = errors.New("Das Konto hat noch ausgeliehene Bücher — bitte zuerst zurückbuchen")

// LoescheKonto löscht einen Systembenutzer endgültig aus der Datenbank und erfasst die Löschung im Audit-Log.
func (a *pgProtokollAbfragen) LoescheKonto(ctx context.Context, kontoID string, bearbeiterID string) error {
	tx, err := a.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer db.SafeRollback(ctx, tx)

	// Snapshot erstellen: Benutzerdaten vor dem Löschen sichern. Die `leser_id` wird
	// mitgelesen, weil sie nach dem DELETE nicht mehr zu finden ist (ON DELETE SET NULL am
	// Konto, und die Zeile selbst ist dann weg).
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

	// Schutz vor verwaisten Ausleihen: Die Bücher hängen an der LESERZEILE des Kontos,
	// und benutzer.leser_id trägt ON DELETE SET NULL. Ein DELETE ließe die Bücher im
	// Status "ausgeliehen" zurück, während die Leserzeile ihren Namen verliert — dauerhaft
	// blockiert und nicht mehr zuordenbar. Deshalb die Löschung verweigern, solange offene
	// Ausleihen bestehen.
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

	// 0 Zeilen = unbekannte ID: Der Snapshot oben toleriert ErrNoRows, der Ausleihen-Zähler
	// steht auf 0 — eine unbekannte Benutzer-ID lief glatt durch und hinterließ einen
	// DELETE-Audit-Eintrag über ein Konto, das es nie gab (Phantom-Erfolg-Sweep 31.08.2026;
	// der Schüler-Purge 140 Zeilen weiter unten prüft längst).
	tag, err := tx.Exec(ctx, "DELETE FROM benutzer WHERE id = $1", kontoID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrKontoNichtGefunden
	}

	// Die Leserzeile des Kontos geht MIT, wenn sie unberührt ist.
	//
	// Anlass: Eine abgelehnte Zugangsanfrage. Die Selbstanmeldung legt ein Konto ohne
	// Leserzeile an, der Wächter `trg_benutzer_hat_leserzeile` hängt eine frische daran.
	// Wird die Anfrage abgelehnt und das Konto gelöscht, blieb diese Zeile als Waise in der
	// Leserdatei stehen: ohne Ausweis, ohne Vorgänge, mit dem aus der E-Mail-Adresse
	// geratenen Namen (Rasterdurchgang 16.09.2026, OFFEN.md 5.18, Fund 1).
	//
	// „Unberührt" heißt: kein Ausweis und nichts, was an ihr hängt. Hängt doch etwas daran —
	// Bücher, ein Schadensfall, ein Foto, eine Vormerkung —, BLEIBT die Zeile stehen; sie
	// gehört dann einem Menschen, der in der Leserdatei steht, und nicht dem Konto.
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

// loescheUnberuehrteLeserzeile entfernt die Leserzeile eines gelöschten Kontos, sofern an
// ihr nichts hängt. Liefert zurück, ob gelöscht wurde, und sonst den Grund.
//
// Die Liste der Tabellen kommt aus dem KATALOG der Datenbank, nicht aus dem Go-Code. Das
// ist der Kern und keine Spielerei: An `leser` hängen Fremdschlüssel mit gemischter
// Löschwirkung — RESTRICT bei Ausleihen und Schadensfällen, CASCADE bei Fotos und
// Vormerkungen, SET NULL bei Bescheiden, Nachbuch-Meldungen und Konten. Eine Aufzählung in
// Go hielte das nur bis zur nächsten Tabelle, die jemand anhängt: Bei CASCADE verschwänden
// deren Zeilen still mit, bei SET NULL verlören sie ihren Bezug. Wer eine Tabelle anhängt,
// bekommt die Prüfung hier geschenkt.
//
// Der Ausweis ist der Sonderfall, den keine Fremdschlüssel-Abfrage sieht: Er steht als
// Spalte in der Zeile selbst. Eine Nummer ist vergeben und wird nie recycelt — wer eine
// hat, steht in der Leserdatei.
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

	kinder, err := fremdschluesselAufLeser(ctx, tx)
	if err != nil {
		return false, "", err
	}
	// Kein Fremdschlüssel gefunden heißt nicht „nichts hängt daran", sondern dass die
	// Abfrage nicht getan hat, was sie soll. Dann lieber die Zeile stehen lassen.
	if len(kinder) == 0 {
		return false, "katalog leer", nil
	}
	tabelle, err := ersteTabelleMitBezug(ctx, tx, kinder, *leserID)
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
	// aber auch kein „gelöscht" — der Audit-Eintrag soll nicht behaupten, was nicht geschah.
	if tag.RowsAffected() == 0 {
		return false, "schon weg", nil
	}
	return true, "", nil
}

// leserKind ist eine Spalte einer anderen Tabelle, die per Fremdschlüssel auf leser zeigt.
type leserKind struct{ tabelle, spalte string }

// fremdschluesselAufLeser liest aus dem Katalog der Datenbank, welche Spalten auf leser zeigen.
func fremdschluesselAufLeser(ctx context.Context, tx pgx.Tx) ([]leserKind, error) {
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

// ersteTabelleMitBezug nennt die erste Tabelle, in der etwas an der Leserzeile hängt, sonst "".
func ersteTabelleMitBezug(ctx context.Context, tx pgx.Tx, kinder []leserKind, leserID string) (string, error) {
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
