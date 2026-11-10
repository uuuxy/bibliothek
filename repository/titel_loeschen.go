package repository

import (
	"context"
	"errors"
	"fmt"

	"bibliothek/db"

	"github.com/jackc/pgx/v5"
)

// ErrTitelNichtGefunden meldet einen Titel, den es unter der genannten Kennung nicht gibt.
var ErrTitelNichtGefunden = errors.New("titel nicht gefunden")

// ErrTitelInzwischenVerliehen meldet ein Exemplar, das zwischen der Rückfrage und dem Löschen
// ausgeliehen wurde. Gelöscht wird nur, was der Mensch gesehen und bestätigt hat.
var ErrTitelInzwischenVerliehen = errors.New("exemplar inzwischen verliehen")

// TitelLoeschauftrag sagt, was beim Löschen von Titeln gilt.
type TitelLoeschauftrag struct {
	TitelIDs []string
	// Grund gilt für jedes Exemplar, das mit dem Löschen aus dem Bestand geht
	// (IstAussonderungsgrund); ein schon ausgesondertes behält seinen.
	Grund string
	// Verliehene sind die verliehenen Exemplare, die der Mensch zum Mitlöschen bestätigt hat.
	Verliehene   []string
	BearbeiterID string
}

// TitelLoeschergebnis nennt die Titel, die gelöscht sind, und die, die bleiben.
type TitelLoeschergebnis struct {
	Geloescht []string `json:"geloescht"`
	// Geblieben sind Titel mit einem verliehenen Exemplar, das nicht bestätigt war. Ihre
	// Exemplare im Haus sind mit dem Grund ausgesondert; der Titel bleibt, bis die verliehenen
	// zurück sind.
	Geblieben []string `json:"geblieben"`
}

// LoescheTitel löscht Titel samt allem, was an ihnen hängt: Schadensfälle, Ausleihen, Exemplare,
// Vormerkungen und Reservierungen. Alles läuft in einer Transaktion; ohne sie hinterließe ein
// Absturz zwischen zwei Schritten Forderungen und Ausleihen gelöscht, Titel und Exemplare aber
// erhalten. Es ist der einzige Weg, auf dem ein Titel mit seinen Exemplaren geht.
//
// Ein verliehenes Exemplar fällt nur, wenn der Mensch es bestätigt hat. Dann liegt das Buch bei
// jemandem zu Hause, und das Programm vergisst es; deshalb steht jede dabei abgeräumte Ausleihe
// einzeln im Protokoll (protokolliereOffeneAusleihen), damit die Rückgabe nachzuschlagen bleibt.
// Der Barcode wird frei, neu vergeben wird er nicht (barcode_seq zählt nur vorwärts). Fehlt die
// Bestätigung für ein verliehenes Exemplar, bleibt sein Titel, und nur seine Exemplare im Haus
// werden mit dem Grund ausgesondert. Jedes Exemplar aus dem Bestand behält seine Zeile in den
// Bestandsbüchern (loescheExemplareMitBuchzeile).
func (a *TitelAbfragen) LoescheTitel(ctx context.Context, auftrag TitelLoeschauftrag) (TitelLoeschergebnis, error) {
	ergebnis := TitelLoeschergebnis{Geloescht: []string{}, Geblieben: []string{}}
	if len(auftrag.TitelIDs) == 0 {
		return ergebnis, nil
	}
	tx, err := a.pool.Begin(ctx)
	if err != nil {
		return ergebnis, fmt.Errorf("löschen konnte nicht begonnen werden: %w", err)
	}
	defer db.Verwerfe(ctx, tx)
	// Als erstes die Sperre der Auflagen: Sie kommt vor jeder Zeilensperre, auch vor der des
	// Aussonderns (ladeWerkeDerTitel nimmt sie beim Löschen der Titel noch einmal).
	if err := nimmAuflagenSperre(ctx, tx); err != nil {
		return ergebnis, err
	}
	// Angaben und Exemplare der Titel, bevor die Zeilen fallen, in der Transaktion: Was ein Lesen
	// davor sähe, wäre nicht zwingend das, was fällt.
	titel, err := leseTitelFuersProtokoll(ctx, tx, auftrag.TitelIDs)
	if err != nil {
		return ergebnis, err
	}
	if len(titel) == 0 {
		return ergebnis, ErrTitelNichtGefunden
	}
	ganz, bleiben, err := teileNachBestaetigung(ctx, tx, titel, auftrag.Verliehene)
	if err != nil {
		return ergebnis, err
	}
	if err := sondereExemplareImHausAus(ctx, tx, kennungenDer(bleiben), auftrag.Grund, auftrag.BearbeiterID); err != nil {
		return ergebnis, err
	}
	cover, err := loescheGanzeTitel(ctx, tx, ganz, auftrag)
	if err != nil {
		return ergebnis, err
	}
	if err := tx.Commit(ctx); err != nil {
		return ergebnis, fmt.Errorf("löschen konnte nicht abgeschlossen werden: %w", err)
	}
	// Erst nach dem Commit: Die Cover-Dateien kommen nicht zurück, wenn das Löschen in der
	// Datenbank doch scheitert.
	loescheCoverDateien(cover)
	ergebnis.Geloescht = kennungenDer(ganz)
	ergebnis.Geblieben = kennungenDer(bleiben)
	return ergebnis, nil
}

