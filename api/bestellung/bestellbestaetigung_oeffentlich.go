package bestellung

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"bibliothek/api/tuer"
	"bibliothek/apierrors"
	"bibliothek/internal/buchetikett"
	"bibliothek/pdf"
	"bibliothek/pkg/bestelllink"
	"bibliothek/pkg/mitteltopf"
	"bibliothek/repository"

	"github.com/jackc/pgx/v5"
)

// Diese Datei bedient die einzige Seite des Systems, die ohne Anmeldung etwas verändern
// darf: die Bestätigungsseite des Lieferanten. Der Token aus dem Link ist der Ausweis.
//
// Der Zuschnitt ist bewusst eng — wer den Link hat, erreicht genau eine Bestellung und
// darf genau zwei Dinge: deren Etiketten drucken und sie einmal bestätigen. Keine
// Schülerdaten, keine Preise, keine anderen Bestellungen, kein Schreibzugriff auf den
// Bestand. Die Preise fehlen bewusst: Der Lieferant kennt seine eigenen, und was er
// nicht braucht, gehört nicht in eine Seite ohne Login.

// oeffentlichePosition ist eine Bestellzeile, wie der Lieferant sie sehen darf.
type oeffentlichePosition struct {
	TitelName string `json:"titel_name"`
	ISBN      string `json:"isbn"`
	Menge     int    `json:"menge"`
}

// oeffentlicheBestellung ist die Ansicht hinter dem Link.
type oeffentlicheBestellung struct {
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
	Positionen      []oeffentlichePosition `json:"positionen"`
	// EtikettenVorhanden: Standen Positionen mit Vorab-Barcode in dieser Bestellung?
	// Ohne sie zeigt die Seite keine Druckknöpfe, statt leere PDFs anzubieten.
	EtikettenVorhanden bool `json:"etiketten_vorhanden"`
	// GrossesEtikett: Gehört zu dieser Bestellung das große Lernmittel-Etikett? Bei einer
	// Bestellung für die Schülerbücherei nicht — die Tür liefert es dann auch nicht.
	GrossesEtikett bool `json:"grosses_etikett"`
	// EtikettenFormate sind die Bogenraster, unter denen der Lieferant für die kleinen
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
// Unbekannt und abgelaufen sind bewusst derselbe Fall (pgx.ErrNoRows): Ein Unterschied
// in der Antwort verriete, dass ein geratener Token einmal echt war.
func (t *Tueren) bestellungPerToken(ctx context.Context, token string) (string, error) {
	if token == "" {
		return "", pgx.ErrNoRows
	}
	return repository.LadeBestellungZuTokenHash(ctx, t.DB.Pool, bestelllink.Hash(token))
}

// sendeTokenFehler bildet jeden Zugriffsfehler auf 404 ab — abgelaufen, zurückgezogen
// und nie existiert sehen von außen gleich aus.
func sendeTokenFehler(w http.ResponseWriter, err error) {
	if errors.Is(err, pgx.ErrNoRows) {
		apierrors.AntworteFehler(w, http.StatusNotFound, errors.New("dieser Link ist nicht (mehr) gültig"))
		return
	}
	apierrors.AntworteFehler(w, http.StatusInternalServerError, err)
}

// OeffentlicheBestellung liefert die Bestellung hinter dem Link (GET, ohne Login).
func (t *Tueren) OeffentlicheBestellung() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		bestellungID, err := t.bestellungPerToken(ctx, r.PathValue("token"))
		if err != nil {
			sendeTokenFehler(w, err)
			return
		}

		ansicht, err := t.ladeOeffentlicheBestellung(ctx, bestellungID)
		if err != nil {
			apierrors.AntworteFehler(w, http.StatusInternalServerError, err)
			return
		}
		tuer.AntworteJSON(w, http.StatusOK, ansicht)
	}
}

