package service

import (
	"context"
	"errors"
	"fmt"
	"log"
	"sort"
	"strings"
	"time"

	"bibliothek/db"
	"bibliothek/inventur"
	"bibliothek/pkg/isbnutil"
	"bibliothek/pkg/schulzeit"
	"bibliothek/repository"
)

// ShipmentGroup helps structure the incoming shipments response.
type ShipmentGroup struct {
	ID           string         `json:"id"`
	SupplierName string         `json:"supplierName"`
	Date         string         `json:"date"`
	Timestamp    time.Time      `json:"-"`
	Items        []*GroupedItem `json:"items"`
}

// GroupedItem represents an item within a ShipmentGroup.
type GroupedItem struct {
	TitelID     string   `json:"titel_id"`
	Titel       string   `json:"titel"`
	ISBN        string   `json:"isbn"`
	CoverURL    string   `json:"cover_url"`
	Menge       int      `json:"menge"`
	ExemplarIDs []string `json:"exemplar_ids"`
}

// GetIncomingShipments liefert die bestellten, noch nicht eingetroffenen Exemplare, gruppiert
// nach Bestellung: Zwei Bestellungen am selben Tag beim selben Händler sind zwei Gruppen, und
// das Datum der Gruppe ist der Kalendertag der Schule. Ein Exemplar ohne Bestellung
// (Altbestand vor Migration 063) steht unter dem Tag seiner Anlage und dem Lieferanten, den
// seine Notiz nennt.
func GetIncomingShipments(ctx context.Context, pool db.PgxPoolIface) ([]*ShipmentGroup, error) {
	exemplare, err := repository.ExemplareImZulauf(ctx, pool)
	if err != nil {
		return nil, err
	}

	groupsMap := make(map[string]*ShipmentGroup)

	for _, e := range exemplare {
		// Kalendertag der Schule: Ein Abend-Auftrag um 23:30 gehört zu seinem Tag, nicht
		// zum nächsten in UTC.
		zeitpunkt := e.ErstelltAm
		groupKey := ""
		groupID := ""
		supplierName := ""
		switch {
		case e.BestellungID != nil && e.LieferantName != nil && e.Bestelldatum != nil:
			zeitpunkt = *e.Bestelldatum
			groupKey = "bestellung|" + *e.BestellungID
			groupID = *e.BestellungID
			supplierName = *e.LieferantName
		default:
			supplierName = resolveSupplierName(e.ZustandNotiz)
			groupKey = zeitpunkt.In(schulzeit.Zone()).Format(dateFormatDE) + "|" + supplierName
			groupID = groupKey
		}
		dateStr := zeitpunkt.In(schulzeit.Zone()).Format(dateFormatDE)

		group, exists := groupsMap[groupKey]
		if !exists {
			group = &ShipmentGroup{
				ID:           groupID,
				SupplierName: supplierName,
				Date:         dateStr,
				Timestamp:    zeitpunkt,
				Items:        []*GroupedItem{},
			}
			groupsMap[groupKey] = group
		}

		addExemplarToGroup(group, exemplarZurGruppe{
			TitelID:    e.TitelID,
			Titel:      e.Titel,
			ISBN:       e.ISBN,
			CoverURL:   e.CoverURL,
			ExemplarID: e.ExemplarID,
		})
	}

	groups := make([]*ShipmentGroup, 0)
	for _, g := range groupsMap {
		groups = append(groups, g)
	}

	sort.Slice(groups, func(i, j int) bool {
		return groups[i].Timestamp.After(groups[j].Timestamp)
	})

	return groups, nil
}

