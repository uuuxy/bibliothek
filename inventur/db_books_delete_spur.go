package inventur

// db_books_delete_spur.go — die Spur, die ein gelöschtes verliehenes Buch hinterlässt.
//
// Seit dem 23.08.2026 löscht „Titel löschen" auch aktuell verliehene Exemplare. Das ist
// so gewollt, aber es erzeugt eine Lage, die es vorher nicht gab: Ein Buch liegt bei
// einem Kind zu Hause, und im System existiert es nicht mehr. Bringt es das Kind zurück,
// findet der Scan nichts — kein Titel, kein Exemplar, keine Ausleihe, keine Mahnung.
//
// Genau dagegen steht diese Datei. Bevor die Zeilen fallen, wird jede offene Ausleihe
// einzeln festgehalten: Barcode, Titel und Entleiher. Der Vorgang ist damit nicht
// rückgängig zu machen, aber nachschlagbar — jemand kann im Protokoll sehen, wer das
// Buch hatte, und es von Hand klären.
//
// Der Titel-Delete schrieb bis dahin ÜBERHAUPT kein Audit. Solange verliehene Exemplare
// die Löschung blockierten, war das vertretbar; jetzt wäre es der Unterschied zwischen
// „alles gelöscht" und „spurlos verschwunden".

import (
	"context"
	"encoding/json"
	"fmt"

	"bibliothek/repository"

	"github.com/jackc/pgx/v5"
)

// offeneAusleihe ist eine laufende Ausleihe, die eine Titel-Löschung mit abräumt.
type offeneAusleihe struct {
	AusleiheID string
	ExemplarID string
	Barcode    string
	Titel      string
	Entleiher  string
	SchuelerID *string // nil bei Kollegiums-Ausleihen (Handapparat)
	Seit       string
}

// zeilenLeser ist, was die Leser vor dem Löschen brauchen — Pool oder Transaktion.
type zeilenLeser interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
}

// loescheAusleihenMitSpur entfernt ALLE Ausleihen der Titel (die laufenden wie die
// abgeschlossenen — sonst hielte ihr ON DELETE RESTRICT das Exemplar fest) und liefert
// die laufenden als Spur zurück: aus demselben Befehl, der sie entfernt.
//
// Bis zum 21.09.2026 wurden die laufenden Ausleihen VOR der Transaktion gelesen und erst
// Befehle später gelöscht: Eine Ausleihe, die dazwischen zustande kam, fiel ohne Spur
// (OFFEN.md 5.5). DELETE … RETURNING schließt das Fenster ohne zusätzliche Sperre — und
// ohne die Sperrreihenfolge Schüler → Ausleihe → Exemplar anzutasten, die eine vorgezogene
// Exemplar-Sperre gekippt hätte (docs/invarianten.md, Abschnitt 1; die Rückgabe hält
// erst die Ausleihe, dann stempelt sie das Exemplar).
//
// Schüler UND Kollegium (Handapparat): Beide Entleiher-Spalten sind polymorph, und ein
// Handapparat-Buch ist genauso weg wie ein Schülerbuch. „(unbekannt)" statt eines
// leeren Feldes, damit die Protokollzeile auch dann etwas aussagt, wenn die Zuordnung
// bereits von der Lesehistorie-Befristung getrennt wurde.
func loescheAusleihenMitSpur(ctx context.Context, tx pgx.Tx, ids []string) ([]offeneAusleihe, error) {
	// Die schreibende CTE sieht sich selbst nicht; die äußere Abfrage liest nur Tabellen,
	// die dieser Befehl nicht anfasst (Exemplare fallen erst mit dem Titel).
	rows, err := tx.Query(ctx, `
		WITH weg AS (
			DELETE FROM ausleihen a
			WHERE a.exemplar_id IN (SELECT id FROM buecher_exemplare WHERE titel_id = ANY($1::uuid[]))
			RETURNING a.id, a.exemplar_id, a.schueler_id, a.ausgeliehen_am, a.rueckgabe_am
		)
		SELECT w.id, e.id, e.barcode_id, t.titel,
		       coalesce(nullif(trim(coalesce(l.vorname,'') || ' ' || coalesce(l.nachname,'')), ''),
		                '(unbekannt)'),
		       w.schueler_id, to_char(w.ausgeliehen_am, 'YYYY-MM-DD')
		FROM weg w
		JOIN buecher_exemplare e ON w.exemplar_id = e.id
		JOIN buecher_titel t     ON e.titel_id = t.id
		LEFT JOIN leser l        ON w.schueler_id = l.id
		WHERE w.rueckgabe_am IS NULL
		ORDER BY e.barcode_id`, ids)
	if err != nil {
		return nil, fmt.Errorf("ausleihen konnten nicht gelöscht werden: %w", err)
	}
	defer rows.Close()

	var offene []offeneAusleihe
	for rows.Next() {
		var o offeneAusleihe
		if err := rows.Scan(&o.AusleiheID, &o.ExemplarID, &o.Barcode, &o.Titel, &o.Entleiher, &o.SchuelerID, &o.Seit); err != nil {
			return nil, fmt.Errorf("spur der laufenden ausleihen konnte nicht gelesen werden: %w", err)
		}
		offene = append(offene, o)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("spur der laufenden ausleihen konnte nicht gelesen werden: %w", err)
	}
	return offene, nil
}

