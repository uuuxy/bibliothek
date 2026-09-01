package api

import (
	"bytes"
	"context"
	"fmt"
	"math"
	"net/http"
	"strings"
	"time"

	"bibliothek/apierrors"
	"bibliothek/pdf"
	"bibliothek/repository"

	"github.com/jung-kurt/gofpdf"
)

// berichtOrder holds one order with its line items for PDF report generation.
type berichtOrder struct {
	ID              string
	LieferantName   string
	Kundennummer    string
	Bestelldatum    time.Time
	Gesamtbetrag    float64
	AnzahlExemplare int
	Positionen      []berichtPosition
}

type berichtPosition struct {
	TitelName   string
	ISBN        string
	Menge       int
	Einzelpreis float64
}

// parseBerichtZeitraum validiert und parst die Zeitraum-Parameter (YYYY-MM-DD).
func parseBerichtZeitraum(vonStr, bisStr string) (time.Time, time.Time, error) {
	if vonStr == "" || bisStr == "" {
		return time.Time{}, time.Time{}, fmt.Errorf("parameter 'von' und 'bis' erforderlich (YYYY-MM-DD)")
	}
	von, err := time.Parse(dateFormatISO, vonStr)
	if err != nil {
		return time.Time{}, time.Time{}, fmt.Errorf("ungültiges Datum 'von': %w", err)
	}
	bis, err := time.Parse(dateFormatISO, bisStr)
	if err != nil {
		return time.Time{}, time.Time{}, fmt.Errorf("ungültiges Datum 'bis': %w", err)
	}
	return von, bis, nil
}

// berichtTitelAbleiten liefert den Berichtstitel: expliziter Titel hat Vorrang,
// sonst wird er aus dem Kontext (Lieferant / Jahresansicht) abgeleitet.
func berichtTitelAbleiten(titel, lieferantID string, jahresansicht bool) string {
	if titel != "" {
		return titel
	}
	if lieferantID != "" {
		return "Lieferantenabrechnung"
	}
	if jahresansicht {
		return "Jahresbericht"
	}
	return "Bestellbericht"
}

// ladeBestellungen liest die Bestellungen im Zeitraum (optional je Lieferant) und
// liefert zusätzlich einen Index ID→Position für das Nachladen der Positionen.
func (s *Server) ladeBestellungen(ctx context.Context, von, bisExklusiv time.Time, lieferantID string) ([]berichtOrder, map[string]int, error) {
	orderQuery := `
		SELECT id, lieferant_name, kundennummer, bestelldatum, gesamtbetrag, anzahl_exemplare
		FROM bestellungen_verlauf
		WHERE bestelldatum >= $1 AND bestelldatum < $2`
	args := []any{von, bisExklusiv}
	if lieferantID != "" {
		orderQuery += " AND lieferant_id = $3"
		args = append(args, lieferantID)
	}
	orderQuery += " ORDER BY bestelldatum ASC"

	orderRows, err := s.DB.Pool.Query(ctx, orderQuery, args...)
	if err != nil {
		return nil, nil, err
	}
	defer orderRows.Close()

	orders := make([]berichtOrder, 0)
	orderIndex := map[string]int{}
	for orderRows.Next() {
		var o berichtOrder
		if err := orderRows.Scan(&o.ID, &o.LieferantName, &o.Kundennummer,
			&o.Bestelldatum, &o.Gesamtbetrag, &o.AnzahlExemplare); err != nil {
			return nil, nil, err
		}
		orderIndex[o.ID] = len(orders)
		orders = append(orders, o)
	}
	if err := orderRows.Err(); err != nil {
		return nil, nil, err
	}
	return orders, orderIndex, nil
}

// ladeBestellPositionen lädt die Positionen aller Bestellungen nach und hängt sie
// den passenden Orders an.
func (s *Server) ladeBestellPositionen(ctx context.Context, orders []berichtOrder, orderIndex map[string]int) error {
	if len(orders) == 0 {
		return nil
	}
	ids := make([]string, len(orders))
	for i, o := range orders {
		ids[i] = o.ID
	}
	posRows, err := s.DB.Pool.Query(ctx, `
		SELECT bestellung_id, titel_name, isbn, menge, einzelpreis
		FROM bestellungen_positionen
		WHERE bestellung_id = ANY($1::uuid[])
		ORDER BY bestellung_id, titel_name`, ids)
	if err != nil {
		return err
	}
	defer posRows.Close()
	for posRows.Next() {
		var bestellungID string
		var pos berichtPosition
		if err := posRows.Scan(&bestellungID, &pos.TitelName, &pos.ISBN, &pos.Menge, &pos.Einzelpreis); err != nil {
			return err
		}
		if idx, ok := orderIndex[bestellungID]; ok {
			orders[idx].Positionen = append(orders[idx].Positionen, pos)
		}
	}
	return posRows.Err()
}

