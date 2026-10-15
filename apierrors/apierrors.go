// Package apierrors schreibt die Fehlerantworten der API in einer Form und ordnet einem Fehler
// seinen HTTP-Status zu. Ein 500 trägt nie den Text des Fehlers; der steht im Protokoll des
// Servers.
package apierrors

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"
)

// Fehler ist die Antwort einer Tür, die scheitert: Status und Meldung gehen an die Oberfläche,
// die Ursache bleibt im Server.
type Fehler struct {
	Status  int    `json:"-"`
	Meldung string `json:"error"`
	Ursache error  `json:"-"`
}

// Error nennt Meldung und Ursache, wie sie das Log des Servers schreibt.
func (e *Fehler) Error() string {
	if e.Ursache != nil {
		return e.Meldung + ": " + e.Ursache.Error()
	}
	return e.Meldung
}

// Unwrap gibt die Ursache frei, damit errors.Is und errors.As sie finden.
func (e *Fehler) Unwrap() error {
	return e.Ursache
}

// New baut einen Fehler aus Status, Meldung und Ursache.
func New(status int, meldung string, ursache error) *Fehler {
	return &Fehler{
		Status:  status,
		Meldung: meldung,
		Ursache: ursache,
	}
}

// NotFound baut einen 404; ohne eigene Meldung gilt die Vorgabe. Die Ursache bleibt am Fehler
// und geht nicht an die Oberfläche.
func NotFound(meldung string, ursache error) *Fehler {
	if meldung == "" {
		meldung = "Ressource nicht gefunden"
	}
	return New(http.StatusNotFound, meldung, ursache)
}

// BadRequest baut einen 400 für einen Bedienfehler, den die Person selbst beheben kann.
func BadRequest(meldung string, ursache error) *Fehler {
	if meldung == "" {
		meldung = "Ungültige Anfrage"
	}
	return New(http.StatusBadRequest, meldung, ursache)
}

// Internal baut einen 500. Über MitFehlerantwort geht die Meldung so hinaus, wie sie hier steht,
// die Ursache nur ins Log; anders als AntworteFehler ersetzt dieser Weg die Meldung eines 500
// nicht durch eine neutrale. Die Meldung nennt deshalb nichts aus dem Server.
func Internal(meldung string, ursache error) *Fehler {
	if meldung == "" {
		meldung = "Ein interner Serverfehler ist aufgetreten"
	}
	return New(http.StatusInternalServerError, meldung, ursache)
}

// Unauthorized baut einen 401 für eine fehlende oder abgelaufene Anmeldung. „Angemeldet, aber
// ohne Recht" ist ein 403, den die Prüfung der Rechte selbst schreibt.
func Unauthorized(meldung string, ursache error) *Fehler {
	if meldung == "" {
		meldung = "Nicht autorisiert"
	}
	return New(http.StatusUnauthorized, meldung, ursache)
}

// Conflict baut einen 409: Die Anfrage war gültig, aber der Stand hat sich inzwischen geändert,
// etwa weil das Exemplar gerade ausgeliehen wurde. Die Oberfläche lädt neu, statt zu
// wiederholen.
func Conflict(meldung string, ursache error) *Fehler {
	if meldung == "" {
		meldung = "Konflikt mit dem aktuellen Stand"
	}
	return New(http.StatusConflict, meldung, ursache)
}

// TuerMitFehler ist eine Tür, die ihren Fehler zurückgibt, statt die Antwort selbst zu
// schreiben.
type TuerMitFehler func(w http.ResponseWriter, r *http.Request) error

// statusAbgebrochen ist der Status 499 aus nginx: Der Browser hat die Verbindung aufgegeben,
// bevor die Antwort fertig war. Das ist kein Fehler des Servers; er steht nicht auf der
// Fehlerstufe im Log und zählt in keinem Alarm.
const statusAbgebrochen = 499

// istAbbruch erkennt einen Fehler aus einem abgebrochenen Kontext der Anfrage; pgx und
// net/http reichen context.Canceled eingewickelt weiter, errors.Is findet es.
func istAbbruch(err error) bool {
	return err != nil && errors.Is(err, context.Canceled)
}

func antworteAbbruch(w http.ResponseWriter) {
	schreibeFehlerJSON(w, statusAbgebrochen, "Anfrage vom Client abgebrochen")
}

// MitFehlerantwort macht aus einer TuerMitFehler eine gewöhnliche Tür. Ein Fehler vom Typ
// Fehler geht mit seinem Status und seiner Meldung hinaus, jeder andere als neutraler 500.
func MitFehlerantwort(tuer TuerMitFehler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		err := tuer(w, r)
		if err != nil {
			// Der Kontext der Anfrage ist die sicherste Quelle: Auch ein Fehler, der
			// context.Canceled nicht einwickelt, ist nach einem Abbruch kein Serverfehler.
			if istAbbruch(err) || errors.Is(r.Context().Err(), context.Canceled) {
				antworteAbbruch(w)
				return
			}
			var fehler *Fehler
			if errors.As(err, &fehler) {
				if fehler.Status >= 500 {
					log.Printf("API Error [HTTP %d]: %v", fehler.Status, fehler.Ursache)
				}
				schreibeFehlerJSON(w, fehler.Status, fehler.Meldung)
			} else {
				AntworteFehler(w, http.StatusInternalServerError, err)
			}
		}
	}
}

