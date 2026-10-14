package lusd

import (
	"context"
	"fmt"
	"time"

	"bibliothek/repository"

	"github.com/jackc/pgx/v5"
)

// WendeAenderungenAn führt den zweiten Durchlauf aus: Schüler des Bestands aktualisieren
// (Klasse und Kontaktdaten) und Neuzugänge anlegen, entlang der Zuordnung aus der
// Klassifizierung, Zeile für Zeile in der Reihenfolge der Datei. Ein Neuzugang wird sofort
// eingefügt: Die nächste Zeile mit derselben LUSD-ID findet ihn dann in der Transaktion.
func WendeAenderungenAn(ctx context.Context, tx pgx.Tx, datei Datei, z Zuordnung) error {
	// Die Ausweisnummern der Neuzugänge kommen aus derselben Quelle wie die der Anlage von
	// Hand, einmal je Lauf gezogen und dann fortlaufend weitergezählt. Einmal und nicht je
	// Zeile: AusweisnummernAbfragen.ZieheNaechste hält eine Sperre in dieser Transaktion, bis sie endet. Der
	// Lauf hat den Nummernkreis damit für sich, und die Nummern sind lückenlos. Der Zähler
	// rechnet über die Tabelle `leser`, nicht über die Sicht `schueler`: Die höchste Nummer
	// kann an einem Kollegen hängen.
	ersteNummer, err := repository.NewAusweisnummernAbfragen(tx).ZieheNaechste(ctx)
	if err != nil {
		return fmt.Errorf("ausweisnummern für die Neuzugänge: %w", err)
	}
	vergeben := 0

	var bestandsZeilen []Zeile
	var bestandsIDs []string

	for i, zeile := range datei.Zeilen {
		if z.ueberspringen[i] {
			continue
		}
		if id, ok := z.zielID[i]; ok {
			zeile.geburtsdatumUebernehmen = z.geburtsdatumSetzen[i]
			bestandsZeilen = append(bestandsZeilen, zeile)
			bestandsIDs = append(bestandsIDs, id)
			continue
		}

		bestandID, err := belegteLUSDID(ctx, tx, datei.Modus, zeile.LUSDID)
		if err != nil {
			return err
		}
		if bestandID != "" {
			bestandsZeilen = append(bestandsZeilen, zeile)
			bestandsIDs = append(bestandsIDs, bestandID)
			continue
		}

		if err := legeNeuenSchuelerAn(ctx, tx, zeile, ersteNummer+vergeben); err != nil {
			return err
		}
		vergeben++
	}

	if len(bestandsZeilen) > 0 {
		return aendereBestandsschueler(ctx, tx, bestandsZeilen, bestandsIDs)
	}

	return nil
}

// belegteLUSDID nennt im ID-Modus die Zeile, die die LUSD-ID schon hält, obwohl die Zuordnung
// sie nicht kennt: den eben adoptierten Waisen oder einen Rückkehrer. Ein INSERT liefe dort auf
// uniq_schueler_lusd_id_active auf und risse den ganzen Import mit. Soft-gelöschte Zeilen
// belegen den Index nicht und entstehen als frischer Datensatz neu.
func belegteLUSDID(ctx context.Context, tx pgx.Tx, modus Modus, lusdID string) (string, error) {
	if modus != ModusID {
		return "", nil
	}
	return repository.FindeAktivenSchuelerZuLUSDID(ctx, tx, lusdID)
}

// adoptiereWaisen heftet die LUSD-ID an bestehende Schüler ohne ID (Adoption über Name und
// Geburtsdatum). Danach behandelt WendeAenderungenAn sie wie Schüler des Bestands. Im
// Abgleich über den Namen ist die LUSD-ID leer: Dann trägt die Adoption nur das Geburtsdatum
// nach, Klasse und Bestätigung übernimmt der Batch. Was die Anweisung gegen einen Wettlauf
// schützt, steht an repository.AdoptiereLUSDWaise.
func adoptiereWaisen(ctx context.Context, tx pgx.Tx, adoptionen []VorschauAdoption) error {
	for _, a := range adoptionen {
		if err := repository.AdoptiereLUSDWaise(ctx, tx, a.SchuelerID, a.LUSDID, a.Geburtsdatum); err != nil {
			return err
		}
	}
	return nil
}

