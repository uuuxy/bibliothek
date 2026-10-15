package repository

// Der Wächter über die nächtlichen Löschroutinen.
//
// Er stellt jede Frage aus loeschfristen.go noch einmal, aber als `count(*)` statt als DELETE
// oder UPDATE, und mit einem Tag Kulanz. Die Antwort ist die einzige, die zählt: Wie viele
// Zeilen müssten seit mindestens einem Tag weg sein und sind es nicht?
//
// Das ist Zustand, nicht Protokoll. Ein Job, der still scheitert (falsche Spalte, fehlende
// Migration, toter Cron), schreibt keine Logzeile, die jemand liest, aber er hinterlässt genau
// diesen Rückstand. Deshalb hat jede Routine ihre Zeile im Wächter.

import (
	"context"
	"fmt"
	"strconv"
	"time"
)

// LoeschRueckstand ist eine überfällige Menge einer Routine.
type LoeschRueckstand struct {
	Routine string `json:"routine"` // Name der Routine, wie er im Befund erscheint
	Frist   string `json:"frist"`   // die geltende Frist im Klartext ("90 Tage")
	Zeilen  int    `json:"zeilen"`  // überfällig und nicht gelöscht
	Aus     bool   `json:"aus"`     // Frist steht auf 0 = abgeschaltet (kein Fehler)
}

// zaehle beantwortet eine Löschbedingung als Anzahl. Sie nimmt die Bedingung als
// ganzes entgegen — Text und Zahlen sind daran nicht mehr getrennt einsetzbar.
func (a *BetriebszustandAbfragen) zaehle(ctx context.Context, tabelle, alias string, b Loeschbedingung) (int, error) {
	von := tabelle
	if alias != "" {
		von += " " + alias
	}
	var n int
	err := a.pool.QueryRow(ctx, `SELECT count(*) FROM `+von+` WHERE `+b.Where, b.Args...).Scan(&n)
	return n, err
}

// ZaehleLoeschRueckstand prüft alle Löschroutinen des Systems und liefert je eine Zeile.
//
// Fehlerverhalten: Der erste Fehler bricht ab und liefert nil — der Aufrufer meldet dann
// „nicht erhoben" (Warnung) statt eines falschen „alles gut". Eine halbe Liste wäre die
// schlechteste Antwort: Sie sähe aus wie eine ganze.
func (a *BetriebszustandAbfragen) ZaehleLoeschRueckstand(ctx context.Context) ([]LoeschRueckstand, error) {
	ctx, abbrechen := context.WithTimeout(ctx, 10*time.Second)
	defer abbrechen()

	einst, err := NewEinstellungsAbfragen(a.pool).Lade(ctx)
	if err != nil {
		return nil, err
	}
	stand, err := a.zaehleRueckstandLeserUndLesehistorie(ctx, einst)
	if err != nil {
		return nil, err
	}
	vorgaenge, err := a.zaehleRueckstandVorgaengeUndProtokoll(ctx, einst)
	if err != nil {
		return nil, err
	}
	return append(stand, vorgaenge...), nil
}

