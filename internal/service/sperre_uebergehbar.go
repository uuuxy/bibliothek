package service

import "errors"

// uebergehbareSperre markiert eine Sperre, die die Theke mit override_block übergehen darf
// — die vier Sperren der Buch-Ausleihe (pruefeSchuelerAusleihbar). Die API setzt dafür den
// Header X-Sperre: uebergehbar, und daran öffnet das Frontend den Override-Dialog.
//
// Bis zum 13.09.2026 entschied das Frontend am Wortlaut der Meldung („Sperre",
// „Sperr-Automatik", „überfällig"). Die Schadens-Sperre traf keins der Wörter, bei der
// System-Sperre hing es am Sperrgrund, den eine Helferin gar nicht sieht
// (loan_checkout_validation_test.go, api/sperr_merkmal_test.go).
//
// Bei SperrGrundFehler gehört das Merkmal in Kern: api.ohneSperrgrund baut den Fehler für
// Aufrufer ohne view_students aus Kern neu, ein Merkmal außen herum ginge dabei verloren.
type uebergehbareSperre struct{ err error }

func (u *uebergehbareSperre) Error() string { return u.err.Error() }

// Unwrap hält errors.Is(err, ErrBlocked) am Leben — der HTTP-Status bleibt 403.
func (u *uebergehbareSperre) Unwrap() error { return u.err }

// UebergehbareSperre markiert err als Sperre, die override_block aufheben darf.
func UebergehbareSperre(err error) error { return &uebergehbareSperre{err: err} }

// IstUebergehbareSperre sagt, ob irgendwo in der Kette von err eine solche Sperre steckt.
func IstUebergehbareSperre(err error) bool {
	var u *uebergehbareSperre
	return errors.As(err, &u)
}
