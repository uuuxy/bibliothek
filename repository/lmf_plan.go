package repository

import (
	"context"
	"fmt"
	"strings"
	"time"

	"bibliothek/pkg/lmfplan"
	"bibliothek/pkg/schulzeit"

	"github.com/jackc/pgx/v5"
)

// Der LMF-Plan als Reihenfolge (Migration 097): Ein Plan je Art und Schuljahr trägt den
// Rahmen (erster Tag, Startstunde, Stunden je Tag), seine Zeilen stehen in lmf_termine
// mit Position — Datum und Stunde hat der Server aus dem Rahmen gerechnet
// (pkg/lmfplan) und schreibt sie mit, damit Portal, PDF und die Frist-Kopplung den Plan
// lesen wie bisher. Ausgelassene Klassen (die Oberstufe organisiert sich selbst) merkt
// sich der Plan, damit sie nicht als „ohne Termin" gelten und das nächste Jahr sie
// wieder auslässt.

// LmfPlan ist der Rahmen eines Plans.
type LmfPlan struct {
	ID              string `json:"id"`
	Art             string `json:"art"`
	SchuljahrBeginn string `json:"schuljahr_beginn"` // YYYY-MM-DD
	// ErsterTag/Startstunde: der Beginn. Beim Rückgabe-Plan GERECHNET aus dem Ende
	// (Migration 101), beim Ausgabe-Plan die Vorgabe des Planers.
	ErsterTag    string `json:"erster_tag"` // YYYY-MM-DD
	Startstunde  int    `json:"startstunde"`
	StundenJeTag int    `json:"stunden_je_tag"`
	// LetzterTag/LetzteStunde: der Anker des Rückgabe-Plans — Donnerstag vor den
	// Sommerferien, 4. Stunde (Peter, 06.09.2026); die Reihenfolge fließt rückwärts
	// davor. Beim Ausgabe-Plan leer ("" / 0): sein Anker ist der Beginn.
	LetzterTag   string `json:"letzter_tag"` // YYYY-MM-DD oder ""
	LetzteStunde int    `json:"letzte_stunde"`
	// VeroeffentlichtAm: nil = Entwurf (Migration 100) — nur im Planer sichtbar, keine
	// Fristen. Gesetzt (RFC 3339) = gilt für Portal, PDF und Frist-Kopplung.
	VeroeffentlichtAm *string `json:"veroeffentlicht_am"`
	// FreieTage: Tage, die dieser Plan überspringt (Migration 099) — bewegliche
	// Ferientage, pädagogische Tage, der Brückentag nach Fronleichnam. Gesetzliche
	// Feiertage stehen hier nicht, die kennt der Server (pkg/lmfplan).
	FreieTage []LmfFreierTag `json:"freie_tage"`
}

// LmfFreierTag ist ein freier Tag des Plans mit Grund.
type LmfFreierTag struct {
	Datum string `json:"datum"` // YYYY-MM-DD
	Grund string `json:"grund"`
}

// LmfPlanZeile ist eine Zeile der Reihenfolge — mit den vom Server gerechneten Plätzen.
// Fest: Datum und Stunde sind vorgegeben, nicht gerechnet (die Klasse mit dem Ausflug,
// Migration 099); die übrigen Zeilen fließen um diesen Platz herum.
type LmfPlanZeile struct {
	Position int      `json:"position"`
	Datum    string   `json:"datum"`
	Stunde   int      `json:"stunde"`
	Fest     bool     `json:"fest"`
	Klassen  []string `json:"klassen"`
	Vermerk  string   `json:"vermerk"`
}

// LmfPlanStand ist ein Plan mit allem, was dazugehört.
type LmfPlanStand struct {
	Plan        LmfPlan        `json:"plan"`
	Zeilen      []LmfPlanZeile `json:"zeilen"`
	Ausgelassen []string       `json:"ausgelassen"`
}

// lmfPlanSpalten ist die eine Spaltenliste des Rahmens — für Lesen, Speichern und
// Veröffentlichen dieselbe, damit kein Weg ein Feld vergisst (scanLmfPlan liest sie).
const lmfPlanSpalten = `id, art, to_char(schuljahr_beginn, 'YYYY-MM-DD'), to_char(erster_tag, 'YYYY-MM-DD'),
		       startstunde, stunden_je_tag, veroeffentlicht_am,
		       COALESCE(to_char(letzter_tag, 'YYYY-MM-DD'), ''), COALESCE(letzte_stunde, 0)`

