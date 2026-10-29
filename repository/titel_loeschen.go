package repository

import (
	"context"
	"errors"
	"fmt"

	"bibliothek/db"

	"github.com/jackc/pgx/v5"
)

// TitelHatAusleihenFehler meldet, dass ein Titel nicht gelöscht werden kann, weil noch
// Exemplare verliehen sind: ein Bedienfehler (400), keine Störung. Barcodes nennt die
// verliehenen Exemplare; die Tür nennt sie in ihrer Meldung.
type TitelHatAusleihenFehler struct{ Barcodes []string }

func (e *TitelHatAusleihenFehler) Error() string {
	return fmt.Sprintf("titel hat verliehene exemplare: %v", e.Barcodes)
}

// ErrTitelNichtGefunden meldet einen Titel, den es unter der genannten Kennung nicht gibt.
var ErrTitelNichtGefunden = errors.New("titel nicht gefunden")

// LoescheTitel löscht einen Titel mit seinen Exemplaren und schreibt die Einträge dazu. Ist ein
// Exemplar verliehen, lehnt es ab. Zurückgegebene Ausleihen und alle Schadensfälle des Titels
// fallen mit, weil ihre Fremdschlüssel den Titel sonst hielten; unbezahlte Forderungen stehen
// vorher im Protokoll (protokolliereOffeneForderungen).
func (a *KatalogAbfragen) LoescheTitel(ctx context.Context, titelID string, bearbeiterID string) error {
	tx, err := a.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer db.Verwerfe(ctx, tx)

	// Gehört der Titel zu einem Buch in mehreren Auflagen, muss das Werk danach noch stimmen.
	// Zuerst, weil die Sperre der Auflagen vor jeder Zeilensperre kommt.
	werke, err := ladeWerkeDerTitel(ctx, tx, []string{titelID})
	if err != nil {
		return err
	}

	vorher, err := leseTitelVorDemLoeschen(ctx, tx, titelID)
	if err != nil {
		return err
	}
	if err = protokolliereWasMitDemTitelFaellt(ctx, tx, titelID); err != nil {
		return err
	}
	// Das lokal gespeicherte Cover fällt mit dem Titel, wie in der Massenaktion der
	// Bestandstabelle (TitelAbfragen.LoescheTitelSamtAusleihen).
	cover, err := ladeLokaleCoverNurDieserTitel(ctx, tx, []string{titelID})
	if err != nil {
		return err
	}
	// Schadensfälle und zurückgegebene Ausleihen zuerst: Ihre Fremdschlüssel hielten die
	// Exemplare sonst fest (ON DELETE RESTRICT).
	if _, err = tx.Exec(ctx, "DELETE FROM schadensfaelle WHERE exemplar_id IN (SELECT id FROM buecher_exemplare WHERE titel_id = $1)", titelID); err != nil {
		return fmt.Errorf("failed to delete damage records for title: %w", err)
	}
	if _, err = tx.Exec(ctx, "DELETE FROM ausleihen WHERE exemplar_id IN (SELECT id FROM buecher_exemplare WHERE titel_id = $1) AND rueckgabe_am IS NOT NULL", titelID); err != nil {
		return fmt.Errorf("failed to delete past loans for title: %w", err)
	}

	if _, err = tx.Exec(ctx, "DELETE FROM buecher_exemplare WHERE titel_id = $1", titelID); err != nil {
		return fmt.Errorf("failed to delete associated copies: %w", err)
	}

	// Keine getroffene Zeile heißt unbekannte ID: Die Aufnahme oben lässt einen fehlenden Titel
	// durch, und ohne diese Prüfung entstünde ein Eintrag über einen Titel, den es nie gab.
	tag, err := tx.Exec(ctx, "DELETE FROM buecher_titel WHERE id = $1", titelID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrTitelNichtGefunden
	}
	if err = raeumeWerkeAuf(ctx, tx, werke); err != nil {
		return err
	}

	if err = protokolliereTitelLoeschung(ctx, tx, vorher, bearbeiterID); err != nil {
		return err
	}

	if err = tx.Commit(ctx); err != nil {
		return err
	}
	loescheCoverDateien(cover)
	return nil
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

// leseTitelVorDemLoeschen liest die Angaben fürs Protokoll und lehnt ab, solange ein Exemplar
// verliehen ist. Einen unbekannten Titel lässt sie durch; den meldet das DELETE.
func leseTitelVorDemLoeschen(ctx context.Context, tx pgx.Tx, titelID string) (titelVorDemLoeschen, error) {
	if err := pruefeKeineAktivenAusleihen(ctx, tx, titelID); err != nil {
		return titelVorDemLoeschen{id: titelID}, err
	}
	gelesen, err := leseTitelFuersProtokoll(ctx, tx, []string{titelID})
	if err != nil || len(gelesen) == 0 {
		return titelVorDemLoeschen{id: titelID}, err
	}
	return gelesen[0], nil
}

// pruefeKeineAktivenAusleihen lehnt das Löschen ab, solange ein Exemplar des Titels verliehen
// ist; der Fehler nennt die Barcodes.
func pruefeKeineAktivenAusleihen(ctx context.Context, tx pgx.Tx, titelID string) error {
	var verliehen []string
	rows, err := tx.Query(ctx, `
		SELECT e.barcode_id 
		FROM ausleihen a 
		JOIN buecher_exemplare e ON a.exemplar_id = e.id 
		WHERE e.titel_id = $1 AND a.rueckgabe_am IS NULL
	`, titelID)
	if err != nil {
		return fmt.Errorf("failed to check active loans for title: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var barcode string
		if err := rows.Scan(&barcode); err != nil {
			return fmt.Errorf("failed to read active loans for title: %w", zeileUnlesbar(err))
		}
		verliehen = append(verliehen, barcode)
	}
	// Bricht die Iteration durch einen Verbindungsfehler vorzeitig ab, heißt das nicht
	// „keine aktiven Ausleihen"; der Titel darf dann nicht gelöscht werden.
	if err := rows.Err(); err != nil {
		return fmt.Errorf("failed to read active loans for title: %w", err)
	}
	if len(verliehen) > 0 {
		return &TitelHatAusleihenFehler{Barcodes: verliehen}
	}
	return nil
}

// leseTitelFuersProtokoll liest Angaben und Exemplare der Titel, bevor sie fallen; beide
// Löschwege schreiben daraus dieselben Einträge (protokolliereTitelLoeschung). Eine unbekannte
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

// protokolliereWasMitDemTitelFaellt hält vor dem Löschen fest, was mit dem Titel verschwindet:
// unbezahlte Forderungen (wer wie viel wofür schuldete) und, per ON DELETE CASCADE, wer auf
// den Titel wartet. Dieselbe Regel wie in der Massenaktion der Bestandstabelle
// (titel_loeschen_wartende.go).
func protokolliereWasMitDemTitelFaellt(ctx context.Context, tx pgx.Tx, titelID string) error {
	if err := protokolliereOffeneForderungen(ctx, tx, titelID); err != nil {
		return err
	}
	wartende, err := ladeWartendeBezuege(ctx, tx, []string{titelID})
	if err != nil {
		return err
	}
	return ProtokolliereWartendeBezuege(ctx, tx, wartende)
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
	// ProtokollAktionTitelGeloescht: Das Exemplar ging mit seinem Titel, an beiden Türen zum
	// Löschen.
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
