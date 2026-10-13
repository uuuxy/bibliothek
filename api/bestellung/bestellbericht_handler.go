package bestellung

import (
	"fmt"
	"net/http"
	"time"

	"bibliothek/api/tuer"
	"bibliothek/apierrors"
	"bibliothek/pdf"
	"bibliothek/pkg/mitteltopf"
	"bibliothek/pkg/schulzeit"
	"bibliothek/repository"
)

// parseBerichtZeitraum liest die zwei Tage des Zeitraums (JJJJ-MM-TT) als Kalendertage der
// Schule; beide sind Pflicht. In UTC gelesen zählte eine Bestellung aus der ersten Stunde eines
// Monats zum Monat davor.
func parseBerichtZeitraum(vonStr, bisStr string) (time.Time, time.Time, error) {
	if vonStr == "" || bisStr == "" {
		return time.Time{}, time.Time{}, fmt.Errorf("parameter 'von' und 'bis' erforderlich (YYYY-MM-DD)")
	}
	von, err := schulzeit.Kalendertag(vonStr)
	if err != nil {
		return time.Time{}, time.Time{}, fmt.Errorf("ungültiges Datum 'von': %w", err)
	}
	bis, err := schulzeit.Kalendertag(bisStr)
	if err != nil {
		return time.Time{}, time.Time{}, fmt.Errorf("ungültiges Datum 'bis': %w", err)
	}
	return von, bis, nil
}

// berichtTitelAbleiten liefert die Überschrift des Berichts: Ein mitgegebener Titel geht vor,
// sonst folgt sie aus Lieferant und Jahresansicht.
func berichtTitelAbleiten(titel, lieferantID string, jahresansicht bool, mittel string) string {
	if titel == "" {
		switch {
		case lieferantID != "":
			titel = "Lieferantenabrechnung"
		case jahresansicht:
			titel = "Jahresbericht"
		default:
			titel = "Bestellbericht"
		}
	}
	// Ist der Bericht auf einen Topf gefiltert, gehört das in die Überschrift: Ein Blatt
	// ohne diesen Zusatz sieht aus wie der Gesamtbericht und wird auch so abgelegt.
	if beschriftung := mitteltopf.Beschriftung(repository.MittelDatenwert(mittel)); mittel != "" && beschriftung != "" {
		titel += " — " + beschriftung
	}
	return titel
}

// GetBestellBerichtPDFHandler liefert den Bericht über die Bestellungen eines Zeitraums als PDF.
// Den Erzeuger trägt pdf/bestellbericht.go; die Tür prüft die Eingabe, liest die Bestellungen
// und füllt seine Eingabe.
//
// Parameter der Query:
//
//	von           JJJJ-MM-TT, erster Tag des Zeitraums
//	bis           JJJJ-MM-TT, letzter Tag des Zeitraums
//	lieferant_id  UUID, wahlweise: nur die Bestellungen eines Lieferanten
//	mittel        wahlweise: nur die Bestellungen eines Topfs
//	titel         wahlweise: eigene Überschrift
//	jahresansicht true, wahlweise: dazu die Übersichten nach Monat und nach Lieferant
func (s *Tueren) GetBestellBerichtPDFHandler() http.HandlerFunc {
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

		// Geprüft wie jede Kennung aus der Query (tuer.UUIDAusQuery): Ein Rohtext wie `x` ginge
		// sonst in `AND lieferant_id = $n` und käme von Postgres als 500 zurück (22P02).
		lieferantID, err := tuer.UUIDAusQuery(r, "lieferant_id")
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusBadRequest, err)
			return
		}
		// Der Topf-Filter macht aus dem Bericht das Blatt, gegen das eine Rechnung geprüft wird.
		// Ein unbekannter Wert ist ein Fehler und kein stiller Gesamtbericht: Sonst prüfte das
		// Sekretariat die Landes-Rechnung gegen eine Liste, in der auch die Schülerbücherei steht.
		mittel := q.Get("mittel")
		if !repository.MittelFilterGueltig(mittel) {
			apierrors.SendHTTPError(w, http.StatusBadRequest, repository.MittelFilterFehler(mittel))
			return
		}
		jahresansicht := q.Get("jahresansicht") == "true"
		berichtTitel := berichtTitelAbleiten(q.Get("titel"), lieferantID, jahresansicht, mittel)

		orders, orderIndex, err := repository.LadeBerichtBestellungen(ctx, s.DB.Pool, von, bisExklusiv, lieferantID, mittel)
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}

		if err := repository.LadeBerichtPositionen(ctx, s.DB.Pool, orders, orderIndex); err != nil {
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

		toepfe, err := bestellberichtToepfe(orders)
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}
		pdfBytes, err := pdf.GenerateBestellberichtPDF(pdf.Bestellbericht{
			Titel:         berichtTitel,
			Von:           von,
			Bis:           bis,
			Jahresansicht: jahresansicht,
			MitPreisen:    settings.PreiseErfassen,
			Toepfe:        toepfe,
		}, schule)
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}

		dateiname := fmt.Sprintf("bestellbericht_%s_%s.pdf", vonStr, bisStr)
		w.Header().Set(tuer.HeaderContentType, tuer.ContentTypePDF)
		w.Header().Set(tuer.HeaderContentDisposition, fmt.Sprintf(`attachment; filename="%s"`, dateiname))
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(pdfBytes) //nolint:errcheck
	}
}

