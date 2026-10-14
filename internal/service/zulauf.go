package service

import (
	"context"
	"sort"
	"strings"
	"time"

	"bibliothek/db"
	"bibliothek/pkg/schulzeit"
	"bibliothek/repository"
)

// Zulaufgruppe ist eine Gruppe der Liste im Zulauf: die Titel einer Bestellung, beim Altbestand
// ohne Bestellung die eines Tages und Lieferanten.
type Zulaufgruppe struct {
	ID            string          `json:"id"`
	LieferantName string          `json:"supplierName"`
	Datum         string          `json:"date"`
	Zeitpunkt     time.Time       `json:"-"`
	Posten        []*ZulaufPosten `json:"items"`
}

// ZulaufPosten ist ein Titel einer Zulaufgruppe mit der Zahl und den Kennungen seiner
// Exemplare.
type ZulaufPosten struct {
	TitelID     string   `json:"titel_id"`
	Titel       string   `json:"titel"`
	ISBN        string   `json:"isbn"`
	CoverURL    string   `json:"cover_url"`
	Menge       int      `json:"menge"`
	ExemplarIDs []string `json:"exemplar_ids"`
}

// LadeZulauf liefert die bestellten, noch nicht eingetroffenen Exemplare, gruppiert nach
// Bestellung: Zwei Bestellungen am selben Tag beim selben Händler sind zwei Gruppen, und das
// Datum der Gruppe ist der Kalendertag der Schule. Ein Exemplar aus dem Altbestand ohne
// Bestellung steht unter dem Tag seiner Anlage und dem Lieferanten, den seine Notiz nennt.
func LadeZulauf(ctx context.Context, pool db.PgxPoolIface) ([]*Zulaufgruppe, error) {
	exemplare, err := repository.LadeExemplareImZulauf(ctx, pool)
	if err != nil {
		return nil, err
	}

	nachSchluessel := make(map[string]*Zulaufgruppe)

	for _, e := range exemplare {
		// Kalendertag der Schule: Ein Abend-Auftrag um 23:30 gehört zu seinem Tag, nicht
		// zum nächsten in UTC.
		zeitpunkt := e.ErstelltAm
		schluessel := ""
		gruppenID := ""
		lieferantName := ""
		switch {
		case e.BestellungID != nil && e.LieferantName != nil && e.Bestelldatum != nil:
			zeitpunkt = *e.Bestelldatum
			schluessel = "bestellung|" + *e.BestellungID
			gruppenID = *e.BestellungID
			lieferantName = *e.LieferantName
		default:
			lieferantName = lieferantAusNotiz(e.ZustandNotiz)
			schluessel = zeitpunkt.In(schulzeit.Zone()).Format(datumsformatDE) + "|" + lieferantName
			gruppenID = schluessel
		}
		datum := zeitpunkt.In(schulzeit.Zone()).Format(datumsformatDE)

		gruppe, da := nachSchluessel[schluessel]
		if !da {
			gruppe = &Zulaufgruppe{
				ID:            gruppenID,
				LieferantName: lieferantName,
				Datum:         datum,
				Zeitpunkt:     zeitpunkt,
				Posten:        []*ZulaufPosten{},
			}
			nachSchluessel[schluessel] = gruppe
		}

		ordneExemplarZu(gruppe, exemplarZurGruppe{
			TitelID:    e.TitelID,
			Titel:      e.Titel,
			ISBN:       e.ISBN,
			CoverURL:   e.CoverURL,
			ExemplarID: e.ExemplarID,
		})
	}

	gruppen := make([]*Zulaufgruppe, 0)
	for _, g := range nachSchluessel {
		gruppen = append(gruppen, g)
	}

	sort.Slice(gruppen, func(i, j int) bool {
		return gruppen[i].Zeitpunkt.After(gruppen[j].Zeitpunkt)
	})

	return gruppen, nil
}

// lieferantAusNotiz liest den Lieferanten aus der Notiz eines Exemplars, für den Altbestand
// ohne Bestellung; sonst nennt ihn die Bestellung.
func lieferantAusNotiz(zustandNotiz string) string {
	switch {
	case strings.HasPrefix(zustandNotiz, "Im Zulauf - "):
		return strings.TrimPrefix(zustandNotiz, "Im Zulauf - ")
	case strings.HasPrefix(zustandNotiz, "Bestellt (Lieferanten-Vorab-Barcode)"):
		return "Vorab-Barcode Bestellung"
	case zustandNotiz == "bestellt":
		return "Automatische Nachbestellung"
	default:
		return "Unbekannter Lieferant"
	}
}

// exemplarZurGruppe sind die Angaben eines Exemplars für ordneExemplarZu. Als Struktur:
// Fünf benachbarte Parameter vom selben Typ ließen sich vertauschen, ohne dass der Compiler
// es meldet, und in der Liste des Zulaufs stünde die ISBN als Titel.
type exemplarZurGruppe struct {
	TitelID    string
	Titel      string
	ISBN       string
	CoverURL   string
	ExemplarID string
}

// ordneExemplarZu zählt ein Exemplar zum Posten seines Titels oder legt den Posten an.
func ordneExemplarZu(gruppe *Zulaufgruppe, e exemplarZurGruppe) {
	for _, posten := range gruppe.Posten {
		if posten.Titel == e.Titel {
			posten.Menge++
			posten.ExemplarIDs = append(posten.ExemplarIDs, e.ExemplarID)
			return
		}
	}
	gruppe.Posten = append(gruppe.Posten, &ZulaufPosten{
		TitelID:     e.TitelID,
		Titel:       e.Titel,
		ISBN:        e.ISBN,
		CoverURL:    e.CoverURL,
		Menge:       1,
		ExemplarIDs: []string{e.ExemplarID},
	})
}
