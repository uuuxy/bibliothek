package api

import (
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"bibliothek/pkg/betrag"
	"bibliothek/repository"

	"github.com/jung-kurt/gofpdf"
)

// Die Protokolleinträge auf dem Blatt der Auskunft: der Vorgang in Worten, darunter die Angaben
// des Eintrags mit Bezeichnung. Welche Schlüssel eine Bezeichnung haben und welche nicht aufs
// Blatt kommen (Kennungen des Programms, das buchende Konto), steht in
// repository/protokoll_auskunft.go. Einen Vorgang oder Schlüssel, den das Blatt nicht kennt,
// druckt es mit seinem Namen, statt ihn wegzulassen.

// dsgvoVorgangsArt ist ein Vorgang der Datensatz-Historie: Tabelle, Aktion und der Merker, den
// der Schreiber unter details.action oder details.anlass ablegt.
type dsgvoVorgangsArt struct{ tabelle, aktion, merker string }

// dsgvoVorgaenge nennt je Vorgang der Datensatz-Historie (audit_log) die Worte des Blatts.
var dsgvoVorgaenge = map[dsgvoVorgangsArt]string{
	{"ausleihen", "CHECKOUT", ""}:                                         "Ausleihe",
	{"ausleihen", "RETURN", ""}:                                           "Rückgabe",
	{"ausleihen", "DELETE", "titel_geloescht_mit_offener_ausleihe"}:       "Titel gelöscht, während das Buch ausgeliehen war",
	{"schueler", "UPDATE", "soft_delete"}:                                 "Leserdatensatz in den Papierkorb gelegt",
	{"schueler", "DELETE", "purge"}:                                       "Leserdatensatz endgültig gelöscht",
	{"schueler", "ZUSAMMENGEFUEHRT", "zusammenfuehren"}:                   "Mit einem zweiten Datensatz derselben Person zusammengeführt",
	{"schueler", "dsgvo_auskunft", ""}:                                    "Auskunft nach Art. 15 DSGVO erteilt",
	{"schadensfaelle", "STORNIERUNG", ""}:                                 "Forderung storniert",
	{"schadensfaelle", "STORNIERUNG", "rueckgabe"}:                        "Forderung storniert, weil das Buch zurück ist",
	{"schadensfaelle", "DELETE", "titel_geloescht_mit_offener_forderung"}: "Titel gelöscht, während eine Forderung offen war",
	{"vormerkungen", "DELETE", "titel_geloescht_mit_offenem_bezug"}:       "Titel gelöscht, während eine Vormerkung offen war",
}

// dsgvoProgrammSaetze sind die Sätze, die das Programm als kontext an einen dieser Vorgänge
// schreibt. Das Blatt nennt den Vorgang in eigenen Worten und lässt sie weg; jeder andere
// kontext steht hinter dem Vorgang.
var dsgvoProgrammSaetze = map[string]bool{
	"Soft-Delete Routine":                                           true,
	"Gebühr storniert":                                              true,
	"Forderung storniert: das Buch ist zurück":                      true,
	"Titel gelöscht, Buch war zu diesem Zeitpunkt verliehen":        true,
	"Titel gelöscht, es stand noch eine unbezahlte Forderung offen": true,
	"Titel gelöscht, eine Vormerkung stand noch offen":              true,
}

// dsgvoVerwaltungsVorgaenge nennt je Aktion des Verwaltungsprotokolls (audit_logs) die Worte
// des Blatts.
var dsgvoVerwaltungsVorgaenge = map[string]string{
	"DELETE_STUDENT":               "Leserdatensatz in den Papierkorb gelegt",
	"RESTORE_STUDENT":              "Leserdatensatz aus dem Papierkorb zurückgeholt",
	"PURGE_STUDENT":                "Leserdatensatz endgültig gelöscht",
	"LESER_GESPERRT":               "Von Hand gesperrt",
	"LESER_ENTSPERRT":              "Sperre aufgehoben",
	"OVERRIDE_BLOCK":               "Ausleihe trotz eines Hinweises gebucht",
	"LUSD_ID_NACHGETRAGEN":         "LUSD-ID nachgetragen",
	"KOLLEGIUMSKONTO_NACHGETRAGEN": "Zugangskonto angelegt: Schul-E-Mail in der Leserakte nachgetragen",
	"SCHUELER_ZUSAMMENGEFUEHRT":    "Mit einem zweiten Datensatz derselben Person zusammengeführt",
	auditBescheidErstellt:          "Schadensersatz-Bescheid erstellt",
}

