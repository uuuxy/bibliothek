package api

import (
	"context"
	"log"
	"net/http"
	"strings"
	"time"

	"bibliothek/apierrors"
	"bibliothek/repository"
)

// etikettenOffenLimit deckelt die Liste. Wer mehr als 300 Etiketten am Stück nachdruckt,
// druckt keinen Nachzügler mehr nach, sondern den halben Bestand — dafür ist der Weg über
// die Titelsuche im Druck-Center gedacht.
const etikettenOffenLimit = 300

// etikettenOffenBedingung ist die EINZIGE Definition von "Etikett steht noch aus".
// Liste und Zähler teilen sie sich — sonst zeigt der Hinweis im Bestellwesen irgendwann
// eine andere Zahl an, als die Liste im Druck-Center Zeilen hat, und keiner der beiden
// Werte ist mehr zu trauen.
//
// Ausgesonderte Exemplare stehen nicht mehr im Regal; für sie ein Etikett zu drucken
// wäre immer falsch.
//
// Der Wert steht seit dem 08.08.2026 in repository/ und wird hier nur noch übernommen:
// Die Bestell-Detailansicht braucht dieselbe Zahl, formuliert ihre Abfrage aber in der
// Repository-Schicht (schichtung_test.go). Zwei Konstanten mit gleichem Inhalt wären
// genau die Drift, gegen die dieser Kommentar seit jeher anschreibt. Alle bisherigen
// Fundstellen — Liste, Zähler und die pg-Tests — lesen unverändert diesen Namen.
const etikettenOffenBedingung = repository.EtikettOffenBedingung

// etikettenStatusBedingung uebersetzt den status-Parameter in sein SQL-Praedikat.
//
// Vorgabe bleibt "offen": Die Liste heisst „Fehlende Etiketten" und soll ohne Zutun
// genau das zeigen. Die anderen Stufen sind Werkzeug, nicht Alltag.
//
// Liste UND Zaehler lesen es hier. Sonst nennt die Fusszeile der Liste ("300 von 30.674")
// irgendwann eine Zahl aus einer anderen Menge als die Zeilen darueber — dieselbe Drift,
// gegen die etikettenOffenBedingung eine Zeile hoeher anschreibt.
func etikettenStatusBedingung(status string) string {
	switch status {
	case "erledigt":
		return `e.etikett_gedruckt = true AND e.ist_ausgesondert = false`
	case "alle":
		return `e.ist_ausgesondert = false`
	default:
		return etikettenOffenBedingung
	}
}

// ExemplarOhneEtikett ist eine Zeile der Nachdruck-Liste. Die Feldnamen barcode_id/titel/
// autor sind KEIN Zufall: In genau dieser Form nimmt der Etikettendruck seine Aufträge
// entgegen (printQueue → labels.svelte.js), die Liste kann also direkt übergeben werden.
type ExemplarOhneEtikett struct {
	BarcodeID  string `json:"barcode_id"`
	Titel      string `json:"titel"`
	Autor      string `json:"autor"`
	ErworbenAm string `json:"erworben_am"`

	// EtikettGedruckt gehört dazu, seit die Liste auch bereits erledigte Exemplare zeigen
	// kann (status=erledigt|alle). Ohne das Feld liesse sich in der gemischten Ansicht
	// nicht unterscheiden, welche Zeile noch ein Etikett braucht.
	EtikettGedruckt bool `json:"etikett_gedruckt"`
}

