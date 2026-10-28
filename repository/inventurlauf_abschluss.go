package repository

import (
	"context"
	"errors"
	"fmt"
)

// SperreLaufFuerAbschluss lädt den offenen Lauf und sperrt seine Zeile mit FOR UPDATE, das
// Gegenstück zum FOR SHARE des Scans: Der Abschluss wartet damit auf alle Scans, die noch
// schreiben, und bestimmt die Verluste erst danach. Läuft in der Transaktion des Abschlusses;
// die Tür reicht sie als db herein.
func (a *InventurAbfragen) SperreLaufFuerAbschluss(ctx context.Context, id string) (*Inventurlauf, error) {
	var s Inventurlauf
	err := a.db.QueryRow(ctx, `
		SELECT id, scope_type, scope_signatur, scope_subject, scope_grade, scope_label,
		       gestartet_von::text, gestartet_am::text,
		       (SELECT count(*) FROM inventur_erfassungen WHERE session_id = $1)
		FROM inventur_sessions
		WHERE id = $1 AND abgeschlossen_am IS NULL
		FOR UPDATE
	`, id).Scan(&s.ID, &s.BereichArt, &s.Signatur, &s.Fach, &s.Jahrgang, &s.Bezeichnung,
		&s.GestartetVon, &s.GestartetAm, &s.Erfasst)
	if err != nil {
		return nil, err
	}
	return &s, nil
}

// SchliesseLaufAb schließt einen Lauf ab: Jedes Exemplar im Bereich, das in diesem Lauf nicht
// erfasst wurde, wird als Verlust ausgesondert. Verliehene Exemplare liegen nicht im Bereich
// (InventurBereich.bedingung); ein Buch beim Schüler wird nie als verloren gebucht. Was ein
// anderer Lauf erfasst hat, bleibt unberührt, weil jede Erfassung an ihrem Lauf hängt.
//
// bearbeiterID nennt, wer abschließt: Der Lauf trägt nur, wer ihn begonnen hat, und jedes als
// Verlust gebuchte Exemplar steht mit dieser Person im Protokoll (protokolliereAussonderung).
func (a *InventurAbfragen) SchliesseLaufAb(ctx context.Context, laufID string, bereich InventurBereich, bearbeiterID string) (int, error) {
	if bearbeiterID == "" {
		return 0, ErrAussonderungOhneBearbeiter
	}
	// Die Bedingung des Bereichs kommt aus der einen Quelle; die ID des Laufs hängt als letzter
	// Platzhalter an.
	bedingung, args := bereich.bedingung(1)
	laufPlatz := len(args) + 1
	args = append(args, laufID)
	// Die Verluste werden mitgeschrieben und nicht nur gezählt: Nach der Aussonderung fallen
	// die Exemplare aus der Bedingung des Bereichs, und ohne die Mitschrift ließe sich weder
	// nachsehen, welche fehlen, noch eine Liste zum Nachsuchen drucken. Titel, Autor und
	// Signatur stehen dort als Abschrift, damit der Bericht lesbar bleibt, wenn Exemplar oder
	// Titel später endgültig gelöscht werden; die Signatur ist t.signatur, der Text vom
	// Buchrücken.
	//
	// Gezählt wird die Aussonderung, nicht die Mitschrift: Die Zeilenzahl der ganzen Anweisung
	// wäre die des abschließenden INSERT, und überspringt ON CONFLICT eine Zeile, meldete der
	// Lauf weniger Verluste, als er gebucht hat. Deshalb liefert das SELECT am Ende die IDs aus
	// dem UPDATE.
	query := fmt.Sprintf(`
		WITH verloren AS (
			UPDATE buecher_exemplare e
			SET ist_ausleihbar = false,
			    ist_ausgesondert = true,
			    aussonderung_grund = 'VERLUST',
			    zustand_notiz = 'Verlust bei Inventur',
			    aktualisiert_am = CURRENT_TIMESTAMP
			FROM buecher_titel t
			WHERE e.titel_id = t.id
			  AND %s
			  AND %s
			RETURNING e.id, e.barcode_id, t.titel, coalesce(t.autor, '') AS autor,
			          coalesce(btrim(t.signatur), '') AS signatur
		), mitschrift AS (
			INSERT INTO inventur_verluste (session_id, exemplar_id, barcode_id, titel, autor, signatur)
			SELECT $%d, v.id, v.barcode_id, v.titel, v.autor, v.signatur
			FROM verloren v
			ON CONFLICT DO NOTHING
			RETURNING 1
		)
		SELECT coalesce(array_agg(v.id::text), '{}') FROM verloren v
	`, bedingung, nichtErfasst(laufPlatz), laufPlatz)
	var verlorenIDs []string
	if err := a.db.QueryRow(ctx, query, args...).Scan(&verlorenIDs); err != nil {
		return 0, fmt.Errorf("verluste markieren fehlgeschlagen: %w", err)
	}
	verloren := len(verlorenIDs)
	if err := protokolliereAussonderung(ctx, a.db, verlorenIDs, bearbeiterID, AussonderungsWegInventur,
		map[string]any{"inventur_session_id": laufID}); err != nil {
		return 0, err
	}

	// erfasst_gemeldet wird mit eingefroren: inventur_erfassungen fällt per ON DELETE CASCADE
	// mit dem Exemplar, und live gezählt sänke die Zahl eines abgeschlossenen Laufs, sobald
	// später ein Buch aus dem Bestand fällt.
	if _, err := a.db.Exec(ctx, `
		UPDATE inventur_sessions
		SET abgeschlossen_am = now(), verloren_gemeldet = $2,
		    erfasst_gemeldet = (SELECT count(*) FROM inventur_erfassungen WHERE session_id = $1)
		WHERE id = $1 AND abgeschlossen_am IS NULL
	`, laufID, verloren); err != nil {
		return 0, fmt.Errorf("session abschliessen fehlgeschlagen: %w", err)
	}
	return verloren, nil
}

