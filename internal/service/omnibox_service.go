package service

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"bibliothek/db"
	"bibliothek/pkg/code39"
	"bibliothek/repository"
)

// OmniboxResult beschreibt die Antwortstruktur der Omnibox nach Verarbeitung einer Eingabe (Scan oder Suche).
type OmniboxResult struct {
	// Type definiert die Art der Antwort (z. B. "student", "ausleihe", "rueckgabe", "search_results", "info").
	//
	// "teacher" gibt es seit Migration 125 nicht mehr: Ein gescannter Ausweis liefert
	// immer einen LESER, und ob er Schüler oder Kollege ist, steht als Art an ihm.
	Type string
	// Message enthält eine optionale Benachrichtigung für das Frontend.
	Message string
	// Student enthält den LESER, dessen Ausweis gescannt oder für den eine Aktion
	// durchgeführt wurde.
	Student *repository.Student
	// Book enthält die Daten des betroffenen Buchs (Ausleihe/Rückgabe).
	Book *repository.BookCopy
	// Geraet enthält die Daten des betroffenen Geräts (Hardware-Ausleihe/Rückgabe).
	Geraet *repository.Geraet
	// DueDate gibt das Rückgabedatum der aktuellen Ausleihe an.
	DueDate *time.Time
	// LoanID ist die ID des verknüpften Ausleihvorgangs.
	LoanID *string
	// Fremdrueckgabe zeigt an, ob die Rückgabe durch eine andere Person erfolgt ist.
	Fremdrueckgabe bool
	// Vorbesitzer enthält den LESER, der das Buch/Gerät zuvor ausgeliehen hatte (bei Fremdrückgabe).
	Vorbesitzer *repository.Student
	// SearchResults enthält Suchergebnisse bei einer allgemeinen Buchtitel-Suche.
	SearchResults []repository.BookTitle
	// HasVormerkung zeigt an, ob für das zurückgegebene Buch eine Reservierung aktiv wurde.
	HasVormerkung bool
	// VormerkungTitel ist der Titel des reservierten Buchs.
	VormerkungTitel string
	// VormerkungUser ist der Name des Schülers, der die Reservierung ausgelöst hat.
	VormerkungUser string
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
	// Fach, bis die 3-Tage-Frist es still an den Nächsten weiterreicht.
	Abholbereit []AbholbereiteVormerkung
}

// AbholbereiteVormerkung ist ein Eintrag des Abholfach-Hinweises: Titel und
// Abholfrist — bewusst ohne IDs, die Theke braucht nur den Griff ins Fach.
type AbholbereiteVormerkung struct {
	Titel             string
	BereitgestelltBis *time.Time
}

// OmniboxQuery bündelt die Eingabe und den Sitzungskontext eines Omnibox-Requests,
// damit ProcessQuery & Co. nicht acht Einzelargumente durchreichen.
type OmniboxQuery struct {
	Query string
	// ActiveLeserID ist die Person, die gerade an der Theke steht — ein Leser, gleich
	// welcher Art. Bis Migration 125 standen hier zwei Felder, und jede Stelle, die sie
	// las, musste sich für eines entscheiden.
	ActiveLeserID      *string
	ConfirmedChecklist bool
	StaffID            string
	StaffRole          string
	OverrideBlock      bool
}

// OmniboxService verarbeitet alle Eingaben aus der zentralen Suche/Scan-Leiste (Omnibox).
type OmniboxService interface {
	// ProcessQuery wertet eine Eingabe (Eingabestring) aus und steuert die passende Domänen-Aktion an.
	ProcessQuery(ctx context.Context, q OmniboxQuery) (*OmniboxResult, error)
}

// defaultOmniboxService ist die Standard-Implementierung des OmniboxService.
type defaultOmniboxService struct {
	pool        db.PgxPoolIface
	studentRepo repository.StudentRepository
	bookRepo    repository.BookRepository
	userRepo    repository.UserRepository
	loanRepo    repository.LoanRepository
	loanSvc     LoanService
	deviceSvc   DeviceService
}

