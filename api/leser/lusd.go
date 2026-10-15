package leser

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"

	"bibliothek/api/tuer"
	"bibliothek/apierrors"
	"bibliothek/auth"
	"bibliothek/internal/lusd"
	"bibliothek/pkg/clientip"
	"bibliothek/pkg/closeutil"
	"bibliothek/repository"
)

// abgaengerKarenzTage liest die Karenzzeit aus den Einstellungen; ohne lesbare
// Einstellungen gilt die Vorgabe — derselbe Rückfall wie im nächtlichen Job.
func (t *Tueren) abgaengerKarenzTage(ctx context.Context) int {
	einst, err := repository.NewEinstellungsAbfragen(t.DB.Pool).Lade(ctx)
	if err != nil {
		return repository.AbgaengerKarenzTageVorgabe
	}
	return repository.AbgaengerKarenzTageOderVorgabe(einst)
}

// leseLUSDDatei liest die hochgeladene Datei und übergibt sie dem Parser des Pakets lusd.
func leseLUSDDatei(r *http.Request) (lusd.Datei, error) {
	datei, _, err := r.FormFile("csvFile")
	if err != nil {
		return lusd.Datei{}, fmt.Errorf("CSV-Datei fehlt: %w", err)
	}
	defer closeutil.Schliesse(datei, "lusd upload")

	inhalt, err := io.ReadAll(datei)
	if err != nil {
		return lusd.Datei{}, fmt.Errorf("CSV konnte nicht gelesen werden: %w", err)
	}
	return lusd.LeseDatei(inhalt)
}

// lusdTuer bündelt, was Vorschau und Import gemeinsam haben: Grenze der hochgeladenen Datei,
// Lesen, Zuordnung der Fehler. uebernehmen unterscheidet die beiden Routen.
func (t *Tueren) lusdTuer(uebernehmen bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) { t.verarbeiteLUSDDatei(w, r, uebernehmen) }
}

// verarbeiteLUSDDatei liest die Datei, fährt den Lauf und antwortet mit Vorschau oder Ergebnis.
func (t *Tueren) verarbeiteLUSDDatei(w http.ResponseWriter, r *http.Request, uebernehmen bool) {
	r.Body = http.MaxBytesReader(w, r.Body, 10<<20)
	if err := r.ParseMultipartForm(10 << 20); err != nil {
		apierrors.AntworteFehler(w, http.StatusBadRequest, err)
		return
	}
	datei, err := leseLUSDDatei(r)
	if err != nil {
		apierrors.AntworteFehler(w, http.StatusBadRequest, err)
		return
	}
	lauf := lusd.Lauf{Anwenden: uebernehmen, MassenabgangBestaetigt: uebernehmen && r.FormValue("confirm_graduates") == "true"}
	if uebernehmen {
		if lauf.Umbenennungen, err = leseUmbenennungsWahl(r.FormValue("umbenennungen")); err != nil {
			apierrors.AntworteFehler(w, http.StatusBadRequest, err)
			return
		}
	}
	// Die Karenzzeit vor der Transaktion des Laufs lesen: eine Einstellung, kein Teil des
	// Bestands, den der Lauf vergleicht.
	lauf.KarenzTage = t.abgaengerKarenzTage(r.Context())
	res, err := lusd.FuehreAus(r.Context(), t.DB.Pool, datei, lauf)
	if err != nil {
		apierrors.AntworteFehler(w, lusdFehlerStatus(err), err)
		return
	}
	if uebernehmen {
		// Der Import ist der einzige Weg, der Schülernamen unumkehrbar anonymisiert. Ins
		// Protokoll gehen Akteur, Modus und Zähler, keine Namen.
		t.protokolliereLUSDImport(r, res, lauf.MassenabgangBestaetigt)
	}
	tuer.AntworteJSON(w, http.StatusOK, res)
}

// lusdFehlerStatus ordnet den Fehler eines Laufs ein: Der Massenabgang wartet auf eine
// Bestätigung (409), eine überholte Umbenennungs-Auswahl ist eine Fehleingabe (400).
func lusdFehlerStatus(err error) int {
	var massenFehler *lusd.MassenabgangFehler
	var wahlFehler *lusd.UmbenennungUngueltigFehler
	switch {
	case errors.As(err, &massenFehler):
		return http.StatusConflict
	case errors.As(err, &wahlFehler):
		return http.StatusBadRequest
	}
	return http.StatusInternalServerError
}

// leseUmbenennungsWahl liest das Formularfeld `umbenennungen` (JSON-Liste aus Zeile +
// schueler_id). Leer heißt: keine Paare bestätigt.
func leseUmbenennungsWahl(roh string) ([]lusd.UmbenennungWahl, error) {
	if strings.TrimSpace(roh) == "" {
		return nil, nil
	}
	var wahl []lusd.UmbenennungWahl
	if err := json.Unmarshal([]byte(roh), &wahl); err != nil {
		return nil, fmt.Errorf("Umbenennungs-Auswahl unlesbar: %w", err) //nolint:staticcheck // ST1005: nutzer-sichtbarer Text
	}
	return wahl, nil
}

// LUSDVorschau liest die CSV und liefert die Vorschau der Änderungen.
func (t *Tueren) LUSDVorschau() http.HandlerFunc { return t.lusdTuer(false) }

// LUSDImport liest die CSV und übernimmt die Änderungen in einer Transaktion.
// Ab der Schwelle des Massenabgangs (lusd.MassenabgangFehler) verlangt er das Formularfeld
// confirm_graduates=true (HTTP 409 sonst) — zweite, bewusste Bestätigung.
func (t *Tueren) LUSDImport() http.HandlerFunc { return t.lusdTuer(true) }

func zaehleBestaetigte(paare []lusd.VorschauUmbenennung) int {
	n := 0
	for _, p := range paare {
		if p.Bestaetigt {
			n++
		}
	}
	return n
}

// protokolliereLUSDImport schreibt den übernehmenden Lauf ins Protokoll der Verwaltung:
// Modus, Zähler je Kategorie und ob der Massenabgang bestätigt wurde. Ohne Namen.
func (t *Tueren) protokolliereLUSDImport(r *http.Request, res *lusd.Vorschau, massenabgangBestaetigt bool) {
	claims, ok := auth.GetClaims(r.Context())
	if !ok || t.DB == nil || t.DB.Pool == nil || res == nil {
		return
	}
	details := map[string]any{
		"modus":                    res.Modus,
		"zeilen":                   res.ZeilenInDatei,
		"neu":                      len(res.Neuzugaenge),
		"klassenwechsel":           len(res.Klassenwechsel),
		"adoptionen":               len(res.Adoptionen),
		"rueckkehrer":              len(res.Rueckkehrer),
		"abgaenger":                len(res.Abgaenger),
		"nicht_im_export":          len(res.NichtImExport),
		"nicht_abgleichbar":        len(res.NichtAbgleichbar),
		"mehrdeutig":               len(res.Mehrdeutig),
		"dubletten_abweichend":     len(res.DublettenAbweichend),
		"umbenennungen_bestaetigt": zaehleBestaetigte(res.Umbenennungen),
		"karenz_tage":              res.KarenzTage,
		"massenabgang_bestaetigt":  massenabgangBestaetigt,
	}
	if err := repository.NewProtokollAbfragen(t.DB.Pool).
		ProtokolliereAdminAktion(r.Context(), claims.UserID, "LUSD_IMPORT", clientip.AusAnfrage(r), details); err != nil {
		log.Printf("LUSD-Import: Audit-Eintrag fehlgeschlagen: %v", err)
	}
}
