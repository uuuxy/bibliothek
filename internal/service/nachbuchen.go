package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"bibliothek/db"
	"bibliothek/pkg/schulzeit"
	"bibliothek/repository"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// Das Nachbuchen: Ein Vorgang, den die Theke ohne Netz gescannt hat, kommt später an. Gebucht
// wird die Wirklichkeit, nicht der Scan: Lag das Buch inzwischen bei jemand anderem, wird dort
// zurückgenommen und neu ausgeliehen; beim selben Kind wird nichts umgekehrt; ist der Scan
// älter als die letzte Bewegung des Exemplars, wird er abgewiesen. Was vom Scan abweicht, steht
// als Meldung, nichts verschwindet still.
//
// Je Eintrag eine Transaktion, gesperrt in der Reihenfolge Schüler, Ausleihe, Exemplar
// (docs/invarianten.md, Abschnitt 1) wie beim Scan am Netz, sonst verklemmen sie sich
// gegeneinander. Die Rücknahme beim Vorbesitzer steht vor dem Savepoint und bleibt, wenn die
// neue Ausleihe an Sperre, Limit oder Vormerkung scheitert; nur die Ausleihe meldet dann
// nicht_gebucht. Gerechnet wird ab dem Scan, höchstens ab der Zeit des Servers: Ausleih- und
// Rückgabedatum, Frist und Bewegungsstempel.

// NachbuchDienst bucht die Einträge nach, die eine Theke ohne Netz gesammelt hat. Er bucht über
// ausleihDienst, damit das Nachbuchen dieselben Schranken und Schreiber hat wie der Scan am Netz.
type NachbuchDienst struct {
	ausleihe *ausleihDienst
}

// NewNachbuchDienst baut den Dienst des Nachbuchens.
func NewNachbuchDienst(pool db.Pool, leserAbfragen leserErmittlung, katalogAbfragen *repository.KatalogAbfragen, ausleihAbfragen repository.AusleihAbfragen, protokollAbfragen repository.ProtokollAbfragen) *NachbuchDienst {
	return &NachbuchDienst{ausleihe: &ausleihDienst{pool: pool, leserAbfragen: leserAbfragen, katalogAbfragen: katalogAbfragen, ausleihAbfragen: ausleihAbfragen, protokollAbfragen: protokollAbfragen}}
}

// Absichten eines Eintrags: was der Theken-Rechner beim Scan meinte.
const (
	NachbuchAbsichtAusleihe  = "ausleihe"
	NachbuchAbsichtRueckgabe = "rueckgabe"
)

// NachbuchEintrag ist ein Eintrag der Warteschlange eines Theken-Rechners.
type NachbuchEintrag struct {
	Schluessel string // Idempotenz-Schlüssel des Eintrags (UUID)
	Absicht    string // NachbuchAbsichtAusleihe | NachbuchAbsichtRueckgabe
	Barcode    string // Buch-Barcode wie gescannt (B-…, Ziffern, LMF-…)
	GescanntAm time.Time
	// UhrVersatz ist der gemessene Versatz der Uhr des Theken-Rechners: Serverzeit beim Empfang
	// der Portion minus Sendezeit des Rechners. Er wird auf GescanntAm addiert, bevor der Wächter
	// vergleicht — Bewegungsstempel sind Serverzeit. Ohne ihn beendete eine offline gebuchte
	// Rückgabe von einem vorgehenden Rechner die jüngere Online-Ausleihe eines anderen Kindes.
	UhrVersatz     time.Duration
	LeserID        *string // Person, wenn der Rechner sie beim Scan noch auflösen konnte
	AusweisBarcode *string // sonst der offline gescannte Ausweis
	// NachFremdrueckgabeVon ist gesetzt, wenn der Versand am Netz unter demselben Schlüssel nur
	// die Fremdrückgabe dieser Ausleihe gebucht hat (das Buch stand auf jemand anderem) und
	// seine Antwort nicht ankam. Die Ausleihe an die Person des Eintrags fehlt noch. Der
	// Wächter lässt den Eintrag durch, wenn sich das Exemplar seit dieser Rückgabe nicht bewegt
	// hat; gebucht wird frühestens ab der Rückgabe. Nach einer vollständigen Ausleihe gilt es
	// nicht: Das Nachbuchen liehe ein inzwischen zurückgegebenes Buch sonst erneut aus.
	NachFremdrueckgabeVon *string
	BearbeiterID          string
}

