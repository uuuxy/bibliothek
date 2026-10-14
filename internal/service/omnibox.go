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

// OmniboxErgebnis beschreibt die Antwortstruktur der Omnibox nach Verarbeitung einer Eingabe (Scan oder Suche).
type OmniboxErgebnis struct {
	// Art definiert die Art der Antwort (z. B. "student", "ausleihe", "rueckgabe", "search_results", "info").
	//
	// "teacher" gibt es seit Migration 125 nicht mehr: Ein gescannter Ausweis liefert
	// immer einen LESER, und ob er Schüler oder Kollege ist, steht als Art an ihm.
	Art string
	// Meldung enthält eine optionale Benachrichtigung für das Frontend.
	Meldung string
	// Leser enthält den LESER, dessen Ausweis gescannt oder für den eine Aktion
	// durchgeführt wurde.
	Leser *repository.Leser
	// Exemplar enthält die Daten des betroffenen Buchs (Ausleihe/Rückgabe).
	Exemplar *repository.Exemplar
	// Geraet enthält die Daten des betroffenen Geräts (Hardware-Ausleihe/Rückgabe).
	Geraet *repository.Geraet
	// Frist gibt das Rückgabedatum der aktuellen Ausleihe an.
	Frist *time.Time
	// AusleiheID ist die ID des verknüpften Ausleihvorgangs.
	AusleiheID *string
	// Fremdrueckgabe zeigt an, ob die Rückgabe durch eine andere Person erfolgt ist.
	Fremdrueckgabe bool
	// Vorbesitzer enthält den LESER, der das Buch/Gerät zuvor ausgeliehen hatte (bei Fremdrückgabe).
	Vorbesitzer *repository.Leser
	// Treffer enthält Suchergebnisse bei einer allgemeinen Buchtitel-Suche.
	Treffer []repository.Titel
	// HatVormerkung zeigt an, ob für das zurückgegebene Buch eine Reservierung aktiv wurde.
	HatVormerkung bool
	// VormerkungTitel ist der Titel des reservierten Buchs.
	VormerkungTitel string
	// VorgemerktFuer ist der Name des Schülers, der die Reservierung ausgelöst hat.
	VorgemerktFuer string
	// RegalfreigabeBarcode: reserviertes Exemplar, das zurück ins Regal muss (der
	// Schüler hat ein anderes Exemplar desselben Titels genommen).
	RegalfreigabeBarcode string
	// AufsichtInformieren: Das zurückgebrachte Buch steht auf einem Bescheid, der schon
	// bei der Schulaufsicht liegt — sie ist unverzüglich zu informieren (#597). Eigenes
	// Feld und nicht Message: Das ist keine Erfolgsmeldung, sondern eine Aufgabe, und die
	// Theke muss sie als solche sehen.
	AufsichtInformieren string
	// Abholbereit: Vormerkungen des GESCANNTEN Schülers, deren Buch im Abholfach
	// liegt (Betreiber-Entscheidung 01.09.2026). Schüler scannen nicht selbst —
	// der Hinweis sagt der Mitarbeiterin am Terminal, dass sie ins Abholfach
	// greifen soll, solange der Schüler vor ihr steht. Ohne ihn läge das Buch im
	// Fach, bis die Abholfrist (repository.BestimmeAbholfrist) es still an den Nächsten weiterreicht.
	Abholbereit []AbholbereiteVormerkung
	// AuflagenHinweis: Das eben ausgeliehene Schulbuch ist eine andere Auflage als die, die
	// Kinder derselben Klasse schon haben (docs/OFFEN.md 4.18, Stufe 5). Aus AusleihErgebnis.
	AuflagenHinweis *repository.AuflagenMischung
}

// AbholbereiteVormerkung ist ein Eintrag des Abholfach-Hinweises: Titel und
// Abholfrist — bewusst ohne IDs, die Theke braucht nur den Griff ins Fach.
type AbholbereiteVormerkung struct {
	Titel             string
	BereitgestelltBis *time.Time
}

