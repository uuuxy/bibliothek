package repository

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"

	"bibliothek/pkg/closeutil"
)

// Die Anweisungen des Bestands-Imports: Eine Liste nennt je Zeile einen Titel und die Nummer
// eines Exemplars, das schon im Regal steht. Welche Zeile zu welchem Titel gehört, entscheidet
// internal/service (import_zeilen.go) über den Titelbestand (TitelBestand.TitelZu); hier steht,
// was geschrieben wird.

// LegeImportTitelAn legt die Titel an, die der Bestands-Import im Bestand nicht gefunden hat,
// und liefert ihre Kennungen in der Reihenfolge der Eingabe. Unbekannte Fächer registriert die
// Funktion vorher in der Systematik und schreibt jedes Fach in der dort geführten Schreibweise:
// subject ist ein Fremdschlüssel.
func LegeImportTitelAn(ctx context.Context, tx pgx.Tx, titel []Titel) ([]string, error) {
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
			closeutil.Schliesse(br, "title insert batch")
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
			closeutil.Schliesse(br, "signatur update batch")
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
	// Zeile ist die Zeile der Datei, für Meldungen.
	Zeile int
}

// ImportNummerIstAusweisFehler meldet die Exemplare, deren Buchnummer der Ausweis eines aktiven
// Lesers ist. Eine Nummer ist entweder ein Buch oder ein Ausweis (nummer_ist_buch_oder_ausweis);
// importiert wird dann nichts. Die Tür nennt die Zeilen, damit die Datei in einem Gang berichtigt
// werden kann.
type ImportNummerIstAusweisFehler struct {
	Exemplare []ImportExemplar
}

func (f *ImportNummerIstAusweisFehler) Error() string {
	return fmt.Sprintf("%d buchnummern der datei sind leserausweise", len(f.Exemplare))
}

// pruefeImportNummernGegenAusweise lehnt die Exemplare ab, deren Nummer der Ausweis eines aktiven
// Lesers ist. Der Wächter der Datenbank wiese das erste von ihnen sonst mitten im Lauf ab, ohne
// Zeile.
func pruefeImportNummernGegenAusweise(ctx context.Context, tx pgx.Tx, exemplare []ImportExemplar) error {
	nummern := make([]string, 0, len(exemplare))
	for _, e := range exemplare {
		nummern = append(nummern, e.Barcode)
	}
	rows, err := tx.Query(ctx, `SELECT barcode_id FROM leser WHERE barcode_id = ANY($1) AND deleted_at IS NULL`, nummern)
	if err != nil {
		return fmt.Errorf("ausweisnummern prüfen: %w", err)
	}
	vergeben, err := pgx.CollectRows(rows, pgx.RowTo[string])
	if err != nil {
		return fmt.Errorf("ausweisnummern prüfen: %w", err)
	}
	if len(vergeben) == 0 {
		return nil
	}
	ausweis := make(map[string]bool, len(vergeben))
	for _, nummer := range vergeben {
		ausweis[nummer] = true
	}
	fehler := &ImportNummerIstAusweisFehler{}
	for _, e := range exemplare {
		if ausweis[e.Barcode] {
			fehler.Exemplare = append(fehler.Exemplare, e)
		}
	}
	return fehler
}

// LegeImportExemplareAn hängt die Exemplare an ihre Titel und zählt, wie viele neu sind und
// wie viele es unter ihrer Nummer schon gab; die bleiben, wie sie sind. Ist eine Nummer der
// Ausweis eines Lesers, legt es nichts an (ImportNummerIstAusweisFehler). Die neuen gelten als
// etikettiert: Der Import übernimmt Bestand, der sein Etikett trägt, und ohne das stünde jedes
// im Druck-Center als offen. Das Zugangsdatum kommt aus der Vorgabe der Spalte, dem Kalendertag
// der Schule; den Standort erbt ein Exemplar von den Exemplaren seines Titels.
//
// Ein Exemplar mit LMF-Nummer bekommt eine kurze Nummer (kurzeNummernFuerLMF), die LMF-Nummer
// steht in lmf_barcode: Sein Etikett scannt weiter, ein neues trägt die kurze Nummer. Gezogen
// werden die kurzen Nummern erst, wenn die übrigen Exemplare der Datei angelegt sind: Die Prüfung
// gegen den Bestand (zieheFreieExemplarBarcodes) sieht dann auch deren Nummern, und kein
// LMF-Exemplar bekommt die Nummer eines Buchs weiter unten in der Datei.
func LegeImportExemplareAn(ctx context.Context, tx pgx.Tx, exemplare []ImportExemplar) (angelegt, vorhanden int, err error) {
	if len(exemplare) == 0 {
		return 0, 0, nil
	}
	if err := pruefeImportNummernGegenAusweise(ctx, tx, exemplare); err != nil {
		return 0, 0, err
	}
	var mitNummer, mitLMF []ImportExemplar
	for _, e := range exemplare {
		if _, istLMF := lmfNummer(e.Barcode); istLMF {
			mitLMF = append(mitLMF, e)
		} else {
			mitNummer = append(mitNummer, e)
		}
	}

	anlegen := `
		INSERT INTO buecher_exemplare (titel_id, barcode_id, ist_ausleihbar, zustand_notiz, etikett_gedruckt, standort)
		VALUES ($1, $2, $3, NULLIF($4, ''), true, ` + sqlGeerbterStandort("$1::uuid") + `)
		ON CONFLICT (barcode_id) DO NOTHING
	`
	batch := &pgx.Batch{}
	for _, e := range mitNummer {
		batch.Queue(anlegen, e.TitelID, e.Barcode, e.IstAusleihbar, e.ZustandNotiz)
	}
	angelegt, vorhanden, err = legeEingereihteAn(ctx, tx, batch, mitNummer)
	if err != nil {
		return 0, 0, err
	}
	neu, schonDa, err := legeLMFExemplareAn(ctx, tx, mitLMF)
	if err != nil {
		return 0, 0, err
	}
	return angelegt + neu, vorhanden + schonDa, nil
}

