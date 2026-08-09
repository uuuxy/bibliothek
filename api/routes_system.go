package api

import (
	"bibliothek/db"
	"bibliothek/repository"
	"net/http"
)

func (s *Server) registerSystemRoutes(mux *http.ServeMux, auditRepo repository.AuditRepository, userRepo repository.UserRepository, dbPool db.PgxPoolIface) {
	// ── BENUTZER (Users) ──
	mux.Handle("GET /api/benutzer", s.RequirePermission("manage_users")(s.ListUsersHandler(userRepo)))
	mux.Handle("POST /api/benutzer", s.RequirePermission("manage_users")(s.CreateUserHandler(userRepo)))
	mux.Handle("PUT /api/benutzer/{id}", s.RequirePermission("manage_users")(s.UpdateUserHandler(userRepo)))
	mux.Handle("DELETE /api/benutzer/{id}", s.RequirePermission("manage_users")(s.DeleteUserHandler(auditRepo, userRepo)))

	// ── EINSTELLUNGEN (Settings) ──
	settingsRepo := repository.NewSystemSettingsRepository(dbPool)
	mux.Handle("GET /api/einstellungen", s.RequirePermission("manage_users")(s.GetSettingsHandler(settingsRepo)))
	mux.Handle("PUT /api/einstellungen", s.RequirePermission("manage_users")(s.UpdateSettingsHandler(settingsRepo)))

	// Ausweis-Design (zentral, für alle vernetzten Arbeitsplätze). Lesen breit (jeder,
	// der Ausweise druckt), Speichern nur administrativ.
	mux.Handle("GET /api/ausweis-layout", s.RequirePermission("view_students")(s.GetAusweisLayoutHandler()))
	mux.Handle("PUT /api/ausweis-layout", s.RequirePermission("manage_users")(s.SaveAusweisLayoutHandler()))

	mailRepo := repository.NewMailSettingsRepository(dbPool)
	mux.Handle("GET /api/admin/settings/mail", s.RequirePermission("manage_users")(s.GetMailSettingsHandler(mailRepo)))
	mux.Handle("PUT /api/admin/settings/mail", s.RequirePermission("manage_users")(s.UpdateMailSettingsHandler(mailRepo)))
	mux.Handle("POST /api/admin/settings/mail/test", s.RequirePermission("manage_users")(s.PostTestMailSettingsHandler()))

	// Permissions
	mux.Handle("GET /api/admin/permissions", s.RequirePermission("manage_users")(s.GetPermissionsHandler()))
	mux.Handle("GET /api/admin/system/backup-status", s.RequirePermission("manage_users")(s.BackupStatusHandler()))
	mux.Handle("PUT /api/admin/permissions", s.RequirePermission("manage_users")(s.UpdatePermissionsHandler()))

	// Audit & Transactions
	mux.Handle("GET /api/audit", s.RequirePermission("audit_logs")(s.GetAuditLogsHandler()))

	// Mail Templates
	mux.Handle("GET /api/mail-templates", s.RequirePermission("manage_users")(s.GetMailTemplatesHandler()))
	mux.Handle("PUT /api/mail-templates/{id}", s.RequirePermission("manage_users")(s.UpdateMailTemplateHandler()))

	// Print / Reports
	mux.Handle("GET /api/reports/overdue-pdf", s.RequirePermission("view_students")(s.GetOverdueReportsPDFHandler()))
	mux.Handle("GET /api/print/rechnung/{schueler_id}", s.RequirePermission("view_students")(PrintRechnungHandler(s.DB.Pool)))
	mux.Handle("GET /api/print/mahnung/klasse/{klasse}", s.RequirePermission("view_students")(PrintMahnungHandler(s.DB.Pool)))
	mux.Handle("POST /api/admin/mahnungen/bulk-print", s.RequirePermission("view_students")(s.BulkPrintMahnungenHandler()))
	mux.Handle("GET /api/print/kontoauszug/{schueler_id}", s.RequirePermission("view_students")(PrintKontoauszugHandler(s.DB.Pool)))

	// Dashboard & Stats
	mux.Handle("GET /api/dashboard/summary", s.RequirePermission("view_students")(s.GetDashboardSummaryHandler()))
	mux.Handle("GET /api/statistiken", s.RequirePermission("view_students")(s.GetStatisticsHandler()))

	// Lookups
	mux.Handle("GET /api/systematics", s.RequirePermission("view_books")(s.GetSystematicsHandler()))
	// Das Vokabular, aus dem das Buchformular die Signatur vorschlägt. War bisher nur
	// lesbar — es gab keinen Weg, es zu füllen, obwohl der Vorschlag davon abhängt.
	mux.Handle("POST /api/systematics", s.RequirePermission("edit_books")(s.CreateSystematikHandler()))
	mux.Handle("PUT /api/systematics/{id}", s.RequirePermission("edit_books")(s.UpdateSystematikHandler()))
	mux.Handle("DELETE /api/systematics/{id}", s.RequirePermission("edit_books")(s.DeleteSystematikHandler()))
	mux.Handle("GET /api/faecher", s.RequirePermission("view_books")(s.GetFaecherHandler()))
	mux.Handle("GET /api/readergroups", s.RequirePermission("view_students")(s.GetReaderGroupsHandler()))

	// Audit Logs
	mux.Handle("GET /api/admin/auditlog", s.RequirePermission("manage_users")(s.GetAdminAuditLogsHandler()))

	// Barcodes & Etiketten
	// edit_books statt view_books: Der Endpunkt zeigt keine Nummer mehr an, er VERGIBT
	// eine aus barcode_seq — die gezogene Nummer ist danach verbraucht. Wer sie nicht
	// speichern darf (das Barcode-PUT verlangt edit_books), soll sie auch nicht ziehen.
	mux.Handle("GET /api/barcode/next", s.RequirePermission("edit_books")(s.NextBarcodeHandler()))
	mux.Handle("GET /api/barcode", s.RequirePermission("view_books")(s.BarcodeHandler()))
	mux.Handle("GET /api/print/etikett/{id}", s.RequirePermission("view_books")(s.PrintErsatzEtikettHandler()))

	// Signaturen — abgeleitet aus buecher_titel.signatur, nicht aus einer Stammtabelle.
	// Die frühere Tabelle `signatures` (samt GET/POST /api/signatures) ist mit
	// Migration 060 entfallen: Sie kannte Namen, die an keinem Buch hingen.
	mux.Handle("GET /api/signaturen", s.RequirePermission("view_books")(s.GetSignaturenHandler()))
	mux.Handle("GET /api/signaturen/buecher", s.RequirePermission("view_books")(s.GetSignaturBuecherHandler()))

	// Real-time Events. perform_actions statt view_students: den SSE-Stream öffnet jeder
	// eingeloggte Client (authStore + Kiosk-Omnibox). Gehörte er weiter zu view_students,
	// liefe die Helfer-Rolle in eine 403-Reconnect-Schleife. Der Stream trägt nur
	// operative Aktions-Events (Typ, IDs, Buchtitel, Zeitstempel) — keine Schüler-PII.
	sseHandler := s.Broker.Handler()
	mux.Handle("GET /events", s.RequirePermission("perform_actions")(sseHandler))
}
