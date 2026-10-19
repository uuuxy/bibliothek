package theke

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"bibliothek/api/tuer"
	"bibliothek/apierrors"
	"bibliothek/auth"
	"bibliothek/internal/service"
	"bibliothek/repository"
)

// Merkmal einer Sperre (internal/service/sperr_merkmal.go): Am Kopf der Antwort wählt der
// Dialog der Theke, was er anbietet: „uebergehbar" das einmalige Übergehen, „leser" das
// Aufheben der Sperre am Leser. Das Feld in der Ablage trägt es durch eine Wiederholung mit
// demselben Idempotenz-Schlüssel (sperr_merkmal_test.go).
const (
	sperrKopf        = "X-Sperre"
	sperrUebergehbar = "uebergehbar"
	sperrAmLeser     = "leser"
	sperrAblageFeld  = "sperre"
)

// sperrMerkmal liest das Merkmal aus der Fehlerkette; leer heißt: keine Sperre, die die
// Theke übergehen oder aufheben lässt.
func sperrMerkmal(err error) string {
	switch {
	case service.IstUebergehbareSperre(err):
		return sperrUebergehbar
	case service.IstSperreAmLeser(err):
		return sperrAmLeser
	}
	return ""
}

// ohneSperrgrund nimmt einem Sperr-Fehler den Freitext (block_reason), wenn der Aufrufer ihn
// nicht sehen darf. Die Theke erfährt weiterhin, dass gesperrt ist (403, allgemeine Meldung),
// aber nicht, warum: Der Grund kann Zahlungsrückstände oder Familieninterna nennen
// (PII-Matrix Stufe 2). Dieselbe Grenze wie die Geräteliste: view_students entscheidet, nicht
// die Route.
func ohneSperrgrund(err error, darfGrundSehen bool) error {
	var sg *service.SperrgrundFehler
	if err == nil || darfGrundSehen || !errors.As(err, &sg) {
		return err
	}
	return fmt.Errorf("%w — bitte an die Bibliotheksleitung wenden", sg.Kern)
}

func statusDesFehlers(err error) int {
	switch {
	case errors.Is(err, service.ErrNichtGefunden):
		return http.StatusNotFound
	case errors.Is(err, service.ErrGesperrt):
		return http.StatusForbidden
	case errors.Is(err, service.ErrUngueltigerZustand):
		return http.StatusBadRequest
	case errors.Is(err, service.ErrKonflikt):
		return http.StatusConflict
	default:
		return http.StatusInternalServerError
	}
}

// idempotenzLage ist das Ergebnis von erlangeIdempotenz: genau eines der drei gilt, oder
// keines (kein Schlüssel, oder die Datenbank hat die Reservierung verweigert; dann läuft die
// Anfrage ohne Schutz).
type idempotenzLage struct {
	reserviert bool                          // der Schlüssel gehört dieser Anfrage
	antwort    *repository.IdempotenzAntwort // eine frühere Anfrage hat schon geantwortet
	inArbeit   bool                          // eine frühere Anfrage arbeitet noch, Warten hat nicht gereicht
}

// idempotenzWartezeit: wie lange eine Anfrage auf die Antwort einer laufenden mit demselben
// Schlüssel wartet, bevor sie „in Arbeit" meldet. Tests setzen IdempotenzWartezeit.
func (t *Tueren) idempotenzWartezeit() time.Duration {
	if t.IdempotenzWartezeit > 0 {
		return t.IdempotenzWartezeit
	}
	return 3 * time.Second
}

// erlangeIdempotenz reserviert den Schlüssel vor der Arbeit. Würde die Antwort erst danach
// eingefügt, fände eine zweite Anfrage mit demselben Schlüssel, die nach dem Commit der ersten
// und vor dem Speichern ihrer Antwort eintrifft, nichts und buchte neu: Das Buch läge schon
// beim Kind und würde zurückgenommen (idempotenz_pg_test.go). Läuft die Arbeit noch, wartet
// die zweite Anfrage auf die Antwort; ist die Reservierung nach einem Serverfehler
// freigegeben, übernimmt sie den Schlüssel.
func (t *Tueren) erlangeIdempotenz(ctx context.Context, schluessel string) idempotenzLage {
	if schluessel == "" {
		return idempotenzLage{}
	}
	frist := time.Now().Add(t.idempotenzWartezeit())
	for {
		reserviert, antwort, err := repository.ReserviereIdempotenzSchluessel(ctx, t.DB.Pool, schluessel)
		if err != nil {
			log.Printf("idempotenz: Schlüssel nicht reservierbar, Anfrage läuft ohne Schutz: %v", err)
			return idempotenzLage{}
		}
		if reserviert {
			return idempotenzLage{reserviert: true}
		}
		if !antwort.IstInArbeit() {
			return idempotenzLage{antwort: antwort}
		}
		if ctx.Err() != nil || time.Now().After(frist) {
			return idempotenzLage{inArbeit: true}
		}
		time.Sleep(100 * time.Millisecond)
	}
}