// dsgvoVorgang schreibt einen Eintrag der Datensatz-Historie in Worten. Kennt das Blatt den
// Vorgang nicht, stehen Aktion und Tabelle so da, wie sie gespeichert sind.
func dsgvoVorgang(e DsgvoAuditEintrag, details map[string]any) string {
	merker := ""
	if text, ok := details["action"].(string); ok {
		merker = text
	} else if text, ok := details["anlass"].(string); ok {
		merker = text
	}
	worte, bekannt := dsgvoVorgaenge[dsgvoVorgangsArt{e.Tabelle, e.Aktion, merker}]
	if !bekannt {
		worte = e.Aktion
		if e.Tabelle != "" {
			worte += " (" + e.Tabelle + ")"
		}
	}
	if e.Kontext != nil && *e.Kontext != "" && (!bekannt || !dsgvoProgrammSaetze[*e.Kontext]) {
		worte += " — " + *e.Kontext
	}
	switch e.Akteur {
	case "USER", "":
		// Steht in der Erläuterung des Abschnitts: gebucht von einem Konto der Bibliothek.
	case "SYSTEM":
		worte += " (automatisch)"
	default:
		worte += " (" + e.Akteur + ")"
	}
	return worte
}

// dsgvoVerwaltungsVorgang schreibt eine Aktion des Verwaltungsprotokolls in Worten; eine
// unbekannte bleibt stehen.
func dsgvoVerwaltungsVorgang(aktion string) string {
	if worte, bekannt := dsgvoVerwaltungsVorgaenge[aktion]; bekannt {
		return worte
	}
	return aktion
}

func dsgvoAuditAbschnitt(p *gofpdf.Fpdf, tr func(string) string, audit []DsgvoAuditEintrag) {
	dsgvoAbschnitt(p, tr, fmt.Sprintf("8. Protokolleinträge zu diesem Datensatz (%d)", len(audit)))
	if len(audit) == 0 {
		dsgvoLeer(p, tr)
		return
	}
	dsgvoHinweis(p, tr, "Jeder Vorgang, den das Programm zu diesem Datensatz festgehalten hat. Gebucht hat ihn "+
		"ein Konto der Bibliothek, wo nicht „automatisch“ dabeisteht; welches Konto, nennt dieses Blatt nicht "+
		"(Art. 15 Abs. 4 DSGVO).")
	p.Ln(1)
	for _, e := range audit {
		details, rest := dsgvoDetails(e.Details)
		zeilen := dsgvoAngabenZeilen(details, e.Tabelle == "schueler", e.Aktion)
		if e.Gegenstand != "" || e.Barcode != "" {
			zeilen = append([]string{fmt.Sprintf("%s (Nummer %s)", dsgvoLeerWert(e.Gegenstand), dsgvoLeerWert(e.Barcode))}, zeilen...)
		} else if e.Tabelle == "ausleihen" && (e.Aktion == "CHECKOUT" || e.Aktion == "RETURN") {
			zeilen = append([]string{"Das Buch oder Gerät ist nicht mehr im Bestand."}, zeilen...)
		}
		dsgvoProtokollEintrag(p, tr, dsgvoZeit(e.Zeitpunkt)+" — "+dsgvoVorgang(e, details), append(zeilen, rest...))
	}
}

// dsgvoVerwaltungAbschnitt listet die Einträge des Verwaltungsprotokolls (audit_logs), die
// den Leser nennen.
func dsgvoVerwaltungAbschnitt(p *gofpdf.Fpdf, tr func(string) string, eintraege []DsgvoVerwaltungsEintrag) {
	dsgvoAbschnitt(p, tr, fmt.Sprintf("9. Verwaltungsprotokolle zu diesem Datensatz (%d)", len(eintraege)))
	if len(eintraege) == 0 {
		dsgvoLeer(p, tr)
		return
	}
	dsgvoHinweis(p, tr, "Eingriffe der Verwaltung an diesem Datensatz. Das Konto, das sie ausgeführt hat, und "+
		"dessen IP-Adresse nennt dieses Blatt nicht (Art. 15 Abs. 4 DSGVO).")
	p.Ln(1)
	for _, e := range eintraege {
		details, rest := dsgvoDetails(e.Details)
		dsgvoProtokollEintrag(p, tr, dsgvoZeit(e.Zeitpunkt)+" — "+dsgvoVerwaltungsVorgang(e.Aktion),
			append(dsgvoAngabenZeilen(details, false, e.Aktion), rest...))
	}
}

