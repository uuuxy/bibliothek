package api

import (
	"net/http"
)

func (s *Server) registerOrderRoutes(mux *http.ServeMux, orderSvc *OrderService, pdfSvc *PDFService) {
	// Bestellungen & Lieferanten
	mux.Handle("GET /api/bestellungen", s.RequirePermission("view_orders")(s.GetReordersHandler()))
	mux.Handle("GET /api/bestellungen/pdf", s.RequirePermission("view_orders")(s.ExportReordersPDFHandler()))
	mux.Handle("GET /api/bestellhistorie", s.RequirePermission("view_orders")(s.GetBestellhistorieHandler()))
	mux.Handle("GET /api/bestellhistorie/bericht", s.RequirePermission("view_orders")(s.GetBestellBerichtPDFHandler()))
	mux.Handle("GET /api/lieferanten", s.RequirePermission("view_orders")(s.ListSuppliersHandler()))
	mux.Handle("POST /api/lieferanten", s.RequirePermission("create_orders")(s.CreateSupplierHandler()))
	mux.Handle("PUT /api/lieferanten/{id}", s.RequirePermission("create_orders")(s.UpdateSupplierHandler()))
	mux.Handle("DELETE /api/lieferanten/{id}", s.RequirePermission("create_orders")(s.DeleteSupplierHandler()))
	mux.Handle("POST /api/bestellungen", s.RequirePermission("create_orders")(s.SubmitOrderHandler(orderSvc, pdfSvc)))
	mux.Handle("GET /api/bestellungen/zulauf", s.RequirePermission("view_orders")(s.GetIncomingShipmentsHandler()))
	mux.Handle("POST /api/bestellungen/suche", s.RequirePermission("view_orders")(s.SearchOrdersHandler()))
	mux.Handle("POST /api/bestellungen/receive", s.RequirePermission("create_orders")(s.ReceiveItemHandler()))
	mux.Handle("POST /api/bestellungen/bulk-receive", s.RequirePermission("create_orders")(s.BulkReceiveOrderHandler()))
}