// EtikettenOffenHandler listet Exemplare, deren Barcode-Etikett noch nicht gedruckt wurde.
//
// Der Anlass: Eine Lieferung kann im System freigegeben sein, ohne dass die Etiketten je
// aus dem Drucker kamen (z. B. weil der Hinweis nach dem Wareneingang weggeklickt wurde).
// Danach gab es keinen Weg mehr zu genau diesen Exemplaren zurück — man hätte jeden Titel
// einzeln suchen müssen, ohne zu wissen, welche es sind.
//
// Sortiert wird NEUESTE ZUERST. Das ist der praktische Fall: Gesucht wird die Lieferung von
// gestern, nicht ein Exemplar von 2019.
//
// Mit status=erledigt oder status=alle zeigt sie zusätzlich Exemplare, die bereits als
// gedruckt vermerkt sind. Das ist der Weg zurück: Nach einem Papierstau oder einem zu weit
// gefassten Altbestands-Stichtag sind Exemplare als erledigt vermerkt, ohne dass ein
// Etikett existiert — vorher waren sie damit dauerhaft aus der Liste verschwunden.
//
// @Summary      Exemplare ohne gedrucktes Etikett
// @Tags         books
// @Produce      json
// @Param        q       query   string  false  "Filter über Titel oder Barcode"
// @Param        status  query   string  false  "offen (Vorgabe) | erledigt | alle"
// @Success      200  {array}  ExemplarOhneEtikett
// @Router       /exemplare/etiketten-offen [get]
func (s *Server) EtikettenOffenHandler() http.HandlerFunc {
	return apierrors.Wrap(func(w http.ResponseWriter, r *http.Request) error {
		suche := strings.TrimSpace(r.URL.Query().Get("q"))

		statusBedingung := etikettenStatusBedingung(r.URL.Query().Get("status"))

		rows, err := s.DB.Pool.Query(r.Context(), `
			SELECT e.barcode_id, t.titel, coalesce(t.autor, ''), to_char(e.erworben_am, 'YYYY-MM-DD'),
			       e.etikett_gedruckt
			FROM buecher_exemplare e
			JOIN buecher_titel t ON t.id = e.titel_id
			WHERE `+statusBedingung+`
			  AND ($1 = '' OR t.titel ILIKE '%' || $1 || '%' OR e.barcode_id ILIKE '%' || $1 || '%')
			ORDER BY e.erworben_am DESC, e.erstellt_am DESC, e.barcode_id
			LIMIT $2
		`, suche, etikettenOffenLimit)
		if err != nil {
			return apierrors.Internal("Fehler beim Laden der offenen Etiketten", err)
		}
		defer rows.Close()

		// Nie nil: Eine leere Liste muss als [] beim Client ankommen, sonst bricht dort
		// .length ab (siehe TestListStudentsLeereListeIstArray).
		liste := make([]ExemplarOhneEtikett, 0)
		for rows.Next() {
			var e ExemplarOhneEtikett
			if err := rows.Scan(&e.BarcodeID, &e.Titel, &e.Autor, &e.ErworbenAm, &e.EtikettGedruckt); err != nil {
				return apierrors.Internal("Fehler beim Lesen der offenen Etiketten", err)
			}
			liste = append(liste, e)
		}
		if err := rows.Err(); err != nil {
			return apierrors.Internal("Fehler beim Lesen der offenen Etiketten", err)
		}

		RespondJSON(w, http.StatusOK, liste)
		return nil
	})
}

// EtikettenOffenAnzahlHandler nennt nur die Anzahl.
//
// Eigene Route statt der Liste, weil der Hinweis im Bestellwesen bei jedem Öffnen lädt:
// Für eine Zahl 300 Zeilen zu übertragen wäre Verschwendung — und ab dem Limit wäre die
// Zahl schlicht falsch (die Liste ist gedeckelt, der Bestand nicht).
//
// Seit dem 04.09.2026 versteht sie dieselben Filter wie die Liste (status, q). Der Grund
// steht in der Fusszeile der Liste: Die zeigt hoechstens 300 Zeilen, sagte aber nicht, wie
// viele es insgesamt sind — am Reiter stand "30674", darunter lagen 300, und nichts
// verband die beiden Zahlen. Ohne die Filter haette die Fusszeile in "Erledigt" und
// "Alle" eine Zahl aus der falschen Menge genannt.
//
// @Summary      Anzahl der Exemplare ohne gedrucktes Etikett
// @Tags         books
// @Produce      json
// @Param        q       query   string  false  "Filter über Titel oder Barcode"
// @Param        status  query   string  false  "offen (Vorgabe) | erledigt | alle"
// @Param        bis     query   string  false  "Stichtag JJJJ-MM-TT (nur mit status=offen sinnvoll)"
// @Success      200  {object}  map[string]int
// @Router       /exemplare/etiketten-offen/anzahl [get]
func (s *Server) EtikettenOffenAnzahlHandler() http.HandlerFunc {
	return apierrors.Wrap(func(w http.ResponseWriter, r *http.Request) error {
		// Optionaler Stichtag: Damit beantwortet dieselbe Route auch "wie viele Exemplare
		// wuerde das Aufraeumen des Altbestands treffen?" — die Zahl, die der Betreiber vor
		// dem Bestaetigen sehen muss.
		bisStr := strings.TrimSpace(r.URL.Query().Get("bis"))
		var bis any
		if bisStr != "" {
			geparst, err := time.Parse(dateFormatISO, bisStr)
			if err != nil {
				return apierrors.BadRequest("Stichtag muss im Format JJJJ-MM-TT angegeben werden", err)
			}
			bis = geparst
		}

		// Derselbe JOIN wie in der Liste. Er ist verlustfrei: titel_id ist NOT NULL.
		var anzahl int
		err := s.DB.Pool.QueryRow(r.Context(), `
			SELECT count(*)
			FROM buecher_exemplare e
			JOIN buecher_titel t ON t.id = e.titel_id
			WHERE `+etikettenStatusBedingung(r.URL.Query().Get("status"))+`
			  AND ($1::date IS NULL OR e.erworben_am <= $1)
			  AND ($2 = '' OR t.titel ILIKE '%' || $2 || '%' OR e.barcode_id ILIKE '%' || $2 || '%')`,
			bis, strings.TrimSpace(r.URL.Query().Get("q")),
		).Scan(&anzahl)
		if err != nil {
			return apierrors.Internal("Fehler beim Zählen der offenen Etiketten", err)
		}

		RespondJSON(w, http.StatusOK, map[string]int{"anzahl": anzahl})
		return nil
	})
}

