package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"bibliothek/db"
	"bibliothek/pkg/schulzeit"

	"github.com/jackc/pgx/v5"
)

// Bescheid ist ein geschriebener Brief.
type Bescheid struct {
	ID             string     `json:"id"`
	SchuelerID     *string    `json:"schueler_id,omitempty"`
	SchuelerName   string     `json:"schueler_name,omitempty"`
	Klasse         string     `json:"klasse,omitempty"`
	Mittel         string     `json:"mittel"`
	Referenznummer string     `json:"referenznummer"`
	Kassenjahr     int        `json:"kassenjahr"`
	LaufendeNr     int        `json:"laufende_nr"`
	BriefDatum     time.Time  `json:"brief_datum"`
	FristBis       time.Time  `json:"frist_bis"`
	Gesamtbetrag   float64    `json:"gesamtbetrag"`
	Status         string     `json:"status"`
	UebergebenAm   *time.Time `json:"uebergeben_am,omitempty"`
	LetzterDruckAm *time.Time `json:"letzter_druck_am,omitempty"`
	// FristAbgelaufen heißt: offen, und die Frist ist vorbei. Diese Bescheide sind die
	// Arbeitsliste.
	FristAbgelaufen bool `json:"frist_abgelaufen"`
	// RueckgabeNachUebergabe merkt, dass ein Buch zurückkam, nachdem der Bescheid übergeben
	// war: Die Schulaufsicht ist dann zu informieren.
	RueckgabeNachUebergabe bool `json:"rueckgabe_nach_uebergabe"`
	// AnzahlPositionen zählt die Forderungen auf dem Brief.
	AnzahlPositionen int `json:"anzahl_positionen"`
}

// BescheidAbfragen liest und schreibt die Schadensersatz-Bescheide und liefert, was der Dialog
// für einen neuen Bescheid vorschlägt.
type BescheidAbfragen struct{ db db.Pool }

// BescheidBriefPosition ist eine Zeile des Briefs, gelesen für Druck und Nachdruck. Den Namen
// des Schülers trägt der Druck aus dem Schnappschuss des Empfängers ein (LadePositionen).
type BescheidBriefPosition struct {
	Art    string  `json:"art"`
	Titel  string  `json:"titel"`
	ISBN   string  `json:"isbn"`
	Betrag float64 `json:"betrag"`
}

// NewBescheidAbfragen liefert die Abfragen der Bescheide an Postgres.
func NewBescheidAbfragen(pool db.Pool) *BescheidAbfragen {
	return &BescheidAbfragen{db: pool}
}

// bescheidHatOffenePosition prüft, ob auf dem Brief b noch eine unbezahlte Forderung steht.
// Zahlung und Storno setzen beide ist_bezahlt (forderung_erledigen.go). Daraus ergibt sich
// „erledigt": Die Spalte status kennt den Wert, aber kein Schreibpfad setzt ihn, weil Zahlung
// und Storno über die Forderung laufen und nicht über den Brief.
const bescheidHatOffenePosition = `EXISTS (SELECT 1 FROM schadensfaelle fo WHERE fo.bescheid_id = b.id AND fo.ist_bezahlt = false)`

// sqlSchulHeute ist der heutige Kalendertag in der Zeitzone der Schule, in SQL gebildet.
// CURRENT_DATE rechnet in der Zeitzone der Sitzung (im Image UTC) und nennt bis 2 Uhr Berliner
// Zeit noch den Vortag.
const sqlSchulHeute = schulzeit.SQLHeute

// bescheidFristAbgelaufen heißt: offen, Frist vorbei, und es ist noch etwas zu zahlen. Die
// Frist ist ein Kalendertag in der Zeitzone der Schule (api/schaden/bescheid_erstellen.go) und
// wird deshalb gegen sqlSchulHeute verglichen: Mit CURRENT_DATE gälte eine Frist von gestern
// zwischen Mitternacht in Berlin und Mitternacht UTC als nicht abgelaufen
// (bescheid_frist_schulzeit_pg_test.go).
const bescheidFristAbgelaufen = `(b.status = 'offen' AND b.frist_bis < ` + sqlSchulHeute + ` AND ` + bescheidHatOffenePosition + `)`

