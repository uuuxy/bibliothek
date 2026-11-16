package repository

import (
	"errors"
	"fmt"
)

// regelFehler ist eine Ablehnung mit ihrem Grund. Art ist der benannte Fehler, an dem die Tür den
// Status erkennt (ErrStandortUngueltig, ErrSchlagwortRegel …), Grund die Einzelheit in Worten der
// Bibliothek, die sie hinter ihren Satz setzt („höchstens 60 Zeichen").
type regelFehler struct {
	Art   error
	Grund string
}

func (e *regelFehler) Error() string { return e.Art.Error() + ": " + e.Grund }

func (e *regelFehler) Unwrap() error { return e.Art }

// GrundAus liefert den Grund eines regelFehler in der Kette, sonst "".
func GrundAus(err error) string {
	var regel *regelFehler
	if errors.As(err, &regel) {
		return regel.Grund
	}
	return ""
}

// MitGrund setzt den Grund eines regelFehler hinter den Satz der Tür; ohne Grund bleibt der Satz
// allein.
func MitGrund(satz string, err error) string {
	if grund := GrundAus(err); grund != "" {
		return satz + ": " + grund
	}
	return satz
}

// regelVerletzt baut einen regelFehler; der Grund ist ein Formattext wie bei fmt.Errorf.
func regelVerletzt(art error, format string, args ...any) error {
	return &regelFehler{Art: art, Grund: fmt.Sprintf(format, args...)}
}