// bestellberichtToepfe füllt die Blöcke des Berichts aus den Zeilen der Abfrage: je Topf ein
// Block in der Reihenfolge der Berichte, unter seiner Beschriftung; ein Topf ohne Bestellung
// bekommt keinen. Bestellungen ohne Zuordnung stehen in einem eigenen Block und werden keinem
// Topf zugeschlagen.
func bestellberichtToepfe(orders []repository.BerichtBestellung) ([]pdf.BestellberichtTopf, error) {
	reihenfolge := mitteltopf.Reihenfolge()
	toepfe := make([]pdf.BestellberichtTopf, 0, len(reihenfolge))
	zugeordnet := 0
	for _, topf := range reihenfolge {
		block := pdf.BestellberichtTopf{Beschriftung: mitteltopf.Beschriftung(topf)}
		for _, o := range orders {
			if o.Mittel == topf {
				block.Bestellungen = append(block.Bestellungen, bestellberichtBestellung(o))
			}
		}
		if len(block.Bestellungen) == 0 {
			continue
		}
		zugeordnet += len(block.Bestellungen)
		toepfe = append(toepfe, block)
	}
	// Eine Bestellung mit einem Topf, den die Reihenfolge nicht nennt, fehlte auf dem Blatt und
	// in jeder Summe. Das Blatt geht in die Abrechnung, deshalb entsteht dann keins.
	if zugeordnet != len(orders) {
		return nil, fmt.Errorf("bestellbericht: %d von %d Bestellungen tragen einen Topf, den der Bericht nicht kennt",
			len(orders)-zugeordnet, len(orders))
	}
	return toepfe, nil
}

// bestellberichtBestellung wandelt eine Zeile der Abfrage in die Bestellung des Berichts.
func bestellberichtBestellung(o repository.BerichtBestellung) pdf.BestellberichtBestellung {
	positionen := make([]pdf.BestellberichtPosition, 0, len(o.Positionen))
	for _, p := range o.Positionen {
		positionen = append(positionen, pdf.BestellberichtPosition(p))
	}
	return pdf.BestellberichtBestellung{
		LieferantName:   o.LieferantName,
		Kundennummer:    o.Kundennummer,
		Bestelldatum:    o.Bestelldatum,
		Gesamtbetrag:    o.Gesamtbetrag,
		AnzahlExemplare: o.AnzahlExemplare,
		Positionen:      positionen,
	}
}
