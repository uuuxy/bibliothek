package api

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"

	"bibliothek/apierrors"
	"bibliothek/pdf"
	"bibliothek/pkg/mitteltopf"
	"bibliothek/repository"
)

// BarcodeLabelDetail ist ein Buchetikett auf der Seite der Tür: der Eintrag eines Druckauftrags
// aus dem Browser und die Zeile eines Exemplars aus der Datenbank. Die Erzeuger in pdf/
// bekommen daraus pdf.BuchEtikett (buchEtiketten).
type BarcodeLabelDetail struct {
	BarcodeID string
	Titel     string
	Autor     string
	ISBN      string
	// AnschaffungsJahr ist das Jahr aus buecher_exemplare.erworben_am und steht als
	// "Ansch.J. 2016" auf dem Etikett. Leer = Zeile entfällt (etwa bei Vorab-Etiketten
	// für eine Bestellung, deren Exemplare es noch gar nicht gibt).
	AnschaffungsJahr string
	// Signatur ist buecher_titel.signatur (z. B. "LMF-Deutsch 5"). Leer = Zeile entfällt,
	// genau wie bei AnschaffungsJahr.
	Signatur string
	// Topf ist der Topf des Exemplars (repository.ExemplarTopfSQL) und entscheidet, welcher
	// Eigentumsvermerk auf dem Etikett steht (EtikettKopf.vermerkFuer). Er kommt immer vom
	// Server: json:"-", damit ihn kein Druckauftrag aus dem Browser mitbringen kann. Leer =
	// Exemplar unbekannt (Vorab-Druck), dann gilt der allgemeine Vermerk.
	Topf string `json:"-"`
}

// EtikettKopf trägt die schulweiten Angaben, die auf jedem Etikett stehen. Sie kommen aus den
// Systemeinstellungen und nicht aus dem einzelnen Exemplar: Das Etikett nennt die Schule, der
// das Buch gehört, und zeigt im Verlustfall den Weg zurück.
type EtikettKopf struct {
	Schulname        string // z. B. "Philipp-Reis-Schule, Friedrichsdorf"
	Eigentumsvermerk string // z. B. "Eigentum des Landes Hessen"
	// EigentumsvermerkSchuelerbuecherei gilt für Exemplare aus Mitteln des Schulträgers.
	// Leer heißt kein Vermerk; eine Werksvorgabe gibt es hier nicht.
	EigentumsvermerkSchuelerbuecherei string
}

// vermerkFuer wählt den Eigentumsvermerk nach dem Topf des Exemplars: Das Eigentum folgt dem
// Geld. Eine Stelle für beide Erzeuger (kleines Etikett ab 30 mm, großes Lernmittel-Etikett);
// buchEtiketten ruft sie je Exemplar und reicht den gewählten Vermerk weiter.
func (k EtikettKopf) vermerkFuer(topf string) string {
	if topf == mitteltopf.Schultraeger {
		return k.EigentumsvermerkSchuelerbuecherei
	}
	return k.Eigentumsvermerk
}

// parseLabelParams liest Format, Startposition und QR-Flag aus den Query-Parametern
// (mit denselben Defaults wie bisher).
func parseLabelParams(r *http.Request) (formatId string, startPos int, isQR bool) {
	formatId = r.URL.Query().Get("format")
	if formatId == "" {
		formatId = "avery_3475" // default as before
	}

	startPos = 1
	if startParam := r.URL.Query().Get("start"); startParam != "" {
		if parsed, err := strconv.Atoi(startParam); err == nil && parsed > 0 {
			startPos = parsed
		}
	}

	isQR = r.URL.Query().Get("qr") == "true"
	return formatId, startPos, isQR
}

// queryLabelItems lädt alle Exemplare (Barcode, Titel, Autor, Anschaffungsjahr, Signatur) eines Titels.
func (s *Server) queryLabelItems(ctx context.Context, id string) ([]BarcodeLabelDetail, error) {
	zeilen, err := repository.EtikettDatenDesTitels(ctx, s.DB.Pool, id)
	if err != nil {
		return nil, err
	}
	return etikettenAus(zeilen), nil
}

