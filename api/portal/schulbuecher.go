package portal

import (
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"bibliothek/api/tuer"
	"bibliothek/apierrors"
	"bibliothek/pdf"
	"bibliothek/pkg/lmf"
	"bibliothek/pkg/schulzeit"
	"bibliothek/repository"
)

// Portal des Kollegiums → Schulbücher je Fach. Beide Türen liegen hinter der Anmeldung, ohne
// view_books (wie /api/portal/klassensaetze): Sie liefern nur Buch- und Zähldaten, keine Daten
// zu Ausleihen oder Personen.

// Schulbuecher ist die Tür GET /api/portal/lernmittel: die Zahlen je Fach und alle
// Schulbücher, beide unter den Filtern der Ansicht. Ein Fach wählt nur der Export; die Ansicht
// ordnet die Titel selbst den Karten der Fächer zu.
func (t *Tueren) Schulbuecher() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		filter := lernmittelFilterParam(r)
		faecher, err := repository.NewTitelAbfragen(t.DB.Pool).LadeLernmittelFaecher(ctx, filter)
		if err != nil {
			apierrors.AntworteFehlerMitMeldung(w, http.StatusInternalServerError, "schulbücher konnten nicht geladen werden", fmt.Errorf("fächer der schulbücher laden: %w", err))
			return
		}
		titel, err := repository.NewTitelAbfragen(t.DB.Pool).LadeLernmittelTitel(ctx, "", true, filter)
		if err != nil {
			apierrors.AntworteFehlerMitMeldung(w, http.StatusInternalServerError, "schulbücher konnten nicht geladen werden", fmt.Errorf("titel der schulbücher laden: %w", err))
			return
		}
		tuer.AntworteJSON(w, http.StatusOK, map[string]any{"faecher": faecher, "titel": titel})
	}
}

// SchulbuecherExport bedient GET /api/portal/lernmittel/export[?fach=…] und
// liefert die Bestandsliste als PDF (eine Zeile je Titel, mit Coverbild). Ohne ?fach
// alle Schulbücher. Die Filter der Ansicht (?jahrgang, ?zweig, ?q) gelten mit, damit
// der Ausdruck dasselbe zeigt wie der Bildschirm.
func (t *Tueren) SchulbuecherExport() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		fach, gewaehlt := r.URL.Query()["fach"]
		fachName := ""
		if gewaehlt {
			fachName = strings.TrimSpace(fach[0])
		}
		filter := lernmittelFilterParam(r)
		titel, err := repository.NewTitelAbfragen(t.DB.Pool).LadeLernmittelTitel(r.Context(), fachName, !gewaehlt, filter)
		if err != nil {
			apierrors.AntworteFehlerMitMeldung(w, http.StatusInternalServerError, "export konnte nicht erstellt werden", fmt.Errorf("titel für den export der schulbücher laden: %w", err))
			return
		}
		ueberschrift := ""
		if gewaehlt {
			ueberschrift = pdf.FachAnzeige(fachName)
		}
		datei, err := pdf.ErzeugeSchulbuecher(schulbuecherFuerPDF(titel), ueberschrift, filterKlartext(filter))
		if err != nil {
			apierrors.AntworteFehlerMitMeldung(w, http.StatusInternalServerError, "export konnte nicht erstellt werden", fmt.Errorf("pdf der schulbücher erzeugen: %w", err))
			return
		}
		name := "schulbuecher"
		if gewaehlt {
			name += "_" + dateinamenTeil(fachName)
		}
		w.Header().Set("Content-Type", "application/pdf")
		w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s_%s.pdf"`, name, schulzeit.Jetzt().Format("2006-01-02")))
		if _, err := w.Write(datei); err != nil {
			slog.Warn("Portal Lernmittel-Export: Antwort an den Browser abgebrochen", "fehler", err)
		}
	}
}

