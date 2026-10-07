package repository

import "fmt"

// SignaturPraefixBedingung liefert das SQL-Prädikat „diese Signatur liegt im Regalbereich
// des Platzhalters $n" — die EINZIGE Auslegung dessen, was eine Signatur als Bereich meint.
//
// Eine Signatur ist eine Regaladresse; eine kürzere Adresse meint einen größeren Bereich.
// "BIB Deu" trifft deshalb "BIB Deu" und "BIB Deu 5 KRÜ" — aber nicht "BIB DeuX": Die
// Grenze läuft am Leerzeichen, sonst reichte eine Adresse in die Nachbaradresse hinein.
//
// Bewusst kein LIKE: Dann müssten %, _ und \ im Signaturtext maskiert werden, und genau
// das vergisst man einmal.
//
// Diese Funktion ist geteilt, weil zwei Stellen dasselbe meinen MÜSSEN: der Inventur-Scope
// (was wird gezählt und beim Abschluss als Verlust gebucht) und die Regalansicht (was
// bekommt man angezeigt). Liefen sie auseinander, würde man etwas anderes inventarisieren,
// als die Liste zeigt — und der Unterschied fiele erst beim Fehlbestand auf.
//
// spalte ist der qualifizierte Spaltenname (z. B. "t.signatur"); platzhalter die Nummer
// des Bind-Parameters, der den Bereich trägt. Der Platzhalter kommt mehrfach vor, das
// Argument wird trotzdem nur EINMAL übergeben.
func SignaturPraefixBedingung(spalte string, platzhalter int) string {
	return fmt.Sprintf(
		"(btrim(%[1]s) = btrim($%[2]d) OR left(btrim(%[1]s), length(btrim($%[2]d)) + 1) = btrim($%[2]d) || ' ')",
		spalte, platzhalter)
}

// SignaturTrenner steht in einer zweiteiligen Signatur zwischen der Regaladresse und dem
// Kürzel des Titels darin: „LMF Deu 7 / Bie". Die Littera-Übernahme setzt die Aufschrift
// des Buchrückens damit zusammen (littera.SignaturAus), die Liste der Signaturen fasst an
// ihm zusammen. Die Leerzeichen gehören dazu: An ihnen endet die Regaladresse für
// SignaturPraefixBedingung.
const SignaturTrenner = " / "

// SQLSignaturRegaladresse liefert den Ausdruck für die Regaladresse einer Signatur: den Teil
// vor dem ersten Trenner, bei einer einteiligen Signatur sie selbst. In der Sicherung von
// 2010 trägt die erste Zeile der Aufschrift nie den Trenner, die zweite 235-mal; geteilt wird
// deshalb am ersten.
func SQLSignaturRegaladresse(spalte string) string {
	return fmt.Sprintf("btrim(split_part(btrim(%s), '%s', 1))", spalte, SignaturTrenner)
}
