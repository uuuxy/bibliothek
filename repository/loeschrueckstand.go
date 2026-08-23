package repository

// loeschrueckstand.go — der Wächter über die nächtlichen Löschroutinen.
//
// Er stellt jede Frage aus loeschfristen.go noch einmal, aber als `count(*)` statt als
// DELETE/UPDATE, und mit einem Tag Kulanz. Die Antwort ist die einzige, die zählt: Wie
// viele Zeilen müssten seit mindestens einem Tag weg sein und sind es nicht?
//
// Das ist bewusst ZUSTAND, nicht Protokoll. Ein Job, der still scheitert (falsche
// Spalte, fehlende Migration, toter Cron), schreibt keine Logzeile, die jemand liest —
// aber er hinterlässt genau diesen Rückstand. Bis zum 23.08.2026 sah das nur EINE der
// sechs Routinen; die anderen fünf liefen jede Nacht unbeobachtet.

import (
	"context"
	"strconv"
	"strings"
	"time"
)

// LoeschRueckstand ist eine überfällige Menge einer Routine.
type LoeschRueckstand struct {
	Routine string `json:"routine"` // Name der Routine, wie er im Befund erscheint
	Frist   string `json:"frist"`   // die geltende Frist im Klartext ("90 Tage")
	Zeilen  int    `json:"zeilen"`  // überfällig und nicht gelöscht
	Aus     bool   `json:"aus"`     // Frist steht auf 0 = abgeschaltet (kein Fehler)
}

// zaehle beantwortet eine Predikat-Frage als Anzahl.
func (r *BetriebszustandRepository) zaehle(ctx context.Context, tabelle, alias, predikat string, args ...any) (int, error) {
	von := tabelle
	if alias != "" {
		von += " " + alias
	}
	var n int
	err := r.pool.QueryRow(ctx, `SELECT count(*) FROM `+von+` WHERE `+predikat, args...).Scan(&n)
	return n, err
}

