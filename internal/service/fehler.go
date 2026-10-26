package service

import (
	"errors"
	"fmt"
)

var (
	// ErrNichtGefunden meldet, dass es den gesuchten Leser, das Exemplar oder das Gerät nicht
	// gibt (404).
	ErrNichtGefunden = errors.New("eintrag nicht gefunden")
	// ErrGesperrt meldet, dass eine Sperre die Ausleihe verweigert (403). Sein Text nennt keine
	// Person, weil er der Kopf jeder Sperrmeldung ist, auch der eines gesperrten Geräts; wer
	// betroffen ist, sagt der angehängte Grund.
	ErrGesperrt = errors.New("die ausleihe ist gesperrt")
	// ErrKonflikt meldet, dass eine Ausleihe mit einer Vormerkung oder einer gleichzeitigen
	// Buchung zusammenstößt (409).
	ErrKonflikt = errors.New("conflict")
	// ErrUngueltigerZustand meldet, dass das Exemplar, das Gerät oder die Eingabe den Vorgang
	// nicht zulässt, etwa ein ausgesondertes Exemplar oder kein geöffneter Leser.
	ErrUngueltigerZustand = errors.New("ungültiger Transaktionszustand")
)

// meldung liefert einen Fehler der Art art (ErrNichtGefunden, ErrKonflikt,
// ErrUngueltigerZustand) mit dem Satz, den die Theke zeigt. Der Satz ist ein Feld des Fehlers,
// das MeldungAus liefert; Error beginnt mit dem Text der Art, der eine interne Bezeichnung ist
// und nicht auf den Bildschirm gehört. errors.Is erkennt die Art weiter; an ihr hängt der
// HTTP-Status.
//
// ErrGesperrt geht nicht hierüber, sondern über Sperre: Sein Text ist der Kopf jeder Sperrmeldung.
func meldung(art error, format string, args ...any) error {
	return &meldungsfehler{art: art, satz: fmt.Sprintf(format, args...)}
}

type meldungsfehler struct {
	art  error
	satz string
}

func (e *meldungsfehler) Error() string { return e.art.Error() + ": " + e.satz }

func (e *meldungsfehler) Unwrap() error { return e.art }

// Sperre liefert eine Sperre mit ihrem Grund, etwa „Gerät ist aktuell gesperrt". Der Grund ist
// ein Feld: Error beginnt klein mit dem Text von ErrGesperrt, MeldungAus setzt für den Bildschirm
// meldungGesperrt davor. Ein Grund, der mit fmt.Errorf an ErrGesperrt hinge, stünde nur im
// Fehlertext (sperre_ratsche_test.go).
func Sperre(format string, args ...any) error {
	return &sperrfehler{grund: fmt.Sprintf(format, args...)}
}

type sperrfehler struct{ grund string }

func (e *sperrfehler) Error() string { return ErrGesperrt.Error() + ": " + e.grund }

// Unwrap hält errors.Is(err, ErrGesperrt) am Leben, damit der HTTP-Status 403 bleibt.
func (e *sperrfehler) Unwrap() error { return ErrGesperrt }

// meldungGesperrt ist der Kopf einer Sperrmeldung auf dem Bildschirm. Der Fehlertext von
// ErrGesperrt beginnt klein wie jeder Fehlertext, ein Satz der Oberfläche groß (STIL.md 11).
const meldungGesperrt = "Die Ausleihe ist gesperrt"

// MeldungAus liefert den Satz, den die Theke zu einem Fehler des Dienstes zeigt: den Satz eines
// Fehlers aus meldung, auch eingewickelt; bei einer Sperre den Kopf mit den Gründen aus den
// Feldern (Sperre, SperrgrundFehler, SperrgrundFehler.OhneGrund); sonst den Text des Fehlers.
func MeldungAus(err error) string {
	var m *meldungsfehler
	var ohne *sperreOhneGrund
	var mit *SperrgrundFehler
	var s *sperrfehler
	switch {
	case errors.As(err, &m):
		return m.satz
	case errors.As(err, &ohne):
		return MeldungAus(ohne.kern) + ". " + meldungBitteAnDieLeitung + "."
	case errors.As(err, &mit):
		return MeldungAus(mit.Kern) + ": " + mit.Grund
	case errors.As(err, &s):
		return meldungGesperrt + ": " + s.grund
	case errors.Is(err, ErrGesperrt):
		// Die Sperre ohne Grund; der der Ehemaligen steht im SperrgrundFehler.
		return meldungGesperrt
	}
	return err.Error()
}