// markEtikettGedruckt vermerkt den Druck eines EINZELNEN Exemplars (Buchakte,
// Ersatz-Etikett). Ein Fehler wird protokolliert, aber nicht durchgereicht: Das PDF ist
// zu diesem Zeitpunkt erzeugt, und ein misslungener Vermerk darf den Druck nicht als
// gescheitert erscheinen lassen. Der Preis ist ein Exemplar, das erneut auf der Liste
// steht — harmlos gegenüber einem Etikett, das niemand mehr nachdruckt.
func (s *Server) markEtikettGedruckt(ctx context.Context, exemplarID string) {
	_, err := s.DB.Pool.Exec(ctx, `
		UPDATE buecher_exemplare SET etikett_gedruckt = true, aktualisiert_am = CURRENT_TIMESTAMP
		WHERE id = $1 AND etikett_gedruckt = false
	`, exemplarID)
	if err != nil {
		log.Printf("Etikettendruck: Vermerk für Exemplar %s fehlgeschlagen: %v", exemplarID, err)
	}
}

// EtikettenAltbestandRequest nennt den Stichtag, bis zu dem aufgeraeumt wird.
type EtikettenAltbestandRequest struct {
	Bis string `json:"bis"` // YYYY-MM-DD, einschliesslich
}

// EtikettenAltbestandHandler vermerkt alle Exemplare bis zu einem Stichtag als gedruckt.
//
// Der Anlass ist eine Altlast, keine Funktion: etikett_gedruckt wurde bis vor Kurzem
// NIRGENDS gesetzt. Fuer den gesamten Bestand steht deshalb "kein Etikett" — nicht weil
// keins da waere, sondern weil es nie jemand vermerkt hat. Die Nachdruck-Liste zeigt so
// den ganzen Bestand, und der Hinweis im Bestellwesen nennt eine Zahl, die nichts bedeutet.
//
// BEWUSST KEINE MIGRATION. Eine Migration haette beim Update stillschweigend zugeschlagen —
// und dabei genau die Exemplare mitversteckt, die wirklich kein Etikett haben (die
// Lieferung, wegen der die Liste ueberhaupt gebaut wurde). Der Stichtag gehoert dem
// Betreiber: Er weiss, ab wann sein Regal beklebt ist, und sieht vorher, wie viele Zeilen
// es trifft.
//
// Umkehrbar ist das nicht — deshalb nennt die Oberflaeche die Zahl vorher und verlangt
// eine ausdrueckliche Bestaetigung.
//
// @Summary      Etiketten des Altbestands als gedruckt vermerken
// @Tags         books
// @Accept       json
// @Success      200  {object}  map[string]int
// @Router       /exemplare/etiketten-altbestand [post]
func (s *Server) EtikettenAltbestandHandler() http.HandlerFunc {
	return apierrors.Wrap(func(w http.ResponseWriter, r *http.Request) error {
		var req EtikettenAltbestandRequest
		if !DecodeAndValidate(w, r, &req) {
			return nil
		}
		bis, err := time.Parse(dateFormatISO, req.Bis)
		if err != nil {
			return apierrors.BadRequest("Stichtag muss im Format JJJJ-MM-TT angegeben werden", err)
		}

		tag, err := s.DB.Pool.Exec(r.Context(), `
			UPDATE buecher_exemplare e SET etikett_gedruckt = true, aktualisiert_am = CURRENT_TIMESTAMP
			WHERE `+etikettenOffenBedingung+` AND e.erworben_am <= $1
		`, bis)
		if err != nil {
			return apierrors.Internal("Fehler beim Vermerken des Altbestands", err)
		}

		RespondJSON(w, http.StatusOK, map[string]int64{"markiert": tag.RowsAffected()})
		return nil
	})
}

// EtikettenGedrucktRequest nennt die Exemplare, deren Etiketten gedruckt wurden.
type EtikettenGedrucktRequest struct {
	BarcodeIDs []string `json:"barcode_ids"`
}

