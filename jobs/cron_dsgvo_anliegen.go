package jobs

import (
	"context"
	"log"
	"time"

	"bibliothek/repository"
)

// ── Erledigte Anliegen befristen ─────────────────────────────────────────────
//
// `lehrer_anliegen` (Wünsche und Meldungen aus dem Kollegiums-Portal) hatte bis zum
// 23.08.2026 keinen Löschpfad — keinen Endpunkt, keinen Job, keine Frist. Die Zeile
// trägt den Freitext der Lehrkraft, ihre Klassenangabe und über den Fremdschlüssel
// ihren Namen. Erledigt ist ihr Zweck erreicht.
//
// Das ist dieselbe Abwägung wie bei den Lesehistorie-Fristen, nur für
// Beschäftigtendaten: so lange wie nötig, nicht so lange wie möglich. Ein Jahr deckt den
// Blick zurück über ein Schuljahr ("war der Titel schon einmal gewünscht?").
//
// Angefasst werden AUSSCHLIESSLICH erledigte Anliegen. Ein offener Wunsch ist eine
// laufende Sache und hat keine Frist — er verschwindet erst, wenn ihn jemand erledigt.
// Gefunden beim Raster-Durchgang vom 23.08.2026 (Frage 8, Lebenszyklus).

// RunAnliegenBefristung löscht erledigte Anliegen nach Ablauf der Frist.
func (s *Scheduler) RunAnliegenBefristung() {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	einst, err := repository.NewSystemSettingsRepository(s.db).GetSettings(ctx)
	if err != nil {
		log.Printf("Scheduler Anliegen: Einstellungen nicht lesbar, Lauf übersprungen: %v", err)
		return
	}
	tage := repository.TageOderStandard(einst.AnliegenTage, repository.StandardAnliegenTage)

	// Anweisung und Bedingung stehen in repository/: dieselbe Bedingung, die der Wächter des
	// Rückstands zählt. Eine Frist von 0 schaltet die Befristung ab; dann löscht die Abfrage
	// nichts, und der Lauf endet hier ohne Eintrag.
	geloescht, err := repository.LoescheErledigteAnliegen(ctx, s.db, tage)
	if err != nil {
		log.Printf("Scheduler Anliegen: Löschen fehlgeschlagen: %v", err)
		return
	}
	if geloescht == 0 {
		return
	}
	log.Printf("Scheduler Anliegen: %d erledigte Anliegen nach %d Tagen gelöscht", geloescht, tage)

	if err := s.auditRepo.LogSystemAktion(ctx, "lehrer_anliegen", "DELETE",
		"DSGVO: erledigte Anliegen nach Frist gelöscht",
		map[string]any{"geloescht": geloescht, "tage": tage}); err != nil {
		log.Printf("Scheduler Anliegen: Audit-Eintrag fehlgeschlagen: %v", err)
	}
}

// ── Erledigte Klassensatz-Reservierungen befristen ───────────────────────────
//
// Reservierungen aus dem Kollegiums-Portal hatten bis zum 29.09.2026 keinen Löschpfad; sie
// tragen die Klasse, eine Notiz und über angefordert_von das Konto der Lehrkraft. Entschieden
// am 28.09.2026 (docs/OFFEN.md 5.19): dieselbe Frist wie die erledigten Anliegen, dieselbe
// Einstellung. Offene Reservierungen bleiben.

// RunKlassensatzBefristung löscht erledigte Klassensatz-Reservierungen nach Ablauf der Frist.
func (s *Scheduler) RunKlassensatzBefristung() {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	einst, err := repository.NewSystemSettingsRepository(s.db).GetSettings(ctx)
	if err != nil {
		log.Printf("Scheduler Klassensatz: Einstellungen nicht lesbar, Lauf übersprungen: %v", err)
		return
	}
	tage := repository.TageOderStandard(einst.AnliegenTage, repository.StandardAnliegenTage)

	// Dieselbe Einstellung wie bei den Anliegen; eine Frist von 0 schaltet auch hier ab.
	geloescht, err := repository.LoescheErledigteKlassensatzReservierungen(ctx, s.db, tage)
	if err != nil {
		log.Printf("Scheduler Klassensatz: Löschen fehlgeschlagen: %v", err)
		return
	}
	if geloescht == 0 {
		return
	}
	log.Printf("Scheduler Klassensatz: %d erledigte Reservierungen nach %d Tagen gelöscht", geloescht, tage)

	if err := s.auditRepo.LogSystemAktion(ctx, "klassensatz_reservierungen", "DELETE",
		"DSGVO: erledigte Klassensatz-Reservierungen nach Frist gelöscht",
		map[string]any{"geloescht": geloescht, "tage": tage}); err != nil {
		log.Printf("Scheduler Klassensatz: Audit-Eintrag fehlgeschlagen: %v", err)
	}
}
