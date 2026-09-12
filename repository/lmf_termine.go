package repository

import (
	"context"
	"time"

	"bibliothek/db"
	"bibliothek/pkg/schulzeit"
)

// Der LMF-Plan (Migration 096): Rückgabe- und Ausgabetermine je Klasse — die Excel-
// Tabelle der Schule als Tabelle im System. Ein Termin hat Datum, Stunde, Art und
// Vermerk und 0..n Klassen aus dem Vokabular; „Bücher setzen" ist ein Termin ohne
// Klasse, „6F1/6F2" einer mit zwei. Geschrieben werden Termine seit Migration 097
// NUR über den Plan (lmf_plan.go): Die Reihenfolge ist die Wahrheit, Datum und Stunde
// sind gerechnet — eine zweite Tür je Zeile gäbe zwei Wahrheiten.

// Die zwei Arten eines Termins: Rückgabe (setzt die Frist der Klasse) und Ausgabe.
const (
	LmfTerminRueckgabe = "rueckgabe"
	LmfTerminAusgabe   = "ausgabe"
)

// LmfTermin ist eine Zeile des Plans, so wie Oberfläche und PDF sie lesen. „Nur
// Rückgabe" ist seit 06.09.2026 kein gerechnetes Feld mehr, sondern Text im Vermerk —
// der Planer belegt ihn im Vorschlag vor, die Bibliothek darf ihn ändern (Peter: „fest
// verankert … das ist nicht gut, es sollte im Feld sein").
type LmfTermin struct {
	ID      string   `json:"id"`
	Datum   string   `json:"datum"` // YYYY-MM-DD
	Stunde  int      `json:"stunde"`
	Art     string   `json:"art"`
	Klassen []string `json:"klassen"`
	Vermerk string   `json:"vermerk"`
}

// LmfListenFilter steuert ListLmfTermine: MitEntwuerfen zeigt auch unveröffentlichte
// Pläne (nur der Planer selbst, für das PDF an die Schulleitung).
type LmfListenFilter struct {
	MitEntwuerfen bool
}

// SchuljahrBeginn liefert den 1. August des Schuljahres, in dem t liegt (Hessen:
// 1. August bis 31. Juli) — in der Schulzeitzone, weil daraus Kalendertage werden.
func SchuljahrBeginn(t time.Time) time.Time {
	t = t.In(schulzeit.Zone())
	jahr := t.Year()
	if t.Month() < time.August {
		jahr--
	}
	return time.Date(jahr, time.August, 1, 0, 0, 0, 0, schulzeit.Zone())
}

// schuljahrVon liefert das Schuljahr eines Zeitpunkts als Zahl — das Jahr seines
// 1. August (Schuljahr 2026/27 → 2026). Die Differenz zweier solcher Zahlen ist die Zahl
// der Schuljahre dazwischen; genau das braucht das Verleihjahr im Schadensersatz-Bescheid.
func schuljahrVon(t time.Time) int {
	return SchuljahrBeginn(t).Year()
}

// LmfTerminRepository liest und schreibt den Plan.
type LmfTerminRepository struct {
	db db.PgxPoolIface
}

// NewLmfTerminRepository baut das Repository über dem Pool.
func NewLmfTerminRepository(pool db.PgxPoolIface) *LmfTerminRepository {
	return &LmfTerminRepository{db: pool}
}

// ListLmfTermine liefert die Termine veröffentlichter Pläne ab einem Datum
// (einschließlich), nach Datum und Stunde sortiert; ab = Nullzeit liefert alle. Die
// Klassen kommen sortiert mit, damit „6F1/6F2" in Oberfläche und PDF gleich aussieht.
// Entwürfe (Migration 100) sieht nur, wer MitEntwuerfen setzt.
func (r *LmfTerminRepository) ListLmfTermine(ctx context.Context, ab time.Time, f LmfListenFilter) ([]LmfTermin, error) {
	rows, err := r.db.Query(ctx, `
		SELECT t.id, to_char(t.datum, 'YYYY-MM-DD'), t.stunde, t.art, t.vermerk,
		       COALESCE((SELECT array_agg(k.klasse ORDER BY klassen_normkey(k.klasse))
		                 FROM lmf_termin_klassen k WHERE k.termin_id = t.id), '{}')
		FROM lmf_termine t
		JOIN lmf_plaene p ON p.id = t.plan_id
		WHERE ($1::date IS NULL OR t.datum >= $1::date)
		  AND ($2 OR p.veroeffentlicht_am IS NOT NULL)
		ORDER BY t.datum, t.stunde, t.position`, nullbaresDatum(ab), f.MitEntwuerfen)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	termine := []LmfTermin{}
	for rows.Next() {
		var t LmfTermin
		if err := rows.Scan(&t.ID, &t.Datum, &t.Stunde, &t.Art, &t.Vermerk, &t.Klassen); err != nil {
			return nil, err
		}
		termine = append(termine, t)
	}
	return termine, rows.Err()
}