// protokolliereOffeneAusleihen schreibt je eine SYSTEM-Zeile ins Audit-Log. Läuft in der
// Transaktion der Löschung: Entweder beides oder nichts.
//
// datensatz_id ist das EXEMPLAR — so sucht man später, wenn das Buch auf dem Tresen liegt
// und der Scan ins Leere geht. akteur 'SYSTEM' ohne bearbeiter_id, weil der Handler den
// Benutzer hier nicht durchreicht; WER gelöscht hat, steht in der Admin-Spur des
// Endpunkts, WAS dabei verschwand, steht hier.
func protokolliereOffeneAusleihen(ctx context.Context, tx pgx.Tx, offene []offeneAusleihe) error {
	for _, o := range offene {
		inhalt := map[string]any{
			"barcode_id":     o.Barcode,
			"titel":          o.Titel,
			"entleiher":      o.Entleiher,
			"ausgeliehen_am": o.Seit,
			"ausleihe_id":    o.AusleiheID,
			"action":         "titel_geloescht_mit_offener_ausleihe",
		}
		// schueler_id ist NICHT nur Information, sondern der Schlüssel, an dem die
		// Lesehistorie-Befristung diese Zeile überhaupt findet: Ihr Prädikat verlangt
		// `details ? 'schueler_id'` (repository/loeschfristen.go). Ohne ihn stünde der
		// Klarname 24 Monate hier (bis zur Audit-Aufbewahrung) statt 90 Tage wie jede
		// andere Ausleihspur — eine PII-Kopie, die länger lebt als das, was sie ersetzt.
		if o.SchuelerID != nil {
			inhalt["schueler_id"] = *o.SchuelerID
		}
		details, err := json.Marshal(inhalt)
		if err != nil {
			return fmt.Errorf("protokoll der laufenden ausleihe: %w", err)
		}
		if _, err := tx.Exec(ctx, `
			INSERT INTO audit_log (tabelle, aktion, datensatz_id, akteur, kontext, details)
			VALUES ('ausleihen', 'DELETE', $1, 'SYSTEM', $2, $3::jsonb)`,
			o.ExemplarID,
			"Titel gelöscht, Buch war zu diesem Zeitpunkt verliehen",
			string(details)); err != nil {
			return fmt.Errorf("protokoll der laufenden ausleihe: %w", err)
		}
	}
	return nil
}

// exemplarSnapshot ist das, was die Tresen-Auskunft nach dem Löschen noch braucht:
// die Exemplar-ID als datensatz_id, der Barcode als Suchschlüssel, der Titel als
// Anzeige der Trefferzeile.
type exemplarSnapshot struct {
	ID      string
	Barcode string
	Titel   string
}

