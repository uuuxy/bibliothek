package service

import "errors"

// Die zwei Merkmale einer Ausleihsperre. Die API setzt dafür den Header X-Sperre
// (api/theke/action.go), und daran wählt der Dialog der Theke, was er anbietet
// (OmniboxBlockAlert.svelte):
//
//   - uebergehbar: ein Hinweis des Programms — offene Forderung, Überfällig-Automatik. Wer
//     Schülerdaten ändern darf, übergeht ihn einmalig (override_block).
//   - leser: eine Sperre am Leser — von Hand oder die der Ehemaligen. Sie lässt nur die
//     Rückgabe zu; der Dialog bietet an, sie aufzuheben (PATCH …/lock).
//
// Welche Sperre welches Merkmal trägt, entscheidet pruefeAusleihSperren. Eine Sperre ohne
// Merkmal (Gerät defekt, Datensatz anonymisiert) meldet die Theke nur.
//
// Bis zum 13.09.2026 entschied das Frontend am Wortlaut der Meldung („Sperre",
// „Sperr-Automatik", „überfällig"). Die Schadens-Sperre traf keins der Wörter, bei der
// System-Sperre hing es am Sperrgrund, den eine Helferin gar nicht sieht
// (ausleih_vorgang_test.go, api/theke/sperr_merkmal_test.go).
//
// Bei SperrgrundFehler gehört das Merkmal in Kern: api.ohneSperrgrund baut den Fehler für
// Aufrufer ohne view_students aus Kern neu, ein Merkmal außen herum ginge dabei verloren.
type uebergehbareSperre struct{ err error }

func (u *uebergehbareSperre) Error() string { return u.err.Error() }

// Unwrap hält errors.Is(err, ErrGesperrt) am Leben — der HTTP-Status bleibt 403.
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

// Unwrap hält errors.Is(err, ErrGesperrt) am Leben — der HTTP-Status bleibt 403.
func (s *sperreAmLeser) Unwrap() error { return s.err }

// SperreAmLeser markiert err als Sperre am Leser.
func SperreAmLeser(err error) error { return &sperreAmLeser{err: err} }

// IstSperreAmLeser sagt, ob irgendwo in der Kette von err eine Sperre am Leser steckt.
func IstSperreAmLeser(err error) bool {
	var s *sperreAmLeser
	return errors.As(err, &s)
}

// SperrgrundFehler trennt den Sperr-Freitext (schueler.block_reason) vom
// generischen Teil der Meldung. Der Freitext ist Verwaltungsinformation —
// er kann Zahlungsrückstände oder Familieninterna nennen (PII-Matrix Stufe 2)
// und darf Aufrufer ohne view_students nicht erreichen. Die HTTP-Schicht
// entscheidet anhand dieses Typs (errors.As), ob sie den Grund mitschickt;
// Error() liefert weiterhin die volle Meldung für berechtigte Aufrufer & Logs.
type SperrgrundFehler struct {
	Kern  error  // ErrGesperrt-Kette mit generischem Text ("…: Manuelle Sperre")
	Grund string // Freitext aus schueler.block_reason
}

func (e *SperrgrundFehler) Error() string { return e.Kern.Error() + ": " + e.Grund }

// Unwrap hält errors.Is(err, ErrGesperrt) am Leben — der HTTP-Status bleibt 403.
func (e *SperrgrundFehler) Unwrap() error { return e.Kern }
