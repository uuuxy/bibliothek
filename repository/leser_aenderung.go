package repository

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// LeserAenderung nennt, was an einer Leserzeile geändert wird. Ein Feld ohne Wert (nil) ist
// nicht genannt und bleibt, wie es ist. Welche Spalte ein Feld setzt, steht in zuweisungen;
// die Tür reicht nur Werte.
type LeserAenderung struct {
	Vorname  *string
	Nachname *string
	// Ausweisnummer, Anschrift und Elternkontakt lassen sich leeren: Ein leerer Wert wird NULL.
	Ausweisnummer *string
	Klasse        *string
	AbgaengerJahr *int
	Geburtsdatum  *time.Time
	Strasse       *string
	Hausnummer    *string
	Plz           *string
	Ort           *string
	ElternEmail   *string
	// LusdID und Art haben eigene Regeln (nur nachtragbar, kein Wechsel über die Grenze zum
	// Schüler). Die Tür prüft sie, bevor sie eines der beiden Felder nennt.
	LusdID *string
	Art    *string
}

// spaltenWerte sammelt die Zuweisungen einer Anweisung, deren SET-Liste von den genannten
// Feldern abhängt.
type spaltenWerte struct {
	spalten []string
	werte   []any
}

func (z *spaltenWerte) setze(spalte string, wert any) {
	z.spalten = append(z.spalten, spalte)
	z.werte = append(z.werte, wert)
}

func (z *spaltenWerte) text(spalte string, wert *string) {
	if wert != nil {
		z.setze(spalte, *wert)
	}
}

// leerbar schreibt einen leeren Wert als NULL, wie der DSGVO-Lauf und die LUSD-Ausleitung:
// Sonst stünde für „gelöscht" je nach Weg NULL oder ein leerer Text in der Spalte, und bei
// der Ausweisnummer stieße der zweite leere Text an die Eindeutigkeit.
func (z *spaltenWerte) leerbar(spalte string, wert *string) {
	if wert == nil {
		return
	}
	if strings.TrimSpace(*wert) == "" {
		z.setze(spalte, nil)
		return
	}
	z.setze(spalte, *wert)
}

// zuweisungen nennt je genanntem Feld die Spalte und ihren Wert, in der Reihenfolge der
// Anweisung.
func (a LeserAenderung) zuweisungen() spaltenWerte {
	var z spaltenWerte
	z.text("vorname", a.Vorname)
	z.text("nachname", a.Nachname)
	z.leerbar("barcode_id", a.Ausweisnummer)
	z.text("klasse", a.Klasse)
	if a.AbgaengerJahr != nil {
		z.setze("abgaenger_jahr", *a.AbgaengerJahr)
	}
	if a.Geburtsdatum != nil {
		z.setze("geburtsdatum", a.Geburtsdatum)
	}
	z.leerbar("strasse", a.Strasse)
	z.leerbar("hausnummer", a.Hausnummer)
	z.leerbar("plz", a.Plz)
	z.leerbar("ort", a.Ort)
	z.leerbar("eltern_email", a.ElternEmail)
	z.text("lusd_id", a.LusdID)
	z.text("art", a.Art)
	return z
}

// Spalten nennt die Spalten, die die Änderung setzt.
func (a LeserAenderung) Spalten() []string {
	return a.zuweisungen().spalten
}

// Leer sagt, ob die Änderung kein Feld nennt.
func (a LeserAenderung) Leer() bool {
	return len(a.Spalten()) == 0
}

// anweisung setzt das UPDATE aus den genannten Feldern zusammen; die Kennung ist der letzte
// Parameter. Die Spaltennamen stammen aus zuweisungen, nie aus der Anfrage.
func (a LeserAenderung) anweisung(id string) (string, []any) {
	z := a.zuweisungen()
	query := "UPDATE leser SET aktualisiert_am = CURRENT_TIMESTAMP"
	args := make([]any, 0, len(z.werte)+1)
	for i, spalte := range z.spalten {
		query += fmt.Sprintf(", %s = $%d", spalte, i+1)
		args = append(args, z.werte[i])
	}
	query += fmt.Sprintf(" WHERE id = $%d", len(z.spalten)+1)
	args = append(args, id)
	return query, args
}

// AendereLeser schreibt die genannten Felder an die Leserzeile und sagt, ob es die Zeile
// gibt. Geschrieben wird die Tabelle leser, nicht die Sicht schueler: In der Sicht steht ein
// Kollege nicht, die Änderung träfe ihn nie. Nennt die Änderung kein Feld, setzt die
// Anweisung nur den Zeitpunkt der Änderung.
func AendereLeser(ctx context.Context, db DBQueryer, id string, a LeserAenderung) (bool, error) {
	query, args := a.anweisung(id)
	tag, err := db.Exec(ctx, query, args...)
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() > 0, nil
}