// GetBestellBerichtPDFHandler generates a printable PDF report for a date range.
//
// Query parameters:
//
//	von           YYYY-MM-DD — start date (inclusive)
//	bis           YYYY-MM-DD — end date (inclusive)
//	lieferant_id  UUID       — optional: restrict to one supplier
//	titel         string     — optional: custom title in PDF header
//	jahresansicht true       — optional: add monthly + per-supplier breakdown
func (s *Server) GetBestellBerichtPDFHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		q := r.URL.Query()

		vonStr := q.Get("von")
		bisStr := q.Get("bis")
		von, bis, err := parseBerichtZeitraum(vonStr, bisStr)
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusBadRequest, err)
			return
		}
		bisExklusiv := bis.AddDate(0, 0, 1)

		lieferantID := q.Get("lieferant_id")
		jahresansicht := q.Get("jahresansicht") == "true"
		berichtTitel := berichtTitelAbleiten(q.Get("titel"), lieferantID, jahresansicht)

		orders, orderIndex, err := s.ladeBestellungen(ctx, von, bisExklusiv, lieferantID)
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}

		if err := s.ladeBestellPositionen(ctx, orders, orderIndex); err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}

		settingsRepo := repository.NewSystemSettingsRepository(s.DB.Pool)
		settings, _ := settingsRepo.GetSettings(ctx) //nolint:errcheck
		schule := pdf.SchuleInfo{
			Name:    settings.SchuleName,
			Strasse: settings.SchuleStrasse,
			PLZ:     settings.SchulePLZ,
			Ort:     settings.SchuleOrt,
		}

		pdfBytes, err := generateBestellBerichtPDF(orders, schule, berichtTitel, von, bis, jahresansicht, settings.PreiseErfassen)
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}

		dateiname := fmt.Sprintf("bestellbericht_%s_%s.pdf", vonStr, bisStr)
		w.Header().Set(headerContentType, contentTypePDF)
		w.Header().Set(headerContentDisposition, fmt.Sprintf(`attachment; filename="%s"`, dateiname))
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(pdfBytes) //nolint:errcheck
	}
}

// --- PDF generation ---

func euroStr(n float64) string {
	rounded := math.Round(n*100) / 100
	s := fmt.Sprintf("%.2f", rounded)
	s = strings.Replace(s, ".", ",", 1)
	return s + " €" // €
}

var monthNames = [...]string{"Januar", "Februar", "März", "April", "Mai", "Juni",
	"Juli", "August", "September", "Oktober", "November", "Dezember"}

type berichtMonthStat struct {
	count, exemplare int
	betrag           float64
}

type berichtSupplierStat struct {
	count     int
	exemplare int
	betrag    float64
}

