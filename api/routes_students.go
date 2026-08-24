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

	// Klebeetiketten der markierten Schüler (Name, Klasse, Barcode). view_students wie
	// der Ausweis-Stapeldruck: Es ist derselbe Vorgang mit anderem Papier, und die
	// Angaben darauf stehen ohnehin schon in der Liste, aus der markiert wurde.
	mux.Handle("POST /api/print/schueler-etiketten", s.RequirePermission("view_students")(s.PrintSchuelerEtikettenHandler(studentRepo)))
	// Ändern und Sperren hängen an edit_students, nicht mehr an create_students:
	// Stammdaten pflegen ist nicht dasselbe wie jemanden neu anlegen, und
	// edit_students war zwar geseedet und an /api/damage/report gebunden, in der
	// Rechte-Oberfläche aber gar nicht sichtbar (Audit-Befund 01.08.2026).
	// Ungefährlich: Beide Rechte sind für alle Rollen identisch geseedet
	// (ADMIN/MITARBEITER ja, LEHRER/HELFER nein) — niemand gewinnt oder verliert Zugriff.
	mux.Handle("PATCH /api/schueler/{id}", s.RequirePermission("edit_students")(s.PatchStudentHandler(auditRepo)))
	mux.Handle("PATCH /api/admin/students/{id}/lock", s.RequirePermission("edit_students")(s.LockStudentHandler()))
	mux.Handle("DELETE /api/schueler/{id}", s.RequirePermission("delete_students")(s.DeleteStudentHandler(auditRepo)))

	// DSGVO-Betroffenenauskunft (Art. 15) — manage_students_admin (seit 24.08.2026, vorher manage_users):
	// Der Export bündelt sämtliche personenbezogenen Daten eines Schülers.
	mux.Handle("GET /api/schueler/{id}/dsgvo-auskunft", s.RequirePermission("manage_students_admin")(s.DsgvoAuskunftHandler()))
	mux.Handle("GET /api/schueler/{id}/dsgvo-auskunft/pdf", s.RequirePermission("manage_students_admin")(s.DsgvoAuskunftPDFHandler()))

	// Papierkorb
	mux.Handle("GET /api/schueler/deleted", s.RequirePermission("delete_students")(s.GetDeletedStudentsHandler()))
	mux.Handle("POST /api/schueler/{id}/restore", s.RequirePermission("delete_students")(s.RestoreStudentHandler()))
	// Endgültig löschen (DSGVO-Purge): getrennte, stärkere Berechtigung als Soft-Delete.
	mux.Handle("DELETE /api/schueler/deleted/{id}", s.RequirePermission("manage_students_admin")(s.PurgeStudentHandler(auditRepo)))

	// Photos — die beiden Richtungen sprechen bewusst UNTERSCHIEDLICHE Kennungen an:
	// hochgeladen wird über die Schüler-UUID (der Aufrufer kennt den Datensatz),
	// ausgeliefert über den Barcode. Letzteres ist kein Zufall: resolveFotoURL
	// (api/student_profile.go), student_profile_queries.go und photo_service.go bauen
	// die Bild-URL alle drei als /api/schueler/<barcode>/photo, und das Frontend rendert
	// sie unverändert als <img src>.
	//
	// Deshalb heißt der Platzhalter im GET {barcode_id} und nicht {id}: Die globale
	// ValidateUUIDParamsMiddleware prüft ausschließlich einen Parameter namens "id" und
	// wies jeden echten Barcode (z. B. "S-abg1-ms3p7309e19de436") mit 400 "ungültiges
	// UUID Format" ab — verifiziert am laufenden System. Kein einziges Schülerfoto wurde
	// je ausgeliefert; die URL blieb dieselbe, nur die Prüfung passte nicht zu ihr.
	mux.Handle("POST /api/schueler/{id}/photo", s.RequirePermission("upload_photos")(s.UploadStudentPhotoHandler()))
	mux.Handle("GET /api/schueler/{barcode_id}/photo", s.RequirePermission("view_students")(s.ServeStudentPhotoHandler()))

	// Klassen
	mux.Handle("GET /api/klassen", s.RequirePermission("view_students")(s.GetClassesHandler(studentRepo)))
	mux.Handle("GET /api/klassen-mapping", s.RequirePermission("manage_settings")(s.GetKlassenMappingHandler()))
	mux.Handle("POST /api/klassen-mapping", s.RequirePermission("manage_settings")(s.UpsertKlassenMappingHandler()))
	mux.Handle("DELETE /api/klassen-mapping/{klasse}", s.RequirePermission("manage_settings")(s.DeleteKlassenMappingHandler()))

	// LUSD Import
	// import_students statt manage_users: Das Recht war geseedet und in der
	// Rechte-Oberfläche als „LUSD / CSV Import" sichtbar — steuerte aber KEINEN
	// einzigen Endpunkt (Audit-Befund 01.08.2026). Ein Schalter, der nichts schaltet,
	// ist schlimmer als keiner: Er behauptet eine Einstellung, die es nicht gibt.
	// Identisch geseedet zu manage_users für ADMIN und MITARBEITER, LEHRER und HELFER
	// bleiben ausgeschlossen — die Zuordnung ändert sich also für niemanden.
	mux.Handle("POST /api/lusd/preview", s.RequirePermission("import_students")(s.PostLusdPreviewHandler()))
	mux.Handle("POST /api/lusd/import", s.RequirePermission("import_students")(s.PostLusdImportHandler()))

	// Promotion
	mux.Handle("POST /api/students/promote", s.RequirePermission("manage_students_admin")(s.PromoteStudentsHandler()))

	// Abgänger (Graduates)
	mux.Handle("GET /api/abgaenger", s.RequirePermission("view_graduates")(s.GetGraduatesHandler()))
	mux.Handle("GET /api/abgaenger/pdf", s.RequirePermission("view_graduates")(s.GetGraduatesPDFHandler()))
	// Versand ist mehr als Lesen: dasselbe Recht wie der Mahnlauf (create_orders),
	// nicht das reine view_graduates der Liste.
	mux.Handle("POST /api/abgaenger/mail", s.RequirePermission("create_orders")(s.SendAbgaengerKontoauszuegeHandler()))

	// Kiosk / Damage
	damageRepo := repository.NewDamageRepository(s.DB.Pool)
	mux.Handle("POST /api/damage/report", s.RequirePermission("edit_students")(s.ReportDamageHandler(damageRepo)))
	mux.Handle("GET /api/schadensfaelle/{id}/pdf", s.RequirePermission("view_students")(s.GenerateDamagePDFHandler()))
	// Gebühren-Erledigung: Lesen wie die Akte (view_students), Erledigen wie das
	// Anlegen (edit_students) — Begründung im Kopf von damage_resolve.go.
	auditRepoGebuehren := repository.NewAuditRepository(s.DB.Pool)
	mux.Handle("GET /api/schueler/{id}/schadensfaelle", s.RequirePermission("view_students")(s.ListStudentSchadensfaelleHandler(damageRepo)))
	mux.Handle("POST /api/schadensfaelle/{id}/bezahlt", s.RequirePermission("edit_students")(s.BezahltGebuehrHandler(auditRepoGebuehren)))
	mux.Handle("POST /api/schadensfaelle/{id}/storno", s.RequirePermission("edit_students")(s.StornoGebuehrHandler(auditRepoGebuehren)))

	// Mahnwesen
	mux.Handle("GET /api/mahnwesen", s.RequirePermission("view_students")(s.GetMahnwesenHandler(mahnRepo)))
	mux.Handle("GET /api/mahnwesen/ueberfaellig_jahrgang", s.RequirePermission("view_students")(s.GetMahnwesenJahrgangHandler(mahnRepo)))
	mux.Handle("GET /api/mahnwesen/pdf", s.RequirePermission("view_students")(s.GetMahnwesenPDFHandler(mahnRepo)))
	mux.Handle("POST /api/mahnwesen/senden", s.RequirePermission("create_orders")(s.SendMahnwesenHandler(mahnRepo)))
	// Massenversand: je überfällige Klasse eine Mahnliste an die Klassenleitung
	// (privacy-by-design — nie direkt an Schüler; siehe mahnwesen_bulk_mail.go).
	mux.Handle("POST /api/mail/send-bulk-overdue", s.RequirePermission("create_orders")(s.SendBulkOverdueHandler(mahnRepo)))
	// Hinweis: GET /api/reports/overdue-pdf (Eltern-Mahnbriefe) wird in registerSystemRoutes
	// über GetOverdueReportsPDFHandler registriert — hier KEIN Alias (sonst doppelte
	// Mux-Registrierung → Panic beim Start, zudem falscher Handler = Mahnliste statt Elternbriefe).
}
