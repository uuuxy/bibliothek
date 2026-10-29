package api

// Jede Liste in der Antwort einer GET-Route nennt, was sie begrenzt.
//
// Die Bugklasse heißt „Unbegrenzte Listen" (docs/sweeps.md): Eine Route, die eine ganze Tabelle
// ausliefert, fällt auf kleinen Datenmengen nicht auf und kippt im Betrieb. Ohne Grenze liefert
// GET /api/audit nach einem Lasttest 72 MB in einer Antwort; der Server braucht dafür 0,6
// Sekunden, der Browser scheitert am Aufbau der Seite. Ein Test je Route fängt das nur dort, wo
// jemand daran gedacht hat.
//
// Dieser Test hängt am Merkmal. Er ruft jede GET-Route über den echten Router auf, mit dem
// Apparat der Prüfung der PII-Antworten (pii_antwort_pg_test.go) und deren Vollständigkeit über
// die Matrix, und sucht in jeder JSON-Antwort die Listen. Jede gefundene Liste braucht einen
// Eintrag in listenGrenze mit der Antwort auf die Frage: Was begrenzt sie, eine Obergrenze in der
// Abfrage, ein Elternteil, eine feste Menge?
//
// Kein Urteil: Ob die genannte Grenze greift, prüft je Liste ihr eigener Test, gegen mehr Zeilen
// als die Grenze. Sieht nicht: Listen in PDF-, CSV- und Bildantworten; Routen, die die Prüfung
// der PII-Antworten mit Begründung auslässt oder die in der gesäten Welt mit einem Fehlerstatus
// antworten; eine Liste, die leer als null oder gar nicht in der Antwort steht; Listen in den
// Zeilen einer Liste und tiefer als zwei Ebenen unter der Wurzel; POST-Routen.

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sort"
	"strings"
	"testing"
	"time"

	"bibliothek/auth"
	"bibliothek/db"
	"bibliothek/internal/pgtest"
	"bibliothek/internal/testdaten"
	"bibliothek/internal/tuertest"
	"bibliothek/sse"

	"github.com/google/go-cmp/cmp"
)