// legeNeuenSchuelerAn legt einen Schüler an, den der Export neu nennt. Das Abgangsjahr folgt
// der Klasse wie bei der Anlage von Hand (repository.AbgaengerJahr): eine Antwort auf dieselbe
// Frage.
func legeNeuenSchuelerAn(ctx context.Context, tx pgx.Tx, zeile Zeile, nummer int) error {
	return repository.LegeLUSDSchuelerAn(ctx, tx, repository.LUSDNeuzugang{
		Ausweisnummer: repository.AusweisNummer(nummer),
		Vorname:       zeile.Vorname,
		Nachname:      zeile.Nachname,
		Klasse:        zeile.Klasse,
		AbgaengerJahr: repository.AbgaengerJahr(zeile.Klasse),
		LUSDID:        zeile.LUSDID,
		Geburtsdatum:  zeile.Geburtsdatum,
		EintrittAm:    zeile.EintrittAm,
		Strasse:       zeile.Strasse,
		Hausnummer:    zeile.Hausnummer,
		PLZ:           zeile.PLZ,
		Ort:           zeile.Ort,
		ElternEmail:   zeile.ElternEmail,
	})
}

// aendereBestandsschueler übergibt die Zeilen des Exports, die zu einem Schüler des
// Bestands gehören. Das Geburtsdatum geht nur für ein bestätigtes Umbenennungs-Paar mit; die
// Regeln der Anweisung stehen an repository.AendereLUSDBestand.
func aendereBestandsschueler(ctx context.Context, tx pgx.Tx, bestandsZeilen []Zeile, ids []string) error {
	zeilen := make([]repository.LUSDAktualisierung, len(bestandsZeilen))
	for i, zeile := range bestandsZeilen {
		var gebFuerPaar *time.Time
		if zeile.geburtsdatumUebernehmen {
			gebFuerPaar = zeile.Geburtsdatum
		}
		zeilen[i] = repository.LUSDAktualisierung{
			SchuelerID:   ids[i],
			Vorname:      zeile.Vorname,
			Nachname:     zeile.Nachname,
			Klasse:       zeile.Klasse,
			Strasse:      zeile.Strasse,
			Hausnummer:   zeile.Hausnummer,
			PLZ:          zeile.PLZ,
			Ort:          zeile.Ort,
			ElternEmail:  zeile.ElternEmail,
			EintrittAm:   zeile.EintrittAm,
			Geburtsdatum: gebFuerPaar,
		}
	}
	return repository.AendereLUSDBestand(ctx, tx, zeilen)
}

// behandleAbgaenger verarbeitet Schüler, die nicht mehr im Export stehen. Mit offenen
// Vorgängen bleiben Name und Kontaktdaten erhalten: Mahnwesen und Schadensrechnung brauchen
// sie noch. Ohne offene Vorgänge entscheidet die Karenzzeit (Einstellung
// abgaenger_karenz_tage): Über 0 wird nur gesperrt, und der nächtliche Lauf anonymisiert
// nach Ablauf (PraedikatAnonymisierung, Uhr abgaenger_seit); bei 0 wird sofort anonymisiert.
// In der Karenz lässt sich eine falsche Zuordnung noch beheben, etwa eine Umbenennung ohne
// Schüler-ID: über die Paarung der Vorschau beim nächsten Lauf oder von Hand
// (Zusammenführen).
func behandleAbgaenger(ctx context.Context, tx pgx.Tx, abgaengerIDs []string, karenzTage int) error {
	if len(abgaengerIDs) == 0 {
		return nil
	}

	// Wartende Vormerkungen zuerst: Sie hielten begehrte Titel für die anderen besetzt.
	if err := repository.LoescheWartendeVormerkungen(ctx, tx, abgaengerIDs); err != nil {
		return err
	}

	offeneAusleihen, err := repository.ZaehleOffeneAusleihenJeSchueler(ctx, tx, abgaengerIDs)
	if err != nil {
		return err
	}

	// Auch ein unbezahlter Schaden hält die Anonymisierung auf: Ohne Name und Anschrift
	// ließe sich der Schüler nicht mehr anschreiben.
	offeneSchaeden, err := repository.ZaehleOffeneSchaedenJeSchueler(ctx, tx, abgaengerIDs)
	if err != nil {
		return err
	}

	for _, schuelerID := range abgaengerIDs {
		offen := offeneAusleihen[schuelerID] > 0 || offeneSchaeden[schuelerID] > 0
		if err := sperreOderAnonymisiere(ctx, tx, schuelerID, offen, karenzTage); err != nil {
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
		return repository.SperreAbgaenger(ctx, tx, schuelerID, repository.AbgaengerSperrgrundOffen)
	case karenzTage > 0:
		return repository.SperreAbgaenger(ctx, tx, schuelerID, repository.AbgaengerSperrgrundKarenz)
	default:
		return repository.AnonymisiereAbgaenger(ctx, tx, schuelerID)
	}
}
