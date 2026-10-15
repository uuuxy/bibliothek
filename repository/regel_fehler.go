package repository

import "fmt"

// RegelFehler ist eine Ablehnung mit ihrem Grund. Art ist der benannte Fehler, an dem die Tür den
// Status erkennt (ErrStandortUngueltig, ErrSchlagwortRegel …), Grund die Einzelheit in Worten der
// Bibliothek, die sie hinter ihren Satz setzt („höchstens 60 Zeichen").
type RegelFehler struct {
	Art   error
	Grund string
}

func (e *RegelFehler) Error() string { return e.Art.Error() + ": " + e.Grund }

func (e *RegelFehler) Unwrap() error { return e.Art }

// mitGrund baut einen RegelFehler; der Grund ist ein Formattext wie bei fmt.Errorf.
func mitGrund(art error, format string, args ...any) error {
	return &RegelFehler{Art: art, Grund: fmt.Sprintf(format, args...)}
}
