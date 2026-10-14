package service

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"bibliothek/db"
	"bibliothek/pkg/betrag"
	"bibliothek/pkg/code39"
	"bibliothek/repository"
)

// OmniboxErgebnis ist die Antwort der Omnibox auf einen Scan oder eine Suche.
type OmniboxErgebnis struct {
	// Art ist "student", "ausleihe", "rueckgabe", "search_results", "info" oder beim Gerät
	// "geraet_check". Ein gescannter Ausweis liefert immer "student", auch für einen Kollegen;
	// ob er Schüler oder Kollege ist, steht als Art am Leser.
	Art string
	// Meldung ist ein Satz für die Theke, etwa nach der Rückkehr eines abgeschriebenen Buchs.
	Meldung string
	// Leser ist der Leser, dessen Ausweis gescannt oder auf den gebucht wurde.
	Leser *repository.Leser
	// Exemplar ist das gebuchte Exemplar.
	Exemplar *repository.Exemplar
	// Geraet ist das gebuchte Gerät.
	Geraet *repository.Geraet
	// Frist ist die Rückgabefrist einer Ausleihe.
	Frist *time.Time
	// AusleiheID ist die Kennung der gebuchten Ausleihe.
	AusleiheID *string
	// Fremdrueckgabe sagt, ob jemand anderes als der Ausleiher zurückgebracht hat.
	Fremdrueckgabe bool
	// Vorbesitzer ist bei einer Fremdrückgabe der Leser, der das Buch oder Gerät ausgeliehen
	// hatte.
	Vorbesitzer *repository.Leser
	// Treffer sind die Titel einer Suche.
	Treffer []repository.Titel
	// HatVormerkung sagt, dass das zurückgegebene Buch jetzt für einen wartenden Schüler
	// bereitliegt.
	HatVormerkung bool
	// VormerkungTitel ist der Titel des vorgemerkten Buchs.
	VormerkungTitel string
	// VorgemerktFuer nennt Name und Klasse des Schülers, für den das Buch bereitliegt.
	VorgemerktFuer string
	// RegalfreigabeBarcode ist das Exemplar im Abholfach, das zurück ins Regal muss, weil der
	// Schüler ein anderes Exemplar desselben Titels genommen hat.
	RegalfreigabeBarcode string
	// AufsichtInformieren ist gesetzt, wenn das zurückgebrachte Buch auf einem Bescheid steht,
	// der schon bei der Schulaufsicht liegt; sie ist unverzüglich zu informieren. Es ist ein
	// eigenes Feld und keine Meldung, weil die Theke es als Aufgabe sehen muss.
	AufsichtInformieren string
	// Abholbereit nennt die Vormerkungen des gescannten Schülers, deren Buch im Abholfach
	// liegt. Der Hinweis sagt der Kraft an der Theke, dass sie ins Fach greifen soll, solange
	// der Schüler vor ihr steht; sonst läge das Buch dort, bis die Abholfrist
	// (repository.BestimmeAbholfrist) es still an den Nächsten weiterreicht.
	Abholbereit []AbholbereiteVormerkung
	// AuflagenHinweis ist gesetzt, wenn Kinder derselben Klasse eine andere Auflage des eben
	// ausgeliehenen Schulbuchs haben.
	AuflagenHinweis *repository.AuflagenMischung
}

// AbholbereiteVormerkung ist ein Eintrag des Hinweises auf das Abholfach: Titel und Abholfrist,
// ohne Kennungen, weil die Theke nur den Griff ins Fach braucht.
type AbholbereiteVormerkung struct {
	Titel             string
	BereitgestelltBis *time.Time
}

// OmniboxEingabe ist ein Scan oder eine Suche an der Theke mit dem, was die Theke dazu weiß:
// den geöffneten Leser, die bestätigte Checkliste und ob eine Sperre übergangen werden soll.
type OmniboxEingabe struct {
	Text string
	// AktiverLeserID ist der Leser, der gerade an der Theke steht, gleich welcher Art.
	AktiverLeserID       *string
	ChecklisteBestaetigt bool
	BearbeiterID         string
	SperreUebergehen     bool
}