// leseExemplarSnapshots sammelt ALLE Exemplare der zu löschenden Titel — in der
// Transaktion der Löschung, bevor die Zeilen fallen. Nicht nur die verliehenen:
// Die Tresen-Auskunft (repository.SucheTresenExemplare) findet gelöschte Exemplare
// ausschließlich über audit_log-Zeilen mit tabelle='buecher_exemplare' und
// details->>'barcode_id'; die Ausleihen-Spur oben sieht sie nicht. Bis zum
// 01.09.2026 fehlte dieser Snapshot hier wie beim Geschwister-Pfad DeleteTitle
// (Befund-Register): Ein per Titel-Löschung verschwundenes Buch war beim Scannen
// „nie gesehen" statt „gelöscht am …".
func leseExemplarSnapshots(ctx context.Context, tx pgx.Tx, ids []string) ([]exemplarSnapshot, error) {
	rows, err := tx.Query(ctx, `
		SELECT e.id, e.barcode_id, t.titel
		FROM buecher_exemplare e
		JOIN buecher_titel t ON e.titel_id = t.id
		WHERE e.titel_id = ANY($1::uuid[])
		ORDER BY e.barcode_id`, ids)
	if err != nil {
		return nil, fmt.Errorf("exemplar-snapshots konnten nicht gelesen werden: %w", err)
	}
	defer rows.Close()

	var snaps []exemplarSnapshot
	for rows.Next() {
		var s exemplarSnapshot
		if err := rows.Scan(&s.ID, &s.Barcode, &s.Titel); err != nil {
			return nil, fmt.Errorf("exemplar-snapshots konnten nicht gelesen werden: %w", err)
		}
		snaps = append(snaps, s)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("exemplar-snapshots konnten nicht gelesen werden: %w", err)
	}
	return snaps, nil
}

// protokolliereGeloeschteExemplare schreibt je Exemplar die Barcode-Spur — dasselbe
// Format wie DeleteCopy und das Verlust-Löschen (repository), damit die Auskunft
// alle drei Wege gleich findet. Läuft in der Transaktion der Löschung.
func protokolliereGeloeschteExemplare(ctx context.Context, tx pgx.Tx, snaps []exemplarSnapshot) error {
	for _, s := range snaps {
		details, err := json.Marshal(map[string]any{
			"barcode_id": s.Barcode,
			"titel":      s.Titel,
			"action":     repository.AuditAktionTitelGeloescht,
		})
		if err != nil {
			return fmt.Errorf("protokoll des gelöschten exemplars: %w", err)
		}
		if _, err := tx.Exec(ctx, `
			INSERT INTO audit_log (tabelle, aktion, datensatz_id, akteur, kontext, details)
			VALUES ('buecher_exemplare', 'DELETE', $1, 'SYSTEM', $2, $3::jsonb)`,
			s.ID,
			"Titel gelöscht — Exemplar mit entfernt",
			string(details)); err != nil {
			return fmt.Errorf("protokoll des gelöschten exemplars: %w", err)
		}
	}
	return nil
}

// offenerSchaden ist eine UNBEZAHLTE Forderung, die eine Titel-Löschung mit abräumt.
type offenerSchaden struct {
	ID         string
	ExemplarID string
	Barcode    string
	Titel      string
	Schuldner  string
	SchuelerID *string
	Betrag     string
	Grund      string
	Seit       string
}

