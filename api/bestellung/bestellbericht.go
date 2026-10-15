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

// leseBerichtZeitraum liest die zwei Tage des Zeitraums (JJJJ-MM-TT) als Kalendertage der
// Schule; beide sind Pflicht. In UTC gelesen zählte eine Bestellung aus der ersten Stunde eines
// Monats zum Monat davor.
func leseBerichtZeitraum(vonText, bisText string) (time.Time, time.Time, error) {
	if vonText == "" || bisText == "" {
		return time.Time{}, time.Time{}, fmt.Errorf("parameter 'von' und 'bis' erforderlich (YYYY-MM-DD)")
	}
	von, err := schulzeit.Kalendertag(vonText)
	if err != nil {
		return time.Time{}, time.Time{}, fmt.Errorf("ungültiges Datum 'von': %w", err)
	}
	bis, err := schulzeit.Kalendertag(bisText)
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

// berichtAnfrage sind die geprüften Angaben aus der Query des Bestellberichts.
type berichtAnfrage struct {
	von, bis            time.Time
	vonText, bisText    string
	lieferantID, mittel string
	jahresansicht       bool
	titel               string
}

// leseBerichtAnfrage prüft die Query des Bestellberichts. Ein Fehler ist die Meldung an die
// Oberfläche: Jede der drei Prüfungen weist eine Eingabe ab.
func leseBerichtAnfrage(r *http.Request) (berichtAnfrage, error) {
	q := r.URL.Query()
	a := berichtAnfrage{
		vonText:       q.Get("von"),
		bisText:       q.Get("bis"),
		mittel:        q.Get("mittel"),
		jahresansicht: q.Get("jahresansicht") == "true",
	}
	var err error
	if a.von, a.bis, err = leseBerichtZeitraum(a.vonText, a.bisText); err != nil {
		return berichtAnfrage{}, err
	}
	// Geprüft wie jede Kennung aus der Query (tuer.LeseUUIDParameter): Ein Rohtext wie `x` ginge
	// sonst in `AND lieferant_id = $n` und käme von Postgres als 500 zurück (22P02).
	if a.lieferantID, err = tuer.LeseUUIDParameter(r, "lieferant_id"); err != nil {
		return berichtAnfrage{}, err
	}
	// Der Topf-Filter macht aus dem Bericht das Blatt, gegen das eine Rechnung geprüft wird.
	// Ein unbekannter Wert ist ein Fehler und kein stiller Gesamtbericht: Sonst prüfte das
	// Sekretariat die Landes-Rechnung gegen eine Liste, in der auch die Schülerbücherei steht.
	if !repository.IstMittelFilterGueltig(a.mittel) {
		return berichtAnfrage{}, repository.MittelFilterFehler(a.mittel)
	}
	a.titel = berichtTitelAbleiten(q.Get("titel"), a.lieferantID, a.jahresansicht, a.mittel)
	return a, nil
}

// BestellberichtPDF liefert den Bericht über die Bestellungen eines Zeitraums als PDF.
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
func (t *Tueren) BestellberichtPDF() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		a, err := leseBerichtAnfrage(r)
		if err != nil {
			apierrors.AntworteFehler(w, http.StatusBadRequest, err)
			return
		}

		bestellungen, bestellIndex, err := repository.LadeBerichtBestellungen(ctx, t.DB.Pool, a.von, a.bis.AddDate(0, 0, 1), a.lieferantID, a.mittel)
		if err != nil {
			apierrors.AntworteFehler(w, http.StatusInternalServerError, err)
			return
		}

		if err := repository.LadeBerichtPositionen(ctx, t.DB.Pool, bestellungen, bestellIndex); err != nil {
			apierrors.AntworteFehler(w, http.StatusInternalServerError, err)
			return
		}

		// Ein Lesefehler ist ein Fehler: Mit den Vorgaben stünde das Blatt ohne den Kopf der
		// Schule da, und die Vorgabe entschiede, ob es Beträge nennt.
		einstellungen, err := repository.NewEinstellungsAbfragen(t.DB.Pool).Lade(ctx)
		if err != nil {
			apierrors.AntworteFehler(w, http.StatusInternalServerError, fmt.Errorf("einstellungen lesen: %w", err))
			return
		}
		schule := pdf.SchuleInfo{
			Name:    einstellungen.SchuleName,
			Strasse: einstellungen.SchuleStrasse,
			PLZ:     einstellungen.SchulePLZ,
			Ort:     einstellungen.SchuleOrt,
		}

		toepfe, err := bestellberichtToepfe(bestellungen)
		if err != nil {
			apierrors.AntworteFehler(w, http.StatusInternalServerError, err)
			return
		}
		pdfDaten, err := pdf.GenerateBestellberichtPDF(pdf.Bestellbericht{
			Titel:         a.titel,
			Von:           a.von,
			Bis:           a.bis,
			Jahresansicht: a.jahresansicht,
			MitPreisen:    einstellungen.PreiseErfassen,
			Toepfe:        toepfe,
		}, schule)
		if err != nil {
			apierrors.AntworteFehler(w, http.StatusInternalServerError, err)
			return
		}

		dateiname := fmt.Sprintf("bestellbericht_%s_%s.pdf", a.vonText, a.bisText)
		w.Header().Set(tuer.HeaderContentType, tuer.ContentTypePDF)
		w.Header().Set(tuer.HeaderContentDisposition, fmt.Sprintf(`attachment; filename="%s"`, dateiname))
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(pdfDaten) //nolint:errcheck
	}
}

// bestellberichtToepfe füllt die Blöcke des Berichts aus den Zeilen der Abfrage: je Topf ein
// Block in der Reihenfolge der Berichte, unter seiner Beschriftung; ein Topf ohne Bestellung
// bekommt keinen. Bestellungen ohne Zuordnung stehen in einem eigenen Block und werden keinem
// Topf zugeschlagen.
func bestellberichtToepfe(bestellungen []repository.BerichtBestellung) ([]pdf.BestellberichtTopf, error) {
	reihenfolge := mitteltopf.Reihenfolge()
	toepfe := make([]pdf.BestellberichtTopf, 0, len(reihenfolge))
	zugeordnet := 0
	for _, topf := range reihenfolge {
		block := pdf.BestellberichtTopf{Beschriftung: mitteltopf.Beschriftung(topf)}
		for _, o := range bestellungen {
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
	if zugeordnet != len(bestellungen) {
		return nil, fmt.Errorf("bestellbericht: %d von %d Bestellungen tragen einen Topf, den der Bericht nicht kennt",
			len(bestellungen)-zugeordnet, len(bestellungen))
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
