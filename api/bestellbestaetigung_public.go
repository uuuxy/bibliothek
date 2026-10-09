package api

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"bibliothek/apierrors"
	"bibliothek/pdf"
	"bibliothek/pkg/bestelllink"
	"bibliothek/pkg/mitteltopf"
	"bibliothek/repository"

	"github.com/jackc/pgx/v5"
)

// Diese Datei bedient die EINZIGE Seite des Systems, die ohne Anmeldung etwas verändern
// darf: die Bestätigungsseite des Lieferanten. Der Token aus dem Link ist der Ausweis.
//
// Der Zuschnitt ist bewusst eng — wer den Link hat, erreicht genau eine Bestellung und
// darf genau zwei Dinge: deren Etiketten drucken und sie einmal bestätigen. Keine
// Schülerdaten, keine Preise, keine anderen Bestellungen, kein Schreibzugriff auf den
// Bestand. Die Preise fehlen bewusst: Der Lieferant kennt seine eigenen, und was er
// nicht braucht, gehört nicht in eine Seite ohne Login.

// OeffentlichePosition ist eine Bestellzeile, wie der Lieferant sie sehen darf.
type OeffentlichePosition struct {
	TitelName string `json:"titel_name"`
	ISBN      string `json:"isbn"`
	Menge     int    `json:"menge"`
}

// OeffentlicheBestellung ist die Ansicht hinter dem Link.
type OeffentlicheBestellung struct {
	SchuleName string `json:"schule_name"`
	// SchuleAnschrift steht unter dem Namen. Für einen Empfänger, der diesen Link aus
	// einer Mail öffnet, ist die Absenderangabe der einzige Beleg, wessen Bestellung er
	// vor sich hat — die Seite verlangt ja bewusst keine Anmeldung.
	SchuleAnschrift string `json:"schule_anschrift"`
	LieferantName   string `json:"lieferant_name"`
	// Mittel nennt den Topf mit dem Wort des Anschreibens (pkg/mitteltopf): Der Händler
	// bekommt am selben Tag zwei gleich aussehende Links und muss sie auseinanderhalten.
	// Leer bei Alt-Bestellungen ohne Zuordnung — ihr Topf wird nicht geraten.
	Mittel          string                 `json:"mittel"`
	Kundennummer    string                 `json:"kundennummer"`
	Bestelldatum    time.Time              `json:"bestelldatum"`
	AnzahlExemplare int                    `json:"anzahl_exemplare"`
	Positionen      []OeffentlichePosition `json:"positionen"`
	// EtikettenVorhanden: Standen Positionen mit Vorab-Barcode in dieser Bestellung?
	// Ohne sie zeigt die Seite keine Druckknöpfe, statt leere PDFs anzubieten.
	EtikettenVorhanden bool `json:"etiketten_vorhanden"`
	// GrossesEtikett: Gehört zu dieser Bestellung das große Lernmittel-Etikett? Bei einer
	// Bestellung für die Schülerbücherei nicht — die Tür liefert es dann auch nicht.
	GrossesEtikett bool `json:"grosses_etikett"`
	// EtikettenFormate sind die Bogenraster, unter denen der Lieferant für die KLEINEN
	// Etiketten wählen kann — er druckt auf sein eigenes Material, und davon gibt es
	// verschiedene. Die Liste kommt aus dem Backend, damit ein neues Format nicht an
	// zwei Stellen nachgetragen werden muss.
	EtikettenFormate []pdf.EtikettFormatAuswahl `json:"etiketten_formate"`
	// EtikettenFormatVorgabe ist die Vorauswahl der Seite.
	EtikettenFormatVorgabe string `json:"etiketten_format_vorgabe"`
	// BestaetigtAm ist NULL, solange niemand bestätigt hat — die Seite entscheidet daran,
	// ob sie den Bestätigen-Knopf oder die Quittung zeigt.
	BestaetigtAm *time.Time `json:"bestaetigt_am,omitempty"`
	// LinkGueltigBis: bis wann diese Seite und ihre Etiketten erreichbar bleiben.
	LinkGueltigBis *time.Time `json:"link_gueltig_bis,omitempty"`
}

// bestellungPerToken übersetzt den Token aus dem Link in eine Bestell-ID.
//
// Unbekannt und abgelaufen sind bewusst DERSELBE Fall (pgx.ErrNoRows): Ein Unterschied
// in der Antwort verriete, dass ein geratener Token einmal echt war.
func (s *Server) bestellungPerToken(ctx context.Context, token string) (string, error) {
	if token == "" {
		return "", pgx.ErrNoRows
	}
	return repository.BestellungZuTokenHash(ctx, s.DB.Pool, bestelllink.Hash(token))
}

// sendeTokenFehler bildet jeden Zugriffsfehler auf 404 ab — abgelaufen, zurückgezogen
// und nie existiert sehen von außen gleich aus.
func sendeTokenFehler(w http.ResponseWriter, err error) {
	if errors.Is(err, pgx.ErrNoRows) {
		apierrors.SendHTTPError(w, http.StatusNotFound, errors.New("dieser Link ist nicht (mehr) gültig"))
		return
	}
	apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
}