// zaehleRueckstandLeserUndLesehistorie zählt die Routinen, die Leser, ihre Lesehistorie und die
// Bearbeiter ihrer Ausleihen betreffen (1 bis 4a).
func (a *BetriebszustandAbfragen) zaehleRueckstandLeserUndLesehistorie(ctx context.Context, einst *SystemEinstellungen) ([]LoeschRueckstand, error) {
	freihandTage := TageOderVorgabe(einst.LesehistorieTage, LesehistorieTageVorgabe)
	lernmittelTage := TageOderVorgabe(einst.LesehistorieLernmittelTage, LesehistorieLernmittelTageVorgabe)
	karenzTage := AbgaengerKarenzTageOderVorgabe(einst)

	stand := []LoeschRueckstand{}
	fehler := func(e error) ([]LoeschRueckstand, error) { return nil, e }

	// 1. Schüler-Anonymisierung (180 Tage nach Soft-Delete / Karenzzeit nach Abgang).
	n, err := a.zaehle(ctx, "schueler", "", PraedikatAnonymisierung(karenzTage, KulanzWaechter))
	if err != nil {
		return fehler(err)
	}
	stand = append(stand, LoeschRueckstand{Routine: "Schüler-Anonymisierung", Frist: fmt.Sprintf("180 Tage (gelöscht) / %d Tage Karenz (Abgänger)", karenzTage), Zeilen: n})

	// 2. Abgänger endgültig löschen. Die Kulanz steckt in AbgaengerStichjahr (30 Tage
	//    nach Jahreswechsel) — ein zusätzlicher Tag wäre hier ohne Bedeutung.
	n, err = a.zaehle(ctx, "schueler", "", praedikatAbgaengerLoeschung(time.Now()))
	if err != nil {
		return fehler(err)
	}
	stand = append(stand, LoeschRueckstand{Routine: "Abgänger endgültig löschen", Frist: "nach der Karenz (anonymisiert), ab 30. Januar des Folgejahres", Zeilen: n})

	// 2a. Gelöschte Kollegen endgültig löschen (180 Tage im Papierkorb).
	n, err = a.zaehle(ctx, "leser", "", praedikatKollegenPapierkorb(KulanzWaechter))
	if err != nil {
		return fehler(err)
	}
	stand = append(stand, LoeschRueckstand{Routine: "Gelöschte Kollegen endgültig löschen", Frist: tageText(AnonymisierungPapierkorbTage) + " im Papierkorb", Zeilen: n})

	// 3./4. Lesehistorie, beide Klassen — Ausleihe und Protokolleintrag. Die
	//       Protokollzeile trägt dieselbe Zuordnung; wer nur die Ausleihe zählt, sieht
	//       die halbe Wahrheit.
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
			zeilen, err := a.zaehleLesehistorie(ctx, k.tage, k.lernmittel)
			if err != nil {
				return fehler(err)
			}
			zeile.Zeilen = zeilen
		}
		stand = append(stand, zeile)
	}

	// 4a. Bearbeiter an zurückgegebenen Ausleihen (feste Frist).
	n, err = a.zaehle(ctx, "ausleihen", "", praedikatBearbeiterKennung(KulanzWaechter))
	if err != nil {
		return fehler(err)
	}
	stand = append(stand, LoeschRueckstand{Routine: "Bearbeiter zurückgegebener Ausleihen", Frist: tageText(BearbeiterKennungTage), Zeilen: n})
	return stand, nil
}

// zaehleLesehistorie zählt, was der Lauf der Lesehistorie für eine Klasse noch vor sich hat:
// Ausleihen und ihre Protokollzeilen, bei der Schülerbücherei dazu die Spuren von
// Vormerkungen, die mit ihrem Titel gelöscht wurden. Sie folgen der Frist der Schülerbücherei,
// der Lauf nimmt ihnen den Leser im selben Durchgang.
func (a *BetriebszustandAbfragen) zaehleLesehistorie(ctx context.Context, tage int, lernmittel bool) (int, error) {
	type frage struct {
		tabelle, alias string
		bedingung      Loeschbedingung
	}
	fragen := []frage{
		{"ausleihen", "a", praedikatLesehistorieAusleihen(lernmittel, tage, KulanzWaechter)},
		{"audit_log", "al", praedikatLesehistorieProtokoll(lernmittel, tage, KulanzWaechter)},
	}
	if !lernmittel {
		fragen = append(fragen, frage{"audit_log", "al", praedikatLesehistorieVormerkspur(tage, KulanzWaechter)})
	}
	summe := 0
	for _, f := range fragen {
		n, err := a.zaehle(ctx, f.tabelle, f.alias, f.bedingung)
		if err != nil {
			return 0, err
		}
		summe += n
	}
	return summe, nil
}

