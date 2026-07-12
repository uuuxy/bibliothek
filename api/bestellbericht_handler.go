package api

import (
	"bytes"
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
	LieferantEmail  string
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
		if vonStr == "" || bisStr == "" {
			apierrors.SendHTTPError(w, http.StatusBadRequest, fmt.Errorf("parameter 'von' und 'bis' erforderlich (YYYY-MM-DD)"))
			return
		}

		von, err := time.Parse("2006-01-02", vonStr)
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusBadRequest, fmt.Errorf("ungültiges Datum 'von': %w", err))
			return
		}
		bis, err := time.Parse("2006-01-02", bisStr)
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusBadRequest, fmt.Errorf("ungültiges Datum 'bis': %w", err))
			return
		}
		bisExklusiv := bis.AddDate(0, 0, 1)

		lieferantID := q.Get("lieferant_id")
		berichtTitel := q.Get("titel")
		jahresansicht := q.Get("jahresansicht") == "true"

		if berichtTitel == "" {
			if lieferantID != "" {
				berichtTitel = "Lieferantenabrechnung"
			} else if jahresansicht {
				berichtTitel = "Jahresbericht"
			} else {
				berichtTitel = "Bestellbericht"
			}
		}

		// Build query dynamically so we share one code path
		orderQuery := `
			SELECT id, lieferant_name, lieferant_email, kundennummer, bestelldatum, gesamtbetrag, anzahl_exemplare
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
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}
		defer orderRows.Close()

		orders := make([]berichtOrder, 0)
		orderIndex := map[string]int{}
		for orderRows.Next() {
			var o berichtOrder
			if err := orderRows.Scan(&o.ID, &o.LieferantName, &o.LieferantEmail, &o.Kundennummer,
				&o.Bestelldatum, &o.Gesamtbetrag, &o.AnzahlExemplare); err != nil {
				apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
				return
			}
			orderIndex[o.ID] = len(orders)
			orders = append(orders, o)
		}
		if err := orderRows.Err(); err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}

		// Load positionen for all found orders
		if len(orders) > 0 {
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
				apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
				return
			}
			defer posRows.Close()
			for posRows.Next() {
				var bestellungID string
				var pos berichtPosition
				if err := posRows.Scan(&bestellungID, &pos.TitelName, &pos.ISBN, &pos.Menge, &pos.Einzelpreis); err != nil {
					apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
					return
				}
				if idx, ok := orderIndex[bestellungID]; ok {
					orders[idx].Positionen = append(orders[idx].Positionen, pos)
				}
			}
			if err := posRows.Err(); err != nil {
				apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
				return
			}
		}

		settingsRepo := repository.NewSystemSettingsRepository(s.DB.Pool)
		settings, _ := settingsRepo.GetSettings(ctx)
		schule := pdf.SchuleInfo{
			Name:    settings.SchuleName,
			Strasse: settings.SchuleStrasse,
			PLZ:     settings.SchulePLZ,
			Ort:     settings.SchuleOrt,
		}

		pdfBytes, err := generateBestellBerichtPDF(orders, schule, berichtTitel, von, bis, jahresansicht)
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}

		dateiname := fmt.Sprintf("bestellbericht_%s_%s.pdf", vonStr, bisStr)
		w.Header().Set("Content-Type", "application/pdf")
		w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, dateiname))
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(pdfBytes)
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

func generateBestellBerichtPDF(orders []berichtOrder, schule pdf.SchuleInfo, titel string, von, bis time.Time, jahresansicht bool) ([]byte, error) {
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
	p.CellFormat(60, 10, tr(fmt.Sprintf("Bestellungen: %d", len(orders))), "1", 0, "C", true, 0, "")
	p.CellFormat(60, 10, tr(fmt.Sprintf("Exemplare: %d", gesamtExemplare)), "1", 0, "C", true, 0, "")
	p.CellFormat(50, 10, tr("Gesamtbetrag: "+euroStr(gesamtBetrag)), "1", 1, "C", true, 0, "")
	p.SetFillColor(255, 255, 255)
	p.Ln(10)

	// Jahresübersicht-Tabellen
	if jahresansicht {
		type monthStat struct{ count, exemplare int; betrag float64 }
		monthly := map[time.Month]*monthStat{}
		for _, o := range orders {
			m := o.Bestelldatum.Month()
			if monthly[m] == nil {
				monthly[m] = &monthStat{}
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
		p.CellFormat(55, 7, tr("Monat"), "1", 0, "L", true, 0, "")
		p.CellFormat(40, 7, tr("Bestellungen"), "1", 0, "C", true, 0, "")
		p.CellFormat(40, 7, tr("Exemplare"), "1", 0, "C", true, 0, "")
		p.CellFormat(35, 7, tr("Betrag"), "1", 1, "R", true, 0, "")
		p.SetFont("Arial", "", 9)
		p.SetFillColor(255, 255, 255)
		for m := time.January; m <= time.December; m++ {
			if stat, ok := monthly[m]; ok {
				p.CellFormat(55, 6, tr(monthNames[m-1]), "1", 0, "L", false, 0, "")
				p.CellFormat(40, 6, fmt.Sprintf("%d", stat.count), "1", 0, "C", false, 0, "")
				p.CellFormat(40, 6, fmt.Sprintf("%d", stat.exemplare), "1", 0, "C", false, 0, "")
				p.CellFormat(35, 6, tr(euroStr(stat.betrag)), "1", 1, "R", false, 0, "")
			}
		}
		p.SetFont("Arial", "B", 9)
		p.SetFillColor(240, 240, 240)
		p.CellFormat(55, 7, tr("Gesamt"), "1", 0, "L", true, 0, "")
		p.CellFormat(40, 7, fmt.Sprintf("%d", len(orders)), "1", 0, "C", true, 0, "")
		p.CellFormat(40, 7, fmt.Sprintf("%d", gesamtExemplare), "1", 0, "C", true, 0, "")
		p.CellFormat(35, 7, tr(euroStr(gesamtBetrag)), "1", 1, "R", true, 0, "")
		p.SetFillColor(255, 255, 255)
		p.Ln(10)

		// Aufteilung nach Lieferant
		type supplierStat struct{ count int; betrag float64 }
		bySupplier := map[string]*supplierStat{}
		for _, o := range orders {
			if bySupplier[o.LieferantName] == nil {
				bySupplier[o.LieferantName] = &supplierStat{}
			}
			bySupplier[o.LieferantName].count++
			bySupplier[o.LieferantName].betrag += o.Gesamtbetrag
		}
		p.SetFont("Arial", "B", 11)
		p.Cell(0, 8, tr("Ausgaben nach Lieferant"))
		p.Ln(8)
		p.SetFont("Arial", "B", 9)
		p.SetFillColor(220, 220, 220)
		p.CellFormat(85, 7, tr("Lieferant"), "1", 0, "L", true, 0, "")
		p.CellFormat(40, 7, tr("Bestellungen"), "1", 0, "C", true, 0, "")
		p.CellFormat(45, 7, tr("Betrag"), "1", 1, "R", true, 0, "")
		p.SetFont("Arial", "", 9)
		p.SetFillColor(255, 255, 255)
		for name, stat := range bySupplier {
			p.CellFormat(85, 6, tr(name), "1", 0, "L", false, 0, "")
			p.CellFormat(40, 6, fmt.Sprintf("%d", stat.count), "1", 0, "C", false, 0, "")
			p.CellFormat(45, 6, tr(euroStr(stat.betrag)), "1", 1, "R", false, 0, "")
		}
		p.Ln(10)
	}

	// Detailliste
	if len(orders) == 0 {
		p.SetFont("Arial", "I", 10)
		p.SetTextColor(120, 120, 120)
		p.Cell(0, 8, tr("Keine Bestellungen im gewählten Zeitraum."))
		p.SetTextColor(0, 0, 0)
	} else {
		p.SetFont("Arial", "B", 11)
		p.Cell(0, 8, tr("Bestellungen im Detail"))
		p.Ln(8)

		for _, o := range orders {
			if p.GetY() > 240 {
				p.AddPage()
			}
			// Bestellkopf
			p.SetFont("Arial", "B", 9)
			p.SetFillColor(235, 235, 245)
			header := fmt.Sprintf("%s  ·  %s  ·  Kd.-Nr. %s",
				o.Bestelldatum.Format(dateFormatDE), o.LieferantName, o.Kundennummer)
			p.CellFormat(0, 7, tr(header), "LTR", 1, "L", true, 0, "")
			p.SetFillColor(255, 255, 255)

			// Spaltenköpfe
			p.SetFont("Arial", "B", 8)
			p.SetFillColor(245, 245, 245)
			p.CellFormat(83, 6, tr("Titel"), "1", 0, "L", true, 0, "")
			p.CellFormat(33, 6, tr("ISBN"), "1", 0, "C", true, 0, "")
			p.CellFormat(14, 6, tr("Menge"), "1", 0, "C", true, 0, "")
			p.CellFormat(20, 6, tr("Einzelpr."), "1", 0, "R", true, 0, "")
			p.CellFormat(20, 6, tr("Gesamt"), "1", 1, "R", true, 0, "")
			p.SetFillColor(255, 255, 255)

			p.SetFont("Arial", "", 8)
			for _, pos := range o.Positionen {
				if p.GetY() > 265 {
					p.AddPage()
				}
				isbn := pos.ISBN
				if isbn == "" {
					isbn = "—"
				}
				p.CellFormat(83, 5, tr(berichtTrunc(pos.TitelName, 62)), "1", 0, "L", false, 0, "")
				p.CellFormat(33, 5, tr(isbn), "1", 0, "C", false, 0, "")
				p.CellFormat(14, 5, fmt.Sprintf("%d", pos.Menge), "1", 0, "C", false, 0, "")
				p.CellFormat(20, 5, tr(euroStr(pos.Einzelpreis)), "1", 0, "R", false, 0, "")
				p.CellFormat(20, 5, tr(euroStr(float64(pos.Menge)*pos.Einzelpreis)), "1", 1, "R", false, 0, "")
			}

			// Summe der Bestellung
			p.SetFont("Arial", "B", 8)
			p.SetFillColor(235, 235, 245)
			p.CellFormat(150, 6, tr(fmt.Sprintf("Summe (%d Exemplare)", o.AnzahlExemplare)), "LBR", 0, "R", true, 0, "")
			p.CellFormat(20, 6, tr(euroStr(o.Gesamtbetrag)), "1", 1, "R", true, 0, "")
			p.SetFillColor(255, 255, 255)
			p.Ln(5)
		}

		// Gesamtbetrag
		p.Ln(4)
		p.SetFont("Arial", "B", 11)
		p.SetFillColor(220, 230, 255)
		p.CellFormat(150, 9, tr(fmt.Sprintf("Gesamtbetrag %s – %s", von.Format(dateFormatDE), bis.Format(dateFormatDE))), "1", 0, "R", true, 0, "")
		p.CellFormat(20, 9, tr(euroStr(gesamtBetrag)), "1", 1, "R", true, 0, "")
		p.SetFillColor(255, 255, 255)
	}

	var buf bytes.Buffer
	if err := p.Output(&buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func berichtTrunc(s string, max int) string {
	runes := []rune(s)
	if len(runes) <= max {
		return s
	}
	return string(runes[:max-1]) + "…"
}
