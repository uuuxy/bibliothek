package auth

import (
	"bibliothek/apierrors"
	"bibliothek/db"
	"bibliothek/pkg/clientip"
	"bibliothek/pkg/httpresp"
	"bibliothek/repository"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/jackc/pgx/v5"
)

// fehlversuche zählt die gescheiterten Anmeldungen eines Schlüssels in seinem Zeitfenster.
type fehlversuche struct {
	anzahl      int
	fensterEnde time.Time
}

// fehlversuchsBremse lässt je Schlüssel höchstens maxFehlversuche gescheiterte Anmeldungen im
// Zeitfenster zu. Sie schützt den Mailserver vor dem Durchprobieren von Passwörtern.
type fehlversuchsBremse struct {
	mu              sync.Mutex
	eintraege       map[string]*fehlversuche
	maxFehlversuche int
	fenster         time.Duration
}

func newFehlversuchsBremse(maxFehlversuche int, fenster time.Duration) *fehlversuchsBremse {
	return &fehlversuchsBremse{
		eintraege:       make(map[string]*fehlversuche),
		maxFehlversuche: maxFehlversuche,
		fenster:         fenster,
	}
}

// istGesperrt meldet, ob der Schlüssel im laufenden Fenster zu viele Fehlversuche hat.
func (b *fehlversuchsBremse) istGesperrt(schluessel string) bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	e, ok := b.eintraege[schluessel]
	if !ok {
		return false
	}
	if time.Now().After(e.fensterEnde) {
		delete(b.eintraege, schluessel)
		return false
	}
	return e.anzahl >= b.maxFehlversuche
}

// merkeFehlversuch zählt einen Fehlversuch; der erste beginnt ein neues Fenster.
func (b *fehlversuchsBremse) merkeFehlversuch(schluessel string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	e, ok := b.eintraege[schluessel]
	if !ok || time.Now().After(e.fensterEnde) {
		b.eintraege[schluessel] = &fehlversuche{anzahl: 1, fensterEnde: time.Now().Add(b.fenster)}
		return
	}
	e.anzahl++
	// Abgelaufene Einträge räumen, damit die Map nicht ohne Grenze wächst.
	if len(b.eintraege) > 2000 {
		for k, v := range b.eintraege {
			if time.Now().After(v.fensterEnde) {
				delete(b.eintraege, k)
			}
		}
	}
}

// adresseDerAnfrage liefert die Adresse des Browsers über pkg/clientip, das den vertrauten
// Proxy kennt. So hängt der Schlüssel der Bremse am Browser und nicht an der Adresse von Caddy.
func adresseDerAnfrage(r *http.Request) string {
	return clientip.AusAnfrage(r)
}

// anmeldeAnfrage ist der Rumpf der Anmeldung: nur E-Mail und Passwort.
//
// Eine Kennung, die der Browser behauptet, gehört nicht hierher: Ein signiertes Token mit einer
// solchen Kennung wäre eine Falle für den Nächsten, der ihr glaubt, weil signiert nach geprüft
// aussieht. Das Token trägt nur Konto, Rolle und Sitzung.
type anmeldeAnfrage struct {
	Email    string `json:"email,omitempty"`
	Passwort string `json:"password,omitempty"`
}

// anmeldeAntwort ist die Antwort einer gelungenen Anmeldung: Konto, Rolle und Rechte.
type anmeldeAntwort struct {
	KontoID string `json:"user_id"`
	// Email ist die Adresse, mit der sich die Person anmeldet. Der Sperrbildschirm braucht sie
	// für die Anmeldung nach dem Wiederherstellen der Sitzung, wo die Oberfläche sie sonst nicht
	// kennt.
	Email    string   `json:"email"`
	Rolle    Rolle    `json:"rolle"`
	Vorname  string   `json:"vorname"`
	Nachname string   `json:"nachname"`
	Rechte   []string `json:"permissions"`
}

// anmeldeFrist umfasst den Mailserver (imapFrist) und die Schritte in der Datenbank danach. Sie
// liegt über imapFrist, sonst endete der Kontext zwischen „Passwort richtig" und dem Lesen des
// Kontos, und eine richtige, langsame Anmeldung zählte als Fehlversuch
// (TestAnmeldeFrist_LaesstIMAPLuft).
const anmeldeFrist = imapFrist + 10*time.Second