// zeichneMonatsuebersicht rendert die Tabelle "Übersicht nach Monat".
//
// Ohne Preiserfassung entfaellt die Betragsspalte, und die verbleibenden Spalten werden
// auf dieselbe Gesamtbreite verteilt — sonst endete die Tabelle mitten auf der Seite.
func zeichneMonatsuebersicht(p *gofpdf.Fpdf, tr func(string) string, orders []berichtOrder, gesamtExemplare int, gesamtBetrag float64, mitPreisen bool) {
	monthly := map[time.Month]*berichtMonthStat{}
	for _, o := range orders {
		m := o.Bestelldatum.Month()
		if monthly[m] == nil {
			monthly[m] = &berichtMonthStat{}
		}
		monthly[m].count++
		monthly[m].exemplare += o.AnzahlExemplare
		monthly[m].betrag += o.Gesamtbetrag
	}

	p.SetFont("Arial", "B", 11)
	p.Cell(0, 8, tr("Übersicht nach Monat"))
	p.Ln(8)
	p.SetFont("Arial", "B", 9)
	p.SetFillColor(220, 220, 220)
	bMonat, bAnzahl, bExemplare := 55.0, 40.0, 40.0
	if !mitPreisen {
		bMonat, bAnzahl, bExemplare = 70.0, 50.0, 50.0 // Summe bleibt 170
	}
	p.CellFormat(bMonat, 7, tr("Monat"), "1", 0, "L", true, 0, "")
	p.CellFormat(bAnzahl, 7, tr("Bestellungen"), "1", 0, "C", true, 0, "")
	if mitPreisen {
		p.CellFormat(bExemplare, 7, tr("Exemplare"), "1", 0, "C", true, 0, "")
		p.CellFormat(35, 7, tr("Betrag"), "1", 1, "R", true, 0, "")
	} else {
		p.CellFormat(bExemplare, 7, tr("Exemplare"), "1", 1, "C", true, 0, "")
	}
	p.SetFont("Arial", "", 9)
	p.SetFillColor(255, 255, 255)
	for m := time.January; m <= time.December; m++ {
		if stat, ok := monthly[m]; ok {
			p.CellFormat(bMonat, 6, tr(monthNames[m-1]), "1", 0, "L", false, 0, "")
			p.CellFormat(bAnzahl, 6, fmt.Sprintf("%d", stat.count), "1", 0, "C", false, 0, "")
			if mitPreisen {
				p.CellFormat(bExemplare, 6, fmt.Sprintf("%d", stat.exemplare), "1", 0, "C", false, 0, "")
				p.CellFormat(35, 6, tr(euroStr(stat.betrag)), "1", 1, "R", false, 0, "")
			} else {
				p.CellFormat(bExemplare, 6, fmt.Sprintf("%d", stat.exemplare), "1", 1, "C", false, 0, "")
			}
		}
	}
	p.SetFont("Arial", "B", 9)
	p.SetFillColor(240, 240, 240)
	p.CellFormat(bMonat, 7, tr("Gesamt"), "1", 0, "L", true, 0, "")
	p.CellFormat(bAnzahl, 7, fmt.Sprintf("%d", len(orders)), "1", 0, "C", true, 0, "")
	if mitPreisen {
		p.CellFormat(bExemplare, 7, fmt.Sprintf("%d", gesamtExemplare), "1", 0, "C", true, 0, "")
		p.CellFormat(35, 7, tr(euroStr(gesamtBetrag)), "1", 1, "R", true, 0, "")
	} else {
		p.CellFormat(bExemplare, 7, fmt.Sprintf("%d", gesamtExemplare), "1", 1, "C", true, 0, "")
	}
	p.SetFillColor(255, 255, 255)
	p.Ln(10)
}

// zeichneLieferantenuebersicht rendert die Aufteilung nach Lieferant.
//
// Ohne Preiserfassung heisst sie nicht mehr "Ausgaben" — es sind dann keine. Statt des
// Betrags stehen die Exemplare, denn die Frage "wie viel haben wir wo bezogen" bleibt
// auch ohne Geld sinnvoll.
func zeichneLieferantenuebersicht(p *gofpdf.Fpdf, tr func(string) string, orders []berichtOrder, mitPreisen bool) {
	bySupplier := map[string]*berichtSupplierStat{}
	for _, o := range orders {
		if bySupplier[o.LieferantName] == nil {
			bySupplier[o.LieferantName] = &berichtSupplierStat{}
		}
		bySupplier[o.LieferantName].count++
		bySupplier[o.LieferantName].exemplare += o.AnzahlExemplare
		bySupplier[o.LieferantName].betrag += o.Gesamtbetrag
	}
	ueberschrift := "Ausgaben nach Lieferant"
	if !mitPreisen {
		ueberschrift = "Bestellungen nach Lieferant"
	}
	p.SetFont("Arial", "B", 11)
	p.Cell(0, 8, tr(ueberschrift))
	p.Ln(8)
	p.SetFont("Arial", "B", 9)
	p.SetFillColor(220, 220, 220)
	p.CellFormat(85, 7, tr("Lieferant"), "1", 0, "L", true, 0, "")
	p.CellFormat(40, 7, tr("Bestellungen"), "1", 0, "C", true, 0, "")
	if mitPreisen {
		p.CellFormat(45, 7, tr("Betrag"), "1", 1, "R", true, 0, "")
	} else {
		p.CellFormat(45, 7, tr("Exemplare"), "1", 1, "C", true, 0, "")
	}
	p.SetFont("Arial", "", 9)
	p.SetFillColor(255, 255, 255)
	for name, stat := range bySupplier {
		p.CellFormat(85, 6, tr(name), "1", 0, "L", false, 0, "")
		p.CellFormat(40, 6, fmt.Sprintf("%d", stat.count), "1", 0, "C", false, 0, "")
		if mitPreisen {
			p.CellFormat(45, 6, tr(euroStr(stat.betrag)), "1", 1, "R", false, 0, "")
		} else {
			p.CellFormat(45, 6, fmt.Sprintf("%d", stat.exemplare), "1", 1, "C", false, 0, "")
		}
	}
	p.Ln(10)
}

