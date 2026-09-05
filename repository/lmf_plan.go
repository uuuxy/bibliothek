package repository

import (
	"context"
	"strings"
	"time"

	"bibliothek/db"
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
	ErsterTag       string `json:"erster_tag"`       // YYYY-MM-DD
	Startstunde     int    `json:"startstunde"`
	StundenJeTag    int    `json:"stunden_je_tag"`
}

// LmfPlanZeile ist eine Zeile der Reihenfolge — mit den vom Server gerechneten Plätzen.
type LmfPlanZeile struct {
	Position int      `json:"position"`
	Datum    string   `json:"datum"`
	Stunde   int      `json:"stunde"`
	Klassen  []string `json:"klassen"`
	Vermerk  string   `json:"vermerk"`
}

// LmfPlanStand ist ein Plan mit allem, was dazugehört.
type LmfPlanStand struct {
	Plan        LmfPlan        `json:"plan"`
	Zeilen      []LmfPlanZeile `json:"zeilen"`
	Ausgelassen []string       `json:"ausgelassen"`
}

// NeuesterLmfPlan liefert den Plan der Art mit dem spätesten ersten Tag — der, an dem
// gearbeitet wird oder der zuletzt galt. pgx.ErrNoRows, wenn es noch keinen gibt.
func (r *LmfTerminRepository) NeuesterLmfPlan(ctx context.Context, art string) (LmfPlanStand, error) {
	var st LmfPlanStand
	err := r.db.QueryRow(ctx, `
		SELECT id, art, to_char(schuljahr_beginn, 'YYYY-MM-DD'), to_char(erster_tag, 'YYYY-MM-DD'),
		       startstunde, stunden_je_tag
		FROM lmf_plaene WHERE art = $1
		ORDER BY erster_tag DESC LIMIT 1`, art).
		Scan(&st.Plan.ID, &st.Plan.Art, &st.Plan.SchuljahrBeginn, &st.Plan.ErsterTag,
			&st.Plan.Startstunde, &st.Plan.StundenJeTag)
	if err != nil {
		return st, err
	}
	if st.Zeilen, err = r.lmfPlanZeilen(ctx, st.Plan.ID); err != nil {
		return st, err
	}
	st.Ausgelassen, err = r.lmfPlanAusgelassen(ctx, st.Plan.ID)
	return st, err
}

