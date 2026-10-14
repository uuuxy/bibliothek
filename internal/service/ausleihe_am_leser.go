package service

import (
	"context"
	"fmt"

	"bibliothek/db"
	"bibliothek/repository"

	"github.com/jackc/pgx/v5"
)

// zaehleAktiveSchuelerAusleihen sperrt die Zeile des Schülers (FOR UPDATE), damit zwei
// gleichzeitige Ausleihen das Limit nicht gemeinsam überschreiten, und zählt seine Ausleihen
// außer den Lernmitteln. Für alle anderen Leser ist das Ergebnis 0.
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

// istEigeneRueckgabe sagt, ob der geöffnete Leser sein eigenes Buch scannt; dann ist es eine
// gewöhnliche Rückgabe und keine Fremdrückgabe.
func istEigeneRueckgabe(vorgang *ausleihVorgang, offeneAusleihe *repository.Ausleihe) bool {
	if offeneAusleihe == nil {
		return false
	}
	return offeneAusleihe.SchuelerID != nil && *offeneAusleihe.SchuelerID == vorgang.leserID
}

// pruefeSchuelerAusleihlimit hält das Ausleihlimit eines Schülers. Lernmittel und jede Rückgabe
// sind ausgenommen: Bei einer Rückgabe entsteht keine Ausleihe, die zählen könnte.
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

// pruefeVormerkungKonflikt weist die Ausleihe ab, wenn das Exemplar für einen anderen Schüler
// abholbereit liegt. Bei jeder Rückgabe entfällt die Prüfung, auch bei der Fremdrückgabe, die
// das Exemplar nur zurücknimmt; die Vormerkung bedient danach bedieneVormerkung.
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

// BucheAmLeser leiht ein freies Exemplar dem geöffneten Leser aus. Ist es verliehen, ist der
// Scan eine Rückgabe: beim Leser selbst eine gewöhnliche, bei jemand anderem eine
// Fremdrückgabe.
func (d *ausleihDienst) BucheAmLeser(
	ctx context.Context,
	exemplar *repository.Exemplar,
	aktiverLeserID *string,
	bearbeiterID string,
	uebergehen bool,
) (*AusleihErgebnis, error) {
	erg := &AusleihErgebnis{}

	if !exemplar.IstAusleihbar {
		return nil, meldung(ErrUngueltigerZustand, "Dieses Buchexemplar ist nicht ausleihbar")
	}

	// Die Sperren prüft erst der Schritt nach dem Lesen der Ausleihe: Sie hängen davon ab, ob
	// das Buch diesem Leser gehört.
	vorgang, err := d.ladeVorgang(ctx, exemplar, aktiverLeserID)
	if err != nil {
		return nil, err
	}

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
	// Eine Ausleihe entsteht nur am freien Exemplar. Beide Rückgaben nehmen bloß zurück; die
	// Fremdrückgabe bekommt den Vorgang nicht einmal.
	neueAusleihe := offeneAusleihe == nil

	// Die Schranken der Ausleihe (Sperren, Ausleihlimit, fremde Vormerkung) gelten nur, wenn
	// wirklich eine Ausleihe entsteht: Wer gesperrt oder am Limit ist, muss sein Buch und auch
	// das eines Mitschülers abgeben können. Die Ausleihe liegt hier unter FOR UPDATE, der Fall
	// ändert sich bis zum Commit nicht mehr.
	//
	// Die Sperren gelten der Art des Lesers nach (pruefeAusleihSperren: ein Kollege wird nie
	// gesperrt). Gezählt wird über den Pool; ein einfaches SELECT wartet auf keine Zeilensperre
	// dieser Transaktion.
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

	if offeneAusleihe == nil {
		return d.verarbeiteAusleihe(ctx, tx, exemplar, vorgang, bearbeiterID, erg)
	}
	if istEigene {
		return d.verarbeiteEigeneRueckgabe(ctx, tx, exemplar, vorgang, offeneAusleihe, bearbeiterID, erg)
	}
	return d.verarbeiteFremdrueckgabe(ctx, tx, exemplar, offeneAusleihe, bearbeiterID, erg)
}
