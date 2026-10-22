package katalog

import (
	"net/http"

	"bibliothek/api/tuer"
	"bibliothek/apierrors"
	"bibliothek/internal/service"
	"bibliothek/repository"
)

// TitelExemplare liefert die Exemplare eines Titels mit Zustand, Eigentum, Standort und dem
// heutigen Ersatzwert.
func (t *Tueren) TitelExemplare(bescheidAbfragen *repository.BescheidAbfragen) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		t.gibTitelExemplare(w, r, bescheidAbfragen)
	}
}

// gibTitelExemplare liest die Exemplare eines Titels mit Eigentum und Ersatzwert.
func (t *Tueren) gibTitelExemplare(w http.ResponseWriter, r *http.Request, bescheidAbfragen *repository.BescheidAbfragen) {
	id := r.PathValue("id")
	if id == "" {
		apierrors.AntworteFehlerMitMeldung(w, http.StatusBadRequest, "Die Kennung des Titels fehlt", nil)
		return
	}

	ctx := r.Context()

	zeilen, err := repository.LadeExemplareDesTitels(ctx, t.DB.Pool, id)
	if err != nil {
		apierrors.AntworteFehler(w, http.StatusInternalServerError, err)
		return
	}

	// exemplarZeile ist ein Exemplar in der Antwort.
	type exemplarZeile struct {
		ID        string `json:"id"`
		BarcodeID string `json:"barcode_id"`
		// LMFBarcode ist die Nummer eines alten LMF-Etiketts, leer ohne. Das Nummernfeld des
		// Druck-Centers findet das Exemplar auch über sie.
		LMFBarcode   string `json:"lmf_barcode"`
		ZustandNotiz string `json:"zustand_notiz"`
		// ZustandAbwertungProzent ist der erfasste Beschädigungsgrad. Er steht hier, weil die
		// Buchakte ihn anzeigt und ihr Status-Editor ihn zurückschickt; ohne den gelesenen Wert
		// wäre jedes Speichern eine Neueingabe.
		ZustandAbwertungProzent int  `json:"zustand_abwertung_prozent"`
		IstAusleihbar           bool `json:"ist_ausleihbar"`
		IstAusgesondert         bool `json:"ist_ausgesondert"`
		// ImBestand: zählt zum Bestand des Titels (repository.SQLExemplarImBestand), also
		// weder ausgesondert noch bestellt. Die Buchmaske listet nur diese.
		ImBestand     bool `json:"im_bestand"`
		IstVerfuegbar bool `json:"ist_verfuegbar"`
		// Ersatzwert und Herleitung: was ein Ersatz für dieses Exemplar heute kostet. Gerechnet
		// wird mit service.ErsatzwertVorschlagAus, derselben Funktion wie im Dialog zum Melden
		// eines Schadens, damit Buchakte und Forderung nie zwei Zahlen für dasselbe Buch zeigen.
		Ersatzwert           float64 `json:"ersatzwert"`
		ErsatzwertHerleitung string  `json:"ersatzwert_herleitung"`
		// ErsatzwertBekannt: Liegt dem Betrag ein Preis zugrunde? Ohne dieses Feld
		// müsste die Karte die Antwort aus dem Herleitungssatz lesen — und 0,00 € heißt
		// zweierlei (Totalschaden oder kein Preis erfasst).
		ErsatzwertBekannt bool `json:"ersatzwert_bekannt"`
		// Eigentum ('land' oder 'schultraeger') und woher es kommt ('littera', 'hand',
		// 'bestellung', 'vorgabe') — dieselbe Regel wie Etikett und Schadensersatz.
		Eigentum         string `json:"eigentum"`
		EigentumHerkunft string `json:"eigentum_herkunft"`
		// LitteraEigentumsvermerk ist der Wortlaut aus Littera (feste Liste), auch wenn er
		// kein Eigentum setzt — die Bücherei sieht, was dort stand.
		LitteraEigentumsvermerk string `json:"littera_eigentumsvermerk"`
		// Standort: wo das Exemplar steht, wenn nicht an seinem Platz nach der Signatur. Leer
		// heißt: nach der Signatur.
		Standort string `json:"standort"`
	}

	exemplare := make([]exemplarZeile, 0, len(zeilen))
	for _, z := range zeilen {
		exemplare = append(exemplare, exemplarZeile{
			ID:                      z.ID,
			BarcodeID:               z.BarcodeID,
			LMFBarcode:              z.LMFBarcode,
			ZustandNotiz:            z.ZustandNotiz,
			ZustandAbwertungProzent: z.ZustandAbwertungProzent,
			IstAusleihbar:           z.IstAusleihbar,
			IstAusgesondert:         z.IstAusgesondert,
			ImBestand:               z.ImBestand,
			IstVerfuegbar:           z.IstVerfuegbar,
			Eigentum:                z.Eigentum,
			EigentumHerkunft:        z.EigentumHerkunft,
			LitteraEigentumsvermerk: z.LitteraEigentumsvermerk,
			Standort:                z.Standort,
		})
	}

	// Der Ersatzwert je Exemplar: eine Abfrage für den ganzen Titel, dann die
	// Staffel im Speicher. Ein Aufruf je Karte wäre bei einem Klassensatz mit
	// 30 Bänden 30 Abfragen für eine Seite.
	//
	// Scheitert das, bleibt die Liste stehen und die Werte fehlen: Der Ersatzwert
	// ist eine Auskunft, kein Grund, die Exemplar-Liste zu verweigern.
	if groessen, err := bescheidAbfragen.LadeGroessenFuerTitel(ctx, id); err == nil {
		quelle := service.LadeErsatzwertPreisquelle(ctx, t.DB.Pool)
		for i := range exemplare {
			g, da := groessen[exemplare[i].ID]
			if !da {
				continue
			}
			v := service.ErsatzwertVorschlagAus(g, quelle)
			exemplare[i].Ersatzwert = v.Betrag
			exemplare[i].ErsatzwertHerleitung = v.Herleitung
			exemplare[i].ErsatzwertBekannt = v.Bekannt
		}
	}

	tuer.AntworteJSON(w, http.StatusOK, exemplare)
}