// IdempotenzSpeicherfrist sagt, wie lange legeAntwortAb nach der Arbeit noch schreiben darf,
// mit WithoutCancel, also auch nach Ablauf der Bearbeitungsfrist. Bearbeitungsfrist plus
// diese Frist müssen unter repository.IdempotenzReservierungsfrist bleiben, sonst kann die
// Waisenübernahme einen Besitzer treffen, der noch lebt (api/idempotenz_fristen_test.go).
const IdempotenzSpeicherfrist = 5 * time.Second

// legeAntwortAb schreibt die Antwort in die Reservierung, mit WithoutCancel (Vorbild
// api/mahnwesen/mahnlisten_versand.go): Bricht der Aufrufer ab, nachdem die Buchung
// committet ist, muss die Antwort trotzdem stehen, sonst bucht die Wiederholung neu. Ein
// Serverfehler wird nicht gespeichert, sondern gibt den Schlüssel frei: Die Wiederholung soll
// es neu versuchen.
func (t *Tueren) legeAntwortAb(ctx context.Context, schluessel string, daten any, status int) {
	if schluessel == "" {
		return
	}
	ctx, abbruch := context.WithTimeout(context.WithoutCancel(ctx), IdempotenzSpeicherfrist)
	defer abbruch()
	if status >= 500 {
		if _, err := repository.GibIdempotenzSchluesselFrei(ctx, t.DB.Pool, schluessel); err != nil {
			log.Printf("idempotenz: Schlüssel nach Serverfehler nicht freigegeben: %v", err)
		}
		return
	}
	antwortDaten, err := json.Marshal(daten)
	if err != nil {
		log.Printf("idempotenz: Antwort konnte nicht serialisiert werden: %v", err)
		return
	}
	if err := repository.SchreibeIdempotenzAntwort(ctx, t.DB.Pool, schluessel, antwortDaten, status); err != nil {
		log.Printf("idempotenz: Antwort nicht gespeichert: %v", err)
	}
}

// antworteAusAblage liefert die gespeicherte Antwort aus, falls unversehrt.
// true = Antwort wurde gesendet (Aufrufer beendet); false = beschädigter Eintrag, Aufrufer
// berechnet neu.
func (t *Tueren) antworteAusAblage(w http.ResponseWriter, antwort *repository.IdempotenzAntwort) bool {
	daten, status := antwort.Daten, antwort.Status

	if status >= 400 {
		var fehlerDaten map[string]string
		if uerr := json.Unmarshal(daten, &fehlerDaten); uerr != nil {
			log.Printf("idempotenz: beschädigte Fehler-Antwort im Cache, wird neu berechnet: %v", uerr)
			return false
		}
		if merkmal := fehlerDaten[sperrAblageFeld]; merkmal == sperrUebergehbar || merkmal == sperrAmLeser {
			w.Header().Set(sperrKopf, merkmal)
		}
		apierrors.AntworteFehlerMitMeldung(w, status, fehlerDaten["error"], nil)
		return true
	}

	var gespeichert eingabeAntwort
	if uerr := json.Unmarshal(daten, &gespeichert); uerr != nil {
		log.Printf("idempotenz: beschädigte Antwort im Cache, wird neu berechnet: %v", uerr)
		return false
	}
	tuer.AntworteJSON(w, status, gespeichert)
	return true
}

// eingabeVerarbeitung erkennt, ob eine Eingabe der Theke einen Leser, ein Exemplar oder ein
// Gerät meint oder eine Suche ist, und bucht, was zu buchen ist (service.OmniboxDienst). Ein
// fachlicher Fehler trägt service.ErrNichtGefunden, ErrGesperrt, ErrUngueltigerZustand oder
// ErrKonflikt; eine Sperre nennt ihren Grund als service.SperrgrundFehler.
type eingabeVerarbeitung interface {
	VerarbeiteEingabe(ctx context.Context, eingabe service.OmniboxEingabe) (*service.OmniboxErgebnis, error)
}

// EingabeVerarbeiten nimmt Scan und Suche der Omnibox an (POST /api/action).
func (t *Tueren) EingabeVerarbeiten(omnibox eingabeVerarbeitung) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		t.verarbeiteEingabe(w, r, omnibox)
	}
}