// EtikettenGedrucktHandler bucht den Druck gegen.
//
// Ohne diesen Schritt wäre die Nachdruck-Liste wertlos: etikett_gedruckt wurde bis hierher
// NIRGENDS auf true gesetzt — der Wert stand seit dem Anlegen der Tabelle auf false und
// blieb es. Die Liste hätte also dauerhaft den kompletten Bestand angezeigt statt der
// Nachzügler, und der Haken "erledigt" wäre eine Anzeige ohne Bedeutung gewesen.
//
// @Summary      Etiketten als gedruckt markieren
// @Tags         books
// @Accept       json
// @Success      200  {object}  map[string]int
// @Router       /exemplare/etiketten-gedruckt [post]
func (s *Server) EtikettenGedrucktHandler() http.HandlerFunc {
	return apierrors.Wrap(func(w http.ResponseWriter, r *http.Request) error {
		var req EtikettenGedrucktRequest
		if !DecodeAndValidate(w, r, &req) {
			return nil
		}
		if len(req.BarcodeIDs) == 0 {
			return apierrors.BadRequest("keine Exemplare angegeben", nil)
		}

		tag, err := s.DB.Pool.Exec(r.Context(), `
			UPDATE buecher_exemplare SET etikett_gedruckt = true, aktualisiert_am = CURRENT_TIMESTAMP
			WHERE barcode_id = ANY($1) AND etikett_gedruckt = false
		`, req.BarcodeIDs)
		if err != nil {
			return apierrors.Internal("Fehler beim Vermerken der gedruckten Etiketten", err)
		}

		RespondJSON(w, http.StatusOK, map[string]int64{"markiert": tag.RowsAffected()})
		return nil
	})
}

// EtikettenZuruecksetzenHandler nimmt den Vermerk „Etikett gedruckt" wieder zurück.
//
// Der Weg zurück, den es bis hierher nicht gab. Alle drei Wege setzten das Kennzeichen nur
// in eine Richtung: der Stapeldruck, der Einzeldruck aus der Buchakte und das einmalige
// Aufräumen des Altbestands. Ging dabei etwas schief, war das Exemplar dauerhaft aus der
// Liste verschwunden — und niemand konnte es zurückholen.
//
// Die beiden Fälle aus dem Betrieb:
//
//  1. PAPIERSTAU. Der Druck wird gegengebucht, sobald das PDF erzeugt ist (siehe
//     markEtikettGedruckt) — ob das Etikett wirklich aus dem Drucker kam, weiss das
//     Programm nicht. Bleibt der Bogen im Gerät, gelten die Exemplare als erledigt.
//  2. ZU WEITER STICHTAG. Beim Altbestand-Aufräumen einen zu späten Tag gewählt, und die
//     frische Lieferung ohne Etikett verschwindet mit. Diese Aktion war deshalb bisher
//     ausdrücklich als unumkehrbar dokumentiert (docs/abnahme_checkliste.md, Flow 4).
//
// Damit ist sie es nicht mehr — und das macht auch das Markieren von Hand erst
// unbedenklich: Keine der beiden Richtungen ist noch eine Einbahnstrasse.
//
// @Summary      Etiketten wieder als offen markieren
// @Tags         books
// @Accept       json
// @Success      200  {object}  map[string]int
// @Router       /exemplare/etiketten-zuruecksetzen [post]
func (s *Server) EtikettenZuruecksetzenHandler() http.HandlerFunc {
	return apierrors.Wrap(func(w http.ResponseWriter, r *http.Request) error {
		var req EtikettenGedrucktRequest
		if !DecodeAndValidate(w, r, &req) {
			return nil
		}
		if len(req.BarcodeIDs) == 0 {
			return apierrors.BadRequest("keine Exemplare angegeben", nil)
		}

		// Ausgesonderte bleiben aussen vor: Für ein Buch, das nicht mehr im Regal steht,
		// wäre ein Etikett immer falsch — dieselbe Regel wie in etikettenOffenBedingung.
		tag, err := s.DB.Pool.Exec(r.Context(), `
			UPDATE buecher_exemplare SET etikett_gedruckt = false, aktualisiert_am = CURRENT_TIMESTAMP
			WHERE barcode_id = ANY($1) AND etikett_gedruckt = true AND ist_ausgesondert = false
		`, req.BarcodeIDs)
		if err != nil {
			return apierrors.Internal("Fehler beim Zurücksetzen der Etiketten", err)
		}

		RespondJSON(w, http.StatusOK, map[string]int64{"zurueckgesetzt": tag.RowsAffected()})
		return nil
	})
}
