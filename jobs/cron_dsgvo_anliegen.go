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

	// Das Prädikat kommt aus repository/loeschfristen.go — DERSELBE String, den der
	// Rückstands-Wächter benutzt (loeschrueckstand.go). Bis zum 31.08.2026 war dieser
	// Job der einzige der sechs Löschroutinen, der seine Bedingung selbst hinschrieb:
	// Genau die Bauform, gegen die der Dateikopf dort geschrieben ist („Der Wächter
	// beruhigt, während der Job schläft"). Beide Fassungen stimmten nur zufällig
	// überein; eine Ausnahme im Prädikat hätte der Job ignoriert, und der Wächter
	// hätte dazu weiter „0 Rückstand" gemeldet.
	bedingung := repository.PredikatAnliegen(tage, repository.KulanzJob)
	tag, err := s.db.Exec(ctx, `DELETE FROM lehrer_anliegen WHERE `+bedingung.Where, bedingung.Args...)
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
	if tage <= 0 {
		// 0 heißt "aus" — dieselbe Einstellung wie bei den Anliegen.
		return
	}

	bedingung := repository.PredikatKlassensatzReservierungen(tage, repository.KulanzJob)
	tag, err := s.db.Exec(ctx, `DELETE FROM klassensatz_reservierungen WHERE `+bedingung.Where, bedingung.Args...)
	if err != nil {
		log.Printf("Scheduler Klassensatz: Löschen fehlgeschlagen: %v", err)
		return
	}

	geloescht := tag.RowsAffected()
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