// OmniboxDienst beantwortet jede Eingabe der Omnibox, des einen Eingabefelds der Theke.
type OmniboxDienst interface {
	// VerarbeiteEingabe erkennt, ob die Eingabe einen Leser, ein Exemplar oder ein Gerät meint
	// oder eine Suche ist, und bucht, was zu buchen ist.
	VerarbeiteEingabe(ctx context.Context, eingabe OmniboxEingabe) (*OmniboxErgebnis, error)
}

// omniboxDienst erfüllt OmniboxDienst; Ausleihe und Gerät bucht es über deren Dienste.
type omniboxDienst struct {
	pool            db.PgxPoolIface
	leserAbfragen   repository.LeserAbfragen
	katalogAbfragen repository.KatalogAbfragen
	ausleihAbfragen repository.AusleihAbfragen
	ausleihe        AusleihDienst
	geraete         GeraeteDienst
}

// NewOmniboxDienst baut den Dienst der Omnibox.
func NewOmniboxDienst(
	pool db.PgxPoolIface,
	leserAbfragen repository.LeserAbfragen,
	katalogAbfragen repository.KatalogAbfragen,
	ausleihAbfragen repository.AusleihAbfragen,
	ausleihe AusleihDienst,
	geraete GeraeteDienst,
) OmniboxDienst {
	return &omniboxDienst{
		pool:            pool,
		leserAbfragen:   leserAbfragen,
		katalogAbfragen: katalogAbfragen,
		ausleihAbfragen: ausleihAbfragen,
		ausleihe:        ausleihe,
		geraete:         geraete,
	}
}

// VerarbeiteEingabe beantwortet einen Scan und gibt einem Aufdruck mit Prüfzeichen eine zweite
// Chance. Ältere Karten und Etiketten tragen Code 39 mit Prüfzeichen, und ein Lesegerät gibt
// das Zeichen als Teil der Nummer zurück: Unter der Karte steht „A-10003", gescannt wird
// „A-100037". Ohne die zweite Chance sähe es an der Theke aus, als täte der Scanner nichts.
//
// Die Nachsicht greift nur, wenn der Scan so, wie er kam, nichts ergab: Bei 43 möglichen
// Zeichen sieht jeder 43. gültige Code so aus, als hinge ein Prüfzeichen dran. Erst als zweiter
// Versuch trifft sie nur dort falsch, wo der gekürzte Wert existiert und der volle nicht, und
// genau diesen Fall soll sie auflösen.
//
// Sie steht hier und nicht in den Zweigen, weil der alte Aufdruck jede Form haben kann:
// „A-100037" geht über den Ausweis, „B-100016" über das Buch, eine nackte Littera-Nummer über
// die Suche.
func (d *omniboxDienst) VerarbeiteEingabe(ctx context.Context, eingabe OmniboxEingabe) (*OmniboxErgebnis, error) {
	erg, err := d.verarbeite(ctx, eingabe)
	if !istOhneTreffer(erg, err) {
		return erg, err
	}

	kern, hatPruefzeichen := code39.OhnePruefzeichen(eingabe.Text)
	if !hatPruefzeichen {
		return erg, err
	}

	zweiterVersuch := eingabe
	zweiterVersuch.Text = kern
	erg2, err2 := d.verarbeite(ctx, zweiterVersuch)
	if istOhneTreffer(erg2, err2) {
		// Auch gekürzt nichts. Dann die erste Antwort zurückgeben: Sie nennt die Nummer,
		// die wirklich gescannt wurde. Eine Meldung über „B-10001" wäre verwirrend, wenn
		// auf dem Etikett „B-1000" steht und der Scanner „B-10001" gelesen hat.
		return erg, err
	}
	return erg2, err2
}

// istOhneTreffer erkennt den Zustand „der Scanner tut nichts" in seinen zwei
// Formen: der laute Fehler der Vorsilben-Zweige (ErrNichtGefunden) und die stille leere
// Trefferliste, in die eine unbekannte nackte Nummer fällt.
//
// Ein Fehler, der kein ErrNichtGefunden ist (Datenbank weg, Sperre, Gerät ohne Checkliste),
// gilt ausdrücklich nicht als „ohne Treffer": Ihn ein zweites Mal auszulösen hieße, eine
// Sperrmeldung doppelt zu schreiben oder eine echte Störung zu verschleiern.
func istOhneTreffer(erg *OmniboxErgebnis, err error) bool {
	if err != nil {
		return errors.Is(err, ErrNichtGefunden)
	}
	return erg != nil && erg.Art == "search_results" && len(erg.Treffer) == 0
}

