package service

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"bibliothek/pkg/betrag"
	"bibliothek/pkg/schulzeit"
	"bibliothek/repository"

	"github.com/jackc/pgx/v5"
)

// bucheRueckgabeNach bucht die Absicht „Rückgabe". Lag das Buch bei jemandem, endet die
// Ausleihe zum Zeitpunkt des Scans, auch wenn es nicht die Person des Eintrags war
// (Fremdrückgabe). War es nicht verliehen, gibt es nichts zu buchen: nur_reaktiviert, wenn es
// abgeschrieben war (dann ist die Forderung beendet), sonst nicht_gebucht.
func (d *AusleihDienst) bucheRueckgabeNach(ctx context.Context, tx pgx.Tx, l *nachbuchLage) (*NachbuchErgebnis, error) {
	if l.offeneAusleihe == nil {
		return d.verarbeiteRueckgabeOhneAusleihe(ctx, tx, l)
	}

	erg := &AusleihErgebnis{}
	fremd := l.leser != nil && !istEigeneAusleihe(l)
	vorbesitzer, err := d.beendeAusleiheZumScan(ctx, tx, l, fremd, erg)
	if errors.Is(err, errScanVeraltet) {
		verwerfeStill(ctx, tx)
		return d.meldeAbweisung(ctx, l, repository.NachbuchVeraltet, "Rückgabe liegt vor der Ausleihe")
	}
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	erg.Art = "rueckgabe"
	erg.Exemplar = l.exemplar
	erg.AusleiheID = &l.offeneAusleihe.ID
	erg.Fremdrueckgabe = fremd
	erg.Vorbesitzer = vorbesitzer
	if !fremd {
		erg.Leser = l.leser
	}
	return &NachbuchErgebnis{Ergebnis: repository.NachbuchZurueckgegeben, Gebucht: erg}, nil
}

// verarbeiteRueckgabeOhneAusleihe beantwortet eine Rückgabe, bei der das Buch nicht verliehen
// war. War es abgeschrieben, ist es jetzt wieder im Umlauf (nur_reaktiviert), und die Meldung
// nennt, was mit der Forderung geschah; sonst nicht_gebucht.
func (d *AusleihDienst) verarbeiteRueckgabeOhneAusleihe(ctx context.Context, tx pgx.Tx, l *nachbuchLage) (*NachbuchErgebnis, error) {
	if !l.reaktiviert {
		verwerfeStill(ctx, tx)
		return d.meldeAbweisung(ctx, l, repository.NachbuchNichtGebucht, "Buch war nicht ausgeliehen")
	}
	hinweis := l.befund.AufsichtHinweis()
	grund := "Buch war abgeschrieben und ist wieder im Umlauf"
	if l.befund.StornierteForderungen > 0 {
		grund = fmt.Sprintf("Buch war abgeschrieben; Forderung über %s storniert", betrag.Euro(l.befund.StornierterBetrag))
	}
	if hinweis != "" {
		grund = hinweis
	}
	if err := repository.SchreibeNachbuchMeldung(ctx, tx, d.baueMeldung(l, repository.NachbuchNurReaktiviert, grund)); err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return &NachbuchErgebnis{Ergebnis: repository.NachbuchNurReaktiviert, Grund: grund, AufsichtHinweis: hinweis}, nil
}

