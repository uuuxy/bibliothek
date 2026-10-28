package inventurlauf

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"

	"bibliothek/auth"
	"bibliothek/internal/service"
	"bibliothek/repository"
)

// rueckgabeBuchung bucht die Rückgabe eines Exemplars, das ohne geöffneten Leser gescannt wurde:
// im Betrieb der Dienst der Theke (service.AusleihDienst).
type rueckgabeBuchung interface {
	NimmZurueck(ctx context.Context, exemplar *repository.Exemplar, bearbeiterID string) (*service.AusleihErgebnis, error)
}

// Die Hinweise der Karte zu einem verliehenen Buch, das im Regal gescannt wurde.
const (
	hinweisVerliehen      = "Buch ist laut System aktuell ausgeliehen."
	hinweisZurueckgebucht = "Zurückgebucht: Das Buch war laut Programm noch ausgeliehen."
	hinweisNichtZurueck   = "Zurückbuchen ging nicht. Bitte das Buch an der Theke scannen."
)

// hinweiseZumVerliehenen bucht ein verliehenes Buch, das im Regal gescannt wurde, zurück und
// liefert die Hinweise für die Karte. Das Buch liegt in der Hand, laut Programm hätte es sonst
// weiter der Schüler, mit Mahnung nach der Frist. Gebucht wird über den Dienst der Theke
// (Ausleihe beenden, Vormerkung bedienen, Protokoll), und die übrigen Arbeitsplätze erfahren es
// wie nach einer Rückgabe an der Theke. Wer an der Theke nicht zurücknehmen darf
// (perform_actions), bekommt nur den Hinweis.
func (t *Tueren) hinweiseZumVerliehenen(r *http.Request, rueckgabe rueckgabeBuchung, barcodeID string) []string {
	claims, ok := auth.ClaimsAusKontext(r.Context())
	if !ok || !t.HatRecht(r, "perform_actions") {
		return []string{hinweisVerliehen}
	}
	ctx := r.Context()
	exemplar, err := repository.NewKatalogAbfragen(t.DB.Pool).FindeExemplar(ctx, barcodeID)
	if err == nil && exemplar == nil {
		err = fmt.Errorf("exemplar %s nach dem scan nicht mehr gefunden", barcodeID)
	}
	var erg *service.AusleihErgebnis
	if err == nil {
		erg, err = rueckgabe.NimmZurueck(ctx, exemplar, claims.KontoID)
	}
	if errors.Is(err, service.ErrUngueltigerZustand) {
		// Die Theke oder ein zweiter Scanner hat das Buch inzwischen zurückgebucht.
		return nil
	}
	if err != nil {
		slog.Error("Inventur: verliehenes Buch nicht zurückgebucht", "barcode", barcodeID, "fehler", err)
		return []string{hinweisVerliehen, hinweisNichtZurueck}
	}

	var leserID string
	if erg.Leser != nil {
		leserID = erg.Leser.ID
	}
	t.SendeThekenEreignis("rueckgabe", leserID)
	return append([]string{hinweisZurueckgebucht}, hinweisAbholfach(erg)...)
}

// hinweisAbholfach sagt wie die Theke, dass ein zurückgebuchtes Buch vorgemerkt ist und ins
// Abholfach gehört. Die Theke zeigt dafür einen Alarm, der das Scanfeld sperrt; in der Inventur
// trifft der nächste Scan schon ein, deshalb steht der Hinweis auf der Karte.
func hinweisAbholfach(erg *service.AusleihErgebnis) []string {
	if !erg.HatVormerkung {
		return nil
	}
	if erg.VorgemerktFuer == "" {
		return []string{"Vorgemerkt: nicht ins Regal stellen, ins Abholfach legen."}
	}
	return []string{fmt.Sprintf("Vorgemerkt für %s: nicht ins Regal stellen, ins Abholfach legen.", erg.VorgemerktFuer)}
}