// anmeldeRumpfMaxBytes begrenzt den Rumpf der Anmeldung (E-Mail und Passwort als JSON).
const anmeldeRumpfMaxBytes = 16 << 10

// Anmelden liefert die Tür der Anmeldung. Angemeldet wird nur mit E-Mail und Passwort gegen den
// Mailserver der Schule (IMAP).
func Anmelden(pool db.Pool, anmeldung *Anmeldung, nurHTTPS bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		adresse := adresseDerAnfrage(r)

		// Eigene Grenze für den Rumpf vor dem Lesen: Die Tür ist die eine ohne Anmeldung, die
		// JSON liest, und die Grenze des Servers (100 MB, für die Importe) ist hier keine.
		// json.Decoder puffert einen Text von 100 MB vollständig, fünfzig gleichzeitige Aufrufe
		// wären fünf Gigabyte Speicher ohne ein gültiges Passwort. E-Mail und Passwort passen in
		// ein paar hundert Byte.
		r.Body = http.MaxBytesReader(w, r.Body, anmeldeRumpfMaxBytes)

		var anfrage anmeldeAnfrage
		if err := json.NewDecoder(r.Body).Decode(&anfrage); err != nil {
			apierrors.AntworteFehlerMitMeldung(w, http.StatusBadRequest, apierrors.MeldungAnfrageUnlesbar, err)
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), anmeldeFrist)
		defer cancel()

		passwort, ok := pruefeAnmeldedaten(w, anfrage)
		if !ok {
			return
		}

		schluessel, ok := anmeldung.pruefeFehlversuche(w, anfrage.Email, adresse)
		if !ok {
			return
		}

		konto, ok := anmeldung.meldeAn(ctx, w, pool, anfrage.Email, passwort, schluessel)
		if !ok {
			return
		}

		rolle := Rolle(konto.rolle)
		// Ohne die Zeile gäbe es ein Token, das sich nach Inaktivität nicht sperren lässt; dann
		// lieber keine Anmeldung.
		sitzungID, err := anmeldung.Sitzungen.beginne(ctx, konto.id, passwort, time.Now().Add(anmeldung.laufzeit))
		if err != nil {
			apierrors.AntworteFehlerMitMeldung(w, http.StatusServiceUnavailable, meldungAnmeldedienstGestoert,
				fmt.Errorf("%w: sitzung anlegen: %w", errAnmeldedienstGestoert, err))
			return
		}
		token, err := anmeldung.ErzeugeToken(konto.id, rolle, sitzungID)
		if err != nil {
			apierrors.AntworteFehler(w, http.StatusInternalServerError, err)
			return
		}

		http.SetCookie(w, &http.Cookie{
			Name:     "session_token",
			Value:    token,
			Path:     "/",
			Expires:  time.Now().Add(anmeldung.laufzeit),
			HttpOnly: true,
			Secure:   nurHTTPS,
			SameSite: http.SameSiteStrictMode, // keine Anfragen von fremden Seiten
		})

		rechte, err := ladeRechteDerRolle(ctx, pool, konto.rolle)
		if err != nil {
			apierrors.AntworteFehler(w, http.StatusInternalServerError, errors.New("berechtigungen konnten nicht geladen werden"))
			return
		}

		w.Header().Set(headerContentType, contentTypeJSON)
		httpresp.SchreibeJSON(w, anmeldeAntwort{
			KontoID:  konto.id,
			Email:    konto.email,
			Rolle:    rolle,
			Vorname:  konto.vorname,
			Nachname: konto.nachname,
			Rechte:   rechte,
		})
	}
}

// anmeldeKonto sind die Angaben aus der Tabelle benutzer, die die Anmeldung braucht.
type anmeldeKonto struct {
	id       string
	email    string
	rolle    string
	vorname  string
	nachname string
	aktiv    bool
	// neuAngelegt: Die Zeile ist gerade durch die Selbstanmeldung entstanden (selbstanmeldung.go)
	// und deshalb inaktiv. Die Person liest „Zugang beantragt" statt „Konto deaktiviert" und
	// probiert nicht immer wieder.
	neuAngelegt bool
	// beantragt: zugang_beantragt_am ist gesetzt, die Selbstanmeldung wartet auf die
	// Freischaltung. Die Freischaltung räumt die Spalte; ein danach deaktiviertes Konto ist kein
	// Antrag mehr und bekommt „Konto deaktiviert".
	beantragt bool
}

