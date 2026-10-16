package mahnwesen

import (
	"fmt"
	"strings"

	"bibliothek/api/tuer"
	"bibliothek/mailservice"
	"bibliothek/pdf"
	"bibliothek/pkg/schulzeit"
	"bibliothek/repository"
)

// Bausteine der Mail mit der Mahnliste einer Klasse: Blatt, Statistik und Aufbau der Mail. Der
// Massenversand (mahnlisten_versand.go) schickt je gewählte Klasse eine Liste an die
// Klassenleitung.

// mahnlisteEmpfaenger: Klasse und Zieladresse einer Klassen-Mahnliste.
type mahnlisteEmpfaenger struct {
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
	return pdf.ErzeugeMahnliste(mahnlisteSeiten(klassen))
}

func mahnStatistik(klassen []repository.MahnwesenKlasse) (schuelerGesamt, medienGesamt int) {
	for _, kl := range klassen {
		schuelerGesamt += len(kl.Schueler)
		for _, sch := range kl.Schueler {
			medienGesamt += len(sch.Medien)
		}
	}
	return schuelerGesamt, medienGesamt
}

// baueMahnMail baut die E-Mail (Text + PDF-Anhang) für eine Klassen-Mahnliste.
func baueMahnMail(empfaenger mahnlisteEmpfaenger, pdfDaten []byte, schuelerGesamt, medienGesamt int) mailservice.Nachricht {
	text := fmt.Sprintf(
		"Sehr geehrte Damen und Herren,\n\n"+
			"anbei erhalten Sie die aktuelle Mahnliste der Schulbibliothek für die Klasse %s (Stand: %s).\n\n"+
			"Betroffene Schüler/innen: %d\n"+
			"Überfällige Medien gesamt: %d\n\n"+
			"Bitte informieren Sie die betroffenen Schüler/innen über die ausstehenden Rückgaben.\n\n"+
			"Mit freundlichen Grüßen,\nSchulbibliothek",
		empfaenger.Klasse,
		schulzeit.Jetzt().Format(tuer.DatumsformatDE),
		schuelerGesamt,
		medienGesamt,
	)

	return mailservice.Nachricht{
		An:      empfaenger.Email,
		Betreff: fmt.Sprintf("Mahnliste Schulbibliothek – Klasse %s – %s", empfaenger.Klasse, schulzeit.Jetzt().Format(tuer.DatumsformatDE)),
		Text:    text,
		Anhaenge: []mailservice.Anhang{
			{
				Name:        fmt.Sprintf("mahnliste_%s_%s.pdf", empfaenger.Klasse, schulzeit.Jetzt().Format(tuer.DatumsformatISO)),
				ContentType: tuer.ContentTypePDF,
				Inhalt:      pdfDaten,
			},
		},
	}
}

// alsMeldung setzt Sätze zur Rückmeldung eines Versands zusammen: ein einzelner ohne Punkt,
// mehrere mit (M3, Content design, Grammar and punctuation).
func alsMeldung(saetze ...string) string {
	if len(saetze) == 1 {
		return saetze[0]
	}
	return strings.Join(saetze, ". ") + "."
}

// klassen nennt eine Zahl von Klassen in Einzahl oder Mehrzahl.
func klassen(n int) string {
	if n == 1 {
		return "1 Klasse"
	}
	return fmt.Sprintf("%d Klassen", n)
}

// versandZiel nennt, wohin der Versand ging: an die Klassenleitungen oder an die Umleitung.
func versandZiel(umleitung string) string {
	if umleitung != "" {
		return "an " + umleitung
	}
	return "an die Klassenleitungen"
}

// versandAusfall nennt, was nicht zugestellt wurde, und nach einem Fehler des Mailservers, was zu
// tun ist; hinweis sagt, warum ein zweiter Lauf nicht schadet.
func versandAusfall(erg versandErgebnis, hinweis string) []string {
	var saetze []string
	if erg.Fehlgeschlagen > 0 {
		saetze = append(saetze, fmt.Sprintf("Für %s nicht zugestellt", klassen(erg.Fehlgeschlagen)))
	}
	if erg.Abgebrochen {
		saetze = append(saetze, "Der Lauf brach nach einem Fehler des Mailservers ab",
			"Bitte den Mailserver prüfen und erneut senden", hinweis)
	}
	return saetze
}
