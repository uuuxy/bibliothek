package db

import (
	"testing"

	"github.com/jackc/pgx/v5"
)

// Die vier Zusicherungen des LMF-Planers standen bis zum 12.09.2026 nur in Go und in
// Kommentaren (Register, Paket 5: „Vier Zusicherungen halten nur per Verabredung").
// Drei davon sind Struktur und gehören deshalb in die Datenbank — sie gelten dann auch
// für den Weg, den kein Go-Code nimmt: psql von Hand, ein Reparaturskript, ein zweiter
// Schreiber.
//
// Die vierte (`len(plaetze) == len(zeilen)`) ist keine Struktur, sondern eine Bedingung
// zwischen zwei Go-Scheiben; sie steht in repository/lmf_plan.go.
//
// Heute ist keine der drei erreichbar: EIN Schreiber (SaveLmfPlanIn) setzt alle Werte
// selbst. Das ist genau der Moment, in dem eine Zusicherung billig ist — sie kostet
// nichts, solange niemand sie verletzt, und sie steht, wenn der zweite Schreiber kommt.
func neuerLmfPlan(t *testing.T, tx pgx.Tx, art string, stundenJeTag, letzteStunde int) string {
	t.Helper()
	var id string
	// chk_lmf_plaene_anker: Der Anker (letzter Tag + letzte Stunde) gehört zum
	// Rückgabe-Plan und NUR zu ihm; beim Ausgabe-Plan muss er NULL sein.
	var letzterTag, stunde any
	ersterTag := "2026-09-10"
	if art == "rueckgabe" {
		letzterTag, stunde, ersterTag = "2027-06-17", letzteStunde, "2027-06-10"
	}
	err := tx.QueryRow(t.Context(), `
		INSERT INTO lmf_plaene (art, schuljahr_beginn, erster_tag, startstunde, stunden_je_tag, letzter_tag, letzte_stunde)
		VALUES ($1, '2026-08-01', $2::date, 1, $3, $4::date, $5) RETURNING id`,
		art, ersterTag, stundenJeTag, letzterTag, stunde).Scan(&id)
	if err != nil {
		t.Fatalf("Plan (%s) anlegen: %v", art, err)
	}
	return id
}

// Zusicherung 1: Die Art der Zeile IST die Art ihres Plans. Eine Zeile mit 'ausgabe'
// unter einem Rückgabe-Plan wäre eine Frist, die im Portal unter der falschen
// Überschrift steht — und für die Frist-Kopplung eine Zeile, die es nicht gibt.
func TestLmfPlan_ZeileFolgtDerArtIhresPlans(t *testing.T) {
	pool := pgTestPool(t)
	inTx(t, pool, func(tx pgx.Tx) {
		planID := neuerLmfPlan(t, tx, "rueckgabe", 6, 4)
		erwarteConstraintVerletzung(t, tx, "fk_lmf_termine_plan_art",
			`INSERT INTO lmf_termine (plan_id, position, datum, stunde, art)
			 VALUES ($1, 1, '2027-06-10', 1, 'ausgabe')`, planID)
		erwarteErfolg(t, tx, "Zeile mit der Art ihres Plans",
			`INSERT INTO lmf_termine (plan_id, position, datum, stunde, art)
			 VALUES ($1, 1, '2027-06-10', 1, 'rueckgabe')`, planID)
	})
}

// Zusicherung 2: Die Position ist die Reihenfolge des Plans — zweimal dieselbe Zahl
// heißt, dass zwei Klassen um denselben Platz streiten und die Anzeige von der
// Speicherreihenfolge abhängt.
func TestLmfPlan_PositionIstJePlanEindeutig(t *testing.T) {
	pool := pgTestPool(t)
	inTx(t, pool, func(tx pgx.Tx) {
		planID := neuerLmfPlan(t, tx, "rueckgabe", 6, 4)
		erwarteErfolg(t, tx, "erste Zeile",
			`INSERT INTO lmf_termine (plan_id, position, datum, stunde, art)
			 VALUES ($1, 1, '2027-06-10', 1, 'rueckgabe')`, planID)
		erwarteConstraintVerletzung(t, tx, "uniq_lmf_termine_plan_position",
			`INSERT INTO lmf_termine (plan_id, position, datum, stunde, art)
			 VALUES ($1, 1, '2027-06-10', 2, 'rueckgabe')`, planID)
		erwarteErfolg(t, tx, "zweite Position",
			`INSERT INTO lmf_termine (plan_id, position, datum, stunde, art)
			 VALUES ($1, 2, '2027-06-10', 2, 'rueckgabe')`, planID)

		// Dieselbe Position unter einem ANDEREN Plan ist richtig — jeder Plan zählt
		// seine Zeilen von vorn.
		zweiter := neuerLmfPlan(t, tx, "ausgabe", 6, 0)
		erwarteErfolg(t, tx, "Position 1 im zweiten Plan",
			`INSERT INTO lmf_termine (plan_id, position, datum, stunde, art)
			 VALUES ($1, 1, '2026-09-10', 1, 'ausgabe')`, zweiter)
	})
}

// Zusicherung 3: Die letzte Stunde des Rückgabe-Plans liegt im Schultag, den derselbe
// Plan beschreibt. Andernfalls rechnet der Planer rückwärts von einer Stunde, die es
// an diesem Tag nicht gibt — die Zeilen landen einen Tag zu spät.
func TestLmfPlan_LetzteStundeLiegtImTag(t *testing.T) {
	pool := pgTestPool(t)
	inTx(t, pool, func(tx pgx.Tx) {
		erwarteConstraintVerletzung(t, tx, "chk_lmf_plaene_letzte_stunde_im_tag",
			`INSERT INTO lmf_plaene (art, schuljahr_beginn, erster_tag, startstunde, stunden_je_tag, letzter_tag, letzte_stunde)
			 VALUES ('rueckgabe', '2026-08-01', '2027-06-10', 1, 6, '2027-06-17', 7)`)
		erwarteErfolg(t, tx, "letzte Stunde = Stunden je Tag",
			`INSERT INTO lmf_plaene (art, schuljahr_beginn, erster_tag, startstunde, stunden_je_tag, letzter_tag, letzte_stunde)
			 VALUES ('rueckgabe', '2026-08-01', '2027-06-10', 1, 6, '2027-06-17', 6)`)
		// Der Ausgabe-Plan hat keine letzte Stunde; NULL darf der Check nicht stören.
		erwarteErfolg(t, tx, "Ausgabe-Plan ohne Anker",
			`INSERT INTO lmf_plaene (art, schuljahr_beginn, erster_tag, startstunde, stunden_je_tag)
			 VALUES ('ausgabe', '2026-08-01', '2026-09-10', 1, 6)`)
	})
}
