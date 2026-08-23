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

	"github.com/jackc/pgx/v5"
)

// offeneAusleihe ist eine laufende Ausleihe, die eine Titel-Löschung mit abräumt.
type offeneAusleihe struct {
	AusleiheID string
	ExemplarID string
	Barcode    string
	Titel      string
	Entleiher  string
	Seit       string
}

// leseOffeneAusleihen sammelt die laufenden Ausleihen der zu löschenden Titel — VOR der
// Löschung, denn danach sind sie nicht mehr da.
//
// Schüler UND Kollegium (Handapparat): Beide Entleiher-Spalten sind polymorph, und ein
// Handapparat-Buch ist genauso weg wie ein Schülerbuch. „(unbekannt)" statt eines
// leeren Feldes, damit die Protokollzeile auch dann etwas aussagt, wenn die Zuordnung
// bereits von der Lesehistorie-Befristung getrennt wurde.
func (repo *BookRepository) leseOffeneAusleihen(ctx context.Context, ids []string) ([]offeneAusleihe, error) {
	rows, err := repo.db.Query(ctx, `
		SELECT a.id, e.id, e.barcode_id, t.titel,
		       coalesce(nullif(trim(coalesce(s.vorname,'') || ' ' || coalesce(s.nachname,'')), ''),
		                nullif(trim(coalesce(b.vorname,'') || ' ' || coalesce(b.nachname,'')), ''),
		                '(unbekannt)'),
		       to_char(a.ausgeliehen_am, 'YYYY-MM-DD')
		FROM ausleihen a
		JOIN buecher_exemplare e ON a.exemplar_id = e.id
		JOIN buecher_titel t     ON e.titel_id = t.id
		LEFT JOIN schueler s     ON a.schueler_id = s.id
		LEFT JOIN benutzer b     ON a.ausleiher_benutzer_id = b.id
		WHERE t.id = ANY($1::uuid[]) AND a.rueckgabe_am IS NULL
		ORDER BY e.barcode_id`, ids)
	if err != nil {
		return nil, fmt.Errorf("laufende ausleihen konnten nicht gelesen werden: %w", err)
	}
	defer rows.Close()

	var offene []offeneAusleihe
	for rows.Next() {
		var o offeneAusleihe
		if err := rows.Scan(&o.AusleiheID, &o.ExemplarID, &o.Barcode, &o.Titel, &o.Entleiher, &o.Seit); err != nil {
			return nil, fmt.Errorf("laufende ausleihen konnten nicht gelesen werden: %w", err)
		}
		offene = append(offene, o)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("laufende ausleihen konnten nicht gelesen werden: %w", err)
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
		details, err := json.Marshal(map[string]any{
			"barcode_id":     o.Barcode,
			"titel":          o.Titel,
			"entleiher":      o.Entleiher,
			"ausgeliehen_am": o.Seit,
			"ausleihe_id":    o.AusleiheID,
			"action":         "titel_geloescht_mit_offener_ausleihe",
		})
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
