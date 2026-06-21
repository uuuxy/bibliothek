package service

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"bibliothek/db"
	"bibliothek/inventur"
	"bibliothek/repository"

	"github.com/jackc/pgx/v5"
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

// GetIncomingShipments returns a list of ordered copies that are currently in transit.
func GetIncomingShipments(ctx context.Context, pool db.PgxPoolIface) ([]*ShipmentGroup, error) {
	query := `
		SELECT e.id, e.titel_id, e.erstellt_am, e.zustand_notiz, t.titel, COALESCE(t.isbn, ''), COALESCE(t.cover_url, '')
		FROM buecher_exemplare e
		JOIN buecher_titel t ON e.titel_id = t.id
		WHERE e.ist_ausleihbar = false 
		  AND (e.zustand_notiz LIKE 'Im Zulauf%' OR e.zustand_notiz = 'bestellt' OR e.zustand_notiz LIKE 'Bestellt%')
		ORDER BY e.erstellt_am DESC
	`

	rows, err := pool.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	groupsMap := make(map[string]*ShipmentGroup)

	for rows.Next() {
		var exemplarID, titelID, zustandNotiz, titel, isbn, coverURL string
		var erstelltAm time.Time
		if err := rows.Scan(&exemplarID, &titelID, &erstelltAm, &zustandNotiz, &titel, &isbn, &coverURL); err != nil {
			return nil, err
		}

		supplierName := "Unbekannter Lieferant"
		if strings.HasPrefix(zustandNotiz, "Im Zulauf - ") {
			supplierName = strings.TrimPrefix(zustandNotiz, "Im Zulauf - ")
		} else if strings.HasPrefix(zustandNotiz, "Bestellt (Lieferanten-Vorab-Barcode)") {
			supplierName = "Vorab-Barcode Bestellung"
		} else if zustandNotiz == "bestellt" {
			supplierName = "Automatische Nachbestellung"
		}

		dateStr := erstelltAm.Format("02.01.2006")
		groupKey := dateStr + "|" + supplierName

		group, exists := groupsMap[groupKey]
		if !exists {
			group = &ShipmentGroup{
				ID:           strconv.FormatInt(erstelltAm.UnixNano(), 10),
				SupplierName: supplierName,
				Date:         dateStr,
				Timestamp:    erstelltAm,
				Items:        []*GroupedItem{},
			}
			groupsMap[groupKey] = group
		}

		var itemFound *GroupedItem
		for _, item := range group.Items {
			if item.Titel == titel {
				itemFound = item
				break
			}
		}

		if itemFound != nil {
			itemFound.Menge++
			itemFound.ExemplarIDs = append(itemFound.ExemplarIDs, exemplarID)
		} else {
			group.Items = append(group.Items, &GroupedItem{
				TitelID:     titelID,
				Titel:       titel,
				ISBN:        isbn,
				CoverURL:    coverURL,
				Menge:       1,
				ExemplarIDs: []string{exemplarID},
			})
		}
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

// ReceiveItem handles the reception of a single ordered item via barcode scan.
func ReceiveItem(ctx context.Context, pool db.PgxPoolIface, auditRepo repository.AuditRepository, titelID, barcode, adminID, ipAddr string) error {
	query := `
		UPDATE buecher_exemplare
		SET barcode_id = $1, ist_ausleihbar = true, zustand_notiz = ''
		WHERE id = (
			SELECT id 
			FROM buecher_exemplare 
			WHERE titel_id = $2 
			  AND ist_ausleihbar = false 
			  AND (zustand_notiz LIKE 'Im Zulauf%' OR zustand_notiz = 'bestellt' OR zustand_notiz LIKE 'Bestellt%')
			LIMIT 1
			FOR UPDATE SKIP LOCKED
		)
		RETURNING id
	`
	var updatedID string
	err := pool.QueryRow(ctx, query, barcode, titelID).Scan(&updatedID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return errors.New("kein offenes (bestelltes) Exemplar für diesen Titel gefunden")
		}
		return err
	}

	_ = auditRepo.LogAdminAktion(ctx, adminID, "RECEIVE_ITEM", ipAddr, map[string]any{
		"titel_id": titelID,
		"barcode":  barcode,
		"message":  "Wareneingang gebucht (Einzel-Scan)",
	})

	return nil
}

type OrderSearchItem struct {
	ID           string `json:"id,omitempty"`
	Titel        string `json:"titel"`
	Autor        string `json:"autor"`
	ISBN         string `json:"isbn"`
	Verlag       string `json:"verlag,omitempty"`
	CoverURL     string `json:"cover_url,omitempty"`
	Source       string `json:"source"`
	CurrentStock int    `json:"current_stock,omitempty"`
	IsDuplicate  bool   `json:"is_duplicate,omitempty"`
}

// SearchOrders searches local DB and DNB for book orders.
func SearchOrders(ctx context.Context, pool db.PgxPoolIface, metaClient *inventur.MetadatenClient, query string) ([]OrderSearchItem, error) {
	var results []OrderSearchItem

	localQuery := `
		SELECT t.id, t.titel, coalesce(t.autor, ''), coalesce(t.isbn, ''), coalesce(t.verlag, ''), coalesce(t.cover_url, ''),
		       (SELECT COUNT(*) FROM buecher_exemplare e WHERE e.titel_id = t.id AND e.ist_ausgesondert = false) AS current_stock
		FROM buecher_titel t
		WHERE 
			t.search_vector @@ plainto_tsquery('german', $1) 
			OR t.titel ILIKE '%' || $1 || '%'
			OR t.autor ILIKE '%' || $1 || '%'
			OR t.isbn ILIKE '%' || $1 || '%'
			OR replace(t.isbn, '-', '') = replace($1, '-', '')
		ORDER BY ts_rank(t.search_vector, plainto_tsquery('german', $1)) DESC, t.titel ASC
		LIMIT 50
	`
	rows, err := pool.Query(ctx, localQuery, query)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var item OrderSearchItem
			item.Source = "local"
			if errScan := rows.Scan(&item.ID, &item.Titel, &item.Autor, &item.ISBN, &item.Verlag, &item.CoverURL, &item.CurrentStock); errScan == nil {
				results = append(results, item)
			}
		}
	}

	dnbResults, errDNB := metaClient.SucheTextDNB(ctx, query)
	if errDNB == nil {
		for _, dr := range dnbResults {
			coverURL := dr.CoverURL
			if coverURL == "" && dr.ISBN != "" {
				coverURL = fmt.Sprintf("https://portal.dnb.de/opac/mvb/cover?isbn=%s", dr.ISBN)
			}

			existsLocally := false
			if dr.ISBN != "" {
				var count int
				_ = pool.QueryRow(ctx, "SELECT COUNT(*) FROM buecher_titel WHERE replace(isbn, '-', '') = $1", dr.ISBN).Scan(&count)
				if count > 0 {
					existsLocally = true
				}
			}

			results = append(results, OrderSearchItem{
				Titel:       dr.Titel,
				Autor:       dr.Autor,
				ISBN:        dr.ISBN,
				Verlag:      dr.Verlag,
				CoverURL:    coverURL,
				Source:      "dnb",
				IsDuplicate: existsLocally,
			})
		}
	}

	return results, nil
}

// BulkReceiveOrder marks all pre-allocated items as received.
func BulkReceiveOrder(ctx context.Context, pool db.PgxPoolIface, auditRepo repository.AuditRepository, exemplarIDs []string, adminID, ipAddr string) (int, error) {
	query := `
		UPDATE buecher_exemplare
		SET ist_ausleihbar = true, zustand_notiz = ''
		WHERE ist_ausleihbar = false
		  AND id = ANY($1)
		RETURNING id
	`

	rows, err := pool.Query(ctx, query, exemplarIDs)
	if err != nil {
		return 0, err
	}
	defer rows.Close()

	var updatedIDs []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			continue
		}
		updatedIDs = append(updatedIDs, id)
	}

	if len(updatedIDs) == 0 {
		return 0, errors.New("keine zu aktualisierenden Exemplare gefunden (bereits freigegeben?)")
	}

	_ = auditRepo.LogAdminAktion(ctx, adminID, "BULK_RECEIVE_ITEMS", ipAddr, map[string]any{
		"received_count": len(updatedIDs),
		"message":        "Wareneingang gebucht (Massen-Freigabe)",
	})

	return len(updatedIDs), nil
}
