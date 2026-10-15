package benutzer

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"bibliothek/apierrors"
	"bibliothek/repository"
)

// Feldprüfungen der Benutzerverwaltung: Rolle normalisieren, E-Mail und Ausweisnummer auf
// Eindeutigkeit prüfen. Getrennt von den Türen (konto_anlegen_aendern.go) und vom Schutz der
// Admin-Konten (admin_schutz.go), weil es drei verschiedene Fragen sind: „ist die Eingabe
// wohlgeformt", „wer ruft auf", „was tut die Tür".

// normalisiereRolle bildet die Rolle der Eingabe auf einen Wert ab, den die Datenbank kennt;
// eine unbekannte Rolle wird "mitarbeiter".
//
// Die Rückfallebene ist die harmloseste Rolle, nicht die nächstliegende: Ein Tippfehler darf
// nie nach oben führen. Wer "admin" schreiben will, muss es genau treffen, und vergeben darf
// es zusätzlich nur, wer pruefeAdminVergabe passiert.
func normalisiereRolle(rolle string) string {
	klein := strings.ToLower(strings.TrimSpace(rolle))
	switch klein {
	case "admin", "leitung", "kollegium", "mitarbeiter", "helfer":
		return klein
	default:
		return "mitarbeiter"
	}
}

// pruefeEmailEindeutig prüft, dass kein anderes Konto die E-Mail trägt (ausgenommeneID leer
// bei einem neuen Konto, sonst die eigene Kennung). Bei einem Konflikt oder Fehler der
// Datenbank ist die Antwort schon geschrieben, und es liefert false.
//
// Die Eindeutigkeit ist hier mehr als Datenhygiene: Die Anmeldung findet das Konto
// über seine E-Mail (auth/anmelden.go). Zwei Datensätze mit derselben Adresse hießen,
// dass nicht mehr feststeht, wessen Konto ein Login öffnet.
func pruefeEmailEindeutig(ctx context.Context, w http.ResponseWriter, kontoAbfragen repository.KontoAbfragen, email, ausgenommeneID string) bool {
	vergeben, err := kontoAbfragen.IstEmailVergeben(ctx, email, ausgenommeneID)
	if err != nil {
		apierrors.AntworteFehler(w, http.StatusInternalServerError, err)
		return false
	}
	if vergeben {
		apierrors.AntworteFehlerMitMeldung(w, http.StatusBadRequest, "Ein Konto mit dieser E-Mail-Adresse gibt es bereits", nil)
		return false
	}
	return true
}

// pruefeKeineLeserzeileOhneKonto bremst, wenn ein Kollege gleichen Namens schon eine
// Leserzeile ohne Konto hat: nach dem Löschen seines Kontos oder aus der
// Littera-Übernahme. Der Wächter trg_benutzer_hat_leserzeile hängt jedem neuen Konto eine
// frische Leserzeile an — dieselbe Person stünde dann zweimal in der Leserdatei, einmal mit
// Ausweis und Geschichte, einmal leer. Der Weg zum Konto an der vorhandenen Zeile ist die
// Schul-E-Mail in der Akte (LegeKollegiumskontoAn); dorthin verweist die Antwort (409).
//
// Was zählt, entscheidet KontoAbfragen.HatLeserzeileOhneKonto.
func pruefeKeineLeserzeileOhneKonto(ctx context.Context, w http.ResponseWriter, kontoAbfragen repository.KontoAbfragen, vorname, nachname string) bool {
	vorhanden, err := kontoAbfragen.HatLeserzeileOhneKonto(ctx, vorname, nachname)
	if err != nil {
		apierrors.AntworteFehler(w, http.StatusInternalServerError, err)
		return false
	}
	if vorhanden {
		apierrors.AntworteFehlerMitMeldung(w, http.StatusConflict, fmt.Sprintf("Für %s %s steht schon ein Eintrag ohne Konto in der Leserdatei. Das Konto entsteht über die Schul-E-Mail in der Akte, sonst gäbe es dieselbe Person zweimal.", vorname, nachname), nil)
		return false
	}
	return true
}

// ausweisPruefung bündelt, was die Prüfung der Ausweisnummer braucht: die Nummer, das Konto,
// das sie schon tragen darf, und die Meldung bei einem Konflikt.
type ausweisPruefung struct {
	nummer          string
	ausgenommeneID  string
	konfliktmeldung string
}

// pruefeAusweisnummerEindeutig liefert die Ausweisnummer als Zeiger, nil ohne Nummer, und prüft,
// dass kein anderes Konto sie trägt. Bei einem Konflikt oder Fehler der Datenbank ist die
// Antwort schon geschrieben (ok=false).
func pruefeAusweisnummerEindeutig(ctx context.Context, w http.ResponseWriter, kontoAbfragen repository.KontoAbfragen, pruefung ausweisPruefung) (nummer *string, ok bool) {
	if pruefung.nummer == "" {
		return nil, true
	}
	vergeben, err := kontoAbfragen.IstAusweisnummerVergeben(ctx, pruefung.nummer, pruefung.ausgenommeneID)
	if err != nil {
		apierrors.AntworteFehler(w, http.StatusInternalServerError, err)
		return nil, false
	}
	if vergeben {
		apierrors.AntworteFehler(w, http.StatusBadRequest, errors.New(pruefung.konfliktmeldung))
		return nil, false
	}
	return &pruefung.nummer, true
}

// meldeAusweisKollision übersetzt die Ablehnung der Datenbank beim Eintragen einer
// Ausweisnummer in eine Auskunft (409) und sagt, ob sie geschrieben wurde. Zwei Wächter greifen
// hinter der Vorprüfung (pruefeAusweisnummerEindeutig, sie kennt nur die Leserzeilen):
// dieselbe Nummer bei einer anderen Person (zwei Arbeitsplätze gleichzeitig) und die Nummer
// eines Buchs. Wie die drei Türen für Schüler übersetzt sie beides in eine Auskunft statt in
// eine 500.
func meldeAusweisKollision(w http.ResponseWriter, err error) bool {
	switch {
	case repository.IstNummerBuchOderAusweisKollision(err):
		apierrors.AntworteFehlerMitMeldung(w, http.StatusConflict, "Diese Nummer ist der Barcode eines Buchs und kann kein Ausweis sein", nil)
		return true
	case repository.IstAusweisKollision(err):
		apierrors.AntworteFehlerMitMeldung(w, http.StatusConflict, "Diese Ausweisnummer trägt bereits eine andere Person (Schüler oder Kollegium)", nil)
		return true
	}
	return false
}
