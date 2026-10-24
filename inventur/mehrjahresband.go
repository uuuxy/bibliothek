package inventur

import "errors"

// Sätze zu den beiden Regeln für die Oberfläche; die Fehlerwerte oben stehen im Log.
const (
	meldungJahrgangsSpanne = "Beim Jahrgang gehören „von“ und „bis“ zusammen oder bleiben beide leer. Beide liegen " +
		"zwischen 1 und 13, „von“ nicht über „bis“."
	meldungMehrjahresband = "Mehrjahresband nur bei einem Lernmittel und nur über mehr als einen Jahrgang, mit „bis“ " +
		"über „von“ und beiden zwischen 1 und 13"
)

// meldungZurSpanne nennt den Satz zu einem Verstoß gegen die Spanne oder den Mehrjahresband;
// ok ist false für jeden anderen Fehler.
func meldungZurSpanne(err error) (meldung string, ok bool) {
	switch {
	case errors.Is(err, errJahrgangsSpanne):
		return meldungJahrgangsSpanne, true
	case errors.Is(err, errMehrjahresband):
		return meldungMehrjahresband, true
	}
	return "", false
}
