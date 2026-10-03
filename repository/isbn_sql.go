package repository

// SQLTitelTraegtISBN ist die Bedingung „der Titel trägt diese ISBN", in jeder Schreibweise
// und in beiden Längen: Verglichen wird die Normalform (isbn_normalform, Migration 133 und
// 157; in Go isbnutil.Normalform). wert ist der SQL-Ausdruck der gefragten Nummer.
//
// Auch die Spalte geht durch die Funktion: Eine Zeile, die Migration 140 oder 157 neben
// einer Dublette stehen ließ, trägt ihre ISBN noch in der alten Schreibweise.
//
// titelAlias ist der Alias der Titeltabelle in der umgebenden Abfrage, leer für keinen.
func SQLTitelTraegtISBN(titelAlias, wert string) string {
	return `isbn_normalform(` + spalteISBN(titelAlias) + `) = isbn_normalform(` + wert + `)`
}

// SQLSuchtextIstISBN ist die Bedingung „der Suchtext ist die ISBN des Titels", für die
// Suchfelder: Eine getippte zehnstellige ISBN findet den Titel, den die Datenbank unter der
// dreizehnstelligen führt, und Bindestriche stören nicht. Der Teilstring-Vergleich der
// Suchen sieht das nicht — beide Längen enden auf verschiedene Prüfzeichen.
//
// Die Spalte steht hier ohne Funktion, damit der UNIQUE-Index auf isbn greift; eine Suche
// läuft bei jedem Tastendruck. suchtext ist der SQL-Ausdruck des Suchtexts.
func SQLSuchtextIstISBN(titelAlias, suchtext string) string {
	return spalteISBN(titelAlias) + ` = isbn_normalform(` + suchtext + `)`
}

func spalteISBN(titelAlias string) string {
	if titelAlias == "" {
		return "isbn"
	}
	return titelAlias + ".isbn"
}