// NachbuchErgebnis ist die Antwort je Eintrag.
type NachbuchErgebnis struct {
	Ergebnis string // repository.Nachbuch*
	Grund    string
	// Schranke ist bei nicht_gebucht der Fehler der Schranke, an der die Ausleihe scheiterte.
	// Grund trägt seinen vollen Text, auch den Freitext einer Sperre (SperrgrundFehler); die Tür
	// kürzt ihn daran für Aufrufer ohne view_students. Die Meldung behält den vollen Grund.
	Schranke        error
	Gebucht         *AusleihErgebnis // bei ausgeliehen, umgebucht, bereits_ausgeliehen, zurueckgegeben
	AufsichtHinweis string           // das Buch stand auf einem Bescheid bei der Aufsicht
}

// nachbuchLage ist der gesperrte Zustand eines Eintrags in seiner Transaktion.
type nachbuchLage struct {
	eintrag        NachbuchEintrag
	gescannt       time.Time
	exemplar       *repository.Exemplar
	leser          *repository.Leser
	ausweis        *string // nicht auflösbarer Ausweis-Barcode, als Text für die Meldung
	offeneAusleihe *repository.Ausleihe
	befund         repository.RueckkehrBefund
	reaktiviert    bool
}

// BucheNach bucht einen Eintrag. Fachliche Ausgänge kommen als Ergebnis zurück; ein
// Fehler ist ein Serverfehler (Datenbank), und der Eintrag bleibt auf dem Rechner liegen.
func (d *NachbuchDienst) BucheNach(ctx context.Context, eintrag NachbuchEintrag) (*NachbuchErgebnis, error) {
	return d.ausleihe.bucheNach(ctx, eintrag)
}

// bucheNach bucht einen Eintrag in einer Transaktion (NachbuchDienst.BucheNach).
func (d *ausleihDienst) bucheNach(ctx context.Context, eintrag NachbuchEintrag) (*NachbuchErgebnis, error) {
	l, abweisung, err := d.loeseNachbuchEintrag(ctx, eintrag)
	if abweisung != nil || err != nil {
		return abweisung, err
	}

	tx, err := d.ausleihAbfragen.BeginneTx(ctx)
	if err != nil {
		return nil, err
	}
	defer db.Verwerfe(ctx, tx)

	// Sperren: Leser, dann die Ausleihe des Exemplars, dann das Exemplar.
	if l.leser != nil {
		if err := repository.SperreLeserzeile(ctx, tx, l.leser.ID); err != nil {
			return nil, err
		}
	}
	if l.offeneAusleihe, err = d.ausleihAbfragen.SperreOffene(ctx, tx, l.exemplar.ID); err != nil {
		return nil, err
	}
	stand, err := repository.SperreExemplarzeile(ctx, tx, l.exemplar.ID)
	if err != nil {
		return nil, err
	}

	if err := verlegeScanAufFremdrueckgabe(ctx, tx, l); err != nil {
		return nil, err
	}

	// Der Wächter: Ein Scan, der älter ist als die letzte Bewegung, beschreibt eine
	// Wirklichkeit, die es nicht mehr gibt.
	if stand.LetzteBewegung != nil && l.gescannt.Before(*stand.LetzteBewegung) {
		db.Verwerfe(ctx, tx)
		return d.meldeAbweisung(ctx, l, repository.NachbuchVeraltet,
			fmt.Sprintf("Scan von %s liegt vor der letzten Bewegung des Exemplars (%s)",
				l.gescannt.Format("02.01.2006 15:04"), stand.LetzteBewegung.In(schulzeit.Zone()).Format("02.01.2006 15:04")))
	}

	if err := holeAbgeschriebenesZurueck(ctx, tx, l, stand.Ausleihbar, stand.Ausgesondert); err != nil {
		return nil, err
	}

	if eintrag.Absicht == NachbuchAbsichtRueckgabe {
		return d.bucheRueckgabeNach(ctx, tx, l)
	}
	return d.bucheAusleiheNach(ctx, tx, l)
}

