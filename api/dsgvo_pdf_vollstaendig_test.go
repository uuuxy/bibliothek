package api

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"testing"
	"time"

	"bibliothek/internal/pdftest"
	"bibliothek/pdf"
	"bibliothek/pkg/schulzeit"
)

// Die gedruckte Auskunft enthält, was die abgerufene enthält (24.09.2026).
//
// Die Person bekommt das PDF, nicht das JSON. Das JSON zeigte seit dem 15.09.2026 die
// Nachbuch-Meldungen (4e898c98), das PDF nicht; Zeitpunkt und Grund einer Stornierung
// standen nur im JSON, obwohl FACHKONZEPT.md sagt, die Auskunft weise sie aus. Die
// vorhandenen PDF-Tests prüften nur, DASS ein PDF entsteht, nicht was darauf steht.
//
// Das Gate hängt am Merkmal, nicht an einer Liste: Es füllt JEDES Feld der Auskunft
// (DsgvoAuskunftResponse, per Reflexion) mit einem eigenen Prüfwert, erzeugt daraus das
// echte PDF und verlangt jeden Wert auf dem Blatt. Ein neues Feld oder ein neuer Teil der
// Auskunft ist damit geprüft, ohne dass jemand dieses Gate anfasst.
//
// Zeitpunkte prüft es am Datum (TT.MM.JJJJ) in der Schulzeitzone: Jeder liegt auf 23:30 UTC,
// in Berlin also schon am nächsten Tag. Ein Abschnitt, der die Zone des Servers druckt (der
// Container läuft in UTC), nennt den Vortag und fällt auf.
//
// BLINDHEIT: Ja/Nein-Werte prüft es nicht — „Ja" steht zu oft auf dem Blatt, um einem
// Feld zugeordnet zu werden.
func TestDsgvoPDF_DrucktJedeAngabeDerAuskunft(t *testing.T) {
	// Ausnahmen: Pfad → Begründung.
	ausnahmen := map[string]string{
		"protokolleintraege[].details": "Rohdaten des Protokolls; sie nennen auch die bearbeitende " +
			"Person. Was davon aufs Blatt gehört, ist offen (OFFEN.md 5.19).",
		"verwaltungsprotokolle[].details": "wie protokolleintraege[].details",
		"zugangskonto.ereignisse_im_verwaltungsprotokoll[].details": "Rohdaten des Protokolls wie " +
			"verwaltungsprotokolle[].details; dieselbe offene Frage (OFFEN.md 5.19).",
	}

	var a DsgvoAuskunftResponse
	w := &pruefwerte{pfade: map[string]bool{}}
	w.fuelle(t, reflect.ValueOf(&a).Elem(), "")
	// Selbstprobe: Läuft die Reflexion künftig leer, soll das Gate auffallen.
	if len(w.erwartet) < 50 {
		t.Fatalf("Liveness: nur %d Angaben in der Auskunft gefunden", len(w.erwartet))
	}

	roh, err := generateDsgvoAuskunftPDF(a, pdf.SchuleInfo{Name: "Testschule", Ort: "Frankfurt"})
	if err != nil {
		t.Fatalf("generateDsgvoAuskunftPDF: %v", err)
	}
	blatt := strings.Join(pdftest.Texte(t, roh), "\n")
	if !strings.Contains(blatt, "Stammdaten") {
		t.Fatal("Liveness: der Leser findet nicht einmal die Überschrift „Stammdaten“")
	}

	for _, e := range w.erwartet {
		if _, ok := ausnahmen[e.pfad]; ok {
			continue
		}
		if !strings.Contains(blatt, e.text) {
			t.Errorf("%s steht in der abgerufenen Auskunft, aber nicht auf dem Blatt (Prüfwert %q) — "+
				"drucken (api/dsgvo_pdf.go) oder hier begründet ausnehmen", e.pfad, e.text)
		}
	}
	for pfad := range ausnahmen {
		if !w.pfade[pfad] {
			t.Errorf("Ausnahme %q nennt keine Angabe der Auskunft mehr — Eintrag entfernen", pfad)
		}
	}
}

// pruefwerte füllt die Auskunft und merkt sich, welcher Text wofür auf dem Blatt stehen muss.
type pruefwerte struct {
	n        int
	erwartet []pruefwert
	pfade    map[string]bool
}

type pruefwert struct{ pfad, text string }

var (
	zeitTyp     = reflect.TypeOf(time.Time{})
	rohdatenTyp = reflect.TypeOf(json.RawMessage(nil))
)

func (w *pruefwerte) merke(pfad, text string) {
	w.pfade[pfad] = true
	w.erwartet = append(w.erwartet, pruefwert{pfad, text})
}

// fuelle setzt jedes Feld auf einen eigenen Wert: Texte auf „PruefwertNNN", Zeitpunkte auf
// je einen eigenen Tag, Zahlen auf je eine eigene Zahl; Listen bekommen genau einen Eintrag.
func (w *pruefwerte) fuelle(t *testing.T, v reflect.Value, pfad string) {
	t.Helper()
	w.n++
	switch {
	case v.Type() == zeitTyp:
		// Drei Tage Abstand: Der UTC-Tag eines Feldes ist nie der Berliner Tag eines anderen.
		zeitpunkt := time.Date(1990, 1, 1, 23, 30, 0, 0, time.UTC).AddDate(0, 0, 3*w.n)
		v.Set(reflect.ValueOf(zeitpunkt))
		w.merke(pfad, zeitpunkt.In(schulzeit.Zone()).Format(dsgvoDatumFormat))
	case v.Type() == rohdatenTyp:
		wort := fmt.Sprintf("Pruefwert%03d", w.n)
		v.Set(reflect.ValueOf(json.RawMessage(`{"pruefwert":"` + wort + `"}`)))
		w.merke(pfad, wort)
	case v.Kind() == reflect.Pointer:
		v.Set(reflect.New(v.Type().Elem()))
		w.fuelle(t, v.Elem(), pfad)
	case v.Kind() == reflect.Struct:
		for i := 0; i < v.NumField(); i++ {
			if f := v.Type().Field(i); f.IsExported() {
				w.fuelle(t, v.Field(i), strings.TrimPrefix(pfad+"."+jsonName(f), "."))
			}
		}
	case v.Kind() == reflect.Slice:
		v.Set(reflect.MakeSlice(v.Type(), 1, 1))
		w.fuelle(t, v.Index(0), pfad+"[]")
	case v.Kind() == reflect.String:
		wort := fmt.Sprintf("Pruefwert%03d", w.n)
		v.SetString(wort)
		w.merke(pfad, wort)
	case v.Kind() == reflect.Int:
		v.SetInt(int64(7000 + w.n))
		w.merke(pfad, fmt.Sprint(7000+w.n))
	case v.Kind() == reflect.Bool:
		v.SetBool(true) // BLINDHEIT, siehe oben
	default:
		t.Fatalf("%s: den Typ %s kennt das Gate nicht — in fuelle ergänzen", pfad, v.Type())
	}
}

// jsonName ist der Name des Feldes in der abgerufenen Auskunft.
func jsonName(f reflect.StructField) string {
	if name, _, _ := strings.Cut(f.Tag.Get("json"), ","); name != "" && name != "-" {
		return name
	}
	return f.Name
}