// filterKlartext beschreibt die aktive Einschränkung für den Kopf des PDFs. Ohne sie
// sähe eine gefilterte Liste wie der volle Bestand aus — der gefährlichste Ausdruck
// überhaupt, weil ihn niemand als Auszug erkennt.
func filterKlartext(f repository.LernmittelFilter) string {
	var teile []string
	if f.Jahrgang > 0 {
		teile = append(teile, fmt.Sprintf("Jahrgang %d", f.Jahrgang))
	}
	switch f.Zweig {
	case "":
	case repository.ZweigOhne:
		teile = append(teile, "ohne Schulzweig")
	default:
		teile = append(teile, f.Zweig)
	}
	if f.Suche != "" {
		teile = append(teile, "Suche: "+f.Suche)
	}
	return strings.Join(teile, " · ")
}

// lernmittelFilterParam liest ?jahrgang= (5–13), ?zweig= und ?q=; alles andere heißt
// „alle". Der Suchtext wird gekappt: Er geht als ILIKE-Muster in die Abfrage, in der Form,
// in der Titeltexte gespeichert sind (repository.TiteltextNormalform).
func lernmittelFilterParam(r *http.Request) repository.LernmittelFilter {
	q := r.URL.Query()
	f := repository.LernmittelFilter{Zweig: strings.TrimSpace(q.Get("zweig")), Suche: repository.TiteltextNormalform(q.Get("q"))}
	// Nach Zeichen kappen, nicht nach Bytes: Ein Schnitt mitten in einem Mehrbyte-Zeichen
	// schickt ungültiges UTF-8 an Postgres, und aus einer Suche würde ein 500er.
	if r := []rune(f.Suche); len(r) > 100 {
		f.Suche = string(r[:100])
	}
	if j, err := strconv.Atoi(q.Get("jahrgang")); err == nil && lmf.IstJahrgang(j) {
		f.Jahrgang = j
	}
	return f
}

// schulbuecherFuerPDF füllt die Zeilen des Blatts aus den Zeilen der Abfrage: dieselben Bücher
// und Zahlen wie auf dem Bildschirm.
func schulbuecherFuerPDF(titel []repository.LernmittelTitel) []pdf.Schulbuch {
	zeilen := make([]pdf.Schulbuch, 0, len(titel))
	for _, buch := range titel {
		var auflagen []pdf.SchulbuchAuflage
		for _, auflage := range buch.AuflagenBestand {
			auflagen = append(auflagen, pdf.SchulbuchAuflage{
				Auflage: auflage.Auflage, Erscheinungsjahr: auflage.Erscheinungsjahr, Gesamt: auflage.Gesamt,
			})
		}
		zeilen = append(zeilen, pdf.Schulbuch{
			Titel: buch.Titel, Autor: buch.Autor, Fach: buch.Fach, CoverURL: buch.CoverURL, ISBN: buch.ISBN,
			JahrgangVon: buch.JahrgangVon, JahrgangBis: buch.JahrgangBis, Zweig: buch.Zweig,
			Gezaehlt: buch.Gezaehlt, Gesamt: buch.Gesamt, Verliehen: buch.Verliehen,
			Verfuegbar: buch.Verfuegbar, Auflagen: auflagen,
		})
	}
	return zeilen
}

// dateinamenTeil macht aus einem Fachnamen einen Dateinamen-Bestandteil.
func dateinamenTeil(s string) string {
	var b strings.Builder
	for _, c := range strings.ToLower(s) {
		switch {
		case c >= 'a' && c <= 'z', c >= '0' && c <= '9':
			b.WriteRune(c)
		case c == 'ä':
			b.WriteString("ae")
		case c == 'ö':
			b.WriteString("oe")
		case c == 'ü':
			b.WriteString("ue")
		case c == 'ß':
			b.WriteString("ss")
		default:
			b.WriteRune('-')
		}
	}
	name := strings.Trim(b.String(), "-")
	for strings.Contains(name, "--") {
		name = strings.ReplaceAll(name, "--", "-")
	}
	if name == "" {
		return "ohne-fach"
	}
	return name
}
