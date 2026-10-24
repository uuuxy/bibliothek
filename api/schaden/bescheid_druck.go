package schaden

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"bibliothek/api/tuer"
	"bibliothek/apierrors"
	"bibliothek/pdf"
	"bibliothek/repository"

	"github.com/jackc/pgx/v5"
)

// Der Brief zu einem Bescheid, beim ersten Druck und bei jedem Nachdruck aus dem Schnappschuss.

// BescheidPDF liefert den Brief — beim ersten Mal und bei jedem Nachdruck
// derselbe, aus dem Snapshot.
func (t *Tueren) BescheidPDF(bescheidAbfragen *repository.BescheidAbfragen) http.HandlerFunc {
	return apierrors.MitFehlerantwort(func(w http.ResponseWriter, r *http.Request) error {
		id := r.PathValue("id")
		if id == "" {
			return apierrors.BadRequest(tuer.MeldungIDLeer, errors.New("missing id"))
		}
		ctx := r.Context()

		brief, bescheid, err := t.bescheidBrief(ctx, bescheidAbfragen, id)
		if err != nil {
			return err
		}
		roh, err := pdf.ErzeugeBescheid(brief)
		if err != nil {
			return apierrors.Internal("Bescheid konnte nicht erzeugt werden", err)
		}
		if err := bescheidAbfragen.VermerkeDruck(ctx, id); err != nil {
			// Der Vermerk ist Buchführung, nicht der Zweck: Der Brief geht trotzdem raus.
			slog.Error("Druckvermerk für Bescheid fehlgeschlagen", "bescheid", id, "fehler", err)
		}

		w.Header().Set("Content-Type", tuer.ContentTypePDF)
		w.Header().Set("Content-Disposition",
			fmt.Sprintf(`attachment; filename="bescheid_%s.pdf"`,
				strings.ReplaceAll(bescheid.Referenznummer, " ", "-")))
		// #nosec G705 - der Körper ist das erzeugte PDF (application/pdf, als Anhang, nosniff); aus der Anfrage stammt nur die Kennung des Bescheids
		if _, err := w.Write(roh); err != nil {
			slog.Warn("Bescheid-PDF: Antwort an den Browser abgebrochen", "fehler", err)
		}
		return nil
	})
}

// bescheidBrief baut das Blatt aus Bescheid, Snapshot, Positionen und Einstellungen.
func (t *Tueren) bescheidBrief(ctx context.Context, bescheidAbfragen *repository.BescheidAbfragen, id string) (pdf.BescheidBrief, *repository.Bescheid, error) {
	bescheid, err := bescheidAbfragen.Lade(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return pdf.BescheidBrief{}, nil, apierrors.NotFound("Bescheid nicht gefunden", err)
		}
		return pdf.BescheidBrief{}, nil, apierrors.Internal("Bescheid konnte nicht gelesen werden", err)
	}
	snapshot, err := bescheidAbfragen.LadeEmpfaengerSnapshot(ctx, id)
	if err != nil {
		return pdf.BescheidBrief{}, nil, apierrors.Internal("Empfänger konnte nicht gelesen werden", err)
	}
	positionen, err := bescheidAbfragen.LadePositionen(ctx, id)
	if err != nil {
		return pdf.BescheidBrief{}, nil, apierrors.Internal("Positionen konnten nicht gelesen werden", err)
	}
	// Die Angaben der Schule, wie sie im Brief standen. Ein älterer Bescheid ohne
	// Schnappschuss nimmt die Einstellungen von heute.
	absender, err := bescheidAbfragen.LadeAbsenderSnapshot(ctx, id)
	if err != nil {
		return pdf.BescheidBrief{}, nil, apierrors.Internal("Angaben der Schule konnten nicht gelesen werden", err)
	}
	if len(absender) == 0 {
		angaben, schule, err := t.bescheidAngaben(ctx)
		if err != nil {
			return pdf.BescheidBrief{}, nil, apierrors.Internal(tuer.MeldungEinstellungenUnlesbar, err)
		}
		absender = bescheidAbsenderAus(angaben, schule)
	}

	brief := pdf.BescheidBrief{
		Schule:            bescheidSchuleAus(absender),
		Empfaenger:        bescheidEmpfaengerAus(snapshot),
		Geschaeftszeichen: absender["geschaeftszeichen"],
		Bearbeiter:        absender["bearbeiter"],
		Durchwahl:         absender["durchwahl"],
		BriefDatum:        bescheid.BriefDatum,
		FristBis:          bescheid.FristBis,
		Gesamtbetrag:      bescheid.Gesamtbetrag,
		Zahlstelle:        absender["zahlstelle"],
		Bankverbindung:    absender["bankverbindung"],
		Referenznummer:    bescheid.Referenznummer,
		Aufsicht:          absender["aufsicht"],
		Schulleitung:      absender["schulleitung"],
	}
	name := brief.Empfaenger.Name
	for _, p := range positionen {
		zeile := pdf.BescheidPosition{SchuelerName: name, Titel: p.Titel, ISBN: p.ISBN, Betrag: p.Betrag}
		if p.Art == "nicht_zurueckgegeben" {
			brief.NichtZurueckgegeben = append(brief.NichtZurueckgegeben, zeile)
			continue
		}
		brief.Beschaedigt = append(brief.Beschaedigt, zeile)
	}
	return brief, bescheid, nil
}