// verarbeite ordnet die Eingabe an ihrer Vorsilbe einem Zweig zu. vorsilben_zwilling_test.go
// liest den Rumpf dieser Funktion und hält die Vorsilben gegen die Einordnung der Theke ohne
// Netz; wer sie umbenennt, zieht den Namen dort nach.
func (d *omniboxDienst) verarbeite(ctx context.Context, eingabe OmniboxEingabe) (*OmniboxErgebnis, error) {
	erg := &OmniboxErgebnis{}

	// A- ist ein Ausweis; S- und L- sind ältere Vorsilben von Ausweisen, die weiter gelesen
	// werden, weil Nummern nie neu vergeben werden. B- ist ein Buch, LMF- ein Lernmittel aus
	// dem Bestand von Littera, G- ein Gerät. Ohne Netz ordnet die Theke dieselben Vorsilben
	// genauso ein (scanEinordnen.js), denn ein Scan muss mit und ohne Netz dasselbe bedeuten;
	// eine unbekannte LMF-Nummer sagt deshalb „nicht gefunden", statt in die Namenssuche zu
	// laufen.
	//
	// leser: ist keine Vorsilbe eines Strichcodes, sondern die Auswahl aus der Trefferliste der
	// Namenssuche. Sie schickt die Kennung und nicht die Ausweisnummer, weil ein Kollege aus
	// der Selbstanmeldung keine Nummer hat; ein Doppelpunkt kommt in keinem Strichcode vor.
	switch {
	case strings.HasPrefix(eingabe.Text, leserIDPraefix):
		return erg, d.verarbeiteLeserID(ctx, strings.TrimPrefix(eingabe.Text, leserIDPraefix), erg)
	case strings.HasPrefix(eingabe.Text, "A-"), strings.HasPrefix(eingabe.Text, "S-"),
		strings.HasPrefix(eingabe.Text, "L-"):
		return erg, d.verarbeiteAusweis(ctx, eingabe.Text, erg)
	case strings.HasPrefix(eingabe.Text, "B-"), strings.HasPrefix(eingabe.Text, "LMF-"):
		return erg, d.verarbeiteExemplar(ctx, eingabe, erg)
	case strings.HasPrefix(eingabe.Text, "G-"):
		geraeteErg, err := d.geraete.VerarbeiteScan(ctx, eingabe.Text, eingabe.AktiverLeserID, eingabe.ChecklisteBestaetigt, eingabe.SperreUebergehen, eingabe.BearbeiterID)
		if err == nil {
			d.uebernimmGeraeteErgebnis(geraeteErg, erg)
		}
		return erg, err
	default:
		return erg, d.verarbeiteOhneVorsilbe(ctx, eingabe, erg)
	}
}

// verarbeiteOhneVorsilbe ordnet eine Eingabe ohne Vorsilbe in der Reihenfolge Buch, Ausweis,
// Suche zu. Die Vorsilben sind eine Abkürzung und keine Voraussetzung: Littera kennt sie nicht,
// und Ausweise und Etiketten aus dem Altbestand tragen nackte Nummern. Die Formen sind
// verschieden genug, dass die Reihenfolge eindeutig entscheidet: Ein Schülerausweis liefert
// etwa `B97601826457` (die Nummer des Kartenherstellers), ein Buchetikett eine EAN-13. Ein
// Ausweis wird über LeserAbfragen.FindeZuBarcode gefunden, gleich welcher Art der Leser ist.
//
// Die Suchen liefern ohne Treffer (nil, nil); ein Fehler ist deshalb einer der Datenbank und
// geht weiter (500), statt als „nicht gefunden" zu verschwinden.
func (d *omniboxDienst) verarbeiteOhneVorsilbe(ctx context.Context, eingabe OmniboxEingabe, erg *OmniboxErgebnis) error {
	exemplar, exemplarErr := d.katalogAbfragen.FindeExemplar(ctx, eingabe.Text)
	if exemplarErr != nil {
		return fmt.Errorf("datenbankfehler bei Barcode-Auflösung: %w", exemplarErr)
	}
	if exemplar != nil {
		return d.verarbeiteExemplar(ctx, eingabe, erg)
	}

	// Littera-Buchetikett: Der Strichcode liefert eine EAN-13, im System steht die
	// kurze Mediennummer (siehe dekodiereLitteraEtikett). Erst rückrechnen, dann
	// erneut nachschlagen — nur ein existierendes Exemplar löst eine Aktion aus,
	// alles andere fällt unverändert zur Ausweis-/Volltextstufe durch.
	if nummer, istEtikett := dekodiereLitteraEtikett(eingabe.Text); istEtikett {
		dekodiert, dekodiertErr := d.katalogAbfragen.FindeExemplar(ctx, nummer)
		if dekodiertErr != nil {
			return fmt.Errorf("datenbankfehler bei Etikett-Auflösung: %w", dekodiertErr)
		}
		if dekodiert != nil {
			eingabe.Text = nummer
			return d.verarbeiteExemplar(ctx, eingabe, erg)
		}
	}

	leser, leserErr := d.leserAbfragen.FindeZuBarcode(ctx, eingabe.Text)
	if leserErr != nil {
		return fmt.Errorf("datenbankfehler bei Ausweis-Auflösung: %w", leserErr)
	}
	if leser != nil {
		d.zeigeLeser(ctx, leser, erg)
		return nil
	}

	// Kein Buch, kein Ausweis: Was jemand getippt hat, ist eine Suche. Diese Stufe
	// trägt die Titelsuche der Theke und darf nie verschwinden.
	return d.verarbeiteSuche(ctx, eingabe.Text, erg)
}

