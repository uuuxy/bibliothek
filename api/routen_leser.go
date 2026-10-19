package api

import (
	"bibliothek/repository"
	"net/http"
)

func (s *Server) registriereLeserRouten(mux *http.ServeMux, leserAbfragen *repository.LeserAbfragen, mahnAbfragen *repository.MahnwesenAbfragen, protokollAbfragen *repository.ProtokollAbfragen) {
	// ── Leser ──
	// Die Klassenliste hinter Jahrgangsfilter und Jahrgangs-Auswahlfeld.
	lmfAbfragen := repository.NewLMFAbfragen(s.DB.Pool)
	mux.Handle("GET /api/schueler", s.verlangeRecht("view_students")(s.leser().Leserliste(leserAbfragen, lmfAbfragen)))
	mux.Handle("GET /api/schueler/{id}", s.verlangeRecht("view_students")(s.leser().Leserakte(leserAbfragen)))
	mux.Handle("POST /api/schueler", s.verlangeRecht("create_students")(s.leser().LeserAnlegen()))

	// Klebeetiketten der markierten Schüler (Name, Klasse, Barcode). view_students wie
	// der Ausweis-Stapeldruck: Es ist derselbe Vorgang mit anderem Papier, und die
	// Angaben darauf stehen ohnehin schon in der Liste, aus der markiert wurde.
	mux.Handle("POST /api/print/schueler-etiketten", s.verlangeRecht("view_students")(s.druck().SchuelerEtiketten(leserAbfragen)))
	// Ändern und Sperren hängen an edit_students, nicht an create_students: Stammdaten pflegen
	// ist nicht dasselbe wie jemanden neu anlegen. Beide Rechte haben ab Werk dieselben Rollen
	// (Admin, Leitung und Mitarbeiter ja, Kollegium und Helfer nein; db/rechte_vorgabe.go).
	mux.Handle("PATCH /api/schueler/{id}", s.verlangeRecht("edit_students")(s.leser().LeserAendern(protokollAbfragen)))
	mux.Handle("PATCH /api/admin/students/{id}/lock", s.verlangeRecht("edit_students")(s.leser().LeserSperren(protokollAbfragen)))
	mux.Handle("DELETE /api/schueler/{id}", s.verlangeRecht("delete_students")(s.leser().LeserLoeschen(leserAbfragen)))

	// Die Auskunft nach Art. 15 DSGVO hängt an manage_students_admin: Sie bündelt alle
	// personenbezogenen Daten eines Lesers. Zeigt auf ihn ein Zugangskonto, verlangt die Tür
	// zusätzlich manage_users (dsgvoKontoRecht).
	mux.Handle("GET /api/schueler/{id}/dsgvo-auskunft", s.verlangeRecht("manage_students_admin")(s.leser().DSGVOAuskunft()))
	mux.Handle("GET /api/schueler/{id}/dsgvo-auskunft/pdf", s.verlangeRecht("manage_students_admin")(s.leser().DSGVOAuskunftPDF()))

	// Zusammenführen zweier Datensätze (umbenannt ohne Schüler-ID, Dublette aus Handanlage und
	// Import): das Recht der Verwaltung wie beim endgültigen Löschen, weil es nicht umkehrbar ist
	// und in die Identität eingreift.
	mux.Handle("POST /api/schueler/{id}/zusammenfuehren", s.verlangeRecht("merge_students")(s.leser().LeserZusammenfuehren(protokollAbfragen)))
	mux.Handle("GET /api/schueler/{id}/zusammenfuehren-kandidaten", s.verlangeRecht("merge_students")(s.leser().Zusammenfuehrungskandidaten()))

	// Papierkorb
	mux.Handle("GET /api/schueler/deleted", s.verlangeRecht("delete_students")(s.leser().Papierkorb()))
	mux.Handle("POST /api/schueler/{id}/restore", s.verlangeRecht("delete_students")(s.leser().LeserWiederherstellen()))
	// Endgültig löschen: ein eigenes, stärkeres Recht als das Löschen in den Papierkorb.
	mux.Handle("DELETE /api/schueler/deleted/{id}", s.verlangeRecht("manage_students_admin")(s.leser().LeserEndgueltigLoeschen(leserAbfragen)))

	// Fotos: Die beiden Richtungen sprechen verschiedene Kennungen an. Hochgeladen wird über die
	// UUID des Lesers (der Aufrufer kennt den Datensatz), ausgeliefert über den Barcode:
	// leseFotoURL (api/leser/leserakte.go) baut die Adresse als /api/schueler/<barcode>/photo, und
	// die Oberfläche setzt sie unverändert als <img src>.
	//
	// Deshalb heißt der Platzhalter im GET {barcode_id} und nicht {id}: uuidPfadMiddleware prüft
	// die Parameter in uuidPfadParameter, darunter "id", und wiese jeden echten Barcode (etwa
	// "S-abg1-ms3p7309e19de436") mit 400 ab; kein Foto käme an.
	mux.Handle("POST /api/schueler/{id}/photo", s.verlangeRecht("upload_photos")(s.leser().FotoHochladen()))
	mux.Handle("GET /api/schueler/{barcode_id}/photo", s.verlangeRecht("view_students")(s.leser().Foto()))

	// Klassen
	mux.Handle("GET /api/klassen", s.verlangeRecht("view_students")(s.leser().Klassen(leserAbfragen)))
	// Die besetzten Jahrgänge für das Auswahlfeld des Filters in der Leserdatei.
	mux.Handle("GET /api/jahrgaenge", s.verlangeRecht("view_students")(s.leser().Jahrgaenge(lmfAbfragen)))
	mux.Handle("GET /api/klassen-mapping", s.verlangeRecht("manage_settings")(s.mahnwesen().Klassenleitungen()))
	mux.Handle("POST /api/klassen-mapping", s.verlangeRecht("manage_settings")(s.mahnwesen().KlassenleitungSetzen()))
	mux.Handle("DELETE /api/klassen-mapping/{klasse}", s.verlangeRecht("manage_settings")(s.mahnwesen().KlassenleitungLoeschen()))

	// Import aus der LUSD: import_students statt manage_users, das Recht, das die Rechte-Oberfläche
	// als „LUSD / CSV Import" zeigt. Ein Schalter, der nichts schaltet, wäre schlimmer als keiner:
	// Er behauptete eine Einstellung, die es nicht gibt. Ab Werk haben es Admin, Leitung und
	// Mitarbeiter (db/rechte_vorgabe.go).
	mux.Handle("POST /api/lusd/preview", s.verlangeRecht("import_students")(s.leser().LUSDVorschau()))
	mux.Handle("POST /api/lusd/import", s.verlangeRecht("import_students")(s.leser().LUSDImport()))

	// Schuljahreswechsel
	mux.Handle("POST /api/students/promote", s.verlangeRecht("manage_students_admin")(s.leser().Schuljahreswechsel()))

	// Abgänger
	mux.Handle("GET /api/abgaenger", s.verlangeRecht("view_graduates")(s.mahnwesen().Abgaenger()))
	mux.Handle("GET /api/abgaenger/pdf", s.verlangeRecht("view_graduates")(s.mahnwesen().AbgaengerKontoauszuegePDF()))
	// Versand ist mehr als Lesen: dasselbe Recht wie der Mahnlauf (create_orders),
	// nicht das reine view_graduates der Liste.
	mux.Handle("POST /api/abgaenger/mail", s.verlangeRecht("create_orders")(s.mahnwesen().AbgaengerKontoauszuegeVersenden()))

	// Schäden
	schadensAbfragen := repository.NewSchadensAbfragen(s.DB.Pool)
	mux.Handle("POST /api/damage/report", s.verlangeRecht("edit_students")(s.schaden().SchadenMelden(schadensAbfragen)))
	// Gebühren-Erledigung: Lesen wie die Akte (view_students), Erledigen wie das
	// Anlegen (edit_students); die Begründung steht im Kopf von api/schaden/forderung_erledigen.go.
	protokollDerBescheide := repository.NewProtokollAbfragen(s.DB.Pool)
	mux.Handle("GET /api/schueler/{id}/schadensfaelle", s.verlangeRecht("view_students")(s.schaden().SchadensfaelleDesSchuelers(schadensAbfragen)))

	// ── Bescheide über Schadensersatz ──
	//
	// Rechte wie an den übrigen Routen der Forderungen: lesen mit view_students (die Akte zeigt
	// die Briefe eines Schülers, die Arbeitsliste die offenen), schreiben mit edit_students; wer
	// eine Forderung anlegen und stornieren darf, schreibt auch den Brief dazu. Ein eigenes Recht
	// bliebe ab Werk bei niemandem und wäre eine Tür, die keiner öffnen kann.
	bescheidAbfragen := repository.NewBescheidAbfragen(s.DB.Pool)
	mux.Handle("GET /api/schueler/{id}/bescheid-vorschlag", s.verlangeRecht("edit_students")(s.schaden().BescheidVorschlag(bescheidAbfragen)))
	// Der Vorschlag für den Betrag im Dialog zum Melden: dasselbe Recht wie das Melden selbst;
	// wer die Forderung anlegen darf, darf auch den Vorschlag dafür sehen.
	mux.Handle("GET /api/buecher/exemplare/{id}/ersatzwert-vorschlag", s.verlangeRecht("edit_students")(s.schaden().ErsatzwertVorschlag(bescheidAbfragen)))
	mux.Handle("GET /api/schueler/{id}/bescheide", s.verlangeRecht("view_students")(s.schaden().BescheideDesSchuelers(bescheidAbfragen)))
	mux.Handle("POST /api/schueler/{id}/bescheide", s.verlangeRecht("edit_students")(s.schaden().BescheidErstellen(bescheidAbfragen, protokollDerBescheide)))
	mux.Handle("GET /api/bescheide", s.verlangeRecht("view_students")(s.schaden().Bescheide(bescheidAbfragen)))
	mux.Handle("GET /api/bescheide/ausstehend", s.verlangeRecht("view_students")(s.schaden().BescheideAusstehend(bescheidAbfragen)))
	mux.Handle("GET /api/bescheide/{id}/pdf", s.verlangeRecht("view_students")(s.schaden().BescheidPDF(bescheidAbfragen)))
	mux.Handle("POST /api/bescheide/{id}/uebergeben", s.verlangeRecht("edit_students")(s.schaden().BescheidUebergeben(bescheidAbfragen, protokollDerBescheide)))
	mux.Handle("POST /api/schadensfaelle/{id}/bezahlt", s.verlangeRecht("edit_students")(s.schaden().ZahlungVerbuchen(schadensAbfragen)))
	mux.Handle("POST /api/schadensfaelle/{id}/storno", s.verlangeRecht("edit_students")(s.schaden().ForderungStornieren(schadensAbfragen)))

	// Mahnwesen
	mux.Handle("GET /api/mahnwesen", s.verlangeRecht("view_students")(s.mahnwesen().UeberfaelligeAusleihen(mahnAbfragen)))
	// Massenversand: je überfällige Klasse eine Mahnliste an die Klassenleitung
	// (nie direkt an Schüler, Datenschutz durch Gestaltung; api/mahnwesen/mahnlisten_versand.go).
	mux.Handle("POST /api/mail/send-bulk-overdue", s.verlangeRecht("create_orders")(s.mahnwesen().MahnlistenVersenden(mahnAbfragen)))
}
