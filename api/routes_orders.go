package api

import (
	"net/http"
)

func (s *Server) registerOrderRoutes(mux *http.ServeMux, orderSvc *OrderService, pdfSvc *PDFService) {
	// Bestellungen & Lieferanten
	mux.Handle("GET /api/bestellungen", s.RequirePermission("view_orders")(s.GetReordersHandler()))
	mux.Handle("GET /api/bestellungen/pdf", s.RequirePermission("view_orders")(s.ExportReordersPDFHandler()))
	mux.Handle("GET /api/lieferanten", s.RequirePermission("view_orders")(s.ListSuppliersHandler()))
	mux.Handle("POST /api/lieferanten", s.RequirePermission("create_orders")(s.CreateSupplierHandler()))
	mux.Handle("PUT /api/lieferanten/{id}", s.RequirePermission("create_orders")(s.UpdateSupplierHandler()))
	mux.Handle("DELETE /api/lieferanten/{id}", s.RequirePermission("create_orders")(s.DeleteSupplierHandler()))
	mux.Handle("POST /api/orders", s.RequirePermission("create_orders")(s.SubmitOrderHandler(orderSvc, pdfSvc)))
	mux.Handle("GET /api/bestellungen/zulauf", s.RequirePermission("view_orders")(s.GetIncomingShipmentsHandler()))
	mux.Handle("POST /api/bestellungen/suche", s.RequirePermission("view_orders")(s.SearchOrdersHandler()))
	mux.Handle("POST /api/orders/receive", s.RequirePermission("create_orders")(s.ReceiveItemHandler()))
	mux.Handle("POST /api/orders/bulk-receive", s.RequirePermission("create_orders")(s.BulkReceiveOrderHandler()))
	mux.Handle("POST /api/orders/release", s.RequirePermission("create_orders")(s.ReleaseOrdersHandler()))
}