// OmniboxEingabe bündelt die Eingabe und den Sitzungskontext eines Omnibox-Requests,
// damit VerarbeiteEingabe & Co. nicht acht Einzelargumente durchreichen.
type OmniboxEingabe struct {
	Text string
	// AktiverLeserID ist die Person, die gerade an der Theke steht — ein Leser, gleich
	// welcher Art. Bis Migration 125 standen hier zwei Felder, und jede Stelle, die sie
	// las, musste sich für eines entscheiden.
	AktiverLeserID       *string
	ChecklisteBestaetigt bool
	BearbeiterID         string
	SperreUebergehen     bool
}

// OmniboxDienst verarbeitet alle Eingaben aus der zentralen Suche/Scan-Leiste (Omnibox).
type OmniboxDienst interface {
	// VerarbeiteEingabe wertet eine Eingabe (Eingabestring) aus und steuert die passende Domänen-Aktion an.
	VerarbeiteEingabe(ctx context.Context, q OmniboxEingabe) (*OmniboxErgebnis, error)
}

// omniboxDienst ist die Standard-Implementierung des OmniboxDienst.
type omniboxDienst struct {
	pool            db.PgxPoolIface
	leserAbfragen   repository.LeserAbfragen
	katalogAbfragen repository.KatalogAbfragen
	ausleihAbfragen repository.AusleihAbfragen
	ausleihe        AusleihDienst
	geraete         GeraeteDienst
}

// NewOmniboxDienst erzeugt eine neue Instanz des standardmäßigen OmniboxDienst.
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

// VerarbeiteEingabe leitet gescannte Barcodes oder Suchanfragen anhand von Präfixen an die jeweilige Fachlogik weiter.
// VerarbeiteEingabe beantwortet einen Scan — und gibt einem Aufdruck von FRÜHER eine zweite
// Chance.
//
// Bis zum 17.09.2026 druckte die Anwendung Code 39 MIT Prüfzeichen. Das Zeichen steht in
// den Strichcode-Daten, und ein Lesegerät gibt es als Teil der Nummer zurück: Unter der
// Karte steht „A-10003", gescannt wird „A-100037". Der Server suchte dann eine Nummer,
// die es nicht gibt, und weil ein unbekannter Scan nur eine leere Trefferliste erzeugt,
// sah es an der Theke aus, als täte der Scanner gar nichts. Gedruckt wird seither Code
// 128 ohne Prüfzeichen — aber die Karten und Etiketten von vorher sind im Umlauf und
// sollen weiter funktionieren (Anforderung von Anfang an, OFFEN.md 9.2).
//
// Die Nachsicht greift NUR, wenn der Scan so, wie er kam, nichts ergeben hat. Das ist
// wichtig: Bei 43 möglichen Zeichen sieht im Schnitt jeder 43. gültige Code zufällig so
// aus, als hinge ein Prüfzeichen dran. Erst als zweiter Versuch ist ein falscher Treffer
// nur dort möglich, wo der gekürzte Wert existiert und der volle nicht — und genau das
// ist der Fall, den sie auflösen soll.
//
// Hier und nicht in den einzelnen Zweigen, weil der Aufdruck von früher jede Form haben
// kann: „A-100037" geht über den Ausweis-Zweig, „B-100016" über den Buch-Zweig, eine
// nackte Littera-Nummer über die Volltextsuche. Drei Stellen wären drei Gelegenheiten,
// eine zu vergessen.
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
		// Auch gekürzt nichts. Dann die ERSTE Antwort zurückgeben: Sie nennt die Nummer,
		// die wirklich gescannt wurde. Eine Meldung über „B-10001" wäre verwirrend, wenn
		// auf dem Etikett „B-1000" steht und der Scanner „B-10001" gelesen hat.
		return erg, err
	}
	return erg2, err2
}