// berichtSpalten sind die Spaltenbreiten der Positionstabelle. Sie hängen davon ab, ob
// der Bericht Preise ausweist: Ohne Preisspalten verteilen sich Titel/ISBN/Menge auf
// dieselben 170 mm.
type berichtSpalten struct {
	Titel, ISBN, Menge float64
	TitelKuerzung      int // Zeichen, nicht Bytes — Umlaute zählen einfach
}

func spaltenFuerBericht(mitPreisen bool) berichtSpalten {
	if mitPreisen {
		return berichtSpalten{Titel: 83, ISBN: 33, Menge: 14, TitelKuerzung: 62}
	}
	// Breitere Titelspalte trägt mehr Text.
	return berichtSpalten{Titel: 103, ISBN: 43, Menge: 24, TitelKuerzung: 78}
}

// berichtRahmen bündelt die Angaben, die für den GANZEN Bericht gelten. Vorher standen
// sie als fünf Einzelparameter in der Signatur (S107: 8 Parameter) — und dieselbe Liste
// hätte jeder ausgelagerte Helfer erneut durchreichen müssen.
type berichtRahmen struct {
	Von, Bis        time.Time
	GesamtBetrag    float64
	GesamtExemplare int
	MitPreisen      bool
}

// zeichneBestellKopf setzt die Kopfzeile einer einzelnen Bestellung (Datum, Lieferant,
// Kundennummer).
func zeichneBestellKopf(p *gofpdf.Fpdf, tr func(string) string, o berichtOrder) {
	p.SetFont("Arial", "B", 9)
	p.SetFillColor(235, 235, 245)
	header := fmt.Sprintf("%s  ·  %s  ·  Kd.-Nr. %s",
		o.Bestelldatum.Format(dateFormatDE), o.LieferantName, o.Kundennummer)
	p.CellFormat(0, 7, tr(header), "LTR", 1, "L", true, 0, "")
	p.SetFillColor(255, 255, 255)
}

// zeichneSpaltenkoepfe setzt die Überschriftenzeile der Positionstabelle.
func zeichneSpaltenkoepfe(p *gofpdf.Fpdf, tr func(string) string, s berichtSpalten, mitPreisen bool) {
	p.SetFont("Arial", "B", 8)
	p.SetFillColor(245, 245, 245)
	p.CellFormat(s.Titel, 6, tr("Titel"), "1", 0, "L", true, 0, "")
	p.CellFormat(s.ISBN, 6, tr("ISBN"), "1", 0, "C", true, 0, "")
	if mitPreisen {
		p.CellFormat(s.Menge, 6, tr("Menge"), "1", 0, "C", true, 0, "")
		p.CellFormat(20, 6, tr("Einzelpr."), "1", 0, "R", true, 0, "")
		p.CellFormat(20, 6, tr("Gesamt"), "1", 1, "R", true, 0, "")
	} else {
		p.CellFormat(s.Menge, 6, tr("Menge"), "1", 1, "C", true, 0, "")
	}
	p.SetFillColor(255, 255, 255)
}

// zeichnePositionen setzt die Titelzeilen einer Bestellung, mit Seitenumbruch am Fuß.
func zeichnePositionen(p *gofpdf.Fpdf, tr func(string) string, positionen []berichtPosition, s berichtSpalten, mitPreisen bool) {
	p.SetFont("Arial", "", 8)
	for _, pos := range positionen {
		if p.GetY() > 265 {
			p.AddPage()
		}
		isbn := pos.ISBN
		if isbn == "" {
			isbn = "—"
		}
		p.CellFormat(s.Titel, 5, tr(kuerzeAufZeichen(pos.TitelName, s.TitelKuerzung)), "1", 0, "L", false, 0, "")
		p.CellFormat(s.ISBN, 5, tr(isbn), "1", 0, "C", false, 0, "")
		if mitPreisen {
			p.CellFormat(s.Menge, 5, fmt.Sprintf("%d", pos.Menge), "1", 0, "C", false, 0, "")
			p.CellFormat(20, 5, tr(euroStr(pos.Einzelpreis)), "1", 0, "R", false, 0, "")
			p.CellFormat(20, 5, tr(euroStr(float64(pos.Menge)*pos.Einzelpreis)), "1", 1, "R", false, 0, "")
		} else {
			p.CellFormat(s.Menge, 5, fmt.Sprintf("%d", pos.Menge), "1", 1, "C", false, 0, "")
		}
	}
}

