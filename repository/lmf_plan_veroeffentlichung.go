package repository

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"
)

// Entwurf und Veröffentlichung des LMF-Plans (Migration 100, Peter 06.09.2026: „der Plan
// nimmt die Schulleitung immer erst ab") und die zwei Regeln, die den Plan je Art
// verschieden machen (Peter, 06.09.2026):
//
//   - Vor den Sommerferien geben ALLE Klassen ihre alten Bücher ab und bekommen direkt die
//     neuen — außer den Abschlussklassen und den Klassen, die zum neuen Schuljahr neu
//     gebildet werden (die 6er werden auf die Zweige H, R, G verteilt): die geben NUR ab.
//   - Nach den Sommerferien bekommen nur die neu gebildeten Klassen ihre Bücher — die
//     Eingangsjahrgänge, Einstellung lmf_eingangsjahrgaenge (Vorgabe „5, 7"; „manchmal
//     kann ja auch mal was dazwischenkommen").

// LmfEingangsjahrgaengeVorgabe ist die Vorgabe der Einstellung: neue 5er und 7er.
const LmfEingangsjahrgaengeVorgabe = "5, 7"

// EingangsjahrgaengeAus liest die Einstellung („5, 7", „5;7", „5 7") als sortierte,
// duplikatfreie Jahrgänge 1–13. Unlesbares fällt weg; ganz ohne lesbaren Wert gilt die
// Vorgabe — eine leere Menge hieße „niemand bekommt nach den Ferien Bücher".
func EingangsjahrgaengeAus(einstellung string) []int {
	gesehen := map[int]bool{}
	var jahrgaenge []int
	for _, teil := range strings.FieldsFunc(einstellung, func(r rune) bool { return r == ',' || r == ';' || r == ' ' }) {
		n, err := strconv.Atoi(strings.TrimPrefix(teil, "0"))
		if err != nil || n < 1 || n > 13 || gesehen[n] {
			continue
		}
		gesehen[n] = true
		jahrgaenge = append(jahrgaenge, n)
	}
	if len(jahrgaenge) == 0 {
		return EingangsjahrgaengeAus(LmfEingangsjahrgaengeVorgabe)
	}
	sort.Ints(jahrgaenge)
	return jahrgaenge
}

// nurRueckgabeSQL ist das Prädikat „diese Klasse gibt vor den Ferien nur ab": Abschluss-
// klasse (AbschlussklasseSQL, die eine Regel) oder ihr nächster Jahrgang ist ein
// Eingangsjahrgang (die Klasse wird neu gebildet). eingang ist der Platzhalter eines
// int[]-Parameters. COALESCE, weil substring auf „Q1" NULL liefert.
func nurRueckgabeSQL(spalte, eingang string) string {
	return fmt.Sprintf(`COALESCE(%s OR (substring(%s from '^\d+')::int + 1) = ANY(%s::int[]), false)`,
		AbschlussklasseSQL(spalte), spalte, eingang)
}

// KlassenNurRueckgabe nennt aus den Namen die Klassen, die vor den Ferien nur abgeben —
// für die Markierung im Planer. Verglichen wird am Namen, Schüler braucht es nicht
// (ein „07G6" aus dem Vorjahr ohne Schüler wird genauso eingeordnet).
func (r *LmfTerminRepository) KlassenNurRueckgabe(ctx context.Context, namen []string, eingang []int) ([]string, error) {
	if len(namen) == 0 {
		return []string{}, nil
	}
	rows, err := r.db.Query(ctx, `
		SELECT DISTINCT u.x FROM unnest($1::text[]) AS u(x)
		WHERE `+nurRueckgabeSQL("u.x", "$2")+`
		ORDER BY u.x`, namen, eingang)
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

// VeroeffentlicheLmfPlan stempelt den Plan; ein schon veröffentlichter behält seinen
// Stempel (idempotent). Liefert den vollständigen Stand wie NeuesterLmfPlan.
func (r *LmfTerminRepository) VeroeffentlicheLmfPlan(ctx context.Context, id string, jetzt time.Time) (LmfPlanStand, error) {
	var st LmfPlanStand
	err := scanLmfPlan(r.db.QueryRow(ctx, `
		UPDATE lmf_plaene SET veroeffentlicht_am = COALESCE(veroeffentlicht_am, $2)
		WHERE id = $1
		RETURNING `+lmfPlanSpalten, id, jetzt), &st.Plan)
	if err != nil {
		return st, err
	}
	return r.ladeLmfPlanTeile(ctx, st)
}