// OeffentlicheBestellungHandler liefert die Bestellung hinter dem Link (GET, ohne Login).
func (s *Server) OeffentlicheBestellungHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		bestellungID, err := s.bestellungPerToken(ctx, r.PathValue("token"))
		if err != nil {
			sendeTokenFehler(w, err)
			return
		}

		ansicht, err := s.ladeOeffentlicheBestellung(ctx, bestellungID)
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}
		RespondJSON(w, http.StatusOK, ansicht)
	}
}

// ladeOeffentlicheBestellung baut die Ansicht aus Kopf, Positionen und Schulname.
func (s *Server) ladeOeffentlicheBestellung(ctx context.Context, bestellungID string) (*OeffentlicheBestellung, error) {
	kopf, err := repository.LadeOeffentlichenBestellkopf(ctx, s.DB.Pool, bestellungID)
	if err != nil {
		return nil, err
	}
	a := OeffentlicheBestellung{
		LieferantName:      kopf.LieferantName,
		Kundennummer:       kopf.Kundennummer,
		Bestelldatum:       kopf.Bestelldatum,
		AnzahlExemplare:    kopf.AnzahlExemplare,
		BestaetigtAm:       kopf.BestaetigtAm,
		LinkGueltigBis:     kopf.LinkGueltigBis,
		EtikettenVorhanden: kopf.EtikettenVorhanden,
	}
	if t, err := mitteltopf.TexteFuer(kopf.Mittel); err == nil {
		a.Mittel = t.Kurz
	}
	a.GrossesEtikett = mitteltopf.GrossesLernmittelEtikettFuer(kopf.Mittel)

	positionen, err := repository.ListeOeffentlichePositionen(ctx, s.DB.Pool, bestellungID)
	if err != nil {
		return nil, err
	}
	a.Positionen = make([]OeffentlichePosition, 0, len(positionen))
	for _, pos := range positionen {
		a.Positionen = append(a.Positionen, OeffentlichePosition(pos))
	}

	// Der Schulname sagt dem Lieferanten, wessen Bestellung er vor sich hat — bei
	// mehreren Schulen im selben Postfach ist das keine Zierde, sondern die Zuordnung.
	a.SchuleName = s.etikettKopf(ctx).Schulname
	a.SchuleAnschrift = s.schulAnschrift(ctx)
	a.EtikettenFormate = pdf.LabelFormatAuswahl()
	a.EtikettenFormatVorgabe = pdf.StandardLabelFormat
	return &a, nil
}

// OeffentlichBestaetigenRequest ist der Body des Bestätigens durch den Lieferanten.
type OeffentlichBestaetigenRequest struct {
	// EtikettenGroesse ist OPTIONAL ('klein'/'gross'/''): Die Schule braucht die Angabe
	// nicht, der Lieferant druckt selbst. Die Seite schickt mit, was sie geladen hat —
	// mehr als eine Notiz in der Historie ist es nicht, und ein leerer Wert ist erlaubt.
	EtikettenGroesse string `json:"etiketten_groesse"`
	// EtikettenFormat ist ebenfalls OPTIONAL und nur bei 'klein' aussagekräftig: auf
	// welchem Bogenraster der Lieferant gedruckt hat. Für die Bibliothek ist das die
	// Antwort auf "wie sehen die Aufkleber aus, die gleich ankommen?".
	EtikettenFormat string `json:"etiketten_format"`
}

// OeffentlichBestaetigenHandler nimmt die Bestätigung des Lieferanten entgegen.
func (s *Server) OeffentlichBestaetigenHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		bestellungID, err := s.bestellungPerToken(ctx, r.PathValue("token"))
		if err != nil {
			sendeTokenFehler(w, err)
			return
		}

		var req OeffentlichBestaetigenRequest
		if !DecodeAndValidate(w, r, &req) {
			return
		}
		if req.EtikettenGroesse != "" && req.EtikettenGroesse != "klein" && req.EtikettenGroesse != "gross" {
			apierrors.SendHTTPError(w, http.StatusBadRequest, errors.New("etiketten_groesse muss 'klein' oder 'gross' sein"))
			return
		}
		if !pdf.IstBekanntesEtikettFormat(req.EtikettenFormat) {
			apierrors.SendHTTPError(w, http.StatusBadRequest, errors.New("unbekanntes etiketten_format"))
			return
		}

		bereits, err := s.bestaetigeBestellung(ctx, bestellungID, req.EtikettenGroesse, req.EtikettenFormat, "lieferant")
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}
		if bereits {
			apierrors.SendHTTPError(w, http.StatusConflict, errors.New("diese Bestellung ist bereits bestätigt"))
			return
		}

		RespondJSON(w, http.StatusOK, map[string]any{"status": "success"})
	}
}

// schulAnschrift baut die einzeilige Anschrift aus den Systemeinstellungen. Leere Felder
// fallen samt Trenner weg — eine Zeile wie „, 61381" wäre schlechter als keine.
func (s *Server) schulAnschrift(ctx context.Context) string {
	settings, err := repository.NewSystemSettingsRepository(s.DB.Pool).GetSettings(ctx)
	if err != nil {
		return ""
	}
	teile := []string{}
	if strasse := strings.TrimSpace(settings.SchuleStrasse); strasse != "" {
		teile = append(teile, strasse)
	}
	ortszeile := strings.TrimSpace(settings.SchulePLZ + " " + settings.SchuleOrt)
	if ortszeile != "" {
		teile = append(teile, ortszeile)
	}
	return strings.Join(teile, " · ")
}
