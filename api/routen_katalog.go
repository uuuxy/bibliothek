package api

import (
	"bibliothek/repository"
	"net/http"
)

func (s *Server) registriereKatalogRouten(mux *http.ServeMux, katalogAbfragen *repository.KatalogAbfragen, protokollAbfragen *repository.ProtokollAbfragen) {
	// ── Titel und Exemplare ──
	mux.Handle("DELETE /api/buecher/titel/{id}", s.verlangeRecht("delete_books")(s.katalog().TitelLoeschen(katalogAbfragen)))
	// Die Signatur aus dem Bestellkorb korrigieren: create_orders, nicht edit_books, dasselbe
	// Recht wie das Anlegen des Titels über /aus-isbn (api/katalog/isbn_zu_titel.go).
	mux.Handle("PUT /api/buecher/titel/{id}/signatur", s.verlangeRecht("create_orders")(s.katalog().TitelSignaturSetzen()))
	// Dasselbe Recht für das Kennzeichen „Lernmittel" eines eben aus der DNB angelegten Titels;
	// es entscheidet über den Topf der Bestellung.
	mux.Handle("PUT /api/buecher/titel/{id}/lernmittel", s.verlangeRecht("create_orders")(s.katalog().TitelLernmittelSetzen()))
	// Schlagworte: Der Bestellkorb liest die vorhandenen, bevor er die Menge ersetzt. Schreiben
	// mit create_orders wie Signatur und Lernmittel daneben; das Buchformular schreibt über
	// PUT /api/books/{id} (edit_books), beide über repository.SetzeSchlagworte.
	mux.Handle("GET /api/buecher/titel/{id}/schlagworte", s.verlangeRecht("view_books")(s.katalog().TitelSchlagworte()))
	mux.Handle("PUT /api/buecher/titel/{id}/schlagworte", s.verlangeRecht("create_orders")(s.katalog().TitelSchlagworteSetzen()))
	// Auflagen eines Schulbuchs: Zusammenfassen und Lösen aus der Titelmaske mit edit_books,
	// „Neue Auflage bestellen" im Bestellbedarf mit create_orders. Alle rufen
	// repository/auflagen.go; dieselbe Tür wie für die Titelmaske, andere Route, anderes Recht,
	// wie bei den Schlagworten.
	mux.Handle("GET /api/buecher/titel/{id}/auflagen", s.verlangeRecht("view_books")(s.katalog().TitelAuflagen()))
	mux.Handle("POST /api/buecher/titel/{id}/auflagen", s.verlangeRecht("edit_books")(s.katalog().AuflagenZusammenfassen()))
	mux.Handle("DELETE /api/buecher/titel/{id}/auflagen", s.verlangeRecht("edit_books")(s.katalog().AuflageLoesen()))
	mux.Handle("POST /api/buecher/titel/{id}/neue-auflage", s.verlangeRecht("create_orders")(s.katalog().AuflagenZusammenfassen()))

	// Exemplare
	// Titel-Tür für Bildschirme außerhalb der Theke (Etiketten-Titelsuche im Druck-Center):
	// dieselbe Suche wie GET /api/search, aber nur Titel und hinter view_books statt
	// perform_actions.
	mux.Handle("GET /api/buecher/titel/suche", s.verlangeRecht("view_books")(s.druck().TitelSuche(katalogAbfragen)))
	// Die Exemplarliste trägt den heutigen Ersatzwert je Exemplar. Das Recht bleibt view_books:
	// Was ein Ersatz kostet, ist eine Angabe über das Buch, kein Datum eines Schülers. Gerechnet
	// wird mit derselben Funktion wie im Dialog zum Melden eines Schadens, gelesen über die
	// BescheidAbfragen, denen die Größen der Staffel gehören.
	mux.Handle("GET /api/buecher/titel/{id}/exemplare", s.verlangeRecht("view_books")(s.katalog().TitelExemplare(repository.NewBescheidAbfragen(s.DB.Pool))))
	// Ausleiher und Historie hinter view_students, nicht view_books: Beide verknüpfen Namen von
	// Schülern mit Titeln, die Historie über Jahre, wie die Vormerkungen. Ohne das Recht zeigt die
	// Buchakte die Liste leer und nennt sie als fehlend (useBuchakte.svelte.js); Exemplare und
	// Verfügbarkeit bleiben view_books.
	mux.Handle("GET /api/buecher/titel/{id}/ausleiher", s.verlangeRecht("view_students")(s.katalog().TitelAusleiher()))
	mux.Handle("GET /api/buecher/titel/{id}/historie", s.verlangeRecht("view_students")(s.katalog().TitelHistorie()))
	mux.Handle("GET /api/buecher/titel/{id}/etiketten", s.verlangeRecht("view_books")(s.druck().TitelEtiketten()))
	mux.Handle("POST /api/print/labels", s.verlangeRecht("view_books")(s.druck().Etikettenbogen()))

	// Nachdruck: Welche Exemplare haben noch kein Etikett, und Gegenbuchung nach dem Druck.
	// edit_books statt view_books, weil die Gegenbuchung Bestandsdaten ändert.
	mux.Handle("GET /api/exemplare/etiketten-offen", s.verlangeRecht("edit_books")(s.druck().EtikettenOffen()))
	mux.Handle("GET /api/exemplare/etiketten-offen/anzahl", s.verlangeRecht("edit_books")(s.druck().EtikettenOffenAnzahl()))
	mux.Handle("POST /api/exemplare/etiketten-gedruckt", s.verlangeRecht("edit_books")(s.druck().EtikettenGedrucktVermerken()))
	// Der Weg zurück nach Papierstau oder zu weitem Altbestands-Stichtag.
	mux.Handle("POST /api/exemplare/etiketten-zuruecksetzen", s.verlangeRecht("edit_books")(s.druck().EtikettenZuruecksetzen()))
	mux.Handle("POST /api/exemplare/etiketten-altbestand", s.verlangeRecht("edit_books")(s.druck().EtikettenAltbestandVermerken()))
	// Eigentum markierter Exemplare: edit_books wie Barcode und Status, Grund
	// Pflicht, jede Änderung im Protokoll.
	mux.Handle("PUT /api/exemplare/eigentum", s.verlangeRecht("edit_books")(s.katalog().ExemplarEigentumSetzen()))
	// Standort markierter Exemplare: edit_books wie das Eigentum daneben, jede Änderung
	// im Protokoll. Die Vorschläge liest nur der Dialog, der ändert.
	mux.Handle("PUT /api/exemplare/standort", s.verlangeRecht("edit_books")(s.katalog().ExemplarStandortSetzen()))
	mux.Handle("GET /api/exemplare/standorte", s.verlangeRecht("edit_books")(s.katalog().ExemplarStandorte()))

	// Das Abgangsbuch: welche Exemplare in einem Zeitraum aus dem Bestand gingen. view_books wie
	// der übrige Blick auf den Bestand; es ist ein Nachweis, der nichts ändert.
	mux.Handle("GET /api/bestand/abgangsbuch", s.verlangeRecht("view_books")(s.katalog().Abgangsbuch()))
	mux.Handle("GET /api/bestand/abgangsbuch/pdf", s.verlangeRecht("view_books")(s.katalog().AbgangsbuchPDF()))
	mux.Handle("GET /api/bestand/zugangsbuch", s.verlangeRecht("view_books")(s.katalog().Zugangsbuch()))
	mux.Handle("GET /api/bestand/zugangsbuch/pdf", s.verlangeRecht("view_books")(s.katalog().ZugangsbuchPDF()))

	mux.Handle("DELETE /api/buecher/exemplare/{id}", s.verlangeRecht("delete_books")(s.katalog().ExemplarAussondern(katalogAbfragen)))

	// Einzelne Felder eines Exemplars ändern
	mux.Handle("PUT /api/buecher/exemplare/{id}/barcode", s.verlangeRecht("edit_books")(s.katalog().ExemplarBarcodeSetzen(katalogAbfragen)))
	mux.Handle("PUT /api/buecher/exemplare/{id}/status", s.verlangeRecht("edit_books")(s.katalog().ExemplarStatusAendern(katalogAbfragen, repository.NewBescheidAbfragen(s.DB.Pool))))

	// ── Ausleihen ──
	einstellungsAbfragen := repository.NewEinstellungsAbfragen(s.DB.Pool)
	mux.Handle("POST /api/ausleihen/{ausleihe_id}/verlaengern", s.verlangeRecht("edit_books")(s.theke().AusleiheVerlaengern(einstellungsAbfragen)))
	mux.Handle("POST /api/ausleihen/global-extend-lmf", s.verlangeRecht("edit_books")(s.lernmittel().LernmittelDerKlasseVerlaengern()))

	// ── LMF-Plan (Termine für Rückgabe und Ausgabe je Klasse) ──
	// Lesen: jede Sitzung (das Kollegium liest den Plan im Portal; PII-Stufe 0, Daten und
	// Klassennamen). Schreiben: edit_books wie die übrige Pflege der Lernmittel.
	mux.Handle("GET /api/lmf-termine", s.VerlangeAnmeldung()(s.lernmittel().LMFTermine()))
	mux.Handle("GET /api/lmf-termine/pdf", s.VerlangeAnmeldung()(s.lernmittel().LMFPlanPDF(false)))
	// Der Entwurf als PDF zur Abnahme durch die Schulleitung, bevor er veröffentlicht ist. Nur
	// der Planer; das Kollegium sieht Entwürfe nirgends.
	mux.Handle("GET /api/lmf-termine/entwurf/pdf", s.verlangeRecht("edit_books")(s.lernmittel().LMFPlanPDF(true)))
	// Geschrieben wird der Plan als Reihenfolge (api/lernmittel/lmf_plan.go): ein Aufruf je Art,
	// die Vorschau über denselben Aufruf. Routen für einzelne Termine gibt es nicht; eine zweite
	// Tür je Zeile gäbe zwei Wahrheiten.
	mux.Handle("GET /api/lmf-plan/{art}", s.verlangeRecht("edit_books")(s.lernmittel().LMFPlan()))
	mux.Handle("PUT /api/lmf-plan/{art}", s.verlangeRecht("edit_books")(s.lernmittel().LMFPlanSchreiben()))
	mux.Handle("DELETE /api/lmf-plan/{art}", s.verlangeRecht("edit_books")(s.lernmittel().LMFPlanVerwerfen()))
	// Veröffentlichen: Speichern legt einen Entwurf an, erst dieser Aufruf macht ihn für
	// Portal und PDF sichtbar und setzt bei einem Rückgabe-Plan die Fristen.
	mux.Handle("POST /api/lmf-plan/{art}/veroeffentlichen", s.verlangeRecht("edit_books")(s.lernmittel().LMFPlanVeroeffentlichen()))
	mux.Handle("PATCH /api/admin/ausleihen/{id}/faelligkeit", s.verlangeRecht("edit_books")(s.theke().AusleihFristSetzen(protokollAbfragen)))

	// Der Titel zu einer ISBN, notfalls aus den Katalogdiensten
	mux.Handle("POST /api/buecher/aus-isbn", s.verlangeRecht("create_orders")(s.katalog().ISBNZuTitel()))

	// Vormerkungen
	vormerkungsAbfragen := repository.NewVormerkungsAbfragen(s.DB.Pool)
	// Eigenes enges Recht manage_vormerkungen, nicht view_books
	// (bewertung/sicherheitsbefund-vormerkungen.md, Weg 2): Die Liste verknüpft Namen von
	// Schülern mit ihrem Leseinteresse und erlaubt Anlegen und Löschen für jeden Leser. Sie zeigt
	// aber nur die Felder der Theke (Name, Klasse); dafür das volle view_students zu verlangen
	// (Leserakte, Adressen, Mahnwesen), wäre zu grob, Helfer verlören die Warteliste ganz. Admin,
	// Leitung und Mitarbeiter haben das Recht ab Werk, Helfer über die Rechte-Matrix (Vorgabe aus,
	// db/rechte_vorgabe.go). Der Einzelfall am Rückgabe-Scan („reserviert für X") bleibt: Den Namen braucht
	// die Theke, um das Buch für die richtige Person beiseitezulegen.
	mux.Handle("GET /api/vormerkungen", s.verlangeRecht("manage_vormerkungen")(s.theke().Vormerkungen(vormerkungsAbfragen)))
	mux.Handle("POST /api/vormerkungen", s.verlangeRecht("manage_vormerkungen")(s.theke().VormerkungAnlegen(vormerkungsAbfragen)))
	mux.Handle("DELETE /api/vormerkungen/{id}", s.verlangeRecht("manage_vormerkungen")(s.theke().VormerkungLoeschen(vormerkungsAbfragen)))

	// Reservierungen von Klassensätzen
	//
	// create_reservations statt view_students: Die Tür fasst keine Daten von Schülern an, sie nimmt
	// Titel, Klasse als Freitext und Anzahl entgegen. Hinge sie an view_students, bräuchte, wer im
	// Portal des Kollegiums reservieren darf, dasselbe Recht wie Schülerdatei und Druck-Center; die
	// eine Funktion des Portals zöge die ganze Schülerdatei mit auf.
	mux.Handle("POST /api/reservierungen/klassensatz", s.verlangeRecht("create_reservations")(s.portal().KlassensatzReservierungAnlegen()))
	mux.Handle("GET /api/reservierungen/klassensatz", s.verlangeRecht("view_orders")(s.portal().KlassensatzReservierungen()))
	mux.Handle("GET /api/reservierungen/klassensatz/anzahl", s.verlangeRecht("view_orders")(s.portal().KlassensatzReservierungenAnzahl()))
	// Warteschlange für das Kollegium: dasselbe Recht wie das Anlegen; wer
	// reservieren darf, darf sehen, wer vor ihm dran ist.
	mux.Handle("GET /api/reservierungen/klassensatz/offen", s.verlangeRecht("create_reservations")(s.portal().OffeneKlassensatzReservierungen()))
	mux.Handle("GET /api/reservierungen/klassensatz/eigene", s.verlangeRecht("create_reservations")(s.portal().EigeneKlassensatzReservierungen()))
	// Der Katalog, in dem das Kollegium sucht und reserviert, und die Schlagworte, die
	// darunter als Filter stehen: wie der öffentliche Katalog, dazu Lernmittel und bestellte
	// Titel (api/portal/katalog_kollegium.go). Dasselbe Recht wie das Reservieren.
	mux.Handle("GET /api/reservierungen/klassensatz/katalog", s.verlangeRecht("create_reservations")(s.portal().KollegiumKatalogSuche()))
	mux.Handle("GET /api/reservierungen/klassensatz/katalog/filter", s.verlangeRecht("create_reservations")(s.portal().KollegiumKatalogFilter()))

	// Geräte (ein Bereich im Medienkatalog): Geräte sind Bestand, Rechte wie für Bücher.
	geraeteAbfragen := repository.NewGeraeteAbfragen(s.DB.Pool)
	mux.Handle("GET /api/geraete", s.verlangeRecht("view_books")(s.katalog().Geraete(geraeteAbfragen)))
	mux.Handle("POST /api/geraete", s.verlangeRecht("edit_books")(s.katalog().GeraetAnlegen(geraeteAbfragen)))
	mux.Handle("PUT /api/geraete/{id}", s.verlangeRecht("edit_books")(s.katalog().GeraetAendern(geraeteAbfragen)))
	mux.Handle("PUT /api/reservierungen/klassensatz/{id}/erledigen", s.verlangeRecht("create_orders")(s.portal().KlassensatzReservierungErledigen()))

	// Anliegen der Lehrkräfte (Wunsch oder Meldung): Anlegen und die eigene Statusliste laufen
	// wie die Reservierung eines Klassensatzes über create_reservations, das Recht des Kollegiums
	// im Portal, nicht über view_students (siehe die Route der Klassensätze). Die Arbeitsliste und
	// das Abhaken gehören der Lernmittelverwaltung: view_orders lesend, create_orders schreibend.
	mux.Handle("POST /api/anliegen", s.verlangeRecht("create_reservations")(s.portal().AnliegenAnlegen()))
	mux.Handle("GET /api/anliegen/eigene", s.verlangeRecht("create_reservations")(s.portal().EigeneAnliegen()))
	mux.Handle("GET /api/anliegen/offen", s.verlangeRecht("view_orders")(s.portal().OffeneAnliegen()))
	mux.Handle("GET /api/anliegen/anzahl", s.verlangeRecht("view_orders")(s.portal().OffeneAnliegenAnzahl()))
	mux.Handle("PUT /api/anliegen/{id}/erledigen", s.verlangeRecht("create_orders")(s.portal().AnliegenErledigen()))
}