// pruefeAnmeldedaten verlangt E-Mail und Passwort. Bei false ist die Fehlerantwort schon
// geschrieben.
func pruefeAnmeldedaten(w http.ResponseWriter, anfrage anmeldeAnfrage) (passwort string, ok bool) {
	if anfrage.Email == "" {
		apierrors.AntworteFehlerMitMeldung(w, http.StatusBadRequest, meldungEmailFehlt, nil)
		return "", false
	}
	if anfrage.Passwort == "" {
		apierrors.AntworteFehlerMitMeldung(w, http.StatusBadRequest, meldungPasswortFehlt, nil)
		return "", false
	}
	return anfrage.Passwort, true
}

// pruefeFehlversuche bildet den Schlüssel der Bremse und weist ab, solange er gesperrt ist. Bei
// false ist die Fehlerantwort schon geschrieben.
//
// Gebremst wird je E-Mail und Adresse, nicht je Adresse allein: In einer Schule, deren Geräte
// alle hinter einer Adresse (NAT) hängen, sperrte sonst ein einziger Mensch mit fünf
// Fehlversuchen die ganze Schule für 15 Minuten aus.
func (a *Anmeldung) pruefeFehlversuche(w http.ResponseWriter, email, adresse string) (schluessel string, ok bool) {
	schluessel = strings.ToLower(strings.TrimSpace(email)) + "|" + adresse
	if a.bremse.istGesperrt(schluessel) {
		apierrors.AntworteFehlerMitMeldung(w, http.StatusTooManyRequests, MeldungZuVieleFehlversuche, nil)
		return "", false
	}
	return schluessel, true
}

// meldeAn prüft Zugangsdaten und Stand des Kontos. Bei false ist die Fehlerantwort schon
// geschrieben.
//
// Ein Fehlversuch zählt nur beim falschen Passwort. Ein Ausfall von Mailserver oder Datenbank
// und ein offener Antrag sind keine Fehlversuche: Wer dort mit dem richtigen Passwort
// weiterprobiert, darf sich nicht selbst aussperren.
func (a *Anmeldung) meldeAn(ctx context.Context, w http.ResponseWriter, pool db.Pool, email, passwort, schluessel string) (anmeldeKonto, bool) {
	konto, err := a.pruefeIMAPZugang(ctx, pool, email, passwort)
	if err != nil {
		// Ein Ausfall des Mailservers ist kein falsches Passwort: 503 ohne Fehlversuch, sonst
		// sperrte sich aus, wer sein richtiges Passwort erneut probiert.
		if errors.Is(err, errMailserverNichtErreichbar) {
			apierrors.AntworteFehlerMitMeldung(w, http.StatusServiceUnavailable, meldungMailserverNichtErreichbar, err)
			return anmeldeKonto{}, false
		}
		// Ebenso ein Aussetzer der Datenbank. Der Grund steht im Log; hat der Browser die
		// Anfrage abgebrochen, ist es keiner (apierrors, Status 499).
		if errors.Is(err, errAnmeldedienstGestoert) {
			apierrors.AntworteFehlerMitMeldung(w, http.StatusServiceUnavailable, meldungAnmeldedienstGestoert, err)
			return anmeldeKonto{}, false
		}
		a.bremse.merkeFehlversuch(schluessel)
		apierrors.AntworteFehlerMitMeldung(w, http.StatusUnauthorized, meldungAnmeldungFalsch, nil)
		return anmeldeKonto{}, false
	}

	// Beides ist 403 und heißt „Zugangsdaten stimmen, Zugang trotzdem nicht"; die
	// Unterscheidung ist für den Menschen davor. „Konto deaktiviert" ließe eine Lehrkraft, die
	// sich gerade zum ersten Mal gemeldet hat, ratlos zurück und verleitete zum Wiederholen; sie
	// soll wissen, dass sie nur warten muss, bei jedem Versuch, solange der Antrag offen ist.
	//
	// Kein Fehlversuch: Das Passwort war richtig. Sonst sperrte sich jemand mit fünf Versuchen
	// selbst aus, während seine Freischaltung noch aussteht.
	if konto.neuAngelegt || (!konto.aktiv && konto.beantragt) {
		apierrors.AntworteFehlerMitMeldung(w, http.StatusForbidden,
			meldungZugangBeantragt, nil)
		return anmeldeKonto{}, false
	}
	if !konto.aktiv {
		apierrors.AntworteFehlerMitMeldung(w, http.StatusForbidden,
			meldungKontoDeaktiviert, nil)
		return anmeldeKonto{}, false
	}
	return konto, true
}

