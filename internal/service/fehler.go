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
// ErrGesperrt geht nicht hierüber: Sein Text ist ein Satz und mit Absicht der Kopf jeder
// Sperrmeldung.
func meldung(art error, format string, args ...any) error {
	return &meldungsfehler{art: art, satz: fmt.Sprintf(format, args...)}
}

type meldungsfehler struct {
	art  error
	satz string
}

func (e *meldungsfehler) Error() string { return e.art.Error() + ": " + e.satz }

func (e *meldungsfehler) Unwrap() error { return e.art }

// MeldungAus liefert den Satz, den die Theke zu einem Fehler des Dienstes zeigt: den Satz eines
// Fehlers aus meldung, auch eingewickelt, sonst den Text des Fehlers, etwa den einer Sperre.
func MeldungAus(err error) string {
	var m *meldungsfehler
	if errors.As(err, &m) {
		return m.satz
	}
	return err.Error()
}