// zeichneBestellSumme setzt die Abschlusszeile einer einzelnen Bestellung.
func zeichneBestellSumme(p *gofpdf.Fpdf, tr func(string) string, o berichtOrder, mitPreisen bool) {
	p.SetFont("Arial", "B", 8)
	p.SetFillColor(235, 235, 245)
	if mitPreisen {
		p.CellFormat(150, 6, tr(fmt.Sprintf("Summe (%d Exemplare)", o.AnzahlExemplare)), "LBR", 0, "R", true, 0, "")
		p.CellFormat(20, 6, tr(euroStr(o.Gesamtbetrag)), "1", 1, "R", true, 0, "")
	} else {
		p.CellFormat(170, 6, tr(fmt.Sprintf("Summe: %d Exemplare", o.AnzahlExemplare)), "LBR", 1, "R", true, 0, "")
	}
	p.SetFillColor(255, 255, 255)
	p.Ln(5)
}

// zeichneGesamtsumme setzt die Schlusszeile über alle Bestellungen des Zeitraums.
func zeichneGesamtsumme(p *gofpdf.Fpdf, tr func(string) string, r berichtRahmen) {
	p.Ln(4)
	p.SetFont("Arial", "B", 11)
	p.SetFillColor(220, 230, 255)
	zeitraum := fmt.Sprintf("%s – %s", r.Von.Format(dateFormatDE), r.Bis.Format(dateFormatDE))
	if r.MitPreisen {
		p.CellFormat(150, 9, tr("Gesamtbetrag "+zeitraum), "1", 0, "R", true, 0, "")
		p.CellFormat(20, 9, tr(euroStr(r.GesamtBetrag)), "1", 1, "R", true, 0, "")
	} else {
		p.CellFormat(150, 9, tr("Gesamt "+zeitraum), "1", 0, "R", true, 0, "")
		p.CellFormat(20, 9, fmt.Sprintf("%d", r.GesamtExemplare), "1", 1, "R", true, 0, "")
	}
	p.SetFillColor(255, 255, 255)
}

// zeichneDetailliste rendert die Bestellungen mit ihren Positionen und den
// Gesamtbetrag.
//
// Die vier Abschnitte (Kopf, Spaltenköpfe, Positionen, Summe) liegen bewusst als
// EIGENSTÄNDIGE Funktionen daneben und nicht als Closures darin: In Go zählt eine
// Closure als zusätzliche Verschachtelungsebene, ein Auslagern nach innen hätte die
// kognitive Komplexität also nicht gesenkt (Projekt-Erfahrung aus früheren
// S3776-Refactorings).
func zeichneDetailliste(p *gofpdf.Fpdf, tr func(string) string, orders []berichtOrder, r berichtRahmen) {
	if len(orders) == 0 {
		p.SetFont("Arial", "I", 10)
		p.SetTextColor(120, 120, 120)
		p.Cell(0, 8, tr("Keine Bestellungen im gewählten Zeitraum."))
		p.SetTextColor(0, 0, 0)
		return
	}

	p.SetFont("Arial", "B", 11)
	p.Cell(0, 8, tr("Bestellungen im Detail"))
	p.Ln(8)

	spalten := spaltenFuerBericht(r.MitPreisen)
	for _, o := range orders {
		if p.GetY() > 240 {
			p.AddPage()
		}
		zeichneBestellKopf(p, tr, o)
		zeichneSpaltenkoepfe(p, tr, spalten, r.MitPreisen)
		zeichnePositionen(p, tr, o.Positionen, spalten, r.MitPreisen)
		zeichneBestellSumme(p, tr, o, r.MitPreisen)
	}

	zeichneGesamtsumme(p, tr, r)
}

