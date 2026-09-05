package api

import (
	"bibliothek/repository"
	"net/http"
)

func (s *Server) registerBookRoutes(mux *http.ServeMux, bookRepo repository.BookRepository, auditRepo repository.AuditRepository) {
	// ── BUECHER (Titles & Copies) ──
	mux.Handle("DELETE /api/buecher/titel/{id}", s.RequirePermission("delete_books")(s.DeleteTitleHandler(auditRepo)))
	// Bestellkorb-Korrektur der Signatur (create_orders, nicht edit_books — dieselbe
	// Berechtigung wie das Anlegen des Titels über /aus-isbn, siehe api/isbn_handler.go).
	mux.Handle("PUT /api/buecher/titel/{id}/signatur", s.RequirePermission("create_orders")(s.UpdateTitelSignaturHandler()))

	// Exemplare (Copies)
	// Titel-Tür für Bildschirme außerhalb der Theke (Etiketten-Titelsuche im Druck-Center):
	// dieselbe Suche wie GET /api/search, aber nur Titel und hinter view_books statt
	// perform_actions (05.09.2026, Befund-Register Entscheidung 3).
	mux.Handle("GET /api/buecher/titel/suche", s.RequirePermission("view_books")(s.TitelSucheHandler(bookRepo)))
	mux.Handle("GET /api/buecher/titel/{id}/exemplare", s.RequirePermission("view_books")(s.GetTitleCopiesHandler()))
	// Ausleiher und Historie hinter view_students, nicht view_books: Beide
	// verknüpfen Schülernamen mit Titeln (die Historie über Jahre) — dieselbe
	// Befundklasse wie die Vormerkungen (bewertung/sicherheitsbefund-*.md),
	// dort aber von der Prüfung übersehen. Die Buchakte fängt den 403 als
	// leere Liste ab (jsonOrEmpty); Exemplare/Verfügbarkeit bleiben view_books.
	mux.Handle("GET /api/buecher/titel/{id}/ausleiher", s.RequirePermission("view_students")(s.GetTitleBorrowersHandler()))
	mux.Handle("GET /api/buecher/titel/{id}/historie", s.RequirePermission("view_students")(s.GetTitleHistoryHandler()))
	mux.Handle("GET /api/buecher/titel/{id}/etiketten", s.RequirePermission("view_books")(s.LabelsHandler()))
	mux.Handle("POST /api/print/labels", s.RequirePermission("view_books")(s.PrintLabelsHandler()))

	// Nachdruck: Welche Exemplare haben noch kein Etikett, und Gegenbuchung nach dem Druck.
	// edit_books statt view_books — die Gegenbuchung ändert Bestandsdaten.
	mux.Handle("GET /api/exemplare/etiketten-offen", s.RequirePermission("edit_books")(s.EtikettenOffenHandler()))
	mux.Handle("GET /api/exemplare/etiketten-offen/anzahl", s.RequirePermission("edit_books")(s.EtikettenOffenAnzahlHandler()))
	mux.Handle("POST /api/exemplare/etiketten-gedruckt", s.RequirePermission("edit_books")(s.EtikettenGedrucktHandler()))
	// Der Weg zurück nach Papierstau oder zu weitem Altbestands-Stichtag.
	mux.Handle("POST /api/exemplare/etiketten-zuruecksetzen", s.RequirePermission("edit_books")(s.EtikettenZuruecksetzenHandler()))
	mux.Handle("POST /api/exemplare/etiketten-altbestand", s.RequirePermission("edit_books")(s.EtikettenAltbestandHandler()))

	mux.Handle("DELETE /api/buecher/exemplare/{id}", s.RequirePermission("delete_books")(s.DeleteCopyHandler(auditRepo)))

	// Update specific copy fields
	damageRepo := repository.NewDamageRepository(s.DB.Pool)
	mux.Handle("POST /api/buecher/exemplare/{id}/schadensnotiz", s.RequirePermission("edit_books")(s.UpdateDamageNoteHandler(bookRepo)))
	mux.Handle("PUT /api/buecher/exemplare/{id}/barcode", s.RequirePermission("edit_books")(s.UpdateCopyBarcodeHandler(bookRepo)))
	mux.Handle("PUT /api/buecher/exemplare/{id}/status", s.RequirePermission("edit_books")(s.UpdateCopyStatusHandler(bookRepo)))
	mux.Handle("POST /api/buecher/exemplare/{id}/defekt", s.RequirePermission("edit_books")(s.MarkCopyDefektHandler(damageRepo)))
	mux.Handle("POST /api/buecher/exemplare/{id}/aussondern", s.RequirePermission("edit_books")(s.AussondernCopyHandler(bookRepo)))

	// ── AUSLEIHEN (Loans) ──
	settingsRepo := repository.NewSystemSettingsRepository(s.DB.Pool)
	mux.Handle("POST /api/ausleihen/{ausleihe_id}/verlaengern", s.RequirePermission("edit_books")(s.ExtendLoanHandler(settingsRepo)))
	mux.Handle("POST /api/ausleihen/global-extend-lmf", s.RequirePermission("edit_books")(s.GlobalExtendLMFHandler()))

	// ── LMF-PLAN (Rückgabe-/Ausgabetermine je Klasse, Migration 096) ──
	// Lesen: jede Sitzung (das Kollegium liest den Plan im Portal; Stufe 0 — Daten und
	// Klassennamen). Schreiben: edit_books wie die übrige Lernmittel-Pflege.
	mux.Handle("GET /api/lmf-termine", s.RequireAuthenticated()(s.GetLmfTermineHandler()))
	mux.Handle("GET /api/lmf-termine/pdf", s.RequireAuthenticated()(s.GetLmfPlanPDFHandler()))
	// Geschrieben wird der Plan als REIHENFOLGE (Migration 097, lmf_plan.go): ein Aufruf
	// je Art, Vorschau über denselben Aufruf. Einzel-Termin-Routen gibt es seit dem
	// 05.09.2026 abends nicht mehr — eine zweite Tür je Zeile gäbe zwei Wahrheiten.
	mux.Handle("GET /api/lmf-plan/{art}", s.RequirePermission("edit_books")(s.GetLmfPlanHandler()))
	mux.Handle("PUT /api/lmf-plan/{art}", s.RequirePermission("edit_books")(s.PutLmfPlanHandler()))
	mux.Handle("DELETE /api/lmf-plan/{art}", s.RequirePermission("edit_books")(s.DeleteLmfPlanHandler()))
	mux.Handle("PATCH /api/admin/ausleihen/{id}/faelligkeit", s.RequirePermission("edit_books")(s.OverrideDueDateHandler(auditRepo)))

	// Live ISBN Lookup
	mux.Handle("POST /api/buecher/aus-isbn", s.RequirePermission("create_orders")(s.ISBNZuTitelHandler()))

	// Vormerkungen
	vormerkungRepo := repository.NewVormerkungRepository(s.DB.Pool)
	// Eigenes enges Recht manage_vormerkungen, NICHT view_books (bewertung/
	// sicherheitsbefund-vormerkungen.md, Weg 2 — Betreiber-Entscheidung 18.08.2026):
	// Die Liste verknüpft Schülernamen mit Leseinteresse und erlaubt Anlegen/Löschen
	// für beliebige schueler_id. Sie zeigt aber nur Kiosk-Felder (Name, Klasse) —
	// dafür das volle view_students zu verlangen (Profile, Adressen, Mahnwesen) war
	// zu grob: Helfer verloren die Warteliste komplett. Jetzt: MITARBEITER hat das
	// Recht ab Werk, HELFER optional per Rechte-Matrix (Standard AUS, db/seed.go).
	// Der Einzelfall am Rückgabe-Scan ("reserviert für X") bleibt bewusst: den
	// Namen braucht die Theke, um das Buch für die richtige Person beiseitezulegen.
	mux.Handle("GET /api/vormerkungen", s.RequirePermission("manage_vormerkungen")(s.ListVormerkungHandler(vormerkungRepo)))
	mux.Handle("POST /api/vormerkungen", s.RequirePermission("manage_vormerkungen")(s.CreateVormerkungHandler(vormerkungRepo)))
	mux.Handle("DELETE /api/vormerkungen/{id}", s.RequirePermission("manage_vormerkungen")(s.DeleteVormerkungHandler(vormerkungRepo)))

	// Klassensatz Reservierungen
	//
	// create_reservations statt view_students (10.08.2026): Der Handler fasst keine
	// Schülerdaten an — er nimmt Titel-ID, Klasse als Freitext und Anzahl entgegen. Mit
	// view_students war das trotzdem gekoppelt: Wer im Kollegiums-Portal reservieren
	// dürfen sollte, brauchte dasselbe Recht, an dem Schülerdatei und Druck-Center
	// hängen — die einzige Portal-Funktion zog also die komplette Schülerdatei auf.
	mux.Handle("POST /api/reservierungen/klassensatz", s.RequirePermission("create_reservations")(s.CreateKlassensatzReservierungHandler()))
	mux.Handle("GET /api/reservierungen/klassensatz", s.RequirePermission("view_orders")(s.GetKlassensatzReservierungenHandler()))
	mux.Handle("GET /api/reservierungen/klassensatz/anzahl", s.RequirePermission("view_orders")(s.GetKlassensatzReservierungenAnzahlHandler()))
	// Warteschlangen-Sicht fürs Kollegium: dasselbe Recht wie das Anlegen — wer
	// reservieren darf, darf sehen, wer vor ihm dran ist.
	mux.Handle("GET /api/reservierungen/klassensatz/offen", s.RequirePermission("create_reservations")(s.OffeneKlassensatzReservierungenHandler()))
	mux.Handle("GET /api/reservierungen/klassensatz/eigene", s.RequirePermission("create_reservations")(s.MeineKlassensatzReservierungenHandler()))

	// Geräte-Verwaltung (Bereich im Medienkatalog): Geräte sind Bestand — Rechte wie Bücher.
	geraeteRepo := repository.NewGeraeteRepository(s.DB.Pool)
	mux.Handle("GET /api/geraete", s.RequirePermission("view_books")(s.ListGeraeteHandler(geraeteRepo)))
	mux.Handle("POST /api/geraete", s.RequirePermission("edit_books")(s.CreateGeraetHandler(geraeteRepo)))
	mux.Handle("PUT /api/geraete/{id}", s.RequirePermission("edit_books")(s.UpdateGeraetHandler(geraeteRepo)))
	mux.Handle("PUT /api/reservierungen/klassensatz/{id}/erledigen", s.RequirePermission("create_orders")(s.ErledigeKlassensatzReservierungHandler()))

	// Anliegen der Lehrkräfte (Wunsch/Meldung, Migration 075): Anlegen und die
	// eigene Statusliste laufen wie die Klassensatz-Reservierung über
	// create_reservations (das Portal-Recht des Kollegiums — bewusst NICHT
	// view_students, siehe Kommentar an der Klassensatz-Route). Die Arbeitsliste
	// und das Abhaken gehören der LMF: view_orders lesend, create_orders schreibend.
	mux.Handle("POST /api/anliegen", s.RequirePermission("create_reservations")(s.CreateAnliegenHandler()))
	mux.Handle("GET /api/anliegen/eigene", s.RequirePermission("create_reservations")(s.ListEigeneAnliegenHandler()))
	mux.Handle("GET /api/anliegen/offen", s.RequirePermission("view_orders")(s.ListOffeneAnliegenHandler()))
	mux.Handle("GET /api/anliegen/anzahl", s.RequirePermission("view_orders")(s.CountOffeneAnliegenHandler()))
	mux.Handle("PUT /api/anliegen/{id}/erledigen", s.RequirePermission("create_orders")(s.ErledigeAnliegenHandler()))
}
