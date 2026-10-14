package service

import (
	"context"
	"errors"
	"log"

	"bibliothek/repository"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// legeAusleiheAn schreibt die Ausleihe. Prallt sie am eindeutigen Index ab, weil zwischen
// Prüfung und Schreiben ein anderer Vorgang das Exemplar gebucht hat, entscheidet
// deuteAusleiheKonflikt, ob das ein Fehler ist.
func (d *ausleihDienst) legeAusleiheAn(
	ctx context.Context,
	tx pgx.Tx,
	exemplar *repository.Exemplar,
	vorgang *ausleihVorgang,
	bearbeiterID string,
) (*repository.Ausleihe, error) {
	// Wer kein Schüler ist, bekommt das Buch als Dauerleihe und zählt nicht in die
	// Überfällig-Automatik (repository.ZaehleUeberfaelligeAusleihen).
	ausleihe, err := d.ausleihAbfragen.LegeAn(ctx, tx, exemplar.ID, vorgang.leserID, bearbeiterID, vorgang.frist, !vorgang.istSchueler())
	if err == nil {
		return ausleihe, nil
	}
	if !errors.Is(err, repository.ErrAusleiheKonflikt) {
		return nil, deuteAnlegeFehler(err)
	}

	// Doppelter Scan desselben Ausleihers: gewünschter Zustand besteht schon.
	deutung := d.deuteAusleiheKonflikt(ctx, tx, exemplar, vorgang)
	var bereits errAusleiheBereitsVorhanden
	if errors.As(deutung, &bereits) {
		return bereits.ausleihe, nil
	}
	return nil, deutung
}

// deuteAusleiheKonflikt klärt nach einem abgeprallten Schreiben, wer das Exemplar jetzt hat.
// Derselbe Leser (der Scanner hat doppelt ausgelöst) hat den gewünschten Zustand schon: Die
// bestehende Ausleihe kommt zurück, der Scan war wirkungslos, aber nicht falsch. Ein anderer
// Leser ist ein Fehler, den der Arbeitsplatz sehen muss; sonst gäbe die Theke ein Buch heraus,
// das auf jemand anderem steht.
//
// Read Committed: Die fremde Transaktion ist committet, ein neues SELECT sieht sie.
func (d *ausleihDienst) deuteAusleiheKonflikt(
	ctx context.Context,
	tx pgx.Tx,
	exemplar *repository.Exemplar,
	vorgang *ausleihVorgang,
) error {
	aktiv, err := d.ausleihAbfragen.SperreOffene(ctx, tx, exemplar.ID)
	if err != nil {
		return err
	}
	if aktiv == nil {
		// Weder geschrieben noch auffindbar — nichts beschönigen.
		return meldung(ErrKonflikt, "Dieses Exemplar konnte nicht verbucht werden, bitte erneut scannen")
	}

	if istSelberAusleiher(aktiv, vorgang) {
		return errAusleiheBereitsVorhanden{ausleihe: aktiv}
	}

	return meldung(ErrKonflikt, "Dieses Exemplar wurde soeben an einem anderen Arbeitsplatz verbucht")
}

// istSelberAusleiher prüft, ob die bestehende Ausleihe demselben Leser gehört wie der
// aktuelle Vorgang.
func istSelberAusleiher(aktiv *repository.Ausleihe, vorgang *ausleihVorgang) bool {
	return aktiv.SchuelerID != nil && *aktiv.SchuelerID == vorgang.leserID
}

// errAusleiheBereitsVorhanden ist kein Fehler nach außen, sondern das Zeichen an
// legeAusleiheAn, mit der bestehenden Ausleihe weiterzumachen (doppelter Scan).
type errAusleiheBereitsVorhanden struct{ ausleihe *repository.Ausleihe }

func (e errAusleiheBereitsVorhanden) Error() string {
	return "ausleihe besteht bereits für denselben Ausleiher"
}

// deuteAnlegeFehler übersetzt die Verletzung des eindeutigen Index (höchstens eine offene
// Ausleihe je Exemplar oder Gerät) in einen Konflikt: Ein zweiter gleichzeitiger Scan desselben
// Exemplars bekommt 409 statt 500.
func deuteAnlegeFehler(err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return meldung(ErrKonflikt, "Dieses Exemplar wurde soeben bereits ausgeliehen")
	}
	return err
}

