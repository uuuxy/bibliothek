package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"bibliothek/db"

	"github.com/jackc/pgx/v5"
)

// Der Schadensersatz-Bescheid: Nummer ziehen, Brief schreiben, Positionen zuordnen —
// alles in EINER Transaktion (Migration 110).
//
// Die Reihenfolge ist der Schutz: Erst wird die laufende Nummer gezogen (UPDATE …
// RETURNING sperrt die Zeile des Generators), dann der Bescheid geschrieben, dann die
// Forderungen zugeordnet. Bricht irgendetwas ab, ist auch die Nummer nicht verbraucht.
// Zwei gleichzeitige Briefe warten am Generator aufeinander, statt dieselbe Nummer zu
// bekommen — eine doppelte Referenznummer macht eine Zahlung unzuordenbar.

// BescheidPositionEingabe ist eine Forderung, die auf den Brief soll, mit dem Betrag,
// den die Schule festgesetzt hat.
type BescheidPositionEingabe struct {
	SchadensfallID string
	Betrag         float64
}

// BescheidEingabe ist alles, was der Aufrufer für einen neuen Bescheid mitbringt.
type BescheidEingabe struct {
	SchuelerID string
	Mittel     string
	Kassenjahr int
	FristBis   time.Time
	Positionen []BescheidPositionEingabe
	// Verluste: überfällige Bücher, die mit dem Brief als Verlust gebucht werden (Stufe 2,
	// bescheid_verlust.go). Ihre Forderungen entstehen in derselben Transaktion und
	// kommen zu den Positionen dazu.
	Verluste   []BescheidVerlustEingabe
	Empfaenger map[string]string
	// Absender: die Angaben der Schule zum Briefdatum (Migration 141) — Grundlage des
	// Nachdrucks wie Snapshot für den Empfänger.
	Absender    map[string]string
	ErstelltVon string
	// Referenznummer baut der Aufrufer aus der laufenden Nummer, die diese Schicht zieht
	// (BescheidAngaben.Referenznummer) — das Format ist eine Sache der Einstellungen,
	// nicht der Datenbank.
	Referenznummer func(laufendeNr int) string
}

// ForderungenNichtZugeordnet lehnt einen Bescheid ab, dessen Forderungen sich nicht alle
// zuordnen ließen: Eine ist bezahlt, storniert, steht schon auf einem Brief, gehört einem anderen
// Leser oder ist kein Buch des Landes. Die Lage hat sich geändert, seit der Dialog offen steht;
// die Tür antwortet mit 409 und diesem Satz.
type ForderungenNichtZugeordnet struct{ Fehlend, Verlangt int }

func (e *ForderungenNichtZugeordnet) Error() string {
	return fmt.Sprintf("%d von %d Forderungen konnten nicht zugeordnet werden — "+
		"gehören sie diesem Schüler, sind sie offen, noch auf keinem Bescheid und Bücher des Landes?",
		e.Fehlend, e.Verlangt)
}

// LegeAn schreibt den Bescheid und ordnet ihm seine Positionen zu.
//
// Geprüft wird IN der Transaktion, dass jede Position dem Schüler gehört, offen ist und
// noch auf keinem Bescheid steht — sonst stünde dieselbe Forderung auf zwei Briefen mit
// zwei Nummern. Verluste (Stufe 2) werden nach dem Brief gebucht, damit ihre Forderung
// die Referenznummer trägt, und vor den Positionen, damit dieselbe Prüfung sie einordnet.
func (a *pgBescheidAbfragen) LegeAn(ctx context.Context, e BescheidEingabe) (*Bescheid, error) {
	if len(e.Positionen) == 0 && len(e.Verluste) == 0 {
		return nil, fmt.Errorf("ein Bescheid ohne Positionen ist kein Bescheid")
	}
	tx, err := a.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer db.SafeRollback(ctx, tx)

	laufendeNr, err := zieheLaufendeNummer(ctx, tx, e.Mittel, e.Kassenjahr)
	if err != nil {
		return nil, err
	}

	snapshot, err := json.Marshal(e.Empfaenger)
	if err != nil {
		return nil, fmt.Errorf("empfänger-snapshot: %w", err)
	}
	// Ohne Absender bleibt die Spalte NULL (wie bei Bescheiden vor Migration 141), nicht
	// das JSON-Wort null.
	var absender *string
	if len(e.Absender) > 0 {
		roh, err := json.Marshal(e.Absender)
		if err != nil {
			return nil, fmt.Errorf("absender-snapshot: %w", err)
		}
		text := string(roh)
		absender = &text
	}

	var summe float64
	for _, p := range e.Positionen {
		summe += p.Betrag
	}
	for _, v := range e.Verluste {
		summe += v.Betrag
	}

	var b Bescheid
	err = tx.QueryRow(ctx, `
		INSERT INTO schadensersatz_bescheide
			(schueler_id, mittel, kassenjahr, laufende_nr, referenznummer, frist_bis,
			 gesamtbetrag, empfaenger_snapshot, erstellt_von, absender_snapshot)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8::jsonb, NULLIF($9, '')::uuid, $10::jsonb)
		RETURNING id, referenznummer, kassenjahr, laufende_nr, brief_datum, frist_bis,
		          gesamtbetrag, status`,
		e.SchuelerID, e.Mittel, e.Kassenjahr, laufendeNr, e.Referenznummer(laufendeNr),
		e.FristBis, summe, string(snapshot), e.ErstelltVon, absender,
	).Scan(&b.ID, &b.Referenznummer, &b.Kassenjahr, &b.LaufendeNr, &b.BriefDatum,
		&b.FristBis, &b.Gesamtbetrag, &b.Status)
	if err != nil {
		return nil, fmt.Errorf("bescheid schreiben: %w", err)
	}
	b.Mittel = e.Mittel
	b.SchuelerID = &e.SchuelerID

	neue, err := bucheVerluste(ctx, tx, e, b.Referenznummer)
	if err != nil {
		return nil, err
	}
	e.Positionen = append(e.Positionen, neue...)

	zugeordnet, err := ordnePositionenZu(ctx, tx, b.ID, e)
	if err != nil {
		return nil, err
	}
	if zugeordnet != len(e.Positionen) {
		return nil, &ForderungenNichtZugeordnet{Fehlend: len(e.Positionen) - zugeordnet, Verlangt: len(e.Positionen)}
	}
	b.AnzahlPositionen = zugeordnet

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return &b, nil
}

