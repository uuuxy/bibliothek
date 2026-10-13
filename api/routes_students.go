package api

import (
	"bibliothek/repository"
	"net/http"
)

func (s *Server) registerStudentRoutes(mux *http.ServeMux, studentRepo repository.LeserAbfragen, mahnRepo *repository.MahnwesenAbfragen, auditRepo repository.ProtokollAbfragen) {
	// ── SCHUELER (Students) ──
	// Die Klassenliste hinter Jahrgangsfilter und Jahrgangs-Auswahlfeld (OFFEN.md 9.5).
	lmfKlassenRepo := repository.NewLMFAbfragen(s.DB.Pool)
	mux.Handle("GET /api/schueler", s.RequirePermission("view_students")(s.leser().ListStudentsHandler(studentRepo, lmfKlassenRepo)))
	mux.Handle("GET /api/schueler/{id}", s.RequirePermission("view_students")(s.leser().GetStudentProfileHandler(studentRepo)))
	mux.Handle("POST /api/schueler", s.RequirePermission("create_students")(s.leser().CreateStudentHandler()))

	// Klebeetiketten der markierten Schüler (Name, Klasse, Barcode). view_students wie
	// der Ausweis-Stapeldruck: Es ist derselbe Vorgang mit anderem Papier, und die
	// Angaben darauf stehen ohnehin schon in der Liste, aus der markiert wurde.
	mux.Handle("POST /api/print/schueler-etiketten", s.RequirePermission("view_students")(s.druck().PrintSchuelerEtikettenHandler(studentRepo)))
	// Ändern und Sperren hängen an edit_students, nicht mehr an create_students:
	// Stammdaten pflegen ist nicht dasselbe wie jemanden neu anlegen, und
	// edit_students war zwar geseedet und an /api/damage/report gebunden, in der
	// Rechte-Oberfläche aber gar nicht sichtbar (Audit-Befund 01.08.2026).
	// Ungefährlich: Beide Rechte sind für alle Rollen identisch geseedet
	// (ADMIN/MITARBEITER ja, LEHRER/HELFER nein) — niemand gewinnt oder verliert Zugriff.
	mux.Handle("PATCH /api/schueler/{id}", s.RequirePermission("edit_students")(s.leser().PatchStudentHandler(auditRepo)))
	mux.Handle("PATCH /api/admin/students/{id}/lock", s.RequirePermission("edit_students")(s.leser().LockStudentHandler(auditRepo)))
	mux.Handle("DELETE /api/schueler/{id}", s.RequirePermission("delete_students")(s.leser().DeleteStudentHandler(auditRepo)))

	// DSGVO-Betroffenenauskunft (Art. 15) — manage_students_admin (seit 24.08.2026, vorher manage_users):
	// Der Export bündelt sämtliche personenbezogenen Daten eines Lesers. Zeigt auf ihn ein
	// Zugangskonto, verlangt der Handler zusätzlich manage_users (dsgvoKontoRecht, 28.09.2026).
	mux.Handle("GET /api/schueler/{id}/dsgvo-auskunft", s.RequirePermission("manage_students_admin")(s.leser().DsgvoAuskunftHandler()))
	mux.Handle("GET /api/schueler/{id}/dsgvo-auskunft/pdf", s.RequirePermission("manage_students_admin")(s.leser().DsgvoAuskunftPDFHandler()))

	// Zusammenführen zweier Datensätze (Umbenennung ohne Schüler-ID, Dublette aus
	// Handanlage + Import): Admin-Recht wie Purge — unumkehrbar, Identitätseingriff.
	mux.Handle("POST /api/schueler/{id}/zusammenfuehren", s.RequirePermission("merge_students")(s.leser().ZusammenfuehrenSchuelerHandler(auditRepo)))
	mux.Handle("GET /api/schueler/{id}/zusammenfuehren-kandidaten", s.RequirePermission("merge_students")(s.leser().ZusammenfuehrenKandidatenHandler()))

	// Papierkorb
	mux.Handle("GET /api/schueler/deleted", s.RequirePermission("delete_students")(s.leser().GetDeletedStudentsHandler()))
	mux.Handle("POST /api/schueler/{id}/restore", s.RequirePermission("delete_students")(s.leser().RestoreStudentHandler()))
	// Endgültig löschen (DSGVO-Purge): getrennte, stärkere Berechtigung als Soft-Delete.
	mux.Handle("DELETE /api/schueler/deleted/{id}", s.RequirePermission("manage_students_admin")(s.leser().PurgeStudentHandler(auditRepo)))

	// Photos — die beiden Richtungen sprechen bewusst UNTERSCHIEDLICHE Kennungen an:
	// hochgeladen wird über die Schüler-UUID (der Aufrufer kennt den Datensatz),
	// ausgeliefert über den Barcode. Letzteres ist kein Zufall: resolveFotoURL
	// (api/leser/student_profile.go), leser_liste.go und photo_service.go bauen
	// die Bild-URL alle drei als /api/schueler/<barcode>/photo, und das Frontend rendert
	// sie unverändert als <img src>.
	//
	// Deshalb heißt der Platzhalter im GET {barcode_id} und nicht {id}: Die globale
	// ValidateUUIDParamsMiddleware prüft ausschließlich einen Parameter namens "id" und
	// wies jeden echten Barcode (z. B. "S-abg1-ms3p7309e19de436") mit 400 "ungültiges
	// UUID Format" ab — verifiziert am laufenden System. Kein einziges Schülerfoto wurde
	// je ausgeliefert; die URL blieb dieselbe, nur die Prüfung passte nicht zu ihr.
	mux.Handle("POST /api/schueler/{id}/photo", s.RequirePermission("upload_photos")(s.leser().UploadStudentPhotoHandler()))
	mux.Handle("GET /api/schueler/{barcode_id}/photo", s.RequirePermission("view_students")(s.leser().ServeStudentPhotoHandler()))

	// Klassen
	mux.Handle("GET /api/klassen", s.RequirePermission("view_students")(s.leser().GetClassesHandler(studentRepo)))
	// Die besetzten Jahrgänge fürs Filter-Auswahlfeld der Leserdatei (OFFEN.md 9.5).
	mux.Handle("GET /api/jahrgaenge", s.RequirePermission("view_students")(s.leser().JahrgaengeHandler(lmfKlassenRepo)))
	mux.Handle("GET /api/klassen-mapping", s.RequirePermission("manage_settings")(s.mahnwesen().GetKlassenMappingHandler()))
	mux.Handle("POST /api/klassen-mapping", s.RequirePermission("manage_settings")(s.mahnwesen().UpsertKlassenMappingHandler()))
	mux.Handle("DELETE /api/klassen-mapping/{klasse}", s.RequirePermission("manage_settings")(s.mahnwesen().DeleteKlassenMappingHandler()))

	// LUSD Import
	// import_students statt manage_users: Das Recht war geseedet und in der
	// Rechte-Oberfläche als „LUSD / CSV Import" sichtbar — steuerte aber KEINEN
	// einzigen Endpunkt (Audit-Befund 01.08.2026). Ein Schalter, der nichts schaltet,
	// ist schlimmer als keiner: Er behauptet eine Einstellung, die es nicht gibt.
	// Identisch geseedet zu manage_users für ADMIN und MITARBEITER, LEHRER und HELFER
	// bleiben ausgeschlossen — die Zuordnung ändert sich also für niemanden.
	mux.Handle("POST /api/lusd/preview", s.RequirePermission("import_students")(s.leser().PostLusdPreviewHandler()))
	mux.Handle("POST /api/lusd/import", s.RequirePermission("import_students")(s.leser().PostLusdImportHandler()))

	// Promotion
	mux.Handle("POST /api/students/promote", s.RequirePermission("manage_students_admin")(s.leser().PromoteStudentsHandler()))

	// Abgänger (Graduates)
	mux.Handle("GET /api/abgaenger", s.RequirePermission("view_graduates")(s.mahnwesen().GetGraduatesHandler()))
	mux.Handle("GET /api/abgaenger/pdf", s.RequirePermission("view_graduates")(s.mahnwesen().GetGraduatesPDFHandler()))
	// Versand ist mehr als Lesen: dasselbe Recht wie der Mahnlauf (create_orders),
	// nicht das reine view_graduates der Liste.
	mux.Handle("POST /api/abgaenger/mail", s.RequirePermission("create_orders")(s.mahnwesen().SendAbgaengerKontoauszuegeHandler()))

	// Kiosk / Damage
	damageRepo := repository.NewSchadensAbfragen(s.DB.Pool)
	mux.Handle("POST /api/damage/report", s.RequirePermission("edit_students")(s.schaden().ReportDamageHandler(damageRepo)))
	mux.Handle("GET /api/schadensfaelle/{id}/pdf", s.RequirePermission("view_students")(s.schaden().GenerateDamagePDFHandler()))
	// Gebühren-Erledigung: Lesen wie die Akte (view_students), Erledigen wie das
	// Anlegen (edit_students) — Begründung im Kopf von api/schaden/damage_resolve.go.
	auditRepoGebuehren := repository.NewProtokollAbfragen(s.DB.Pool)
	mux.Handle("GET /api/schueler/{id}/schadensfaelle", s.RequirePermission("view_students")(s.schaden().ListStudentSchadensfaelleHandler(damageRepo)))

	// ── SCHADENSERSATZ-BESCHEIDE (Migration 110) ──
	//
	// Rechte wie an den übrigen Forderungs-Routen: LESEN mit view_students (die Akte
	// zeigt die Briefe eines Schülers, die Arbeitsliste die offenen), SCHREIBEN mit
	// edit_students — wer eine Forderung anlegen und stornieren darf, schreibt auch den
	// Brief dazu. Ein eigenes Recht bliebe ab Werk bei niemandem und wäre eine Tür, die
	// keiner öffnen kann; die Entscheidung steht als E-Frage im Konzept.
	bescheidRepo := repository.NewBescheidAbfragen(s.DB.Pool)
	mux.Handle("GET /api/schueler/{id}/bescheid-vorschlag", s.RequirePermission("edit_students")(s.schaden().BescheidVorschlagHandler(bescheidRepo)))
	// Der Betragsvorschlag des Melde-Dialogs (OFFEN.md 9.3 a): dasselbe Recht wie das
	// Melden selbst — wer die Forderung anlegen darf, darf auch den Vorschlag dafür sehen.
	mux.Handle("GET /api/buecher/exemplare/{id}/ersatzwert-vorschlag", s.RequirePermission("edit_students")(s.schaden().ErsatzwertVorschlagHandler(bescheidRepo)))
	mux.Handle("GET /api/schueler/{id}/bescheide", s.RequirePermission("view_students")(s.schaden().BescheidSchuelerListeHandler(bescheidRepo)))
	mux.Handle("POST /api/schueler/{id}/bescheide", s.RequirePermission("edit_students")(s.schaden().BescheidErstellenHandler(bescheidRepo, auditRepoGebuehren)))
	mux.Handle("GET /api/bescheide", s.RequirePermission("view_students")(s.schaden().BescheidListeHandler(bescheidRepo)))
	mux.Handle("GET /api/bescheide/ausstehend", s.RequirePermission("view_students")(s.schaden().BescheidAusstehendHandler(bescheidRepo)))
	mux.Handle("GET /api/bescheide/{id}/pdf", s.RequirePermission("view_students")(s.schaden().BescheidPDFHandler(bescheidRepo)))
	mux.Handle("POST /api/bescheide/{id}/uebergeben", s.RequirePermission("edit_students")(s.schaden().BescheidUebergebenHandler(bescheidRepo, auditRepoGebuehren)))
	mux.Handle("POST /api/schadensfaelle/{id}/bezahlt", s.RequirePermission("edit_students")(s.schaden().BezahltGebuehrHandler(auditRepoGebuehren)))
	mux.Handle("POST /api/schadensfaelle/{id}/storno", s.RequirePermission("edit_students")(s.schaden().StornoGebuehrHandler(auditRepoGebuehren)))

	// Mahnwesen
	mux.Handle("GET /api/mahnwesen", s.RequirePermission("view_students")(s.mahnwesen().GetMahnwesenHandler(mahnRepo)))
	// Massenversand: je überfällige Klasse eine Mahnliste an die Klassenleitung
	// (privacy-by-design — nie direkt an Schüler; siehe api/mahnwesen/mahnwesen_bulk_mail.go).
	mux.Handle("POST /api/mail/send-bulk-overdue", s.RequirePermission("create_orders")(s.mahnwesen().SendBulkOverdueHandler(mahnRepo)))
}
