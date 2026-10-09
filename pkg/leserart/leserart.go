// Package leserart bündelt das Wissen über die Arten eines Lesers: welche es gibt, wie sie
// heißen und was an ihnen hängt.
//
// Die Werte sind dieselben wie in chk_leser_art in der Datenbank und in
// frontend/src/lib/leserArt.js. Server und Browser lesen dieselben Prüffälle
// (frontend/src/lib/leserArt.faelle.json); api/leser_art_pg_test.go hält sie zusätzlich gegen
// die Datenbank.
//
// Entschieden wird an der Art zweierlei: Schüler oder nicht (IstSchueler, IstKollegium) und,
// im Kollegium, ob ein Zugang zu „Mein Portal" dazugehört (MitKonto). Alles andere ist
// Bezeichnung. Das Paket steht unter pkg/, weil Türen, Fachlogik und Abfragen dieselbe
// Antwort brauchen und repository/ und internal/service/ api/ nicht einbinden können.
package leserart

import "strings"

// Schueler ist die Art, an der die Schüler-Pflichten hängen, und die Vorgabe der Spalte
// leser.art.
const Schueler = "schueler"

// arten nennt jede Art mit ihrem Wort, in der Reihenfolge der Auswahl „Art des Lesers".
// Praktikum, Sekretariat, U-plus und Fachbereich sind die Sonderkonten aus Littera.
var arten = []struct{ art, wort string }{
	{Schueler, "Schüler"},
	{"lehrkraft", "Lehrkraft"},
	{"liv", "LiV"},
	{"praktikum", "Praktikum"},
	{"sekretariat", "Sekretariat"},
	{"uplus", "U-plus"},
	{"fachbereich", "Fachbereich"},
}

// Bekannt sagt, ob die Art zum Vokabular gehört. Eine unbekannte Art ist ein Tippfehler und
// kein neuer Personenkreis: Die Datenbank wiese sie ab, aber als Datenbankfehler statt mit
// einer Auskunft.
func Bekannt(art string) bool {
	for _, a := range arten {
		if a.art == art {
			return true
		}
	}
	return false
}

// IstSchueler sagt, ob für diese Art die Schüler-Pflichten gelten. Eine leere Art ist hier
// kein Schüler; wo die Art fehlen kann, fragt man IstKollegium.
func IstSchueler(art string) bool { return art == Schueler }

// IstKollegium sagt, ob die Art keine Schüler-Art ist. Leer heißt Schüler: Eine Abfrage über
// die Sicht schueler füllt die Art nicht (repository.Student.Art), und
// frontend/src/lib/leserArt.js antwortet genauso.
func IstKollegium(art string) bool { return art != "" && art != Schueler }

// MitKonto sagt, ob zu dieser Art ein Zugang zu „Mein Portal" gehört und damit die
// Schul-E-Mail, aus der er entsteht: Lehrkraft, LiV, Sekretariat und U-plus.
//
// Praktikum und Fachbereich nicht: Ein Fachbereich ist keine Person, sondern ein Sammelkonto,
// das die Kollegen des Fachs benutzen; ein Praktikant leiht aus, braucht aber keinen Zugang.
// Für sie legt das Programm kein Konto an, weder beim Anlegen noch beim Nachtragen in der
// Akte, noch in der Littera-Übernahme. Ein Konto, das schon besteht (etwa nach dem
// Zusammenführen mit einer Selbstanmeldung), bleibt stehen; über den Zugang entscheidet dann
// die Benutzerverwaltung. Ein Schüler hat nie ein Konto.
func MitKonto(art string) bool {
	switch art {
	case "lehrkraft", "liv", "sekretariat", "uplus":
		return true
	}
	return false
}

// Bezeichnung ist das Wort zur Art, wie Leserdatei und Theke es zeigen. Eine unbekannte Art
// bleibt stehen, wie sie gespeichert ist, statt still zu verschwinden.
func Bezeichnung(art string) string {
	for _, a := range arten {
		if a.art == art {
			return a.wort
		}
	}
	return art
}

// KlasseOderArt ist, was in einer Liste in der Spalte „Klasse" steht: beim Schüler die
// Klasse, bei jedem anderen das Wort seiner Art. Theke, Ausleiher-Liste und Ausleihgeschichte
// eines Buchs lesen es hier.
func KlasseOderArt(klasse, art string) string {
	if IstKollegium(art) {
		return Bezeichnung(art)
	}
	return klasse
}

// Moegliche zählt die Arten für eine Fehlermeldung auf: „Schüler, Lehrkraft, …".
func Moegliche() string {
	woerter := make([]string, 0, len(arten))
	for _, a := range arten {
		woerter = append(woerter, a.wort)
	}
	return strings.Join(woerter, ", ")
}