// zaehleRueckstandVorgaengeUndProtokoll zählt die Routinen für erledigte Vorgänge und die
// Aufbewahrung der Protokolle (5 und 6).
func (a *BetriebszustandAbfragen) zaehleRueckstandVorgaengeUndProtokoll(ctx context.Context, einst *SystemEinstellungen) ([]LoeschRueckstand, error) {
	anliegenTage := TageOderVorgabe(einst.AnliegenTage, AnliegenTageVorgabe)
	auditMonate := ProtokollAufbewahrungMonateOderVorgabe(einst.ProtokollAufbewahrungMonate)

	stand := []LoeschRueckstand{}
	fehler := func(e error) ([]LoeschRueckstand, error) { return nil, e }
	var err error

	// 5. Erledigte Anliegen.
	anliegen := LoeschRueckstand{Routine: "Erledigte Anliegen", Frist: tageText(anliegenTage), Aus: anliegenTage <= 0}
	if !anliegen.Aus {
		if anliegen.Zeilen, err = a.zaehle(ctx, "lehrer_anliegen", "", praedikatAnliegen(anliegenTage, KulanzWaechter)); err != nil {
			return fehler(err)
		}
	}
	stand = append(stand, anliegen)

	// 5a. Erledigte Klassensatz-Reservierungen: dieselbe Einstellung wie die Anliegen.
	reservierungen := LoeschRueckstand{Routine: "Erledigte Klassensatz-Reservierungen", Frist: tageText(anliegenTage), Aus: anliegenTage <= 0}
	if !reservierungen.Aus {
		if reservierungen.Zeilen, err = a.zaehle(ctx, "klassensatz_reservierungen", "", praedikatKlassensatzReservierungen(anliegenTage, KulanzWaechter)); err != nil {
			return fehler(err)
		}
	}
	stand = append(stand, reservierungen)

	// 5b. Quittierte Nachbuch-Meldungen: Lesehistorie-Frist, höchstens 30 Tage.
	nachbuchTage := NachbuchMeldungenTage(einst)
	nachbuch := LoeschRueckstand{Routine: "Quittierte Nachbuch-Meldungen", Frist: tageText(nachbuchTage)}
	if nachbuch.Zeilen, err = a.zaehle(ctx, "nachbuch_meldungen", "", praedikatNachbuchMeldungen(nachbuchTage, KulanzWaechter)); err != nil {
		return fehler(err)
	}
	stand = append(stand, nachbuch)

	// 6. Audit-Aufbewahrung, beide Protokolltabellen.
	datensatz, err := a.zaehle(ctx, "audit_log", "", praedikatProtokoll(auditMonate, KulanzWaechter))
	if err != nil {
		return fehler(err)
	}
	admin, err := a.zaehle(ctx, "audit_logs", "", praedikatVerwaltungsProtokoll(auditMonate, KulanzWaechter))
	if err != nil {
		return fehler(err)
	}
	stand = append(stand, LoeschRueckstand{Routine: "Audit-Aufbewahrung", Frist: monateText(auditMonate), Zeilen: datensatz + admin})

	return stand, nil
}

// ProtokollAufbewahrungMonateOderVorgabe liest die Aufbewahrungsfrist aus den Einstellungen und
// setzt die Untergrenze durch: die eine Quelle für den Job (jobs.FuehreProtokollAufbewahrungAus) und den
// Wächter. Eine zweite Stelle, die die Zeile an den Einstellungen vorbei läse, wäre eine zweite
// Wahrheit für denselben Wert.
func ProtokollAufbewahrungMonateOderVorgabe(v *int) int {
	if v == nil || *v < protokollAufbewahrungMonateMinimum {
		return ProtokollAufbewahrungMonateVorgabe
	}
	return *v
}

// tageText und monateText formulieren eine Frist für den Befundtext. 0 heißt „aus" —
// eine erlaubte Entscheidung der Schule, kein Fehler.
func tageText(tage int) string {
	if tage <= 0 {
		return "abgeschaltet (0)"
	}
	return TageMitZahl(tage)
}

// TageMitZahl nennt eine Zahl von Tagen mit dem passenden Wort: „1 Tag", sonst „N Tage".
// Selbstprüfung und Datenschutz-Auskunft schreiben ihre Fristen damit; die Frist der
// Schülerbücherei ist ein Tag, die Eins kommt dort also vor.
func TageMitZahl(tage int) string {
	if tage == 1 {
		return "1 Tag"
	}
	return strconv.Itoa(tage) + " Tage"
}

func monateText(monate int) string { return strconv.Itoa(monate) + " Monate" }
