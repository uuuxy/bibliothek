package api

import (
	"bibliothek/auth"
	"bibliothek/repository"
	"net/http"
)

func (s *Server) registerSystemRoutes(mux *http.ServeMux, auditRepo repository.AuditRepository, userRepo repository.UserRepository) {
	// ── BENUTZER (Users) ──
	mux.Handle("GET /api/benutzer", s.RequirePermission("manage_users")(s.ListUsersHandler(userRepo)))
	mux.Handle("POST /api/benutzer", s.RequirePermission("manage_users")(s.CreateUserHandler(userRepo)))
	mux.Handle("PUT /api/benutzer/{id}", s.RequirePermission("manage_users")(s.UpdateUserHandler(userRepo)))
	mux.Handle("DELETE /api/benutzer/{id}", s.RequirePermission("manage_users")(s.DeleteUserHandler(auditRepo)))

	// ── EINSTELLUNGEN (Settings) ──
	mux.Handle("GET /api/einstellungen", s.RequirePermission("manage_users")(s.GetSettingsHandler()))
	mux.Handle("PUT /api/einstellungen", s.RequirePermission("manage_users")(s.UpdateSettingsHandler()))

	// Permissions
	mux.Handle("GET /api/admin/permissions", s.RequirePermission("manage_users")(s.GetPermissionsHandler()))
	mux.Handle("PUT /api/admin/permissions", s.RequirePermission("manage_users")(s.UpdatePermissionsHandler()))

	// Audit & Transactions
	mux.Handle("GET /api/audit", s.RequirePermission("audit_logs")(s.GetAuditLogsHandler()))
	mux.Handle("GET /api/transactions/recent", s.Auth.RequireRoles(auth.RoleAdmin, auth.RoleMitarbeiter, auth.RoleHelfer)(s.GetRecentTransactionsHandler()))

	// Mail Templates
	mux.Handle("GET /api/mail-templates", s.RequirePermission("manage_users")(s.GetMailTemplatesHandler()))
	mux.Handle("PUT /api/mail-templates/{id}", s.RequirePermission("manage_users")(s.UpdateMailTemplateHandler()))
	mux.Handle("POST /api/mail/send-overdue-notification/{schuelerID}", s.RequirePermission("manage_users")(s.PostSendOverdueNotificationHandler()))
	mux.Handle("POST /api/mail/send-notification/{schuelerID}", s.RequirePermission("manage_users")(s.PostSendNotificationHandler()))
	mux.Handle("POST /api/mail/send-bulk-overdue", s.RequirePermission("manage_users")(s.PostSendBulkOverdueHandler()))

	// Print / Reports
	mux.Handle("GET /api/reports/overdue-pdf", s.RequirePermission("view_students")(s.GetOverdueReportsPDFHandler()))
	mux.Handle("GET /api/print/rechnung/{schueler_id}", s.RequirePermission("view_students")(PrintRechnungHandler(s.DB.Pool)))
	mux.Handle("GET /api/print/mahnung/klasse/{klasse}", s.RequirePermission("view_students")(PrintMahnungHandler(s.DB.Pool)))
	mux.Handle("GET /api/print/kontoauszug/{schueler_id}", s.RequirePermission("view_students")(PrintKontoauszugHandler(s.DB.Pool)))

	// Dashboard & Stats
	mux.Handle("GET /api/dashboard/summary", s.RequirePermission("view_students")(s.GetDashboardSummaryHandler()))
	mux.Handle("GET /api/statistiken", s.RequirePermission("view_students")(s.GetStatisticsHandler()))

	// Lookups
	mux.Handle("GET /api/systematics", s.RequirePermission("view_books")(s.GetSystematicsHandler()))
	mux.Handle("GET /api/readergroups", s.RequirePermission("view_students")(s.GetReaderGroupsHandler()))

	// Audit Logs
	mux.Handle("GET /api/admin/auditlog", s.RequirePermission("manage_users")(s.GetAdminAuditLogsHandler()))

	// Barcodes & Etiketten
	mux.Handle("GET /api/barcode/next", s.RequirePermission("view_books")(s.NextBarcodeHandler()))
	mux.Handle("GET /api/barcode", s.RequirePermission("view_books")(s.BarcodeHandler()))
	mux.Handle("GET /api/print/etikett/{id}", s.RequirePermission("view_books")(s.PrintErsatzEtikettHandler()))

	// Signaturen (Master Data Management)
	mux.Handle("GET /api/signatures", s.RequirePermission("view_books")(s.GetSignaturesHandler()))
	mux.Handle("POST /api/signatures", s.RequirePermission("manage_users")(s.CreateSignatureHandler()))
	mux.Handle("PUT /api/signatures/{id}", s.RequirePermission("manage_users")(s.UpdateSignatureHandler()))
	mux.Handle("DELETE /api/signatures/{id}", s.RequirePermission("manage_users")(s.DeleteSignatureHandler()))

	// Real-time Events
	sseHandler := s.Broker.Handler()
	mux.Handle("GET /events", s.RequirePermission("view_students")(sseHandler))
}
