package api

import (
	"bytes"
	"fmt"
	"net/http"
	"strings"
	"time"

	"bibliothek/apierrors"
	"bibliothek/pdf"
	"bibliothek/repository"

	"github.com/jung-kurt/gofpdf"
)

const (
	dsgvoDatumFormat = "02.01.2006"
	dsgvoZeitFormat  = "02.01.2006, 15:04 Uhr"
)

// DsgvoAuskunftPDFHandler liefert die Betroffenenauskunft nach Art. 15 DSGVO als
// lesbares PDF. Inhaltlich identisch zur JSON-Variante (dieselbe sammleDsgvoDaten-
// Quelle), nur menschenlesbar aufbereitet — gedacht zur Aushändigung an Schüler bzw.
// Erziehungsberechtigte (JSON ist für diesen Zweck ungeeignet). Die Erteilung wird
// wie bei der JSON-Auskunft im Audit-Log protokolliert (Rechenschaftspflicht).
// @Summary      DSGVO-Betroffenenauskunft (Art. 15) als PDF
// @Tags         students
// @Produce      application/pdf
// @Param        id   path      string  true  "Student ID (UUID)"
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

		pdfBytes, err := generateDsgvoAuskunftPDF(daten, schule)
		if err != nil {
			return apierrors.Internal("PDF-Erzeugung fehlgeschlagen", err)
		}

		w.Header().Set(headerContentType, contentTypePDF)
		w.Header().Set(headerContentDisposition, `attachment; filename="DSGVO-Auskunft.pdf"`)
		w.Header().Set(headerContentLength, fmt.Sprint(len(pdfBytes)))
		http.ServeContent(w, r, "DSGVO-Auskunft.pdf", time.Now(), bytes.NewReader(pdfBytes))
		return nil
	})
}

