package theke

import (
	"context"
	"errors"
	"log"
	"net/http"
	"time"

	"bibliothek/api/tuer"
	"bibliothek/apierrors"
	"bibliothek/auth"
	"bibliothek/internal/service"
	"bibliothek/repository"
)

// Die Nachbuch-Tür der Theke.
//
// Der Theken-Rechner schickt seine Warteschlange, der Server bucht je Eintrag die Wirklichkeit
// und antwortet je Eintrag mit einem Ergebnis. Abweichungen vom Scan stehen zusätzlich als
// Nachbuch-Meldung am Server, bis jemand sie quittiert; nichts verschwindet still.
//
// Die Tür ist dünn: Sie prüft die Form, misst die Uhr des Rechners, klärt den
// Idempotenz-Schlüssel (nachbuchen_schluessel.go) und reicht durch. Die Regeln liegen im
// Dienst (internal/service/nachbuchen.go), das SQL im repository, wie überall.

// nachbuchenHoechstens: 1–50 Einträge je Aufruf. Mehr wäre eine Transaktion, die zu lange
// offen steht; die Theke schickt in Portionen.
const nachbuchenHoechstens = 50

// uhrVersatzWarnenAb: Ab diesem Versatz steht die Uhr des Theken-Rechners im Log. Gebucht wird
// trotzdem richtig (der Versatz wird herausgerechnet); die Zeile sagt nur, dass die Uhr eines
// Rechners gestellt werden sollte.
const uhrVersatzWarnenAb = 2 * time.Minute

// nachbuchAnfrage ist eine Portion der Warteschlange.
type nachbuchAnfrage struct {
	// GesendetAm ist die Uhrzeit des Theken-Rechners beim Versand dieser Portion — von derselben
	// Uhr wie gescannt_am. Der Server misst daran den Versatz der Rechner-Uhr und rechnet die
	// Scan-Zeitpunkte der Portion auf seine Uhr um. Pflicht: Ohne sie ließe sich eine falsch
	// gehende Uhr nicht erkennen.
	GesendetAm time.Time         `json:"gesendet_am" validate:"required"`
	Eintraege  []nachbuchEintrag `json:"eintraege" validate:"required,min=1,max=50,dive"`
}

// nachbuchEintrag ist ein offline gescannter Vorgang.
type nachbuchEintrag struct {
	// Schluessel ist der Idempotenz-Schlüssel des Eintrags — derselbe, den der
	// Online-Versand benutzt hätte. Daran erkennt der Server einen abgebrochenen Versand.
	Schluessel string `json:"schluessel" validate:"required,uuid"`
	// Absicht: was der Bediener beim Scan meinte ("ausleihe" | "rueckgabe").
	Absicht string `json:"absicht" validate:"required,oneof=ausleihe rueckgabe"`
	// Barcode des Buchs, wie gescannt (B-…, nackte Ziffern, LMF-…).
	Barcode string `json:"barcode" validate:"required"`
	// GescanntAm ist der Zeitpunkt am Theken-Rechner, nach seiner Uhr. Der Server rechnet den
	// Versatz der Uhr heraus (gesendet_am) und nimmt höchstens seine eigene Zeit.
	GescanntAm time.Time `json:"gescannt_am" validate:"required"`
	// Person: was der Rechner beim Scan schon auflösen konnte …
	LeserID *string `json:"leser_id,omitempty" validate:"omitempty,uuid_oder_leer"`
	// … sonst der offline gescannte Ausweis, den der Server auflöst.
	AusweisBarcode *string `json:"ausweis_barcode,omitempty"`
}

// nachbuchErgebnis ist die Antwort zu einem Eintrag.
type nachbuchErgebnis struct {
	Schluessel string `json:"schluessel"`
	// Ergebnis: ausgeliehen · umgebucht · bereits_ausgeliehen · zurueckgegeben ·
	// nur_reaktiviert · nicht_gebucht · veraltet · bereits_gebucht · wiederholen.
	Ergebnis string `json:"ergebnis"`
	// Grund steht bei nicht_gebucht, veraltet und wiederholen — der Satz, den die Meldung trägt.
	Grund string `json:"grund,omitempty"`
	// Daten ist die gebuchte Wirkung in der Form des Theken-Scans (bei Erfolg und bei
	// bereits_gebucht die Wirkung des Online-Versands).
	Daten *eingabeAntwort `json:"daten,omitempty"`
	// AufsichtInformieren: Das Buch stand auf einem Bescheid, der schon bei der
	// Schulaufsicht liegt — eigenes Feld, weil das eine Aufgabe ist, keine Meldung.
	AufsichtInformieren string `json:"aufsicht_informieren,omitempty"`
}

// nachbuchAntwort ist die Antwort auf eine Portion.
type nachbuchAntwort struct {
	// UhrVersatzSekunden ist der gemessene Versatz der Rechner-Uhr (Serverzeit minus Rechnerzeit):
	// negativ = die Uhr des Rechners geht vor. Die Theke kann eine falsch gehende Uhr damit melden.
	UhrVersatzSekunden int                `json:"uhr_versatz_sekunden"`
	Ergebnisse         []nachbuchErgebnis `json:"ergebnisse"`
}