// entferneErfuellteVormerkung löscht die Vormerkung des Schülers für den Titel, den er gerade
// ausgeliehen hat. Lag für ihn ein anderes Exemplar im Abholfach als das, das er genommen hat,
// muss es zurück ins Regal: Seine Nummer geht als Regal-Hinweis in die Antwort. Einen Fehler
// meldet die Funktion nicht weiter, sie schreibt ihn ins Log.
func entferneErfuellteVormerkung(ctx context.Context, tx pgx.Tx, exemplar *repository.Exemplar, schuelerID string, erg *AusleihErgebnis) {
	bereitgestellt, err := repository.LoescheErfuellteVormerkung(ctx, tx, exemplar.TitelID, schuelerID)
	if err != nil {
		log.Printf("ausleihe: Vormerkung für Titel %s konnte nicht entfernt werden: %v", exemplar.TitelID, err)
		return
	}
	if bereitgestellt == nil || *bereitgestellt == exemplar.ID {
		return // keine Vormerkung, nichts reserviert, oder genau dieses Exemplar wurde genommen
	}

	barcode, err := repository.LadeExemplarNummer(ctx, tx, *bereitgestellt)
	if err != nil {
		log.Printf("ausleihe: Barcode des reservierten Exemplars %s nicht ladbar: %v", *bereitgestellt, err)
		return
	}
	erg.RegalfreigabeBarcode = barcode
}

// verarbeiteAusleihe bucht die Ausleihe eines freien Exemplars an den geöffneten Leser.
func (d *ausleihDienst) verarbeiteAusleihe(
	ctx context.Context,
	tx pgx.Tx,
	exemplar *repository.Exemplar,
	vorgang *ausleihVorgang,
	bearbeiterID string,
	erg *AusleihErgebnis,
) (*AusleihErgebnis, error) {
	ausleihe, err := d.legeAusleiheAn(ctx, tx, exemplar, vorgang, bearbeiterID)
	if err != nil {
		return nil, err
	}

	if vorgang.istSchueler() {
		entferneErfuellteVormerkung(ctx, tx, exemplar, vorgang.leserID, erg)
	}

	if err := d.protokollAbfragen.ProtokolliereAusleihe(ctx, tx, exemplar.ID, vorgang.leserID, "", bearbeiterID); err != nil {
		return nil, err
	}
	erg.Leser = vorgang.leser

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	protokolliereUebergangen(ctx, d.protokollAbfragen, bearbeiterID, vorgang.leserID, vorgang.uebergangen)

	erg.Art = "ausleihe"
	erg.Exemplar = exemplar
	if ausleihe != nil {
		erg.Frist = &ausleihe.RueckgabeFrist
	}
	if vorgang.istSchueler() && exemplar.IstLernmittel {
		erg.AuflagenHinweis = d.ladeAuflagenHinweis(ctx, exemplar.TitelID, vorgang.leserID)
	}
	return erg, nil
}

// ladeAuflagenHinweis fragt nach der Buchung, ob die Klasse gemischte Auflagen bekommt
// (repository.LadeAuflagenMischungInKlasse). Ein Fehler hier darf die gebuchte Ausleihe nicht
// als gescheitert melden: Er landet im Log, und die Theke zeigt keinen Hinweis.
func (d *ausleihDienst) ladeAuflagenHinweis(ctx context.Context, titelID, schuelerID string) *repository.AuflagenMischung {
	m, err := repository.LadeAuflagenMischungInKlasse(ctx, d.pool, titelID, schuelerID)
	if err != nil {
		log.Printf("auflagen-hinweis: %v", err)
		return nil
	}
	return m
}

