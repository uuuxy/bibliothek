package isbnutil

import "strconv"

// pruefzeichen10 ist das Prüfzeichen einer ISBN-10 zu ihren ersten neun Ziffern.
func pruefzeichen10(neun string) string {
	summe := 0
	for i := 0; i < 9; i++ {
		summe += int(neun[i]-'0') * (10 - i)
	}
	rest := (11 - summe%11) % 11
	if rest == 10 {
		return "X"
	}
	return strconv.Itoa(rest)
}

// pruefziffer13 ist die Prüfziffer einer ISBN-13 zu ihren ersten zwölf Ziffern.
func pruefziffer13(zwoelf string) string {
	summe := 0
	for i := 0; i < 12; i++ {
		gewicht := 1
		if i%2 == 1 {
			gewicht = 3
		}
		summe += int(zwoelf[i]-'0') * gewicht
	}
	return strconv.Itoa((10 - summe%10) % 10)
}