// ErrInventurlaufNichtGefunden meldet eine unbekannte ID beim Verwerfen.
var ErrInventurlaufNichtGefunden = errors.New("inventur-session nicht gefunden")

// VerwerfeLauf beendet einen Lauf ohne Verlustbuchung, für abgebrochene oder liegengebliebene
// Inventuren. Die Erfassungen bleiben; der Bereich wird wieder frei für einen neuen Lauf.
// verworfen = true unterscheidet das Verwerfen vom Abschluss ohne Verlust: Die Liste früherer
// Inventuren nennte den Lauf sonst „vollständig".
//
// Trifft die Anweisung keine Zeile, gibt es den Lauf nicht (Fehler), oder er war schon
// abgeschlossen (in Ordnung, der Bereich ist frei). Ein Abschluss wird dabei nicht nachträglich
// zum Verwerfen: Die Bedingung abgeschlossen_am IS NULL lässt die Zeile stehen.
func (a *InventurAbfragen) VerwerfeLauf(ctx context.Context, laufID string) error {
	tag, err := a.db.Exec(ctx, `
		UPDATE inventur_sessions
		SET abgeschlossen_am = now(), verloren_gemeldet = 0, verworfen = true,
		    erfasst_gemeldet = (SELECT count(*) FROM inventur_erfassungen WHERE session_id = $1)
		WHERE id = $1 AND abgeschlossen_am IS NULL
	`, laufID)
	if err != nil {
		return fmt.Errorf("session abbrechen fehlgeschlagen: %w", err)
	}
	if tag.RowsAffected() == 0 {
		var existiert bool
		if err := a.db.QueryRow(ctx,
			`SELECT EXISTS(SELECT 1 FROM inventur_sessions WHERE id = $1)`, laufID,
		).Scan(&existiert); err != nil {
			return fmt.Errorf("session abbrechen: existenz prüfen fehlgeschlagen: %w", err)
		}
		if !existiert {
			return ErrInventurlaufNichtGefunden
		}
	}
	return nil
}
