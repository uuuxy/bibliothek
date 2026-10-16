package jobs

import (
	"context"
	"log"
	"time"

	"bibliothek/repository"
)

// ── Erledigte Anliegen befristen ─────────────────────────────────────────────
//
// `lehrer_anliegen` (Wünsche und Meldungen aus dem Portal des Kollegiums) trägt den Freitext der
// Lehrkraft, ihre Klassenangabe und über den Fremdschlüssel ihren Namen. Erledigt ist der Zweck
// erreicht.
//
// Das ist dieselbe Abwägung wie bei den Fristen der Lesehistorie, nur für Daten von
// Beschäftigten: so lange wie nötig, nicht so lange wie möglich. Ein Jahr deckt den Blick zurück
// über ein Schuljahr („war der Titel schon einmal gewünscht?").
//
// Angefasst werden nur erledigte Anliegen. Ein offener Wunsch ist eine laufende Sache und hat
// keine Frist; er verschwindet erst, wenn ihn jemand erledigt.

// fuehreAnliegenBefristungAus löscht erledigte Anliegen nach Ablauf der Frist.
func (z *Zeitplan) fuehreAnliegenBefristungAus() {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	einst, err := repository.NewEinstellungsAbfragen(z.pool).Lade(ctx)
	if err != nil {
		log.Printf("Scheduler Anliegen: Einstellungen nicht lesbar, Lauf übersprungen: %v", err)
		return
	}
	tage := repository.TageOderVorgabe(einst.AnliegenTage, repository.AnliegenTageVorgabe)

	// Anweisung und Bedingung stehen in repository/: dieselbe Bedingung, die der Wächter des
	// Rückstands zählt. Eine Frist von 0 schaltet die Befristung ab; dann löscht die Abfrage
	// nichts, und der Lauf endet hier ohne Eintrag.
	geloescht, err := repository.LoescheErledigteAnliegen(ctx, z.pool, tage)
	if err != nil {
		log.Printf("Scheduler Anliegen: Löschen fehlgeschlagen: %v", err)
		return
	}
	if geloescht == 0 {
		return
	}
	log.Printf("Scheduler Anliegen: %d erledigte Anliegen nach %d Tagen gelöscht", geloescht, tage)

	if err := z.protokollAbfragen.ProtokolliereSystemAktion(ctx, "lehrer_anliegen", "DELETE",
		"DSGVO: erledigte Anliegen nach Frist gelöscht",
		map[string]any{"geloescht": geloescht, "tage": tage}); err != nil {
		log.Printf("Scheduler Anliegen: Audit-Eintrag fehlgeschlagen: %v", err)
	}
}

// ── Erledigte Reservierungen von Klassensätzen befristen ─────────────────────
//
// Reservierungen aus dem Portal des Kollegiums tragen die Klasse, eine Notiz und über
// angefordert_von das Konto der Lehrkraft. Sie fallen nach derselben Frist wie die erledigten
// Anliegen, mit derselben Einstellung. Offene Reservierungen bleiben.

// fuehreKlassensatzBefristungAus löscht erledigte Reservierungen von Klassensätzen nach der Frist.
func (z *Zeitplan) fuehreKlassensatzBefristungAus() {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	einst, err := repository.NewEinstellungsAbfragen(z.pool).Lade(ctx)
	if err != nil {
		log.Printf("Scheduler Klassensatz: Einstellungen nicht lesbar, Lauf übersprungen: %v", err)
		return
	}
	tage := repository.TageOderVorgabe(einst.AnliegenTage, repository.AnliegenTageVorgabe)

	// Dieselbe Einstellung wie bei den Anliegen; eine Frist von 0 schaltet auch hier ab.
	geloescht, err := repository.LoescheErledigteKlassensatzReservierungen(ctx, z.pool, tage)
	if err != nil {
		log.Printf("Scheduler Klassensatz: Löschen fehlgeschlagen: %v", err)
		return
	}
	if geloescht == 0 {
		return
	}
	log.Printf("Scheduler Klassensatz: %d erledigte Reservierungen nach %d Tagen gelöscht", geloescht, tage)

	if err := z.protokollAbfragen.ProtokolliereSystemAktion(ctx, "klassensatz_reservierungen", "DELETE",
		"DSGVO: erledigte Klassensatz-Reservierungen nach Frist gelöscht",
		map[string]any{"geloescht": geloescht, "tage": tage}); err != nil {
		log.Printf("Scheduler Klassensatz: Audit-Eintrag fehlgeschlagen: %v", err)
	}
}