func (r *LmfTerminRepository) lmfPlanZeilen(ctx context.Context, planID string) ([]LmfPlanZeile, error) {
	rows, err := r.db.Query(ctx, `
		SELECT t.position, to_char(t.datum, 'YYYY-MM-DD'), t.stunde, t.vermerk,
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
		if err := rows.Scan(&z.Position, &z.Datum, &z.Stunde, &z.Vermerk, &z.Klassen); err != nil {
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

// SaveLmfPlan legt den Plan der Art für das Schuljahr des ersten Tages an oder schreibt
// ihn um — Rahmen, Zeilen (vollständig ersetzt, mit Platz und Position) und ausgelassene
// Klassen in einer Transaktion. Klassennamen laufen durch das Vokabular; die Antwort
// trägt die kanonisierten Namen.
func (r *LmfTerminRepository) SaveLmfPlan(ctx context.Context, plan LmfPlan, zeilen []LmfPlanZeile, plaetze []lmfplan.Platz, ausgelassen []string) (LmfPlanStand, error) {
	ersterTag, err := time.ParseInLocation("2006-01-02", plan.ErsterTag, schulzeit.Zone())
	if err != nil {
		return LmfPlanStand{}, err
	}
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return LmfPlanStand{}, err
	}
	defer db.SafeRollback(ctx, tx)

	sjb := SchuljahrBeginn(ersterTag)
	var st LmfPlanStand
	if err := tx.QueryRow(ctx, `
		INSERT INTO lmf_plaene (art, schuljahr_beginn, erster_tag, startstunde, stunden_je_tag)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (art, schuljahr_beginn) DO UPDATE
		  SET erster_tag = EXCLUDED.erster_tag, startstunde = EXCLUDED.startstunde,
		      stunden_je_tag = EXCLUDED.stunden_je_tag
		RETURNING id, art, to_char(schuljahr_beginn, 'YYYY-MM-DD'), to_char(erster_tag, 'YYYY-MM-DD'),
		          startstunde, stunden_je_tag`,
		plan.Art, sjb, ersterTag, plan.Startstunde, plan.StundenJeTag).
		Scan(&st.Plan.ID, &st.Plan.Art, &st.Plan.SchuljahrBeginn, &st.Plan.ErsterTag,
			&st.Plan.Startstunde, &st.Plan.StundenJeTag); err != nil {
		return st, err
	}
	// Zeilen vollständig ersetzen: Der Plan IST die Reihenfolge, Einzel-IDs gibt es nicht.
	if _, err := tx.Exec(ctx, `DELETE FROM lmf_termine WHERE plan_id = $1`, st.Plan.ID); err != nil {
		return st, err
	}
	if _, err := tx.Exec(ctx, `DELETE FROM lmf_plan_ausgelassen WHERE plan_id = $1`, st.Plan.ID); err != nil {
		return st, err
	}
	st.Zeilen = make([]LmfPlanZeile, 0, len(zeilen))
	for i, z := range zeilen {
		var id string
		if err := tx.QueryRow(ctx, `
			INSERT INTO lmf_termine (plan_id, position, datum, stunde, art, vermerk)
			VALUES ($1, $2, $3, $4, $5, $6) RETURNING id`,
			st.Plan.ID, i+1, plaetze[i].Datum, plaetze[i].Stunde, plan.Art, z.Vermerk).Scan(&id); err != nil {
			return st, err
		}
		kanonisch, err := schreibeKlassen(ctx, tx, `INSERT INTO lmf_termin_klassen (termin_id, klasse) VALUES ($1, $2)
			ON CONFLICT DO NOTHING RETURNING klasse`, id, z.Klassen)
		if err != nil {
			return st, err
		}
		st.Zeilen = append(st.Zeilen, LmfPlanZeile{Position: i + 1, Datum: plaetze[i].Datum.Format("2006-01-02"),
			Stunde: plaetze[i].Stunde, Klassen: kanonisch, Vermerk: z.Vermerk})
	}
	if st.Ausgelassen, err = schreibeKlassen(ctx, tx, `INSERT INTO lmf_plan_ausgelassen (plan_id, klasse) VALUES ($1, $2)
		ON CONFLICT DO NOTHING RETURNING klasse`, st.Plan.ID, ausgelassen); err != nil {
		return st, err
	}
	return st, tx.Commit(ctx)
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

// DeleteLmfPlan entfernt einen Plan samt Zeilen und Auslassungen (CASCADE).
func (r *LmfTerminRepository) DeleteLmfPlan(ctx context.Context, id string) (bool, error) {
	tag, err := r.db.Exec(ctx, `DELETE FROM lmf_plaene WHERE id = $1`, id)
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() > 0, nil
}

// FreieTage liefert Ferien und Schließzeiten, die den Zeitraum berühren — der Plan
// überspringt sie (pkg/lmfplan.Schultage).
func (r *LmfTerminRepository) FreieTage(ctx context.Context, von, bis time.Time) ([]lmfplan.Zeitraum, error) {
	rows, err := r.db.Query(ctx, `
		SELECT start_datum, end_datum FROM ferien_schliesszeiten
		WHERE end_datum >= $1::date AND start_datum <= $2::date`, von, bis)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	frei := []lmfplan.Zeitraum{}
	for rows.Next() {
		var z lmfplan.Zeitraum
		if err := rows.Scan(&z.Von, &z.Bis); err != nil {
			return nil, err
		}
		frei = append(frei, z)
	}
	return frei, rows.Err()
}

// KlasseImPlan ist eine Klasse für den Vorschlag der Reihenfolge.
type KlasseImPlan struct {
	Name      string
	Abschluss bool
	Oberstufe bool
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
		var jahrgang int
		if err := rows.Scan(&k.Name, &k.Abschluss, &jahrgang); err != nil {
			return nil, err
		}
		k.Oberstufe = jahrgang >= 11
		klassen = append(klassen, k)
	}
	return klassen, rows.Err()
}
