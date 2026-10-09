package api

import (
	"fmt"

	"bibliothek/pdf"
	"bibliothek/pkg/schulzeit"
	"bibliothek/repository"
)

// Bausteine der Mail mit der Mahnliste einer Klasse: Blatt, Statistik und Aufbau der Mail. Der
// Massenversand (mahnwesen_bulk_mail.go) schickt je gewählte Klasse eine Liste an die
// Klassenleitung.

// mahnwesenSendenRequest: Klasse und Zieladresse einer Klassen-Mahnliste.
type mahnwesenSendenRequest struct {
	Klasse string `json:"klasse"`
	Email  string `json:"email"`
}

// mahnlisteSeiten füllt die Eingabe der Mahnliste: je Schüler eine Seite, in der Reihenfolge
// der Gruppen. Klassenleitung und Gruppe stehen nicht auf dem Blatt.
func mahnlisteSeiten(klassen []repository.MahnwesenKlasse) []pdf.MahnlisteSchueler {
	var seiten []pdf.MahnlisteSchueler
	for _, kl := range klassen {
		for _, sch := range kl.Schueler {
			medien := make([]pdf.MahnlisteMedium, 0, len(sch.Medien))
			for _, med := range sch.Medien {
				medien = append(medien, pdf.MahnlisteMedium{
					Titel:            med.Titel,
					Autor:            med.Autor,
					Barcode:          med.Barcode,
					CoverURL:         med.CoverURL,
					FaelligAm:        med.FaelligAm,
					TageUeberfaellig: med.TageUeberfaellig,
				})
			}
			seiten = append(seiten, pdf.MahnlisteSchueler{Name: sch.Name, Klasse: sch.Klasse, Medien: medien})
		}
	}
	return seiten
}

// erzeugeMahnliste setzt die Mahnliste der Gruppen als PDF.
func erzeugeMahnliste(klassen []repository.MahnwesenKlasse) ([]byte, error) {
	return pdf.GenerateMahnlistePDF(mahnlisteSeiten(klassen))
}

func zaehleMahnStatistik(klassen []repository.MahnwesenKlasse) (totalSchueler, totalMedien int) {
	for _, kl := range klassen {
		totalSchueler += len(kl.Schueler)
		for _, sch := range kl.Schueler {
			totalMedien += len(sch.Medien)
		}
	}
	return totalSchueler, totalMedien
}

// baueMahnMailRequest baut die E-Mail (Text + PDF-Anhang) für eine Klassen-Mahnliste.
func baueMahnMailRequest(req mahnwesenSendenRequest, pdfBytes []byte, totalSchueler, totalMedien int) MailRequest {
	emailBody := fmt.Sprintf(
		"Sehr geehrte Damen und Herren,\n\n"+
			"anbei erhalten Sie die aktuelle Mahnliste der Schulbibliothek für die Klasse %s (Stand: %s).\n\n"+
			"Betroffene Schüler/innen: %d\n"+
			"Überfällige Medien gesamt: %d\n\n"+
			"Bitte informieren Sie die betroffenen Schüler/innen über die ausstehenden Rückgaben.\n\n"+
			"Mit freundlichen Grüßen,\nSchulbibliothek",
		req.Klasse,
		schulzeit.Jetzt().Format(dateFormatDE),
		totalSchueler,
		totalMedien,
	)

	return MailRequest{
		To:      req.Email,
		Subject: fmt.Sprintf("Mahnliste Schulbibliothek – Klasse %s – %s", req.Klasse, schulzeit.Jetzt().Format(dateFormatDE)),
		Body:    emailBody,
		Attachments: []MailAttachment{
			{
				Name:        fmt.Sprintf("mahnliste_%s_%s.pdf", req.Klasse, schulzeit.Jetzt().Format(dateFormatISO)),
				ContentType: contentTypePDF,
				Data:        pdfBytes,
			},
		},
	}
}
