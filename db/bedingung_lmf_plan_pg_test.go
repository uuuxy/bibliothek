package db

import (
	"testing"

	"github.com/jackc/pgx/v5"
)

// Drei der vier Zusagen des LMF-Planers sind Struktur und stehen deshalb in der Datenbank: So
// gelten sie auch für den Weg, den kein Go-Code nimmt, psql von Hand, ein Skript zur Reparatur,
// einen zweiten Schreiber. Die vierte (`len(plaetze) == len(zeilen)`) ist eine Bedingung zwischen
// zwei Slices in Go und steht in repository/lmf_plan.go.
//
// Erreichbar ist keine der drei, solange LMFAbfragen.SchreibePlan der einzige Schreiber ist und
// alle Werte selbst setzt. Die Zusage kostet nichts, solange niemand sie verletzt, und sie steht,
// wenn der zweite Schreiber kommt.
//
// legeLMFPlanAn legt einen Plan der Art art an und liefert seine Kennung.
func legeLMFPlanAn(t *testing.T, tx pgx.Tx, art string, stundenJeTag, letzteStunde int) string {
	t.Helper()
	var id string
	// chk_lmf_plaene_anker: Der Anker (letzter Tag und letzte Stunde) gehört zum Rückgabe-Plan
	// und nur zu ihm; beim Ausgabe-Plan ist er NULL.
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

// Zusage 1: Die Art der Zeile ist die Art ihres Plans. Eine Zeile mit 'ausgabe' unter einem
// Rückgabe-Plan wäre eine Frist, die im Portal unter der falschen Überschrift steht, und für die
// Kopplung der Fristen eine Zeile, die es nicht gibt.
func TestLMFPlan_ZeileFolgtDerArtIhresPlans(t *testing.T) {
	pool := pgTestPool(t)
	inTx(t, pool, func(tx pgx.Tx) {
		planID := legeLMFPlanAn(t, tx, "rueckgabe", 6, 4)
		erwarteVerletzteBedingung(t, tx, "fk_lmf_termine_plan_art",
			`INSERT INTO lmf_termine (plan_id, position, datum, stunde, art)
			 VALUES ($1, 1, '2027-06-10', 1, 'ausgabe')`, planID)
		erwarteErfolg(t, tx, "Zeile mit der Art ihres Plans",
			`INSERT INTO lmf_termine (plan_id, position, datum, stunde, art)
			 VALUES ($1, 1, '2027-06-10', 1, 'rueckgabe')`, planID)
	})
}

// Zusage 2: Die Position ist die Reihenfolge des Plans. Zweimal dieselbe Zahl hieße, dass zwei
// Klassen um denselben Platz streiten und die Anzeige von der Reihenfolge des Speicherns abhängt.
func TestLMFPlan_PositionIstJePlanEindeutig(t *testing.T) {
	pool := pgTestPool(t)
	inTx(t, pool, func(tx pgx.Tx) {
		planID := legeLMFPlanAn(t, tx, "rueckgabe", 6, 4)
		erwarteErfolg(t, tx, "erste Zeile",
			`INSERT INTO lmf_termine (plan_id, position, datum, stunde, art)
			 VALUES ($1, 1, '2027-06-10', 1, 'rueckgabe')`, planID)
		erwarteVerletzteBedingung(t, tx, "uniq_lmf_termine_plan_position",
			`INSERT INTO lmf_termine (plan_id, position, datum, stunde, art)
			 VALUES ($1, 1, '2027-06-10', 2, 'rueckgabe')`, planID)
		erwarteErfolg(t, tx, "zweite Position",
			`INSERT INTO lmf_termine (plan_id, position, datum, stunde, art)
			 VALUES ($1, 2, '2027-06-10', 2, 'rueckgabe')`, planID)

		// Dieselbe Position unter einem anderen Plan ist richtig: Jeder Plan zählt seine Zeilen
		// von vorn.
		zweiter := legeLMFPlanAn(t, tx, "ausgabe", 6, 0)
		erwarteErfolg(t, tx, "Position 1 im zweiten Plan",
			`INSERT INTO lmf_termine (plan_id, position, datum, stunde, art)
			 VALUES ($1, 1, '2026-09-10', 1, 'ausgabe')`, zweiter)
	})
}

// Zusage 3: Die letzte Stunde des Rückgabe-Plans liegt im Schultag, den derselbe Plan beschreibt.
// Sonst rechnet der Planer rückwärts von einer Stunde, die es an diesem Tag nicht gibt, und die
// Zeilen landen einen Tag zu spät.
func TestLMFPlan_LetzteStundeLiegtImTag(t *testing.T) {
	pool := pgTestPool(t)
	inTx(t, pool, func(tx pgx.Tx) {
		erwarteVerletzteBedingung(t, tx, "chk_lmf_plaene_letzte_stunde_im_tag",
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