// pruefeIMAPZugang prüft die Zugangsdaten am Mailserver und lädt danach das Konto. Ein Fehler
// heißt, dass die Anmeldung nicht gilt. ErrMailserverNichtErreichbar geht unverändert weiter,
// damit die Tür den Ausfall vom falschen Passwort unterscheidet (503 statt 401 mit Fehlversuch).
func (a *Anmeldung) pruefeIMAPZugang(ctx context.Context, pool db.Pool, email, passwort string) (anmeldeKonto, error) {
	if err := a.pruefeIMAPAnmeldung(ctx, email, passwort); err != nil {
		return anmeldeKonto{}, err
	}

	konto, err := repository.LadeAnmeldeKonto(ctx, pool, email)
	if err != nil {
		// Nur „keine Zeile" ist der Fall der Selbstanmeldung. Jeder andere Fehler
		// (Verbindungsabriss, Pool erschöpft, Frist) ist ein Ausfall der Anmeldung und kein
		// Fehlversuch.
		if !errors.Is(err, pgx.ErrNoRows) {
			return anmeldeKonto{}, fmt.Errorf("%w: benutzer lesen: %w", errAnmeldedienstGestoert, err)
		}
		// Ohne Konto: Ist die Selbstanmeldung für die Domain freigegeben, entsteht eine inaktive
		// Anfrage, ein Eintrag, den die Bibliothek freischalten kann. Sonst gibt es keine
		// Anmeldung.
		neu, anlegeFehler := legeZugangsanfrageAn(ctx, pool, email)
		if anlegeFehler != nil {
			// „Nicht erlaubt" ist der gewöhnliche Fall (Domain nicht freigegeben) und braucht
			// keine Zeile im Log. Jeder andere Fehler steht dort, sonst endete er als nacktes 401.
			if !errors.Is(anlegeFehler, errSelbstanmeldungNichtErlaubt) {
				slog.Error("Selbstanmeldung: Zugangsanfrage konnte nicht angelegt werden", "fehler", anlegeFehler)
			}
			return anmeldeKonto{}, anlegeFehler
		}
		return neu, nil
	}
	return anmeldeKontoAus(konto), nil
}

// anmeldeKontoAus füllt die Angaben der Anmeldung aus dem gelesenen Konto: die eine Stelle für
// die Anmeldung und für die Selbstanmeldung, die auf ein vorhandenes Konto trifft.
func anmeldeKontoAus(k repository.AnmeldeKonto) anmeldeKonto {
	return anmeldeKonto{
		id:        k.ID,
		email:     k.Email,
		rolle:     k.Rolle,
		vorname:   k.Vorname,
		nachname:  k.Nachname,
		aktiv:     k.Aktiv,
		beantragt: k.ZugangBeantragt,
	}
}

// ladeRechteDerRolle lädt die Rechte der Rolle aus role_permissions, damit die Oberfläche genau
// zeigt, was der Admin in der Matrix freigeschaltet hat. Der Admin hat alle Rechte ("*"), wie
// in verlangeRecht.
func ladeRechteDerRolle(ctx context.Context, pool db.Pool, rolle string) ([]string, error) {
	if strings.EqualFold(rolle, string(RolleAdmin)) {
		return []string{"*"}, nil
	}

	return repository.LadeErlaubteRechteDerRolle(ctx, pool, rolle)
}