// teileNachBestaetigung trennt die Titel, die ganz gehen, von denen, die bleiben: Ein Titel
// bleibt, solange eines seiner verliehenen Exemplare nicht bestätigt ist.
func teileNachBestaetigung(ctx context.Context, tx pgx.Tx, titel []titelVorDemLoeschen, bestaetigt []string) (ganz, bleiben []titelVorDemLoeschen, err error) {
	zeilen, err := tx.Query(ctx, `
		SELECT e.titel_id::text, e.id::text FROM buecher_exemplare e
		WHERE e.titel_id = ANY($1::uuid[]) AND `+sqlIstVerliehen, kennungenDer(titel))
	if err != nil {
		return nil, nil, fmt.Errorf("verliehene exemplare lesen: %w", err)
	}
	defer zeilen.Close()
	ja := make(map[string]bool, len(bestaetigt))
	for _, id := range bestaetigt {
		ja[id] = true
	}
	offen := map[string]bool{}
	for zeilen.Next() {
		var titelID, exemplarID string
		if err := zeilen.Scan(&titelID, &exemplarID); err != nil {
			return nil, nil, fmt.Errorf("verliehene exemplare lesen: %w", zeileUnlesbar(err))
		}
		if !ja[exemplarID] {
			offen[titelID] = true
		}
	}
	// Bräche die Iteration ab, sähe ein Titel mit verliehenem Exemplar aus wie einer ohne.
	if err := zeilen.Err(); err != nil {
		return nil, nil, fmt.Errorf("verliehene exemplare lesen: %w", err)
	}
	for _, t := range titel {
		if offen[t.id] {
			bleiben = append(bleiben, t)
		} else {
			ganz = append(ganz, t)
		}
	}
	return ganz, bleiben, nil
}

// sondereExemplareImHausAus sondert die Exemplare im Haus der Titel aus, die bleiben, mit dem
// Grund des Auftrags und dem Eintrag dazu. Ein Exemplar, das inzwischen verliehen wurde, bleibt
// unberührt (sqlIstImHaus).
func sondereExemplareImHausAus(ctx context.Context, tx pgx.Tx, titelIDs []string, grund, bearbeiterID string) error {
	if len(titelIDs) == 0 {
		return nil
	}
	if err := sperreExemplareDerTitel(ctx, tx, titelIDs); err != nil {
		return err
	}
	zeilen, err := tx.Query(ctx, `
		UPDATE buecher_exemplare e SET `+sqlSetzeAusgesondert+`
		WHERE e.titel_id = ANY($1::uuid[]) AND `+sqlIstImHaus+`
		RETURNING e.id::text`, titelIDs, grund)
	if err != nil {
		return fmt.Errorf("exemplare im haus aussondern: %w", err)
	}
	var ids []string
	for zeilen.Next() {
		var id string
		if err := zeilen.Scan(&id); err != nil {
			zeilen.Close()
			return fmt.Errorf("exemplare im haus aussondern: %w", zeileUnlesbar(err))
		}
		ids = append(ids, id)
	}
	zeilen.Close()
	if err := zeilen.Err(); err != nil {
		return fmt.Errorf("exemplare im haus aussondern: %w", err)
	}
	return protokolliereAussonderung(ctx, tx, ids, bearbeiterID, AussonderungsWegTitelLoeschen, nil)
}