// uebernimmGeraeteErgebnis überträgt das Ergebnis des Geräts in die Antwort der Omnibox.
func (d *omniboxDienst) uebernimmGeraeteErgebnis(geraeteErg *GeraeteErgebnis, erg *OmniboxErgebnis) {
	if geraeteErg == nil {
		return
	}
	erg.Art = geraeteErg.Art
	erg.Geraet = geraeteErg.Geraet
	erg.Leser = geraeteErg.Leser
	erg.Frist = geraeteErg.Frist
	erg.AusleiheID = geraeteErg.AusleiheID
	erg.Fremdrueckgabe = geraeteErg.Fremdrueckgabe
	erg.Vorbesitzer = geraeteErg.Vorbesitzer
}

// verarbeiteAusweis lädt den Leser zu einem Ausweis mit Vorsilbe. Die Vorsilbe sagt nur
// „Ausweis" und nicht die Art des Lesers: Die Übernahme aus Littera gibt auch einem Schüler
// ohne eindeutige Nummer „L-<Littera-Nummer>". Eine unbekannte Nummer bleibt ein lauter Fehler
// und verschwindet nicht in der Suche.
func (d *omniboxDienst) verarbeiteAusweis(ctx context.Context, nummer string, erg *OmniboxErgebnis) error {
	leser, err := d.leserAbfragen.FindeZuBarcode(ctx, nummer)
	if err != nil {
		return err
	}
	if leser == nil {
		return meldung(ErrNichtGefunden, "Ausweis %s ist nicht registriert", nummer)
	}
	d.zeigeLeser(ctx, leser, erg)
	return nil
}

// leserIDPraefix markiert die Auswahl aus der Trefferliste (siehe verarbeite).
const leserIDPraefix = "leser:"

// verarbeiteLeserID lädt den Leser, den jemand in der Trefferliste angeklickt hat.
//
// Dieselbe Tür wie der Ausweis-Scan — nur der Schlüssel ist ein anderer: Die
// Trefferliste kennt die ID, der Scanner die Nummer. Beide enden in zeigeLeser, damit
// der Abholfach-Hinweis nicht an einem der beiden Wege fehlt.
//
// Eine unbekannte ID ist ein lauter Fehler und fällt nicht in die Volltextsuche: Dort
// stünde „keine Treffer", und der Klick sähe aus, als sei nichts passiert.
func (d *omniboxDienst) verarbeiteLeserID(ctx context.Context, id string, erg *OmniboxErgebnis) error {
	leser, err := d.leserAbfragen.Finde(ctx, id)
	if err != nil {
		return fmt.Errorf("datenbankfehler bei der Leser-Auflösung: %w", err)
	}
	if leser == nil {
		return meldung(ErrNichtGefunden, "Dieser Leser steht nicht mehr in der Leserdatei")
	}
	d.zeigeLeser(ctx, leser, erg)
	return nil
}