// dsgvoKontoEreignisse druckt die Einträge des Verwaltungsprotokolls über ein Zugangskonto.
func dsgvoKontoEreignisse(p *gofpdf.Fpdf, tr func(string) string, ereignisse []repository.DsgvoKontoEreignis) {
	for _, e := range ereignisse {
		details, rest := dsgvoDetails(e.Details)
		dsgvoProtokollEintrag(p, tr, dsgvoZeit(e.Zeitpunkt)+" — "+dsgvoKontoAktion(e.Aktion),
			append(dsgvoAngabenZeilen(details, false, e.Aktion), rest...))
	}
}

// dsgvoProtokollEintrag druckt einen Eintrag: die Kopfzeile wie ein Listeneintrag, darunter die
// Angaben in Grau.
func dsgvoProtokollEintrag(p *gofpdf.Fpdf, tr func(string) string, kopf string, zeilen []string) {
	dsgvoEintragTitel(p, tr, kopf)
	p.SetFont("Arial", "", 8)
	p.SetTextColor(90, 90, 90)
	for _, zeile := range zeilen {
		p.MultiCell(0, 4.5, tr(zeile), "", "L", false)
	}
	p.SetTextColor(0, 0, 0)
	p.Ln(1)
}

// dsgvoDetails liest die Angaben eines Eintrags. Sind sie kein Objekt, steht ihr Text als Zeile
// in rest.
func dsgvoDetails(roh json.RawMessage) (details map[string]any, rest []string) {
	text := strings.TrimSpace(string(roh))
	if text == "" || text == "null" {
		return nil, nil
	}
	if err := json.Unmarshal(roh, &details); err != nil {
		return nil, []string{text}
	}
	return details, nil
}

// dsgvoAngabenZeilen schreibt die Angaben eines Eintrags: zuerst die einfachen in einer Zeile,
// dann je Objekt (das Abbild eines Datensatzes, das Umgehängte) eine eigene.
func dsgvoAngabenZeilen(details map[string]any, amLeser bool, aktion string) []string {
	einfach, abschnitte := dsgvoAngaben(details, amLeser, aktion)
	var zeilen []string
	if len(einfach) > 0 {
		zeilen = append(zeilen, strings.Join(einfach, " · "))
	}
	return append(zeilen, abschnitte...)
}

// dsgvoAngaben liefert die Angaben eines Objekts in der Reihenfolge des Blatts
// (repository.ProtokollAngaben), danach die Schlüssel ohne Bezeichnung.
func dsgvoAngaben(objekt map[string]any, amLeser bool, aktion string) (einfach, abschnitte []string) {
	for _, a := range repository.ProtokollAngaben() {
		wert, da := objekt[a.Schluessel]
		if !da || wert == nil {
			continue
		}
		angabe, _ := repository.ProtokollAngabeZu(a.Schluessel, amLeser, aktion)
		if teil, ok := wert.(map[string]any); ok && angabe.Art == repository.ProtokollAbschnitt {
			abschnitte = append(abschnitte, dsgvoAbschnittZeile(angabe, teil, aktion)...)
			continue
		}
		if text := dsgvoProtokollWert(angabe.Art, wert); text != "" {
			einfach = append(einfach, angabe.Bezeichnung+": "+text)
		}
	}
	return append(einfach, dsgvoAngabenOhneBezeichnung(objekt)...), abschnitte
}

// dsgvoAbschnittZeile schreibt ein Objekt innerhalb eines Eintrags als eigene Zeile. Das Abbild
// eines Datensatzes beschreibt die Leserzeile; das Umgehängte zählt Zeilen anderer Tabellen.
func dsgvoAbschnittZeile(angabe repository.ProtokollAngabe, teil map[string]any, aktion string) []string {
	innen, tiefer := dsgvoAngaben(teil, angabe.Schluessel != "gewandert", aktion)
	text := strings.Join(append(innen, tiefer...), " · ")
	if text == "" {
		return nil
	}
	return []string{angabe.Bezeichnung + ": " + text}
}