// generateBestellBerichtPDF baut den Bericht.
//
// mitPreisen entscheidet, ob er ein Geld- oder ein Mengenbericht ist. Das ist keine
// Formatierungsfrage: Ohne gepflegte Preise summierte der Bericht durchweg Nullen und sah
// dabei aus wie ein Ausgabennachweis. Ein Nachweis, der Null behauptet, ist schlimmer als
// keiner — er wird geglaubt.
func generateBestellBerichtPDF(orders []berichtOrder, schule pdf.SchuleInfo, titel string, von, bis time.Time, jahresansicht, mitPreisen bool) ([]byte, error) {
	p := gofpdf.New("P", "mm", "A4", "")
	p.SetMargins(20, 20, 20)
	p.SetAutoPageBreak(true, 20)
	tr := p.UnicodeTranslatorFromDescriptor("")
	p.AddPage()

	// Briefkopf
	p.SetFont("Arial", "B", 12)
	p.Cell(0, 8, tr(schule.Name))
	p.Ln(5)
	p.SetFont("Arial", "", 8)
	p.SetTextColor(100, 100, 100)
	p.Cell(0, 4, tr(schule.Absenderzeile()))
	p.SetTextColor(0, 0, 0)
	p.Ln(12)

	p.SetFont("Arial", "", 10)
	p.CellFormat(0, 6, tr(schule.OrtDatum(time.Now().Format(dateFormatDE))), "", 1, "R", false, 0, "")
	p.Ln(6)

	// Berichtstitel
	p.SetFont("Arial", "B", 14)
	p.Cell(0, 10, tr(titel))
	p.Ln(7)
	p.SetFont("Arial", "", 10)
	p.SetTextColor(80, 80, 80)
	p.Cell(0, 6, tr(fmt.Sprintf("Zeitraum: %s bis %s", von.Format(dateFormatDE), bis.Format(dateFormatDE))))
	p.SetTextColor(0, 0, 0)
	p.Ln(12)

	// Zusammenfassung
	var gesamtBetrag float64
	var gesamtExemplare int
	for _, o := range orders {
		gesamtBetrag += o.Gesamtbetrag
		gesamtExemplare += o.AnzahlExemplare
	}

	p.SetFillColor(240, 245, 255)
	p.SetFont("Arial", "B", 9)
	if mitPreisen {
		p.CellFormat(60, 10, tr(fmt.Sprintf("Bestellungen: %d", len(orders))), "1", 0, "C", true, 0, "")
		p.CellFormat(60, 10, tr(fmt.Sprintf("Exemplare: %d", gesamtExemplare)), "1", 0, "C", true, 0, "")
		p.CellFormat(50, 10, tr("Gesamtbetrag: "+euroStr(gesamtBetrag)), "1", 1, "C", true, 0, "")
	} else {
		p.CellFormat(85, 10, tr(fmt.Sprintf("Bestellungen: %d", len(orders))), "1", 0, "C", true, 0, "")
		p.CellFormat(85, 10, tr(fmt.Sprintf("Exemplare: %d", gesamtExemplare)), "1", 1, "C", true, 0, "")
	}
	p.SetFillColor(255, 255, 255)
	p.Ln(10)

	// Jahresübersicht-Tabellen
	if jahresansicht {
		zeichneMonatsuebersicht(p, tr, orders, gesamtExemplare, gesamtBetrag, mitPreisen)
		zeichneLieferantenuebersicht(p, tr, orders, mitPreisen)
	}

	// Detailliste
	zeichneDetailliste(p, tr, orders, berichtRahmen{
		Von: von, Bis: bis,
		GesamtBetrag: gesamtBetrag, GesamtExemplare: gesamtExemplare,
		MitPreisen: mitPreisen,
	})

	var buf bytes.Buffer
	if err := p.Output(&buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// kuerzeAufZeichen kürzt s auf höchstens max ZEICHEN (das Auslassungszeichen
// eingerechnet) — nicht auf max Bytes.
//
// Der Unterschied ist in einer deutschen Schulbibliothek kein Randfall: len(s) und
// s[:n] rechnen in Bytes, ein Umlaut belegt in UTF-8 aber zwei. Ein Schnitt mitten
// durch „ä" hinterlässt ein halbes Zeichen, und der Unicode-Übersetzer von gofpdf
// macht daraus sichtbaren Zeichensalat — auf einem Schreiben, das an Eltern geht.
//
// max ist die Gesamtlänge der Ausgabe, damit sich Spaltenbreiten direkt ablesen lassen.
func kuerzeAufZeichen(s string, max int) string {
	runes := []rune(s)
	if len(runes) <= max {
		return s
	}
	return string(runes[:max-1]) + "…"
}
