package api

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"bibliothek/apierrors"
	"bibliothek/inventur"
	"bibliothek/pkg/isbnutil"
	"bibliothek/repository"

	"github.com/jackc/pgx/v5"
)

// parseErscheinungsjahr parst ein plausibles Erscheinungsjahr (1001–2099) aus rohem Text.
func parseErscheinungsjahr(raw string) *int {
	if raw == "" {
		return nil
	}
	var j int
	if _, err := fmt.Sscanf(raw, "%d", &j); err == nil && j > 1000 && j < 2100 {
		return &j
	}
	return nil
}

// findeLokalenTitel sucht einen Titel im lokalen Katalog anhand der (entstrichenen) ISBN.
// Rückgabe (nil, nil) bedeutet: nicht im Katalog vorhanden.
//
// Liefert die vorhandene Signatur mit — die Bestellung übernimmt sie unverändert und
// fragt dafür nie erneut DNB/eine Kategorisierung ab (siehe upsertTitelAusMetadaten,
// die dieser Funktion vorgeschaltet ist und nur bei NICHT gefundenem Titel läuft).
func (s *Server) findeLokalenTitel(ctx context.Context, isbn string) (*ISBNLookupResponse, error) {
	resp := ISBNLookupResponse{ISBN: isbn}
	err := s.DB.Pool.QueryRow(ctx, `
		SELECT id, titel, coalesce(autor,''), coalesce(verlag,''), coalesce(cover_url,''), coalesce(signatur,''), ist_lernmittel
		FROM buecher_titel WHERE isbn_normalform(isbn) = isbn_normalform($1) LIMIT 1
	`, isbn).Scan(&resp.TitelID, &resp.Titel, &resp.Autor, &resp.Verlag, &resp.CoverURL, &resp.Signatur, &resp.IstLernmittel)
	if err == nil {
		resp.Exists = true
		return &resp, nil
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return nil, err
}

// findeTitelUnterAndererForm sucht den Titel, der dieselbe ISBN in der anderen Länge trägt
// (isbnutil.AndereForm: ISBN-10 ↔ ISBN-13 mit 978). Rückgabe (nil, nil): Es gibt keine andere
// Form, oder unter ihr steht nichts im Katalog.
func (s *Server) findeTitelUnterAndererForm(ctx context.Context, isbn string) (*ISBNLookupResponse, error) {
	andere := isbnutil.AndereForm(isbn)
	if andere == "" {
		return nil, nil
	}
	return s.findeLokalenTitel(ctx, andere)
}

func (s *Server) upsertTitelAusMetadaten(ctx context.Context, isbn string, meta *inventur.MetadatenErgebnis) (ISBNLookupResponse, error) {
	jahrInt := parseErscheinungsjahr(meta.Jahr)

	// subject ist FK auf die Systematik (Migration 078): das Fach aus der
	// Titel-Heuristik erst registrieren, die kanonische Schreibweise schreiben.
	kanonisch, err := inventur.StelleFaecherSicher(ctx, s.DB.Pool, []string{meta.Fach})
	if err != nil {
		return ISBNLookupResponse{}, err
	}

	// Untertitel und Ladenpreis gehen seit dem 23.09.2026 mit (OFFEN.md 5.5). Der Preis nach
	// derselben Regel wie beim Anlegen über das Buchformular: 0 heißt „nicht ermittelbar"
	// und füllt nichts. Steht der Titel schon da (ON CONFLICT), gewinnt, was erfasst ist.
	// Die Altersangabe (Zielgruppe) bleibt ungespeichert: Es gibt für sie keine Spalte und
	// keinen Leser.
	listenpreis := inventur.ListenpreisAusNachschlagen(nil, meta.Preis)
	resp := ISBNLookupResponse{ISBN: isbn}
	err = s.DB.Pool.QueryRow(ctx, `
		INSERT INTO buecher_titel (titel, autor, isbn, verlag, erscheinungsjahr, cover_url, subject, untertitel, listenpreis)
		VALUES ($1, $2, $3, $4, $5, $6, NULLIF($7, ''), NULLIF(btrim($8), ''), $9)
		ON CONFLICT (isbn) DO UPDATE
			SET titel      = EXCLUDED.titel,
			    autor      = EXCLUDED.autor,
			    verlag     = EXCLUDED.verlag,
			    erscheinungsjahr = EXCLUDED.erscheinungsjahr,
			    cover_url  = COALESCE(NULLIF(EXCLUDED.cover_url, ''), buecher_titel.cover_url),
			    untertitel = COALESCE(NULLIF(buecher_titel.untertitel, ''), EXCLUDED.untertitel),
			    listenpreis = COALESCE(buecher_titel.listenpreis, EXCLUDED.listenpreis),
			    aktualisiert_am = CURRENT_TIMESTAMP
		RETURNING id, titel, coalesce(autor,''), coalesce(verlag,''), coalesce(cover_url,''), coalesce(signatur,''), ist_lernmittel
	`, meta.Titel, meta.Autor, isbn, meta.Verlag, jahrInt, meta.CoverURL, kanonisch[meta.Fach],
		meta.Untertitel, listenpreis).
		Scan(&resp.TitelID, &resp.Titel, &resp.Autor, &resp.Verlag, &resp.CoverURL, &resp.Signatur, &resp.IstLernmittel)
	if err != nil {
		return ISBNLookupResponse{}, err
	}
	resp.Exists = false
	return resp, nil
}

// ISBNLookupResponse is the result of a live ISBN metadata query.
// exists=true means the title is already in the catalog and has a stable titel_id.
type ISBNLookupResponse struct {
	Exists   bool   `json:"exists"`
	TitelID  string `json:"titel_id"`
	Titel    string `json:"titel"`
	Autor    string `json:"autor"`
	ISBN     string `json:"isbn"`
	Verlag   string `json:"verlag,omitempty"`
	CoverURL string `json:"cover_url,omitempty"`
	// Signatur ist bei exists=true die BEREITS VORHANDENE Regalsignatur (unverändert
	// übernommen) und bei exists=false leer: Die Signaturen der Schülerbücherei sind die
	// Littera-Codes am Regal („Sk", „JF", „MANGA"), und die kennt nur der Bestand — bis zum
	// 22.09.2026 schlug die DNB-Kategorie hier „BIB Jugendbuch" vor, ein Wort, das in keinem
	// Regal steht. Eingetragen wird sie im Bestellkorb, mit der Vorschlagsliste aus dem
	// Bestand (GET /api/signaturen); siehe PUT /api/buecher/titel/{id}/signatur.
	Signatur string `json:"signatur,omitempty"`
	// IstLernmittel: bei exists=true das Kennzeichen des vorhandenen Titels, bei
	// exists=false immer false — ein über die DNB neu angelegter Titel ist bis zur
	// Rückfrage im Staging-Fenster kein Lernmittel (PUT /api/buecher/titel/{id}/lernmittel).
	// Bis zum 10.09.2026 gab es diese Rückfrage nicht: Jedes neue Schulbuch entstand als
	// Bücherei-Titel und blieb es (Frist, Katalog, Löschfrist, Bestellbedarf lesen die Spalte).
	IstLernmittel bool `json:"ist_lernmittel"`
	// SchlagwortVorschlaege: nur bei exists=false — die Wörter der eigenen Liste, die der
	// DNB-Satz als Gattung oder Verlagswort nennt (repository.SchlagworteAusStichwoertern,
	// docs/OFFEN.md 4.20). Ein Vorschlag, kein Eintrag: Am Titel steht davon nichts, bis im
	// Bestellkorb jemand einen übernimmt.
	SchlagwortVorschlaege []string `json:"schlagwort_vorschlaege,omitempty"`
	// AndereForm: Unter dieser Schreibweise steht die ISBN nicht im Katalog, wohl aber in der
	// anderen Länge (ISBN-10 ↔ ISBN-13, docs/OFFEN.md 4.18 Stufe 4 und 5.5). Dann ist NICHTS
	// angelegt, titel_id ist leer, und hier steht der gefundene Titel. Die Oberfläche fragt
	// „Diesen Titel nehmen" oder „Neu anlegen"; Neu anlegen schickt dieselbe ISBN mit
	// neu_anlegen. Vorgeschlagen statt still übernommen: Am Testserver führt die Rechnung von
	// einer ISBN-10 mit falschem Prüfzeichen auf die ISBN-13 eines anderen Buchs.
	AndereForm *ISBNLookupResponse `json:"andere_form,omitempty"`
}

// titelAusNachschlagen legt den Titel aus einem Treffer der Katalogdienste an und gibt den
// Schlagwort-Vorschlag mit. Der Vorschlag wird VOR dem Anlegen gelesen: Scheitert die
// Abfrage, entsteht kein Titel — sonst stünde ein angelegter Titel hinter einer
// Fehlermeldung, und der zweite Versuch fände ihn als vorhanden, ohne Vorschlag.
func (s *Server) titelAusNachschlagen(ctx context.Context, isbn string, meta *inventur.MetadatenErgebnis) (ISBNLookupResponse, error) {
	vorschlaege, err := repository.SchlagworteAusStichwoertern(ctx, s.DB.Pool, meta.Stichwoerter)
	if err != nil {
		return ISBNLookupResponse{}, err
	}
	resp, err := s.upsertTitelAusMetadaten(ctx, isbn, meta)
	if err != nil {
		return ISBNLookupResponse{}, err
	}
	resp.SchlagwortVorschlaege = vorschlaege
	return resp, nil
}

// ISBNZuTitelHandler handles POST /api/buecher/aus-isbn.
// It receives an ISBN, checks the local catalog, and—if the title is not
// yet catalogued—fetches metadata from DNB / Google Books / OpenLibrary and
// creates a new buecher_titel record. The response contains a titel_id that the order
// workspace can add to the cart immediately — außer, der Katalog hat dieselbe ISBN in der
// anderen Länge: Dann trägt sie nur andere_form, und angelegt ist nichts.
func (s *Server) ISBNZuTitelHandler() http.HandlerFunc {
	return s.isbnZuTitel(inventur.NeuerMetadatenClient())
}

// isbnZuTitel ist die Tür mit ihrem Client für die Katalogdienste als Parameter: Ein Test
// stellt die DNB-Antwort nach (MetadatenClient.SetzeHTTPClientFuerTest), statt sie im Netz
// abzufragen — so ist auch geprüft, dass die Tür den Schlagwort-Vorschlag weitergibt.
func (s *Server) isbnZuTitel(metaClient *inventur.MetadatenClient) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			ISBN string `json:"isbn"`
			// NeuAnlegen: Die Antwort hat einen Titel unter der anderen Form vorgeschlagen, und
			// der Besteller hat „Neu anlegen" gewählt — dann wird nach ihr nicht mehr gesucht.
			NeuAnlegen bool `json:"neu_anlegen"`
		}
		if !DecodeAndValidate(w, r, &req) {
			return
		}
		// Normalise: strip dashes and spaces
		req.ISBN = strings.TrimSpace(strings.NewReplacer("-", "", " ", "").Replace(req.ISBN))
		if req.ISBN == "" {
			apierrors.SendHTTPError(w, http.StatusBadRequest, errors.New("isbn fehlt"))
			return
		}

		ctx := r.Context()

		// 1. Check whether the title is already in the local catalog.
		lokal, err := s.findeLokalenTitel(ctx, req.ISBN)
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}
		if lokal != nil {
			RespondJSON(w, http.StatusOK, *lokal)
			return
		}
		if !req.NeuAnlegen {
			andere, err := s.findeTitelUnterAndererForm(ctx, req.ISBN)
			if err != nil {
				apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
				return
			}
			if andere != nil {
				RespondJSON(w, http.StatusOK, ISBNLookupResponse{ISBN: req.ISBN, AndereForm: andere})
				return
			}
		}

		// 2. Not yet in catalog – fetch metadata from DNB / Google / OpenLibrary.
		meta, err := metaClient.SucheNachISBN(ctx, req.ISBN)
		if err != nil {
			if errors.Is(err, inventur.ErrKatalogdiensteNichtErreichbar) {
				// Netzausfall ist kein Nicht-Treffer: Bei 404 legt die Theke während
				// einer WLAN-Störung Titel von Hand an, die längst in der DNB stehen.
				apierrors.SendHTTPError(w, http.StatusBadGateway, errors.New("katalogdienste (DNB, Google, OpenLibrary) nicht erreichbar — bitte später erneut versuchen"))
				return
			}
			apierrors.SendHTTPError(w, http.StatusNotFound, fmt.Errorf("keine Metadaten für ISBN %s gefunden", req.ISBN))
			return
		}

		// 3. Insert new title; use ON CONFLICT as safety net for concurrent inserts.
		resp, err := s.titelAusNachschlagen(ctx, req.ISBN, meta)
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}

		RespondJSON(w, http.StatusOK, resp)
	}
}