// NewOmniboxService erzeugt eine neue Instanz des standardmäßigen OmniboxService.
func NewOmniboxService(
	pool db.PgxPoolIface,
	studentRepo repository.StudentRepository,
	bookRepo repository.BookRepository,
	userRepo repository.UserRepository,
	loanRepo repository.LoanRepository,
	loanSvc LoanService,
	deviceSvc DeviceService,
) OmniboxService {
	return &defaultOmniboxService{
		pool:        pool,
		studentRepo: studentRepo,
		bookRepo:    bookRepo,
		userRepo:    userRepo,
		loanRepo:    loanRepo,
		loanSvc:     loanSvc,
		deviceSvc:   deviceSvc,
	}
}

// ProcessQuery leitet gescannte Barcodes oder Suchanfragen anhand von Präfixen an die jeweilige Fachlogik weiter.
// ProcessQuery beantwortet einen Scan — und gibt einem Aufdruck von FRÜHER eine zweite
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
func (s *defaultOmniboxService) ProcessQuery(ctx context.Context, q OmniboxQuery) (*OmniboxResult, error) {
	resp, err := s.verarbeite(ctx, q)
	if !scanBliebOhneTreffer(resp, err) {
		return resp, err
	}

	kern, hatPruefzeichen := code39.OhnePruefzeichen(q.Query)
	if !hatPruefzeichen {
		return resp, err
	}

	zweiterVersuch := q
	zweiterVersuch.Query = kern
	resp2, err2 := s.verarbeite(ctx, zweiterVersuch)
	if scanBliebOhneTreffer(resp2, err2) {
		// Auch gekürzt nichts. Dann die ERSTE Antwort zurückgeben: Sie nennt die Nummer,
		// die wirklich gescannt wurde. Eine Meldung über „B-10001" wäre verwirrend, wenn
		// auf dem Etikett „B-1000" steht und der Scanner „B-10001" gelesen hat.
		return resp, err
	}
	return resp2, err2
}

// scanBliebOhneTreffer erkennt den Zustand „der Scanner tut nichts" in seinen ZWEI
// Formen: der laute Fehler der Vorsilben-Zweige (ErrNotFound) und die stille leere
// Trefferliste, in die eine unbekannte nackte Nummer fällt.
//
// Ein Fehler, der KEIN ErrNotFound ist (Datenbank weg, Sperre, Gerät ohne Checkliste),
// gilt ausdrücklich nicht als „ohne Treffer": Ihn ein zweites Mal auszulösen hieße, eine
// Sperrmeldung doppelt zu schreiben oder eine echte Störung zu verschleiern.
func scanBliebOhneTreffer(resp *OmniboxResult, err error) bool {
	if err != nil {
		return errors.Is(err, ErrNotFound)
	}
	return resp != nil && resp.Type == "search_results" && len(resp.SearchResults) == 0
}