// listenGrenze: je Liste (Route und Ort in der Antwort, „$" ist die Wurzel) die Antwort.
var listenGrenze = map[string]string{
	"GET /api/audit $":            "LIMIT 1000, die jüngsten zuerst (auditLogMaxZeilen, api/system/protokoll.go)",
	"GET /api/admin/auditlog $":   "LIMIT 1000, die jüngsten zuerst (api/system/verwaltungs_protokoll.go)",
	"GET /api/schueler/deleted $": "LIMIT 1000, die jüngsten Löschungen zuerst (papierkorbGrenze, api/leser/papierkorb.go)",
	"GET /api/books $": "Kappung bei 50000 Titeln mit Warnung im Log (titelListeGrenze, " +
		"repository/titel_liste.go)",
	"GET /api/signaturen $": "keine Obergrenze in der Abfrage; eine Zeile je Regaladresse (GROUP BY über die " +
		"Titel), die Liste wächst mit den Regalen, nicht mit den Büchern (repository.LadeSignaturGruppen)",
	"GET /api/auth/me $.permissions":                                      "keine Obergrenze in der Abfrage; die erlaubten Rechte der Rolle aus role_permissions, höchstens der Katalog der Rechte (db.SchreibeRechteVorgabe), beim Admin nur \"*\" (auth.ladeRechteDerRolle)",
	"GET /api/admin/permissions $":                                        "keine Obergrenze in der Abfrage; die ganze Matrix aus role_permissions, Rollen mal Rechte des Katalogs (repository.LadeRollenRechte)",
	"GET /api/jahrgaenge $":                                               "keine Obergrenze in der Abfrage; je besetztem Jahrgang ein Eintrag, aus den Klassen mit Schülern (jahrgaengeMitSchuelern, api/leser/schueler_jahrgang_filter.go)",
	"GET /api/dashboard/summary $.overdue_buckets":                        "vier feste Zeitspannen: 1–14, 15–30, 31–60 und über 60 Tage (api/system/mahn_uebersicht.go)",
	"GET /api/buecher/titel/{id}/zusammenfuehren $.uebernommen":           "höchstens vierzehn feste Felder eines Titels, die die Quelle auffüllt (repository.auffuellbar, Lernmittel, Jahrgang, Mehrjahresband und Cover)",
	"GET /api/buecher/titel/{id}/zusammenfuehren $.abweichend":            "höchstens sechzehn feste Felder eines Titels (repository.fuegeTitelZusammen)",
	"GET /api/statistiken $.monats_trend":                                 "zwölf Monate, die letzten elf und der laufende (generate_series in repository.LadeMonatsTrend)",
	"GET /api/statistiken $.nicht_gelesen":                                "höchstens vier, je gescheiterter Abfrage ein Abschnitt (api/system/statistik.go)",
	"GET /api/statistiken $.popular_titles":                               "LIMIT aus ?limit=, Vorgabe 5, höchstens 200 (leseListenGrenze, api/system/statistik.go; repository.LadeRenner)",
	"GET /api/statistiken $.shelf_warmers":                                "LIMIT aus ?limit=, Vorgabe 5, höchstens 200 (leseListenGrenze, api/system/statistik.go; repository.LadeLadenhueter)",
	"GET /api/monitor/slides $.beliebt":                                   "LIMIT 5 (monitorBeliebtAnzahl, repository/oeffentlich.go)",
	"GET /api/monitor/slides $.neu_eingetroffen":                          "LIMIT 10 (monitorNeuAnzahl, repository/oeffentlich.go)",
	"GET /api/search $.books":                                             "LIMIT 10 (grenze in api/theke/suche.go, KatalogAbfragen.FindeTitelUnscharf); die Zahl vor der Grenze steht in books_total",
	"GET /api/search $.students":                                          "LIMIT 10 (grenze in api/theke/suche.go, LeserAbfragen.FindeUnscharf); die Zahl vor der Grenze steht in students_total",
	"GET /api/buecher/titel/suche $.books":                                "LIMIT 10 (api/druck/titel_suche.go, KatalogAbfragen.FindeTitelUnscharf)",
	"GET /api/public/opac/suche $":                                        "LIMIT 50 (opacGrenze, api/portal/opac.go; repository.FindeImKatalog)",
	"GET /api/mail-templates $":                                           "keine Obergrenze in der Abfrage; je Art der Vorlage eine Zeile, typ ist eindeutig (UNIQUE in mail_vorlagen; repository.LadeMailVorlagen)",
	"GET /api/admin/system/betriebsbereitschaft $.befunde":                "einundzwanzig feste Bereiche, je einer ein Befund (bereitschaft.Pruefe)",
	"GET /api/admin/system/fehler-im-log $.meldungen":                     "höchstens hundert, je Meldung eine Zeile; bei mehr fällt die Meldung, deren letzte Zeile am längsten zurückliegt (hoechstzahlGruppen, pkg/logger/fehlerstand.go)",
	"GET /api/faecher $":                                                  "keine Obergrenze in der Abfrage; die verschiedenen Fächer der Titel (DISTINCT), wächst mit den Fächern, nicht mit den Büchern (repository.LadeFaecher)",
	"GET /api/klassen $":                                                  "keine Obergrenze in der Abfrage; die verschiedenen Klassen der aktiven Schüler (DISTINCT), wächst mit der Schule, nicht mit den Jahren (LeserAbfragen.LadeKlassen)",
	"GET /api/buecher/titel/{id}/auflagen $.auflagen":                     "keine Obergrenze in der Abfrage; die Auflagen desselben Werks (werk_id) wie der eine Titel (repository.LadeAuflagenDesTitels)",
	"GET /api/buecher/titel/{id}/ausleiher $":                             "keine Obergrenze in der Abfrage; die offenen Ausleihen des einen Titels, höchstens so viele wie seine Exemplare (repository.LadeAusleiherDesTitels)",
	"GET /api/buecher/titel/{id}/exemplare $":                             "keine Obergrenze in der Abfrage; die Exemplare des einen Titels, bei einem Klassensatz einige Hundert (repository.LadeExemplareDesTitels)",
	"GET /api/buecher/titel/{id}/historie $":                              "LIMIT 200, die jüngsten Ausleihen des einen Titels zuerst (repository.LadeAusleihhistorieDesTitels)",
	"GET /api/buecher/titel/{id}/schlagworte $.schlagworte":               "keine Obergrenze in der Abfrage; die Schlagworte des einen Titels, beim Schreiben auf SchlagworteJeTitelMax begrenzt (repository.LadeSchlagworteDesTitels)",
	"GET /api/books/{id} $.schlagworte":                                   "keine Obergrenze in der Abfrage; die Schlagworte des einen Titels, beim Schreiben auf SchlagworteJeTitelMax begrenzt (api/katalog/titel_lesen.go)",
	"GET /api/schueler/{id}/zusammenfuehren-kandidaten $":                 "LIMIT 20 (api/leser/leser_zusammenfuehren.go; repository.FindeZusammenfuehrenKandidaten)",
	"GET /api/schueler/{id}/bescheide $":                                  "keine Obergrenze in der Abfrage; die Bescheide des einen Schülers (BescheidAbfragen.LadeZuSchueler)",
	"GET /api/schueler/{id}/schadensfaelle $":                             "keine Obergrenze in der Abfrage; die Schadensfälle des einen Lesers (SchadensAbfragen.LadeZuLeser)",
	"GET /api/schueler/{id} $.entliehene_buecher":                         "keine Obergrenze in der Abfrage; die offenen Ausleihen des einen Lesers, so viele, wie er gerade hat (LeserAbfragen.LadeLaufendeAusleihen)",
	"GET /api/schueler/{id}/bescheid-vorschlag $.ausleihen":               "keine Obergrenze in der Abfrage; die überfälligen Ausleihen des einen Schülers ohne Forderung (BescheidAbfragen.LadeUeberfaelligeAusleihen)",
	"GET /api/schueler/{id}/bescheid-vorschlag $.fehlende_angaben":        "höchstens fünf feste Pflichtangaben des Bescheids (repository.BescheidAngaben.FehlendeAngaben)",
	"GET /api/schueler/{id}/bescheid-vorschlag $.positionen":              "keine Obergrenze in der Abfrage; die offenen Forderungen des einen Schülers ohne Bescheid (BescheidAbfragen.LadeOffeneForderungen)",
	"GET /api/schueler/{id}/dsgvo-auskunft $.ausleihhistorie":             "keine Obergrenze, mit Absicht: Die Auskunft nach Art. 15 DSGVO ist vollständig; die Ausleihen des einen Lesers (repository.LadeDSGVOAusleihen)",
	"GET /api/schueler/{id}/dsgvo-auskunft $.fruehere_zugangskonten":      "keine Obergrenze, mit Absicht: Die Auskunft nach Art. 15 DSGVO ist vollständig; die gelöschten Zugangskonten des einen Lesers (repository.LadeDSGVOFruehereZugangskonten)",
	"GET /api/schueler/{id}/dsgvo-auskunft $.nachbuch_meldungen":          "keine Obergrenze, mit Absicht: Die Auskunft nach Art. 15 DSGVO ist vollständig; die Meldungen, in denen der eine Leser Ausleiher oder Vorbesitzer ist (repository.LadeDSGVONachbuchMeldungen)",
	"GET /api/schueler/{id}/dsgvo-auskunft $.protokolleintraege":          "keine Obergrenze, mit Absicht: Die Auskunft nach Art. 15 DSGVO ist vollständig; die Einträge des Protokolls zu dem einen Leser (repository.LadeDSGVOProtokollEintraege)",
	"GET /api/schueler/{id}/dsgvo-auskunft $.schadensersatz_bescheide":    "keine Obergrenze, mit Absicht: Die Auskunft nach Art. 15 DSGVO ist vollständig; die Bescheide des einen Lesers (repository.LadeDSGVOBescheide)",
	"GET /api/schueler/{id}/dsgvo-auskunft $.schadensfaelle":              "keine Obergrenze, mit Absicht: Die Auskunft nach Art. 15 DSGVO ist vollständig; die Schadensfälle des einen Lesers (repository.LadeDSGVOSchadensfaelle)",
	"GET /api/schueler/{id}/dsgvo-auskunft $.verarbeitungsangaben.zwecke": "feste Liste der Zwecke je Leserart (auskunft.Pflichtangaben, internal/auskunft/pflichtangaben.go)",
	"GET /api/schueler/{id}/dsgvo-auskunft $.verwaltungsprotokolle":       "keine Obergrenze, mit Absicht: Die Auskunft nach Art. 15 DSGVO ist vollständig; die Einträge des Verwaltungsprotokolls, die den einen Leser nennen (repository.LadeDSGVOVerwaltungsEintraege)",
	"GET /api/schueler/{id}/dsgvo-auskunft $.vormerkungen":                "keine Obergrenze, mit Absicht: Die Auskunft nach Art. 15 DSGVO ist vollständig; die Vormerkungen des einen Lesers (repository.LadeDSGVOVormerkungen)",
	"GET /api/abgaenger $.abgaenger":                                      "keine Obergrenze in der Abfrage; je Schüler der Abschlussklassen mit offenen Büchern eine Zeile, nur in der Saison vom 01.05. bis 31.07., sonst leer (repository.LadeAbgaengerMitOffenenAusleihen)",
	"GET /api/action/buchbarcodes $.barcodes":                             "keine Obergrenze, mit Absicht: alle Barcodes aller Exemplare, auch der ausgesonderten, damit die Theke ohne Netz Buch und Ausweis unterscheidet; nur Nummern, mit ETag, unverändert 304 (api/theke/buchbarcodes.go)",
	"GET /api/action/nachbuch-meldungen $":                                "LIMIT 500, die jüngsten zuerst; ohne ?alle=1 nur die offenen (repository.LadeNachbuchMeldungen)",
	"GET /api/anliegen/eigene $":                                          "LIMIT 50, das neueste zuerst (AnliegenAbfragen.LadeEigene)",
	"GET /api/anliegen/offen $":                                           "keine Obergrenze in der Abfrage; nur die offenen Anliegen (erledigt_am IS NULL), eine Arbeitsliste, die mit dem Erledigen schrumpft (AnliegenAbfragen.LadeOffene)",
	"GET /api/audit/tresen-auskunft $.ereignisse":                         "LIMIT 100 (thekenMaxEreignisse, api/system/theken_auskunft.go; repository.FindeThekenEreignisse)",
	"GET /api/audit/tresen-auskunft $.exemplare":                          "LIMIT 20: die Exemplare, die der eine Barcode heute oder früher bezeichnete (repository.FindeThekenExemplare)",
	"GET /api/benutzer $":                                                 "keine Obergrenze in der Abfrage; alle Zugangskonten, gelöschte verlassen die Tabelle (KontoAbfragen.LadeAlle)",
	"GET /api/bescheide $":                                                "LIMIT 500, die mit abgelaufener Frist zuerst, danach die neuesten (BescheidAbfragen.LadeListe)",
	"GET /api/bescheide/ausstehend $":                                     "LIMIT 500, je Kind eine Zeile, die ältesten Forderungen zuerst (BescheidAbfragen.LadeForderungenOhneBescheid)",
	"GET /api/bestand/abgangsbuch $.abschnitte":                           "höchstens drei, je Topf einer (abschnitteAus, api/katalog/bestandsbuch.go); die Zeilen darin begrenzt der Zeitraum ?von/?bis, Vorgabe das laufende Halbjahr, höchstens ein Jahr (bestandsbuchZeitraum)",
	"GET /api/bestand/zugangsbuch $.abschnitte":                           "höchstens drei, je Topf einer (abschnitteAus, api/katalog/bestandsbuch.go); die Zeilen darin begrenzt der Zeitraum ?von/?bis, Vorgabe das laufende Halbjahr, höchstens ein Jahr (bestandsbuchZeitraum)",
	"GET /api/bestellhistorie $":                                          "LIMIT aus ?limit=, Vorgabe 200, höchstens 500 (bestellhistorieVorgabeGrenze, api/bestellung/bestellhistorie.go; repository.LadeBestellVerlauf); die Summen zählt /api/bestellhistorie/uebersicht über alles",
	"GET /api/bestellhistorie/uebersicht $.nach_mittel":                   "drei feste Töpfe: Land, Schulträger, ohne Zuordnung (mitteltopf.Reihenfolge)",
	"GET /api/bestellungen $":                                             "keine Obergrenze in der Abfrage; je Buch unter der Bestellschwelle eine Zeile, höchstens so viele wie Bücher im Katalog; bei abgeschalteter Warnung leer (repository.LadeBestellbedarf)",
	"GET /api/bestellungen/zulauf $":                                      "keine Obergrenze in der Abfrage; die bestellten, noch nicht eingetroffenen Exemplare, je Bestellung eine Gruppe (repository.LadeExemplareImZulauf, service.LadeZulauf)",
	"GET /api/class-books $":                                              "keine Obergrenze in der Abfrage; je Klasse eine Gruppe, wächst mit der Schule, nicht mit den Jahren (TitelAbfragen.LadeKlassensatzGruppen)",
	"GET /api/portal/klassensaetze $":                                     "keine Obergrenze in der Abfrage; je Klasse eine Gruppe, wächst mit der Schule, nicht mit den Jahren (TitelAbfragen.LadeKlassensatzGruppen)",
	"GET /api/admin/class-books $":                                        "keine Obergrenze in der Abfrage; je Klasse eine Gruppe, wächst mit der Schule, nicht mit den Jahren (TitelAbfragen.LadeKlassensatzGruppen)",
	"GET /api/admin/books/external-covers $":                              "LIMIT 300, die Titel mit Cover auf fremder Adresse (FremdeCoverListe, api/katalog/cover_fremd.go; TitelAbfragen.LadeTitelMitFremdemCover)",
	"GET /api/einstellungen $.sommerferien_programm":                      "feste Tabelle der hessischen Sommerferien im Programm, je Jahr ein Eintrag (lmfplan.ProgrammEintraege, pkg/lmfplan/ferien_einstellung.go)",
	"GET /api/exemplare/etiketten-offen $":                                "LIMIT 300, die neuesten zuerst (etikettenOffenLimit, api/druck/etiketten_offen.go); die Zahl ohne Grenze liefert /api/exemplare/etiketten-offen/anzahl",
	"GET /api/exemplare/standorte $":                                      "LIMIT 1000, die häufigsten zuerst (exemplarStandorteMax, repository.LadeExemplarStandorte)",
	"GET /api/geraete $":                                                  "keine Obergrenze in der Abfrage; die Geräte im Bestand, ausgesonderte nicht (GeraeteAbfragen.LadeAlle)",
	"GET /api/inventur/abgeschlossen $.laeufe":                            "LIMIT 10 je Seite, die zuletzt abgeschlossenen zuerst; ältere über naechste_seite, die Oberfläche nennt sie in „Mehr laden (N weitere)“. Kein Index: Die Tabelle wächst um wenige Läufe im Jahr (laeufeJeSeite, api/inventurlauf/inventur_laeufe.go; InventurAbfragen.LadeAbgeschlosseneLaeufe)",
	"GET /api/inventur/sessions $":                                        "keine Obergrenze in der Abfrage; nur laufende Inventuren, je Bereich höchstens eine (UNIQUE idx_inv_session_offen_*; InventurAbfragen.LadeOffeneLaeufe)",
	"GET /api/klassen-mapping $":                                          "keine Obergrenze in der Abfrage; je Klasse höchstens eine Zuordnung (PRIMARY KEY klasse; repository.LadeKlassenleitungen)",
	"GET /api/lieferanten $":                                              "keine Obergrenze in der Abfrage; alle Lieferanten, von Hand gepflegt (repository.LadeLieferanten)",
	"GET /api/lmf-plan/{art} $.ausgelassen":                               "keine Obergrenze in der Abfrage; die ausgelassenen Klassen des einen neuesten Plans der Art (LMFAbfragen.ladeAusgelassene)",
	"GET /api/lmf-plan/{art} $.ausgelassen_regel":                         "eine Teilmenge von $.klassen: die Klassen, die die Regel der Art auslässt (service.IstAusgelassenImLMFPlan)",
	"GET /api/lmf-plan/{art} $.eingangsjahrgaenge":                        "höchstens dreizehn: die Jahrgänge 1 bis 13 aus der Einstellung, jeder einmal (repository.EingangsjahrgaengeAus)",
	"GET /api/lmf-plan/{art} $.klassen":                                   "keine Obergrenze in der Abfrage; die Klassen mit aktiven Schülern, wächst mit der Schule, nicht mit den Jahren (LMFAbfragen.LadeKlassenMitSchuelern)",
	"GET /api/lmf-plan/{art} $.plan.freie_tage":                           "keine Obergrenze in der Abfrage; die freien Tage des einen Plans (LMFAbfragen.ladeFreieTage)",
	"GET /api/lmf-plan/{art} $.zeilen":                                    "keine Obergrenze in der Abfrage; die Zeilen des einen neuesten Plans der Art, je Gruppe von Klassen eine (LMFAbfragen.ladePlanZeilen)",
	"GET /api/lmf-termine $.eingangsjahrgaenge":                           "höchstens dreizehn: die Jahrgänge 1 bis 13 aus der Einstellung, jeder einmal (repository.EingangsjahrgaengeAus)",
	"GET /api/lmf-termine $.termine":                                      "keine Obergrenze in der Abfrage; die Termine veröffentlichter Pläne ab Beginn des laufenden Schuljahres (lmfPlanAb, api/lernmittel/lmf_termine.go); mit ?alle=1 alle Jahre, zwei Pläne je Schuljahr (LMFAbfragen.LadeTermine)",
	"GET /api/mahnwesen $.klassen":                                        "keine Obergrenze in der Abfrage; je Klasse mit überfälligen Ausleihen eine Gruppe, die Ehemaligen als eigene am Ende (MahnwesenAbfragen.LadeUeberfaelligeNachKlasse)",
	"GET /api/portal/lernmittel $.faecher":                                "keine Obergrenze in der Abfrage; je Fach der Schulbücher eine Zeile, dazu eine ohne Fach (TitelAbfragen.LadeLernmittelFaecher)",
	"GET /api/portal/lernmittel $.titel":                                  "keine Obergrenze in der Abfrage; die Schulbücher eines Fachs, ohne ?fach alle, ein Buch in mehreren Auflagen einmal; wächst mit dem Bestand an Lernmitteln (TitelAbfragen.LadeLernmittelTitel)",
	"GET /api/reservierungen/klassensatz $":                               "keine Obergrenze in der Abfrage; die offenen und die erledigten, die erledigten löscht der Nachtlauf nach der Frist der Anliegen, Vorgabe 365 Tage; mit Frist 0 wächst die Liste mit den Jahren (ReservierungsAbfragen.LadeAlle, repository.LoescheErledigteKlassensatzReservierungen)",
	"GET /api/reservierungen/klassensatz/eigene $":                        "LIMIT 50 (ReservierungsAbfragen.LadeEigene)",
	"GET /api/reservierungen/klassensatz/katalog $":                       "LIMIT 50 (opacGrenze, api/portal/opac.go; repository.FindeImKatalog)",
	"GET /api/reservierungen/klassensatz/katalog/filter $":                "keine Obergrenze in der Abfrage; die Schlagworte, die auf der Pflegeseite als Filter markiert sind (ist_filter), von Hand gesetzt (repository.LadeSchlagwortFilterImKatalog)",
	"GET /api/reservierungen/klassensatz/offen $":                         "keine Obergrenze in der Abfrage; nur die offenen (erledigt = false), eine Warteschlange, die mit dem Erledigen schrumpft (ReservierungsAbfragen.LadeOffene)",
	"GET /api/schlagworte $":                                              "LIMIT 500 ohne Suche, die häufigsten zuerst; mit ?suche= LIMIT 50 (schlagwortVorschlaegeMax, schlagwortVorschlaegeSucheMax; repository.FindeSchlagwortVorschlaege)",
	"GET /api/schlagworte/pflege $.zeilen":                                "LIMIT 200, alphabetisch; die Zahlen der ganzen Tabelle stehen daneben (schlagwortPflegeTrefferMax, repository.LadeSchlagworteZurPflege)",
	"GET /api/schueler $":                                                 "LIMIT 500 ohne Suchbegriff (leserListeMax, repository/leser_liste.go); mit Suchbegriff keine Obergrenze, mit Absicht: Eine Suche muss jede Zeile erreichen",
	"GET /api/signaturen/buecher $.buecher":                               "500 Titel; eine Zeile mehr wird gelesen, damit die Antwort die Kappung ansagt (regalansichtGrenze, api/katalog/signaturen.go; repository.LadeTitelUnterSignatur)",
	"GET /api/systematics $":                                              "keine Obergrenze in der Abfrage; alle Sachgruppen der Systematik, wächst mit der Ordnung des Bestands, nicht mit den Büchern (repository.LadeSystematikKategorien)",
	"GET /api/vormerkungen $":                                             "keine Obergrenze in der Abfrage; die Vormerkungen des einen Titels oder Schülers, der Filter ist Pflicht; erfüllte und verfallene werden gelöscht, die Tabelle hält nur offene (VormerkungsAbfragen.LadeListe; Migration 040)",
}