// AntworteFehler schreibt den Fehler ins Log und antwortet mit einer Meldung, die keine
// Einzelheit des Servers verrät. Unter 500 ist der Text des Fehlers die Meldung; heraus
// gefiltert wird nur, was istDatenbankFehler erkennt. Gehört die Ursache eines Fehlers nicht
// an die Oberfläche, nimmt der Rufer AntworteFehlerMitMeldung.
func AntworteFehler(w http.ResponseWriter, status int, ursache error) {
	AntworteFehlerMitMeldung(w, status, "", ursache)
}

// AntworteFehlerMitMeldung trennt, was die Oberfläche liest, von dem, was ins Log geht: Die
// Meldung geht hinaus, die Ursache ganz ins Log. Der Rufer nennt den Satz, weil eine Ursache wie
// ein Verbindungsabriss weder SQL noch eine Bedingung nennt und am Filter vorbeiginge.
//
// Eine leere Meldung nimmt den Text des Fehlers. Ohne Ursache steht die Meldung im Log, wie sonst
// die Ursache. Ein 500 bleibt auch mit Meldung neutral; eine Diagnose braucht 400, 409 oder 502.
func AntworteFehlerMitMeldung(w http.ResponseWriter, status int, meldung string, ursache error) {
	if istAbbruch(ursache) {
		antworteAbbruch(w)
		return
	}
	if zeile, ok := logZeile(status, meldung, ursache); ok {
		log.Print(zeile)
	}

	text := http.StatusText(status)
	if text == "" {
		text = "Unknown Error"
	}

	if status != http.StatusInternalServerError {
		switch {
		case meldung != "":
			text = meldung
		case ursache != nil:
			text = ursache.Error()
		}
	}

	// Geprüft wird der Text, der hinausgeht, nicht die Ursache: Ohne Meldung sind beide
	// gleich, mit Meldung kommt es auf sie an.
	ausDerDatenbank := istDatenbankFehler(strings.ToLower(text))

	if status == http.StatusInternalServerError || ausDerDatenbank {
		text = neutraleMeldung(ursache)
	}

	schreibeFehlerJSON(w, status, text)
}

// logZeile ist der Eintrag im Log zu einer Fehlerantwort: die Ursache, ohne sie die Meldung. Ohne
// beides gibt es keinen Eintrag.
func logZeile(status int, meldung string, ursache error) (string, bool) {
	switch {
	case ursache != nil:
		return fmt.Sprintf("API Error [HTTP %d]: %v (path: %s)", status, ursache, "unknown"), true
	case meldung != "":
		return fmt.Sprintf("API Error [HTTP %d]: %s (path: %s)", status, meldung, "unknown"), true
	}
	return "", false
}

// schreibeFehlerJSON legt fest, wie jede Fehlerantwort aussieht: {"error": meldung} mit Status
// und Content-Type. AntworteFehler und MitFehlerantwort schreiben beide hierüber, damit sich die
// Oberfläche auf eine Form verlassen kann.
func schreibeFehlerJSON(w http.ResponseWriter, status int, meldung string) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": meldung}) //nolint:errcheck
}

// istDatenbankFehler erkennt einen Text aus der Datenbank (SQL, Bedingungen, Meldungen des
// Treibers), der nie ungefiltert an die Oberfläche geht. Der Text ist klein geschrieben.
func istDatenbankFehler(text string) bool {
	return strings.Contains(text, "pgx") ||
		strings.Contains(text, "pgconn") ||
		strings.Contains(text, "sql:") ||
		strings.Contains(text, "unique constraint") ||
		strings.Contains(text, "foreign key") ||
		strings.Contains(text, "duplicate key") ||
		strings.Contains(text, "violates") ||
		strings.Contains(text, "insert into") ||
		strings.Contains(text, "select ") ||
		strings.Contains(text, "update ") ||
		strings.Contains(text, "delete ")
}

// neutraleMeldung liefert die Meldung für einen Fehler im Server; zwei bekannte Verletzungen
// einer Bedingung nennt sie in Worten der Bibliothek.
func neutraleMeldung(ursache error) string {
	if ursache == nil {
		return "Internal Server Error"
	}
	text := ursache.Error()
	switch {
	case strings.Contains(text, "23505") || strings.Contains(text, "unique_violation") || strings.Contains(text, "duplicate key"):
		return "Ein Eintrag mit diesen eindeutigen Eigenschaften existiert bereits."
	case strings.Contains(text, "23503") || strings.Contains(text, "foreign_key_violation"):
		return "Diese Aktion kann nicht durchgeführt werden, da verknüpfte Daten existieren."
	default:
		return "Ein interner Datenbankfehler ist aufgetreten."
	}
}