// verarbeite ist der Scan-Weg selbst — unverändert der Schalter, der er immer war.
//
// ACHTUNG beim Umbenennen: vorsilben_zwilling_test.go liest den Rumpf DIESER Funktion,
// um die Vorsilben gegen die Offline-Einordnung zu halten. Der Test hat einen
// Sanity-Floor und wird laut, wenn er ins Leere greift — aber die Meldung liest sich
// dann nach „Vorsilbe fehlt" und nicht nach „Funktion umbenannt".
func (s *defaultOmniboxService) verarbeite(ctx context.Context, q OmniboxQuery) (*OmniboxResult, error) {
	resp := &OmniboxResult{}

	// Präfix-Erkennung (Scanner-Steuerung):
	// A- steht für einen Ausweis. S- und L- sind die Vorsilben von FRÜHER (Handanlage
	// bzw. Littera-Personenlauf); vergeben wird seit dem 16.09.2026 nur noch A-, gelesen
	// werden alle drei — es gibt Nummern aus der Zeit davor, und Nummern werden nie
	// recycelt (handleAusweisAction)
	// B- steht für Buch (Book), LMF- ebenso (Lernmittel aus dem Littera-Bestand)
	// G- steht für Gerät (Hardware-Geräte)
	//
	// LMF- stand bis zum 17.09.2026 nicht in diesem Switch. Es funktionierte trotzdem,
	// weil resolveOhnePraefix zuerst als Buch nachschlägt — aber auf einem anderen Weg
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
	case strings.HasPrefix(q.Query, leserIDPraefix):
		return resp, s.handleLeserIDAction(ctx, strings.TrimPrefix(q.Query, leserIDPraefix), resp)
	case strings.HasPrefix(q.Query, "A-"), strings.HasPrefix(q.Query, "S-"),
		strings.HasPrefix(q.Query, "L-"):
		return resp, s.handleAusweisAction(ctx, q.Query, resp)
	case strings.HasPrefix(q.Query, "B-"), strings.HasPrefix(q.Query, "LMF-"):
		return resp, s.handleBookAction(ctx, q, resp)
	case strings.HasPrefix(q.Query, "G-"):
		dr, err := s.deviceSvc.HandleDeviceAction(ctx, q.Query, q.ActiveLeserID, q.ConfirmedChecklist, q.StaffID)
		if err == nil {
			s.mapDeviceResult(dr, resp)
		}
		return resp, err
	default:
		return resp, s.resolveOhnePraefix(ctx, q, resp)
	}
}

// resolveOhnePraefix löst einen Barcode/eine Query ohne bekanntes Präfix auf.
// Auflösungsreihenfolge: Buch → Schülerausweis → Lehrerausweis → Volltextsuche.
//
// Die Präfixe S-/L-/B-/G- sind eine Abkürzung, keine Voraussetzung: Littera kennt sie
// nicht, und die Ausweise aus dem Altbestand tragen nackte Nummern. Ein Schülerausweis
// liefert gemessen `B97601826457` (Nummer des Kartenherstellers), ein Buchetikett eine
// 13-stellige EAN-13 — die Formen sind verschieden genug, dass die Reihenfolge hier
// eindeutig entscheidet.
//
// Die Lehrer-Stufe fehlte lange, und das war kein bewusster Ausschluss: Lehrkräfte
// stehen bei uns in `benutzer`, Schüler in `schueler`. Ein gescannter Lehrerausweis lief
// deshalb bis in die Volltextsuche und meldete „keine Treffer" — obwohl handleTeacherAction
// die passende Abfrage längst hatte, nur eben allein hinter dem L--Präfix. In Littera
// gibt es diesen Unterschied nicht; die Karte ist dieselbe, nur der Aufdruck lautet
// „Lehrerausweis".
//
// Die Lookups liefern bei Nichttreffer (nil, nil); ein non-nil Fehler ist daher ein
// echter DB-Fehler und wird propagiert (→ HTTP 500), statt ihn als "nicht gefunden" zu
// verschlucken.
func (s *defaultOmniboxService) resolveOhnePraefix(ctx context.Context, q OmniboxQuery, resp *OmniboxResult) error {
	copy, lookupErr := s.bookRepo.GetCopyByBarcode(ctx, q.Query)
	if lookupErr != nil {
		return fmt.Errorf("datenbankfehler bei Barcode-Auflösung: %w", lookupErr)
	}
	if copy != nil {
		return s.handleBookAction(ctx, q, resp)
	}

	// Littera-Buchetikett: Der Strichcode liefert eine EAN-13, im System steht die
	// kurze Mediennummer (siehe dekodiereLitteraEtikett). Erst rückrechnen, dann
	// erneut nachschlagen — nur ein existierendes Exemplar löst eine Aktion aus,
	// alles andere fällt unverändert zur Ausweis-/Volltextstufe durch.
	if nummer, istEtikett := dekodiereLitteraEtikett(q.Query); istEtikett {
		dekodiert, dekodiertErr := s.bookRepo.GetCopyByBarcode(ctx, nummer)
		if dekodiertErr != nil {
			return fmt.Errorf("datenbankfehler bei Etikett-Auflösung: %w", dekodiertErr)
		}
		if dekodiert != nil {
			q.Query = nummer
			return s.handleBookAction(ctx, q, resp)
		}
	}

	leser, leserErr := s.studentRepo.GetLeserByBarcode(ctx, q.Query)
	if leserErr != nil {
		return fmt.Errorf("datenbankfehler bei Ausweis-Auflösung: %w", leserErr)
	}
	if leser != nil {
		s.zeigeLeser(ctx, leser, resp)
		return nil
	}

	// Kein Buch, kein Ausweis: Was jemand getippt hat, ist eine Suche. Diese Stufe
	// trägt die Titelsuche der Theke und darf nie verschwinden.
	return s.handleSearchAction(ctx, q.Query, resp)
}

