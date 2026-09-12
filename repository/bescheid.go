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
	SchuelerID  string
	Mittel      string
	Kassenjahr  int
	FristBis    time.Time
	Positionen  []BescheidPositionEingabe
	Snapshot    map[string]string
	ErstelltVon string
	// Referenznummer baut der Aufrufer aus der laufenden Nummer, die diese Schicht zieht
	// (BescheidAngaben.Referenznummer) — das Format ist eine Sache der Einstellungen,
	// nicht der Datenbank.
	Referenznummer func(laufendeNr int) string
}

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

// BescheidRepository sind die Datenbankzugriffe des Bescheids.
type BescheidRepository interface {
	Erstelle(ctx context.Context, e BescheidEingabe) (*Bescheid, error)
	Liste(ctx context.Context, nurOffen bool) ([]Bescheid, error)
	ZuSchueler(ctx context.Context, schuelerID string) ([]Bescheid, error)
	Lies(ctx context.Context, id string) (*Bescheid, error)
	Snapshot(ctx context.Context, id string) (map[string]string, error)
	Positionen(ctx context.Context, id string) ([]BescheidBriefPosition, error)
	DruckVermerken(ctx context.Context, id string) error
	Uebergebe(ctx context.Context, id string) error
	EmpfaengerFuerBescheid(ctx context.Context, schuelerID string) (BescheidEmpfaengerDaten, error)
	OffeneForderungen(ctx context.Context, schuelerID string) ([]OffeneForderung, error)
}

// BescheidBriefPosition ist eine Zeile des Briefs, gelesen für Druck und Nachdruck.
type BescheidBriefPosition struct {
	Art          string  `json:"art"`
	SchuelerName string  `json:"schueler_name"`
	Titel        string  `json:"titel"`
	ISBN         string  `json:"isbn"`
	Betrag       float64 `json:"betrag"`
}

type pgBescheidRepository struct{ db db.PgxPoolIface }

// NewBescheidRepository erstellt das Repository.
func NewBescheidRepository(pool db.PgxPoolIface) BescheidRepository {
	return &pgBescheidRepository{db: pool}
}

// Erstelle schreibt den Bescheid und ordnet ihm seine Positionen zu.
//
// Geprüft wird IN der Transaktion, dass jede Position dem Schüler gehört, offen ist und
// noch auf keinem Bescheid steht — sonst stünde dieselbe Forderung auf zwei Briefen mit
// zwei Nummern.
func (r *pgBescheidRepository) Erstelle(ctx context.Context, e BescheidEingabe) (*Bescheid, error) {
	if len(e.Positionen) == 0 {
		return nil, fmt.Errorf("ein Bescheid ohne Positionen ist kein Bescheid")
	}
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer db.SafeRollback(ctx, tx)

	laufendeNr, err := ziehLaufendeNummer(ctx, tx, e.Mittel, e.Kassenjahr)
	if err != nil {
		return nil, err
	}

	snapshot, err := json.Marshal(e.Snapshot)
	if err != nil {
		return nil, fmt.Errorf("empfänger-snapshot: %w", err)
	}

	var summe float64
	for _, p := range e.Positionen {
		summe += p.Betrag
	}

	var b Bescheid
	err = tx.QueryRow(ctx, `
		INSERT INTO schadensersatz_bescheide
			(schueler_id, mittel, kassenjahr, laufende_nr, referenznummer, frist_bis,
			 gesamtbetrag, empfaenger_snapshot, erstellt_von)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8::jsonb, NULLIF($9, '')::uuid)
		RETURNING id, referenznummer, kassenjahr, laufende_nr, brief_datum, frist_bis,
		          gesamtbetrag, status`,
		e.SchuelerID, e.Mittel, e.Kassenjahr, laufendeNr, e.Referenznummer(laufendeNr),
		e.FristBis, summe, string(snapshot), e.ErstelltVon,
	).Scan(&b.ID, &b.Referenznummer, &b.Kassenjahr, &b.LaufendeNr, &b.BriefDatum,
		&b.FristBis, &b.Gesamtbetrag, &b.Status)
	if err != nil {
		return nil, fmt.Errorf("bescheid schreiben: %w", err)
	}
	b.Mittel = e.Mittel
	b.SchuelerID = &e.SchuelerID

	zugeordnet, err := ordnePositionenZu(ctx, tx, b.ID, e)
	if err != nil {
		return nil, err
	}
	if zugeordnet != len(e.Positionen) {
		return nil, fmt.Errorf("%d von %d Forderungen konnten nicht zugeordnet werden — "+
			"gehören sie diesem Schüler, sind sie offen, noch auf keinem Bescheid und Lernmittel?",
			len(e.Positionen)-zugeordnet, len(e.Positionen))
	}
	b.AnzahlPositionen = zugeordnet

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return &b, nil
}

