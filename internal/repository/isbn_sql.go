package repository

// sqlTitelTraegtISBN ist die Bedingung „der Titel trägt diese ISBN", in jeder Schreibweise und
// in beiden Längen: Verglichen wird die Normalform (isbn_normalform; in Go
// isbnutil.Normalform). wert ist der SQL-Ausdruck der gefragten Nummer.
//
// Auch die Spalte geht durch die Funktion: Eine Zeile, deren Normalform schon ein anderer Titel
// trägt (eine Dublette), führt ihre ISBN noch in der alten Schreibweise.
//
// titelAlias ist der Alias der Titeltabelle in der umgebenden Abfrage, leer für keinen.
func sqlTitelTraegtISBN(titelAlias, wert string) string {
	return `isbn_normalform(` + spalteISBN(titelAlias) + `) = isbn_normalform(` + wert + `)`
}

func spalteISBN(titelAlias string) string {
	if titelAlias == "" {
		return "isbn"
	}
	return titelAlias + ".isbn"
}