// nachbuchWiederholen: Der Server konnte diesen Eintrag nicht beurteilen (Datenbank weg, oder
// eine Buchung unter demselben Schlüssel läuft noch). Er bleibt auf dem Theken-Rechner liegen und
// wird später erneut geschickt — deshalb steht er nicht in den Ergebnissen der Meldungstabelle.
const nachbuchWiederholen = "wiederholen"

// nachbuchBereitsGebucht: Der Online-Versand unter demselben Schlüssel hat den Scan schon
// vollständig gebucht, nur seine Antwort kam nicht an. Nichts wird gebucht, nichts gemeldet;
// Daten trägt die online gebuchte Wirkung. Kein Wort der Meldungstabelle.
const nachbuchBereitsGebucht = "bereits_gebucht"

// nachbuchung bucht einen Eintrag der Warteschlange nach (service.NachbuchDienst). Fachliche
// Ausgänge kommen als Ergebnis zurück; ein Fehler ist ein Serverfehler, und der Eintrag bleibt
// auf dem Rechner liegen.
type nachbuchung interface {
	BucheNach(ctx context.Context, eintrag service.NachbuchEintrag) (*service.NachbuchErgebnis, error)
}

// WarteschlangeNachbuchen nimmt eine Portion der Warteschlange entgegen.
// @Summary      Offline-Warteschlange nachbuchen
// @Description  Bucht je Eintrag die Wirklichkeit (Umbuchung, Rückgabe, Rückholen) und meldet jede Abweichung vom Scan.
// @Tags         theke
// @Accept       json
// @Produce      json
// @Param        body body nachbuchAnfrage true "Einträge der Warteschlange"
// @Success      200 {object} nachbuchAntwort
// @Router       /action/nachbuchen [post]
func (t *Tueren) WarteschlangeNachbuchen(nachbuchDienst nachbuchung) http.HandlerFunc {
	return apierrors.MitFehlerantwort(func(w http.ResponseWriter, r *http.Request) error {
		return t.bucheWarteschlangeNach(w, r, nachbuchDienst)
	})
}

// bucheWarteschlangeNach bucht eine Portion der Warteschlange, Eintrag für Eintrag.
func (t *Tueren) bucheWarteschlangeNach(w http.ResponseWriter, r *http.Request, nachbuchDienst nachbuchung) error {
	// Die Empfangszeit wird vor dem Einlesen festgehalten. Der gemessene Versatz enthält die
	// Laufzeit der Anfrage bis hierher: Umgerechnet liegt ein Scan um diese Laufzeit zu spät,
	// im Schulnetz Millisekunden — weniger, als ein Buch braucht, um von einer Theke zur
	// anderen zu kommen.
	empfangen := time.Now()
	claims, ok := auth.ClaimsAusKontext(r.Context())
	if !ok {
		return apierrors.Unauthorized(auth.MeldungNichtAngemeldet, errors.New("missing session information"))
	}
	var req nachbuchAnfrage
	if !tuer.LeseAnfrage(w, r, &req) {
		return nil
	}
	if len(req.Eintraege) > nachbuchenHoechstens {
		return apierrors.BadRequest("Höchstens 50 Einträge je Aufruf", errors.New("zu viele einträge"))
	}

	versatz := empfangen.Sub(req.GesendetAm)
	if versatz >= uhrVersatzWarnenAb || versatz <= -uhrVersatzWarnenAb {
		log.Printf("nachbuchen: WARNUNG die Uhr des Theken-Rechners weicht um %s ab (Konto %s) — Scan-Zeitpunkte werden umgerechnet, die Uhr sollte gestellt werden",
			(-versatz).Round(time.Second), claims.KontoID)
	}

	ctx := r.Context()
	antwort := nachbuchAntwort{
		UhrVersatzSekunden: int(versatz.Round(time.Second) / time.Second),
		Ergebnisse:         make([]nachbuchErgebnis, 0, len(req.Eintraege)),
	}
	// Die Zahl vor dem Nachbuchen. Gemeldet wird hinterher an der Wirkung, nicht an
	// einer Liste der Ergebnisse, die eine Meldung erzeugen: Welche das sind, weiß der
	// Dienst, und eine zweite Liste hier wäre die zweite Wahrheitsquelle — sie stimmte
	// genau bis zur nächsten Meldung, die jemand hinzufügt.
	offenVorher, zaehlbar := t.zaehleMeldungen(ctx)
	absender := nachbuchAbsender{bearbeiterID: claims.KontoID, uhrVersatz: versatz, darfSperrgrundSehen: t.HatRecht(r, "view_students")}
	for _, e := range req.Eintraege {
		antwort.Ergebnisse = append(antwort.Ergebnisse, t.bucheEintragNach(ctx, nachbuchDienst, e, absender))
	}
	if zaehlbar {
		if offenNachher, ok := t.zaehleMeldungen(ctx); ok && offenNachher != offenVorher {
			t.meldeMeldungsstand()
		}
	}
	tuer.AntworteJSON(w, http.StatusOK, antwort)
	return nil
}

