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
// ErrUngueltigerZustand), dessen Text allein die Meldung für die Theke ist: Die Theke zeigt
// err.Error(), und der Text der Art ist eine interne Bezeichnung. errors.Is erkennt die Art
// weiter; an ihr hängt der HTTP-Status.
//
// ErrGesperrt geht nicht hierüber: Sein Text ist ein Satz und mit Absicht der Kopf jeder
// Sperrmeldung.
func meldung(art error, format string, args ...any) error {
	return &meldungsfehler{art: art, text: fmt.Sprintf(format, args...)}
}

type meldungsfehler struct {
	art  error
	text string
}

func (e *meldungsfehler) Error() string { return e.text }

func (e *meldungsfehler) Unwrap() error { return e.art }