// listenPfade nennt die Listen einer JSON-Antwort: die Wurzel selbst und Felder bis zwei
// Ebenen darunter. In die Zeilen einer Liste steigt sie nicht hinab.
func listenPfade(rumpf []byte) []string {
	var wert any
	if err := json.Unmarshal(rumpf, &wert); err != nil {
		return nil
	}
	var aus []string
	sammleListen(wert, "$", 0, &aus)
	sort.Strings(aus)
	return aus
}

func sammleListen(wert any, pfad string, tiefe int, aus *[]string) {
	switch w := wert.(type) {
	case []any:
		*aus = append(*aus, pfad)
	case map[string]any:
		if tiefe >= 2 {
			return
		}
		for feld, inhalt := range w {
			sammleListen(inhalt, pfad+"."+feld, tiefe+1, aus)
		}
	}
}

func TestListenPfade_ErkenntDieFormen(t *testing.T) {
	faelle := []struct {
		rumpf string
		soll  []string
	}{
		{`[{"id":1}]`, []string{"$"}},
		{`[]`, []string{"$"}},
		{`{"data":[],"gesamt":0}`, []string{"$.data"}},
		{`{"plan":{"freie_tage":[]},"zeilen":[{"klassen":["5a"]}]}`, []string{"$.plan.freie_tage", "$.zeilen"}},
		{`{"a":{"b":{"c":[]}}}`, nil},
		{`{"name":"x","anzahl":3}`, nil},
		{`null`, nil},
		{`kein json`, nil},
	}
	for _, f := range faelle {
		if ist := listenPfade([]byte(f.rumpf)); !cmp.Equal(ist, f.soll) {
			t.Errorf("listenPfade(%s) = %v, erwartet %v", f.rumpf, ist, f.soll)
		}
	}
}

