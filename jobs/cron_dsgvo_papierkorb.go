package jobs

import (
	"context"
	"log"
	"time"

	"bibliothek/pkg/logger"
	"bibliothek/repository"
)

// ── Gelöschte Kollegen endgültig löschen (180 Tage im Papierkorb) ──────────────
//
// Die Anonymisierung nimmt nur Schüler (PraedikatAnonymisierung), und einen Kollegen darf sie
// nicht anonymisieren (chk_leser_nur_schueler_werden_abgaenger). Ein gelöschter Kollege wird
// deshalb nach derselben Frist wie ein Schüler endgültig gelöscht. Sein Zugangskonto ist schon
// beim Löschen gegangen (LegeLeserInPapierkorb).

// FuehrePapierkorbLoeschungAus löscht Kollegen endgültig, die seit 180 Tagen im Papierkorb
// liegen. Derselbe Weg wie „Endgültig löschen" von Hand: LoescheLeserEndgueltig prüft Papierkorb
// und offene Vorgänge ein zweites Mal, in der Transaktion der Löschung.
func (z *Zeitplan) FuehrePapierkorbLoeschungAus() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	// Die Kennungen sind vorab gelesen und die Zeilen geschlossen, bevor die Löschungen je
	// eine eigene Transaktion öffnen.
	ids, err := repository.LadeKollegenImPapierkorb(ctx, z.pool)
	if err != nil {
		log.Printf("Scheduler Papierkorb: Kollegen konnten nicht gelesen werden: %v", err)
		return
	}
	if len(ids) == 0 {
		return
	}

	geloescht := 0
	var fehlschlaege []string
	for _, id := range ids {
		// Leerer Bearbeiter: Das Protokoll nennt SYSTEM als Akteur statt einer Person.
		if err := z.leserLoeschung.LoescheLeserEndgueltig(ctx, id, ""); err != nil {
			log.Printf("Scheduler Papierkorb: Leser %s nicht gelöscht: %v", logger.OhneUmbruch(id), err)
			fehlschlaege = append(fehlschlaege, id)
			continue
		}
		geloescht++
	}

	if err := z.protokollAbfragen.ProtokolliereSystemAktion(ctx, "schueler", "BATCH_DELETE",
		"DSGVO: gelöschte Kollegen nach 180 Tagen im Papierkorb endgültig gelöscht",
		map[string]any{
			"geloescht":    geloescht,
			"fehlschlaege": len(fehlschlaege),
			"tage":         repository.AnonymisierungPapierkorbTage,
		}); err != nil {
		log.Printf("Scheduler Papierkorb: Audit-Eintrag fehlgeschlagen: %v", err)
	}
	log.Printf("Scheduler Papierkorb: %d Kollegen endgültig gelöscht, %d Fehlschläge", geloescht, len(fehlschlaege))
}
