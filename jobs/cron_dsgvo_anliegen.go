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
	if tage <= 0 {
		// 0 heißt "aus" — eine bewusste Entscheidung der Schule, kein Fehler.
		return
	}

	// erledigt_am IS NOT NULL ist die eigentliche Bedingung: Ein offenes Anliegen hat
	// keine Frist. Die Rechnung läuft über den Erledigungszeitpunkt, nicht über das
	// Anlegen — ein Wunsch, der ein Jahr lang offen lag, verschwindet nicht am Tag
	// seiner Erledigung.
	// make_interval statt Text-Verkettung: `$1 || ' days'` zwingt Postgres, den
	// Parameter als text zu lesen — pgx schickt ihn als int4, und der Lauf stirbt an
	// "cannot find encode plan". Dieselbe Form benutzt die Lesehistorie-Befristung.
	tag, err := s.db.Exec(ctx, `
		DELETE FROM lehrer_anliegen
		WHERE erledigt_am IS NOT NULL
		  AND erledigt_am < now() - make_interval(days => $1)
	`, tage)
	if err != nil {
		log.Printf("Scheduler Anliegen: Löschen fehlgeschlagen: %v", err)
		return
	}

	geloescht := tag.RowsAffected()
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
