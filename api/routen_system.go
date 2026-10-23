package api

import (
	"bibliothek/db"
	"bibliothek/repository"
	"net/http"
)

func (s *Server) registriereSystemRouten(mux *http.ServeMux, protokollAbfragen *repository.ProtokollAbfragen, kontoAbfragen *repository.KontoAbfragen, pool db.Pool) {
	// ── Benutzer und Rechte ──
	mux.Handle("GET /api/benutzer", s.verlangeRecht("manage_users")(s.benutzer().Konten(kontoAbfragen)))
	mux.Handle("POST /api/benutzer", s.verlangeRecht("manage_users")(s.benutzer().KontoAnlegen(kontoAbfragen)))
	mux.Handle("PUT /api/benutzer/{id}", s.verlangeRecht("manage_users")(s.benutzer().KontoAendern(kontoAbfragen)))
	mux.Handle("DELETE /api/benutzer/{id}", s.verlangeRecht("manage_users")(s.benutzer().KontoLoeschen(kontoAbfragen)))

	// ── Einstellungen ──
	einstellungsAbfragen := repository.NewEinstellungsAbfragen(pool)
	mux.Handle("GET /api/einstellungen", s.verlangeRecht("manage_settings")(s.system().Einstellungen(einstellungsAbfragen)))
	mux.Handle("PUT /api/einstellungen", s.verlangeRecht("manage_settings")(s.system().EinstellungenSchreiben(einstellungsAbfragen)))
	// Die Fristen der Sitzung (Theke leeren, Sperrbildschirm) braucht jede angemeldete Oberfläche,
	// auch Helfer und Kollegium, die /api/einstellungen nicht lesen dürfen. Nur die zwei
	// Zahlen, keine Schul- oder Betriebsdaten.
	mux.Handle("GET /api/einstellungen/sitzung", s.VerlangeAnmeldung()(s.system().SitzungsEinstellungen(einstellungsAbfragen)))

	// Ausweis-Design (zentral, für alle vernetzten Arbeitsplätze). Lesen breit (jeder,
	// der Ausweise druckt), Speichern nur administrativ.
	mux.Handle("GET /api/ausweis-layout", s.verlangeRecht("view_students")(s.druck().AusweisLayout()))
	mux.Handle("PUT /api/ausweis-layout", s.verlangeRecht("manage_settings")(s.druck().AusweisLayoutSchreiben()))

	mailEinstellungsAbfragen := repository.NewMailEinstellungsAbfragen(pool)
	mux.Handle("GET /api/admin/settings/mail", s.verlangeRecht("manage_settings")(s.system().MailEinstellungen(mailEinstellungsAbfragen)))
	mux.Handle("PUT /api/admin/settings/mail", s.verlangeRecht("manage_settings")(s.system().MailEinstellungenSchreiben(mailEinstellungsAbfragen)))
	mux.Handle("POST /api/admin/settings/mail/test", s.verlangeRecht("manage_settings")(s.system().TestmailSenden()))

	// Rechte
	mux.Handle("GET /api/admin/permissions", s.verlangeRecht("manage_users")(s.benutzer().RechteMatrix()))
	mux.Handle("GET /api/admin/system/backup-status", s.verlangeRecht("manage_settings")(s.system().BackupStand()))
	// Die Verallgemeinerung des Wächters über das Backup: Was ist eingerichtet, aber nicht in
	// Betrieb? Siehe internal/bereitschaft/bereitschaft.go.
	//
	// Bewusst eine Zeile: pii_matrix_test.go liest die Registrierungen zeilenweise und sähe bei
	// einem Umbruch das Recht nicht. Lieber eine lange Zeile als eine Prüfung, die man entschärfen
	// muss.
	mux.Handle("GET /api/admin/system/betriebsbereitschaft", s.verlangeRecht("manage_settings")(s.system().Betriebsbereitschaft(einstellungsAbfragen, mailEinstellungsAbfragen, repository.NewBetriebszustandAbfragen(pool))))
	mux.Handle("PUT /api/admin/permissions", s.verlangeRecht("manage_users")(s.benutzer().RollenRechtSetzen()))

	// Protokoll
	mux.Handle("GET /api/audit", s.verlangeRecht("audit_logs")(s.system().Protokoll()))
	// Der eine zweckgebundene Weg, audit_log.details zu lesen (der Fall an der Theke): eigenes Recht,
	// jeder Abruf schreibt sich selbst ins Protokoll (api/system/theken_auskunft.go).
	mux.Handle("GET /api/audit/tresen-auskunft", s.verlangeRecht("audit_details")(s.system().ThekenAuskunft(protokollAbfragen)))

	// Vorlagen der Mails
	mux.Handle("GET /api/mail-templates", s.verlangeRecht("manage_settings")(s.system().MailVorlagen()))
	mux.Handle("PUT /api/mail-templates/{id}", s.verlangeRecht("manage_settings")(s.system().MailVorlageAendern()))

	// Druck und Berichte
	mux.Handle("GET /api/print/rechnung/{schueler_id}", s.verlangeRecht("view_students")(s.schaden().RechnungPDF()))
	// Der Druck zählt die Mahnung: dasselbe Recht wie der Mahnversand (create_orders), nicht
	// das Leserecht der Seite.
	mux.Handle("POST /api/admin/mahnungen/bulk-print", s.verlangeRecht("create_orders")(s.mahnwesen().MahnbriefeDrucken()))
	mux.Handle("GET /api/print/kontoauszug/{schueler_id}", s.verlangeRecht("view_students")(s.leser().KontoauszugPDF()))

	// Übersicht und Statistik
	mux.Handle("GET /api/dashboard/summary", s.verlangeRecht("view_students")(s.system().MahnUebersicht()))
	// view_stats, nicht view_students: Der Schalter, den ein Admin in der Rechte-Oberfläche
	// umlegt, heißt „Statistiken anzeigen", und das Menü blendet den Punkt genau danach ein
	// (frontend/src/lib/menue.js). Verlangte die Route view_students, gewänne still der Server,
	// sobald ein Admin die beiden trennt: Die Statistik antwortete trotz entzogenem Recht weiter,
	// oder der Menüpunkt lieferte jedes Mal 403. Die Antwort trägt keine Personendaten
	// (PII-Stufe 0), view_students wäre also auch fachlich die falsche Schranke.
	mux.Handle("GET /api/statistiken", s.verlangeRecht("view_stats")(s.system().Statistik()))

	// Nachschlagelisten
	mux.Handle("GET /api/systematics", s.verlangeRecht("view_books")(s.katalog().Systematik()))
	// Das Vokabular, aus dem das Buchformular die Signatur vorschlägt; die Türen darunter füllen
	// es.
	mux.Handle("POST /api/systematics", s.verlangeRecht("edit_books")(s.katalog().SachgruppeAnlegen()))
	mux.Handle("PUT /api/systematics/{id}", s.verlangeRecht("edit_books")(s.katalog().SachgruppeAendern()))
	mux.Handle("DELETE /api/systematics/{id}", s.verlangeRecht("edit_books")(s.katalog().SachgruppeLoeschen()))
	mux.Handle("GET /api/faecher", s.verlangeRecht("view_books")(s.katalog().Faecher()))

	// Protokoll der Verwaltung
	mux.Handle("GET /api/admin/auditlog", s.verlangeRecht("manage_users")(s.system().VerwaltungsProtokoll()))

	// Barcodes & Etiketten
	// edit_books statt view_books: Der Endpunkt zeigt keine Nummer an, er vergibt eine aus
	// barcode_seq, und die gezogene Nummer ist danach verbraucht. Wer sie nicht
	// speichern darf (das Barcode-PUT verlangt edit_books), soll sie auch nicht ziehen.
	mux.Handle("GET /api/barcode/next", s.verlangeRecht("edit_books")(s.katalog().ExemplarBarcodeZiehen()))
	mux.Handle("GET /api/barcode", s.verlangeRecht("view_books")(s.druck().BarcodeBild()))

	// Signaturen, abgeleitet aus buecher_titel.signatur, nicht aus einer eigenen Tabelle.
	mux.Handle("GET /api/signaturen", s.verlangeRecht("view_books")(s.katalog().Signaturen()))
	mux.Handle("GET /api/signaturen/buecher", s.verlangeRecht("view_books")(s.katalog().TitelUnterSignatur()))
	// Schlagworte: die Vorschlagsliste der Eingabefelder, aus dem Bestand.
	mux.Handle("GET /api/schlagworte", s.verlangeRecht("view_books")(s.katalog().SchlagwortVorschlaege()))
	// Der Vorschlag aus der DNB zu einer ISBN: für das Buchformular
	// und das Nachbestellen. Recht wie die Liste darüber und wie GET /api/lookup/{isbn}, das
	// dieselbe DNB fragt; der Vorschlag liest nur, eingetragen wird über die Türen des Titels.
	mux.Handle("GET /api/schlagworte/dnb-vorschlag", s.verlangeRecht("view_books")(s.katalog().DNBSchlagwortVorschlag()))
	// Pflege der Schlagworte: wie die Systematik mit edit_books; die Pflege ändert den Katalog,
	// nicht den Bestand an Exemplaren.
	mux.Handle("GET /api/schlagworte/pflege", s.verlangeRecht("edit_books")(s.katalog().SchlagworteZurPflege()))
	mux.Handle("PUT /api/schlagworte/{id}/wort", s.verlangeRecht("edit_books")(s.katalog().SchlagwortUmbenennen()))
	mux.Handle("PUT /api/schlagworte/{id}/filter", s.verlangeRecht("edit_books")(s.katalog().SchlagwortFilterSetzen()))
	mux.Handle("POST /api/schlagworte/{id}/zusammenfuehren", s.verlangeRecht("edit_books")(s.katalog().SchlagworteZusammenfuehren()))
	mux.Handle("POST /api/schlagworte/verweise", s.verlangeRecht("edit_books")(s.katalog().SchlagwortVerweisSetzen()))
	mux.Handle("POST /api/schlagworte/loeschen", s.verlangeRecht("edit_books")(s.katalog().SchlagworteLoeschen()))

	// Der Live-Strom: nur Sitzung, kein Fachrecht. Jede angemeldete Oberfläche öffnet ihn
	// (authStore und die Omnibox der Theke), und an ihm hängt der Herzschlag für die Anzeige
	// „offline". Hinge er an einem Fachrecht, liefe jede Rolle ohne dieses Recht in eine Schleife
	// aus 403 und neuem Verbindungsaufbau. Der Strom trägt nur Ereignisse der Abläufe (Typ,
	// Kennungen, Zeitstempel), keinen Namen und kein Buch (thekenEreignis in
	// api/theke/eingabe_anfrage_antwort.go, VerlangeAnmeldung in api/tuer/sitzung.go).
	liveLeitung := s.Broker.Tuer()
	mux.Handle("GET /events", s.VerlangeAnmeldung()(liveLeitung))
}
