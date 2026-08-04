package littera

import (
	"context"
	"fmt"
	"sort"
	"time"

	"bibliothek/internal/uebernahme"

	"github.com/jackc/pgx/v5"
)

// AusleihBericht ist die Bilanz des Ausleihteils.
type AusleihBericht struct {
	QuellAusleihen  int
	Geschrieben     int
	OhneExemplar    int // das Buch gibt es im übernommenen Bestand nicht
	OhneEntleiher   int // die Person wurde nicht übernommen
	Doppelbelegung  int // zweite offene Ausleihe desselben Exemplars
	Uebersprungen   int // an einem Schreibfehler gescheitert
	FristVorAusgabe int

	IstAusleihen int
	AbgleichOK   bool
}

const sqlAusleiheEinfuegen = `
	INSERT INTO ausleihen
		(exemplar_id, schueler_id, ausleiher_benutzer_id,
		 ausgeliehen_am, rueckgabe_frist, rueckgabe_am,
		 ist_handapparat, mahnstufe)
	VALUES ($1,$2,$3,$4,$5,$6,$7,$8)`

// SchreibeAusleihen überträgt die Ausleihen.
//
// Der Fremdschlüssel auf den Entleiher zeigt in Littera auf Leser.BUCHUNGSNUMMER, nicht
// auf die Lesernummer — über die zweite träfen nur 792 von 15.615 Zeilen, und die
// getroffenen wären die falschen Personen.
//
// Geschrieben wird nur, was beide Enden findet. Eine Ausleihe ins Leere wäre ein
// Fremdschlüsselfehler und risse den Lauf mitten in der Transaktion ab; sie
// stillschweigend zu verwerfen verschwiege, dass ein Buch verliehen war. Also: einzeln
// überspringen und einzeln melden.
func (s *Schreiber) SchreibeAusleihen(
	ctx context.Context, ab *Altbestand, bestand BestandBericht, personen PersonenBericht,
) (AusleihBericht, error) {
	bericht := AusleihBericht{QuellAusleihen: len(ab.Ausleihen)}

	vorher, err := s.zaehle(ctx, `SELECT count(*) FROM ausleihen`)
	if err != nil {
		return bericht, err
	}

	lauf := &ausleihlauf{s: s, bericht: &bericht,
		exemplare: bestand.ExemplarIDs, entleiher: personen.EntleiherIDs}
	if err := lauf.alleBatches(ctx, lauf.aussortieren(ab.Ausleihen)); err != nil {
		return bericht, err
	}

	nachher, err := s.zaehle(ctx, `SELECT count(*) FROM ausleihen`)
	if err != nil {
		return bericht, fmt.Errorf("der Abgleich nach der Übernahme schlug fehl: %w", err)
	}
	bericht.IstAusleihen = nachher - vorher
	bericht.AbgleichOK = bericht.IstAusleihen == bericht.Geschrieben
	return bericht, nil
}

// buchbar ist eine Ausleihe, die beide Enden gefunden hat — mit dem bereits geklärten
// Rückgabedatum. Es wird einmal bestimmt und nicht zweimal, damit die Frage „bleibt diese
// Ausleihe offen" bei der Doppelbelegungsprüfung dieselbe Antwort gibt wie beim INSERT.
type buchbar struct {
	Ausleihe
	RueckgabeAm *time.Time
}

func (b buchbar) bleibtOffen() bool { return b.RueckgabeAm == nil }

type ausleihlauf struct {
	s         *Schreiber
	bericht   *AusleihBericht
	exemplare map[string]string    // Littera-Exemplar → UUID
	entleiher map[string]Entleiher // Littera-Leser → Ziel
}

// aussortieren siebt die nicht buchbaren Zeilen aus, bevor auch nur eine Transaktion
// aufgeht — jede von ihnen wird einzeln protokolliert.
//
// Die Doppelbelegung ist die unauffälligste der drei: uniq_ausleihen_aktiv_exemplar
// lässt höchstens EINE offene Ausleihe je Exemplar zu. Im Altbestand tragen zwei
// Exemplare je zwei offene Ausleihen — ein Buch kann nicht bei zwei Leuten liegen. Es
// gewinnt die jüngste; die ältere ist die Karteileiche.
func (l *ausleihlauf) aussortieren(ausleihen []Ausleihe) []buchbar {
	sortiert := make([]Ausleihe, len(ausleihen))
	copy(sortiert, ausleihen)
	sort.SliceStable(sortiert, func(i, j int) bool {
		return sortiert[i].AusgeliehenAm.After(sortiert[j].AusgeliehenAm)
	})

	offenJeExemplar := map[string]bool{}
	liste := make([]buchbar, 0, len(sortiert))
	for _, a := range sortiert {
		if !l.beideEndenGefunden(a) {
			continue
		}
		b := buchbar{Ausleihe: a, RueckgabeAm: l.rueckgabeAm(a)}
		if b.bleibtOffen() && offenJeExemplar[a.ExemplarID] {
			l.bericht.Doppelbelegung++
			l.s.prot.Fehler(a.ID, a.ExemplarID,
				"zweite offene Ausleihe desselben Exemplars – ein Buch liegt nur bei einer Person; "+
					"die jüngere Ausleihe wurde übernommen, diese nicht")
			continue
		}
		if b.bleibtOffen() {
			offenJeExemplar[a.ExemplarID] = true
		}
		l.meldeFristVorAusgabe(a)
		liste = append(liste, b)
	}
	return liste
}