// loescheSchaedenMitSpur entfernt ALLE Schadensfälle der Titel und liefert die
// unbezahlten, nicht stornierten als Spur zurück — aus demselben Befehl, wie bei den
// Ausleihen.
//
// Warum das eine eigene Spur braucht (Rasterdurchgang 06.09.2026): Beide Löschwege
// räumen `schadensfaelle` ohne Rücksicht auf `ist_bezahlt` ab; der Funktionskommentar in
// repository/audit_books.go behauptete sogar, nur ABGESCHLOSSENE Fälle würden bereinigt.
// Ein unbezahlter Schadensfall ist aber Geld, das ein Schüler der Schule schuldet, und er
// steuert sechs Entscheidungen — Kontoanzeige, Lösch-Sperre, Zusammenführen,
// Abgänger-Wächter, LUSD-Anonymisierungsbremse und das DSGVO-Löschprädikat. Mit dem Titel
// verschwand die Forderung samt allen sechs Wirkungen, und niemand konnte es später
// sehen. Für die offenen AUSLEIHEN gibt es diese Spur seit dem 23.08.2026; fürs Geld
// fehlte sie.
func loescheSchaedenMitSpur(ctx context.Context, tx pgx.Tx, ids []string) ([]offenerSchaden, error) {
	rows, err := tx.Query(ctx, `
		WITH weg AS (
			DELETE FROM schadensfaelle sf
			WHERE sf.exemplar_id IN (SELECT id FROM buecher_exemplare WHERE titel_id = ANY($1::uuid[]))
			RETURNING sf.id, sf.exemplar_id, sf.schueler_id, sf.betrag, sf.beschreibung,
			          sf.erstellt_am, sf.ist_bezahlt, sf.storniert_am
		)
		SELECT w.id, e.id, e.barcode_id, t.titel,
		       coalesce(nullif(trim(coalesce(l.vorname,'') || ' ' || coalesce(l.nachname,'')), ''),
		                '(unbekannt)'),
		       w.schueler_id, to_char(w.betrag, 'FM9999990.00'), w.beschreibung,
		       to_char(w.erstellt_am, 'YYYY-MM-DD')
		FROM weg w
		JOIN buecher_exemplare e ON w.exemplar_id = e.id
		JOIN buecher_titel t     ON e.titel_id = t.id
		LEFT JOIN leser l        ON w.schueler_id = l.id
		WHERE w.ist_bezahlt = false AND w.storniert_am IS NULL
		ORDER BY e.barcode_id`, ids)
	if err != nil {
		return nil, fmt.Errorf("schadensfälle konnten nicht gelöscht werden: %w", err)
	}
	defer rows.Close()
	var alle []offenerSchaden
	for rows.Next() {
		var o offenerSchaden
		if err := rows.Scan(&o.ID, &o.ExemplarID, &o.Barcode, &o.Titel, &o.Schuldner,
			&o.SchuelerID, &o.Betrag, &o.Grund, &o.Seit); err != nil {
			return nil, fmt.Errorf("spur der offenen schadensfälle konnte nicht gelesen werden: %w", err)
		}
		alle = append(alle, o)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("spur der offenen schadensfälle konnte nicht gelesen werden: %w", err)
	}
	return alle, nil
}

// protokolliereOffeneSchaeden hält jede offene Forderung fest, bevor sie mit dem Titel
// fällt: Wer schuldet wie viel, wofür, seit wann. Wie bei den offenen Ausleihen ist
// `schueler_id` nicht nur Information, sondern der Schlüssel, an dem die
// Lesehistorie-Befristung die Zeile findet.
func protokolliereOffeneSchaeden(ctx context.Context, tx pgx.Tx, offene []offenerSchaden) error {
	for _, o := range offene {
		inhalt := map[string]any{
			"schadensfall_id": o.ID,
			"barcode_id":      o.Barcode,
			"titel":           o.Titel,
			"schuldner":       o.Schuldner,
			"betrag":          o.Betrag,
			"beschreibung":    o.Grund,
			"erstellt_am":     o.Seit,
			"action":          "titel_geloescht_mit_offener_forderung",
		}
		if o.SchuelerID != nil {
			inhalt["schueler_id"] = *o.SchuelerID
		}
		details, err := json.Marshal(inhalt)
		if err != nil {
			return fmt.Errorf("protokoll der offenen forderung: %w", err)
		}
		if _, err := tx.Exec(ctx, `
			INSERT INTO audit_log (tabelle, aktion, datensatz_id, akteur, kontext, details)
			VALUES ('schadensfaelle', 'DELETE', $1, 'SYSTEM', $2, $3::jsonb)`,
			o.ExemplarID,
			"Titel gelöscht, es stand noch eine unbezahlte Forderung offen",
			string(details)); err != nil {
			return fmt.Errorf("protokoll der offenen forderung: %w", err)
		}
	}
	return nil
}