// mapDeviceResult mappt die Felder aus DeviceResult in die flache OmniboxResult-Struktur.
func (s *defaultOmniboxService) mapDeviceResult(dr *DeviceResult, resp *OmniboxResult) {
	if dr == nil {
		return
	}
	resp.Type = dr.Type
	resp.Geraet = dr.Geraet
	resp.Student = dr.Student
	resp.DueDate = dr.DueDate
	resp.LoanID = dr.LoanID
	resp.Fremdrueckgabe = dr.Fremdrueckgabe
	resp.Vorbesitzer = dr.Vorbesitzer
}

// handleAusweisAction lädt den Leser zu einem Ausweis mit Vorsilbe „S-" oder „L-".
//
// Bis zum 15.09.2026 entschied die Vorsilbe die Tabelle: „S-" nur Schüler, „L-" nur Lehrkräfte.
// Littera kennt die Vorsilben nicht, auf dem Ausweis stehen sie nicht, und die Littera-Übernahme
// gibt einem Schüler ohne eindeutige Nummer „L-<Littera-Nummer>" (internal/littera, ausweis) —
// der war an der Theke weder über den Ausweis noch über die Namenssuche ladbar (OFFEN.md 5.15).
// Die Vorsilbe sagt jetzt nur noch „Ausweis": Eine unbekannte Nummer bleibt ein lauter Fehler
// und verschwindet nicht in der Volltextsuche.
func (s *defaultOmniboxService) handleAusweisAction(ctx context.Context, query string, resp *OmniboxResult) error {
	leser, err := s.studentRepo.GetLeserByBarcode(ctx, query)
	if err != nil {
		return err
	}
	if leser == nil {
		return fmt.Errorf("%w: Ausweis %s ist nicht registriert", ErrNotFound, query)
	}
	s.zeigeLeser(ctx, leser, resp)
	return nil
}

// leserIDPraefix markiert die Auswahl aus der Trefferliste (siehe ProcessQuery).
const leserIDPraefix = "leser:"

// handleLeserIDAction lädt den Leser, den jemand in der Trefferliste angeklickt hat.
//
// Dieselbe Tür wie der Ausweis-Scan — nur der Schlüssel ist ein anderer: Die
// Trefferliste kennt die ID, der Scanner die Nummer. Beide enden in zeigeLeser, damit
// der Abholfach-Hinweis nicht an einem der beiden Wege fehlt.
//
// Eine unbekannte ID ist ein lauter Fehler und fällt NICHT in die Volltextsuche: Dort
// stünde „keine Treffer", und der Klick sähe aus, als sei nichts passiert.
func (s *defaultOmniboxService) handleLeserIDAction(ctx context.Context, id string, resp *OmniboxResult) error {
	leser, err := s.studentRepo.GetLeserByID(ctx, id)
	if err != nil {
		return fmt.Errorf("datenbankfehler bei der Leser-Auflösung: %w", err)
	}
	if leser == nil {
		return fmt.Errorf("%w: dieser Leser steht nicht mehr in der Leserdatei", ErrNotFound)
	}
	s.zeigeLeser(ctx, leser, resp)
	return nil
}