// nullbaresDatum macht aus der Nullzeit ein SQL-NULL, damit „alle" ohne zweites
// Statement geht.
func nullbaresDatum(t time.Time) *time.Time {
	if t.IsZero() {
		return nil
	}
	return &t
}

// RueckgabeTerminFuerKlasse liefert den nächsten Rückgabe-Termin der Klasse ab dem
// Datum (einschließlich) — die Frist ihrer Lernmittel (Register, Entscheidung 3a:
// „das wäre doch logisch"). ok = false, wenn der Plan für die Klasse nichts nennt;
// dann gilt der globale Stichtag. Verglichen wird über den Normschlüssel. Nur
// veröffentlichte Pläne (Migration 100): Ein Entwurf setzt keine Frist — auch nicht
// still beim Ausleihen, während die Schulleitung ihn noch prüft.
func (r *LmfTerminRepository) RueckgabeTerminFuerKlasse(ctx context.Context, klasse string, ab time.Time) (time.Time, bool, error) {
	var datum *time.Time
	err := r.db.QueryRow(ctx, `
		SELECT min(t.datum)
		FROM lmf_termine t
		JOIN lmf_termin_klassen k ON k.termin_id = t.id
		JOIN lmf_plaene p ON p.id = t.plan_id
		WHERE t.art = 'rueckgabe' AND t.datum >= $1::date AND p.veroeffentlicht_am IS NOT NULL
		  AND klassen_normkey(k.klasse) = klassen_normkey($2)`, ab, klasse).Scan(&datum)
	if err != nil || datum == nil {
		return time.Time{}, false, err
	}
	d := *datum
	return time.Date(d.Year(), d.Month(), d.Day(), 0, 0, 0, 0, schulzeit.Zone()), true, nil
}

// SetzeLernmittelFristFuerKlassenIn schreibt die Frist offener Lernmittel-Ausleihen der
// genannten Klassen um — dieselbe Regel wie die Massenverlängerung (api/ausleihe.go):
// nur aktive, nicht gesperrte Schüler, nur Lernmittel, Mahnstufe zurück, wenn die neue
// Frist in der Zukunft liegt. Zusätzlich: nur Fristen im Schuljahr [von, bis) — eine
// mehrjährige Ausleihe (Frist im übernächsten Sommer) folgt dem Plan nicht. Ist
// nurWennFristAm gesetzt, werden nur Ausleihen angefasst, deren Frist genau an diesem
// Tag liegt (Rückweg: ein gelöschter Termin gibt die Frist an den Stichtag zurück, ohne
// Fristen zu berühren, die jemand von Hand gesetzt hat).
//
// Arbeitet auf einem Executor des Aufrufers — die Plan-Handler koppeln die Fristen in
// DERSELBEN Transaktion wie den Plan.
func (r *LmfTerminRepository) SetzeLernmittelFristFuerKlassenIn(ctx context.Context, ex DBQueryer, klassen []string, frist time.Time, von, bis time.Time, nurWennFristAm *time.Time) (int64, error) {
	if len(klassen) == 0 {
		return 0, nil
	}
	var nurTag *time.Time
	if nurWennFristAm != nil {
		t := nurWennFristAm.In(schulzeit.Zone())
		tag := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, schulzeit.Zone())
		nurTag = &tag
	}
	tag, err := ex.Exec(ctx, `
		UPDATE ausleihen a
		SET rueckgabe_frist = $1,
		    mahnstufe = CASE WHEN $1 > CURRENT_TIMESTAMP THEN 0 ELSE a.mahnstufe END,
		    letztes_mahndatum = CASE WHEN $1 > CURRENT_TIMESTAMP THEN NULL ELSE a.letztes_mahndatum END
		FROM schueler s, buecher_exemplare e, buecher_titel t
		WHERE a.schueler_id = s.id
		  AND a.exemplar_id = e.id
		  AND e.titel_id = t.id
		  AND a.rueckgabe_am IS NULL
		  AND s.deleted_at IS NULL
		  AND s.ist_gesperrt = false
		  AND COALESCE(s.is_manually_blocked, false) = false
		  -- EINE Normalform für beide Seiten (klassen_normkey, Migration 079): „09H1" in der
		  -- Akte und „9h1" aus dem Plan sind dieselbe Klasse. KlassenSchluessel (Go) kennt
		  -- die führende Null nicht — deshalb hier nicht.
		  AND klassen_normkey(s.klasse) IN (SELECT klassen_normkey(x) FROM unnest($2::text[]) AS x)
		  AND t.ist_lernmittel
		  AND a.rueckgabe_frist >= $3 AND a.rueckgabe_frist < $4
		  AND ($5::date IS NULL OR (a.rueckgabe_frist AT TIME ZONE $6)::date = $5::date)`,
		frist, klassen, von, bis, nurTag, schulzeit.Zone().String())
	if err != nil {
		return 0, err
	}
	return tag.RowsAffected(), nil
}
