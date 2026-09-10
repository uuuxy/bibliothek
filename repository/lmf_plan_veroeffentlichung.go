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

// NormalisiereEingangsjahrgaenge prüft die Eingabe und gibt sie in Normalform zurück
// („5, 7"). Fehler statt stiller Vorgabe: Bis zum Rasterdurchgang am 06.09.2026 wurde
// jeder Text gespeichert und angezeigt, aber beim Lesen still verworfen — „8/9" (mit
// Schrägstrich, so wie Klassen geschrieben werden) hieß gespeichert „8/9" und gerechnet
// „5, 7". Hier stand bis zum 10.09.2026, der Nachbar lmf_stichtag
// prüfe „seit jeher" — er tat es nicht; seitdem NormalisiereLmfStichtag.
func NormalisiereEingangsjahrgaenge(text string) (string, error) {
	if strings.TrimSpace(text) == "" {
		return "", nil // leer heißt Vorgabe — das entscheidet der Patch-Pfad
	}
	teile := strings.FieldsFunc(text, func(r rune) bool { return r == ',' || r == ';' || r == ' ' })
	gesehen := map[int]bool{}
	var jahrgaenge []int
	for _, teil := range teile {
		n, err := strconv.Atoi(strings.TrimPrefix(teil, "0"))
		if err != nil || n < 1 || n > 13 {
			//nolint:staticcheck // ST1005: nutzer-sichtbarer Text, erscheint als 400-Meldung im Formular
			return "", fmt.Errorf("Eingangsjahrgänge: %q ist kein Jahrgang zwischen 1 und 13 — erwartet wird etwa „5, 7“", teil)
		}
		if gesehen[n] {
			continue
		}
		gesehen[n] = true
		jahrgaenge = append(jahrgaenge, n)
	}
	sort.Ints(jahrgaenge)
	worte := make([]string, 0, len(jahrgaenge))
	for _, j := range jahrgaenge {
		worte = append(worte, strconv.Itoa(j))
	}
	return strings.Join(worte, ", "), nil
}

// tageImMonat eines Nicht-Schaltjahres: Ein Stichtag 29.02. gäbe es drei von vier Jahren nicht.
var tageImMonat = [13]int{0, 31, 28, 31, 30, 31, 30, 31, 31, 30, 31, 30, 31}

// NormalisiereLmfStichtag prüft den Lernmittel-Stichtag: „MM-TT", ein echter Kalendertag.
// Leer heißt Vorgabe (das entscheidet der Patch-Pfad). Bis zum 10.09.2026 prüfte der
// Server ihn nicht: „30.06." wurde gespeichert und angezeigt, gerechnet wurde still mit
// 07-31 (lmfStichtagMonatTag), „02-30" wurde zum 2. März (Bestands-Durchgang).
func NormalisiereLmfStichtag(text string) (string, error) {
	s := strings.TrimSpace(text)
	if s == "" {
		return "", nil
	}
	if len(s) == 5 && s[2] == '-' {
		m, err1 := strconv.Atoi(s[:2])
		d, err2 := strconv.Atoi(s[3:])
		if err1 == nil && err2 == nil && m >= 1 && m <= 12 && d >= 1 && d <= tageImMonat[m] {
			return s, nil
		}
	}
	//nolint:staticcheck // ST1005: nutzer-sichtbarer Text, erscheint als 400-Meldung im Formular
	return "", fmt.Errorf("Lernmittel-Stichtag: %q ist kein Kalendertag im Format MM-TT — erwartet wird etwa „07-31“ für den 31. Juli", text)
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

// VeroeffentlicheLmfPlanIn stempelt den Plan; ein schon veröffentlichter behält seinen
// Stempel (idempotent). Liefert den vollständigen Stand wie NeuesterLmfPlan.
//
// Arbeitet auf einem Executor des Aufrufers (Transaktion des
// Handlers, im Test der Pool). Die Teile (Zeilen, freie Tage, Auslassungen) liest sie
// weiter am Pool: Sie wurden in dieser Transaktion nicht geschrieben, es gibt nichts
// zurückzulesen.
func (r *LmfTerminRepository) VeroeffentlicheLmfPlanIn(ctx context.Context, ex DBQueryer, id string, jetzt time.Time) (LmfPlanStand, error) {
	var st LmfPlanStand
	err := scanLmfPlan(ex.QueryRow(ctx, `
		UPDATE lmf_plaene SET veroeffentlicht_am = COALESCE(veroeffentlicht_am, $2)
		WHERE id = $1
		RETURNING `+lmfPlanSpalten, id, jetzt), &st.Plan)
	if err != nil {
		return st, err
	}
	return r.ladeLmfPlanTeile(ctx, st)
}