// legeLMFExemplareAn legt die Exemplare mit LMF-Nummer unter einer kurzen Nummer an. Trägt schon
// ein Exemplar die LMF-Nummer, zählt es als vorhanden und verbraucht keine Nummer.
func legeLMFExemplareAn(ctx context.Context, tx pgx.Tx, exemplare []ImportExemplar) (angelegt, vorhanden int, err error) {
	kurz, err := kurzeNummernFuerLMF(ctx, tx, exemplare)
	if err != nil {
		return 0, 0, err
	}
	anlegenMitLMF := `
		INSERT INTO buecher_exemplare (titel_id, barcode_id, lmf_barcode, ist_ausleihbar, zustand_notiz, etikett_gedruckt, standort)
		VALUES ($1, $2, $3, $4, NULLIF($5, ''), true, ` + sqlGeerbterStandort("$1::uuid") + `)
		ON CONFLICT (lmf_barcode) DO NOTHING
	`
	batch := &pgx.Batch{}
	var eingereiht []ImportExemplar
	for _, e := range exemplare {
		lmf, _ := lmfNummer(e.Barcode)
		if kurz[lmf] == "" {
			vorhanden++
			continue
		}
		// Nennt die Datei dieselbe LMF-Nummer noch einmal, greift ON CONFLICT (lmf_barcode).
		batch.Queue(anlegenMitLMF, e.TitelID, kurz[lmf], lmf, e.IstAusleihbar, e.ZustandNotiz)
		eingereiht = append(eingereiht, e)
	}
	angelegt, schonDa, err := legeEingereihteAn(ctx, tx, batch, eingereiht)
	return angelegt, vorhanden + schonDa, err
}

// legeEingereihteAn schickt die eingereihten Anlagen, eine je Exemplar in derselben Reihenfolge,
// und zählt, welche neu sind und welche es unter ihrer Nummer schon gab.
func legeEingereihteAn(ctx context.Context, tx pgx.Tx, batch *pgx.Batch, exemplare []ImportExemplar) (angelegt, vorhanden int, err error) {
	if len(exemplare) == 0 {
		return 0, 0, nil
	}
	br := tx.SendBatch(ctx, batch)
	for _, e := range exemplare {
		tag, fehler := br.Exec()
		if fehler != nil {
			closeutil.Schliesse(br, "copy insert batch")
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

// lmfVorsilbe beginnt die Nummer eines LMF-Etiketts (LMF-, ISBN-13, laufende Nummer).
const lmfVorsilbe = "LMF-"

// lmfNummer erkennt die Nummer eines LMF-Etiketts und schreibt die Vorsilbe groß, wie der Scan
// sie liefert (normalisiereScan in frontend/src/lib/scanEinordnen.js).
func lmfNummer(nummer string) (string, bool) {
	if len(nummer) <= len(lmfVorsilbe) || !strings.EqualFold(nummer[:len(lmfVorsilbe)], lmfVorsilbe) {
		return "", false
	}
	return lmfVorsilbe + nummer[len(lmfVorsilbe):], true
}

// kurzeNummernFuerLMF gibt jeder LMF-Nummer der Datei, die noch kein Exemplar trägt, eine kurze
// Nummer aus barcode_seq (zieheFreieExemplarBarcodes). Gezogen wird nur für diese: Ein zweiter
// Lauf derselben Datei verbraucht keine Nummern.
func kurzeNummernFuerLMF(ctx context.Context, tx pgx.Tx, exemplare []ImportExemplar) (map[string]string, error) {
	var lmf []string
	gesehen := map[string]bool{}
	for _, e := range exemplare {
		if n, ok := lmfNummer(e.Barcode); ok && !gesehen[n] {
			gesehen[n] = true
			lmf = append(lmf, n)
		}
	}
	if len(lmf) == 0 {
		return nil, nil
	}

	rows, err := tx.Query(ctx, `
		SELECT lmf_barcode FROM buecher_exemplare WHERE lmf_barcode = ANY($1)
		UNION
		SELECT barcode_id FROM buecher_exemplare WHERE barcode_id = ANY($1)`, lmf)
	if err != nil {
		return nil, fmt.Errorf("lmf-nummern prüfen: %w", err)
	}
	getragen, err := pgx.CollectRows(rows, pgx.RowTo[string])
	if err != nil {
		return nil, fmt.Errorf("lmf-nummern prüfen: %w", err)
	}
	for _, n := range getragen {
		delete(gesehen, n)
	}
	var neu []string
	for _, n := range lmf {
		if gesehen[n] {
			neu = append(neu, n)
		}
	}

	nummern, err := zieheFreieExemplarBarcodes(ctx, tx, len(neu))
	if err != nil {
		return nil, fmt.Errorf("kurze nummern für lmf-exemplare: %w", err)
	}
	kurz := make(map[string]string, len(neu))
	for i, n := range neu {
		kurz[n] = nummern[i]
	}
	return kurz, nil
}