// ladeOeffentlicheBestellung baut die Ansicht aus Kopf, Positionen und Schulname.
func (t *Tueren) ladeOeffentlicheBestellung(ctx context.Context, bestellungID string) (*oeffentlicheBestellung, error) {
	kopf, err := repository.LadeOeffentlichenBestellkopf(ctx, t.DB.Pool, bestellungID)
	if err != nil {
		return nil, err
	}
	a := oeffentlicheBestellung{
		LieferantName:      kopf.LieferantName,
		Kundennummer:       kopf.Kundennummer,
		Bestelldatum:       kopf.Bestelldatum,
		AnzahlExemplare:    kopf.AnzahlExemplare,
		BestaetigtAm:       kopf.BestaetigtAm,
		LinkGueltigBis:     kopf.LinkGueltigBis,
		EtikettenVorhanden: kopf.EtikettenVorhanden,
	}
	if texte, err := mitteltopf.TexteFuer(kopf.Mittel); err == nil {
		a.Mittel = texte.Kurz
	}
	a.GrossesEtikett = mitteltopf.HatGrossesLernmittelEtikett(kopf.Mittel)

	positionen, err := repository.LadeOeffentlichePositionen(ctx, t.DB.Pool, bestellungID)
	if err != nil {
		return nil, err
	}
	a.Positionen = make([]oeffentlichePosition, 0, len(positionen))
	for _, pos := range positionen {
		a.Positionen = append(a.Positionen, oeffentlichePosition(pos))
	}

	// Der Schulname sagt dem Lieferanten, wessen Bestellung er vor sich hat — bei
	// mehreren Schulen im selben Postfach ist das keine Zierde, sondern die Zuordnung.
	a.SchuleName = buchetikett.LadeKopf(ctx, t.DB.Pool).Schulname
	a.SchuleAnschrift = t.schulAnschrift(ctx)
	a.EtikettenFormate = pdf.LabelFormatAuswahl()
	a.EtikettenFormatVorgabe = pdf.StandardLabelFormat
	return &a, nil
}

// oeffentlichBestaetigenAnfrage ist der Rumpf des Bestätigens durch den Lieferanten.
type oeffentlichBestaetigenAnfrage struct {
	// EtikettenGroesse ist wahlweise ('klein'/'gross'/''): Die Schule braucht die Angabe
	// nicht, der Lieferant druckt selbst. Die Seite schickt mit, was sie geladen hat —
	// mehr als eine Notiz in der Historie ist es nicht, und ein leerer Wert ist erlaubt.
	EtikettenGroesse string `json:"etiketten_groesse"`
	// EtikettenFormat ist ebenfalls wahlweise und nur bei 'klein' aussagekräftig: auf
	// welchem Bogenraster der Lieferant gedruckt hat. Für die Bibliothek ist das die
	// Antwort auf "wie sehen die Aufkleber aus, die gleich ankommen?".
	EtikettenFormat string `json:"etiketten_format"`
}

// OeffentlichBestaetigen nimmt die Bestätigung des Lieferanten entgegen.
func (t *Tueren) OeffentlichBestaetigen() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		bestellungID, err := t.bestellungPerToken(ctx, r.PathValue("token"))
		if err != nil {
			sendeTokenFehler(w, err)
			return
		}

		var req oeffentlichBestaetigenAnfrage
		if !tuer.LeseAnfrage(w, r, &req) {
			return
		}
		if req.EtikettenGroesse != "" && req.EtikettenGroesse != "klein" && req.EtikettenGroesse != "gross" {
			apierrors.AntworteFehler(w, http.StatusBadRequest, errors.New("etiketten_groesse muss 'klein' oder 'gross' sein"))
			return
		}
		if !pdf.IstBekanntesEtikettFormat(req.EtikettenFormat) {
			apierrors.AntworteFehler(w, http.StatusBadRequest, errors.New("unbekanntes etiketten_format"))
			return
		}

		bereits, err := t.bestaetigeBestellung(ctx, bestellungID, req.EtikettenGroesse, req.EtikettenFormat, "lieferant")
		if err != nil {
			apierrors.AntworteFehler(w, http.StatusInternalServerError, err)
			return
		}
		if bereits {
			apierrors.AntworteFehler(w, http.StatusConflict, errors.New("diese Bestellung ist bereits bestätigt"))
			return
		}

		tuer.AntworteJSON(w, http.StatusOK, map[string]any{"status": "success"})
	}
}

// schulAnschrift baut die einzeilige Anschrift aus den Systemeinstellungen. Leere Felder
// fallen samt Trenner weg — eine Zeile wie „, 61381" wäre schlechter als keine.
func (t *Tueren) schulAnschrift(ctx context.Context) string {
	einstellungen, err := repository.NewEinstellungsAbfragen(t.DB.Pool).Lade(ctx)
	if err != nil {
		return ""
	}
	teile := []string{}
	if strasse := strings.TrimSpace(einstellungen.SchuleStrasse); strasse != "" {
		teile = append(teile, strasse)
	}
	ortszeile := strings.TrimSpace(einstellungen.SchulePLZ + " " + einstellungen.SchuleOrt)
	if ortszeile != "" {
		teile = append(teile, ortszeile)
	}
	return strings.Join(teile, " · ")
}