// ZaehleLoeschRueckstand prüft ALLE Löschroutinen des Systems und liefert je eine Zeile.
//
// Fehlerverhalten: Der erste Fehler bricht ab und liefert nil — der Aufrufer meldet dann
// „nicht erhoben" (Warnung) statt eines falschen „alles gut". Eine halbe Liste wäre die
// schlechteste Antwort: Sie sähe aus wie eine ganze.
func (r *BetriebszustandRepository) ZaehleLoeschRueckstand(ctx context.Context) ([]LoeschRueckstand, error) {
	ctx, abbrechen := context.WithTimeout(ctx, 10*time.Second)
	defer abbrechen()

	einst, err := NewSystemSettingsRepository(r.pool).GetSettings(ctx)
	if err != nil {
		return nil, err
	}
	freihandTage := TageOderStandard(einst.LesehistorieTage, StandardLesehistorieTage)
	lernmittelTage := TageOderStandard(einst.LesehistorieLernmittelTage, StandardLesehistorieLernmittelTage)
	anliegenTage := TageOderStandard(einst.AnliegenTage, StandardAnliegenTage)
	auditMonate := r.AuditAufbewahrungMonate(ctx)

	stand := []LoeschRueckstand{}
	fehler := func(e error) ([]LoeschRueckstand, error) { return nil, e }

	// 1. Schüler-Anonymisierung (180 Tage nach Soft-Delete / 360 Tage Abgänger).
	n, err := r.zaehle(ctx, "schueler", "", PredikatAnonymisierung(),
		StandardAnonymisierungSoftDeleteTage, StandardAnonymisierungAbgaengerTage, KulanzWaechter)
	if err != nil {
		return fehler(err)
	}
	stand = append(stand, LoeschRueckstand{Routine: "Schüler-Anonymisierung", Frist: "180 Tage (gelöscht) / 360 Tage (Abgänger)", Zeilen: n})

	// 2. Abgänger endgültig löschen. Die Kulanz steckt in AbgaengerStichjahr (30 Tage
	//    nach Jahreswechsel) — ein zusätzlicher Tag wäre hier ohne Bedeutung.
	n, err = r.zaehle(ctx, "schueler", "", PredikatAbgaengerLoeschung(), AbgaengerStichjahr(time.Now()))
	if err != nil {
		return fehler(err)
	}
	stand = append(stand, LoeschRueckstand{Routine: "Abgänger endgültig löschen", Frist: "30 Tage nach Schuljahresende", Zeilen: n})

	// 3./4. Lesehistorie, beide Klassen — Ausleihe UND Protokolleintrag. Die
	//       Protokollzeile trägt dieselbe Zuordnung; wer nur die Ausleihe zählt, sieht
	//       die halbe Wahrheit (Prüfung 22.08.2026, A5).
	for _, k := range []struct {
		routine    string
		tage       int
		lernmittel bool
	}{
		{"Lesehistorie Schülerbücherei", freihandTage, false},
		{"Lesehistorie Lernmittel", lernmittelTage, true},
	} {
		zeile := LoeschRueckstand{Routine: k.routine, Frist: tageText(k.tage), Aus: k.tage <= 0}
		if !zeile.Aus {
			ausleihen, err := r.zaehle(ctx, "ausleihen", "a", PredikatLesehistorieAusleihen(k.lernmittel), k.tage, KulanzWaechter)
			if err != nil {
				return fehler(err)
			}
			protokoll, err := r.zaehle(ctx, "audit_log", "al", PredikatLesehistorieProtokoll(k.lernmittel), k.tage, KulanzWaechter)
			if err != nil {
				return fehler(err)
			}
			zeile.Zeilen = ausleihen + protokoll
		}
		stand = append(stand, zeile)
	}

	// 5. Erledigte Anliegen.
	anliegen := LoeschRueckstand{Routine: "Erledigte Anliegen", Frist: tageText(anliegenTage), Aus: anliegenTage <= 0}
	if !anliegen.Aus {
		if anliegen.Zeilen, err = r.zaehle(ctx, "lehrer_anliegen", "", PredikatAnliegen(), anliegenTage, KulanzWaechter); err != nil {
			return fehler(err)
		}
	}
	stand = append(stand, anliegen)

	// 6. Audit-Aufbewahrung, beide Protokolltabellen.
	datensatz, err := r.zaehle(ctx, "audit_log", "", PredikatAuditLog(), auditMonate, KulanzWaechter)
	if err != nil {
		return fehler(err)
	}
	admin, err := r.zaehle(ctx, "audit_logs", "", PredikatAuditLogs(), auditMonate, KulanzWaechter)
	if err != nil {
		return fehler(err)
	}
	stand = append(stand, LoeschRueckstand{Routine: "Audit-Aufbewahrung", Frist: monateText(auditMonate), Zeilen: datensatz + admin})

	return stand, nil
}

// AuditAufbewahrungMonate liest die Aufbewahrungsfrist der Protokolltabellen — die
// EINE Quelle für den Job (jobs.RunAuditAufbewahrung) und den Wächter. Unlesbar,
// unparsbar oder unter der Untergrenze: die Vorgabe.
func (r *BetriebszustandRepository) AuditAufbewahrungMonate(ctx context.Context) int {
	var wert string
	if err := r.pool.QueryRow(ctx,
		`SELECT wert FROM system_einstellungen WHERE schluessel = $1`,
		AuditAufbewahrungSchluessel).Scan(&wert); err != nil {
		return StandardAuditAufbewahrungMonate
	}
	return AufbewahrungAusText(wert)
}

// AufbewahrungAusText liest den Einstellungswert der Audit-Aufbewahrung. Unparsbar oder
// unter der Untergrenze: die Vorgabe. Job und Wächter lesen die Zeile über denselben
// Weg — sonst könnte der Wächter mit 24 Monaten rechnen, während der Job mit 6 löscht.
func AufbewahrungAusText(wert string) int {
	monate, err := strconv.Atoi(strings.TrimSpace(wert))
	if err != nil || monate < MindestAuditAufbewahrungMonate {
		return StandardAuditAufbewahrungMonate
	}
	return monate
}

// tageText und monateText formulieren eine Frist für den Befundtext. 0 heißt „aus" —
// eine erlaubte Entscheidung der Schule, kein Fehler.
func tageText(tage int) string {
	if tage <= 0 {
		return "abgeschaltet (0)"
	}
	return strconv.Itoa(tage) + " Tage"
}

func monateText(monate int) string { return strconv.Itoa(monate) + " Monate" }
