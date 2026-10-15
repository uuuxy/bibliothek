package katalog

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"bibliothek/api/tuer"
	"bibliothek/apierrors"
	"bibliothek/inventur"
	"bibliothek/repository"

	"github.com/jackc/pgx/v5"
)

// leseErscheinungsjahr liest ein plausibles Erscheinungsjahr (1001 bis 2099) aus rohem Text.
func leseErscheinungsjahr(roh string) *int {
	if roh == "" {
		return nil
	}
	var j int
	if _, err := fmt.Sscanf(roh, "%d", &j); err == nil && j > 1000 && j < 2100 {
		return &j
	}
	return nil
}

// findeLokalenTitel sucht den Titel, der die ISBN trägt, in jeder Schreibweise und in beiden
// Längen. Die Rückgabe (nil, nil) heißt: nicht im Katalog.
//
// Die vorhandene Signatur kommt mit; die Bestellung übernimmt sie unverändert und fragt dafür
// weder die DNB noch eine Kategorie ab. schreibeTitelAusMetadaten läuft erst danach und nur,
// wenn kein Titel gefunden ist.
func (t *Tueren) findeLokalenTitel(ctx context.Context, isbn string) (*isbnZuTitelAntwort, error) {
	titel, err := repository.FindeTitelZuISBN(ctx, t.DB.Pool, isbn)
	if err == nil {
		antwort := isbnZuTitelAntwortAus(titel)
		antwort.Vorhanden = true
		return &antwort, nil
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return nil, err
}

func (t *Tueren) schreibeTitelAusMetadaten(ctx context.Context, isbn string, meta *inventur.MetadatenErgebnis) (isbnZuTitelAntwort, error) {
	jahr := leseErscheinungsjahr(meta.Jahr)

	// subject ist ein Fremdschlüssel auf die Systematik: das Fach, das die Katalogdienste
	// nahelegen, erst anlegen, dann seine kanonische Schreibweise schreiben.
	kanonisch, err := inventur.StelleFaecherSicher(ctx, t.DB.Pool, []string{meta.Fach})
	if err != nil {
		return isbnZuTitelAntwort{}, err
	}

	// Untertitel und Ladenpreis gehen mit, der Preis nach derselben Regel wie beim Anlegen über
	// das Buchformular: 0 heißt „nicht ermittelbar" und füllt nichts. Steht der Titel schon da
	// (ON CONFLICT), gewinnt, was erfasst ist. Die Altersangabe (Zielgruppe) bleibt ungespeichert:
	// Es gibt für sie keine Spalte und keinen Leser.
	listenpreis := inventur.ListenpreisAusNachschlagen(nil, meta.Preis)
	titel, err := repository.LegeTitelAusMetadatenAn(ctx, t.DB.Pool, repository.TitelMetadaten{
		Titel:            meta.Titel,
		Autor:            meta.Autor,
		ISBN:             isbn,
		Verlag:           meta.Verlag,
		Erscheinungsjahr: jahr,
		CoverURL:         meta.CoverURL,
		Fach:             kanonisch[meta.Fach],
		Untertitel:       meta.Untertitel,
		Listenpreis:      listenpreis,
	})
	if err != nil {
		return isbnZuTitelAntwort{}, err
	}
	antwort := isbnZuTitelAntwortAus(titel)
	antwort.Vorhanden = false
	return antwort, nil
}

// isbnZuTitelAntwortAus übernimmt die Angaben eines Titels aus dem Katalog in die Antwort.
func isbnZuTitelAntwortAus(titel repository.TitelZuISBN) isbnZuTitelAntwort {
	return isbnZuTitelAntwort{
		TitelID:       titel.TitelID,
		Titel:         titel.Titel,
		Autor:         titel.Autor,
		Verlag:        titel.Verlag,
		CoverURL:      titel.CoverURL,
		Signatur:      titel.Signatur,
		IstLernmittel: titel.IstLernmittel,
		ISBN:          titel.ISBN,
	}
}

// isbnZuTitelAntwort ist der Titel zu einer ISBN. exists=true heißt: Der Titel stand schon im
// Katalog; sonst ist er eben aus den Katalogdiensten angelegt. In beiden Fällen trägt titel_id
// die Kennung, die die Bestellung in den Warenkorb legt.
type isbnZuTitelAntwort struct {
	Vorhanden bool   `json:"exists"`
	TitelID   string `json:"titel_id"`
	Titel     string `json:"titel"`
	Autor     string `json:"autor"`
	// ISBN ist die Nummer, wie der Katalog sie trägt: Eine zehnstellig eingegebene ISBN
	// steht dort dreizehnstellig (isbn_normalform).
	ISBN     string `json:"isbn"`
	Verlag   string `json:"verlag,omitempty"`
	CoverURL string `json:"cover_url,omitempty"`
	// Signatur ist bei exists=true die vorhandene Regalsignatur, unverändert übernommen, und bei
	// exists=false leer: Die Signaturen der Schülerbücherei sind die Littera-Codes am Regal („Sk",
	// „JF", „MANGA"), und die kennt nur der Bestand; eine Kategorie der DNB wie „BIB Jugendbuch"
	// steht in keinem Regal. Eingetragen wird sie im Bestellkorb, mit der Vorschlagsliste aus dem
	// Bestand (GET /api/signaturen, PUT /api/buecher/titel/{id}/signatur).
	Signatur string `json:"signatur,omitempty"`
	// IstLernmittel: bei exists=true das Kennzeichen des vorhandenen Titels, bei exists=false
	// immer false. Ein über die DNB neu angelegter Titel ist kein Lernmittel, bis das Fenster der
	// Bestellsuche nachfragt (PUT /api/buecher/titel/{id}/lernmittel); Frist, Katalog, Löschfrist
	// und Bestellbedarf lesen die Spalte.
	IstLernmittel bool `json:"ist_lernmittel"`
	// SchlagwortVorschlaege: nur bei exists=false die Wörter der eigenen Liste, die der Satz der
	// DNB als Gattung, Verlagswort oder Normdatei-Schlagwort nennt
	// (repository.LadeSchlagwortVorschlagAusDNB). Ein Vorschlag, kein Eintrag: Am Titel steht
	// davon nichts, bis im Bestellkorb jemand einen übernimmt. Für einen Titel, den es schon gibt,
	// fragt das Bestellfenster auf Knopfdruck GET /api/schlagworte/dnb-vorschlag, nach derselben
	// Regel.
	SchlagwortVorschlaege []string `json:"schlagwort_vorschlaege,omitempty"`
	// SchlagwortVorschlaegeNeu: Normdatei-Schlagwörter des Satzes, die die eigene Liste noch
	// nicht kennt. Angeboten werden sie mit dem Zusatz „neu" und nie vorbelegt; wer eines
	// anklickt, legt es mit dem Speichern in der Liste an.
	SchlagwortVorschlaegeNeu []string `json:"schlagwort_vorschlaege_neu,omitempty"`
}

// legeTitelAusTrefferAn legt den Titel aus einem Treffer der Katalogdienste an und gibt den
// Schlagwort-Vorschlag mit. Der Vorschlag wird vor dem Anlegen gelesen: Scheitert die Abfrage,
// entsteht kein Titel; sonst stünde ein angelegter Titel hinter einer Fehlermeldung, und der
// zweite Versuch fände ihn als vorhanden, ohne Vorschlag.
func (t *Tueren) legeTitelAusTrefferAn(ctx context.Context, isbn string, meta *inventur.MetadatenErgebnis) (isbnZuTitelAntwort, error) {
	liste, neu, err := repository.LadeSchlagwortVorschlagAusDNB(ctx, t.DB.Pool, meta.Stichwoerter, meta.Normdaten)
	if err != nil {
		return isbnZuTitelAntwort{}, err
	}
	antwort, err := t.schreibeTitelAusMetadaten(ctx, isbn, meta)
	if err != nil {
		return isbnZuTitelAntwort{}, err
	}
	antwort.SchlagwortVorschlaege = liste
	antwort.SchlagwortVorschlaegeNeu = neu
	return antwort, nil
}

// ISBNZuTitel nimmt eine ISBN und liefert den Titel dazu (POST /api/buecher/aus-isbn): aus dem
// Katalog, wenn es ihn gibt, sonst neu angelegt aus den Metadaten von DNB, Google Books oder
// OpenLibrary. Die Antwort trägt die titel_id, die die Bestellung sofort in den Warenkorb legen
// kann. Die zehn- und die dreizehnstellige Form einer ISBN sind dabei dieselbe Nummer: Der
// Katalog führt sie dreizehnstellig.
func (t *Tueren) ISBNZuTitel() http.HandlerFunc {
	return t.isbnZuTitelTuer(inventur.NeuerMetadatenClient())
}

// isbnZuTitelTuer ist die Tür mit den Katalogdiensten als Parameter: Ein Test stellt die
// Antwort der DNB nach (MetadatenClient.SetzeHTTPClientFuerTest), statt sie im Netz
// abzufragen; so ist auch geprüft, dass die Tür den Schlagwort-Vorschlag weitergibt.
func (t *Tueren) isbnZuTitelTuer(metadaten *inventur.MetadatenClient) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			ISBN string `json:"isbn"`
		}
		if !tuer.LeseAnfrage(w, r, &req) {
			return
		}
		// Bindestriche und Leerzeichen entfernen.
		req.ISBN = strings.TrimSpace(strings.NewReplacer("-", "", " ", "").Replace(req.ISBN))
		if req.ISBN == "" {
			apierrors.AntworteFehler(w, http.StatusBadRequest, errors.New("isbn fehlt"))
			return
		}

		ctx := r.Context()

		// 1. Steht der Titel schon im Katalog?
		lokal, err := t.findeLokalenTitel(ctx, req.ISBN)
		if err != nil {
			apierrors.AntworteFehler(w, http.StatusInternalServerError, err)
			return
		}
		if lokal != nil {
			tuer.AntworteJSON(w, http.StatusOK, *lokal)
			return
		}

		// 2. Noch nicht im Katalog: die Metadaten bei DNB, Google Books und OpenLibrary holen.
		meta, err := metadaten.SucheNachISBN(ctx, req.ISBN)
		if err != nil {
			if errors.Is(err, inventur.ErrKatalogdiensteNichtErreichbar) {
				// Netzausfall ist kein Nicht-Treffer: Bei 404 legt die Theke während
				// einer WLAN-Störung Titel von Hand an, die längst in der DNB stehen.
				apierrors.AntworteFehler(w, http.StatusBadGateway, errors.New("katalogdienste (DNB, Google, OpenLibrary) nicht erreichbar — bitte später erneut versuchen"))
				return
			}
			apierrors.AntworteFehler(w, http.StatusNotFound, fmt.Errorf("keine Metadaten für ISBN %s gefunden", req.ISBN))
			return
		}

		// 3. Den Titel anlegen; ON CONFLICT fängt zwei gleichzeitige Anfragen auf.
		antwort, err := t.legeTitelAusTrefferAn(ctx, req.ISBN, meta)
		if err != nil {
			apierrors.AntworteFehler(w, http.StatusInternalServerError, err)
			return
		}

		tuer.AntworteJSON(w, http.StatusOK, antwort)
	}
}