// resolveSupplierName leitet den Lieferanten aus der Notiz eines Exemplars ab, für den
// Altbestand ohne Bestellung (vor Migration 063); sonst nennt ihn die Bestellung.
func resolveSupplierName(zustandNotiz string) string {
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

// exemplarZurGruppe sind die Angaben eines Exemplars für addExemplarToGroup. Als Struktur:
// Fünf benachbarte Parameter vom selben Typ ließen sich vertauschen, ohne dass der Compiler
// es meldet, und in der Liste des Zulaufs stünde die ISBN als Titel.
type exemplarZurGruppe struct {
	TitelID    string
	Titel      string
	ISBN       string
	CoverURL   string
	ExemplarID string
}

// addExemplarToGroup ordnet ein Exemplar dem passenden GroupedItem (nach Titel) zu oder
// legt einen neuen Item-Eintrag in der Gruppe an.
func addExemplarToGroup(group *ShipmentGroup, e exemplarZurGruppe) {
	for _, item := range group.Items {
		if item.Titel == e.Titel {
			item.Menge++
			item.ExemplarIDs = append(item.ExemplarIDs, e.ExemplarID)
			return
		}
	}
	group.Items = append(group.Items, &GroupedItem{
		TitelID:     e.TitelID,
		Titel:       e.Titel,
		ISBN:        e.ISBN,
		CoverURL:    e.CoverURL,
		Menge:       1,
		ExemplarIDs: []string{e.ExemplarID},
	})
}

// OrderSearchItem ist ein Suchtreffer der Bestellsuche auf Dienstebene. Welche Felder
// gefüllt sind, hängt von der Herkunft ab — die Feldkommentare unten sagen, welche nur
// bei source="local" existieren.
type OrderSearchItem struct {
	ID       string `json:"id,omitempty"`
	Titel    string `json:"titel"`
	Autor    string `json:"autor"`
	ISBN     string `json:"isbn"`
	Verlag   string `json:"verlag,omitempty"`
	CoverURL string `json:"cover_url,omitempty"`
	// Signatur ist nur bei source="local" gesetzt — die Regalsignatur eines bereits
	// katalogisierten Titels. DNB-Treffer haben noch keinen lokalen Titel und damit
	// keine Signatur; die entsteht erst als Vorschlag beim Anlegen über /aus-isbn.
	Signatur     string `json:"signatur,omitempty"`
	Source       string `json:"source"`
	CurrentStock int    `json:"current_stock,omitempty"`
	IsDuplicate  bool   `json:"is_duplicate,omitempty"`
	// PreisVorschlag ist der Ladenpreis laut DNB (MARC21 020 $c), 0 heißt keiner. Er gilt
	// zum Erscheinen und ist nicht der Preis der Schule: Die Oberfläche füllt damit das
	// Preisfeld vor und übernimmt ihn nie ungefragt als Ausgabe.
	PreisVorschlag float64 `json:"preis_vorschlag,omitempty"`
	// IstLernmittel: nur bei source="local" aussagekräftig — der Vorschlag für den
	// Topf der Bestellung (Lernmittel → Land, sonst Schulträger; Warenkorb, mittel.js).
	// Ein DNB-Treffer ist noch kein Titel und trägt deshalb false; das Staging-Fenster
	// fragt beim Anlegen nach.
	IstLernmittel bool `json:"ist_lernmittel"`
}

// SearchOrders sucht einen Titel zum Bestellen im eigenen Katalog und bei der DNB. dnbAusfall
// sagt, dass die DNB nicht geantwortet hat: Die Liste trägt dann nur den eigenen Katalog, und
// ein Buch, das die DNB kennt, sähe ohne das Merkmal aus wie eines, das sie nicht kennt.
func SearchOrders(ctx context.Context, pool db.PgxPoolIface, metaClient *inventur.MetadatenClient, query string) (treffer []OrderSearchItem, dnbAusfall bool, err error) {
	treffer = searchLocalOrders(ctx, pool, query)
	vonDNB, dnbAusfall := searchDNBOrders(ctx, pool, metaClient, query)
	return append(treffer, vonDNB...), dnbAusfall, nil
}

// searchLocalOrders durchsucht den eigenen Katalog (repository.SucheTitelZumBestellen). Scheitert
// die Abfrage, bleibt die Liste leer: Die Treffer der DNB kommen ohnehin getrennt dazu.
func searchLocalOrders(ctx context.Context, pool db.PgxPoolIface, query string) []OrderSearchItem {
	treffer, err := repository.SucheTitelZumBestellen(ctx, pool, query)
	if err != nil {
		return nil
	}
	var results []OrderSearchItem
	for _, t := range treffer {
		results = append(results, OrderSearchItem{
			ID:            t.ID,
			Titel:         t.Titel,
			Autor:         t.Autor,
			ISBN:          t.ISBN,
			Verlag:        t.Verlag,
			CoverURL:      t.CoverURL,
			Signatur:      t.Signatur,
			Source:        "local",
			CurrentStock:  t.Bestand,
			IstLernmittel: t.IstLernmittel,
		})
	}
	return results
}

// searchDNBOrders fragt die DNB nach Titeln und markiert bereits lokal vorhandene ISBNs.
// Jeder Fehler der Abfrage ist ein Ausfall: Eine Suche ohne Treffer beantwortet die DNB mit
// einer leeren Liste, nicht mit einem Fehler.
func searchDNBOrders(ctx context.Context, pool db.PgxPoolIface, metaClient *inventur.MetadatenClient, query string) (treffer []OrderSearchItem, ausfall bool) {
	dnbResults, errDNB := metaClient.SucheTextDNB(ctx, query)
	if errDNB != nil {
		log.Printf("bestellsuche: die DNB hat nicht geantwortet: %v", errDNB)
		return nil, true
	}

	var isbns []string
	for _, dr := range dnbResults {
		if n := isbnutil.Normalform(dr.ISBN); n != "" {
			isbns = append(isbns, n)
		}
	}

	existingISBNs := sammleExistierendeISBNs(ctx, pool, isbns)

	for _, dr := range dnbResults {
		treffer = append(treffer, baueDNBSuchItem(dr, existingISBNs))
	}
	return treffer, false
}

// sammleExistierendeISBNs fragt, welche der ISBNs schon ein Titel trägt, und liefert sie als
// Menge (repository.ISBNsImKatalog). Beide Seiten stehen in der Normalform: Die DNB nennt eine
// ISBN mit Bindestrichen und bei älteren Sätzen zehnstellig, der Katalog führt sie
// dreizehnstellig; der Treffer hieße sonst „Neu". Ein Fehler wird protokolliert und ergibt
// eine leere Menge; die Suche in der DNB scheitert daran nicht.
func sammleExistierendeISBNs(ctx context.Context, pool db.PgxPoolIface, isbns []string) map[string]struct{} {
	existing, err := repository.ISBNsImKatalog(ctx, pool, isbns)
	if err != nil {
		log.Printf("order-service: Bulk ISBN-Existenzprüfung fehlgeschlagen: %v", err)
		return map[string]struct{}{}
	}
	return existing
}

// baueDNBSuchItem macht aus einem Treffer der DNB einen Eintrag der Bestellsuche: ohne
// eigenes Cover die Cover-Adresse der DNB, und „Vorhanden", wenn die ISBN in der Menge der
// schon getragenen steht (sammleExistierendeISBNs, Normalform).
func baueDNBSuchItem(dr inventur.MetadatenErgebnis, existingISBNs map[string]struct{}) OrderSearchItem {
	coverURL := dr.CoverURL
	if coverURL == "" && dr.ISBN != "" {
		coverURL = fmt.Sprintf("https://portal.dnb.de/opac/mvb/cover?isbn=%s", dr.ISBN)
	}

	_, existsLocally := existingISBNs[isbnutil.Normalform(dr.ISBN)]

	return OrderSearchItem{
		Titel:          dr.Titel,
		Autor:          dr.Autor,
		ISBN:           dr.ISBN,
		Verlag:         dr.Verlag,
		CoverURL:       coverURL,
		Source:         "dnb",
		IsDuplicate:    existsLocally,
		PreisVorschlag: dr.Preis,
	}
}

// ReceivedItem describes a received exemplar, including whether its
// barcode label still needs printing (drives the print suggestion in the UI).
type ReceivedItem struct {
	BarcodeID       string `json:"barcode_id"`
	Titel           string `json:"titel"`
	Autor           string `json:"autor"`
	EtikettGedruckt bool   `json:"etikett_gedruckt"`
}

// ErrNichtsEinzubuchen meldet, dass keines der genannten Exemplare im Zulauf stand: Ein
// anderer Platz war schneller, oder die Liste ist veraltet. Die Tür antwortet mit 404 und
// erkennt den Fall am Fehlerwert, nicht am Wortlaut der Meldung.
var ErrNichtsEinzubuchen = errors.New("keine zu aktualisierenden Exemplare gefunden (bereits freigegeben?)")

// BulkReceiveParams encapsulates the parameters needed to bulk receive orders.
type BulkReceiveParams struct {
	ExemplarIDs []string
	AdminID     string
	IPAddr      string
}

// BulkReceiveOrder marks all pre-allocated items as received.
func BulkReceiveOrder(ctx context.Context, pool db.PgxPoolIface, auditRepo repository.AuditRepository, params BulkReceiveParams) ([]ReceivedItem, error) {
	eingebucht, err := repository.BucheZulaufEin(ctx, pool, params.ExemplarIDs)
	if err != nil {
		return nil, err
	}
	if len(eingebucht) == 0 {
		return nil, ErrNichtsEinzubuchen
	}
	items := make([]ReceivedItem, 0, len(eingebucht))
	for _, e := range eingebucht {
		items = append(items, ReceivedItem(e))
	}

	logAuditErr("wareneingang-bulk", auditRepo.LogAdminAktion(ctx, params.AdminID, "BULK_RECEIVE_ITEMS", params.IPAddr, map[string]any{
		"received_count": len(items),
		"message":        "Wareneingang gebucht (Massen-Freigabe)",
	}))

	return items, nil
}