// zeigeLeser legt den gefundenen Leser in die Antwort, mit dem Abholfach-Hinweis.
//
// Der Hinweis gilt Vormerkungen, und die legen nur Schüler an — für einen Kollegen ist
// die Liste leer, ohne dass es hier einer Fallunterscheidung bedarf.
func (s *defaultOmniboxService) zeigeLeser(ctx context.Context, leser *repository.Student, resp *OmniboxResult) {
	resp.Type = "student"
	resp.Student = leser
	resp.Abholbereit = s.ladeAbholbereiteVormerkungen(ctx, leser.ID)
}

// ladeAbholbereiteVormerkungen holt die abholbereiten Vormerkungen des Schülers
// für den Abholfach-Hinweis. Ein Fehler hier bricht den Scan NICHT ab — die
// Theke wäre sonst wegen eines Hinweises arbeitsunfähig — sondern wird geloggt
// (dieselbe Abwägung wie resolveFotoURL im Profil).
func (s *defaultOmniboxService) ladeAbholbereiteVormerkungen(ctx context.Context, schuelerID string) []AbholbereiteVormerkung {
	// Die Unit-Tests der Scan-Weiche bauen den Service ohne Pool (Stub-Repos);
	// dort gibt es keine Vormerkungen und nichts zu laden.
	if s.pool == nil {
		return nil
	}
	rows, err := s.pool.Query(ctx, `
		SELECT t.titel, v.bereitgestellt_bis
		FROM vormerkungen v
		JOIN buecher_titel t ON t.id = v.titel_id
		WHERE v.schueler_id = $1 AND v.status = 'abholbereit'
		ORDER BY v.bereitgestellt_bis ASC NULLS LAST
		LIMIT 5`, schuelerID)
	if err != nil {
		log.Printf("omnibox: Abholfach-Hinweis nicht ladbar für Schüler %s: %v", schuelerID, err)
		return nil
	}
	defer rows.Close()

	var out []AbholbereiteVormerkung
	for rows.Next() {
		var v AbholbereiteVormerkung
		if err := rows.Scan(&v.Titel, &v.BereitgestelltBis); err != nil {
			log.Printf("omnibox: Abholfach-Hinweis unlesbar: %v", err)
			return nil
		}
		out = append(out, v)
	}
	if err := rows.Err(); err != nil {
		log.Printf("omnibox: Abholfach-Hinweis unvollständig: %v", err)
		return nil
	}
	return out
}

// handleSearchAction führt eine Volltextsuche über Buchtitel, Autoren, ISBN und Systematik aus.
func (s *defaultOmniboxService) handleSearchAction(ctx context.Context, query string, resp *OmniboxResult) error {
	titles, err := s.bookRepo.SearchTitles(ctx, query)
	if err != nil {
		return err
	}
	resp.Type = "search_results"
	resp.SearchResults = titles
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
func (s *defaultOmniboxService) versucheReaktivierung(ctx context.Context, query string, copy *repository.BookCopy, staffID string, resp *OmniboxResult) (fertig bool, err error) {
	activeLoan, err := s.loanRepo.GetActiveLoanByCopyID(ctx, copy.ID)
	if err != nil {
		return false, err
	}

	if activeLoan == nil {
		befund, err := s.holeExemplarZurueck(ctx, copy.ID, staffID)
		if err != nil {
			return false, err
		}
		copy.IstAusleihbar = true
		copy.IstAusgesondert = false
		copy.ZustandNotiz = ""

		resp.Message = rueckkehrMeldung(befund)
		resp.AufsichtInformieren = aufsichtHinweis(befund)
		resp.Type = "info"
		return true, nil
	}

	if copy.IstAusgesondert {
		return false, fmt.Errorf("%w: Buchexemplar %s ist ausgesondert und kann nicht ausgeliehen werden", ErrInvalidState, query)
	}
	return false, fmt.Errorf("%w: Buchexemplar ist nicht ausleihbar", ErrInvalidState)
}

// holeExemplarZurueck ist der Online-Scan über dem Baustein repository.HoleExemplarZurueck:
// eigene Transaktion, weil hier nichts weiter folgt. Das Nachbuchen (Stufe 2) legt den
// Baustein in seine eigene Buchungstransaktion.
func (s *defaultOmniboxService) holeExemplarZurueck(ctx context.Context, exemplarID, staffID string) (repository.RueckkehrBefund, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return repository.RueckkehrBefund{}, err
	}
	defer db.SafeRollback(ctx, tx)

	befund, err := repository.HoleExemplarZurueck(ctx, tx, exemplarID, staffID, nil)
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
		return fmt.Sprintf("Buch reaktiviert. Die Forderung über %.2f € wurde storniert — das Buch ist zurück.", b.StornierterBetrag)
	}
	return "Buch reaktiviert"
}

