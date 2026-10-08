package api

import (
	"context"
	"fmt"
	"time"

	"bibliothek/repository"

	"github.com/jackc/pgx/v5"
)

// wendeLusdAenderungenAn führt den zweiten Durchlauf aus: bestehende Schüler
// aktualisieren (Klasse + Kontaktdaten) und Neuzugänge anlegen — entlang der
// Zuordnung aus der Klassifizierung, Zeile für Zeile in Dateireihenfolge.
func wendeLusdAenderungenAn(ctx context.Context, tx pgx.Tx, datei lusdDatei, z lusdZuordnung) error {
	// Die Ausweisnummern der Neuzugänge kommen aus DERSELBEN Quelle wie die der
	// Handanlage — einmal je Lauf gezogen, dann fortlaufend weitergezählt.
	//
	// Einmal und nicht je Zeile: NaechsteAusweisnummer (ausweis_nummer_start, Migration 136)
	// hält einen Advisory-Lock in DIESER Transaktion, bis sie endet. Der Lauf hat den Nummernkreis damit für sich, und die
	// Nummern sind lückenlos. Gefragt wird die TABELLE `leser`, nicht die Sicht
	// `schueler`: Die höchste Nummer kann seit Migration 125 an einem Kollegen hängen,
	// und über die Sicht gerechnet gäbe der Generator sie ein zweites Mal aus.
	startNum, err := repository.NewSequenceRepository(tx).NaechsteAusweisnummer(ctx)
	if err != nil {
		return fmt.Errorf("ausweisnummern für die Neuzugänge: %w", err)
	}
	barcodeCounter := 0

	var batchRecords []parsedStudentRow
	var batchIDs []string

	for i, rec := range datei.Zeilen {
		if z.ueberspringen[i] {
			continue
		}
		if id, ok := z.zielID[i]; ok {
			rec.geburtsdatumUebernehmen = z.geburtsdatumSetzen[i]
			batchRecords = append(batchRecords, rec)
			batchIDs = append(batchIDs, id)
			continue
		}

		bestandID, err := belegteLusdID(ctx, tx, datei.Modus, rec.LusdID)
		if err != nil {
			return err
		}
		if bestandID != "" {
			batchRecords = append(batchRecords, rec)
			batchIDs = append(batchIDs, bestandID)
			continue
		}

		if err := legeNeuenSchuelerAn(ctx, tx, rec, startNum+barcodeCounter); err != nil {
			return err
		}
		barcodeCounter++
	}

	if len(batchRecords) > 0 {
		return aktualisiereBestandsschuelerBatch(ctx, tx, batchRecords, batchIDs)
	}

	return nil
}

// belegteLusdID nennt im ID-Modus die Zeile, die die LUSD-ID schon hält, obwohl die Zuordnung
// sie nicht kennt: den eben adoptierten Waisen oder einen Rückkehrer. Ein INSERT liefe dort auf
// uniq_schueler_lusd_id_active auf und risse den ganzen Import mit. Soft-gelöschte Zeilen
// belegen den Index nicht und entstehen als frischer Datensatz neu.
func belegteLusdID(ctx context.Context, tx pgx.Tx, modus lusdModus, lusdID string) (string, error) {
	if modus != lusdModusID {
		return "", nil
	}
	return repository.FindeAktivenSchuelerNachLusdID(ctx, tx, lusdID)
}

// adoptiereWaisen heftet die LUSD-ID an bestehende Schüler ohne ID (Adoption über Name und
// Geburtsdatum). Danach behandelt wendeLusdAenderungenAn sie wie Schüler des Bestands. Im
// Abgleich über den Namen ist die LUSD-ID leer: Dann trägt die Adoption nur das Geburtsdatum
// nach, Klasse und Bestätigung übernimmt der Batch. Was die Anweisung gegen einen Wettlauf
// schützt, steht an repository.AdoptiereLusdWaise.
func adoptiereWaisen(ctx context.Context, tx pgx.Tx, adoptionen []AdoptionDiff) error {
	for _, a := range adoptionen {
		if err := repository.AdoptiereLusdWaise(ctx, tx, a.SchuelerID, a.LusdID, a.Geburtsdatum); err != nil {
			return err
		}
	}
	return nil
}

// legeNeuenSchuelerAn legt einen Schüler an, den der Export neu nennt. Das Abgangsjahr folgt
// der Klasse wie bei der Anlage von Hand (repository.AbgaengerJahr): eine Antwort auf dieselbe
// Frage.
func legeNeuenSchuelerAn(ctx context.Context, tx pgx.Tx, rec parsedStudentRow, barcodeCounter int) error {
	return repository.LegeLusdSchuelerAn(ctx, tx, repository.LusdNeuzugang{
		Ausweisnummer: generateImportBarcode(barcodeCounter),
		Vorname:       rec.Vorname,
		Nachname:      rec.Nachname,
		Klasse:        rec.Klasse,
		AbgaengerJahr: repository.AbgaengerJahr(rec.Klasse),
		LusdID:        rec.LusdID,
		Geburtsdatum:  rec.GebDatum,
		EintrittAm:    rec.EintrittAm,
		Strasse:       rec.Strasse,
		Hausnummer:    rec.Hausnummer,
		PLZ:           rec.PLZ,
		Ort:           rec.Ort,
		ElternEmail:   rec.ElternEmail,
	})
}