// loescheGanzeTitel löscht die Titel, deren verliehene Exemplare bestätigt sind, und liefert die
// lokalen Cover, die nach dem Commit von der Platte gehen.
func loescheGanzeTitel(ctx context.Context, tx pgx.Tx, titel []titelVorDemLoeschen, auftrag TitelLoeschauftrag) ([]string, error) {
	if len(titel) == 0 {
		return nil, nil
	}
	ids := kennungenDer(titel)
	// Bücher in mehreren Auflagen: Ihre Werke müssen danach noch stimmen.
	werke, err := ladeWerkeDerTitel(ctx, tx, ids)
	if err != nil {
		return nil, err
	}
	// Vormerkungen, Reservierungen und Zuordnungen von Klassensätzen fallen über ON DELETE CASCADE
	// mit dem Titel; die DDL sagt das, der Code soll es auch sagen. Vorher lesen, danach ist es
	// weg.
	wartende, err := ladeWartendeBezuege(ctx, tx, ids)
	if err != nil {
		return nil, err
	}
	cover, err := ladeLokaleCoverNurDieserTitel(ctx, tx, ids)
	if err != nil {
		return nil, err
	}
	// Die Reihenfolge Schadensfall, Ausleihe, Exemplar, Titel erzwingen die RESTRICT-FKs. Die
	// beiden ersten Löschbefehle liefern ihre Spur gleich mit (RETURNING): die unbezahlten
	// Forderungen und die laufenden Ausleihen, genau die Zeilen, die fallen.
	offeneSchaeden, err := loescheSchaedenMitSpur(ctx, tx, ids)
	if err != nil {
		return nil, err
	}
	offene, err := loescheAusleihenMitSpur(ctx, tx, ids)
	if err != nil {
		return nil, err
	}
	if err := pruefeVerlieheneBestaetigt(offene, auftrag.Verliehene); err != nil {
		return nil, err
	}
	// Die Exemplare vor ihren Titeln: Über den CASCADE des Titels fielen sie ohne ihre Zeile in
	// den Bestandsbüchern.
	if _, err := loescheExemplareMitBuchzeile(ctx, tx, exemplareNachTitel, ids, auftrag.Grund); err != nil {
		return nil, err
	}
	geloescht, err := tx.Exec(ctx, `DELETE FROM buecher_titel WHERE id = ANY($1::uuid[])`, ids)
	if err != nil {
		return nil, fmt.Errorf("bücher konnten nicht gelöscht werden: %w", err)
	}
	if geloescht.RowsAffected() == 0 {
		return nil, ErrTitelNichtGefunden
	}
	if err := raeumeWerkeAuf(ctx, tx, werke); err != nil {
		return nil, err
	}
	// In derselben Transaktion: entweder die Löschung und ihre Spur oder keins von beidem.
	if err := protokolliereWasMitDenTitelnFiel(ctx, tx, offeneSchaeden, wartende, offene, titel, auftrag.BearbeiterID); err != nil {
		return nil, err
	}
	return cover, nil
}

// pruefeVerlieheneBestaetigt lehnt ab, wenn das Löschen eine laufende Ausleihe abräumen würde,
// deren Exemplar der Mensch nicht bestätigt hat: Es wurde nach der Rückfrage ausgeliehen.
func pruefeVerlieheneBestaetigt(offene []offeneAusleihe, bestaetigt []string) error {
	ja := make(map[string]bool, len(bestaetigt))
	for _, id := range bestaetigt {
		ja[id] = true
	}
	for _, o := range offene {
		if !ja[o.ExemplarID] {
			return ErrTitelInzwischenVerliehen
		}
	}
	return nil
}

// kennungenDer liefert die Kennungen der Titel in ihrer Reihenfolge.
func kennungenDer(titel []titelVorDemLoeschen) []string {
	ids := make([]string, 0, len(titel))
	for _, t := range titel {
		ids = append(ids, t.id)
	}
	return ids
}

// titelVorDemLoeschen hält fest, was nach dem Löschen nicht mehr zu lesen ist: die Angaben
// des Titels und die Barcodes seiner Exemplare, für das Protokoll.
type titelVorDemLoeschen struct {
	id, titel, autor, isbn string
	exemplare              []titelExemplar
}