// zieheLaufendeNummer holt die nächste Nummer für Topf und Kassenjahr.
//
// Das INSERT … ON CONFLICT DO UPDATE ist beides in einem Schritt: die erste Nummer eines
// Jahres anlegen und jede weitere hochzählen. Die Zeile bleibt bis zum Ende der
// Transaktion gesperrt.
func zieheLaufendeNummer(ctx context.Context, tx pgx.Tx, mittel string, kassenjahr int) (int, error) {
	var nr int
	err := tx.QueryRow(ctx, `
		INSERT INTO schadensersatz_nummern (mittel, kassenjahr, letzte_nr)
		VALUES ($1, $2, 1)
		ON CONFLICT (mittel, kassenjahr)
		DO UPDATE SET letzte_nr = schadensersatz_nummern.letzte_nr + 1
		RETURNING letzte_nr`, mittel, kassenjahr).Scan(&nr)
	if err != nil {
		return 0, fmt.Errorf("laufende nummer ziehen: %w", err)
	}
	return nr, nil
}

// ordnePositionenZu hängt die Forderungen an den Brief und setzt ihren Betrag auf den
// festgesetzten Wert. Die WHERE-Bedingung ist die Prüfung: Sie lässt nur offene,
// unzugeordnete Forderungen DIESES Schülers durch — und nur solche aus dem Topf des
// Briefs: Bücher des Landes auf den Brief des Landes, alles andere nicht (ein Brief = ein
// Topf, Konzept 4.6). Der Topf ist das Eigentum des Exemplars nach ExemplarTopfSQL, dieselbe
// Regel wie am Etikett (seit dem 29.09.2026; vorher ist_lernmittel des Titels). Eine
// Forderung ohne Exemplar (Geräteschaden) gehört in keinen der beiden.
func ordnePositionenZu(ctx context.Context, tx pgx.Tx, bescheidID string, e BescheidEingabe) (int, error) {
	var zugeordnet int
	for _, p := range e.Positionen {
		tag, err := tx.Exec(ctx, `
			UPDATE schadensfaelle
			   SET bescheid_id = $1, betrag = $2, aktualisiert_am = CURRENT_TIMESTAMP
			 WHERE id = $3
			   AND schueler_id = $4
			   AND bescheid_id IS NULL
			   AND ist_bezahlt = false
			   AND storniert_am IS NULL
			   AND EXISTS (SELECT 1 FROM buecher_exemplare e JOIN buecher_titel t ON t.id = e.titel_id
			               `+ExemplarTopfJoin+`
			               WHERE e.id = schadensfaelle.exemplar_id AND `+ExemplarTopfSQL+` = $5)`,
			bescheidID, p.Betrag, p.SchadensfallID, e.SchuelerID, e.Mittel)
		if err != nil {
			return 0, fmt.Errorf("position %s zuordnen: %w", p.SchadensfallID, err)
		}
		zugeordnet += int(tag.RowsAffected())
	}
	return zugeordnet, nil
}
