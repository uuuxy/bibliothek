package littera

import (
	"context"
	"fmt"
	"slices"

	"bibliothek/internal/uebernahme"
	"bibliothek/repository"

	"github.com/jackc/pgx/v5"
)

// SchlagwortBericht ist die Bilanz der Schlagworte (docs/OFFEN.md 4.20, Stufe 2).
type SchlagwortBericht struct {
	QuellTitel       int // Titel mit mindestens einer Zuordnung in Littera
	QuellZuordnungen int // Zeilen in Schlag_zuord
	OhneWort         int // Zuordnungen auf ein Wort, das es in der Tabelle Schlagworte nicht gibt
	OhneTitel        int // Titel mit Schlagworten, die es im Export nicht gibt oder die der Bestand nicht übernahm
	Weggelassen      int // leere oder zu lange Wörter
	Gekuerzt         int // Titel mit mehr als repository.SchlagworteJeTitelMax Wörtern
	Uebersprungen    int // Titel, deren Schlagworte am Schreibpfad scheiterten
	Titel            int // Titel mit geschriebenen Schlagworten
	Zuordnungen      int // gemeldet: geschrieben, nach dem Zusammenlegen gleicher Wörter

	// Die Verweise werden gezählt, nicht übernommen (schlagworte.go).
	VerweisWoerter, VerweisZuordnungen int

	// Die Interessenkreise kommen als Schlagworte mit (schlagworte.go, entschieden am
	// 30.09.2026); gezählt wie die Schlagworte an der Quelle. Titel, Zuordnungen und der
	// Abgleich oben schließen sie ein.
	KreisQuellTitel, KreisQuellZuordnungen, KreisOhneWort int

	IstZuordnungen int // tatsächlicher Zuwachs in titel_schlagworte
	IstWoerter     int // tatsächlicher Zuwachs in schlagworte
	AbgleichOK     bool
}

// schlagwortAuftrag sind die Wörter eines übernommenen Titels.
type schlagwortAuftrag struct {
	litteraID, titelID string
	woerter            []string
}

// SchreibeSchlagworte schreibt die Schlagworte der übernommenen Titel über den einen
// Schreibpfad der Anwendung (repository.SetzeSchlagworte): dieselben Regeln wie im
// Buchformular, und ein Wort, das es in anderer Groß- und Kleinschreibung schon gibt, wird
// nicht neu angelegt. Die Titel laufen in der Reihenfolge des Exports, damit ein Lauf wie der
// andere dieselbe Schreibweise behält.
//
// Die atomare Einheit ist der Titel, im eigenen Savepoint wie im Bestand: Scheitert seine
// Liste, fehlen nur seine Schlagworte, und die Warnung nennt ihn. Das Buch selbst ist dann
// schon übernommen — ein Schlagwort darf nie ein Buch kosten.
func (s *Schreiber) SchreibeSchlagworte(ctx context.Context, ab *Altbestand, bestand BestandBericht) (SchlagwortBericht, error) {
	q, kreise := ab.Schlagworte, ab.Interessenkreise
	b := SchlagwortBericht{
		QuellTitel: len(q.JeTitel), QuellZuordnungen: q.Zuordnungen, OhneWort: q.OhneWort,
		VerweisWoerter: q.VerweisWoerter, VerweisZuordnungen: q.VerweisZuordnungen,
		KreisQuellTitel: len(kreise.JeTitel), KreisQuellZuordnungen: kreise.Zuordnungen, KreisOhneWort: kreise.OhneWort,
	}
	if q.VerweisWoerter+q.VerweisZuordnungen > 0 {
		s.prot.Warnung("", "", fmt.Sprintf("Verweise zu Schlagworten nicht übernommen – %d Wörter, %d Zuordnungen "+
			"(ihre Form ist an echten Daten noch nicht geprüft, docs/OFFEN.md 4.20)", q.VerweisWoerter, q.VerweisZuordnungen))
	}

	vorherZ, vorherW, err := s.zaehleSchlagworte(ctx)
	if err != nil {
		return b, err
	}

	// Gezählt über die Zuordnungen, nicht über die Titel des Exports: Eine Zuordnung zu einem
	// Titel, den es dort gar nicht gibt, fiele sonst nirgends auf. Ein Titel mit Schlagworten
	// und Interessenkreisen zählt einmal.
	ohneTitel := map[string]bool{}
	for _, jeTitel := range []map[string][]string{q.JeTitel, kreise.JeTitel} {
		for litteraID := range jeTitel {
			if _, uebernommen := bestand.TitelIDs[litteraID]; !uebernommen {
				ohneTitel[litteraID] = true
			}
		}
	}
	b.OhneTitel = len(ohneTitel)
	var auftraege []schlagwortAuftrag
	for _, t := range ab.Titel {
		// Die Interessenkreise hinter den Schlagworten; ein gleichlautendes Wort („U plus")
		// zählt einmal (repository.SchlagworteAusFremddaten).
		roh := append(slices.Clone(q.JeTitel[t.ID]), kreise.JeTitel[t.ID]...)
		titelID, uebernommen := bestand.TitelIDs[t.ID]
		if len(roh) == 0 || !uebernommen {
			continue
		}
		if woerter := s.bereiteSchlagworteVor(t, roh, &b); len(woerter) > 0 {
			auftraege = append(auftraege, schlagwortAuftrag{t.ID, titelID, woerter})
		}
	}
	for i := 0; i < len(auftraege); i += s.opt.BatchGroesse {
		ende := min(i+s.opt.BatchGroesse, len(auftraege))
		if err := s.schlagworteEinBatch(ctx, auftraege[i:ende], &b); err != nil {
			return b, fmt.Errorf("abgebrochen ab Titel %s: %w", auftraege[i].litteraID, err)
		}
	}

	nachherZ, nachherW, err := s.zaehleSchlagworte(ctx)
	if err != nil {
		return b, fmt.Errorf("der Abgleich nach der Übernahme schlug fehl: %w", err)
	}
	b.IstZuordnungen, b.IstWoerter = nachherZ-vorherZ, nachherW-vorherW
	b.AbgleichOK = b.IstZuordnungen == b.Zuordnungen
	return b, nil
}