// bescheidSpalten ist die gemeinsame Auswahl der Lesewege, damit Liste, Akte und Einzelabruf
// dieselben Felder in derselben Reihenfolge liefern.
const bescheidSpalten = `
	b.id, b.schueler_id, coalesce(s.vorname || ' ' || s.nachname, ''), coalesce(s.klasse, ''),
	b.mittel, b.referenznummer, b.kassenjahr, b.laufende_nr, b.brief_datum, b.frist_bis,
	b.gesamtbetrag,
	CASE WHEN b.status = 'offen' AND NOT ` + bescheidHatOffenePosition + ` THEN 'erledigt' ELSE b.status END,
	b.uebergeben_am, b.letzter_druck_am,
	` + bescheidFristAbgelaufen + `,
	b.rueckgabe_nach_uebergabe,
	(SELECT count(*) FROM schadensfaelle f WHERE f.bescheid_id = b.id)::int`

func leseBescheid(rows pgx.Rows) (Bescheid, error) {
	var b Bescheid
	err := rows.Scan(&b.ID, &b.SchuelerID, &b.SchuelerName, &b.Klasse, &b.Mittel,
		&b.Referenznummer, &b.Kassenjahr, &b.LaufendeNr, &b.BriefDatum, &b.FristBis,
		&b.Gesamtbetrag, &b.Status, &b.UebergebenAm, &b.LetzterDruckAm,
		&b.FristAbgelaufen, &b.RueckgabeNachUebergabe, &b.AnzahlPositionen)
	return b, err
}

// LadeListe liefert die Bescheide, die mit abgelaufener Frist zuerst, danach die neuesten;
// nurOffen beschränkt auf die, die weder übergeben noch erledigt sind.
func (a *BescheidAbfragen) LadeListe(ctx context.Context, nurOffen bool) ([]Bescheid, error) {
	bedingung := ""
	if nurOffen {
		bedingung = "WHERE b.status = 'offen' AND " + bescheidHatOffenePosition
	}
	// LIMIT, weil die Liste über Jahre wächst; die abgelaufenen Fristen stehen durch die
	// Sortierung oben.
	rows, err := a.db.Query(ctx, `
		SELECT `+bescheidSpalten+`
		FROM schadensersatz_bescheide b
		LEFT JOIN schueler s ON s.id = b.schueler_id
		`+bedingung+`
		ORDER BY `+bescheidFristAbgelaufen+` DESC, b.brief_datum DESC
		LIMIT 500`)
	if err != nil {
		return nil, err
	}
	return sammleBescheide(rows)
}

// LadeZuSchueler liefert die Bescheide eines Schülers für die Akte.
func (a *BescheidAbfragen) LadeZuSchueler(ctx context.Context, schuelerID string) ([]Bescheid, error) {
	rows, err := a.db.Query(ctx, `
		SELECT `+bescheidSpalten+`
		FROM schadensersatz_bescheide b
		LEFT JOIN schueler s ON s.id = b.schueler_id
		WHERE b.schueler_id = $1
		ORDER BY b.brief_datum DESC`, schuelerID)
	if err != nil {
		return nil, err
	}
	return sammleBescheide(rows)
}

func sammleBescheide(rows pgx.Rows) ([]Bescheid, error) {
	defer rows.Close()
	out := []Bescheid{}
	for rows.Next() {
		b, err := leseBescheid(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, b)
	}
	return out, rows.Err()
}

// Lade lädt einen Bescheid; pgx.ErrNoRows, wenn es ihn nicht gibt.
func (a *BescheidAbfragen) Lade(ctx context.Context, id string) (*Bescheid, error) {
	rows, err := a.db.Query(ctx, `
		SELECT `+bescheidSpalten+`
		FROM schadensersatz_bescheide b
		LEFT JOIN schueler s ON s.id = b.schueler_id
		WHERE b.id = $1`, id)
	if err != nil {
		return nil, err
	}
	alle, err := sammleBescheide(rows)
	if err != nil {
		return nil, err
	}
	if len(alle) == 0 {
		return nil, pgx.ErrNoRows
	}
	return &alle[0], nil
}