// etikettKopf lädt Schulname und Eigentumsvermerk aus den Systemeinstellungen.
//
// Fehlt der Schulname, bleibt die Zeile LEER statt auf einen erfundenen Wert
// zurückzufallen: Ein Etikett, das die falsche Schule nennt, führt ein gefundenes Buch
// in die Irre. Der Eigentumsvermerk hat dagegen eine sinnvolle Vorgabe, weil er für
// alle Bücher desselben Trägers gleich lautet.
func (s *Server) etikettKopf(ctx context.Context) EtikettKopf {
	settings, err := repository.NewSystemSettingsRepository(s.DB.Pool).GetSettings(ctx)
	if err != nil {
		log.Printf("Etiketten: Einstellungen nicht lesbar, drucke ohne Schulnamen: %v", err)
		return EtikettKopf{Eigentumsvermerk: repository.StandardEigentumsvermerk}
	}
	return etikettKopfAus(settings)
}

// etikettKopfAus baut den Kopf aus den Einstellungen: eine Stelle für Selbstdruck,
// Lieferanten-Link und Mailanhang, damit die Regel „leer = Werksvorgabe" nur einmal steht.
//
// Die Werksvorgabe gilt nur für den allgemeinen Vermerk. Der Vermerk der Schülerbücherei
// bleibt leer, wenn nichts hinterlegt ist — leer heißt dort kein Vermerk.
func etikettKopfAus(settings *repository.SystemEinstellungen) EtikettKopf {
	kopf := EtikettKopf{
		Schulname:                         settings.SchuleName,
		Eigentumsvermerk:                  repository.StandardEigentumsvermerk,
		EigentumsvermerkSchuelerbuecherei: settings.EtikettEigentumsvermerkSchuelerbuecherei,
	}
	if settings.EtikettEigentumsvermerk != "" {
		kopf.Eigentumsvermerk = settings.EtikettEigentumsvermerk
	}
	return kopf
}

// buchEtiketten füllt die Eingabe der Etiketten-Erzeuger aus den Etiketten eines Auftrags
// und dem Kopf aus den Einstellungen. Der Eigentumsvermerk wird hier je Exemplar gewählt,
// für alle Druckwege an dieser einen Stelle: Der Erzeuger kennt den Topf nicht und druckt
// den Vermerk, den er bekommt.
func buchEtiketten(items []BarcodeLabelDetail, kopf EtikettKopf) []pdf.BuchEtikett {
	etiketten := make([]pdf.BuchEtikett, 0, len(items))
	for _, item := range items {
		etiketten = append(etiketten, pdf.BuchEtikett{
			Schulname:        kopf.Schulname,
			BarcodeID:        item.BarcodeID,
			Titel:            item.Titel,
			Autor:            item.Autor,
			AnschaffungsJahr: item.AnschaffungsJahr,
			Signatur:         item.Signatur,
			Eigentumsvermerk: kopf.vermerkFuer(item.Topf),
		})
	}
	return etiketten
}

// LabelsHandler returns a handler that generates an A4 PDF containing 3x8 Avery labels
// for all copies of a given book title.
func (s *Server) LabelsHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		if id == "" {
			apierrors.SendHTTPError(w, http.StatusBadRequest, fmt.Errorf("id is required"))
			return
		}

		ctx := r.Context()
		formatId, startPos, isQR := parseLabelParams(r)

		items, err := s.queryLabelItems(ctx, id)
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}
		if len(items) == 0 {
			apierrors.SendHTTPError(w, http.StatusBadRequest, fmt.Errorf("keine exemplare für diesen titel vorhanden"))
			return
		}

		bogen, err := pdf.GenerateLabelsPDF(formatId, startPos, isQR, buchEtiketten(items, s.etikettKopf(ctx)))
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, fmt.Errorf("fehler bei der pdf generierung: %w", err))
			return
		}

		w.Header().Set(headerContentType, contentTypePDF)
		w.Header().Set(headerContentDisposition, fmt.Sprintf("inline; filename=\"etiketten_%s.pdf\"", id))

		if err := bogen.Output(w); err != nil {
			log.Printf("Fehler beim Senden des PDFs: %v", err)
		}
	}
}