// generateDsgvoAuskunftPDF rendert die vollständige Art.-15-Auskunft als PDF.
func generateDsgvoAuskunftPDF(daten *dsgvoDaten, schule pdf.SchuleInfo) ([]byte, error) {
	p := gofpdf.New("P", "mm", "A4", "")
	p.SetMargins(20, 20, 20)
	p.SetAutoPageBreak(true, 20)
	tr := p.UnicodeTranslatorFromDescriptor("") // UTF-8 → Latin-1 für Umlaute
	p.AddPage()

	dsgvoKopf(p, tr, schule, daten.stammdaten)
	dsgvoStammdatenAbschnitt(p, tr, daten.stammdaten)
	dsgvoFotoAbschnitt(p, tr, daten.foto)
	dsgvoAusleihAbschnitt(p, tr, daten.ausleihen)
	dsgvoSchadensAbschnitt(p, tr, daten.schaeden)
	dsgvoVormerkAbschnitt(p, tr, daten.vormerkungen)
	dsgvoAuditAbschnitt(p, tr, daten.auditEintraege)
	dsgvoVerarbeitungAbschnitt(p, tr)

	var buf bytes.Buffer
	if err := p.Output(&buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func dsgvoKopf(p *gofpdf.Fpdf, tr func(string) string, schule pdf.SchuleInfo, st *DsgvoStammdaten) {
	p.SetFont("Arial", "B", 14)
	p.Cell(0, 8, tr(schule.Name))
	p.Ln(7)
	p.SetFont("Arial", "", 8)
	p.SetTextColor(100, 100, 100)
	p.Cell(0, 4, tr(schule.Absenderzeile()))
	p.SetTextColor(0, 0, 0)
	p.Ln(11)

	p.SetFont("Arial", "B", 16)
	p.Cell(0, 10, tr("Auskunft nach Art. 15 DSGVO"))
	p.Ln(9)
	p.SetFont("Arial", "", 10)
	p.SetTextColor(90, 90, 90)
	p.Cell(0, 6, tr(fmt.Sprintf("Betroffene Person: %s %s (Klasse %s)", st.Vorname, st.Nachname, st.Klasse)))
	p.Ln(5)
	p.Cell(0, 6, tr("Erstellt am: "+time.Now().Format(dsgvoZeitFormat)))
	p.SetTextColor(0, 0, 0)
	p.Ln(9)
	p.SetFont("Arial", "", 9)
	p.MultiCell(0, 5, tr("Diese Auskunft enthält alle zu der oben genannten Person in diesem System "+
		"gespeicherten personenbezogenen Daten sowie die Pflichtangaben nach Art. 15 Abs. 1 DSGVO."), "", "L", false)
}

func dsgvoStammdatenAbschnitt(p *gofpdf.Fpdf, tr func(string) string, st *DsgvoStammdaten) {
	dsgvoAbschnitt(p, tr, "1. Stammdaten")
	dsgvoZeile(p, tr, "Interne ID", st.ID)
	dsgvoZeile(p, tr, "Ausweis-Barcode", st.BarcodeID)
	dsgvoZeile(p, tr, "Vorname", st.Vorname)
	dsgvoZeile(p, tr, "Nachname", st.Nachname)
	dsgvoZeile(p, tr, "Klasse", st.Klasse)
	dsgvoZeile(p, tr, "Geburtsdatum", dsgvoStrPtr(st.Geburtsdatum))
	dsgvoZeile(p, tr, "Abgangsjahr", fmt.Sprint(st.AbgaengerJahr))
	dsgvoZeile(p, tr, "LUSD-ID", dsgvoStrPtr(st.LusdID))
	dsgvoZeile(p, tr, "Straße/Nr.", strings.TrimSpace(st.Strasse+" "+st.Hausnummer))
	dsgvoZeile(p, tr, "PLZ/Ort", strings.TrimSpace(st.Plz+" "+st.Ort))
	dsgvoZeile(p, tr, "Eltern-E-Mail", st.ElternEmail)
	dsgvoZeile(p, tr, "Gesperrt", dsgvoJaNein(st.IstGesperrt))
	dsgvoZeile(p, tr, "Manuell gesperrt", dsgvoJaNein(st.IsManuallyBlocked))
	dsgvoZeile(p, tr, "Sperrgrund", dsgvoStrPtr(st.BlockReason))
	dsgvoZeile(p, tr, "Abgänger", dsgvoJaNein(st.IstAbgaenger))
	dsgvoZeile(p, tr, "Erfasst am", st.ErstelltAm.Format(dsgvoZeitFormat))
	dsgvoZeile(p, tr, "Zuletzt geändert", st.AktualisiertAm.Format(dsgvoZeitFormat))
	if st.GeloeschtAm != nil {
		dsgvoZeile(p, tr, "Gelöscht am (Papierkorb)", st.GeloeschtAm.Format(dsgvoZeitFormat))
	}
}

func dsgvoFotoAbschnitt(p *gofpdf.Fpdf, tr func(string) string, foto DsgvoFoto) {
	dsgvoAbschnitt(p, tr, "2. Ausweisfoto")
	dsgvoZeile(p, tr, "Vorhanden", dsgvoJaNein(foto.Vorhanden))
	if foto.AktualisiertAm != nil {
		dsgvoZeile(p, tr, "Aktualisiert am", foto.AktualisiertAm.Format(dsgvoZeitFormat))
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
			rueck = a.RueckgabeAm.Format(dsgvoDatumFormat)
		}
		dsgvoEintragTitel(p, tr, a.Gegenstand)
		dsgvoEintragZeile(p, tr, fmt.Sprintf("Barcode: %s · ausgeliehen: %s · Frist: %s · zurückgegeben: %s",
			dsgvoLeerWert(a.Barcode), a.AusgeliehenAm.Format(dsgvoDatumFormat),
			a.RueckgabeFrist.Format(dsgvoDatumFormat), rueck))
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
		if f.StorniertAm != nil {
			status = "storniert"
		}
		dsgvoEintragTitel(p, tr, f.Beschreibung)
		dsgvoEintragZeile(p, tr, fmt.Sprintf("Betrag: %s EUR · Status: %s · gemeldet: %s",
			f.Betrag, status, f.ErstelltAm.Format(dsgvoDatumFormat)))
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
			v.Status, v.ErstelltAm.Format(dsgvoDatumFormat), dsgvoNotiz(v.Notiz)))
	}
}

func dsgvoAuditAbschnitt(p *gofpdf.Fpdf, tr func(string) string, audit []DsgvoAuditEintrag) {
	dsgvoAbschnitt(p, tr, fmt.Sprintf("6. Protokolleinträge zu diesem Datensatz (%d)", len(audit)))
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
			e.Zeitpunkt.Format(dsgvoZeitFormat), e.Aktion, e.Akteur, kontext)), "", "L", false)
	}
}

func dsgvoVerarbeitungAbschnitt(p *gofpdf.Fpdf, tr func(string) string) {
	va := dsgvoVerarbeitungsangaben()
	dsgvoAbschnitt(p, tr, "7. Angaben zur Verarbeitung (Art. 15 Abs. 1 DSGVO)")
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