// LadeEmpfaengerSnapshot liest die Angaben des Empfängers zum Datum des Briefs, die Grundlage
// des Nachdrucks. Nach der Anonymisierung sind sie leer; der Nachdruck sagt dann, dass die
// Anschrift getilgt ist, statt eine falsche zu drucken.
func (a *BescheidAbfragen) LadeEmpfaengerSnapshot(ctx context.Context, id string) (map[string]string, error) {
	var roh []byte
	if err := a.db.QueryRow(ctx,
		`SELECT empfaenger_snapshot FROM schadensersatz_bescheide WHERE id = $1`, id).Scan(&roh); err != nil {
		return nil, err
	}
	out := map[string]string{}
	if err := json.Unmarshal(roh, &out); err != nil {
		return nil, fmt.Errorf("empfänger-snapshot lesen: %w", err)
	}
	return out, nil
}

// LadeAbsenderSnapshot liest die Angaben der Schule zum Datum des Briefs. Ältere Bescheide
// haben keine; das Ergebnis ist dann leer, und der Nachdruck nimmt die Einstellungen von heute.
func (a *BescheidAbfragen) LadeAbsenderSnapshot(ctx context.Context, id string) (map[string]string, error) {
	var roh []byte
	if err := a.db.QueryRow(ctx,
		`SELECT absender_snapshot FROM schadensersatz_bescheide WHERE id = $1`, id).Scan(&roh); err != nil {
		return nil, err
	}
	out := map[string]string{}
	if roh == nil {
		return out, nil
	}
	if err := json.Unmarshal(roh, &out); err != nil {
		return nil, fmt.Errorf("absender-snapshot lesen: %w", err)
	}
	return out, nil
}

// LadePositionen liest die Zeilen des Briefs. Den Namen des Schülers trägt der Rufer aus dem
// Snapshot des Empfängers ein und nicht aus dem heutigen Datensatz, damit der Nachdruck
// dasselbe Blatt ergibt.
func (a *BescheidAbfragen) LadePositionen(ctx context.Context, id string) ([]BescheidBriefPosition, error) {
	rows, err := a.db.Query(ctx, `
		SELECT f.art, coalesce(t.titel, f.beschreibung), coalesce(t.isbn, ''), f.betrag
		FROM schadensfaelle f
		LEFT JOIN buecher_exemplare e ON e.id = f.exemplar_id
		LEFT JOIN buecher_titel t ON t.id = e.titel_id
		WHERE f.bescheid_id = $1
		ORDER BY f.art, coalesce(t.titel, f.beschreibung) COLLATE deutsch`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []BescheidBriefPosition{}
	for rows.Next() {
		var p BescheidBriefPosition
		if err := rows.Scan(&p.Art, &p.Titel, &p.ISBN, &p.Betrag); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// VermerkeDruck hält fest, wann der Brief zuletzt gedruckt wurde. Nummer und Betrag bleiben:
// Ein Nachdruck ist derselbe Bescheid. Trifft die Anweisung keine Zeile, liefert sie
// pgx.ErrNoRows; der Rufer schickt den Brief trotzdem und protokolliert den Fehler, statt einen
// Erfolg auf einer ID zu melden, die es nicht gibt.
func (a *BescheidAbfragen) VermerkeDruck(ctx context.Context, id string) error {
	tag, err := a.db.Exec(ctx,
		`UPDATE schadensersatz_bescheide SET letzter_druck_am = NOW() WHERE id = $1`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

// Uebergebe setzt den Bescheid auf „übergeben": nur aus dem Zustand „offen", nur nach Ablauf
// der Frist und nur, solange noch etwas zu zahlen ist. Die Bedingung ist die Regel: Kein Brief
// geht vor Fristablauf weiter, eine zweite Übergabe überschreibt den Zeitpunkt nicht, und ein
// bezahlter Fall geht nicht zur Vollstreckung. pgx.ErrNoRows heißt, der Zustand passt nicht.
func (a *BescheidAbfragen) Uebergebe(ctx context.Context, id string) error {
	var gesetzt string
	err := a.db.QueryRow(ctx, `
		UPDATE schadensersatz_bescheide b
		   SET status = 'uebergeben', uebergeben_am = NOW()
		 WHERE b.id = $1 AND `+bescheidFristAbgelaufen+`
		RETURNING b.id`, id).Scan(&gesetzt)
	return err
}