// aufsichtHinweis ist der Satz für den Fall, den die Anwendung NICHT erledigen kann: Der
// Bescheid liegt bei der Schulaufsicht, die Forderung bleibt offen, und jemand muss zum
// Telefon greifen. Leer, wenn nichts zu tun ist.
func aufsichtHinweis(b repository.RueckkehrBefund) string {
	return b.AufsichtHinweis()
}

// handleBookAction verarbeitet das Scannen eines Buch-Barcodes.
// Wenn kein aktiver Ausleiher vorhanden ist, wird das Buch zurückgegeben.
// Ist ein Schüler oder Lehrer aktiv, wird das Buch an diesen ausgeliehen.
func (s *defaultOmniboxService) handleBookAction(ctx context.Context, q OmniboxQuery, resp *OmniboxResult) error {
	copy, err := s.bookRepo.GetCopyByBarcode(ctx, q.Query)
	if err != nil {
		return err
	}
	if copy == nil {
		return fmt.Errorf("%w: Buchexemplar-Barcode %s wurde nicht gefunden", ErrNotFound, q.Query)
	}

	// Gesperrte/ausgesonderte Exemplare ggf. automatisch reaktivieren.
	if !copy.IstAusleihbar || copy.IstAusgesondert {
		fertig, err := s.versucheReaktivierung(ctx, q.Query, copy, q.StaffID, resp)
		if err != nil {
			return err
		}
		if fertig {
			return nil
		}
	}

	// Ausleihe durchführen, falls ein aktiver Ausleiher vorhanden ist
	if q.ActiveLeserID != nil && *q.ActiveLeserID != "" {
		lr, err := s.loanSvc.HandleUnifiedCheckout(ctx, copy, q.ActiveLeserID, q.StaffID, q.OverrideBlock)
		if err != nil {
			return err
		}
		s.mapLoanResult(lr, resp)
		return nil
	}

	// Rückgabe durchführen, wenn kein aktiver Ausleiher vorhanden ist
	lr, err := s.loanSvc.HandleSimpleReturn(ctx, copy, q.StaffID)
	if err != nil {
		return err
	}
	s.mapLoanResult(lr, resp)
	return nil
}

// mapLoanResult mappt die Felder aus LoanResult in die OmniboxResult-Struktur.
func (s *defaultOmniboxService) mapLoanResult(lr *LoanResult, resp *OmniboxResult) {
	if lr == nil {
		return
	}
	resp.Type = lr.Type
	resp.Book = lr.Book
	resp.Student = lr.Student
	resp.DueDate = lr.DueDate
	resp.LoanID = lr.LoanID
	resp.Fremdrueckgabe = lr.Fremdrueckgabe
	resp.Vorbesitzer = lr.Vorbesitzer
	resp.HasVormerkung = lr.HasVormerkung
	resp.VormerkungTitel = lr.VormerkungTitel
	resp.VormerkungUser = lr.VormerkungUser
	resp.RegalfreigabeBarcode = lr.RegalfreigabeBarcode
}