// dsgvoAngabenOhneBezeichnung nennt die Schlüssel, die das Blatt nicht kennt, mit ihrem Namen,
// nach dem Alphabet.
func dsgvoAngabenOhneBezeichnung(objekt map[string]any) []string {
	var unbekannt []string
	for schluessel, wert := range objekt {
		_, bekannt := repository.ProtokollAngabeZu(schluessel, false, "")
		if !bekannt && !repository.ProtokollOhneAngabe(schluessel) && wert != nil {
			unbekannt = append(unbekannt, schluessel)
		}
	}
	sort.Strings(unbekannt)
	var angaben []string
	for _, schluessel := range unbekannt {
		if text := dsgvoWertAlsText(objekt[schluessel]); text != "" {
			angaben = append(angaben, schluessel+": "+text)
		}
	}
	return angaben
}

// dsgvoProtokollWert schreibt einen Wert nach seiner Art. Passt der gespeicherte Wert nicht zur
// Art, steht er als Text da.
func dsgvoProtokollWert(art repository.ProtokollWertArt, wert any) string {
	if text, passt := dsgvoWertNachArt(art, wert); passt {
		return text
	}
	return dsgvoWertAlsText(wert)
}

func dsgvoWertNachArt(art repository.ProtokollWertArt, wert any) (string, bool) {
	switch w := wert.(type) {
	case string:
		switch art {
		case repository.ProtokollZeit:
			return dsgvoProtokollZeit(w), true
		case repository.ProtokollBetrag:
			return dsgvoBetrag(w) + " EUR", true
		case repository.ProtokollRolle:
			return dsgvoRolle(w), true
		case repository.ProtokollLeserart:
			return dsgvoLeserart(w), true
		}
	case float64:
		if art == repository.ProtokollBetrag {
			return betrag.Text(w) + " EUR", true
		}
	case []any:
		if art == repository.ProtokollAnzahl {
			return strconv.Itoa(len(w)), true
		}
	}
	return "", false
}

// dsgvoWertAlsText schreibt einen Wert ohne eigene Form: Text, Zahl, Ja oder Nein; alles andere
// so, wie es gespeichert ist.
func dsgvoWertAlsText(wert any) string {
	switch w := wert.(type) {
	case string:
		return strings.TrimSpace(w)
	case float64:
		return strconv.FormatFloat(w, 'f', -1, 64)
	case bool:
		return dsgvoJaNein(w)
	}
	roh, err := json.Marshal(wert)
	if err != nil {
		return fmt.Sprint(wert)
	}
	return string(roh)
}

// dsgvoProtokollZeitformen sind die Formen, in denen die Schreiber einen Zeitpunkt oder Tag in
// details ablegen: RFC 3339, der Text eines Zeitstempels aus Postgres, ein Tag.
var dsgvoProtokollZeitformen = []struct {
	form     string
	nurDatum bool
}{
	{time.RFC3339Nano, false},
	{"2006-01-02 15:04:05.999999999Z07", false},
	{"2006-01-02 15:04:05.999999999Z07:00", false},
	{"2006-01-02 15:04:05.999999999", false},
	{"2006-01-02", true},
}

// dsgvoProtokollZeit schreibt einen gespeicherten Zeitpunkt wie die übrigen des Blatts; einen
// Text in anderer Form lässt es stehen.
func dsgvoProtokollZeit(text string) string {
	text = strings.TrimSpace(text)
	for _, f := range dsgvoProtokollZeitformen {
		t, err := time.Parse(f.form, text)
		if err != nil {
			continue
		}
		if f.nurDatum {
			return t.Format(dsgvoDatumFormat)
		}
		return dsgvoZeit(t)
	}
	return text
}

// dsgvoBetrag schreibt einen Betrag aus der Datenbank („12.50") wie Briefe und Berichte
// („12,50"); ein Text, der keine Zahl ist, bleibt stehen.
func dsgvoBetrag(text string) string {
	zahl, err := strconv.ParseFloat(strings.TrimSpace(text), 64)
	if err != nil {
		return text
	}
	return betrag.Text(zahl)
}
