package api

import (
	"bibliothek/repository"
	"net/http"
)

func (s *Server) registriereLeserRouten(mux *http.ServeMux, leserAbfragen repository.LeserAbfragen, mahnAbfragen *repository.MahnwesenAbfragen, protokollAbfragen repository.ProtokollAbfragen) {
	// ── SCHUELER (Students) ──
	// Die Klassenliste hinter Jahrgangsfilter und Jahrgangs-Auswahlfeld (OFFEN.md 9.5).
	lmfAbfragen := repository.NewLMFAbfragen(s.DB.Pool)
	mux.Handle("GET /api/schueler", s.verlangeRecht("view_students")(s.leser().Leserliste(leserAbfragen, lmfAbfragen)))
	mux.Handle("GET /api/schueler/{id}", s.verlangeRecht("view_students")(s.leser().Leserakte(leserAbfragen)))
	mux.Handle("POST /api/schueler", s.verlangeRecht("create_students")(s.leser().LeserAnlegen()))

	// Klebeetiketten der markierten Schüler (Name, Klasse, Barcode). view_students wie
	// der Ausweis-Stapeldruck: Es ist derselbe Vorgang mit anderem Papier, und die
	// Angaben darauf stehen ohnehin schon in der Liste, aus der markiert wurde.
	mux.Handle("POST /api/print/schueler-etiketten", s.verlangeRecht("view_students")(s.druck().SchuelerEtiketten(leserAbfragen)))
	// Ändern und Sperren hängen an edit_students, nicht mehr an create_students:
	// Stammdaten pflegen ist nicht dasselbe wie jemanden neu anlegen, und
	// edit_students war zwar geseedet und an /api/damage/report gebunden, in der
	// Rechte-Oberfläche aber gar nicht sichtbar (Audit-Befund 01.08.2026).
	// Ungefährlich: Beide Rechte sind für alle Rollen identisch geseedet
	// (ADMIN/MITARBEITER ja, LEHRER/HELFER nein) — niemand gewinnt oder verliert Zugriff.
	mux.Handle("PATCH /api/schueler/{id}", s.verlangeRecht("edit_students")(s.leser().LeserAendern(protokollAbfragen)))
	mux.Handle("PATCH /api/admin/students/{id}/lock", s.verlangeRecht("edit_students")(s.leser().LeserSperren(protokollAbfragen)))
	mux.Handle("DELETE /api/schueler/{id}", s.verlangeRecht("delete_students")(s.leser().LeserLoeschen(protokollAbfragen)))

	// DSGVO-Betroffenenauskunft (Art. 15) — manage_students_admin (seit 24.08.2026, vorher manage_users):
	// Der Export bündelt sämtliche personenbezogenen Daten eines Lesers. Zeigt auf ihn ein
	// Zugangskonto, verlangt der Handler zusätzlich manage_users (dsgvoKontoRecht, 28.09.2026).
	mux.Handle("GET /api/schueler/{id}/dsgvo-auskunft", s.verlangeRecht("manage_students_admin")(s.leser().DSGVOAuskunft()))
	mux.Handle("GET /api/schueler/{id}/dsgvo-auskunft/pdf", s.verlangeRecht("manage_students_admin")(s.leser().DSGVOAuskunftPDF()))

	// Zusammenführen zweier Datensätze (Umbenennung ohne Schüler-ID, Dublette aus
	// Handanlage + Import): Admin-Recht wie Purge — unumkehrbar, Identitätseingriff.
	mux.Handle("POST /api/schueler/{id}/zusammenfuehren", s.verlangeRecht("merge_students")(s.leser().LeserZusammenfuehren(protokollAbfragen)))
	mux.Handle("GET /api/schueler/{id}/zusammenfuehren-kandidaten", s.verlangeRecht("merge_students")(s.leser().Zusammenfuehrungskandidaten()))

	// Papierkorb
	mux.Handle("GET /api/schueler/deleted", s.verlangeRecht("delete_students")(s.leser().Papierkorb()))
	mux.Handle("POST /api/schueler/{id}/restore", s.verlangeRecht("delete_students")(s.leser().LeserWiederherstellen()))
	// Endgültig löschen (DSGVO-Purge): getrennte, stärkere Berechtigung als Soft-Delete.
	mux.Handle("DELETE /api/schueler/deleted/{id}", s.verlangeRecht("manage_students_admin")(s.leser().LeserEndgueltigLoeschen(protokollAbfragen)))

	// Photos — die beiden Richtungen sprechen bewusst UNTERSCHIEDLICHE Kennungen an:
	// hochgeladen wird über die Schüler-UUID (der Aufrufer kennt den Datensatz),
	// ausgeliefert über den Barcode. Letzteres ist kein Zufall: leseFotoURL
	// (api/leser/leserakte.go), leser_liste.go und foto.go bauen
	// die Bild-URL alle drei als /api/schueler/<barcode>/photo, und das Frontend rendert
	// sie unverändert als <img src>.
	//
	// Deshalb heißt der Platzhalter im GET {barcode_id} und nicht {id}: Die globale
	// uuidPfadMiddleware prüft ausschließlich einen Parameter namens "id" und
	// wies jeden echten Barcode (z. B. "S-abg1-ms3p7309e19de436") mit 400 "ungültiges
	// UUID Format" ab — verifiziert am laufenden System. Kein einziges Schülerfoto wurde
	// je ausgeliefert; die URL blieb dieselbe, nur die Prüfung passte nicht zu ihr.
	mux.Handle("POST /api/schueler/{id}/photo", s.verlangeRecht("upload_photos")(s.leser().FotoHochladen()))
	mux.Handle("GET /api/schueler/{barcode_id}/photo", s.verlangeRecht("view_students")(s.leser().Foto()))

	// Klassen
	mux.Handle("GET /api/klassen", s.verlangeRecht("view_students")(s.leser().Klassen(leserAbfragen)))
	// Die besetzten Jahrgänge fürs Filter-Auswahlfeld der Leserdatei (OFFEN.md 9.5).
	mux.Handle("GET /api/jahrgaenge", s.verlangeRecht("view_students")(s.leser().Jahrgaenge(lmfAbfragen)))
	mux.Handle("GET /api/klassen-mapping", s.verlangeRecht("manage_settings")(s.mahnwesen().Klassenleitungen()))
	mux.Handle("POST /api/klassen-mapping", s.verlangeRecht("manage_settings")(s.mahnwesen().KlassenleitungSetzen()))
	mux.Handle("DELETE /api/klassen-mapping/{klasse}", s.verlangeRecht("manage_settings")(s.mahnwesen().KlassenleitungLoeschen()))

	// LUSD Import
	// import_students statt manage_users: Das Recht war geseedet und in der
	// Rechte-Oberfläche als „LUSD / CSV Import" sichtbar — steuerte aber KEINEN
	// einzigen Endpunkt (Audit-Befund 01.08.2026). Ein Schalter, der nichts schaltet,
	// ist schlimmer als keiner: Er behauptet eine Einstellung, die es nicht gibt.
	// Identisch geseedet zu manage_users für ADMIN und MITARBEITER, LEHRER und HELFER
	// bleiben ausgeschlossen — die Zuordnung ändert sich also für niemanden.
	mux.Handle("POST /api/lusd/preview", s.verlangeRecht("import_students")(s.leser().LUSDVorschau()))
	mux.Handle("POST /api/lusd/import", s.verlangeRecht("import_students")(s.leser().LUSDImport()))

	// Promotion
	mux.Handle("POST /api/students/promote", s.verlangeRecht("manage_students_admin")(s.leser().Schuljahreswechsel()))

	// Abgänger (Graduates)
	mux.Handle("GET /api/abgaenger", s.verlangeRecht("view_graduates")(s.mahnwesen().Abgaenger()))
	mux.Handle("GET /api/abgaenger/pdf", s.verlangeRecht("view_graduates")(s.mahnwesen().AbgaengerKontoauszuegePDF()))
	// Versand ist mehr als Lesen: dasselbe Recht wie der Mahnlauf (create_orders),
	// nicht das reine view_graduates der Liste.
	mux.Handle("POST /api/abgaenger/mail", s.verlangeRecht("create_orders")(s.mahnwesen().AbgaengerKontoauszuegeVersenden()))

	// Kiosk / Damage
	schadensAbfragen := repository.NewSchadensAbfragen(s.DB.Pool)
	mux.Handle("POST /api/damage/report", s.verlangeRecht("edit_students")(s.schaden().SchadenMelden(schadensAbfragen)))
	mux.Handle("GET /api/schadensfaelle/{id}/pdf", s.verlangeRecht("view_students")(s.schaden().ElternbriefPDF()))
	// Gebühren-Erledigung: Lesen wie die Akte (view_students), Erledigen wie das
	// Anlegen (edit_students) — Begründung im Kopf von api/schaden/forderung_erledigen.go.
	protokollDerBescheide := repository.NewProtokollAbfragen(s.DB.Pool)
	mux.Handle("GET /api/schueler/{id}/schadensfaelle", s.verlangeRecht("view_students")(s.schaden().SchadensfaelleDesSchuelers(schadensAbfragen)))

	// ── SCHADENSERSATZ-BESCHEIDE (Migration 110) ──
	//
	// Rechte wie an den übrigen Forderungs-Routen: LESEN mit view_students (die Akte
	// zeigt die Briefe eines Schülers, die Arbeitsliste die offenen), SCHREIBEN mit
	// edit_students — wer eine Forderung anlegen und stornieren darf, schreibt auch den
	// Brief dazu. Ein eigenes Recht bliebe ab Werk bei niemandem und wäre eine Tür, die
	// keiner öffnen kann; die Entscheidung steht als E-Frage im Konzept.
	bescheidAbfragen := repository.NewBescheidAbfragen(s.DB.Pool)
	mux.Handle("GET /api/schueler/{id}/bescheid-vorschlag", s.verlangeRecht("edit_students")(s.schaden().BescheidVorschlag(bescheidAbfragen)))
	// Der Betragsvorschlag des Melde-Dialogs (OFFEN.md 9.3 a): dasselbe Recht wie das
	// Melden selbst — wer die Forderung anlegen darf, darf auch den Vorschlag dafür sehen.
	mux.Handle("GET /api/buecher/exemplare/{id}/ersatzwert-vorschlag", s.verlangeRecht("edit_students")(s.schaden().ErsatzwertVorschlag(bescheidAbfragen)))
	mux.Handle("GET /api/schueler/{id}/bescheide", s.verlangeRecht("view_students")(s.schaden().BescheideDesSchuelers(bescheidAbfragen)))
	mux.Handle("POST /api/schueler/{id}/bescheide", s.verlangeRecht("edit_students")(s.schaden().BescheidErstellen(bescheidAbfragen, protokollDerBescheide)))
	mux.Handle("GET /api/bescheide", s.verlangeRecht("view_students")(s.schaden().Bescheide(bescheidAbfragen)))
	mux.Handle("GET /api/bescheide/ausstehend", s.verlangeRecht("view_students")(s.schaden().BescheideAusstehend(bescheidAbfragen)))
	mux.Handle("GET /api/bescheide/{id}/pdf", s.verlangeRecht("view_students")(s.schaden().BescheidPDF(bescheidAbfragen)))
	mux.Handle("POST /api/bescheide/{id}/uebergeben", s.verlangeRecht("edit_students")(s.schaden().BescheidUebergeben(bescheidAbfragen, protokollDerBescheide)))
	mux.Handle("POST /api/schadensfaelle/{id}/bezahlt", s.verlangeRecht("edit_students")(s.schaden().ZahlungVerbuchen(protokollDerBescheide)))
	mux.Handle("POST /api/schadensfaelle/{id}/storno", s.verlangeRecht("edit_students")(s.schaden().ForderungStornieren(protokollDerBescheide)))

	// Mahnwesen
	mux.Handle("GET /api/mahnwesen", s.verlangeRecht("view_students")(s.mahnwesen().UeberfaelligeAusleihen(mahnAbfragen)))
	// Massenversand: je überfällige Klasse eine Mahnliste an die Klassenleitung
	// (privacy-by-design — nie direkt an Schüler; siehe api/mahnwesen/mahnlisten_versand.go).
	mux.Handle("POST /api/mail/send-bulk-overdue", s.verlangeRecht("create_orders")(s.mahnwesen().MahnlistenVersenden(mahnAbfragen)))
}