// leseSitzung liest das Cookie der Sitzung und prüft das Token, ohne auf die Sperre nach
// Inaktivität zu achten. Bei false ist die Fehlerantwort schon geschrieben.
func leseSitzung(w http.ResponseWriter, r *http.Request, anmeldung *Anmeldung) (*Claims, bool) {
	cookie, err := r.Cookie("session_token")
	if errors.Is(err, http.ErrNoCookie) {
		apierrors.AntworteFehlerMitMeldung(w, http.StatusUnauthorized, MeldungNichtAngemeldet, nil)
		return nil, false
	}
	if err != nil {
		apierrors.AntworteFehlerMitMeldung(w, http.StatusBadRequest, apierrors.MeldungAnfrageUnlesbar, err)
		return nil, false
	}

	claims, err := anmeldung.PruefeTokenTrotzSperre(r.Context(), cookie.Value)
	if errors.Is(err, ErrPruefungGestoert) {
		// Ein Aussetzer der Datenbank ist keine abgelaufene Sitzung: 503 statt 401, sonst meldet
		// die Oberfläche ab. Sie liest nur den Satz des Fehlerwerts; die Ursache („sperrliste:
		// connection reset by peer") bleibt im Log.
		apierrors.AntworteFehlerMitMeldung(w, http.StatusServiceUnavailable, MeldungPruefungGestoert, err)
		return nil, false
	}
	if err != nil {
		apierrors.AntworteFehlerMitMeldung(w, http.StatusUnauthorized, MeldungNichtAngemeldet, err)
		return nil, false
	}
	return claims, true
}

// ladeKontoAntwort liest Rolle, Stammdaten und Rechte des Kontos aus der Datenbank und nicht aus
// den Claims: Rolle oder Status können sich seit dem Ausstellen des Tokens geändert haben. Bei
// false ist die Fehlerantwort schon geschrieben.
func ladeKontoAntwort(w http.ResponseWriter, r *http.Request, pool db.Pool, kontoID string) (anmeldeAntwort, bool) {
	ctx := r.Context()

	konto, err := repository.LadeSitzungsKonto(ctx, pool, kontoID)
	// Ein fehlendes und ein deaktiviertes Konto sind beide keine aktive Sitzung. Ein Fehler der
	// Datenbank ist es nicht: 500 statt 401, sonst meldete die Oberfläche ab.
	keineSitzung := errors.Is(err, pgx.ErrNoRows) || (err == nil && !konto.Aktiv)
	switch {
	case keineSitzung:
		apierrors.AntworteFehlerMitMeldung(w, http.StatusUnauthorized, MeldungNichtAngemeldet, nil)
		return anmeldeAntwort{}, false
	case err != nil:
		apierrors.AntworteFehler(w, http.StatusInternalServerError, err)
		return anmeldeAntwort{}, false
	}

	rechte, err := ladeRechteDerRolle(ctx, pool, konto.Rolle)
	if err != nil {
		apierrors.AntworteFehler(w, http.StatusInternalServerError, errors.New("berechtigungen konnten nicht geladen werden"))
		return anmeldeAntwort{}, false
	}

	return anmeldeAntwort{
		KontoID:  kontoID,
		Email:    konto.Email,
		Rolle:    Rolle(konto.Rolle),
		Vorname:  konto.Vorname,
		Nachname: konto.Nachname,
		Rechte:   rechte,
	}, true
}

// AngemeldetesKonto liefert die Tür, die das Konto der aktuellen Sitzung nennt, mit derselben
// Antwort wie die Anmeldung. Die Oberfläche stellt damit beim Laden eine bestehende Sitzung
// wieder her, statt jedes Mal die Anmeldung zu zeigen.
//
// Eine gesperrte Anmeldung beantwortet sie mit 423, der E-Mail-Adresse und dem Grund der Sperre:
// Der Sperrbildschirm zeigt, wer angemeldet ist und warum er fragt, mehr nicht.
func AngemeldetesKonto(pool db.Pool, anmeldung *Anmeldung) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims, ok := leseSitzung(w, r, anmeldung)
		if !ok {
			return
		}

		grund, err := anmeldung.Sitzungen.grundDerSperre(r.Context(), claims.SitzungID)
		if err != nil {
			apierrors.AntworteFehlerMitMeldung(w, http.StatusServiceUnavailable, MeldungPruefungGestoert, err)
			return
		}

		antwort, ok := ladeKontoAntwort(w, r, pool, claims.KontoID)
		if !ok {
			return
		}

		w.Header().Set(headerContentType, contentTypeJSON)
		if grund != sperrGrundKeiner {
			meldung := MeldungSitzungGesperrt
			if grund == sperrGrundHoechstdauer {
				meldung = MeldungHoechstdauerErreicht
			}
			w.WriteHeader(http.StatusLocked)
			httpresp.SchreibeJSON(w, map[string]string{"error": meldung, "email": antwort.Email, "grund": string(grund)})
			return
		}
		httpresp.SchreibeJSON(w, antwort)
	}
}

