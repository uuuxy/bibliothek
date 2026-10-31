package leser

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"

	"bibliothek/api/tuer"
	"bibliothek/apierrors"
	"bibliothek/auth"
	"bibliothek/internal/lusd"
	"bibliothek/pkg/clientip"
	"bibliothek/pkg/closeutil"
	"bibliothek/pkg/csvutil"
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
		return lusd.Datei{}, fmt.Errorf("csv-datei fehlt: %w", err)
	}
	defer closeutil.Schliesse(datei, "lusd upload")

	inhalt, err := io.ReadAll(datei)
	if err != nil {
		return lusd.Datei{}, fmt.Errorf("csv nicht gelesen: %w", err)
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
		apierrors.AntworteFehlerMitMeldung(w, http.StatusBadRequest, meldungLUSDHochladen, err)
		return
	}
	datei, err := leseLUSDDatei(r)
	if err != nil {
		apierrors.AntworteFehlerMitMeldung(w, http.StatusBadRequest, lusdDateiMeldung(err), err)
		return
	}
	lauf := lusd.Lauf{Anwenden: uebernehmen, MassenabgangBestaetigt: uebernehmen && r.FormValue("confirm_graduates") == "true"}
	if uebernehmen {
		if lauf.Umbenennungen, err = leseUmbenennungsWahl(r.FormValue("umbenennungen")); err != nil {
			apierrors.AntworteFehlerMitMeldung(w, http.StatusBadRequest, meldungUmbenennungUnlesbar, err)
			return
		}
	}
	// Die Karenzzeit vor der Transaktion des Laufs lesen: eine Einstellung, kein Teil des
	// Bestands, den der Lauf vergleicht.
	lauf.KarenzTage = t.abgaengerKarenzTage(r.Context())
	res, err := lusd.FuehreAus(r.Context(), t.DB.Pool, datei, lauf)
	if err != nil {
		status, meldung := lusdLaufFehler(err)
		apierrors.AntworteFehlerMitMeldung(w, status, meldung, err)
		return
	}
	if uebernehmen {
		// Der Import ist der einzige Weg, der Schülernamen unumkehrbar anonymisiert. Ins
		// Protokoll gehen Akteur, Modus und Zähler, keine Namen.
		t.protokolliereLUSDImport(r, res, lauf.MassenabgangBestaetigt)
	}
	tuer.AntworteJSON(w, http.StatusOK, res)
}

// Sätze zum Hochladen und Lesen der Datei aus der LUSD.
const (
	meldungLUSDHochladen       = "Die Datei ließ sich nicht hochladen. Möglich sind CSV und .xlsx bis 10 MB."
	meldungLUSDDateiFehlt      = "Bitte die Exportdatei aus der LUSD auswählen"
	meldungLUSDDateiUnlesbar   = "Die Datei ließ sich nicht lesen"
	meldungUmbenennungUnlesbar = "Die Auswahl der Umbenennungen ließ sich nicht lesen. Bitte die Vorschau neu laden."
)

// lusdDateiMeldung nennt zu einer Datei, die sich nicht lesen lässt, einen Satz mit Anleitung:
// fehlende Datei, altes Excel-Format, fehlende Pflichtspalte, leere Datei oder die Zeile, an der
// das Lesen scheiterte.
func lusdDateiMeldung(err error) string {
	var spalte *lusd.PflichtspalteFehler
	var zeile *lusd.ZeileFehler
	switch {
	case errors.Is(err, http.ErrMissingFile):
		return meldungLUSDDateiFehlt
	case errors.As(err, &spalte):
		return fmt.Sprintf("In der Kopfzeile fehlt die Pflichtspalte „%s“. Ist das die Exportdatei aus der LUSD?", spalte.Spalte)
	case errors.Is(err, lusd.ErrAltesExcel):
		return "Die Datei ist ein altes Excel-Binärformat (.xls). " +
			"Bitte in Excel „Speichern unter“ → .xlsx oder CSV wählen und erneut hochladen."
	case errors.Is(err, csvutil.ErrKodierung):
		return csvutil.MeldungKodierung
	case errors.As(err, &zeile):
		return lusdZeilenMeldung(zeile)
	case errors.Is(err, lusd.ErrDateiLeer):
		return "Die Datei ist leer"
	case errors.Is(err, lusd.ErrKeinBlatt):
		return "Die Excel-Datei enthält kein Blatt mit Daten"
	case errors.Is(err, lusd.ErrExcelUnlesbar):
		return "Die Excel-Datei ließ sich nicht öffnen. Bitte als .xlsx oder CSV neu speichern."
	}
	return meldungLUSDDateiUnlesbar
}

// lusdZeilenMeldung nennt die Zeile, an der das Lesen scheiterte, und was dort fehlt.
func lusdZeilenMeldung(zeile *lusd.ZeileFehler) string {
	switch {
	case errors.Is(zeile, lusd.ErrPflichtfeldLeer):
		return fmt.Sprintf("In Zeile %d ist Vorname, Nachname oder Klasse leer", zeile.Zeile)
	case errors.Is(zeile, lusd.ErrGeburtsdatumFehlt):
		return fmt.Sprintf("In Zeile %d fehlt das Geburtsdatum oder ist unlesbar. Ohne LUSD-ID ordnen Name und "+
			"Geburtsdatum die Zeile zu, deshalb muss es in jeder Zeile stehen.", zeile.Zeile)
	}
	return fmt.Sprintf("Zeile %d der Datei ist nicht lesbar", zeile.Zeile)
}

// lusdLaufFehler ordnet den Fehler eines Laufs ein und nennt den Satz dazu: Der Massenabgang
// wartet auf eine Bestätigung (409), eine überholte Umbenennungs-Auswahl ist eine Fehleingabe
// (400), alles andere ein Serverfehler ohne eigenen Satz.
func lusdLaufFehler(err error) (status int, meldung string) {
	var massenabgang *lusd.MassenabgangFehler
	var wahl *lusd.UmbenennungUngueltigFehler
	switch {
	case errors.As(err, &massenabgang):
		return http.StatusConflict, fmt.Sprintf("%d von %d aktiven Schülern würden zu Abgängern (Schwelle: %d%%). "+
			"Bitte die Datei prüfen. Ist der Massenabgang beabsichtigt (Schuljahreswechsel), den Import mit "+
			"Bestätigung wiederholen.", massenabgang.Abgaenger, massenabgang.Aktive, massenabgang.SchwelleProzent)
	case errors.As(err, &wahl):
		return http.StatusBadRequest, fmt.Sprintf("Die Zuordnung für Zeile %d ist nicht mehr gültig. Bitte die Vorschau neu laden.", wahl.Zeile)
	}
	return http.StatusInternalServerError, ""
}

// leseUmbenennungsWahl liest das Formularfeld `umbenennungen` (JSON-Liste aus Zeile +
// schueler_id). Leer heißt: keine Paare bestätigt.
func leseUmbenennungsWahl(roh string) ([]lusd.UmbenennungWahl, error) {
	if strings.TrimSpace(roh) == "" {
		return nil, nil
	}
	var wahl []lusd.UmbenennungWahl
	if err := json.Unmarshal([]byte(roh), &wahl); err != nil {
		return nil, err
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
	claims, ok := auth.ClaimsAusKontext(r.Context())
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
		ProtokolliereAdminAktion(r.Context(), claims.KontoID, "LUSD_IMPORT", clientip.AusAnfrage(r), details); err != nil {
		slog.Error("LUSD-Import: Audit-Eintrag fehlgeschlagen", "fehler", err)
	}
}
