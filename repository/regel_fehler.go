package repository

import (
	"errors"
	"fmt"
)

// RegelFehler ist eine Ablehnung mit ihrem Grund. Art ist der benannte Fehler, an dem die Tür den
// Status erkennt (ErrStandortUngueltig, ErrSchlagwortRegel …), Grund die Einzelheit in Worten der
// Bibliothek, die sie hinter ihren Satz setzt („höchstens 60 Zeichen").
type RegelFehler struct {
	Art   error
	Grund string
}

func (e *RegelFehler) Error() string { return e.Art.Error() + ": " + e.Grund }

func (e *RegelFehler) Unwrap() error { return e.Art }

// GrundAus liefert den Grund eines RegelFehler in der Kette, sonst "".
func GrundAus(err error) string {
	var regel *RegelFehler
	if errors.As(err, &regel) {
		return regel.Grund
	}
	return ""
}

// MitGrund setzt den Grund eines RegelFehler hinter den Satz der Tür; ohne Grund bleibt der Satz
// allein.
func MitGrund(satz string, err error) string {
	if grund := GrundAus(err); grund != "" {
		return satz + ": " + grund
	}
	return satz
}

// regelVerletzt baut einen RegelFehler; der Grund ist ein Formattext wie bei fmt.Errorf.
func regelVerletzt(art error, format string, args ...any) error {
	return &RegelFehler{Art: art, Grund: fmt.Sprintf(format, args...)}
}
