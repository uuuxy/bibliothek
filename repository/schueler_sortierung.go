package repository

// Die Sortierung der Leserliste (Protokoll des Medienzentrums vom 16.09.2026, Punkt 5:
// „Sortier- ODER Filteroption"; der Filter steht seit dem 17.09.2026, das hier ist die
// andere Hälfte).
//
// Eine geschlossene Menge und KEIN Text aus der Anfrage: Der Wert landet in einem
// ORDER BY. Ein durchgereichter String wäre eine SQL-Injektion mit Anlauf, und eine
// Prüfung „enthält keine Anführungszeichen" wäre die Art Schutz, die man einmal
// vergisst.

// SchuelerSortierspalte ist die Spalte, nach der der Benutzer sortiert hat.
type SchuelerSortierspalte string

const (
	// SortiereKartei ist die Vorgabe: die gewohnte Reihenfolge der Kartei (Klasse,
	// Nachname, Vorname) bzw. bei den Ehemaligen der jüngste Abgang zuerst.
	SortiereKartei SchuelerSortierspalte = ""
	// SortiereName sortiert nach Nachname, dann Vorname.
	SortiereName SchuelerSortierspalte = "name"
	// SortiereKlasse sortiert nach Klasse, innerhalb der Klasse nach Namen.
	SortiereKlasse SchuelerSortierspalte = "klasse"
	// SortiereAusgeliehen sortiert nach der Zahl der geliehenen Bücher — die Spalte,
	// nach der jemand sucht, der wissen will, wer viel draußen hat.
	SortiereAusgeliehen SchuelerSortierspalte = "ausgeliehen"
)

// SchuelerSortierspalten sind die erlaubten Werte, für die Prüfung an der API-Tür.
// Eine Liste und keine 15 if-Zweige: Sie steht auch in der Fehlermeldung.
var SchuelerSortierspalten = []SchuelerSortierspalte{
	SortiereName, SortiereKlasse, SortiereAusgeliehen,
}

// SchuelerSortierung ist Spalte und Richtung zusammen — ein Wert, weil die Richtung
// ohne Spalte nichts bedeutet.
type SchuelerSortierung struct {
	Spalte     SchuelerSortierspalte
	Absteigend bool
}

// Erlaubt meldet, ob die Spalte eine der bekannten ist. Die leere Spalte (Vorgabe)
// gehört dazu.
func (s SchuelerSortierung) Erlaubt() bool {
	if s.Spalte == SortiereKartei {
		return true
	}
	for _, erlaubt := range SchuelerSortierspalten {
		if s.Spalte == erlaubt {
			return true
		}
	}
	return false
}

// sqlOrderBy baut die ORDER-BY-Liste.
//
// Die GRUPPE steht in der Leserdatei immer vorn, auch wenn der Benutzer sortiert: Die
// ungefilterte Liste ist bei 500 Zeilen gekappt, und ohne diesen Schlüssel fiele ein
// Kollege bei 875 Schülern lautlos hinten heraus — dieselbe Falle, gegen die die
// Reihenfolge ursprünglich gebaut wurde. Sortiert wird also INNERHALB der Gruppen.
//
// Der zweite Schlüssel ist immer der Name: Ohne ihn stünden bei gleicher Klasse (oder
// gleicher Ausleihzahl) die Zeilen in der Reihenfolge, die Postgres gerade liefert, und
// die Liste sähe bei jedem Laden anders aus.
func (s SchuelerSortierung) sqlOrderBy(art listenArt, suchRang string) string {
	gruppe := ""
	if art == listeAlleLeser {
		gruppe = "(s.art = 'schueler'), "
	}

	richtung := " ASC"
	if s.Absteigend {
		richtung = " DESC"
	}

	switch s.Spalte {
	case SortiereName:
		return gruppe + "s.nachname" + richtung + ", s.vorname" + richtung
	case SortiereKlasse:
		return gruppe + "s.klasse" + richtung + ", s.nachname, s.vorname"
	case SortiereAusgeliehen:
		return gruppe + "ausgeliehen_anzahl" + richtung + ", s.nachname, s.vorname"
	}

	// Vorgabe. Bei einer Suche zuerst die besten Treffer — aber nur, solange der
	// Benutzer nicht selbst eine Spalte gewählt hat: Wer auf „Name" klickt, will nach
	// Namen sortiert werden und nicht nach Trefferqualität.
	//
	// Die Gruppe steht auch hier vorn. Bis zum 17.09.2026 fiel sie bei einer Suche weg,
	// weil die Kappung dort nicht greift — aber „Kollegium oben" ist eine Regel, die der
	// Benutzer SIEHT, und eine Regel, die je nach Suchfeld gilt oder nicht, ist keine.
	if suchRang != "" {
		return gruppe + suchRang + ", s.nachname ASC, s.vorname ASC"
	}
	if art == listeEhemalige {
		return "s.abgaenger_jahr DESC, s.nachname, s.vorname"
	}
	return gruppe + "s.klasse, s.nachname, s.vorname"
}