// ergaenzeServerfelder füllt die Etikettenfelder nach, die NICHT aus der Anfrage
// stammen: Anschaffungsjahr und Signatur.
//
// Der Nachdruck-Dialog schickt nur Barcode, Titel und Autor — beides stünde sonst nie
// auf einem nachgedruckten Etikett. Es dem Client mitzugeben wäre der falsche Weg: Es
// ist Serverwissen, und ein Feld, das die Oberfläche mitschicken MUSS, wird irgendwann
// vergessen (dieselbe Klasse Fehler hat hier schon Buchdaten still verschluckt).
//
// Genau das ist der Signatur passiert: Diese Nachfüllung gab es zuerst nur fürs Jahr,
// also trug dasselbe Buch je nach Druckweg einen anderen Aufkleber — über
// GET /api/buecher/titel/{id}/etiketten mit Signatur, über POST /api/print/labels ohne.
// Wer hier ein Feld ergänzt, ergänzt es auch in queryLabelItems und
// ladeBestellEtiketten; TestEtikettenWegeDruckenDasselbe hält die Wege am fertigen PDF
// zusammen.
//
// Eine Abfrage für alle Barcodes, kein N+1. Unbekannte Barcodes bleiben unangetastet —
// beim Vorab-Druck für eine Bestellung existieren die Exemplare noch gar nicht, dort
// ist das der richtige Zustand und kein Fehler.
func (s *Server) ergaenzeServerfelder(ctx context.Context, items []BarcodeLabelDetail) {
	barcodes := make([]string, 0, len(items))
	for i := range items {
		if items[i].BarcodeID != "" {
			barcodes = append(barcodes, items[i].BarcodeID)
		}
	}
	if len(barcodes) == 0 {
		return
	}

	bekannt, err := repository.EtikettServerfelderZuBarcodes(ctx, s.DB.Pool, barcodes)
	if err != nil {
		if errors.Is(err, repository.ErrZeileUnlesbar) {
			log.Printf("Etiketten: Serverfelder unvollständig gelesen: %v", err)
			return
		}
		log.Printf("Etiketten: Serverfelder nicht ermittelbar, drucke ohne: %v", err)
		return
	}

	for i := range items {
		if felder, ok := bekannt[items[i].BarcodeID]; ok {
			items[i].AnschaffungsJahr = felder.Jahr
			items[i].Signatur = felder.Signatur
			items[i].Topf = felder.Topf
		}
	}
}

// PrintLabelsRequest represents a request to generate a PDF label sheet.
type PrintLabelsRequest struct {
	FormatID      string               `json:"formatId"`
	StartPosition int                  `json:"startPosition"`
	IsQR          bool                 `json:"isQR"`
	Items         []BarcodeLabelDetail `json:"items"`
}

// PrintLabelsHandler generates an A4 PDF containing labels dynamically.
func (s *Server) PrintLabelsHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req PrintLabelsRequest
		if !DecodeAndValidate(w, r, &req) {
			return
		}

		if len(req.Items) == 0 {
			apierrors.SendHTTPError(w, http.StatusBadRequest, fmt.Errorf("keine exemplare angegeben"))
			return
		}

		ctx := r.Context()
		s.ergaenzeServerfelder(ctx, req.Items)

		bogen, err := pdf.GenerateLabelsPDF(req.FormatID, req.StartPosition, req.IsQR, buchEtiketten(req.Items, s.etikettKopf(ctx)))
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, fmt.Errorf("fehler bei der pdf generierung: %w", err))
			return
		}

		w.Header().Set(headerContentType, contentTypePDF)
		w.Header().Set(headerContentDisposition, "inline; filename=\"etiketten_custom.pdf\"")

		if err := bogen.Output(w); err != nil {
			log.Printf("Fehler beim Senden des PDFs: %v", err)
		}
	}
}
