package repository

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"

	"bibliothek/pkg/closeutil"
)

// Die Anweisungen des Bestands-Imports: Eine Liste nennt je Zeile einen Titel und die Nummer
// eines Exemplars, das schon im Regal steht. Welche Zeile zu welchem Titel gehört, entscheidet
// internal/service (import_dynamic.go) über LadeTitelBestand; hier steht, was geschrieben wird.

// LegeImportTitelAn legt die Titel an, die der Bestands-Import im Bestand nicht gefunden hat,
// und liefert ihre Kennungen in der Reihenfolge der Eingabe. Unbekannte Fächer registriert die
// Funktion vorher in der Systematik und schreibt jedes Fach in der dort geführten Schreibweise:
// subject ist ein Fremdschlüssel.
func LegeImportTitelAn(ctx context.Context, tx pgx.Tx, titel []BookTitle) ([]string, error) {
	if len(titel) == 0 {
		return nil, nil
	}
	// Vor dem Sammelauftrag: Danach ist die Verbindung belegt, bis er geschlossen ist.
	kanonisch, err := StelleFaecherSicher(ctx, tx, faecherDerTitel(titel))
	if err != nil {
		return nil, err
	}

	batch := &pgx.Batch{}
	for _, t := range titel {
		t.Fach = kanonisch[t.Fach]
		reiheTitelAnlageEin(batch, t)
	}

	br := tx.SendBatch(ctx, batch)
	ids := make([]string, 0, len(titel))
	for range titel {
		var id string
		if err := br.QueryRow().Scan(&id); err != nil {
			closeutil.LogClose(br, "title insert batch")
			return nil, fmt.Errorf("failed to insert title batch: %w", err)
		}
		ids = append(ids, id)
	}
	if err := br.Close(); err != nil {
		return nil, fmt.Errorf("failed to close title insert batch: %w", err)
	}
	return ids, nil
}

// ImportSignatur ist die Signatur, die der Bestands-Import an einem Titel einträgt.
type ImportSignatur struct {
	TitelID  string
	Signatur string
	// Lernmittel sagt, dass die Signatur die Kennung der Lernmittel trägt.
	Lernmittel bool
}

// SetzeImportSignaturen trägt die Signaturen der Datei an ihren Titeln ein. Trägt eine die
// Kennung der Lernmittel, wird der Titel zugleich als Lernmittel markiert; ein gesetztes
// Kennzeichen nimmt der Import nie zurück. Einen Eintrag ohne Signatur übergeht die Funktion:
// Das Etikett am Buchrücken gilt, eine Datei ohne Angabe überschreibt es nicht.
func SetzeImportSignaturen(ctx context.Context, tx pgx.Tx, signaturen []ImportSignatur) error {
	batch := &pgx.Batch{}
	for _, s := range signaturen {
		if s.Signatur == "" {
			continue
		}
		batch.Queue(
			"UPDATE buecher_titel SET signatur = $2, ist_lernmittel = ist_lernmittel OR $3, aktualisiert_am = CURRENT_TIMESTAMP WHERE id = $1",
			s.TitelID, s.Signatur, s.Lernmittel,
		)
	}
	if batch.Len() == 0 {
		return nil
	}
	br := tx.SendBatch(ctx, batch)
	for range batch.Len() {
		if _, err := br.Exec(); err != nil {
			closeutil.LogClose(br, "signatur update batch")
			return fmt.Errorf("failed to update signatur batch: %w", err)
		}
	}
	return br.Close()
}

// ImportExemplar ist ein Exemplar, das der Bestands-Import mit seiner Nummer übernimmt.
type ImportExemplar struct {
	TitelID       string
	Barcode       string
	IstAusleihbar bool
	ZustandNotiz  string
}

// LegeImportExemplareAn hängt die Exemplare an ihre Titel und zählt, wie viele neu sind und
// wie viele es unter ihrer Nummer schon gab; die bleiben, wie sie sind. Die neuen gelten als
// etikettiert: Der Import übernimmt Bestand, der sein Etikett trägt, und ohne das stünde jedes
// im Druck-Center als offen. Das Zugangsdatum kommt aus der Vorgabe der Spalte, dem Kalendertag
// der Schule; den Standort erbt ein Exemplar von den Exemplaren seines Titels.
func LegeImportExemplareAn(ctx context.Context, tx pgx.Tx, exemplare []ImportExemplar) (angelegt, vorhanden int, err error) {
	if len(exemplare) == 0 {
		return 0, 0, nil
	}

	anlegen := `
		INSERT INTO buecher_exemplare (titel_id, barcode_id, ist_ausleihbar, zustand_notiz, etikett_gedruckt, standort)
		VALUES ($1, $2, $3, NULLIF($4, ''), true, ` + SQLGeerbterStandort("$1::uuid") + `)
		ON CONFLICT (barcode_id) DO NOTHING
	`
	batch := &pgx.Batch{}
	for _, e := range exemplare {
		batch.Queue(anlegen, e.TitelID, e.Barcode, e.IstAusleihbar, e.ZustandNotiz)
	}

	br := tx.SendBatch(ctx, batch)
	for _, e := range exemplare {
		tag, fehler := br.Exec()
		if fehler != nil {
			closeutil.LogClose(br, "copy insert batch")
			return 0, 0, fmt.Errorf("exemplar %q zu titel %s anlegen: %w", e.Barcode, e.TitelID, fehler)
		}
		if tag.RowsAffected() == 1 {
			angelegt++
		} else {
			vorhanden++
		}
	}
	if err := br.Close(); err != nil {
		return 0, 0, fmt.Errorf("failed to close copy insert batch: %w", err)
	}
	return angelegt, vorhanden, nil
}
