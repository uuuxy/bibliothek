package jobs

import (
	"context"
	"log"
	"time"

	"bibliothek/repository"
)

// Quittierte Nachbuch-Meldungen tragen Ausleiher und Vorbesitzer mit Namen. Erledigt sind sie
// mit dem Quittieren; danach halten sie nur noch fest, was jemand schon gesehen hat. Sie fallen
// nach der Frist der Lesehistorie, höchstens nach 30 Tagen. Offene Meldungen haben keine Frist:
// Sie sind ausstehende Arbeit, und der Wächter der Betriebsbereitschaft nennt sie.
//
// Anweisung und Bedingung stehen in repository/: dieselbe Bedingung wie im Wächter des
// Rückstands (loeschrueckstand.go), nur mit anderer Kulanz.

// FuehreNachbuchMeldungenBefristungAus löscht quittierte Meldungen nach der Frist.
func (z *Zeitplan) FuehreNachbuchMeldungenBefristungAus() {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	einst, err := repository.NewEinstellungsAbfragen(z.pool).Lade(ctx)
	if err != nil {
		log.Printf("Scheduler Nachbuch-Meldungen: Einstellungen nicht lesbar, Lauf übersprungen: %v", err)
		return
	}
	tage := repository.NachbuchMeldungenTage(einst)
	geloescht, err := repository.LoescheQuittierteNachbuchMeldungen(ctx, z.pool, tage)
	if err != nil {
		log.Printf("Scheduler Nachbuch-Meldungen: Löschen fehlgeschlagen: %v", err)
		return
	}
	if geloescht == 0 {
		return
	}
	log.Printf("Scheduler Nachbuch-Meldungen: %d quittierte Meldungen nach %d Tagen gelöscht", geloescht, tage)
	if err := z.protokollAbfragen.ProtokolliereSystemAktion(ctx, "nachbuch_meldungen", "DELETE",
		"DSGVO: quittierte Nachbuch-Meldungen nach Frist gelöscht",
		map[string]any{"geloescht": geloescht, "tage": tage}); err != nil {
		log.Printf("Scheduler Nachbuch-Meldungen: Audit-Eintrag fehlgeschlagen: %v", err)
	}
}