// verarbeiteEingabe nimmt einen Scan oder eine Suche der Theke entgegen und reicht sie an die
// Omnibox.
func (t *Tueren) verarbeiteEingabe(w http.ResponseWriter, r *http.Request, omnibox eingabeVerarbeitung) {
	claims, ok := auth.ClaimsAusKontext(r.Context())
	if !ok {
		apierrors.AntworteFehlerMitMeldung(w, http.StatusUnauthorized, auth.MeldungNichtAngemeldet, nil)
		return
	}

	var req EingabeAnfrage
	if !tuer.LeseAnfrage(w, r, &req) {
		return
	}

	req.Text = strings.TrimSpace(req.Text)
	if req.Text == "" {
		apierrors.AntworteFehlerMitMeldung(w, http.StatusBadRequest, "Bitte einen Barcode scannen oder einen Suchbegriff eingeben", nil)
		return
	}

	ctx := r.Context()

	lage := t.erlangeIdempotenz(ctx, req.IdempotenzSchluessel)
	if lage.antwort != nil && t.antworteAusAblage(w, lage.antwort) {
		return
	}
	if lage.inArbeit {
		apierrors.AntworteFehlerMitMeldung(w, http.StatusConflict,
			"Dieser Scan wird gerade gebucht. Bitte einen Moment warten und erneut scannen.", nil)
		return
	}

	// Einen Hinweis übergehen ist ein Verwaltungsakt, kein Theken-Vorgang: override_block wirkt
	// nur mit edit_students, demselben Recht, das auch das Sperren und Entsperren erlaubt. Wer
	// das Feld ohne das Recht setzt, wird behandelt, als hätte er es nicht gesetzt. Übergangen
	// werden am Buch wie am Gerät nur die Hinweise (offene Forderung, Überfällig-Automatik),
	// keine Sperre am Leser.
	ergebnis, err := omnibox.VerarbeiteEingabe(ctx, service.OmniboxEingabe{
		Text:                 req.Text,
		AktiverLeserID:       req.AktiverLeserID,
		ChecklisteBestaetigt: req.ChecklisteBestaetigt,
		BearbeiterID:         claims.KontoID,
		SperreUebergehen:     req.SperreUebergehen && t.HatRecht(r, "edit_students"),
	})

	if err != nil {
		// Vor dem Ablegen kürzen, sonst läge der Freitext in der Ablage.
		err = ohneSperrgrund(err, t.HatRecht(r, "view_students"))
		status := statusDesFehlers(err)
		meldung := service.MeldungAus(err)
		ablageDaten := map[string]string{"error": meldung}
		if merkmal := sperrMerkmal(err); merkmal != "" {
			// Merkmal für den Sperr-Dialog der Theke: im Kopf, damit der Rumpf die eine
			// Fehlerform behält, und in der Ablage, damit eine Wiederholung es nicht verliert
			// (sperr_merkmal_test.go).
			w.Header().Set(sperrKopf, merkmal)
			ablageDaten[sperrAblageFeld] = merkmal
		}
		t.legeAntwortAb(ctx, req.IdempotenzSchluessel, ablageDaten, status)
		apierrors.AntworteFehlerMitMeldung(w, status, meldung, err)
		return
	}

	antwort := zurEingabeAntwort(ergebnis)

	t.legeAntwortAb(ctx, req.IdempotenzSchluessel, antwort, http.StatusOK)

	// Nach Ausleihe und Rückgabe erfahren es die übrigen Arbeitsplätze über die Live-Leitung.
	if antwort.Art == "ausleihe" || antwort.Art == "rueckgabe" {
		t.sendeThekenEreignis(*antwort)
	}

	tuer.AntworteJSON(w, http.StatusOK, antwort)
}

func zurEingabeAntwort(ergebnis *service.OmniboxErgebnis) *eingabeAntwort {
	if ergebnis == nil {
		return nil
	}
	return &eingabeAntwort{
		Art:                  ergebnis.Art,
		Meldung:              ergebnis.Meldung,
		Leser:                zumThekenLeser(ergebnis.Leser),
		Exemplar:             ergebnis.Exemplar,
		Geraet:               ergebnis.Geraet,
		Frist:                ergebnis.Frist,
		AusleiheID:           ergebnis.AusleiheID,
		Fremdrueckgabe:       ergebnis.Fremdrueckgabe,
		Vorbesitzer:          zumThekenLeser(ergebnis.Vorbesitzer),
		Treffer:              ergebnis.Treffer,
		HatVormerkung:        ergebnis.HatVormerkung,
		VormerkungTitel:      ergebnis.VormerkungTitel,
		VorgemerktFuer:       ergebnis.VorgemerktFuer,
		RegalfreigabeBarcode: ergebnis.RegalfreigabeBarcode,
		AufsichtInformieren:  ergebnis.AufsichtInformieren,
		Abholbereit:          zuAbholbereitenVormerkungen(ergebnis.Abholbereit),
		AuflagenHinweis:      ergebnis.AuflagenHinweis,
	}
}

// zuAbholbereitenVormerkungen überträgt den Abholfach-Hinweis in die Form der Antwort.
func zuAbholbereitenVormerkungen(vs []service.AbholbereiteVormerkung) []abholbereiteVormerkung {
	if len(vs) == 0 {
		return nil
	}
	out := make([]abholbereiteVormerkung, 0, len(vs))
	for _, v := range vs {
		out = append(out, abholbereiteVormerkung{Titel: v.Titel, BereitgestelltBis: v.BereitgestelltBis})
	}
	return out
}