// bucheAusleiheNach bucht die Absicht „Ausleihe", die ein Scan mit Person ohne Netz trägt. Liegt
// das Buch schon bei dieser Person, entscheidet bucheEigeneAusleiheNach. Lag es bei jemand
// anderem, wird dort zurückgenommen, vor dem Savepoint, damit die Rücknahme bleibt, wenn die neue
// Ausleihe an Sperre, Limit oder Vormerkung scheitert.
func (d *AusleihDienst) bucheAusleiheNach(ctx context.Context, tx pgx.Tx, l *nachbuchLage) (*NachbuchErgebnis, error) {
	if l.offeneAusleihe != nil && istEigeneAusleihe(l) {
		return d.bucheEigeneAusleiheNach(ctx, tx, l)
	}

	erg := &AusleihErgebnis{}
	var vorbesitzer *repository.Leser
	if l.offeneAusleihe != nil {
		var err error
		vorbesitzer, err = d.beendeAusleiheZumScan(ctx, tx, l, true, erg)
		if errors.Is(err, errScanVeraltet) {
			verwerfeStill(ctx, tx)
			return d.meldeAbweisung(ctx, l, repository.NachbuchVeraltet, "Scan liegt vor der Ausleihe des Vorbesitzers")
		}
		if err != nil {
			return nil, err
		}
	}

	// Savepoint: Was ab hier scheitert, nimmt die Rücknahme nicht mit.
	ausleihe, abweisung, err := d.leiheImSavepointAus(ctx, tx, l, erg)
	if abweisung != nil || err != nil {
		return abweisung, err
	}

	ergebnis := repository.NachbuchAusgeliehen
	if vorbesitzer != nil {
		ergebnis = repository.NachbuchUmgebucht
		m := d.baueMeldung(l, ergebnis, "lag bei jemand anderem — dort zurückgenommen, neu ausgeliehen")
		m.VorbesitzerSchuelerID = &vorbesitzer.ID
		if err := repository.SchreibeNachbuchMeldung(ctx, tx, m); err != nil {
			return nil, err
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	erg.Art = "ausleihe"
	erg.Exemplar = l.exemplar
	erg.Leser = l.leser
	erg.Vorbesitzer = vorbesitzer
	erg.Fremdrueckgabe = vorbesitzer != nil
	if ausleihe != nil {
		erg.Frist = &ausleihe.RueckgabeFrist
		erg.AusleiheID = &ausleihe.ID
	}
	return &NachbuchErgebnis{Ergebnis: ergebnis, Gebucht: erg, AufsichtHinweis: l.befund.AufsichtHinweis()}, nil
}

// bucheEigeneAusleiheNach: Das Buch liegt schon bei der Person des Eintrags. Lag es schon vor dem
// Besuch dort, ist der Scan eine Rückgabe, wie am Netz (BucheAmLeser). Entstand ihre Ausleihe
// erst in diesem Besuch, wird nichts umgekehrt (bereits_ausgeliehen): Das ist der Doppelscan, oder
// das Nachbuchen hat den verwaisten Schlüssel einer Buchung übernommen, die schon steht.
func (d *AusleihDienst) bucheEigeneAusleiheNach(ctx context.Context, tx pgx.Tx, l *nachbuchLage) (*NachbuchErgebnis, error) {
	if l.ausleiheVorDemBesuch() {
		return d.bucheRueckgabeNach(ctx, tx, l)
	}
	verwerfeStill(ctx, tx)
	erg := &AusleihErgebnis{Art: "ausleihe", Exemplar: l.exemplar, Leser: l.leser, AusleiheID: &l.offeneAusleihe.ID}
	frist := l.offeneAusleihe.RueckgabeFrist
	erg.Frist = &frist
	return &NachbuchErgebnis{Ergebnis: repository.NachbuchBereitsAusgeliehen, Gebucht: erg}, nil
}

// besuchsToleranz trennt eine Ausleihe von vor dem Besuch von der aus seinem ersten Scan:
// Besuchsbeginn und Scan-Zeitpunkt kommen aus zwei Uhren des Rechners, auf Millisekunden
// gerundet.
const besuchsToleranz = 2 * time.Second

// ausleiheVorDemBesuch sagt, ob die offene Ausleihe der Person schon bestand, bevor sie an die
// Theke trat. Ohne Besuchsbeginn (ein Rechner, der ihn nicht schickt) heißt das nein.
func (l *nachbuchLage) ausleiheVorDemBesuch() bool {
	return l.besuchBeginn != nil &&
		l.offeneAusleihe.AusgeliehenAm.Before(l.besuchBeginn.Add(-besuchsToleranz))
}

// leiheImSavepointAus prüft die Schranken und legt die Ausleihe an, beides im Savepoint.
// Scheitert es an Sperre, Limit, Vormerkung oder an einer gleichzeitigen Buchung, ist der
// Eintrag abgewiesen: Die Abweisung kommt als zweiter Wert zurück, und die Transaktion ist dann
// abgeschlossen (weiseImSavepointAb).
func (d *AusleihDienst) leiheImSavepointAus(ctx context.Context, tx pgx.Tx, l *nachbuchLage, erg *AusleihErgebnis) (*repository.Ausleihe, *NachbuchErgebnis, error) {
	sp, err := tx.Begin(ctx)
	if err != nil {
		return nil, nil, err
	}
	vorgang, err := d.baueNachbuchVorgang(ctx, tx, l)
	if err != nil {
		return nil, nil, err
	}
	if err := d.pruefeNachbuchSchranken(ctx, sp, l, vorgang); err != nil {
		if !errors.Is(err, ErrGesperrt) && !errors.Is(err, ErrKonflikt) {
			return nil, nil, err
		}
		abweisung, abwErr := d.weiseImSavepointAb(ctx, tx, sp, l, MeldungAus(err), err)
		return nil, abweisung, abwErr
	}

	ausleihe, err := repository.LegeAusleiheAn(ctx, sp, repository.AusleiheNeu{
		ExemplarID:     l.exemplar.ID,
		LeserID:        l.leser.ID,
		BearbeiterID:   l.eintrag.BearbeiterID,
		RueckgabeFrist: vorgang.frist,
		IstDauerleihe:  !vorgang.istSchueler(),
		Zeitpunkt:      &l.gescannt,
	})
	if errors.Is(err, repository.ErrAusleiheKonflikt) {
		abweisung, abwErr := d.weiseImSavepointAb(ctx, tx, sp, l, "Exemplar wurde soeben an einem anderen Arbeitsplatz verbucht", nil)
		return nil, abweisung, abwErr
	}
	if err != nil {
		return nil, nil, err
	}
	if vorgang.istSchueler() {
		entferneErfuellteVormerkung(ctx, sp, l.exemplar, l.leser.ID, erg)
	}
	if err := d.protokollAbfragen.ProtokolliereAusleihe(ctx, sp, l.exemplar.ID, l.leser.ID, "", l.eintrag.BearbeiterID); err != nil {
		return nil, nil, err
	}
	if err := sp.Commit(ctx); err != nil {
		return nil, nil, err
	}
	return ausleihe, nil, nil
}

// weiseImSavepointAb nimmt den Savepoint zurück und schließt die Transaktion ab — die
// Rücknahme beim Vorbesitzer bleibt —, dann schreibt es die Abweisung als Meldung. schranke
// ist der Fehler der Schranke, wenn es an einer lag; er reist mit dem Ergebnis zur Tür.
func (d *AusleihDienst) weiseImSavepointAb(ctx context.Context, tx, sp pgx.Tx, l *nachbuchLage, grund string, schranke error) (*NachbuchErgebnis, error) {
	if err := sp.Rollback(ctx); err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	abweisung, err := d.meldeAbweisung(ctx, l, repository.NachbuchNichtGebucht, grund)
	if err != nil {
		return nil, err
	}
	abweisung.Schranke = schranke
	return abweisung, nil
}

// errScanVeraltet meldet, dass die Rückgabe vor der Ausleihe läge (check_return_date).
var errScanVeraltet = errors.New("scan veraltet")

// beendeAusleiheZumScan beendet die offene Ausleihe zum Zeitpunkt des Scans, bedient die
// Vormerkung und schreibt das Protokoll; es liefert den Vorbesitzer für Ergebnis und Meldung.
func (d *AusleihDienst) beendeAusleiheZumScan(ctx context.Context, tx pgx.Tx, l *nachbuchLage, fremd bool, erg *AusleihErgebnis) (*repository.Leser, error) {
	aktiv := l.offeneAusleihe
	if err := repository.BucheRueckgabeZum(ctx, tx, aktiv.ID, l.eintrag.BearbeiterID, fremd, &l.gescannt); err != nil {
		if istPruefungVerletzt(err) {
			return nil, errScanVeraltet
		}
		return nil, err
	}
	if err := d.bedieneVormerkung(ctx, tx, l.exemplar, erg, aktiv.SchuelerID); err != nil {
		return nil, err
	}
	if aktiv.SchuelerID == nil {
		return nil, nil
	}
	if err := d.protokollAbfragen.ProtokolliereRueckgabe(ctx, tx, l.exemplar.ID, *aktiv.SchuelerID, "", l.eintrag.BearbeiterID); err != nil {
		return nil, err
	}
	if !fremd {
		return nil, nil
	}
	return d.leserAbfragen.Finde(ctx, *aktiv.SchuelerID)
}

// baueNachbuchVorgang baut den Vorgang der Ausleihe mit der Frist ab dem Scan.
func (d *AusleihDienst) baueNachbuchVorgang(ctx context.Context, tx pgx.Tx, l *nachbuchLage) (*ausleihVorgang, error) {
	vorgang := &ausleihVorgang{leserID: l.leser.ID, leser: l.leser}
	if !vorgang.istSchueler() {
		vorgang.frist = schulzeit.TagesEnde(l.gescannt.AddDate(1, 0, 0))
		return vorgang, nil
	}
	frist, err := d.bestimmeFristAm(ctx, l.exemplar, l.leser.Klasse, l.gescannt)
	if err != nil {
		return nil, err
	}
	vorgang.frist = frist
	return vorgang, nil
}

// pruefeNachbuchSchranken prüft dieselben Schranken wie der Scan am Netz (Sperren,
// Ausleihlimit, fremde Vormerkung); gezählt wird in der Transaktion des Eintrags. Übergehen
// gibt es beim Nachbuchen nicht, weil niemand daneben steht.
func (d *AusleihDienst) pruefeNachbuchSchranken(ctx context.Context, q pgx.Tx, l *nachbuchLage, vorgang *ausleihVorgang) error {
	if _, err := pruefeAusleihSperren(ctx, q, l.leser, l.exemplar.IstLernmittel, false); err != nil {
		return err
	}
	if vorgang.istSchueler() {
		anzahl, err := d.zaehleAktiveSchuelerAusleihen(ctx, q, vorgang)
		if err != nil {
			return err
		}
		if err := d.pruefeSchuelerAusleihlimit(ctx, vorgang, l.exemplar, anzahl, true); err != nil {
			return err
		}
	}
	return d.pruefeVormerkungKonflikt(ctx, q, l.exemplar.ID, vorgang, true)
}

// verwerfeStill verwirft die Transaktion, wenn fachlich nichts zu buchen ist. Ein Fehler dabei
// steht im Log und wird nicht gemeldet: Gebucht wurde ohnehin nichts, und der Aufrufer soll
// seine fachliche Antwort behalten.
func verwerfeStill(ctx context.Context, tx pgx.Tx) {
	if err := tx.Rollback(ctx); err != nil && !errors.Is(err, pgx.ErrTxClosed) {
		log.Printf("nachbuchen: Rollback fehlgeschlagen: %v", err)
	}
}
