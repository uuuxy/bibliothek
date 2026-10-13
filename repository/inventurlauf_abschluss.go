package repository

import (
	"context"
	"errors"
	"fmt"
)

// SperreLaufFuerAbschluss lädt die offene Session und sperrt ihre Zeile
// EXKLUSIV (FOR UPDATE) — die Gegenseite zum FOR SHARE des Scans. Der Abschluss wartet
// damit auf alle noch schreibenden Scans und zieht seinen Verlust-Snapshot erst danach.
// Muss innerhalb der Abschluss-Transaktion laufen (der Handler übergibt die Tx als db).
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

// SchliesseLaufAb schließt eine Session ab: Alle physisch erwartbaren Exemplare
// im Scope, die in DIESER Session nicht erfasst wurden, werden als Verlust markiert.
// Weil "im Scope" verliehene Bücher ausschliesst (InventurBereich.bedingung), wird ein
// beim Schüler befindliches Buch nie fälschlich als verloren gebucht.
//
// Entscheidend gegenüber dem alten Modell: Nur die NICHT in dieser Session erfassten
// Exemplare gelten als vermisst — der Fortschritt einer parallelen Session bleibt
// unberührt, weil er session-gebunden in inventur_erfassungen liegt.
//
// bearbeiterID nennt, wer abschließt: Die Inventur trägt nur, wer sie begonnen hat, und jedes
// als Verlust gebuchte Exemplar steht mit dieser Person im Protokoll
// (ProtokolliereAussonderung).
func (a *InventurAbfragen) SchliesseLaufAb(ctx context.Context, laufID string, bereich InventurBereich, bearbeiterID string) (int, error) {
	if bearbeiterID == "" {
		return 0, ErrAussonderungOhneBearbeiter
	}
	// Scope-Prädikat (physisch + Dimensionen) aus der einen Quelle; die Session-ID hängt
	// als letzter Platzhalter hinten dran.
	bedingung, args := bereich.bedingung(1)
	laufPlatz := len(args) + 1
	args = append(args, laufID)
	// Die gebuchten Verluste werden zugleich MITGESCHRIEBEN, nicht nur gezählt.
	//
	// Vorher gab dieser Aufruf allein eine Zahl zurück: „47 Verluste". Welche 47, erfuhr
	// niemand — und danach war es auch nicht mehr herleitbar, weil die Exemplare durch die
	// Aussonderung aus der Scope-Bedingung fallen. Damit liess sich weder nachsehen, ob ein
	// Buch wirklich fehlt oder nur im falschen Regal stand, noch eine Liste zum Nachsuchen
	// ausdrucken.
	//
	// Titel, Autor und Signatur werden als ABSCHRIFT übernommen (siehe Migration 059):
	// Der Bericht muss auch dann noch lesbar sein, wenn Exemplar oder Titel später
	// endgültig gelöscht werden — und genau dann braucht man ihn, wenn jemand nachfragt.
	// Gezaehlt wird die AUSSONDERUNG, nicht die Mitschrift.
	//
	// Die Signatur kam zunaechst per Join auf die Tabelle `signatures` ueber
	// t.signature_id — eine Spalte, die ausser Migration 021 nie jemand geschrieben hat.
	// Die Abschrift war damit IMMER leer, und ausgerechnet die Sortierung "nach Signatur",
	// mit der man mit dem Zettel durchs Regal laeuft, sortierte nach nichts. Quelle ist
	// jetzt t.signatur, der Text vom Buchruecken (Migration 060).
	//
	// Der erste Entwurf las tag.RowsAffected() des Gesamt-Statements — das ist bei einem
	// abschliessenden INSERT dessen Zeilenzahl, nicht die des UPDATE. Solange beide gleich
	// sind, faellt das nicht auf; ueberspringt ON CONFLICT je eine Zeile, meldete die
	// Inventur weniger Verluste, als sie tatsaechlich gebucht hat. Deshalb liefert das
	// abschliessende SELECT die Kennungen aus dem UPDATE-CTE; ihre Zahl ist die der Verluste.
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
			  AND NOT EXISTS (
			      SELECT 1 FROM inventur_erfassungen ie
			      WHERE ie.session_id = $%d AND ie.exemplar_id = e.id
			  )
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
	`, bedingung, laufPlatz, laufPlatz)
	var verlorenIDs []string
	if err := a.db.QueryRow(ctx, query, args...).Scan(&verlorenIDs); err != nil {
		return 0, fmt.Errorf("verluste markieren fehlgeschlagen: %w", err)
	}
	verloren := len(verlorenIDs)
	if err := ProtokolliereAussonderung(ctx, a.db, verlorenIDs, bearbeiterID, AussonderungsWegInventur,
		map[string]any{"inventur_session_id": laufID}); err != nil {
		return 0, err
	}

	// erfasst_gemeldet mit einfrieren (Migration 103): inventur_erfassungen fällt per
	// ON DELETE CASCADE mit dem Exemplar. Live gezählt sank die Zahl eines längst
	// abgeschlossenen Durchgangs, sobald später ein Buch aus dem Bestand fiel — neben
	// einem verloren_gemeldet, das feststand. Was gezählt wurde, wurde gezählt.
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

// ErrInventurlaufNichtGefunden meldet eine unbekannte Session-ID beim Abbrechen.
var ErrInventurlaufNichtGefunden = errors.New("inventur-session nicht gefunden")

// VerwerfeLauf verwirft eine Session ohne Verlustbuchung — für abgebrochene
// oder hängengebliebene Inventuren. Die Erfassungen bleiben (CASCADE räumt sie erst
// beim echten Löschen); der Scope wird dadurch wieder frei für einen Neustart.
//
// verworfen = true (Migration 149): abgeschlossen_am allein machte aus dem Verwerfen einen
// Abschluss ohne Verlust, und die Liste früherer Inventuren schrieb „vollständig".
//
// 0 Zeilen hatte hier ZWEI Bedeutungen und beide wurden als „abgebrochen" quittiert
// (Phantom-Erfolg-Sweep 31.08.2026): Session existiert nicht (→ Fehler, der Aufrufer
// redet über die falsche ID) oder war schon abgeschlossen (→ idempotent in Ordnung,
// das Ziel „Scope frei" ist erreicht). Ein Abschluss wird dabei nicht nachträglich zum
// Verwerfen: Die Bedingung abgeschlossen_am IS NULL lässt die Zeile stehen.
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