// bereiteSchlagworteVor bringt die Wörter eines Titels in die Form des Schreibpfads
// (repository.SchlagworteAusFremddaten, dieselbe Regel wie im Katalogisat-Import) und schreibt
// ins Protokoll, was dabei wegfiel.
func (s *Schreiber) bereiteSchlagworteVor(t Titel, roh []string, b *SchlagwortBericht) []string {
	woerter, auf := repository.SchlagworteAusFremddaten(roh)
	for range auf.ZuLang {
		s.prot.Warnung(t.ID, t.ISBN, fmt.Sprintf("Schlagwort länger als %d Zeichen – weggelassen",
			repository.SchlagwortMaxZeichen))
	}
	for range auf.Leer {
		s.prot.Warnung(t.ID, t.ISBN, "Schlagwort leer – weggelassen")
	}
	b.Weggelassen += auf.Leer + auf.ZuLang
	if auf.Gekuerzt {
		b.Gekuerzt++
		s.prot.Warnung(t.ID, t.ISBN, fmt.Sprintf("mehr als %d Schlagworte – die ersten %d übernommen",
			repository.SchlagworteJeTitelMax, repository.SchlagworteJeTitelMax))
	}
	return woerter
}

// schlagworteEinBatch schreibt eine Transaktion voller Titel, jeden im eigenen Savepoint.
func (s *Schreiber) schlagworteEinBatch(ctx context.Context, auftraege []schlagwortAuftrag, b *SchlagwortBericht) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("konnte die Transaktion nicht öffnen: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }() //nolint:errcheck

	type ergebnis struct{ zuordnungen int }
	var geschrieben []ergebnis
	for _, a := range auftraege {
		var gespeichert []string
		erg, err := uebernahme.ImSavepoint(ctx, tx, "littera_id="+a.litteraID, func(sp pgx.Tx) error {
			var innerErr error
			gespeichert, innerErr = repository.SetzeSchlagworte(ctx, sp, a.titelID, a.woerter)
			return innerErr
		})
		if err != nil {
			return err
		}
		if !erg.Uebernommen {
			b.Uebersprungen++
			s.prot.Warnung(a.litteraID, "", "Schlagworte nicht übernommen – "+
				uebernahme.BeschreibeFehler(erg.Zurueckgerollt))
			continue
		}
		geschrieben = append(geschrieben, ergebnis{len(gespeichert)})
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("der COMMIT schlug fehl – KEIN Titel dieses Batches trägt seine Schlagworte: %w", err)
	}
	// Erst nach dem COMMIT zählen: Scheitert er, ist nichts davon geschrieben.
	for _, g := range geschrieben {
		if g.zuordnungen > 0 {
			b.Titel++
		}
		b.Zuordnungen += g.zuordnungen
	}
	return nil
}

// zaehleSchlagworte liest die Zuordnungen und die Wörter für den Abgleich.
func (s *Schreiber) zaehleSchlagworte(ctx context.Context) (zuordnungen, woerter int, err error) {
	err = s.pool.QueryRow(ctx,
		`SELECT (SELECT count(*) FROM titel_schlagworte), (SELECT count(*) FROM schlagworte)`).
		Scan(&zuordnungen, &woerter)
	if err != nil {
		return 0, 0, fmt.Errorf("eine Zählung der Schlagworte schlug fehl: %w", err)
	}
	return zuordnungen, woerter, nil
}