// aktualisiereBestandsschuelerBatch übergibt die Zeilen des Exports, die zu einem Schüler des
// Bestands gehören. Das Geburtsdatum geht nur für ein bestätigtes Umbenennungs-Paar mit; die
// Regeln der Anweisung stehen an repository.AktualisiereLusdBestand.
func aktualisiereBestandsschuelerBatch(ctx context.Context, tx pgx.Tx, records []parsedStudentRow, ids []string) error {
	zeilen := make([]repository.LusdAktualisierung, len(records))
	for i, rec := range records {
		var gebFuerPaar *time.Time
		if rec.geburtsdatumUebernehmen {
			gebFuerPaar = rec.GebDatum
		}
		zeilen[i] = repository.LusdAktualisierung{
			SchuelerID:   ids[i],
			Vorname:      rec.Vorname,
			Nachname:     rec.Nachname,
			Klasse:       rec.Klasse,
			Strasse:      rec.Strasse,
			Hausnummer:   rec.Hausnummer,
			PLZ:          rec.PLZ,
			Ort:          rec.Ort,
			ElternEmail:  rec.ElternEmail,
			EintrittAm:   rec.EintrittAm,
			Geburtsdatum: gebFuerPaar,
		}
	}
	return repository.AktualisiereLusdBestand(ctx, tx, zeilen)
}

// behandleAbgaenger verarbeitet Schüler, die nicht mehr im Export stehen.
// Mit offenen Ausleihen bleiben Name UND Kontaktdaten erhalten (fürs Mahnwesen und
// die Schadens-Rechnung noch nötig). Ohne offene Vorgänge entscheidet die Karenzzeit
// (karenzTage, Einstellung abgaenger_karenz_tage): > 0 heißt nur sperren — der
// nächtliche Job anonymisiert nach Ablauf (PredikatAnonymisierung, Uhr abgaenger_seit);
// 0 heißt sofort anonymisieren, wie bis zum 02.09.2026. Die Karenz ist der Raum, in
// dem eine falsche Zuordnung (Umbenennung ohne Schüler-ID) noch repariert werden kann —
// per Vorschau-Paarung beim nächsten Lauf oder von Hand (Zusammenführen).
func behandleAbgaenger(ctx context.Context, tx pgx.Tx, gradIDs []string, karenzTage int) error {
	if len(gradIDs) == 0 {
		return nil
	}

	// Wartende Vormerkungen zuerst: Sie hielten begehrte Titel für die anderen besetzt.
	if err := repository.LoescheWartendeVormerkungen(ctx, tx, gradIDs); err != nil {
		return err
	}

	pendingCounts, err := repository.ZaehleOffeneAusleihenJeSchueler(ctx, tx, gradIDs)
	if err != nil {
		return err
	}

	// Auch ein unbezahlter Schaden hält die Anonymisierung auf: Ohne Name und Anschrift
	// ließe sich der Schüler nicht mehr anschreiben.
	offeneSchaedenCounts, err := repository.ZaehleOffeneSchaedenJeSchueler(ctx, tx, gradIDs)
	if err != nil {
		return err
	}

	for _, sID := range gradIDs {
		offen := pendingCounts[sID] > 0 || offeneSchaedenCounts[sID] > 0
		if err := sperreOderAnonymisiere(ctx, tx, sID, offen, karenzTage); err != nil {
			return err
		}
	}

	return nil
}

// sperreOderAnonymisiere schließt einen Abgänger ab: Mit offenen Vorgängen und während der
// Karenzzeit wird er nur gesperrt, sonst sofort anonymisiert.
func sperreOderAnonymisiere(ctx context.Context, tx pgx.Tx, schuelerID string, offen bool, karenzTage int) error {
	switch {
	case offen:
		return repository.SperreAbgaenger(ctx, tx, schuelerID, abgaengerSperrgrundOffen)
	case karenzTage > 0:
		return repository.SperreAbgaenger(ctx, tx, schuelerID, abgaengerSperrgrundKarenz)
	default:
		return repository.AnonymisiereAbgaenger(ctx, tx, schuelerID)
	}
}

// Die beiden automatischen Sperrgründe teilen das Präfix, an dem der Rückkehrer-Pfad
// (aktualisiereBestandsschuelerBatch) und das Zusammenführen die Automatik erkennen.
const (
	abgaengerSperrgrundOffen  = repository.AbgaengerSperrgrundOffen
	abgaengerSperrgrundKarenz = repository.AbgaengerSperrgrundKarenz
)
