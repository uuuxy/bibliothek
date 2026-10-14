package service

import (
	"context"
	"fmt"

	"bibliothek/db"
	"bibliothek/repository"

	"github.com/jackc/pgx/v5"
)

// zaehleAktiveSchuelerAusleihen setzt einen Row-Level-Lock auf den Schüler
// (FOR UPDATE, gegen parallele Ausleihen) und zählt dessen reguläre (Nicht-LMF-)
// Ausleihen. Für Nicht-Schüler ist das Ergebnis 0.
func (d *ausleihDienst) zaehleAktiveSchuelerAusleihen(ctx context.Context, tx pgx.Tx, vorgang *ausleihVorgang) (int, error) {
	if !vorgang.istSchueler() {
		return 0, nil
	}
	if err := repository.SperreLeserzeile(ctx, tx, vorgang.leserID); err != nil {
		return 0, err
	}
	// Lernmittel zählen nicht ins Limit.
	return repository.ZaehleOffeneBuechereiAusleihen(ctx, tx, vorgang.leserID)
}

// istEigeneRueckgabe erkennt, ob der aktive Ausleiher sein eigenes Buch scannt
// (also eine reguläre Rückgabe statt einer Fremdrückgabe). Eine Spalte, ein Vergleich:
// Vorher waren es zwei, und ob der richtige griff, entschied der borrowerType.
func istEigeneRueckgabe(vorgang *ausleihVorgang, offeneAusleihe *repository.Ausleihe) bool {
	if offeneAusleihe == nil {
		return false
	}
	return offeneAusleihe.SchuelerID != nil && *offeneAusleihe.SchuelerID == vorgang.leserID
}

// pruefeSchuelerAusleihlimit erzwingt das Ausleihlimit für Schüler. Ausgenommen sind
// LMF-Bücher und jede Rückgabe — bei einer Rückgabe entsteht keine Ausleihe, die gegen
// das Limit zählen könnte.
func (d *ausleihDienst) pruefeSchuelerAusleihlimit(ctx context.Context, vorgang *ausleihVorgang, exemplar *repository.Exemplar, anzahlOffene int, neueAusleihe bool) error {
	if !vorgang.istSchueler() {
		return nil
	}
	if !neueAusleihe {
		return nil
	}
	einst, err := d.ladeEinstellungen(ctx)
	if err != nil {
		return err
	}
	if !exemplar.IstLernmittel && anzahlOffene >= einst.MaxAusleihenSchueler {
		return fmt.Errorf("%w: Ausleihlimit von %d Büchern überschritten (aktuell: %d)", ErrGesperrt, einst.MaxAusleihenSchueler, anzahlOffene)
	}
	return nil
}

// pruefeVormerkungKonflikt blockiert die Ausleihe, wenn das Exemplar für einen
// anderen Schüler abholbereit reserviert ist. Bei jeder Rückgabe entfällt die Prüfung —
// auch bei der Fremdrückgabe, die das Exemplar nur zurücknimmt (die Vormerkung bedient
// danach bedieneVormerkung).
func (d *ausleihDienst) pruefeVormerkungKonflikt(ctx context.Context, tx pgx.Tx, exemplarID string, vorgang *ausleihVorgang, neueAusleihe bool) error {
	if !neueAusleihe {
		return nil
	}
	reserviert, bereit, err := repository.FindeVormerkungAmExemplar(ctx, tx, exemplarID)
	if err != nil {
		return err
	}
	if bereit && (!vorgang.istSchueler() || vorgang.leserID != reserviert.LeserID) {
		return meldung(ErrKonflikt, "Achtung: dieses Exemplar ist noch für %s %s reserviert", reserviert.Vorname, reserviert.Nachname)
	}
	return nil
}

