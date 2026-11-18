package jobs

import (
	"context"
	"log/slog"
	"time"

	"bibliothek/internal/pkg/logger"
	"bibliothek/internal/repository"
)

// ── Gelöschte Leser endgültig löschen (180 Tage im Papierkorb) ───────────────
//
// Nach der Frist ist nichts mehr wiederherzustellen. Einen Schüler hat die Anonymisierung derselben
// Nacht schon geleert (PraedikatAnonymisierung); einen Kollegen darf sie nicht anonymisieren
// (chk_leser_nur_schueler_werden_abgaenger), sein Zugangskonto ging schon beim Löschen
// (LegeLeserInPapierkorb). Beide fallen hier ganz.

// fuehrePapierkorbLoeschungAus löscht Leser endgültig, die seit 180 Tagen im Papierkorb
// liegen. LoescheAusDemPapierkorb stellt dieselbe Frage wie die Liste ein zweites Mal, in
// der Transaktion der Löschung; wer dazwischen nicht mehr löschreif ist, bleibt.
func (z *Zeitplan) fuehrePapierkorbLoeschungAus(ctx context.Context) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()

	// Die Kennungen sind vorab gelesen und die Zeilen geschlossen, bevor die Löschungen je
	// eine eigene Transaktion öffnen.
	ids, err := repository.LadeLoeschreifeImPapierkorb(ctx, z.pool)
	if err != nil {
		slog.Error("Scheduler Papierkorb: Leser konnten nicht gelesen werden", "fehler", err)
		return
	}
	if len(ids) == 0 {
		return
	}

	geloescht := 0
	var fehlschlaege []string
	for _, id := range ids {
		// Das Protokoll nennt SYSTEM als Akteur statt einer Person.
		weg, err := z.leserLoeschung.LoescheAusDemPapierkorb(ctx, id)
		if err != nil {
			slog.Error("Scheduler Papierkorb: Leser nicht gelöscht", "leser", logger.OhneUmbruch(id), "fehler", err)
			fehlschlaege = append(fehlschlaege, id)
			continue
		}
		if !weg {
			slog.Info("Scheduler Papierkorb: Leser ist nicht mehr löschreif, bleibt", "leser", logger.OhneUmbruch(id))
			continue
		}
		geloescht++
	}

	if err := z.protokollAbfragen.ProtokolliereSystemAktion(ctx, "schueler", "BATCH_DELETE",
		"DSGVO: gelöschte Leser nach 180 Tagen im Papierkorb endgültig gelöscht",
		map[string]any{
			"geloescht":    geloescht,
			"fehlschlaege": len(fehlschlaege),
			"tage":         repository.PapierkorbTage,
		}); err != nil {
		slog.Error("Scheduler Papierkorb: Audit-Eintrag fehlgeschlagen", "fehler", err)
	}
	slog.Info("Scheduler Papierkorb: Leser endgültig gelöscht", "geloescht", geloescht, "fehlschlaege", len(fehlschlaege))
}
