package api

import (
	"bytes"
	"fmt"
	"net/http"
	"strings"
	"time"

	"bibliothek/apierrors"
	"bibliothek/pdf"
	"bibliothek/pkg/pdfzeichen"
	"bibliothek/pkg/schulzeit"
	"bibliothek/repository"

	"github.com/jung-kurt/gofpdf"
)

const (
	dsgvoDatumFormat = "02.01.2006"
	dsgvoZeitFormat  = "02.01.2006, 15:04 Uhr"
)

// DsgvoAuskunftPDFHandler liefert die Betroffenenauskunft nach Art. 15 DSGVO als
// lesbares PDF. Inhaltlich identisch zur JSON-Variante (dieselbe sammleDsgvoDaten-
// Quelle), nur menschenlesbar aufbereitet — gedacht zur Aushändigung an die Person bzw.
// die Erziehungsberechtigten (JSON ist für diesen Zweck ungeeignet). Die Erteilung wird
// wie bei der JSON-Auskunft im Audit-Log protokolliert (Rechenschaftspflicht).
// @Summary      DSGVO-Betroffenenauskunft (Art. 15) als PDF
// @Tags         students
// @Produce      application/pdf
// @Param        id   path      string  true  "Reader ID (UUID)"
// @Success      200  {file}    binary
// @Router       /schueler/{id}/dsgvo-auskunft/pdf [get]
func (s *Server) DsgvoAuskunftPDFHandler() http.HandlerFunc {
	return apierrors.Wrap(func(w http.ResponseWriter, r *http.Request) error {
		id := r.PathValue("id")
		if id == "" {
			return apierrors.BadRequest("missing student ID parameter", nil)
		}
		ctx := r.Context()

		daten, err := s.sammleDsgvoDaten(ctx, id)
		if err != nil {
			return err
		}

		// Rechenschaftspflicht: Auskunftserteilung protokollieren (wie JSON-Variante).
		s.protokolliereDsgvoAuskunft(ctx, id)

		settingsRepo := repository.NewSystemSettingsRepository(s.DB.Pool)
		settings, _ := settingsRepo.GetSettings(ctx) //nolint:errcheck // Header fällt sonst auf Defaults zurück
		schule := pdf.SchuleInfo{
			Name:    settings.SchuleName,
			Strasse: settings.SchuleStrasse,
			PLZ:     settings.SchulePLZ,
			Ort:     settings.SchuleOrt,
		}

		pdfBytes, err := generateDsgvoAuskunftPDF(dsgvoAntwort(daten, schulzeit.Jetzt()), schule)
		if err != nil {
			return apierrors.Internal("PDF-Erzeugung fehlgeschlagen", err)
		}

		w.Header().Set(headerContentType, contentTypePDF)
		w.Header().Set(headerContentDisposition, `attachment; filename="DSGVO-Auskunft.pdf"`)
		w.Header().Set(headerContentLength, fmt.Sprint(len(pdfBytes)))
		http.ServeContent(w, r, "DSGVO-Auskunft.pdf", schulzeit.Jetzt(), bytes.NewReader(pdfBytes))
		return nil
	})
}

