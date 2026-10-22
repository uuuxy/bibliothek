// Package buchetikett füllt die Eingabe der Etiketten-Erzeuger für Bücher: aus den Etiketten
// eines Druckauftrags oder den Zeilen einer Abfrage und aus dem Kopf der Schule. Den
// Eigentumsvermerk wählt es je Exemplar nach dem Topf. Die Daten eines Etiketts lädt es auf
// zwei Wegen: für die Exemplare einer Bestellung und als Nachtrag an einem Druckauftrag aus dem
// Browser.
//
// Das Paket steht unter den Türen, weil zwei Themen Buchetiketten drucken, das Druck-Center
// und die Bestellung, und kein Thema ein anderes einbindet.
package buchetikett

import (
	"context"
	"log"

	"bibliothek/db"
	"bibliothek/pdf"
	"bibliothek/pkg/mitteltopf"
	"bibliothek/repository"
)

// Etikett ist ein Buchetikett auf der Seite der Tür: der Eintrag eines Druckauftrags
// aus dem Browser und die Zeile eines Exemplars aus der Datenbank. Die Erzeuger in pdf/
// bekommen daraus pdf.BuchEtikett (PDFEtiketten).
type Etikett struct {
	BarcodeID string
	Titel     string
	Autor     string
	ISBN      string
	// AnschaffungsJahr ist das Jahr des Zugangs (zugang_am, sonst erworben_am) und steht als
	// „Ansch.J. 2016" auf dem Etikett. Leer entfällt die Zeile.
	AnschaffungsJahr string
	// Signatur ist buecher_titel.signatur (etwa „LMF-Deutsch 5"). Leer entfällt die Zeile.
	Signatur string
	// Topf ist der Topf des Exemplars (repository.ExemplarTopfSQL) und entscheidet, welcher
	// Eigentumsvermerk auf dem Etikett steht (Kopf.vermerkFuer). Er kommt nur vom Server,
	// deshalb json:"-": Kein Druckauftrag aus dem Browser kann ihn mitbringen. Leer heißt
	// Exemplar unbekannt, dann gilt der allgemeine Vermerk.
	Topf string `json:"-"`
}

// Kopf trägt die Angaben der Schule, die auf jedem Etikett stehen. Sie kommen aus den
// Einstellungen und nicht vom einzelnen Exemplar: Das Etikett nennt die Schule, der das Buch
// gehört, und zeigt im Verlustfall den Weg zurück.
type Kopf struct {
	Schulname        string // etwa „Philipp-Reis-Schule, Friedrichsdorf"
	Eigentumsvermerk string // etwa „Eigentum des Landes Hessen"
	// EigentumsvermerkSchuelerbuecherei gilt für Exemplare aus Mitteln des Schulträgers.
	// Leer heißt kein Vermerk; eine Vorgabe gibt es dafür nicht.
	EigentumsvermerkSchuelerbuecherei string
}

// vermerkFuer wählt den Eigentumsvermerk nach dem Topf des Exemplars, weil das Eigentum dem
// Geld folgt. Eine Stelle für beide Erzeuger (Etikett ab 30 mm Höhe, großes
// Lernmittel-Etikett); niedrigere Etiketten drucken keinen Vermerk.
func (k Kopf) vermerkFuer(topf string) string {
	if topf == mitteltopf.Schultraeger {
		return k.EigentumsvermerkSchuelerbuecherei
	}
	return k.Eigentumsvermerk
}

// LadeKopf lädt Schulname und Eigentumsvermerk aus den Einstellungen. Sind sie nicht lesbar,
// bleibt der Schulname leer statt erfunden, weil ein Etikett mit der falschen Schule ein
// gefundenes Buch in die Irre führt; der Eigentumsvermerk nimmt die Vorgabe.
func LadeKopf(ctx context.Context, pool db.Pool) Kopf {
	einstellungen, err := repository.NewEinstellungsAbfragen(pool).Lade(ctx)
	if err != nil {
		log.Printf("Etiketten: Einstellungen nicht lesbar, drucke ohne Schulnamen: %v", err)
		return Kopf{Eigentumsvermerk: repository.EigentumsvermerkVorgabe}
	}
	return KopfAus(einstellungen)
}

// KopfAus baut den Kopf aus den Einstellungen, eine Stelle für Selbstdruck, Lieferanten-Link
// und Mailanhang. Ein leerer allgemeiner Vermerk nimmt die Vorgabe; der Vermerk der
// Schülerbücherei bleibt leer, weil leer dort kein Vermerk heißt.
func KopfAus(einstellungen *repository.SystemEinstellungen) Kopf {
	kopf := Kopf{
		Schulname:                         einstellungen.SchuleName,
		Eigentumsvermerk:                  repository.EigentumsvermerkVorgabe,
		EigentumsvermerkSchuelerbuecherei: einstellungen.EtikettEigentumsvermerkSchuelerbuecherei,
	}
	if einstellungen.EtikettEigentumsvermerk != "" {
		kopf.Eigentumsvermerk = einstellungen.EtikettEigentumsvermerk
	}
	return kopf
}

// PDFEtiketten füllt die Eingabe der Etiketten-Erzeuger aus den Etiketten eines Auftrags und
// dem Kopf. Der Eigentumsvermerk wird hier je Exemplar gewählt, für alle Druckwege an einer
// Stelle: Der Erzeuger kennt den Topf nicht und druckt den Vermerk, den er bekommt.
func PDFEtiketten(auftrag []Etikett, kopf Kopf) []pdf.BuchEtikett {
	etiketten := make([]pdf.BuchEtikett, 0, len(auftrag))
	for _, e := range auftrag {
		etiketten = append(etiketten, pdf.BuchEtikett{
			Schulname:        kopf.Schulname,
			BarcodeID:        e.BarcodeID,
			Titel:            e.Titel,
			Autor:            e.Autor,
			AnschaffungsJahr: e.AnschaffungsJahr,
			Signatur:         e.Signatur,
			Eigentumsvermerk: kopf.vermerkFuer(e.Topf),
		})
	}
	return etiketten
}

// etikettenAus übernimmt die Zeilen einer Abfrage in die Form eines Druckauftrags; aus ihr
// füllt PDFEtiketten die Eingabe der Erzeuger.
func etikettenAus(zeilen []repository.EtikettDaten) []Etikett {
	if zeilen == nil {
		return nil
	}
	etiketten := make([]Etikett, 0, len(zeilen))
	for _, z := range zeilen {
		etiketten = append(etiketten, Etikett(z))
	}
	return etiketten
}