// BucheAmLeser wickelt die Ausleihe eines Buchexemplars an den aktiven LESER ab.
// Wenn das Buch bereits ausgeliehen ist, entscheidet die Methode, ob es sich um eine reguläre Rückgabe handelt
// (Ausleiher scannt sein eigenes Buch) oder um eine Fremdrückgabe mit anschließender Neuausleihe.
func (d *ausleihDienst) BucheAmLeser(
	ctx context.Context,
	exemplar *repository.Exemplar,
	aktiverLeserID *string,
	bearbeiterID string,
	uebergehen bool,
) (*AusleihErgebnis, error) {
	erg := &AusleihErgebnis{}

	// Sicherheitsschranke: Nur ausleihbare Exemplare dürfen verarbeitet werden
	if !exemplar.IstAusleihbar {
		return nil, meldung(ErrUngueltigerZustand, "Dieses Buchexemplar ist nicht ausleihbar")
	}

	// 1. Ausleiher und Frist auflösen (ausleih_vorgang.go). Die Sperrprüfung folgt
	// erst nach dem Lesen der Ausleihe — sie hängt davon ab, ob das Buch diesem Ausleiher gehört.
	vorgang, err := d.ladeVorgang(ctx, exemplar, aktiverLeserID)
	if err != nil {
		return nil, err
	}

	// 2. Transaktion gegen Race Conditions
	tx, err := d.ausleihAbfragen.BeginneTx(ctx)
	if err != nil {
		return nil, err
	}
	defer db.SafeRollback(ctx, tx)

	anzahlOffene, err := d.zaehleAktiveSchuelerAusleihen(ctx, tx, vorgang)
	if err != nil {
		return nil, err
	}

	offeneAusleihe, err := d.ausleihAbfragen.SperreOffene(ctx, tx, exemplar.ID)
	if err != nil {
		return nil, err
	}

	istEigene := istEigeneRueckgabe(vorgang, offeneAusleihe)
	// Eine Ausleihe entsteht nur am freien Exemplar. Beide Rückgabe-Fälle nehmen bloß
	// zurück — die Fremdrückgabe kennt den Ausleiher der Sitzung nicht einmal
	// (verarbeiteFremdrueckgabe bekommt keinen ausleihVorgang).
	neueAusleihe := offeneAusleihe == nil

	// 3. Die Schranken der Ausleihe — Sperrgründe, Ausleihlimit, fremde Vormerkung — nur,
	// wenn wirklich eine Ausleihe entsteht. Sie liefen bis zum 11.09.2026 beim Auflösen des
	// Ausleihers, bevor feststand, wem das Buch gehört (ein wegen Überfälligkeit gesperrtes
	// Kind wurde mit genau diesen Büchern abgewiesen), und danach bis zum 12.09.2026 an
	// `!isReturningThis` — was die Fremdrückgabe einschloss: Wer gesperrt oder am Limit war,
	// konnte das Buch eines Mitschülers nicht abgeben, während dieselbe Rückgabe ohne offene
	// Sitzung (NimmZurueck) durchging. Die Ausleihe liegt hier unter FOR UPDATE, der
	// Fall kann sich bis zum Commit nicht mehr ändern.
	//
	// Die Sperren gelten der Art nach (pruefeAusleihSperren: ein Kollege wird nie gesperrt);
	// bis zum 24.09.2026 stand dafür hier ein istSchueler() davor. Gezählt wird über den
	// Pool wie bisher; ein einfaches SELECT wartet auf keine Zeilensperre dieser Transaktion.
	if neueAusleihe {
		lage, err := pruefeAusleihSperren(ctx, d.pool, vorgang.leser, exemplar.IstLernmittel, uebergehen)
		if err != nil {
			return nil, err
		}
		vorgang.uebergangen = lage.uebergangen
	}

	if err := d.pruefeSchuelerAusleihlimit(ctx, vorgang, exemplar, anzahlOffene, neueAusleihe); err != nil {
		return nil, err
	}

	if err := d.pruefeVormerkungKonflikt(ctx, tx, exemplar.ID, vorgang, neueAusleihe); err != nil {
		return nil, err
	}

	// 5. Fall-Logik ausführen (ausleihe_faelle.go)
	if offeneAusleihe == nil {
		return d.verarbeiteAusleihe(ctx, tx, exemplar, vorgang, bearbeiterID, erg)
	}
	if istEigene {
		return d.verarbeiteEigeneRueckgabe(ctx, tx, exemplar, vorgang, offeneAusleihe, bearbeiterID, erg)
	}
	return d.verarbeiteFremdrueckgabe(ctx, tx, exemplar, offeneAusleihe, bearbeiterID, erg)
}