// ziehLaufendeNummer holt die nächste Nummer für Topf und Kassenjahr.
//
// Das INSERT … ON CONFLICT DO UPDATE ist beides in einem Schritt: die erste Nummer eines
// Jahres anlegen und jede weitere hochzählen. Die Zeile bleibt bis zum Ende der
// Transaktion gesperrt.
func ziehLaufendeNummer(ctx context.Context, tx pgx.Tx, mittel string, kassenjahr int) (int, error) {
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
// Briefs: Lernmittel auf den Brief des Landes, alles andere nicht (ein Brief = ein Topf,
// Konzept 4.6). Eine Forderung ohne Exemplar (Geräteschaden) gehört in keinen der beiden.
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
			   AND EXISTS (SELECT 1 FROM buecher_exemplare ex JOIN buecher_titel t ON t.id = ex.titel_id
			               WHERE ex.id = schadensfaelle.exemplar_id AND t.ist_lernmittel = ($5 = 'land'))`,
			bescheidID, p.Betrag, p.SchadensfallID, e.SchuelerID, e.Mittel)
		if err != nil {
			return 0, fmt.Errorf("position %s zuordnen: %w", p.SchadensfallID, err)
		}
		zugeordnet += int(tag.RowsAffected())
	}
	return zugeordnet, nil
}

// bescheidHatOffenePosition: Steht auf dem Brief b noch eine unbezahlte Forderung?
// Bezahlt UND Storno setzen ist_bezahlt (audit_system.go) — beide erledigen die Position.
//
// Daraus leitet sich „erledigt" ab (Konzept 4.3): Die Spalte status kennt den Wert, aber
// kein Schreibpfad setzt ihn — Bezahlen und Storno laufen über die Forderung, nicht über
// den Brief. Bis zum 10.09.2026 fehlte die Ableitung ganz: Ein bezahlter Bescheid galt
// nach Fristablauf als überfällig und ließ sich an die Schulaufsicht übergeben.
const bescheidHatOffenePosition = `EXISTS (SELECT 1 FROM schadensfaelle fo WHERE fo.bescheid_id = b.id AND fo.ist_bezahlt = false)`

// bescheidFristAbgelaufen: offen, Frist vorbei, und es ist noch etwas zu zahlen.
const bescheidFristAbgelaufen = `(b.status = 'offen' AND b.frist_bis < CURRENT_DATE AND ` + bescheidHatOffenePosition + `)`

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

func scanBescheid(rows pgx.Rows) (Bescheid, error) {
	var b Bescheid
	err := rows.Scan(&b.ID, &b.SchuelerID, &b.SchuelerName, &b.Klasse, &b.Mittel,
		&b.Referenznummer, &b.Kassenjahr, &b.LaufendeNr, &b.BriefDatum, &b.FristBis,
		&b.Gesamtbetrag, &b.Status, &b.UebergebenAm, &b.LetzterDruckAm,
		&b.FristAbgelaufen, &b.RueckgabeNachUebergabe, &b.AnzahlPositionen)
	return b, err
}

// Liste liefert die Bescheide, die neuesten zuerst; nurOffen beschränkt auf die, die
// noch nicht übergeben oder erledigt sind.
func (r *pgBescheidRepository) Liste(ctx context.Context, nurOffen bool) ([]Bescheid, error) {
	bedingung := ""
	if nurOffen {
		bedingung = "WHERE b.status = 'offen' AND " + bescheidHatOffenePosition
	}
	// LIMIT, weil auch diese Liste über Jahre wächst (Register „Unbegrenzte
	// Listen-Endpunkte"): die abgelaufenen Fristen stehen dank ORDER BY oben.
	rows, err := r.db.Query(ctx, `
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

// ZuSchueler liefert die Bescheide eines Schülers für die Akte.
func (r *pgBescheidRepository) ZuSchueler(ctx context.Context, schuelerID string) ([]Bescheid, error) {
	rows, err := r.db.Query(ctx, `
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
		b, err := scanBescheid(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, b)
	}
	return out, rows.Err()
}

// Lies holt einen Bescheid; pgx.ErrNoRows, wenn es ihn nicht gibt.
func (r *pgBescheidRepository) Lies(ctx context.Context, id string) (*Bescheid, error) {
	rows, err := r.db.Query(ctx, `
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

// Snapshot liest die Empfänger-Angaben zum Briefdatum — die Grundlage des Nachdrucks.
// Nach der Anonymisierung ist er leer; der Nachdruck sagt dann, dass die Anschrift
// getilgt ist, statt eine falsche zu drucken.
func (r *pgBescheidRepository) Snapshot(ctx context.Context, id string) (map[string]string, error) {
	var roh []byte
	if err := r.db.QueryRow(ctx,
		`SELECT empfaenger_snapshot FROM schadensersatz_bescheide WHERE id = $1`, id).Scan(&roh); err != nil {
		return nil, err
	}
	out := map[string]string{}
	if err := json.Unmarshal(roh, &out); err != nil {
		return nil, fmt.Errorf("empfänger-snapshot lesen: %w", err)
	}
	return out, nil
}

// Positionen liest die Zeilen des Briefs. Der Name kommt aus dem Titelsatz bzw. dem
// Snapshot des Schülers — nicht aus dem lebenden Schülerdatensatz, damit der Nachdruck
// dasselbe Blatt ergibt.
func (r *pgBescheidRepository) Positionen(ctx context.Context, id string) ([]BescheidBriefPosition, error) {
	rows, err := r.db.Query(ctx, `
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

// DruckVermerken hält fest, wann der Brief zuletzt gedruckt wurde. Die Nummer und der
// Betrag bleiben unberührt — ein Nachdruck ist derselbe Bescheid, kein neuer.
//
// Trifft das UPDATE keine Zeile, ist das ein Fehler und kein Erfolg: pgx.ErrNoRows. Der
// Aufrufer schickt den Brief trotzdem (der Vermerk ist Buchführung, nicht der Zweck),
// protokolliert es aber — ein stiller Erfolg auf einer ID, die es nicht gibt, wäre die
// Bugklasse „Phantom-Erfolg".
func (r *pgBescheidRepository) DruckVermerken(ctx context.Context, id string) error {
	tag, err := r.db.Exec(ctx,
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
func (r *pgBescheidRepository) Uebergebe(ctx context.Context, id string) error {
	var gesetzt string
	err := r.db.QueryRow(ctx, `
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

// BescheidEmpfaengerDaten sind die Angaben, aus denen der Snapshot entsteht.
type BescheidEmpfaengerDaten struct {
	Vorname     string
	Nachname    string
	Klasse      string
	Strasse     string
	Hausnummer  string
	PLZ         string
	Ort         string
	Volljaehrig bool
}

// OffeneForderung ist eine Forderung, die auf einen Brief könnte — mit den Größen, aus
// denen der Betragsvorschlag entsteht.
type OffeneForderung struct {
	SchadensfallID string
	Art            string
	Titel          string
	ISBN           string
	Kaufpreis      float64
	IstLernmittel  bool
	// Die beiden Größen für das Verleihjahr, und beide zählen SCHULJAHRE: in wie vielen
	// war das Exemplar ausgeliehen, und wie viele liegen seit seiner Beschaffung. Beide
	// sind unvollständig (der Altbestand kam ohne Ausleihhistorie), deshalb rechnet
	// ersatzwert.Verleihjahr mit dem Maximum.
	//
	// Bis zum 12.09.2026 standen hier die Zahl der AUSLEIHEN und die Differenz der
	// KALENDERJAHRE. Sechs Ausleihen in einem Schuljahr ergaben das 6. Verleihjahr (10 %
	// statt 100 %), und ein im Februar gekauftes Buch blieb bis Silvester im ersten.
	SchuljahreMitAusleihe int
	SchuljahreImBestand   int
}

// EmpfaengerFuerBescheid liest die Angaben des Schülers für Anrede und Anschriftfeld.
//
// Volljährig entscheidet über die Anrede: Bei minderjährigen Schülern geht der Brief an
// die Erziehungsberechtigten. Ohne Geburtsdatum (Altdaten) gilt minderjährig — das ist
// an einer Schule der Regelfall und der schonendere Fehler.
func (r *pgBescheidRepository) EmpfaengerFuerBescheid(ctx context.Context, schuelerID string) (BescheidEmpfaengerDaten, error) {
	var d BescheidEmpfaengerDaten
	err := r.db.QueryRow(ctx, `
		SELECT vorname, nachname, coalesce(klasse, ''), coalesce(strasse, ''),
		       coalesce(hausnummer, ''), coalesce(plz, ''), coalesce(ort, ''),
		       coalesce(geburtsdatum <= CURRENT_DATE - INTERVAL '18 years', false)
		FROM schueler WHERE id = $1`, schuelerID).
		Scan(&d.Vorname, &d.Nachname, &d.Klasse, &d.Strasse, &d.Hausnummer, &d.PLZ, &d.Ort, &d.Volljaehrig)
	return d, err
}

// OffeneForderungen liest die Forderungen, die noch auf keinem Brief stehen.
func (r *pgBescheidRepository) OffeneForderungen(ctx context.Context, schuelerID string) ([]OffeneForderung, error) {
	// Das SQL liefert Datumswerte, keine Schuljahre: Wann das Schuljahr wechselt, steht
	// in SchuljahrBeginn und soll nicht ein zweites Mal in einer Abfrage stehen — eine
	// abweichende Auslegung verschöbe hier jeden Betrag um eine Stufe der Staffel.
	rows, err := r.db.Query(ctx, `
		SELECT f.id, f.art, coalesce(t.titel, f.beschreibung), coalesce(t.isbn, ''),
		       coalesce(e.einkaufspreis, 0)::float8,
		       coalesce(t.ist_lernmittel, false),
		       f.exemplar_id, e.erworben_am
		FROM schadensfaelle f
		LEFT JOIN buecher_exemplare e ON e.id = f.exemplar_id
		LEFT JOIN buecher_titel t ON t.id = e.titel_id
		WHERE f.schueler_id = $1
		  AND f.bescheid_id IS NULL
		  AND f.ist_bezahlt = false
		  AND f.storniert_am IS NULL
		ORDER BY f.art, coalesce(t.titel, f.beschreibung)`, schuelerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	heute := schuljahrVon(schulzeit.Jetzt())
	out := []OffeneForderung{}
	// exemplarJeForderung ist indexgleich zu out: Geräteschäden haben kein Exemplar und
	// stehen deshalb mit leerem String drin.
	exemplarJeForderung := []string{}
	for rows.Next() {
		var f OffeneForderung
		var exemplarID *string
		var erworben *time.Time
		if err := rows.Scan(&f.SchadensfallID, &f.Art, &f.Titel, &f.ISBN, &f.Kaufpreis,
			&f.IstLernmittel, &exemplarID, &erworben); err != nil {
			return nil, err
		}
		if erworben != nil {
			f.SchuljahreImBestand = heute - schuljahrVon(*erworben)
		}
		out = append(out, f)
		exemplarJeForderung = append(exemplarJeForderung, zeigerText(exemplarID))
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	schuljahre, err := r.schuljahreMitAusleihe(ctx, exemplarJeForderung)
	if err != nil {
		return nil, err
	}
	for i := range out {
		out[i].SchuljahreMitAusleihe = schuljahre[exemplarJeForderung[i]]
	}
	return out, nil
}

// schuljahreMitAusleihe zählt je Exemplar, in wie vielen SCHULJAHREN es ausgeliehen war.
//
// Die Zeitpunkte kommen roh aus der Datenbank, gezählt wird in Go — mit SchuljahrBeginn,
// derselben Grenze wie überall sonst. Ein „count(DISTINCT …)" im SQL müsste den 1. August
// ein zweites Mal kennen, und zwei Auslegungen derselben Grenze verschöben den Betrag im
// Bescheid um eine ganze Stufe der Staffel.
func (r *pgBescheidRepository) schuljahreMitAusleihe(ctx context.Context, exemplarIDs []string) (map[string]int, error) {
	je := map[string]int{}
	gefragt := []string{}
	gesehen := map[string]bool{}
	for _, id := range exemplarIDs {
		if id == "" || gesehen[id] {
			continue
		}
		gesehen[id] = true
		gefragt = append(gefragt, id)
	}
	if len(gefragt) == 0 {
		return je, nil
	}

	rows, err := r.db.Query(ctx,
		`SELECT exemplar_id, ausgeliehen_am FROM ausleihen WHERE exemplar_id = ANY($1)`, gefragt)
	if err != nil {
		return nil, fmt.Errorf("ausleihzeitpunkte des exemplars lesen: %w", err)
	}
	defer rows.Close()

	jahreJeExemplar := map[string]map[int]bool{}
	for rows.Next() {
		var exemplarID string
		var ausgeliehenAm time.Time
		if err := rows.Scan(&exemplarID, &ausgeliehenAm); err != nil {
			return nil, err
		}
		if jahreJeExemplar[exemplarID] == nil {
			jahreJeExemplar[exemplarID] = map[int]bool{}
		}
		jahreJeExemplar[exemplarID][schuljahrVon(ausgeliehenAm)] = true
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	for id, jahre := range jahreJeExemplar {
		je[id] = len(jahre)
	}
	return je, nil
}

// zeigerText macht aus einem nullbaren Textfeld einen String; NULL wird zu "".
func zeigerText(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
