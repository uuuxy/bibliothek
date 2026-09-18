package api

import (
	"fmt"

	"bibliothek/pkg/schulzeit"
	"bibliothek/repository"
)

// Bausteine der Klassen-Mahnliste per E-Mail: Statistik und Mail-Aufbau. Benutzt vom
// Massenversand (mahnwesen_bulk_mail.go), der je gewählte Klasse eine Liste an die
// Klassenleitung schickt.
//
// Bis zum 18.09.2026 stand hier auch der Einzelversand POST /api/mahnwesen/senden — eine
// Klasse, eine frei eingetippte Adresse. Sein Knopf in der Mahnwesen-Tabelle war am
// 21.06.2026 in einem Refactoring verschwunden; seitdem war die Route ohne Aufrufer und
// die Fähigkeit doppelt, denn der Massenversand kann dasselbe (Klassenauswahl plus
// abweichende Adresse). Eine Adresse ohne Aufrufer ist Angriffsfläche und Pflege für
// nichts; sie ist mit Handler, Dialog und Audit-Test (EINZEL_MAHN_MAIL) entfernt.

// mahnwesenSendenRequest: Klasse und Zieladresse einer Klassen-Mahnliste.
type mahnwesenSendenRequest struct {
	Klasse string `json:"klasse"`
	Email  string `json:"email"`
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
			"anbei erhalten Sie die aktuelle Mahntliste der Schulbibliothek für die Klasse %s (Stand: %s).\n\n"+
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
