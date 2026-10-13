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
	// FristAbgelaufen: offen und die Frist ist vorbei — das ist die Arbeitsliste.
	FristAbgelaufen bool `json:"frist_abgelaufen"`
	// RueckgabeNachUebergabe: Merker für „die Aufsicht ist zu informieren".
	RueckgabeNachUebergabe bool `json:"rueckgabe_nach_uebergabe"`
	// AnzahlPositionen: wie viele Forderungen auf dem Brief stehen.
	AnzahlPositionen int `json:"anzahl_positionen"`
}

// BescheidAbfragen sind die Datenbankzugriffe des Bescheids.
type BescheidAbfragen interface {
	LegeAn(ctx context.Context, e BescheidEingabe) (*Bescheid, error)
	LadeListe(ctx context.Context, nurOffen bool) ([]Bescheid, error)
	LadeZuSchueler(ctx context.Context, schuelerID string) ([]Bescheid, error)
	Lade(ctx context.Context, id string) (*Bescheid, error)
	LadeEmpfaengerSnapshot(ctx context.Context, id string) (map[string]string, error)
	LadeAbsenderSnapshot(ctx context.Context, id string) (map[string]string, error)
	LadePositionen(ctx context.Context, id string) ([]BescheidBriefPosition, error)
	VermerkeDruck(ctx context.Context, id string) error
	Uebergebe(ctx context.Context, id string) error
	LadeEmpfaenger(ctx context.Context, schuelerID string) (BescheidEmpfaengerDaten, error)
	LadeOffeneForderungen(ctx context.Context, schuelerID string) ([]OffeneForderung, error)
	// LadeForderungenOhneBescheid: je Kind die offenen Forderungen, die noch auf keinem Brief stehen.
	LadeForderungenOhneBescheid(ctx context.Context) ([]ForderungOhneBescheid, error)
	// LadeUeberfaelligeAusleihen: die überfälligen Bücher eines Kindes ohne Forderung —
	// die Kandidaten für den Brief (Stufe 2).
	LadeUeberfaelligeAusleihen(ctx context.Context, schuelerID string) ([]UeberfaelligeAusleihe, error)
	// LadeGroessenFuerExemplar: die Zahlen des Staffel-Vorschlags für EIN Exemplar, für den
	// Dialog „Verlust/Schaden melden" — dort gibt es noch keine Forderung zu lesen.
	LadeGroessenFuerExemplar(ctx context.Context, exemplarID string) (ErsatzwertGroessen, error)

	// LadeGroessenFuerTitel: dieselben Zahlen für ALLE Exemplare eines Titels, in einer
	// Abfrage — die Buchakte zeigt den heutigen Buchwert an jeder Exemplar-Karte.
	LadeGroessenFuerTitel(ctx context.Context, titelID string) (map[string]ErsatzwertGroessen, error)
}

// BescheidBriefPosition ist eine Zeile des Briefs, gelesen für Druck und Nachdruck.
type BescheidBriefPosition struct {
	Art          string  `json:"art"`
	SchuelerName string  `json:"schueler_name"`
	Titel        string  `json:"titel"`
	ISBN         string  `json:"isbn"`
	Betrag       float64 `json:"betrag"`
}

type pgBescheidAbfragen struct{ db db.PgxPoolIface }

// NewBescheidAbfragen erstellt das Repository.
func NewBescheidAbfragen(pool db.PgxPoolIface) BescheidAbfragen {
	return &pgBescheidAbfragen{db: pool}
}

// bescheidHatOffenePosition: Steht auf dem Brief b noch eine unbezahlte Forderung?
// Bezahlt UND Storno setzen ist_bezahlt (forderung_erledigen.go) — beide erledigen die Position.
//
// Daraus leitet sich „erledigt" ab (Konzept 4.3): Die Spalte status kennt den Wert, aber
// kein Schreibpfad setzt ihn — Bezahlen und Storno laufen über die Forderung, nicht über
// den Brief. Bis zum 10.09.2026 fehlte die Ableitung ganz: Ein bezahlter Bescheid galt
// nach Fristablauf als überfällig und ließ sich an die Schulaufsicht übergeben.
const bescheidHatOffenePosition = `EXISTS (SELECT 1 FROM schadensfaelle fo WHERE fo.bescheid_id = b.id AND fo.ist_bezahlt = false)`

// sqlSchulHeute ist der heutige Kalendertag in der Schulzeitzone, in SQL gebildet.
// CURRENT_DATE rechnet in der Zeitzone der Sitzung (im Image UTC) und nennt bis 2 Uhr
// Berliner Zeit noch den Vortag.
const sqlSchulHeute = schulzeit.SQLHeute

// bescheidFristAbgelaufen: offen, Frist vorbei, und es ist noch etwas zu zahlen.
//
// Die Frist ist ein Kalendertag in der Schulzeitzone (api/schaden/bescheid_erstellen.go), also wird
// sie gegen sqlSchulHeute verglichen. Bis zum 16.09.2026 stand hier CURRENT_DATE: Zwischen
// Mitternacht in Berlin und Mitternacht UTC galt eine Frist von gestern als nicht
// abgelaufen, die Übergabe an die Schulaufsicht fand keine Zeile
// (bescheid_frist_schulzeit_pg_test.go).
const bescheidFristAbgelaufen = `(b.status = 'offen' AND b.frist_bis < ` + sqlSchulHeute + ` AND ` + bescheidHatOffenePosition + `)`

