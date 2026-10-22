package inventur

// Die Spur, die ein gelöschtes verliehenes Buch hinterlässt.
//
// „Titel löschen" löscht auch gerade verliehene Exemplare. Das ist gewollt, aber es erzeugt eine
// Lage, in der ein Buch bei einem Kind zu Hause liegt und im Programm nicht mehr besteht. Bringt
// es das Kind zurück, findet der Scan nichts: keinen Titel, kein Exemplar, keine Ausleihe, keine
// Mahnung.
//
// Genau dagegen steht diese Datei. Bevor die Zeilen fallen, wird jede offene Ausleihe einzeln
// festgehalten: Barcode, Titel und Entleiher. Der Vorgang lässt sich damit nicht rückgängig
// machen, aber nachschlagen: Jemand kann im Protokoll sehen, wer das Buch hatte, und es von Hand
// klären. Ohne diese Spur wäre „alles gelöscht" dasselbe wie „spurlos verschwunden".

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
	SchuelerID *string // nil, wenn die Ausleihe keinem Leser mehr zugeordnet ist
	Seit       string
}

// loescheAusleihenMitSpur entfernt alle Ausleihen der Titel (die laufenden wie die
// abgeschlossenen, sonst hielte ihr ON DELETE RESTRICT das Exemplar fest) und liefert die
// laufenden als Spur zurück, aus demselben Befehl, der sie entfernt.
//
// Würden die laufenden Ausleihen vor der Transaktion gelesen und erst Befehle später gelöscht,
// fiele eine Ausleihe, die dazwischen zustande kommt, ohne Spur. DELETE … RETURNING schließt das
// Fenster ohne zusätzliche Sperre und ohne die Reihenfolge der Sperren Schüler → Ausleihe →
// Exemplar anzutasten, die eine vorgezogene Sperre des Exemplars gekippt hätte
// (docs/invarianten.md, Abschnitt 1; die Rückgabe hält erst die Ausleihe, dann stempelt sie das
// Exemplar).
//
// Schüler und Kollegium (Handapparat) stehen beide in leser, und ein Buch aus dem Handapparat ist
// genauso weg wie ein Schülerbuch. „(unbekannt)" statt eines leeren Feldes, damit die Zeile im
// Protokoll auch dann etwas aussagt, wenn die Befristung der Lesehistorie die Zuordnung schon
// getrennt hat.
func loescheAusleihenMitSpur(ctx context.Context, tx pgx.Tx, ids []string) ([]offeneAusleihe, error) {
	// Die schreibende CTE sieht sich selbst nicht; die äußere Abfrage liest nur Tabellen,
	// die dieser Befehl nicht anfasst (Exemplare fallen erst mit dem Titel).
	zeilen, err := tx.Query(ctx, `
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
	defer zeilen.Close()

	var offene []offeneAusleihe
	for zeilen.Next() {
		var o offeneAusleihe
		if err := zeilen.Scan(&o.AusleiheID, &o.ExemplarID, &o.Barcode, &o.Titel, &o.Entleiher, &o.SchuelerID, &o.Seit); err != nil {
			return nil, fmt.Errorf("spur der laufenden ausleihen konnte nicht gelesen werden: %w", err)
		}
		offene = append(offene, o)
	}
	if err := zeilen.Err(); err != nil {
		return nil, fmt.Errorf("spur der laufenden ausleihen konnte nicht gelesen werden: %w", err)
	}
	return offene, nil
}

// protokolliereOffeneAusleihen schreibt je eine SYSTEM-Zeile ins Audit-Log. Läuft in der
// Transaktion der Löschung: Entweder beides oder nichts.
//
// datensatz_id ist das Exemplar — so sucht man später, wenn das Buch auf der Theke liegt
// und der Scan ins Leere geht. akteur 'SYSTEM' ohne bearbeiter_id, weil die Tür das
// Konto hier nicht durchreicht; wer gelöscht hat, steht in der Admin-Spur des
// Endpunkts, was dabei verschwand, steht hier.
func protokolliereOffeneAusleihen(ctx context.Context, tx pgx.Tx, offene []offeneAusleihe) error {
	for _, o := range offene {
		inhalt := map[string]any{
			"barcode_id":     o.Barcode,
			"titel":          o.Titel,
			"entleiher":      o.Entleiher,
			"ausgeliehen_am": o.Seit,
			"ausleihe_id":    o.AusleiheID,
			"action":         repository.ProtokollAktionTitelMitAusleihe,
		}
		// schueler_id ist nicht nur Information, sondern der Schlüssel, an dem die
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

// exemplarSnapshot ist das, was die Theken-Auskunft nach dem Löschen noch braucht:
// die Exemplar-ID als datensatz_id, Barcode und altes LMF-Etikett als Suchschlüssel, der Titel
// als Anzeige der Trefferzeile.
type exemplarSnapshot struct {
	ID           string
	Barcode      string
	LMFBarcode   string
	Titel        string
	WarImBestand bool
}

// leseExemplarSnapshots sammelt alle Exemplare der zu löschenden Titel, in der Transaktion der
// Löschung, bevor die Zeilen fallen. Nicht nur die verliehenen: Die Auskunft an der Theke
// (repository.FindeThekenExemplare) findet gelöschte Exemplare nur über Zeilen in audit_log mit
// tabelle='buecher_exemplare' und details->>'barcode_id'; die Spur der Ausleihen oben sieht sie
// nicht. Ohne den Schnappschuss wäre ein durch das Löschen eines Titels verschwundenes Buch beim
// Scannen „nie gesehen" statt „gelöscht am …".
func leseExemplarSnapshots(ctx context.Context, tx pgx.Tx, ids []string) ([]exemplarSnapshot, error) {
	zeilen, err := tx.Query(ctx, `
		SELECT e.id, e.barcode_id, coalesce(e.lmf_barcode, ''), t.titel, `+repository.SQLWarImBestand+`
		FROM buecher_exemplare e
		JOIN buecher_titel t ON e.titel_id = t.id
		WHERE e.titel_id = ANY($1::uuid[])
		ORDER BY e.barcode_id`, ids)
	if err != nil {
		return nil, fmt.Errorf("exemplar-snapshots konnten nicht gelesen werden: %w", err)
	}
	defer zeilen.Close()

	var schnappschuesse []exemplarSnapshot
	for zeilen.Next() {
		var s exemplarSnapshot
		if err := zeilen.Scan(&s.ID, &s.Barcode, &s.LMFBarcode, &s.Titel, &s.WarImBestand); err != nil {
			return nil, fmt.Errorf("exemplar-snapshots konnten nicht gelesen werden: %w", err)
		}
		schnappschuesse = append(schnappschuesse, s)
	}
	if err := zeilen.Err(); err != nil {
		return nil, fmt.Errorf("exemplar-snapshots konnten nicht gelesen werden: %w", err)
	}
	return schnappschuesse, nil
}

// protokolliereGeloeschteExemplare schreibt je Exemplar die Barcode-Spur — dasselbe
// Format wie SondereExemplarAus und das Verlust-Löschen (repository), damit die Auskunft
// alle drei Wege gleich findet. Läuft in der Transaktion der Löschung.
func protokolliereGeloeschteExemplare(ctx context.Context, tx pgx.Tx, schnappschuesse []exemplarSnapshot) error {
	for _, s := range schnappschuesse {
		spur := map[string]any{
			"barcode_id": s.Barcode,
			"titel":      s.Titel,
			"action":     repository.ProtokollAktionTitelGeloescht,

			repository.ProtokollDetailWarImBestand: s.WarImBestand,
		}
		repository.MerkeLMFBarcode(spur, s.LMFBarcode)
		details, err := json.Marshal(spur)
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

// offenerSchaden ist eine unbezahlte Forderung, die eine Titel-Löschung mit abräumt.
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

// loescheSchaedenMitSpur entfernt alle Schadensfälle der Titel und liefert die unbezahlten, nicht
// stornierten als Spur zurück, aus demselben Befehl wie bei den Ausleihen.
//
// Warum das eine eigene Spur braucht: Beide Wege der Löschung räumen `schadensfaelle` ohne
// Rücksicht auf `ist_bezahlt` ab. Ein unbezahlter Schadensfall ist aber Geld, das ein Schüler
// der Schule schuldet, und er steuert sechs Entscheidungen: Anzeige des Kontos, Sperre der
// Löschung, Zusammenführen, Wächter der Abgänger, Bremse der Anonymisierung beim Abgleich mit der
// LUSD und die Bedingung der Löschung nach DSGVO. Mit dem Titel verschwände die Forderung samt
// allen sechs Wirkungen, und niemand könnte es später sehen.
func loescheSchaedenMitSpur(ctx context.Context, tx pgx.Tx, ids []string) ([]offenerSchaden, error) {
	zeilen, err := tx.Query(ctx, `
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
	defer zeilen.Close()
	var alle []offenerSchaden
	for zeilen.Next() {
		var o offenerSchaden
		if err := zeilen.Scan(&o.ID, &o.ExemplarID, &o.Barcode, &o.Titel, &o.Schuldner,
			&o.SchuelerID, &o.Betrag, &o.Grund, &o.Seit); err != nil {
			return nil, fmt.Errorf("spur der offenen schadensfälle konnte nicht gelesen werden: %w", err)
		}
		alle = append(alle, o)
	}
	if err := zeilen.Err(); err != nil {
		return nil, fmt.Errorf("spur der offenen schadensfälle konnte nicht gelesen werden: %w", err)
	}
	return alle, nil
}

// protokolliereOffeneSchaeden hält jede offene Forderung fest, bevor sie mit dem Titel
// fällt: Wer schuldet wie viel, wofür, seit wann. `schueler_id` ist nicht nur Information,
// sondern der Schlüssel, an dem die Tilgung die Zeile findet — beim Anonymisieren fallen
// schuldner und beschreibung (repository.spurTilgungen). Anders als bei den offenen
// Ausleihen erreicht die Lesehistorie-Befristung diese Zeile nicht; sie arbeitet nur auf
// tabelle = 'ausleihen'.
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
			"action":          repository.ProtokollAktionTitelMitForderung,
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