// SitzungVerlaengern liefert die Tür, die eine gültige Sitzung still verlängert: Ist das Token
// gültig und nicht widerrufen, kommt ein neues mit frischer Laufzeit. Das alte kommt nicht auf
// die Sperrliste, weil gleichzeitige Anfragen es noch benutzen. So muss sich niemand mitten in
// einer Schicht neu anmelden.
//
// Erneuert wird nur ein Token mit einer Zeile in sitzungen; das neue trägt dieselbe Kennung, eine
// Sperre nach Inaktivität bleibt also bestehen. Ein Token ohne Zeile läuft aus: Es ließe sich
// sonst beliebig lange halten, ohne je sperrbar zu sein.
func SitzungVerlaengern(anmeldung *Anmeldung, nurHTTPS bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie("session_token")
		if err != nil {
			if errors.Is(err, http.ErrNoCookie) {
				apierrors.AntworteFehlerMitMeldung(w, http.StatusUnauthorized, MeldungNichtAngemeldet, nil)
				return
			}
			apierrors.AntworteFehlerMitMeldung(w, http.StatusBadRequest, apierrors.MeldungAnfrageUnlesbar, err)
			return
		}

		// Eine gesperrte Anmeldung wird verlängert wie jede andere: Sie läuft weiter, bis jemand
		// aufschließt.
		claims, err := anmeldung.PruefeTokenTrotzSperre(r.Context(), cookie.Value)
		if errors.Is(err, ErrPruefungGestoert) {
			// Ein Aussetzer der Datenbank ist keine abgelaufene Sitzung: 503 statt 401, sonst
			// meldet die Oberfläche ab. Die Ursache bleibt im Log.
			apierrors.AntworteFehlerMitMeldung(w, http.StatusServiceUnavailable, MeldungPruefungGestoert, err)
			return
		}
		if err != nil {
			apierrors.AntworteFehlerMitMeldung(w, http.StatusUnauthorized, MeldungNichtAngemeldet, err)
			return
		}
		erneuereSitzung(w, r, anmeldung, claims, nurHTTPS)
	}
}

// erneuereSitzung stellt für eine geprüfte Sitzung ein neues Token aus, wenn weniger als die
// halbe Laufzeit übrig ist und die Zeile in sitzungen sich verlängern lässt; sonst bleibt es beim
// alten.
func erneuereSitzung(w http.ResponseWriter, r *http.Request, anmeldung *Anmeldung, claims *Claims, nurHTTPS bool) {
	// Erst ab der halben Laufzeit, sonst entstünde bei häufigen Anfragen ein Token nach dem
	// anderen.
	if claims.ExpiresAt != nil {
		rest := time.Until(claims.ExpiresAt.Time)
		if rest > anmeldung.laufzeit/2 {
			w.Header().Set(headerContentType, contentTypeJSON)
			httpresp.SchreibeJSON(w, map[string]string{"status": "ok", "refresh": "skipped"})
			return
		}
	}

	// claims.Rolle ist hier schon die aktuelle Rolle aus der Datenbank: PruefeToken ersetzt die im
	// alten Token signierte. Schriebe die Verlängerung die alte fort, bliebe ein entzogenes Recht
	// unbegrenzt erhalten.
	verlaengert, err := anmeldung.Sitzungen.verlaengere(r.Context(), claims.SitzungID, time.Now().Add(anmeldung.laufzeit))
	if err != nil {
		apierrors.AntworteFehlerMitMeldung(w, http.StatusServiceUnavailable, MeldungPruefungGestoert, err)
		return
	}
	if !verlaengert {
		w.Header().Set(headerContentType, contentTypeJSON)
		httpresp.SchreibeJSON(w, map[string]string{"status": "ok", "refresh": "skipped"})
		return
	}

	neuesToken, err := anmeldung.ErzeugeToken(claims.KontoID, claims.Rolle, claims.SitzungID)
	if err != nil {
		apierrors.AntworteFehler(w, http.StatusInternalServerError, err)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "session_token",
		Value:    neuesToken,
		Path:     "/",
		Expires:  time.Now().Add(anmeldung.laufzeit),
		HttpOnly: true,
		Secure:   nurHTTPS,
		SameSite: http.SameSiteStrictMode,
	})

	w.Header().Set(headerContentType, contentTypeJSON)
	httpresp.SchreibeJSON(w, map[string]string{"status": "ok", "refresh": "renewed"})
}