// bescheidSpalten ist die gemeinsame Auswahl der Leser — EIN Ort, damit Liste, Akte und
// Einzelabruf dieselben Felder in derselben Reihenfolge liefern.
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

// LadeListe liefert die Bescheide, die neuesten zuerst; nurOffen beschränkt auf die, die
// noch nicht übergeben oder erledigt sind.
func (a *pgBescheidAbfragen) LadeListe(ctx context.Context, nurOffen bool) ([]Bescheid, error) {
	bedingung := ""
	if nurOffen {
		bedingung = "WHERE b.status = 'offen' AND " + bescheidHatOffenePosition
	}
	// LIMIT, weil auch diese Liste über Jahre wächst (Register „Unbegrenzte
	// Listen-Endpunkte"): die abgelaufenen Fristen stehen dank ORDER BY oben.
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
func (a *pgBescheidAbfragen) LadeZuSchueler(ctx context.Context, schuelerID string) ([]Bescheid, error) {
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

// Lade holt einen Bescheid; pgx.ErrNoRows, wenn es ihn nicht gibt.
func (a *pgBescheidAbfragen) Lade(ctx context.Context, id string) (*Bescheid, error) {
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

// LadeEmpfaengerSnapshot liest die Empfänger-Angaben zum Briefdatum — die Grundlage des Nachdrucks.
// Nach der Anonymisierung ist er leer; der Nachdruck sagt dann, dass die Anschrift
// getilgt ist, statt eine falsche zu drucken.
func (a *pgBescheidAbfragen) LadeEmpfaengerSnapshot(ctx context.Context, id string) (map[string]string, error) {
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

// LadeAbsenderSnapshot liest die Angaben der Schule zum Briefdatum (Migration 141). Bescheide
// von vorher haben keinen; dann ist die Karte leer, und der Nachdruck nimmt die
// Einstellungen von heute.
func (a *pgBescheidAbfragen) LadeAbsenderSnapshot(ctx context.Context, id string) (map[string]string, error) {
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

// LadePositionen liest die Zeilen des Briefs. Der Name kommt aus dem Titelsatz bzw. dem
// Snapshot des Schülers — nicht aus dem lebenden Schülerdatensatz, damit der Nachdruck
// dasselbe Blatt ergibt.
func (a *pgBescheidAbfragen) LadePositionen(ctx context.Context, id string) ([]BescheidBriefPosition, error) {
	rows, err := a.db.Query(ctx, `
		SELECT f.art, coalesce(t.titel, f.beschreibung), coalesce(t.isbn, ''), f.betrag
		FROM schadensfaelle f
		LEFT JOIN buecher_exemplare e ON e.id = f.exemplar_id
		LEFT JOIN buecher_titel t ON t.id = e.titel_id
		WHERE f.bescheid_id = $1
		ORDER BY f.art, coalesce(t.titel, f.beschreibung)`, id)
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

// VermerkeDruck hält fest, wann der Brief zuletzt gedruckt wurde. Die Nummer und der
// Betrag bleiben unberührt — ein Nachdruck ist derselbe Bescheid, kein neuer.
//
// Trifft das UPDATE keine Zeile, ist das ein Fehler und kein Erfolg: pgx.ErrNoRows. Der
// Aufrufer schickt den Brief trotzdem (der Vermerk ist Buchführung, nicht der Zweck),
// protokolliert es aber — ein stiller Erfolg auf einer ID, die es nicht gibt, wäre die
// Bugklasse „Phantom-Erfolg".
func (a *pgBescheidAbfragen) VermerkeDruck(ctx context.Context, id string) error {
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

// Uebergebe setzt den Bescheid auf „übergeben" — nur aus dem Zustand „offen" und nur,
// wenn die Frist vorbei ist. Die WHERE-Bedingung ist die Regel: Sie verhindert, dass ein
// Brief vor Fristablauf weitergegeben wird, und dass eine zweite Übergabe den Zeitpunkt
// überschreibt — und nur, solange noch etwas zu zahlen ist: Ein bezahlter Fall geht
// nicht zur Vollstreckung. pgx.ErrNoRows heißt: Der Zustand passt nicht.
func (a *pgBescheidAbfragen) Uebergebe(ctx context.Context, id string) error {
	var gesetzt string
	err := a.db.QueryRow(ctx, `
		UPDATE schadensersatz_bescheide b
		   SET status = 'uebergeben', uebergeben_am = NOW()
		 WHERE b.id = $1 AND `+bescheidFristAbgelaufen+`
		RETURNING b.id`, id).Scan(&gesetzt)
	return err
}

// ── Der Vorschlag für einen neuen Bescheid ───────────────────────────────────
//
// Die drei Lesepfade des Dialogs liegen HIER und nicht im Handler: Das Schichtungs-Gate
// verlangt es, und die Regel dahinter ist die richtige — die Bedingung „offen, nicht
// storniert, noch auf keinem Brief" steht damit an EINER Stelle, dieselbe, die beim
// Erstellen zuordnet (ordnePositionenZu).