// scanLmfPlan liest lmfPlanSpalten in den Rahmen; der Stempel kommt als RFC 3339 in der
// Schulzeitzone, weil die Oberfläche ihn nur anzeigt. Das Ende ist beim Ausgabe-Plan
// NULL — COALESCE, nicht Zeigertyp (NULL-Scan-Bugklasse).
func scanLmfPlan(row pgx.Row, p *LmfPlan) error {
	var veroeffentlicht *time.Time
	if err := row.Scan(&p.ID, &p.Art, &p.SchuljahrBeginn, &p.ErsterTag, &p.Startstunde, &p.StundenJeTag, &veroeffentlicht,
		&p.LetzterTag, &p.LetzteStunde); err != nil {
		return err
	}
	p.VeroeffentlichtAm = nil
	if veroeffentlicht != nil {
		s := veroeffentlicht.In(schulzeit.Zone()).Format(time.RFC3339)
		p.VeroeffentlichtAm = &s
	}
	return nil
}

// NeuesterLmfPlan liefert den Plan der Art mit dem spätesten ersten Tag — der, an dem
// gearbeitet wird oder der zuletzt galt, Entwurf oder veröffentlicht. pgx.ErrNoRows,
// wenn es noch keinen gibt.
func (r *LmfTerminRepository) NeuesterLmfPlan(ctx context.Context, art string) (LmfPlanStand, error) {
	var st LmfPlanStand
	err := scanLmfPlan(r.db.QueryRow(ctx, `
		SELECT `+lmfPlanSpalten+`
		FROM lmf_plaene WHERE art = $1
		ORDER BY erster_tag DESC LIMIT 1`, art), &st.Plan)
	if err != nil {
		return st, err
	}
	return r.ladeLmfPlanTeile(ctx, st)
}

// ladeLmfPlanTeile ergänzt einen gelesenen Rahmen um Zeilen, freie Tage und Auslassungen.
func (r *LmfTerminRepository) ladeLmfPlanTeile(ctx context.Context, st LmfPlanStand) (LmfPlanStand, error) {
	var err error
	if st.Zeilen, err = r.lmfPlanZeilen(ctx, st.Plan.ID); err != nil {
		return st, err
	}
	if st.Plan.FreieTage, err = r.lmfPlanFreieTage(ctx, st.Plan.ID); err != nil {
		return st, err
	}
	st.Ausgelassen, err = r.lmfPlanAusgelassen(ctx, st.Plan.ID)
	return st, err
}

