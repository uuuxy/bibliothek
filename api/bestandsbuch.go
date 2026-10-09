package api

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"bibliothek/pkg/mitteltopf"
	"bibliothek/pkg/schulzeit"
)

// Die Abschnitte eines Bestandsnachweises — EIN Ort für Reihenfolge und Überschriften.
//
// Zugangs- und Abgangsbuch zeigen ihre Zeilen getrennt nach Topf: Die Finanzen von Land und
// Schulträger werden getrennt geführt (docs/mittel_konzept.md 1.1/1.2). Beide Bücher gibt es
// zweimal — auf dem Bildschirm und als Blatt.
//
// Genau daran ist das Abgangsbuch beim ersten Wurf auseinandergelaufen: Das Blatt schrieb
// „Lernmittelfreiheit (Land)" (mitteltopf.Beschriftung), die Oberfläche „Lernmittel (Land)" — zwei
// Wörter für denselben Topf auf demselben Nachweis. Seit dem 17.09.2026 baut deshalb der
// SERVER die Abschnitte samt Überschrift, und der Ausdruck und der Bildschirm zeigen dieselbe
// Liste.

// Abschnitt ist ein Topf-Block eines Bestandsnachweises.
type Abschnitt[T any] struct {
	// Topf ist der gespeicherte Wert ('land', 'schultraeger', '' = ohne Zuordnung).
	Topf string `json:"topf"`
	// Titel ist die Überschrift, wie ein Mensch sie liest.
	Titel  string `json:"titel"`
	Zeilen []T    `json:"zeilen"`
}

// abschnitteAus gruppiert Zeilen nach Topf — in der Reihenfolge, die in diesem Programm
// überall gilt (Lernmittel zuerst, dann Schülerbücherei, zuletzt ohne Zuordnung).
//
// Die beiden echten Töpfe stehen immer da, auch leer: Der Bildschirm zeigt jeden Abschnitt als
// Feld mit seiner Zahl, und die Null sagt, dass nichts kam oder ging; ein fehlendes Feld sähe
// nach einem Filterfehler aus. Der Ausdruck überspringt die leeren, weil ein Blatt zum
// Abheften keine Überschrift ohne Inhalt tragen soll.
//
// „Ohne Zuordnung" erscheint nur, wenn es solche Zeilen gibt. Beim Abgangsbuch kann es sie
// nicht geben (repository.ExemplarTopfSQL endet in der Faustregel aus dem Titel und hat immer
// eine Antwort), beim Zugangsbuch schon: Ein Exemplar ohne Eigentum am Exemplar und ohne
// Bestellung trägt keinen Beleg darüber, aus welchem Geld es bezahlt wurde — und geraten wird
// das nicht (repository.ExemplarTopfBelegtSQL).
func abschnitteAus[T any](zeilen []T, topfVon func(T) string) []Abschnitt[T] {
	nach := map[string][]T{}
	for _, z := range zeilen {
		topf := topfVon(z)
		nach[topf] = append(nach[topf], z)
	}
	aus := make([]Abschnitt[T], 0, len(mitteltopf.Reihenfolge()))
	for _, topf := range mitteltopf.Reihenfolge() {
		if topf == "" && len(nach[topf]) == 0 {
			continue
		}
		zeilenDesTopfs := nach[topf]
		if zeilenDesTopfs == nil {
			zeilenDesTopfs = []T{}
		}
		aus = append(aus, Abschnitt[T]{Topf: topf, Titel: mitteltopf.Beschriftung(topf), Zeilen: zeilenDesTopfs})
	}
	return aus
}

// bestandsbuchZeitraum liest von/bis aus der Anfrage; fehlt eines, gilt das laufende
// Schulhalbjahr (Stichtage 15.3./15.9., siehe schulzeit.Halbjahr).
//
// EINE Stelle für beide Bücher und für beide Türen jedes Buchs (Bildschirm und Blatt):
// Zwei Auslegungen von „laufendes Halbjahr" ergäben einen Ausdruck, der einen anderen
// Zeitraum abdeckt als die Liste, aus der er entstand.
func bestandsbuchZeitraum(r *http.Request) (von, bis time.Time, err error) {
	von, bis = schulzeit.Halbjahr(schulzeit.Jetzt())
	lies := func(schluessel string, ziel *time.Time) error {
		roh := r.URL.Query().Get(schluessel)
		if roh == "" {
			return nil
		}
		t, fehler := schulzeit.Kalendertag(roh)
		if fehler != nil {
			return fmt.Errorf("%s muss ein Datum sein (JJJJ-MM-TT)", schluessel)
		}
		*ziel = t
		return nil
	}
	if err = lies("von", &von); err != nil {
		return von, bis, err
	}
	if err = lies("bis", &bis); err != nil {
		return von, bis, err
	}
	if bis.Before(von) {
		return von, bis, errors.New("das Ende des Zeitraums liegt vor seinem Anfang")
	}
	return von, bis, nil
}