// verarbeiteEigeneRueckgabe bucht die Rückgabe, wenn der geöffnete Leser sein eigenes Buch
// scannt.
func (d *ausleihDienst) verarbeiteEigeneRueckgabe(
	ctx context.Context,
	tx pgx.Tx,
	exemplar *repository.Exemplar,
	vorgang *ausleihVorgang,
	offeneAusleihe *repository.Ausleihe,
	bearbeiterID string,
	erg *AusleihErgebnis,
) (*AusleihErgebnis, error) {
	if err := d.ausleihAbfragen.BucheRueckgabe(ctx, tx, offeneAusleihe.ID, bearbeiterID, false); err != nil {
		return nil, err
	}

	// Die eigene Vormerkung des Schülers zählt nicht, sonst stellte er sich das Buch selbst
	// wieder bereit.
	if err := d.bedieneVormerkung(ctx, tx, exemplar, erg, offeneAusleihe.SchuelerID); err != nil {
		return nil, err
	}

	if err := d.protokollAbfragen.ProtokolliereRueckgabe(ctx, tx, exemplar.ID, vorgang.leserID, "", bearbeiterID); err != nil {
		return nil, err
	}
	erg.Leser = vorgang.leser

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	erg.Art = "rueckgabe"
	erg.Exemplar = exemplar
	erg.AusleiheID = &offeneAusleihe.ID
	return erg, nil
}

// verarbeiteFremdrueckgabe nimmt ein Buch zurück, das bei geöffnetem Leser gescannt wurde und
// auf jemand anderem steht. Es bucht nur die Rückgabe und nicht auf den geöffneten Leser um:
// Bringt ein Kind das Buch eines Freundes zurück, landete es sonst still auf dem falschen
// Konto. Soll der geöffnete Leser das Buch haben, leiht es der nächste Scan aus.
func (d *ausleihDienst) verarbeiteFremdrueckgabe(
	ctx context.Context,
	tx pgx.Tx,
	exemplar *repository.Exemplar,
	offeneAusleihe *repository.Ausleihe,
	bearbeiterID string,
	erg *AusleihErgebnis,
) (*AusleihErgebnis, error) {
	// Der Vorbesitzer kann ein Kollege sein: LeserAbfragen.Finde liest alle Leser, die Sicht
	// schueler nur die Schüler.
	var vorbesitzer *repository.Leser
	var err error

	if offeneAusleihe.SchuelerID != nil {
		vorbesitzer, err = d.leserAbfragen.Finde(ctx, *offeneAusleihe.SchuelerID)
		if err != nil {
			log.Printf("fremdrückgabe: Vorbesitzer konnte nicht geladen werden: %v", err)
		}
	}

	if err = d.ausleihAbfragen.BucheRueckgabe(ctx, tx, offeneAusleihe.ID, bearbeiterID, true); err != nil {
		return nil, err
	}

	// Wie bei jeder Rückgabe wird das Buch frei, und eine wartende Vormerkung bekommt es; die
	// des Vorbesitzers zählt nicht.
	if err := d.bedieneVormerkung(ctx, tx, exemplar, erg, offeneAusleihe.SchuelerID); err != nil {
		return nil, err
	}

	if offeneAusleihe.SchuelerID != nil {
		if err := d.protokollAbfragen.ProtokolliereRueckgabe(ctx, tx, exemplar.ID, *offeneAusleihe.SchuelerID, "", bearbeiterID); err != nil {
			return nil, err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	erg.Art = "rueckgabe"
	erg.Exemplar = exemplar
	erg.AusleiheID = &offeneAusleihe.ID
	// Leser ist der Vorbesitzer: Die Aktualisierung der anderen Arbeitsplätze trifft das Konto,
	// das sich geändert hat, und der geöffnete Leser bleibt, wie er ist.
	erg.Leser = vorbesitzer
	erg.Fremdrueckgabe = true
	erg.Vorbesitzer = vorbesitzer
	return erg, nil
}