// istOhneTreffer erkennt den Zustand „der Scanner tut nichts" in seinen ZWEI
// Formen: der laute Fehler der Vorsilben-Zweige (ErrNichtGefunden) und die stille leere
// Trefferliste, in die eine unbekannte nackte Nummer fällt.
//
// Ein Fehler, der KEIN ErrNichtGefunden ist (Datenbank weg, Sperre, Gerät ohne Checkliste),
// gilt ausdrücklich nicht als „ohne Treffer": Ihn ein zweites Mal auszulösen hieße, eine
// Sperrmeldung doppelt zu schreiben oder eine echte Störung zu verschleiern.
func istOhneTreffer(erg *OmniboxErgebnis, err error) bool {
	if err != nil {
		return errors.Is(err, ErrNichtGefunden)
	}
	return erg != nil && erg.Art == "search_results" && len(erg.Treffer) == 0
}

// verarbeite ist der Scan-Weg selbst — unverändert der Schalter, der er immer war.
//
// ACHTUNG beim Umbenennen: vorsilben_zwilling_test.go liest den Rumpf DIESER Funktion,
// um die Vorsilben gegen die Offline-Einordnung zu halten. Der Test hat einen
// Sanity-Floor und wird laut, wenn er ins Leere greift — aber die Meldung liest sich
// dann nach „Vorsilbe fehlt" und nicht nach „Funktion umbenannt".
func (d *omniboxDienst) verarbeite(ctx context.Context, eingabe OmniboxEingabe) (*OmniboxErgebnis, error) {
	erg := &OmniboxErgebnis{}

	// Präfix-Erkennung (Scanner-Steuerung):
	// A- steht für einen Ausweis. S- und L- sind die Vorsilben von FRÜHER (Handanlage
	// bzw. Littera-Personenlauf); vergeben wird seit dem 16.09.2026 nur noch A-, gelesen
	// werden alle drei — es gibt Nummern aus der Zeit davor, und Nummern werden nie
	// recycelt (verarbeiteAusweis)
	// B- steht für Buch (Book), LMF- ebenso (Lernmittel aus dem Littera-Bestand)
	// G- steht für Gerät (Hardware-Geräte)
	//
	// LMF- stand bis zum 17.09.2026 nicht in diesem Switch. Es funktionierte trotzdem,
	// weil verarbeiteOhneVorsilbe zuerst als Buch nachschlägt — aber auf einem anderen Weg
	// als offline, wo `LMF-` seit jeher eine Buch-Vorsilbe ist (scanEinordnen.js). Zwei
	// Wege zur selben Antwort sind einer zu viel: Ein Scan muss mit und ohne Netz dasselbe
	// bedeuten. Seither ist es derselbe Weg — und eine unbekannte LMF-Nummer sagt „nicht
	// gefunden", statt still in die Namenssuche zu laufen.
	//
	// leser: ist KEIN Scanner-Präfix, sondern die Auswahl aus der Trefferliste der
	// Namenssuche. Sie schickt die ID und nicht die Ausweisnummer, weil ein Kollege aus
	// der Selbstanmeldung gar keine hat — die Auswahl hätte sonst eine leere Eingabe
	// losgeschickt und nichts getan. Ein Doppelpunkt kommt aus keinem Strichcode.
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

// verarbeiteOhneVorsilbe löst einen Barcode/eine Query ohne bekanntes Präfix auf.
// Auflösungsreihenfolge: Buch → Ausweis → Volltextsuche.
//
// Die Präfixe A-/B-/G- (und S-/L- von früher) sind eine Abkürzung, keine Voraussetzung:
// Littera kennt sie nicht, und die Ausweise aus dem Altbestand tragen nackte Nummern. Ein
// Schülerausweis liefert gemessen `B97601826457` (Nummer des Kartenherstellers), ein
// Buchetikett eine 13-stellige EAN-13 — die Formen sind verschieden genug, dass die
// Reihenfolge hier eindeutig entscheidet.
//
// Bis Migration 125 standen Lehrkräfte in `benutzer` und Schüler in `schueler`, und die
// Reihenfolge brauchte eine eigene Lehrer-Stufe; ohne sie lief ein gescannter
// Lehrerausweis bis in die Volltextsuche. Seitdem stehen alle Leser in `leser`, und eine
// Stufe (LeserAbfragen.FindeZuBarcode) findet den Ausweis, gleich welcher Art. In Littera gibt es den
// Unterschied nicht; die Karte ist dieselbe, nur der Aufdruck lautet „Lehrerausweis".
//
// Die Lookups liefern bei Nichttreffer (nil, nil); ein non-nil Fehler ist daher ein
// echter DB-Fehler und wird propagiert (→ HTTP 500), statt ihn als "nicht gefunden" zu
// verschlucken.
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

// uebernimmGeraeteErgebnis mappt die Felder aus GeraeteErgebnis in die flache OmniboxErgebnis-Struktur.
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

// verarbeiteAusweis lädt den Leser zu einem Ausweis mit Vorsilbe „S-" oder „L-".
//
// Bis zum 15.09.2026 entschied die Vorsilbe die Tabelle: „S-" nur Schüler, „L-" nur Lehrkräfte.
// Littera kennt die Vorsilben nicht, auf dem Ausweis stehen sie nicht, und die Littera-Übernahme
// gibt einem Schüler ohne eindeutige Nummer „L-<Littera-Nummer>" (internal/littera, ausweis) —
// der war an der Theke weder über den Ausweis noch über die Namenssuche ladbar (OFFEN.md 5.15).
// Die Vorsilbe sagt jetzt nur noch „Ausweis": Eine unbekannte Nummer bleibt ein lauter Fehler
// und verschwindet nicht in der Volltextsuche.
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

// leserIDPraefix markiert die Auswahl aus der Trefferliste (siehe VerarbeiteEingabe).
const leserIDPraefix = "leser:"

// verarbeiteLeserID lädt den Leser, den jemand in der Trefferliste angeklickt hat.
//
// Dieselbe Tür wie der Ausweis-Scan — nur der Schlüssel ist ein anderer: Die
// Trefferliste kennt die ID, der Scanner die Nummer. Beide enden in zeigeLeser, damit
// der Abholfach-Hinweis nicht an einem der beiden Wege fehlt.
//
// Eine unbekannte ID ist ein lauter Fehler und fällt NICHT in die Volltextsuche: Dort
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

// ladeAbholbereiteVormerkungen holt für den Abholfach-Hinweis die Bücher, die für den Leser
// bereitliegen. Ein Fehler bricht den Scan nicht ab, er steht im Log: Die Theke soll wegen
// eines Hinweises nicht stehen (dieselbe Abwägung wie resolveFotoURL im Profil).
func (d *omniboxDienst) ladeAbholbereiteVormerkungen(ctx context.Context, schuelerID string) []AbholbereiteVormerkung {
	// Die Unit-Tests der Scan-Weiche bauen den Service ohne Pool (Stub-Repos);
	// dort gibt es keine Vormerkungen und nichts zu laden.
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

// verarbeiteSuche führt eine Volltextsuche über Buchtitel, Autoren, ISBN und Systematik aus.
func (d *omniboxDienst) verarbeiteSuche(ctx context.Context, text string, erg *OmniboxErgebnis) error {
	titel, err := d.katalogAbfragen.FindeTitel(ctx, text)
	if err != nil {
		return err
	}
	erg.Art = "search_results"
	erg.Treffer = titel
	return nil
}

// versucheReaktivierung behandelt gesperrte/ausgesonderte Exemplare: Liegt keine aktive
// Ausleihe vor, holt der Scan das Exemplar zurück (Umlauf und Forderung in einer
// Transaktion) und endet mit „info" — die Ausleihe folgt erst mit dem nächsten Scan.
// fertig=true bedeutet, dass resp bereits final gesetzt wurde; fertig=false ohne Fehler
// heißt: weiter zur Ausleihe (heute kein Weg dorthin, siehe unten).
//
// Bis zum 15.09.2026 kannte dieser Zweig eine Reservierung über die Zustandsnotiz
// „Reserviert für:" und hätte für das vormerkende Kind gleich die Ausleihe angeschlossen.
// Den Schreiber der Notiz hat daf6b370 am 16.06.2026 entfernt (die Reservierung läuft
// seitdem über bereitgestellt_exemplar_id); der Leser blieb, und in Prod trug kein
// Exemplar die Notiz (Zählung 15.09.2026). Hinter der toten Tür lag ein stiller NULL-Scan
// (vormerkungen.notiz in einen string), der dem berechtigten Kind 403 gemeldet hätte.
// Beides ist mit dem Zweig gefallen (OFFEN.md 5.14).
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

// holeExemplarZurueck ist der Online-Scan über dem Baustein repository.HoleExemplarZurueck:
// eigene Transaktion, weil hier nichts weiter folgt. Das Nachbuchen (Stufe 2) legt den
// Baustein in seine eigene Buchungstransaktion.
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

// aufsichtHinweis ist der Satz für den Fall, den die Anwendung NICHT erledigen kann: Der
// Bescheid liegt bei der Schulaufsicht, die Forderung bleibt offen, und jemand muss zum
// Telefon greifen. Leer, wenn nichts zu tun ist.
func aufsichtHinweis(b repository.RueckkehrBefund) string {
	return b.AufsichtHinweis()
}

// verarbeiteExemplar verarbeitet das Scannen eines Buch-Barcodes.
// Wenn kein aktiver Ausleiher vorhanden ist, wird das Buch zurückgegeben.
// Ist ein Schüler oder Lehrer aktiv, wird das Buch an diesen ausgeliehen.
func (d *omniboxDienst) verarbeiteExemplar(ctx context.Context, eingabe OmniboxEingabe, erg *OmniboxErgebnis) error {
	exemplar, err := d.katalogAbfragen.FindeExemplar(ctx, eingabe.Text)
	if err != nil {
		return err
	}
	if exemplar == nil {
		return meldung(ErrNichtGefunden, "Buchexemplar-Barcode %s wurde nicht gefunden", eingabe.Text)
	}

	// Gesperrte/ausgesonderte Exemplare ggf. automatisch reaktivieren.
	if !exemplar.IstAusleihbar || exemplar.IstAusgesondert {
		fertig, err := d.versucheReaktivierung(ctx, eingabe.Text, exemplar, eingabe.BearbeiterID, erg)
		if err != nil {
			return err
		}
		if fertig {
			return nil
		}
	}

	// Ausleihe durchführen, falls ein aktiver Ausleiher vorhanden ist
	if eingabe.AktiverLeserID != nil && *eingabe.AktiverLeserID != "" {
		ausleihErg, err := d.ausleihe.BucheAmLeser(ctx, exemplar, eingabe.AktiverLeserID, eingabe.BearbeiterID, eingabe.SperreUebergehen)
		if err != nil {
			return err
		}
		d.uebernimmAusleihErgebnis(ausleihErg, erg)
		return nil
	}

	// Rückgabe durchführen, wenn kein aktiver Ausleiher vorhanden ist
	ausleihErg, err := d.ausleihe.NimmZurueck(ctx, exemplar, eingabe.BearbeiterID)
	if err != nil {
		return err
	}
	d.uebernimmAusleihErgebnis(ausleihErg, erg)
	return nil
}

// uebernimmAusleihErgebnis mappt die Felder aus AusleihErgebnis in die OmniboxErgebnis-Struktur.
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

// OmniboxErgebnisAus überträgt ein Ausleih-Ergebnis in die Omnibox-Form — dieselbe
// Abbildung, die der Online-Scan nutzt (uebernimmAusleihErgebnis), damit die Nachbuch-Antwort für
// die Theke dieselbe Gestalt hat.
func OmniboxErgebnisAus(ausleihErg *AusleihErgebnis) *OmniboxErgebnis {
	erg := &OmniboxErgebnis{}
	(&omniboxDienst{}).uebernimmAusleihErgebnis(ausleihErg, erg)
	return erg
}