// generateDsgvoAuskunftPDF rendert die vollständige Art.-15-Auskunft als PDF — aus
// demselben Objekt, das die JSON-Antwort ist (dsgvoAntwort).
func generateDsgvoAuskunftPDF(a DsgvoAuskunftResponse, schule pdf.SchuleInfo) ([]byte, error) {
	p := gofpdf.New("P", "mm", "A4", "")
	p.SetMargins(20, 20, 20)
	p.SetAutoPageBreak(true, 20)
	tr := pdfzeichen.Uebersetzer(p.UnicodeTranslatorFromDescriptor("")) // UTF-8 → Latin-1 für Umlaute
	p.AddPage()

	dsgvoKopf(p, tr, schule, a)
	dsgvoStammdatenAbschnitt(p, tr, &a.Stammdaten)
	dsgvoFotoAbschnitt(p, tr, a.Foto)
	dsgvoAusleihAbschnitt(p, tr, a.Ausleihen)
	dsgvoSchadensAbschnitt(p, tr, a.Schadensfaelle)
	dsgvoVormerkAbschnitt(p, tr, a.Vormerkungen)
	dsgvoBescheidAbschnitt(p, tr, a.Bescheide)
	dsgvoNachbuchAbschnitt(p, tr, a.NachbuchMeldungen)
	dsgvoAuditAbschnitt(p, tr, a.AuditEintraege)
	dsgvoVerwaltungAbschnitt(p, tr, a.Verwaltung)
	dsgvoKontoAbschnitt(p, tr, a.Zugangskonto)
	dsgvoVerarbeitungAbschnitt(p, tr, a.Verarbeitungsangaben)

	var buf bytes.Buffer
	if err := p.Output(&buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func dsgvoKopf(p *gofpdf.Fpdf, tr func(string) string, schule pdf.SchuleInfo, a DsgvoAuskunftResponse) {
	p.SetFont("Arial", "B", 14)
	p.Cell(0, 8, tr(schule.Name))
	p.Ln(7)
	p.SetFont("Arial", "", 8)
	p.SetTextColor(100, 100, 100)
	p.Cell(0, 4, tr(schule.Absenderzeile()))
	p.SetTextColor(0, 0, 0)
	p.Ln(11)

	p.SetFont("Arial", "B", 16)
	p.Cell(0, 10, tr(a.Art))
	p.Ln(9)
	p.SetFont("Arial", "", 10)
	p.SetTextColor(90, 90, 90)
	// Ein Kollege hat keine Klasse; bis zum 24.09.2026 gab es seine Auskunft nicht, und die
	// Zeile hätte „(Klasse )" gelautet.
	st := a.Stammdaten
	zuordnung := "Klasse " + st.Klasse
	if st.Klasse == "" {
		zuordnung = dsgvoLeserart(st.Art)
	}
	p.Cell(0, 6, tr(fmt.Sprintf("Betroffene Person: %s %s (%s)", st.Vorname, st.Nachname, zuordnung)))
	p.Ln(5)
	p.Cell(0, 6, tr("Erstellt am: "+dsgvoZeit(a.ErstelltAm)))
	p.SetTextColor(0, 0, 0)
	p.Ln(9)
	p.SetFont("Arial", "", 9)
	p.MultiCell(0, 5, tr("Diese Auskunft enthält alle zu der oben genannten Person in diesem System "+
		"gespeicherten personenbezogenen Daten sowie die Pflichtangaben nach Art. 15 Abs. 1 DSGVO."), "", "L", false)
}

func dsgvoStammdatenAbschnitt(p *gofpdf.Fpdf, tr func(string) string, st *DsgvoStammdaten) {
	dsgvoAbschnitt(p, tr, "1. Stammdaten")
	dsgvoZeile(p, tr, "Interne ID", st.ID)
	// Art und Zugangskonto kamen mit Migration 123 in die Auskunft, aber nicht auf dieses
	// Blatt — die gedruckte Auskunft war damit kürzer als die abgerufene. Nachgetragen am
	// 23.09.2026 samt Ratsche (TestDsgvoPDF_DrucktJedesStammdatenfeld).
	dsgvoZeile(p, tr, "Art des Lesers", dsgvoLeserart(st.Art))
	dsgvoZeile(p, tr, "Ausweis-Barcode", st.BarcodeID)
	dsgvoZeile(p, tr, "Vorname", st.Vorname)
	dsgvoZeile(p, tr, "Nachname", st.Nachname)
	dsgvoZeile(p, tr, "Klasse", st.Klasse)
	dsgvoZeile(p, tr, "Geburtsdatum", dsgvoStrPtr(st.Geburtsdatum))
	dsgvoZeile(p, tr, "Schuleintritt", dsgvoStrPtr(st.SchulEintrittAm))
	dsgvoZeile(p, tr, "Abgangsjahr", fmt.Sprint(st.AbgaengerJahr))
	dsgvoZeile(p, tr, "Abgänger seit", dsgvoZeitPtr(st.AbgaengerSeit))
	dsgvoZeile(p, tr, "Letzter abgeschlossener Vorgang", dsgvoZeitPtr(st.LetzterVorgangAm))
	dsgvoZeile(p, tr, "LUSD-ID", dsgvoStrPtr(st.LusdID))
	dsgvoZeile(p, tr, "Zuletzt im LUSD-Export bestätigt", dsgvoZeitPtr(st.LusdBestaetigtAm))
	dsgvoZeile(p, tr, "Anonymisiert am", dsgvoZeitPtr(st.AnonymisiertAm))
	dsgvoZeile(p, tr, "Straße/Nr.", strings.TrimSpace(st.Strasse+" "+st.Hausnummer))
	dsgvoZeile(p, tr, "PLZ/Ort", strings.TrimSpace(st.Plz+" "+st.Ort))
	dsgvoZeile(p, tr, "Eltern-E-Mail", st.ElternEmail)
	dsgvoZeile(p, tr, "Gesperrt", dsgvoJaNein(st.IstGesperrt))
	dsgvoZeile(p, tr, "Manuell gesperrt", dsgvoJaNein(st.IsManuallyBlocked))
	dsgvoZeile(p, tr, "Sperrgrund", dsgvoStrPtr(st.BlockReason))
	dsgvoZeile(p, tr, "Abgänger", dsgvoJaNein(st.IstAbgaenger))
	dsgvoZeile(p, tr, "Erfasst am", dsgvoZeit(st.ErstelltAm))
	dsgvoZeile(p, tr, "Zuletzt geändert", dsgvoZeit(st.AktualisiertAm))
	if st.GeloeschtAm != nil {
		dsgvoZeile(p, tr, "Gelöscht am (Papierkorb)", dsgvoZeit(*st.GeloeschtAm))
	}
	// Die Anmeldedaten selbst stehen nicht hier, sondern in Abschnitt 10 (sie gehören zum
	// Konto, nicht zum Leser); dass es eines gibt, ist aber eine Angabe über diese Person.
	dsgvoZeile(p, tr, "Zugangskonto vorhanden", dsgvoJaNein(st.HatZugangskonto))
}

// dsgvoLeserart schreibt die gespeicherte Art aus. Der Wert steht so in der Datenbank
// (leser.art); ein unbekannter bleibt unverändert stehen, statt still zu verschwinden.
func dsgvoLeserart(art string) string {
	switch art {
	case "schueler":
		return "Schüler/in"
	case "lehrkraft":
		return "Lehrkraft"
	case "liv":
		return "Lehrkraft im Vorbereitungsdienst"
	}
	return art
}

func dsgvoFotoAbschnitt(p *gofpdf.Fpdf, tr func(string) string, foto DsgvoFoto) {
	dsgvoAbschnitt(p, tr, "2. Ausweisfoto")
	dsgvoZeile(p, tr, "Vorhanden", dsgvoJaNein(foto.Vorhanden))
	if foto.AktualisiertAm != nil {
		dsgvoZeile(p, tr, "Aktualisiert am", dsgvoZeit(*foto.AktualisiertAm))
	}
	dsgvoZeile(p, tr, "Hinweis", foto.Hinweis)
}

func dsgvoAusleihAbschnitt(p *gofpdf.Fpdf, tr func(string) string, ausleihen []DsgvoAusleihe) {
	dsgvoAbschnitt(p, tr, fmt.Sprintf("3. Ausleihhistorie (%d Einträge)", len(ausleihen)))
	if len(ausleihen) == 0 {
		dsgvoLeer(p, tr)
		return
	}
	for _, a := range ausleihen {
		rueck := "offen"
		if a.RueckgabeAm != nil {
			rueck = dsgvoDatum(*a.RueckgabeAm)
		}
		dsgvoEintragTitel(p, tr, a.Gegenstand)
		dsgvoEintragZeile(p, tr, fmt.Sprintf("Barcode: %s · ausgeliehen: %s · Frist: %s · zurückgegeben: %s",
			dsgvoLeerWert(a.Barcode), dsgvoDatum(a.AusgeliehenAm),
			dsgvoDatum(a.RueckgabeFrist), rueck))
	}
}

func dsgvoSchadensAbschnitt(p *gofpdf.Fpdf, tr func(string) string, schaeden []DsgvoSchadensfall) {
	dsgvoAbschnitt(p, tr, fmt.Sprintf("4. Schadens- und Verlustfälle (%d)", len(schaeden)))
	if len(schaeden) == 0 {
		dsgvoLeer(p, tr)
		return
	}
	for _, f := range schaeden {
		status := "offen"
		if f.IstBezahlt {
			status = "bezahlt"
		}
		// Zeitpunkt und Grund der Stornierung standen bis zum 24.09.2026 nur in der
		// abgerufenen Auskunft, nicht auf diesem Blatt.
		if f.StorniertAm != nil {
			status = "storniert am " + dsgvoDatum(*f.StorniertAm)
			if f.Stornierungsgrund != nil && *f.Stornierungsgrund != "" {
				status += " (Grund: " + *f.Stornierungsgrund + ")"
			}
		}
		dsgvoEintragTitel(p, tr, f.Beschreibung)
		dsgvoEintragZeile(p, tr, fmt.Sprintf("Betrag: %s EUR · Status: %s · gemeldet: %s",
			f.Betrag, status, dsgvoDatum(f.ErstelltAm)))
	}
}

func dsgvoVormerkAbschnitt(p *gofpdf.Fpdf, tr func(string) string, vormerkungen []DsgvoVormerkung) {
	dsgvoAbschnitt(p, tr, fmt.Sprintf("5. Vormerkungen (%d)", len(vormerkungen)))
	if len(vormerkungen) == 0 {
		dsgvoLeer(p, tr)
		return
	}
	for _, v := range vormerkungen {
		dsgvoEintragTitel(p, tr, v.Titel)
		dsgvoEintragZeile(p, tr, fmt.Sprintf("Status: %s · erstellt: %s%s",
			v.Status, dsgvoDatum(v.ErstelltAm), dsgvoNotiz(v.Notiz)))
	}
}

// dsgvoBescheidAbschnitt nennt die Schadensersatz-Bescheide, die an diese Person
// gerichtet waren (Migration 110). Die Positionen stehen als Schadensfälle in Abschnitt 4;
// hier steht der Brief selbst — Nummer, Frist, Summe, Zustand.
func dsgvoBescheidAbschnitt(p *gofpdf.Fpdf, tr func(string) string, bescheide []DsgvoBescheid) {
	dsgvoAbschnitt(p, tr, fmt.Sprintf("6. Schadensersatz-Bescheide (%d)", len(bescheide)))
	if len(bescheide) == 0 {
		dsgvoLeer(p, tr)
		return
	}
	for _, b := range bescheide {
		dsgvoEintragTitel(p, tr, "Bescheid "+b.Referenznummer)
		dsgvoEintragZeile(p, tr, fmt.Sprintf("vom %s · Frist: %s · Betrag: %s EUR · Status: %s",
			dsgvoDatum(b.BriefDatum), dsgvoDatum(b.FristBis),
			b.Gesamtbetrag, b.Status))
	}
}

// dsgvoNachbuchAbschnitt nennt die Nachbuch-Meldungen, in denen die Person steht
// (Migration 117): Die Theke hat ohne Netz gescannt, und beim späteren Buchen wich das
// Ergebnis vom Scan ab. In der abgerufenen Auskunft seit dem 15.09.2026 (4e898c98), auf
// diesem Blatt seit dem 24.09.2026. Eine Meldung nennt keine andere Person — nur, in
// welcher Rolle diese hier beteiligt war.
func dsgvoNachbuchAbschnitt(p *gofpdf.Fpdf, tr func(string) string, meldungen []DsgvoNachbuchMeldung) {
	dsgvoAbschnitt(p, tr, fmt.Sprintf("7. Meldungen zu Buchungen nach einem Netzausfall (%d)", len(meldungen)))
	if len(meldungen) == 0 {
		dsgvoLeer(p, tr)
		return
	}
	for _, m := range meldungen {
		ergebnis := dsgvoNachbuchErgebnis(m.Ergebnis)
		dsgvoEintragTitel(p, tr, ergebnis)
		grund := ""
		if m.Grund != nil && *m.Grund != "" && *m.Grund != ergebnis {
			grund = " · Grund: " + *m.Grund
		}
		bearbeitet := "noch nicht bearbeitet"
		if m.QuittiertAm != nil {
			bearbeitet = "bearbeitet am " + dsgvoDatum(*m.QuittiertAm)
		}
		dsgvoEintragZeile(p, tr, fmt.Sprintf("Buch: %s · gescannt: %s · beteiligt als %s%s · %s",
			dsgvoLeerWert(m.Barcode), dsgvoZeit(m.GescanntAm),
			dsgvoNachbuchRolle(m.Rolle), grund, bearbeitet))
	}
}

// dsgvoNachbuchErgebnis schreibt das Ergebniswort der Nachbuch-Tür aus, mit demselben
// Wortlaut wie die Meldungsliste der Bibliothek (NachbuchMeldungen.svelte). Ein unbekanntes
// Wort bleibt stehen, statt still zu verschwinden.
func dsgvoNachbuchErgebnis(ergebnis string) string {
	switch ergebnis {
	case repository.NachbuchUmgebucht:
		return "lag bei jemand anderem — dort zurückgenommen, neu ausgeliehen"
	case repository.NachbuchNurReaktiviert:
		return "Buch war abgeschrieben und ist wieder im Umlauf"
	case repository.NachbuchNichtGebucht:
		return "nicht gebucht"
	case repository.NachbuchVeraltet:
		return "nicht gebucht — es gab schon eine neuere Buchung"
	case repository.NachbuchBereitsAusgeliehen:
		return "war schon ausgeliehen"
	case repository.NachbuchZurueckgegeben:
		return "zurückgegeben"
	case repository.NachbuchAusgeliehen:
		return "ausgeliehen"
	}
	return ergebnis
}

// dsgvoNachbuchRolle sagt, als wer die Person an der Meldung beteiligt war
// (dsgvoQueryNachbuchMeldungen). Ein unbekannter Wert bleibt stehen.
func dsgvoNachbuchRolle(rolle string) string {
	switch rolle {
	case "ausleiher":
		return "Ausleiher/in"
	case "vorbesitzer":
		return "bisherige/r Ausleiher/in"
	}
	return rolle
}

func dsgvoAuditAbschnitt(p *gofpdf.Fpdf, tr func(string) string, audit []DsgvoAuditEintrag) {
	dsgvoAbschnitt(p, tr, fmt.Sprintf("8. Protokolleinträge zu diesem Datensatz (%d)", len(audit)))
	if len(audit) == 0 {
		dsgvoLeer(p, tr)
		return
	}
	p.SetFont("Arial", "", 8)
	for _, e := range audit {
		kontext := ""
		if e.Kontext != nil && *e.Kontext != "" {
			kontext = " · " + *e.Kontext
		}
		p.MultiCell(0, 5, tr(fmt.Sprintf("%s — %s — %s%s",
			dsgvoZeit(e.Zeitpunkt), e.Aktion, e.Akteur, kontext)), "", "L", false)
	}
}

// dsgvoVerwaltungAbschnitt listet die Verwaltungsprotokolle (audit_logs) — seit dem
// 31.08.2026 Teil der Auskunft; vorher war audit_logs die eine Quelle mit Schülerbezug,
// die die Auskunft nicht las.
func dsgvoVerwaltungAbschnitt(p *gofpdf.Fpdf, tr func(string) string, eintraege []DsgvoVerwaltungsEintrag) {
	dsgvoAbschnitt(p, tr, fmt.Sprintf("9. Verwaltungsprotokolle zu diesem Datensatz (%d)", len(eintraege)))
	if len(eintraege) == 0 {
		dsgvoLeer(p, tr)
		return
	}
	p.SetFont("Arial", "", 8)
	for _, e := range eintraege {
		p.MultiCell(0, 5, tr(fmt.Sprintf("%s — %s",
			dsgvoZeit(e.Zeitpunkt), e.Aktion)), "", "L", false)
	}
}

func dsgvoVerarbeitungAbschnitt(p *gofpdf.Fpdf, tr func(string) string, va DsgvoVerarbeitungsangaben) {
	dsgvoAbschnitt(p, tr, "11. Angaben zur Verarbeitung (Art. 15 Abs. 1 DSGVO)")
	dsgvoAbsatz(p, tr, "Verarbeitungszwecke", strings.Join(va.Zwecke, "; "))
	dsgvoAbsatz(p, tr, "Rechtsgrundlage", va.Rechtsgrundlage)
	dsgvoAbsatz(p, tr, "Empfänger", va.Empfaenger)
	dsgvoAbsatz(p, tr, "Speicherdauer", va.Speicherdauer)
	dsgvoAbsatz(p, tr, "Herkunft der Daten", va.Herkunft)
	dsgvoAbsatz(p, tr, "Betroffenenrechte", va.Betroffenenrechte)
}

// ── Layout-Helfer ────────────────────────────────────────────────────────────

// dsgvoAbschnitt schreibt eine Abschnittsüberschrift mit Trennlinie.
func dsgvoAbschnitt(p *gofpdf.Fpdf, tr func(string) string, titel string) {
	p.Ln(5)
	p.SetFont("Arial", "B", 12)
	p.SetTextColor(30, 30, 30)
	p.Cell(0, 8, tr(titel))
	p.Ln(9)
	y := p.GetY()
	p.SetDrawColor(200, 200, 200)
	p.Line(20, y, 190, y)
	p.Ln(3)
	p.SetTextColor(0, 0, 0)
}

// dsgvoZeile schreibt eine Label:Wert-Zeile (Wert wrappt bei Bedarf).
func dsgvoZeile(p *gofpdf.Fpdf, tr func(string) string, label, wert string) {
	p.SetFont("Arial", "B", 9)
	p.CellFormat(55, 6, tr(label), "", 0, "L", false, 0, "")
	p.SetFont("Arial", "", 9)
	p.MultiCell(0, 6, tr(dsgvoLeerWert(wert)), "", "L", false)
}

// dsgvoAbsatz schreibt einen fetten Titel über einem umbrechenden Fließtext.
func dsgvoAbsatz(p *gofpdf.Fpdf, tr func(string) string, label, text string) {
	p.SetFont("Arial", "B", 9)
	p.MultiCell(0, 5, tr(label), "", "L", false)
	p.SetFont("Arial", "", 9)
	p.MultiCell(0, 5, tr(text), "", "L", false)
	p.Ln(2)
}

// dsgvoEintragTitel/-Zeile rendern einen Listeneintrag (fetter Titel + graue Detailzeile).
func dsgvoEintragTitel(p *gofpdf.Fpdf, tr func(string) string, titel string) {
	p.SetFont("Arial", "B", 9)
	p.MultiCell(0, 5, tr(dsgvoLeerWert(titel)), "", "L", false)
}

func dsgvoEintragZeile(p *gofpdf.Fpdf, tr func(string) string, text string) {
	p.SetFont("Arial", "", 8)
	p.SetTextColor(90, 90, 90)
	p.MultiCell(0, 5, tr(text), "", "L", false)
	p.SetTextColor(0, 0, 0)
	p.Ln(1)
}

func dsgvoLeer(p *gofpdf.Fpdf, tr func(string) string) {
	p.SetFont("Arial", "I", 9)
	p.SetTextColor(120, 120, 120)
	p.MultiCell(0, 5, tr("Keine Einträge vorhanden."), "", "L", false)
	p.SetTextColor(0, 0, 0)
}

func dsgvoStrPtr(s *string) string {
	if s == nil {
		return "—"
	}
	return dsgvoLeerWert(*s)
}

func dsgvoLeerWert(s string) string {
	if strings.TrimSpace(s) == "" {
		return "—"
	}
	return s
}

// dsgvoZeit und dsgvoDatum schreiben einen Zeitpunkt so, wie die Schule ihn liest: in der
// Schulzeitzone. Zeitpunkte aus der Datenbank kommen über pgx in der Zone des Prozesses an
// (time.Local), und der Container läuft in UTC. Bis zum 24.09.2026 stand deshalb jede
// Uhrzeit dieses Blatts im Sommer zwei Stunden zu früh da, ein Vorgang kurz nach
// Mitternacht am Vortag. Ein reines Datum (DATE, Mitternacht UTC) bleibt beim Umrechnen
// derselbe Tag, weil Berlin vor UTC liegt.
func dsgvoZeit(t time.Time) string {
	return t.In(schulzeit.Zone()).Format(dsgvoZeitFormat)
}

func dsgvoDatum(t time.Time) string {
	return t.In(schulzeit.Zone()).Format(dsgvoDatumFormat)
}

// dsgvoZeitPtr: Zeitpunkt oder Strich — wie dsgvoStrPtr für Texte.
func dsgvoZeitPtr(t *time.Time) string {
	if t == nil {
		return "—"
	}
	return dsgvoZeit(*t)
}

func dsgvoJaNein(b bool) string {
	if b {
		return "Ja"
	}
	return "Nein"
}

func dsgvoNotiz(n *string) string {
	if n == nil || *n == "" {
		return ""
	}
	return " · Notiz: " + *n
}