// nachbuchAbsender ist, was für jeden Eintrag einer Portion gleich ist: wer sie schickt, wie
// seine Uhr geht und ob er den Freitext einer Sperre lesen darf (view_students).
type nachbuchAbsender struct {
	bearbeiterID        string
	uhrVersatz          time.Duration
	darfSperrgrundSehen bool
}

// bucheEintragNach bucht einen Eintrag. Ein Serverfehler wird nicht zum Fehler des ganzen
// Aufrufs: Die übrigen Einträge sollen durchlaufen, und der gescheiterte bleibt auf dem
// Rechner liegen („wiederholen"). Schweigen wäre hier das Schlimmste — die Theke hielte
// ihn für erledigt, und erledigt ist nur, was der Server gebucht hat.
func (t *Tueren) bucheEintragNach(ctx context.Context, dienst nachbuchung, e nachbuchEintrag, absender nachbuchAbsender) nachbuchErgebnis {
	lage, err := t.ergreifeNachbuchSchluessel(ctx, e)
	if err != nil {
		log.Printf("nachbuchen: Schlüssel %s nicht lesbar: %v", e.Schluessel, err)
		return nachbuchErgebnis{Schluessel: e.Schluessel, Ergebnis: nachbuchWiederholen, Grund: "Server nicht erreichbar"}
	}
	if lage.antwort != nil {
		fertig := *lage.antwort
		fertig.Schluessel = e.Schluessel
		return fertig
	}
	erg, err := dienst.BucheNach(ctx, service.NachbuchEintrag{
		Schluessel: e.Schluessel, Absicht: e.Absicht, Barcode: e.Barcode,
		GescanntAm: e.GescanntAm, UhrVersatz: absender.uhrVersatz,
		LeserID: e.LeserID, AusweisBarcode: e.AusweisBarcode,
		NachFremdrueckgabeVon: lage.fremdrueckgabeVon, BearbeiterID: absender.bearbeiterID,
	})
	if err != nil {
		log.Printf("nachbuchen: Eintrag %s konnte nicht gebucht werden: %v", e.Schluessel, err)
		t.gibNachbuchSchluesselZurueck(ctx, e.Schluessel, lage)
		return nachbuchErgebnis{Schluessel: e.Schluessel, Ergebnis: nachbuchWiederholen, Grund: "konnte nicht gebucht werden"}
	}
	out := nachbuchErgebnis{
		Schluessel: e.Schluessel, Ergebnis: erg.Ergebnis, Grund: erg.Grund,
		AufsichtInformieren: erg.AufsichtHinweis,
	}
	if erg.Schranke != nil {
		// Abgelegt wird der Grund ohne den Freitext der Sperre, gleich wer die Portion schickt:
		// Die Warteschlange gehört dem Rechner, und die Wiederholung kann unter der Anmeldung
		// einer Person kommen, die ihn nicht lesen darf.
		out.Grund = service.MeldungAus(ohneSperrgrund(erg.Schranke, false))
	}
	if erg.Gebucht != nil {
		out.Daten = zurEingabeAntwort(service.OmniboxErgebnisAus(erg.Gebucht))
	}
	t.legeNachbuchErgebnisAb(ctx, out)
	if erg.Schranke != nil {
		// Die Antwort an diesen Absender kürzt dieselbe Funktion wie an der Online-Theke.
		out.Grund = service.MeldungAus(ohneSperrgrund(erg.Schranke, absender.darfSperrgrundSehen))
	}
	return out
}

// zaehleMeldungen liefert die Zahl der offenen Nachbuch-Meldungen. Der zweite Wert ist
// false, wenn sie sich nicht lesen ließ — dann wird nichts gemeldet, statt eine Änderung
// zu behaupten. Der Zähler holt sich seinen Stand bei der nächsten Anmeldung ohnehin neu.
func (t *Tueren) zaehleMeldungen(ctx context.Context) (int, bool) {
	n, err := repository.ZaehleOffeneNachbuchMeldungen(ctx, t.DB.Pool)
	if err != nil {
		log.Printf("nachbuchen: Meldungen nicht zählbar: %v", err)
		return 0, false
	}
	return n, true
}

// meldeMeldungsstand sagt allen Arbeitsplätzen, dass sich die Zahl der offenen Meldungen
// geändert hat. Ohne Inhalt: Die Zeilen nennen Schüler und Vorbesitzer, und die Leitung
// geht an jede Sitzung — auch an die eines Helfers, der die Liste nicht sehen darf. Wer
// sie sehen darf, holt sie sich hinter seinem Recht ab.
func (t *Tueren) meldeMeldungsstand() {
	if t.Broker != nil {
		t.Broker.Verteile("nachbuch-meldungen", "{}")
	}
}