type titelExemplar struct {
	id, barcode, lmfBarcode string
	warImBestand            bool
}

// leseTitelFuersProtokoll liest Angaben und Exemplare der Titel, bevor sie fallen; daraus
// schreibt das Löschen die Einträge (protokolliereTitelLoeschung). Eine unbekannte
// Kennung fehlt in der Antwort. Die Theken-Auskunft findet gelöschte Exemplare nur über
// Protokollzeilen mit tabelle='buecher_exemplare' und details->>'barcode_id'
// (FindeThekenExemplare); ohne sie hieße ein später gescanntes Buch „nie gesehen" statt
// „gelöscht am …".
func leseTitelFuersProtokoll(ctx context.Context, tx pgx.Tx, ids []string) ([]titelVorDemLoeschen, error) {
	rows, err := tx.Query(ctx,
		`SELECT id::text, coalesce(titel,''), coalesce(autor,''), coalesce(isbn,'')
		 FROM buecher_titel WHERE id = ANY($1::uuid[]) ORDER BY id`, ids)
	if err != nil {
		return nil, fmt.Errorf("failed to snapshot titles for audit: %w", err)
	}
	var titel []titelVorDemLoeschen
	for rows.Next() {
		var v titelVorDemLoeschen
		if err := rows.Scan(&v.id, &v.titel, &v.autor, &v.isbn); err != nil {
			rows.Close()
			return nil, fmt.Errorf("failed to scan title snapshot: %w", zeileUnlesbar(err))
		}
		titel = append(titel, v)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to read title snapshots: %w", err)
	}

	stelle := make(map[string]int, len(titel))
	for i, v := range titel {
		stelle[v.id] = i
	}
	rows, err = tx.Query(ctx,
		`SELECT e.titel_id::text, e.id::text, e.barcode_id, coalesce(e.lmf_barcode, ''), `+sqlWarImBestand+`
		 FROM buecher_exemplare e WHERE e.titel_id = ANY($1::uuid[]) ORDER BY e.barcode_id`, ids)
	if err != nil {
		return nil, fmt.Errorf("failed to snapshot copies for audit: %w", err)
	}
	for rows.Next() {
		var titelID string
		var ex titelExemplar
		if err := rows.Scan(&titelID, &ex.id, &ex.barcode, &ex.lmfBarcode, &ex.warImBestand); err != nil {
			rows.Close()
			return nil, fmt.Errorf("failed to scan copy snapshot: %w", zeileUnlesbar(err))
		}
		if i, ok := stelle[titelID]; ok {
			titel[i].exemplare = append(titel[i].exemplare, ex)
		}
	}
	rows.Close()
	// Bricht die Iteration vorzeitig ab, fehlten Einträge still.
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to read copy snapshots: %w", err)
	}
	return titel, nil
}

// protokolliereTitelLoeschung schreibt den Eintrag zum Titel und je Exemplar den
// Barcode-Eintrag in der Form der Geschwister-Pfade (SondereExemplarAus, Verlust-Löschen), in der
// Transaktion des Löschens: entweder Löschung und Spur oder keins von beiden.
func protokolliereTitelLoeschung(ctx context.Context, tx pgx.Tx, vorher titelVorDemLoeschen, bearbeiterID string) error {
	if err := schreibeProtokollEintrag(ctx, tx, protokollEintrag{
		Tabelle: "buecher_titel", Aktion: "DELETE", DatensatzID: vorher.id,
		BearbeiterID: &bearbeiterID, Akteur: "USER",
		Details: map[string]any{"titel": vorher.titel, "autor": vorher.autor, "isbn": vorher.isbn},
	}); err != nil {
		return err
	}
	kontext := "Titel gelöscht — Exemplar mit entfernt"
	for _, ex := range vorher.exemplare {
		details := map[string]any{
			"barcode_id": ex.barcode, "titel": vorher.titel, "titel_id": vorher.id,
			"action": ProtokollAktionTitelGeloescht, ProtokollDetailWarImBestand: ex.warImBestand,
		}
		merkeLMFBarcode(details, ex.lmfBarcode)
		if err := schreibeProtokollEintrag(ctx, tx, protokollEintrag{
			Tabelle: "buecher_exemplare", Aktion: "DELETE", DatensatzID: ex.id,
			BearbeiterID: &bearbeiterID, Akteur: "USER", Kontext: &kontext,
			Details: details,
		}); err != nil {
			return err
		}
	}
	return nil
}

