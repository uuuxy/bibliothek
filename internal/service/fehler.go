package service

import (
	"errors"
	"fmt"
)

var (
	// ErrNichtGefunden wird zurückgegeben, wenn ein angeforderter Datensatz (z. B. Schüler oder Buch) nicht existiert.
	ErrNichtGefunden = errors.New("eintrag nicht gefunden")
	// ErrGesperrt wird zurückgegeben, wenn eine Ausleihe wegen einer Sperre verweigert wird.
	//
	// Der Text nennt bewusst KEINE Person: Er ist der Kopf jeder Sperrmeldung, und das
	// Gerät ist der Gegenbeweis. Bis zum 17.09.2026 stand hier „ausleihe für diese/n
	// Schüler/in ist gesperrt"; beim gesperrten GERÄT las die Theke damit „ausleihe für
	// diese/n Schüler/in ist gesperrt: Gerät ist aktuell gesperrt" — die Bibliothekskraft
	// sucht dann nach einem Kind, das es gar nicht gibt. Wer betroffen ist, sagt der
	// angehängte Grund; wer an der Theke steht, steht auf dem Bildschirm.
	ErrGesperrt = errors.New("die ausleihe ist gesperrt")
	// ErrKonflikt wird zurückgegeben, wenn eine Aktion mit bestehenden Reservierungen oder Sperren kollidiert.
	ErrKonflikt = errors.New("conflict")
	// ErrUngueltigerZustand wird zurückgegeben, wenn sich ein Objekt oder eine Transaktion in einem ungültigen Zustand befindet.
	ErrUngueltigerZustand = errors.New("ungültiger Transaktionszustand")
)

// meldung liefert einen Fehler der Art art (ErrNichtGefunden, ErrKonflikt, ErrUngueltigerZustand), dessen
// Text allein die Meldung für die Theke ist. Die Texte dieser drei Arten sind interne
// Bezeichnungen: api/theke/action.go zeigt err.Error(), und mit fmt.Errorf("%w: …") stand bis zum
// 28.09.2026 davor „ungültiger Transaktionszustand:", „conflict:" oder „eintrag nicht gefunden:"
// (Generalprobe, OFFEN 5.33). errors.Is erkennt die Art weiter; an ihr hängt der HTTP-Status.
//
// ErrGesperrt bleibt bei fmt.Errorf: „die ausleihe ist gesperrt" ist ein Satz, und er ist mit
// Absicht der Kopf jeder Sperrmeldung.
func meldung(art error, format string, args ...any) error {
	return &meldungsfehler{art: art, text: fmt.Sprintf(format, args...)}
}

type meldungsfehler struct {
	art  error
	text string
}

func (e *meldungsfehler) Error() string { return e.text }

func (e *meldungsfehler) Unwrap() error { return e.art }