func TestListenNennenIhreGrenze(t *testing.T) {
	pool := pgtest.Pool(t)
	t.Setenv("RATE_LIMIT", "100000")
	// Ein definierter Anfang: Was ein früherer Test liegen ließ, änderte sonst, welche Felder
	// einer Antwort gefüllt sind.
	testdaten.LeereBestandsdaten(t, pool)
	if err := (&db.Verbindung{Pool: pool}).SchreibeRechteVorgabe(context.Background()); err != nil {
		t.Fatalf("SchreibeRechteVorgabe: %v", err)
	}
	authenticator, err := auth.NewAnmeldung(
		"listen-gate-testgeheimnis-mind-32-bytes-lang!", pool, time.Hour)
	if err != nil {
		t.Fatalf("NewAnmeldung: %v", err)
	}
	srv := NewServer(&db.Verbindung{Pool: pool}, authenticator, sse.NewBroker(), ServerKonfig{})
	srv.Uhr = testdaten.SaisonUhr
	router := srv.Routes()

	welt := baueKanarienWelt(t, pool, authenticator)
	aufrufe := bauePIIAufrufe(welt)
	matrix := leseMatrix(t)

	routen := make([]string, 0, len(aufrufe))
	for route := range aufrufe {
		routen = append(routen, route)
	}
	sort.Strings(routen)

	gefunden := map[string]bool{}
	for _, route := range routen {
		permission, mitSitzung := rechtFuerZeile(matrix[route].Recht)
		tuertest.SetzeGenauEinRecht(t, pool, &srv.Rechte, permission)
		req := httptest.NewRequest(http.MethodGet, aufrufe[route].URL, nil)
		if mitSitzung {
			req.AddCookie(&http.Cookie{Name: "session_token", Value: welt.anmeldeToken})
		}
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		if rec.Code >= 400 || !strings.HasPrefix(rec.Header().Get("Content-Type"), "application/json") {
			continue
		}
		for _, pfad := range listenPfade(rec.Body.Bytes()) {
			gefunden[route+" "+pfad] = true
		}
	}
	// Nicht-leer-Garantie: Findet der Lauf kaum Listen, erreicht er die Routen nicht mehr.
	if len(gefunden) < 40 {
		t.Fatalf("nur %d Listen gefunden — die Aufrufe erreichen die Routen nicht mehr, dieses Gate wäre still grün", len(gefunden))
	}

	var neu, weg []string
	for liste := range gefunden {
		if strings.TrimSpace(listenGrenze[liste]) == "" {
			neu = append(neu, liste)
		}
	}
	for liste := range listenGrenze {
		if !gefunden[liste] {
			weg = append(weg, liste)
		}
	}
	sort.Strings(neu)
	sort.Strings(weg)
	if len(neu) > 0 {
		t.Errorf("Liste ohne Antwort:\n  %s\n\n"+
			"Was begrenzt sie? Eine Tabelle, die mit der Zeit wächst, braucht eine Obergrenze in der "+
			"Abfrage, einen Index auf der Sortierspalte und die Ansage der Kappung in der Oberfläche; "+
			"der Test dazu läuft gegen mehr Zeilen als die Grenze. Die Antwort gehört nach "+
			"listenGrenze. War die Liste schon da und erscheint erst jetzt, weil die gesäte Welt "+
			"sie füllt, gilt dieselbe Frage.", strings.Join(neu, "\n  "))
	}
	if len(weg) > 0 {
		t.Errorf("Eintrag ohne Liste in der Antwort:\n  %s\n\nEintrag austragen oder den Ort berichtigen.",
			strings.Join(weg, "\n  "))
	}
}