// zeigeLeser legt den gefundenen Leser in die Antwort, mit dem Abholfach-Hinweis.
//
// Der Hinweis gilt Vormerkungen, und die legen nur Schüler an — für einen Kollegen ist
// die Liste leer, ohne dass es hier einer Fallunterscheidung bedarf.
func (d *omniboxDienst) zeigeLeser(ctx context.Context, leser *repository.Leser, erg *OmniboxErgebnis) {
	erg.Art = "student"
	erg.Leser = leser
	erg.Abholbereit = d.ladeAbholbereiteVormerkungen(ctx, leser.ID)
}

// ladeAbholbereiteVormerkungen lädt für den Hinweis auf das Abholfach die Bücher, die für den
// Leser bereitliegen. Ein Fehler bricht den Scan nicht ab und steht im Log: Die Theke soll
// wegen eines Hinweises nicht stehen.
func (d *omniboxDienst) ladeAbholbereiteVormerkungen(ctx context.Context, schuelerID string) []AbholbereiteVormerkung {
	// Die Tests der Weiche bauen den Dienst ohne Pool; dort gibt es nichts zu laden.
	if d.pool == nil {
		return nil
	}
	buecher, err := repository.LadeAbholbereiteTitel(ctx, d.pool, schuelerID)
	if err != nil {
		log.Printf("omnibox: Abholfach-Hinweis nicht ladbar für Schüler %s: %v", schuelerID, err)
		return nil
	}
	var liste []AbholbereiteVormerkung
	for _, b := range buecher {
		liste = append(liste, AbholbereiteVormerkung(b))
	}
	return liste
}

// verarbeiteSuche sucht in Titel, Autor, ISBN und Systematik.
func (d *omniboxDienst) verarbeiteSuche(ctx context.Context, text string, erg *OmniboxErgebnis) error {
	titel, err := d.katalogAbfragen.FindeTitel(ctx, text)
	if err != nil {
		return err
	}
	erg.Art = "search_results"
	erg.Treffer = titel
	return nil
}

// versucheReaktivierung behandelt ein gesperrtes oder ausgesondertes Exemplar: Liegt keine
// offene Ausleihe vor, holt der Scan es zurück (Umlauf und Forderung in einer Transaktion) und
// endet mit „info"; ausgeliehen wird erst mit dem nächsten Scan. fertig heißt, dass das
// Ergebnis steht.
func (d *omniboxDienst) versucheReaktivierung(ctx context.Context, nummer string, exemplar *repository.Exemplar, bearbeiterID string, erg *OmniboxErgebnis) (fertig bool, err error) {
	offeneAusleihe, err := d.ausleihAbfragen.FindeOffene(ctx, exemplar.ID)
	if err != nil {
		return false, err
	}

	if offeneAusleihe == nil {
		befund, err := d.holeExemplarZurueck(ctx, exemplar.ID, bearbeiterID)
		if err != nil {
			return false, err
		}
		exemplar.IstAusleihbar = true
		exemplar.IstAusgesondert = false
		exemplar.ZustandNotiz = ""

		erg.Meldung = rueckkehrMeldung(befund)
		erg.AufsichtInformieren = aufsichtHinweis(befund)
		erg.Art = "info"
		return true, nil
	}

	if exemplar.IstAusgesondert {
		return false, meldung(ErrUngueltigerZustand, "Buchexemplar %s ist ausgesondert und kann nicht ausgeliehen werden", nummer)
	}
	return false, meldung(ErrUngueltigerZustand, "Buchexemplar ist nicht ausleihbar")
}