// loeseNachbuchEintrag prüft die Absicht, bestimmt den Scan-Zeitpunkt und löst Buch und Person
// auf, noch ohne Sperre. Ist das Buch unbekannt oder fehlt zur Ausleihe die Person, ist der
// Eintrag damit beantwortet: Die Abweisung steht als Meldung und kommt als zweiter Wert zurück.
func (d *ausleihDienst) loeseNachbuchEintrag(ctx context.Context, eintrag NachbuchEintrag) (*nachbuchLage, *NachbuchErgebnis, error) {
	if eintrag.Absicht != NachbuchAbsichtAusleihe && eintrag.Absicht != NachbuchAbsichtRueckgabe {
		return nil, nil, meldung(ErrUngueltigerZustand, "Unbekannte Absicht %q", eintrag.Absicht)
	}
	jetzt := d.heute()
	gescannt := eintrag.GescanntAm.Add(eintrag.UhrVersatz).In(schulzeit.Zone())
	if gescannt.After(jetzt) {
		gescannt = jetzt // höchstens Serverzeit — auch nach der Umrechnung datiert nichts vor
	}
	l := &nachbuchLage{eintrag: eintrag, gescannt: gescannt}

	exemplar, err := d.loeseExemplar(ctx, eintrag.Barcode)
	if err != nil {
		return nil, nil, err
	}
	if exemplar == nil {
		abweisung, err := d.meldeAbweisung(ctx, l, repository.NachbuchNichtGebucht, "Buch unbekannt: "+eintrag.Barcode)
		return nil, abweisung, err
	}
	l.exemplar = exemplar
	if err := d.loesePerson(ctx, l); err != nil {
		return nil, nil, err
	}
	if eintrag.Absicht == NachbuchAbsichtAusleihe && l.leser == nil {
		grund := "Ausweis unbekannt"
		if l.ausweis != nil {
			grund += ": " + *l.ausweis
		}
		abweisung, err := d.meldeAbweisung(ctx, l, repository.NachbuchNichtGebucht, grund)
		return nil, abweisung, err
	}
	return l, nil, nil
}

// verlegeScanAufFremdrueckgabe behandelt die Ausnahme: Der Online-Versand hat nur die
// Fremdrückgabe gebucht, die Ausleihe fehlt. Der Scan wird auf den Zeitpunkt dieser Rückgabe
// gelegt. Die Ausleihe überlappt die beendete dann nicht, und der Wächter entscheidet wie immer:
// Ist die Rückgabe noch die letzte Bewegung, kommt der Eintrag durch; hat sich das Exemplar
// danach bewegt (Rückgabe, Aussonderung), liegt der Stempel später, und er meldet „veraltet".
// Das trägt nur, weil der Stempel nie rückwärts läuft (repository.sqlStempelVor) und jede
// Bewegung stempelt (docs/invarianten.md).
// „Nicht verliehen" ist trotzdem nötig: Eine Ausleihe, die selbst so nachgeholt wurde, liegt
// auf demselben Rückgabezeitpunkt und schiebt den Stempel nicht darüber hinaus — ein zweiter
// Eintrag mit Verweis auf dieselbe Fremdrückgabe käme sonst durch und buchte sie um
// (TestBucheNach_WaechterUndFremdrueckgabe).
func verlegeScanAufFremdrueckgabe(ctx context.Context, tx pgx.Tx, l *nachbuchLage) error {
	e := l.eintrag
	if e.NachFremdrueckgabeVon != nil && e.Absicht == NachbuchAbsichtAusleihe && l.offeneAusleihe == nil {
		rueckgabe, err := repository.FindeFremdrueckgabeZeitpunkt(ctx, tx, *e.NachFremdrueckgabeVon, l.exemplar.ID)
		if err != nil {
			return err
		}
		if rueckgabe != nil && l.gescannt.Before(*rueckgabe) {
			l.gescannt = *rueckgabe
		}
	}
	return nil
}

