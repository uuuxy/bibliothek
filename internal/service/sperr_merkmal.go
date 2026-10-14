package service

import "errors"

// Die zwei Merkmale einer Ausleihsperre. Die Tür setzt dafür den Kopf X-Sperre
// (api/theke/action.go), und daran wählt der Dialog der Theke, was er anbietet
// (OmniboxBlockAlert.svelte):
//
//   - uebergehbar: ein Hinweis des Programms (offene Forderung, Überfällig-Automatik). Wer
//     Schülerdaten ändern darf, übergeht ihn einmalig (override_block).
//   - leser: eine Sperre am Leser, von Hand oder die der Ehemaligen. Sie lässt nur die
//     Rückgabe zu; der Dialog bietet an, sie aufzuheben (PATCH …/lock).
//
// Welche Sperre welches Merkmal trägt, entscheidet pruefeAusleihSperren. Eine Sperre ohne
// Merkmal (Gerät defekt, Datensatz anonymisiert) meldet die Theke nur. Am Wortlaut der Meldung
// ließe es sich nicht entscheiden: Die Sperre wegen eines Schadens trägt keins der Wörter
// „Sperre" oder „überfällig", und den Sperrgrund sieht nicht jeder (ausleih_vorgang_test.go,
// api/theke/sperr_merkmal_test.go).
//
// Bei SperrgrundFehler gehört das Merkmal in Kern: ohneSperrgrund (api/theke/action.go) baut
// den Fehler für Aufrufer ohne view_students aus Kern neu, ein Merkmal außen herum ginge dabei
// verloren.
type uebergehbareSperre struct{ err error }

func (u *uebergehbareSperre) Error() string { return u.err.Error() }

// Unwrap hält errors.Is(err, ErrGesperrt) am Leben, damit der HTTP-Status 403 bleibt.
func (u *uebergehbareSperre) Unwrap() error { return u.err }

// UebergehbareSperre markiert err als Sperre, die override_block aufheben darf.
func UebergehbareSperre(err error) error { return &uebergehbareSperre{err: err} }

// IstUebergehbareSperre sagt, ob irgendwo in der Kette von err eine solche Sperre steckt.
func IstUebergehbareSperre(err error) bool {
	var u *uebergehbareSperre
	return errors.As(err, &u)
}

// sperreAmLeser ist das Merkmal einer Sperre, die an der Person hängt. override_block
// übergeht sie nicht; aufgehoben wird sie in der Akte.
type sperreAmLeser struct{ err error }

func (s *sperreAmLeser) Error() string { return s.err.Error() }

// Unwrap hält errors.Is(err, ErrGesperrt) am Leben, damit der HTTP-Status 403 bleibt.
func (s *sperreAmLeser) Unwrap() error { return s.err }

// SperreAmLeser markiert err als Sperre am Leser.
func SperreAmLeser(err error) error { return &sperreAmLeser{err: err} }

// IstSperreAmLeser sagt, ob irgendwo in der Kette von err eine Sperre am Leser steckt.
func IstSperreAmLeser(err error) bool {
	var s *sperreAmLeser
	return errors.As(err, &s)
}

// SperrgrundFehler trennt den Freitext einer Sperre (Spalte block_reason) vom allgemeinen Teil
// der Meldung. Der Freitext kann Zahlungsrückstände oder Familiäres nennen und darf Aufrufer
// ohne view_students nicht erreichen; die Tür entscheidet an diesem Typ (errors.As), ob sie ihn
// mitschickt. Error liefert die volle Meldung für berechtigte Aufrufer und das Log.
type SperrgrundFehler struct {
	Kern  error  // Kette um ErrGesperrt mit dem allgemeinen Text ("…: Manuelle Sperre")
	Grund string // Freitext aus der Spalte block_reason
}

func (e *SperrgrundFehler) Error() string { return e.Kern.Error() + ": " + e.Grund }

// Unwrap hält errors.Is(err, ErrGesperrt) am Leben, damit der HTTP-Status 403 bleibt.
func (e *SperrgrundFehler) Unwrap() error { return e.Kern }