// Die Marken im Protokoll der Türen, die ein Exemplar körperlich entfernen. Konstanten und kein
// getippter Text an jeder Tür, weil das Abgangsbuch sie liest (abgangsbuch.go): Ein gelöschtes
// Exemplar fällt aus jeder Abfrage über buecher_exemplare, der Nachweis zählt es nur über diese
// Spur. Wer den Text an einer Tür ändert, ohne die Abfrage zu kennen, senkt die Zahl auf 0, und
// ein Nachweis, der schweigt, sieht aus wie ein vollständiger.
const (
	// ProtokollAktionTitelGeloescht: Das Exemplar ging mit seinem Titel.
	ProtokollAktionTitelGeloescht = "titel_geloescht"
	// ProtokollAktionVerlustEndgueltigGeloescht: Ein als Verlust gebuchtes Exemplar wurde in
	// der Inventur endgültig entfernt (api/inventurlauf/inventur_verlust_aktionen.go).
	ProtokollAktionVerlustEndgueltigGeloescht = "verlust_endgueltig_geloescht"
	// ProtokollDetailWarImBestand steht an jeder dieser Spuren: ob das Exemplar ein Zugangsdatum
	// trug. Ein bestelltes, nie eingetroffenes Exemplar zählt das Abgangsbuch damit nicht.
	ProtokollDetailWarImBestand = "war_im_bestand"
	// ProtokollDetailLMFBarcode nennt das alte LMF-Etikett des Exemplars (lmf_barcode), wenn es
	// eins hatte; die Theken-Auskunft findet das gelöschte Exemplar auch darüber.
	ProtokollDetailLMFBarcode = "lmf_barcode"
)

// merkeLMFBarcode trägt das alte LMF-Etikett in die Spur eines gelöschten Exemplars ein, wenn es
// eins hatte (ProtokollDetailLMFBarcode).
func merkeLMFBarcode(details map[string]any, lmfBarcode string) {
	if lmfBarcode != "" {
		details[ProtokollDetailLMFBarcode] = lmfBarcode
	}
}

// Die Merker der Spuren, die eine Titel-Löschung zu einer Person hinterlässt. Konstanten aus
// demselben Grund: Das Auskunftsblatt nennt den Vorgang über sie in Worten
// (internal/auskunft/blatt_protokoll.go), bei einem anderen Text stünde dort „DELETE (ausleihen)".
const (
	// ProtokollAktionTitelMitAusleihe: Der Titel ging, während ein Exemplar verliehen war.
	ProtokollAktionTitelMitAusleihe = "titel_geloescht_mit_offener_ausleihe"
	// ProtokollAktionTitelMitForderung: Der Titel ging, während eine Forderung offen stand.
	ProtokollAktionTitelMitForderung = "titel_geloescht_mit_offener_forderung"
	// ProtokollAktionTitelMitBezug: Der Titel ging, während eine Vormerkung oder Reservierung
	// wartete oder ein Klassensatz ihn führte (ProtokolliereWartendeBezuege).
	ProtokollAktionTitelMitBezug = "titel_geloescht_mit_offenem_bezug"
)

// protokolliereWasMitDenTitelnFiel schreibt fest, was das Löschen mitgenommen hat: unbezahlte
// Forderungen, Wartende und laufende Ausleihen, dann je Titel den Eintrag mit dem Bearbeiter und
// die Barcodes seiner Exemplare, wie beim Löschen eines einzelnen Titels.
func protokolliereWasMitDenTitelnFiel(ctx context.Context, tx pgx.Tx, schaeden []offenerSchaden, wartende []WartenderBezug, ausleihen []offeneAusleihe, titel []titelVorDemLoeschen, bearbeiterID string) error {
	if err := protokolliereOffeneSchaeden(ctx, tx, schaeden); err != nil {
		return err
	}
	if err := ProtokolliereWartendeBezuege(ctx, tx, wartende); err != nil {
		return err
	}
	if err := protokolliereOffeneAusleihen(ctx, tx, ausleihen); err != nil {
		return err
	}
	for _, vorher := range titel {
		if err := protokolliereTitelLoeschung(ctx, tx, vorher, bearbeiterID); err != nil {
			return err
		}
	}
	return nil
}
