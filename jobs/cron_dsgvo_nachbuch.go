package jobs

import (
	"context"
	"log"
	"time"

	"bibliothek/repository"
)

// Quittierte Nachbuch-Meldungen (Migration 117) tragen Ausleiher und Vorbesitzer mit
// Namen. Erledigt sind sie mit dem Quittieren; danach halten sie nur noch fest, was
// jemand schon gesehen hat. Sie fallen nach der Lesehistorie-Frist, höchstens nach
// 30 Tagen (entschieden am 13.09.2026). Offene Meldungen haben keine Frist — sie
// sind ausstehende Arbeit und werden vom Wächter der Betriebsbereitschaft genannt.
//
// Das Prädikat kommt aus repository/loeschfristen.go — DERSELBE String wie im
// Rückstands-Wächter (loeschrueckstand.go), nur mit anderer Kulanz.

// RunNachbuchMeldungenBefristung löscht quittierte Meldungen nach Ablauf der Frist.
func (s *Scheduler) RunNachbuchMeldungenBefristung() {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	einst, err := repository.NewSystemSettingsRepository(s.db).GetSettings(ctx)
	if err != nil {
		log.Printf("Scheduler Nachbuch-Meldungen: Einstellungen nicht lesbar, Lauf übersprungen: %v", err)
		return
	}
	tage := repository.NachbuchMeldungenTage(einst)
	bedingung := repository.PredikatNachbuchMeldungen(tage, repository.KulanzJob)
	tag, err := s.db.Exec(ctx, `DELETE FROM nachbuch_meldungen WHERE `+bedingung.Where, bedingung.Args...)
	if err != nil {
		log.Printf("Scheduler Nachbuch-Meldungen: Löschen fehlgeschlagen: %v", err)
		return
	}
	geloescht := tag.RowsAffected()
	if geloescht == 0 {
		return
	}
	log.Printf("Scheduler Nachbuch-Meldungen: %d quittierte Meldungen nach %d Tagen gelöscht", geloescht, tage)
	if err := s.auditRepo.LogSystemAktion(ctx, "nachbuch_meldungen", "DELETE",
		"DSGVO: quittierte Nachbuch-Meldungen nach Frist gelöscht",
		map[string]any{"geloescht": geloescht, "tage": tage}); err != nil {
		log.Printf("Scheduler Nachbuch-Meldungen: Audit-Eintrag fehlgeschlagen: %v", err)
	}
}