// holeAbgeschriebenesZurueck: Ist das Exemplar abgeschrieben oder gesperrt, aber nicht verliehen,
// kommt es zurück — Umlauf und Forderung in der Transaktion des Eintrags.
func holeAbgeschriebenesZurueck(ctx context.Context, tx pgx.Tx, l *nachbuchLage, ausleihbar, ausgesondert bool) error {
	if (ausgesondert || !ausleihbar) && l.offeneAusleihe == nil {
		var err error
		if l.befund, err = repository.HoleExemplarZurueck(ctx, tx, l.exemplar.ID, l.eintrag.BearbeiterID, &l.gescannt); err != nil {
			return err
		}
		l.reaktiviert = true
		l.exemplar.IstAusleihbar, l.exemplar.IstAusgesondert, l.exemplar.ZustandNotiz = true, false, ""
	}
	return nil
}

// loeseExemplar findet das Exemplar zum gescannten Barcode — direkt oder über die
// EAN-13 eines Littera-Etiketts (dieselben Formen wie der Online-Scan).
func (d *ausleihDienst) loeseExemplar(ctx context.Context, barcode string) (*repository.Exemplar, error) {
	exemplar, err := d.katalogAbfragen.FindeExemplar(ctx, barcode)
	if err != nil || exemplar != nil {
		return exemplar, err
	}
	if nummer, istEtikett := dekodiereLitteraEtikett(barcode); istEtikett {
		return d.katalogAbfragen.FindeExemplar(ctx, nummer)
	}
	return nil, nil
}

// loesePerson bestimmt, wer ausleiht: die schon aufgelöste Person des Eintrags, sonst der ohne
// Netz gescannte Ausweis. Bleibt der Ausweis unbekannt, steht sein Text in der Meldung.
func (d *ausleihDienst) loesePerson(ctx context.Context, l *nachbuchLage) error {
	e := l.eintrag
	if e.LeserID != nil && *e.LeserID != "" {
		leser, err := d.leserAbfragen.Finde(ctx, *e.LeserID)
		if err != nil {
			return err
		}
		l.leser = leser
		return nil
	}
	if e.AusweisBarcode == nil || *e.AusweisBarcode == "" {
		return nil
	}
	leser, err := d.leserAbfragen.FindeZuBarcode(ctx, *e.AusweisBarcode)
	if err != nil {
		return err
	}
	if leser != nil {
		l.leser = leser
		return nil
	}
	l.ausweis = e.AusweisBarcode
	return nil
}

// meldeAbweisung schreibt nicht_gebucht/veraltet in eigener Transaktion — nach dem
// Rollback der Buchung, damit die Meldung steht, obwohl nichts gebucht wurde.
func (d *ausleihDienst) meldeAbweisung(ctx context.Context, l *nachbuchLage, ergebnis, grund string) (*NachbuchErgebnis, error) {
	if err := repository.SchreibeNachbuchMeldung(ctx, d.pool, d.baueMeldung(l, ergebnis, grund)); err != nil {
		return nil, err
	}
	return &NachbuchErgebnis{Ergebnis: ergebnis, Grund: grund}, nil
}

// baueMeldung baut die Meldung aus der Lage; Ausleiher ist die Person des Eintrags.
func (d *ausleihDienst) baueMeldung(l *nachbuchLage, ergebnis, grund string) repository.NachbuchMeldungEingabe {
	m := repository.NachbuchMeldungEingabe{
		IdempotenzSchluessel: l.eintrag.Schluessel, Barcode: l.eintrag.Barcode, Ergebnis: ergebnis, Grund: grund,
		AusweisText: l.ausweis, GescanntAm: l.gescannt,
	}
	if l.exemplar != nil {
		m.ExemplarID = &l.exemplar.ID
	}
	if l.leser != nil {
		m.AusleiherSchuelerID = &l.leser.ID
	}
	return m
}

// istPruefungVerletzt erkennt eine verletzte CHECK-Bedingung (23514) — beim Nachbuchen
// heißt das: Die Rückgabe läge vor der Ausleihe (check_return_date), der Scan ist veraltet.
func istPruefungVerletzt(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23514"
}

// istEigeneAusleihe sagt, ob die aktive Ausleihe der Person des Eintrags gehört.
func istEigeneAusleihe(l *nachbuchLage) bool {
	if l.offeneAusleihe == nil || l.leser == nil {
		return false
	}
	return l.offeneAusleihe.SchuelerID != nil && *l.offeneAusleihe.SchuelerID == l.leser.ID
}
