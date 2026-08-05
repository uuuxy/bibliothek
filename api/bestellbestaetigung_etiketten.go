package api

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/http"

	"bibliothek/apierrors"
	"bibliothek/pkg/httpresp"
)

// Die Etikettenseite des Lieferanten-Links. Sie druckt GENAU die Etiketten, die auch im
// Mailanhang lagen — dieselben Barcodes, dieselbe Auswahl, dieselben Generatoren
// (klein = Bogen wie im Druck-Center, groß = Lernmittel-Etikett). Zwei Wege zum selben
// Buch dürfen nicht zwei verschiedene Aufkleber ergeben.

// ladeBestellEtiketten holt die Exemplare EINER Bestellung, beschränkt auf die
// Positionen, die auch auf dem Barcodebogen der Bestellmail standen.
//
// Ohne die Einschränkung über mit_vorab_barcode bekäme der Lieferant Etiketten für
// Exemplare, die bewusst ohne Vorab-Barcode bestellt wurden — die beklebt dann die
// Bibliothek selbst, und beide würden dasselbe Buch bekleben.
func (s *Server) ladeBestellEtiketten(ctx context.Context, bestellungID string) ([]BarcodeLabelDetail, error) {
	rows, err := s.DB.Pool.Query(ctx, `
		SELECT e.barcode_id, t.titel, coalesce(t.autor, ''), coalesce(t.isbn, ''), coalesce(t.signatur, ''),
		       to_char(e.erworben_am, 'YYYY')
		FROM buecher_exemplare e
		JOIN buecher_titel t ON t.id = e.titel_id
		WHERE e.bestellung_id = $1
		  AND EXISTS (SELECT 1 FROM bestellungen_positionen p
		               WHERE p.bestellung_id = e.bestellung_id
		                 AND p.titel_id = e.titel_id
		                 AND p.mit_vorab_barcode)
		ORDER BY t.titel, e.barcode_id
	`, bestellungID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	etiketten := []BarcodeLabelDetail{}
	for rows.Next() {
		var d BarcodeLabelDetail
		if err := rows.Scan(&d.BarcodeID, &d.Titel, &d.Autor, &d.ISBN, &d.Signatur, &d.AnschaffungsJahr); err != nil {
			return nil, err
		}
		// Das Anschaffungsjahr gehört auf das Etikett: Auf der physischen Vorlage der Schule
		// steht es als „Ansch.J. 2022" unter dem Titel. Es blieb hier zunächst leer, um dem
		// Mailanhang zu gleichen — das war der falsche Bezugspunkt, also trägt jetzt auch der
		// Anhang das Jahr (api/order_service.go).
		etiketten = append(etiketten, d)
	}
	return etiketten, rows.Err()
}

// OeffentlicheEtikettenHandler liefert den Etikettenbogen als PDF (ohne Login, per Token).
func (s *Server) OeffentlicheEtikettenHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		groesse := r.PathValue("groesse")
		if groesse != "klein" && groesse != "gross" {
			apierrors.SendHTTPError(w, http.StatusBadRequest, errors.New("groesse muss 'klein' oder 'gross' sein"))
			return
		}

		bestellungID, err := s.bestellungPerToken(ctx, r.PathValue("token"))
		if err != nil {
			sendeTokenFehler(w, err)
			return
		}

		etiketten, err := s.ladeBestellEtiketten(ctx, bestellungID)
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}
		if len(etiketten) == 0 {
			apierrors.SendHTTPError(w, http.StatusNotFound, errors.New("zu dieser Bestellung gibt es keine Etiketten"))
			return
		}

		daten, err := s.baueEtikettenPDF(ctx, groesse, etiketten)
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}

		w.Header().Set(headerContentType, "application/pdf")
		// inline: Der Lieferant soll den Bogen im Browser sehen und direkt drucken können.
		w.Header().Set("Content-Disposition", fmt.Sprintf("inline; filename=\"etiketten_%s.pdf\"", groesse))
		httpresp.Write(w, daten)
	}
}

// baueEtikettenPDF erzeugt den Bogen in der gewünschten Größe.
func (s *Server) baueEtikettenPDF(ctx context.Context, groesse string, etiketten []BarcodeLabelDetail) ([]byte, error) {
	kopf := s.etikettKopf(ctx)

	if groesse == "gross" {
		return GenerateLernmittelEtikettenPDF(etiketten, kopf)
	}

	doc, err := GenerateLabelsPDF("zweckform_l4760", 1, false, etiketten, kopf)
	if err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	if err := doc.Output(&buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