func (r *LmfTerminRepository) lmfPlanFreieTage(ctx context.Context, planID string) ([]LmfFreierTag, error) {
	rows, err := r.db.Query(ctx, `
		SELECT to_char(datum, 'YYYY-MM-DD'), grund FROM lmf_plan_freie_tage
		WHERE plan_id = $1 ORDER BY datum`, planID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	tage := []LmfFreierTag{}
	for rows.Next() {
		var t LmfFreierTag
		if err := rows.Scan(&t.Datum, &t.Grund); err != nil {
			return nil, err
		}
		tage = append(tage, t)
	}
	return tage, rows.Err()
}

func (r *LmfTerminRepository) lmfPlanZeilen(ctx context.Context, planID string) ([]LmfPlanZeile, error) {
	rows, err := r.db.Query(ctx, `
		SELECT t.position, to_char(t.datum, 'YYYY-MM-DD'), t.stunde, t.fest, t.vermerk,
		       COALESCE((SELECT array_agg(k.klasse ORDER BY klassen_normkey(k.klasse))
		                 FROM lmf_termin_klassen k WHERE k.termin_id = t.id), '{}')
		FROM lmf_termine t WHERE t.plan_id = $1
		ORDER BY t.position, t.datum, t.stunde`, planID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	zeilen := []LmfPlanZeile{}
	for rows.Next() {
		var z LmfPlanZeile
		if err := rows.Scan(&z.Position, &z.Datum, &z.Stunde, &z.Fest, &z.Vermerk, &z.Klassen); err != nil {
			return nil, err
		}
		zeilen = append(zeilen, z)
	}
	return zeilen, rows.Err()
}

func (r *LmfTerminRepository) lmfPlanAusgelassen(ctx context.Context, planID string) ([]string, error) {
	rows, err := r.db.Query(ctx, `
		SELECT klasse FROM lmf_plan_ausgelassen WHERE plan_id = $1
		ORDER BY klassen_normkey(klasse)`, planID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	klassen := []string{}
	for rows.Next() {
		var k string
		if err := rows.Scan(&k); err != nil {
			return nil, err
		}
		klassen = append(klassen, k)
	}
	return klassen, rows.Err()
}

// SaveLmfPlanIn legt den Plan der Art für das Schuljahr des ersten Tages an oder schreibt
// ihn um — Rahmen (mit freien Tagen), Zeilen (vollständig ersetzt, mit Platz, Position
// und fest-Marke) und ausgelassene Klassen in einer Transaktion. Klassennamen laufen
// durch das Vokabular; die Antwort trägt die kanonisierten Namen. Der Veröffentlichungs-
// Stempel bleibt, wie er ist: Ein neuer Plan ist Entwurf, ein veröffentlichter bleibt
// veröffentlicht (Migration 100).
//
// Geschrieben wird in eine Transaktion, die der AUFRUFER hält — damit
// Plan und Frist-Kopplung in einer Klammer stehen (api/lmf_plan.go). Bis 07.09.2026
// hieß das SaveLmfPlan und committete selbst, und die Kopplung lief danach am Pool:
// Scheiterte sie, war der Plan geschrieben und die Fristen halb — ein Zustand, den ein
// zweiter Anlauf nicht mehr reparieren konnte, weil der alte Plan nicht mehr lesbar war.
// Eine Hülle mit eigener Transaktion gibt es bewusst nicht mehr: Sie rief niemand, und
// eine Tür, durch die niemand geht, ist eine, die irgendwann jemand falsch benutzt.
func (r *LmfTerminRepository) SaveLmfPlanIn(ctx context.Context, tx pgx.Tx, plan LmfPlan, zeilen []LmfPlanZeile, plaetze []lmfplan.Platz, ausgelassen []string) (LmfPlanStand, error) {
	// Je Zeile ein Platz — die Schleife unten greift mit plaetze[i] in die zweite
	// Scheibe. Bis zum 12.09.2026 sicherte das nur der eine Aufrufer zu (Register,
	// Paket 5: „vier Zusicherungen halten nur per Verabredung"); hier steht sie an der
	// Tür, die sie braucht, und VOR dem ersten Schreiben: Ein Missverhältnis wäre sonst
	// ein Indexfehler mitten in einer offenen Transaktion — halb geschriebener Plan,
	// keine lesbare Meldung.
	if len(plaetze) != len(zeilen) {
		return LmfPlanStand{}, fmt.Errorf("lmf-plan: %d Plätze für %d Zeilen", len(plaetze), len(zeilen))
	}
	ersterTag, err := time.ParseInLocation("2006-01-02", plan.ErsterTag, schulzeit.Zone())
	if err != nil {
		return LmfPlanStand{}, err
	}

	sjb := SchuljahrBeginn(ersterTag)
	var st LmfPlanStand
	// Das Ende (Anker des Rückgabe-Plans) als NULL, wenn leer — der Check der Tabelle
	// verlangt es beim Rückgabe-Plan und verbietet es beim Ausgabe-Plan.
	if err := scanLmfPlan(tx.QueryRow(ctx, `
		INSERT INTO lmf_plaene (art, schuljahr_beginn, erster_tag, startstunde, stunden_je_tag, letzter_tag, letzte_stunde)
		VALUES ($1, $2, $3, $4, $5, NULLIF($6, '')::date, NULLIF($7, 0))
		ON CONFLICT (art, schuljahr_beginn) DO UPDATE
		  SET erster_tag = EXCLUDED.erster_tag, startstunde = EXCLUDED.startstunde,
		      stunden_je_tag = EXCLUDED.stunden_je_tag,
		      letzter_tag = EXCLUDED.letzter_tag, letzte_stunde = EXCLUDED.letzte_stunde
		RETURNING `+lmfPlanSpalten,
		plan.Art, sjb, ersterTag, plan.Startstunde, plan.StundenJeTag, plan.LetzterTag, plan.LetzteStunde), &st.Plan); err != nil {
		return st, err
	}
	// Zeilen vollständig ersetzen: Der Plan IST die Reihenfolge, Einzel-IDs gibt es nicht.
	if _, err := tx.Exec(ctx, `DELETE FROM lmf_termine WHERE plan_id = $1`, st.Plan.ID); err != nil {
		return st, err
	}
	if _, err := tx.Exec(ctx, `DELETE FROM lmf_plan_ausgelassen WHERE plan_id = $1`, st.Plan.ID); err != nil {
		return st, err
	}
	if _, err := tx.Exec(ctx, `DELETE FROM lmf_plan_freie_tage WHERE plan_id = $1`, st.Plan.ID); err != nil {
		return st, err
	}
	st.Plan.FreieTage = []LmfFreierTag{}
	for _, f := range plan.FreieTage {
		var t LmfFreierTag
		if err := tx.QueryRow(ctx, `
			INSERT INTO lmf_plan_freie_tage (plan_id, datum, grund) VALUES ($1, $2::date, $3)
			ON CONFLICT (plan_id, datum) DO UPDATE SET grund = EXCLUDED.grund
			RETURNING to_char(datum, 'YYYY-MM-DD'), grund`, st.Plan.ID, f.Datum, f.Grund).Scan(&t.Datum, &t.Grund); err != nil {
			return st, err
		}
		st.Plan.FreieTage = append(st.Plan.FreieTage, t)
	}
	st.Zeilen = make([]LmfPlanZeile, 0, len(zeilen))
	for i, z := range zeilen {
		var id string
		if err := tx.QueryRow(ctx, `
			INSERT INTO lmf_termine (plan_id, position, datum, stunde, fest, art, vermerk)
			VALUES ($1, $2, $3, $4, $5, $6, $7) RETURNING id`,
			st.Plan.ID, i+1, plaetze[i].Datum, plaetze[i].Stunde, z.Fest, plan.Art, z.Vermerk).Scan(&id); err != nil {
			return st, err
		}
		kanonisch, err := schreibeKlassen(ctx, tx, `INSERT INTO lmf_termin_klassen (termin_id, klasse) VALUES ($1, $2)
			ON CONFLICT DO NOTHING RETURNING klasse`, id, z.Klassen)
		if err != nil {
			return st, err
		}
		st.Zeilen = append(st.Zeilen, LmfPlanZeile{Position: i + 1, Datum: plaetze[i].Datum.Format("2006-01-02"),
			Stunde: plaetze[i].Stunde, Fest: z.Fest, Klassen: kanonisch, Vermerk: z.Vermerk})
	}
	if st.Ausgelassen, err = schreibeKlassen(ctx, tx, `INSERT INTO lmf_plan_ausgelassen (plan_id, klasse) VALUES ($1, $2)
		ON CONFLICT DO NOTHING RETURNING klasse`, st.Plan.ID, ausgelassen); err != nil {
		return st, err
	}
	return st, nil
}

// schreibeKlassen fügt Klassen einer Elternzeile hinzu und liefert die vom Vokabular-
// Trigger kanonisierten Namen („5f1" → „05F1"); Leerwerte und Dubletten fallen weg.
func schreibeKlassen(ctx context.Context, tx pgx.Tx, sql, elternID string, klassen []string) ([]string, error) {
	kanonisch := make([]string, 0, len(klassen))
	for _, k := range klassen {
		if k = strings.TrimSpace(k); k == "" {
			continue
		}
		var name string
		err := tx.QueryRow(ctx, sql, elternID, k).Scan(&name)
		if err == pgx.ErrNoRows {
			continue
		}
		if err != nil {
			return nil, err
		}
		kanonisch = append(kanonisch, name)
	}
	return kanonisch, nil
}

// DeleteLmfPlanIn entfernt einen Plan samt Zeilen und Auslassungen (CASCADE) — auf einem
// Executor des Aufrufers (Transaktion des Handlers, im Test der Pool).
func (r *LmfTerminRepository) DeleteLmfPlanIn(ctx context.Context, ex DBQueryer, id string) (bool, error) {
	tag, err := ex.Exec(ctx, `DELETE FROM lmf_plaene WHERE id = $1`, id)
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() > 0, nil
}

// KlasseImPlan ist eine Klasse für den Vorschlag der Reihenfolge. Jahrgang ist die
// führende Zahl des Namens (99 ohne Ziffer) — der Ausgabe-Plan wählt danach die
// Eingangsjahrgänge aus.
type KlasseImPlan struct {
	Name      string
	Abschluss bool
	Oberstufe bool
	Jahrgang  int
}

// KlassenMitSchuelern nennt die Klassen aktiver Schüler in der Reihenfolge, in der die
// Schule sie abarbeitet: Abschlussklassen zuerst (AbschlussklasseSQL — die eine Regel
// für das Ende eines Bildungsgangs), dann Jahrgang absteigend, dann Name. Oberstufe
// (Jahrgang ab 11 oder ohne führende Ziffer, „E1", „Q3") ist markiert: Sie organisiert
// Rückgabe und Ausgabe an dieser Schule selbst und steht im ersten Plan unten.
func (r *LmfTerminRepository) KlassenMitSchuelern(ctx context.Context) ([]KlasseImPlan, error) {
	rows, err := r.db.Query(ctx, `
		SELECT s.klasse,
		       `+AbschlussklasseSQL("s.klasse")+` AS abschluss,
		       COALESCE(substring(s.klasse from '^\d+')::int, 99) AS jahrgang
		FROM schueler s
		WHERE s.deleted_at IS NULL AND s.ist_abgaenger = false AND btrim(s.klasse) <> ''
		GROUP BY s.klasse
		ORDER BY abschluss DESC, jahrgang DESC, s.klasse`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	klassen := []KlasseImPlan{}
	for rows.Next() {
		var k KlasseImPlan
		if err := rows.Scan(&k.Name, &k.Abschluss, &k.Jahrgang); err != nil {
			return nil, err
		}
		k.Oberstufe = k.Jahrgang >= 11
		klassen = append(klassen, k)
	}
	return klassen, rows.Err()
}