func (l *ausleihlauf) beideEndenGefunden(a Ausleihe) bool {
	if _, ok := l.exemplare[a.ExemplarID]; !ok {
		l.bericht.OhneExemplar++
		l.s.prot.Fehler(a.ID, a.ExemplarID,
			"das ausgeliehene Exemplar steht nicht im übernommenen Bestand – Ausleihe nicht übernommen")
		return false
	}
	if _, ok := l.entleiher[a.LeserID]; !ok {
		l.bericht.OhneEntleiher++
		l.s.prot.Fehler(a.ID, a.LeserID,
			"der Entleiher wurde nicht übernommen (Sammelkonto, unklare Gruppe oder Fehler) – "+
				"Ausleihe nicht übernommen")
		return false
	}
	return true
}

func (l *ausleihlauf) meldeFristVorAusgabe(a Ausleihe) {
	if a.Frist.IsZero() || a.AusgeliehenAm.IsZero() || !a.Frist.Before(a.AusgeliehenAm) {
		return
	}
	l.bericht.FristVorAusgabe++
	l.s.prot.Warnung(a.ID, a.ExemplarID,
		"Rückgabefrist liegt vor dem Verleihdatum – so aus Littera übernommen, "+
			"die Ausleihe erscheint sofort als überfällig")
}

// rueckgabeAm liefert das Rückgabedatum, aber nur für tatsächlich zurückgegebene Ausleihen.
//
// check_return_date verlangt rueckgabe_am >= ausgeliehen_am. Ein Rückgabedatum vor der
// Ausgabe ist ein Datenfehler in Littera; die Zeile deswegen zu verlieren wäre unnötig,
// also wird das Datum verworfen und die Ausleihe bleibt offen. Ebenso bei „zurückgegeben"
// ohne Datum: Ohne rueckgabe_am gilt die Zeile in dieser Anwendung als laufend.
func (l *ausleihlauf) rueckgabeAm(a Ausleihe) *time.Time {
	if !a.Zurueckgegeben || a.RueckgabeAm.IsZero() {
		return nil
	}
	if a.RueckgabeAm.Before(a.AusgeliehenAm) {
		l.s.prot.Warnung(a.ID, a.ExemplarID,
			"Rückgabedatum liegt vor dem Verleihdatum – verworfen, die Ausleihe bleibt offen")
		return nil
	}
	t := a.RueckgabeAm
	return &t
}

func (l *ausleihlauf) alleBatches(ctx context.Context, ausleihen []buchbar) error {
	for i := 0; i < len(ausleihen); i += l.s.opt.BatchGroesse {
		ende := min(i+l.s.opt.BatchGroesse, len(ausleihen))
		if err := l.einBatch(ctx, ausleihen[i:ende]); err != nil {
			return fmt.Errorf("abgebrochen ab Ausleihe %s: %w", ausleihen[i].ID, err)
		}
	}
	return nil
}

func (l *ausleihlauf) einBatch(ctx context.Context, batch []buchbar) error {
	tx, err := l.s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("konnte die Transaktion nicht öffnen: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }() //nolint:errcheck

	for _, a := range batch {
		if err := l.eineAusleihe(ctx, tx, a); err != nil {
			return err
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("der COMMIT schlug fehl – keine Ausleihe dieses Batches wurde geschrieben: %w", err)
	}
	return nil
}

func (l *ausleihlauf) eineAusleihe(ctx context.Context, tx pgx.Tx, a buchbar) error {
	erg, err := uebernahme.ImSavepoint(ctx, tx, "littera_id="+a.ID, func(sp pgx.Tx) error {
		return l.schreibeAusleihe(ctx, sp, a)
	})
	if err != nil {
		return err
	}
	if !erg.Uebernommen {
		l.bericht.Uebersprungen++
		l.s.prot.Fehler(a.ID, a.ExemplarID,
			"Ausleihe übersprungen – "+uebernahme.BeschreibeFehler(erg.Zurueckgerollt))
		return nil
	}
	l.bericht.Geschrieben++
	return nil
}

func (l *ausleihlauf) schreibeAusleihe(ctx context.Context, tx pgx.Tx, a buchbar) error {
	person := l.entleiher[a.LeserID]

	// ist_handapparat kennzeichnet die Ausleihe an eine Lehrkraft — dieselbe Bedeutung,
	// die die Anwendung dem Feld gibt.
	istLehrkraft := person.BenutzerID != ""

	_, err := tx.Exec(ctx, sqlAusleiheEinfuegen,
		l.exemplare[a.ExemplarID],
		uebernahme.Nullbar(person.SchuelerID),
		uebernahme.Nullbar(person.BenutzerID),
		a.AusgeliehenAm,
		l.frist(a.Ausleihe),
		a.RueckgabeAm,
		istLehrkraft,
		a.Mahnungen,
	)
	if err != nil {
		return fmt.Errorf("bei der Ausleihe von Exemplar %s: %w", a.ExemplarID, err)
	}
	return nil
}

// frist liefert die Rückgabefrist. ausleihen.rueckgabe_frist ist NOT NULL; im Altbestand
// trägt jede der 15.615 Zeilen eine Frist. Fehlt sie doch einmal, gilt das Verleihdatum —
// die Ausleihe erscheint dann als überfällig, was näher an der Wahrheit liegt als eine
// erfundene Frist in der Zukunft.
func (l *ausleihlauf) frist(a Ausleihe) time.Time {
	if !a.Frist.IsZero() {
		return a.Frist
	}
	l.s.prot.Warnung(a.ID, a.ExemplarID,
		"keine Rückgabefrist im Altbestand – Verleihdatum eingesetzt (rueckgabe_frist ist NOT NULL)")
	return a.AusgeliehenAm
}