// holeExemplarZurueck ist der Scan am Netz über dem Baustein repository.HoleExemplarZurueck, in
// eigener Transaktion, weil hier nichts weiter folgt. Das Nachbuchen legt den Baustein in die
// Transaktion seiner Buchung.
func (d *omniboxDienst) holeExemplarZurueck(ctx context.Context, exemplarID, bearbeiterID string) (repository.RueckkehrBefund, error) {
	tx, err := d.pool.Begin(ctx)
	if err != nil {
		return repository.RueckkehrBefund{}, err
	}
	defer db.SafeRollback(ctx, tx)

	befund, err := repository.HoleExemplarZurueck(ctx, tx, exemplarID, bearbeiterID, nil)
	if err != nil {
		return repository.RueckkehrBefund{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return repository.RueckkehrBefund{}, err
	}
	return befund, nil
}

// rueckkehrMeldung ist der Satz für die Theke. Er nennt die Folge, nicht den Vorgang:
// Die Mitarbeiterin muss wissen, ob das Kind jetzt frei ist — und im Fall der schon
// übergebenen Forderung, dass sie etwas tun muss, das die Anwendung nicht kann.
func rueckkehrMeldung(b repository.RueckkehrBefund) string {
	if b.StornierteForderungen > 0 {
		return fmt.Sprintf("Buch reaktiviert. Die Forderung über %s wurde storniert — das Buch ist zurück.", betrag.Euro(b.StornierterBetrag))
	}
	return "Buch reaktiviert"
}

// aufsichtHinweis ist der Satz für den Fall, den die Anwendung nicht erledigen kann: Der
// Bescheid liegt bei der Schulaufsicht, die Forderung bleibt offen, und jemand muss zum
// Telefon greifen. Leer, wenn nichts zu tun ist.
func aufsichtHinweis(b repository.RueckkehrBefund) string {
	return b.AufsichtHinweis()
}

// verarbeiteExemplar bucht den Scan eines Buchs: ohne geöffneten Leser die Rückgabe, mit ihm
// die Buchung am Leser.
func (d *omniboxDienst) verarbeiteExemplar(ctx context.Context, eingabe OmniboxEingabe, erg *OmniboxErgebnis) error {
	exemplar, err := d.katalogAbfragen.FindeExemplar(ctx, eingabe.Text)
	if err != nil {
		return err
	}
	if exemplar == nil {
		return meldung(ErrNichtGefunden, "Buchexemplar-Barcode %s wurde nicht gefunden", eingabe.Text)
	}

	// Ein gesperrtes oder ausgesondertes Exemplar holt der Scan zurück.
	if !exemplar.IstAusleihbar || exemplar.IstAusgesondert {
		fertig, err := d.versucheReaktivierung(ctx, eingabe.Text, exemplar, eingabe.BearbeiterID, erg)
		if err != nil {
			return err
		}
		if fertig {
			return nil
		}
	}

	if eingabe.AktiverLeserID != nil && *eingabe.AktiverLeserID != "" {
		ausleihErg, err := d.ausleihe.BucheAmLeser(ctx, exemplar, eingabe.AktiverLeserID, eingabe.BearbeiterID, eingabe.SperreUebergehen)
		if err != nil {
			return err
		}
		d.uebernimmAusleihErgebnis(ausleihErg, erg)
		return nil
	}

	ausleihErg, err := d.ausleihe.NimmZurueck(ctx, exemplar, eingabe.BearbeiterID)
	if err != nil {
		return err
	}
	d.uebernimmAusleihErgebnis(ausleihErg, erg)
	return nil
}

// uebernimmAusleihErgebnis überträgt das Ergebnis der Ausleihe in die Antwort der Omnibox.
func (d *omniboxDienst) uebernimmAusleihErgebnis(ausleihErg *AusleihErgebnis, erg *OmniboxErgebnis) {
	if ausleihErg == nil {
		return
	}
	erg.Art = ausleihErg.Art
	erg.Exemplar = ausleihErg.Exemplar
	erg.Leser = ausleihErg.Leser
	erg.Frist = ausleihErg.Frist
	erg.AusleiheID = ausleihErg.AusleiheID
	erg.Fremdrueckgabe = ausleihErg.Fremdrueckgabe
	erg.Vorbesitzer = ausleihErg.Vorbesitzer
	erg.HatVormerkung = ausleihErg.HatVormerkung
	erg.VormerkungTitel = ausleihErg.VormerkungTitel
	erg.VorgemerktFuer = ausleihErg.VorgemerktFuer
	erg.RegalfreigabeBarcode = ausleihErg.RegalfreigabeBarcode
	erg.AuflagenHinweis = ausleihErg.AuflagenHinweis
}

// OmniboxErgebnisAus überträgt ein Ergebnis der Ausleihe in die Form der Omnibox, mit derselben
// Abbildung wie der Scan am Netz (uebernimmAusleihErgebnis), damit die Antwort des Nachbuchens
// dieselbe Gestalt hat.
func OmniboxErgebnisAus(ausleihErg *AusleihErgebnis) *OmniboxErgebnis {
	erg := &OmniboxErgebnis{}
	(&omniboxDienst{}).uebernimmAusleihErgebnis(ausleihErg, erg)
	return erg
}
