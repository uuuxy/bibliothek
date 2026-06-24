package api

import (
	"bibliothek/repository"
	"net/http"
)

func (s *Server) registerStudentRoutes(mux *http.ServeMux, studentRepo repository.StudentRepository, mahnRepo *repository.MahnwesenRepository, auditRepo repository.AuditRepository) {
	// ── SCHUELER (Students) ──
	mux.Handle("GET /api/schueler", s.RequirePermission("view_students")(s.ListStudentsHandler(studentRepo)))
	mux.Handle("GET /api/schueler/{id}", s.RequirePermission("view_students")(s.GetStudentProfileHandler(studentRepo)))
	mux.Handle("POST /api/schueler", s.RequirePermission("create_students")(s.CreateStudentHandler()))
	mux.Handle("PATCH /api/schueler/{id}", s.RequirePermission("create_students")(s.PatchStudentHandler()))
	mux.Handle("PATCH /api/admin/students/{id}/lock", s.RequirePermission("create_students")(s.LockStudentHandler()))
	mux.Handle("DELETE /api/schueler/{id}", s.RequirePermission("delete_students")(s.DeleteStudentHandler(auditRepo)))

	// Papierkorb
	mux.Handle("GET /api/schueler/deleted", s.RequirePermission("delete_students")(s.GetDeletedStudentsHandler()))
	mux.Handle("POST /api/schueler/{id}/restore", s.RequirePermission("delete_students")(s.RestoreStudentHandler()))

	// Photos
	mux.Handle("POST /api/schueler/{id}/photo", s.RequirePermission("upload_photos")(s.UploadStudentPhotoHandler()))
	mux.Handle("GET /api/schueler/{id}/photo", s.RequirePermission("view_students")(s.ServeStudentPhotoHandler()))

	// Klassen
	mux.Handle("GET /api/klassen", s.RequirePermission("view_students")(s.GetClassesHandler(studentRepo)))
	mux.Handle("GET /api/klassen-mapping", s.RequirePermission("manage_users")(s.GetKlassenMappingHandler()))
	mux.Handle("POST /api/klassen-mapping", s.RequirePermission("manage_users")(s.UpsertKlassenMappingHandler()))
	mux.Handle("DELETE /api/klassen-mapping/{klasse}", s.RequirePermission("manage_users")(s.DeleteKlassenMappingHandler()))

	// LUSD Import
	mux.Handle("POST /api/import/students", s.RequirePermission("import_students")(s.ImportStudentsHandler()))
	mux.Handle("POST /api/import/lusd", s.RequirePermission("import_students")(s.ImportLUSDHandler(studentRepo)))
	mux.Handle("POST /api/students/import", s.RequirePermission("import_students")(s.ImportStudentsLUSDHandler()))
	mux.Handle("POST /api/lusd/preview", s.RequirePermission("manage_users")(s.PostLusdPreviewHandler()))
	mux.Handle("POST /api/lusd/import", s.RequirePermission("manage_users")(s.PostLusdImportHandler()))
	mux.Handle("POST /api/schueler/import-lusd", s.RequirePermission("manage_users")(s.PostSchuelerImportLusdHandler()))

	// Promotion
	mux.Handle("POST /api/students/promote", s.RequirePermission("manage_users")(s.PromoteStudentsHandler()))

	// Abgänger (Graduates)
	mux.Handle("GET /api/abgaenger", s.RequirePermission("view_graduates")(s.GetGraduatesHandler()))
	mux.Handle("GET /api/abgaenger/pdf", s.RequirePermission("view_graduates")(s.GetGraduatesPDFHandler()))

	// Kiosk / Damage
	damageRepo := repository.NewDamageRepository(s.DB.Pool)
	mux.Handle("POST /api/damage/report", s.RequirePermission("edit_students")(s.ReportDamageHandler(damageRepo)))
	mux.Handle("GET /api/schadensfaelle/{id}/pdf", s.RequirePermission("view_students")(s.GenerateDamagePDFHandler()))

	// Mahnwesen
	mux.Handle("GET /api/mahnwesen", s.RequirePermission("view_students")(s.GetMahnwesenHandler(mahnRepo)))
	mux.Handle("GET /api/mahnwesen/ueberfaellig_jahrgang", s.RequirePermission("view_students")(s.GetMahnwesenJahrgangHandler(mahnRepo)))
	mux.Handle("GET /api/mahnwesen/pdf", s.RequirePermission("view_students")(s.GetMahnwesenPDFHandler(mahnRepo)))
	mux.Handle("POST /api/mahnwesen/senden", s.RequirePermission("create_orders")(s.SendMahnwesenHandler(mahnRepo)))
	// Alias für das Frontend (downloadElternPDF ruft /api/reports/overdue-pdf auf)
	mux.Handle("GET /api/reports/overdue-pdf", s.RequirePermission("view_students")(s.GetMahnwesenPDFHandler(mahnRepo)))
}
