package api

import (
	"errors"
	"net/http"
	"sync"
	"time"

	"bibliothek/apierrors"
	"bibliothek/pkg/clientip"
)

// fehlversuche stores the count of failed login attempts and the time of the first failure in a window.
type fehlversuche struct {
	anzahl int
	erster time.Time
}

// maxAnmeldeFehlversucheJeAdresse ist die VOLUMETRISCHE Obergrenze dieser Middleware —
// bewusst hoch, nicht als Konto-Schutz gedacht.
//
// Vorher standen hier 5. Das war dieselbe Zahl wie im Handler (auth/handlers.go),
// nur mit dem entscheidenden Unterschied, dass der Handler pro (E-Mail|IP) zählt
// und diese Schicht rein pro IP. In einer Schule, deren Geräte über EINE NAT-Adresse
// herauskommen, genügten damit fünf Vertipper irgendeines Nutzers, um den Login für
// ALLE anderen 15 Minuten lang zu sperren — die Härtung des Handlers auf (E-Mail|IP)
// wurde von der äußeren Schicht wieder eingesammelt.
//
// Der Kontoschutz gegen gezieltes Raten liegt jetzt allein beim (E-Mail|IP)-Limiter
// im Handler. Was hier bleibt, ist die Bremse gegen Credential-Stuffing, das sich
// über viele Konten verteilt und den IMAP-Server der Schule flutet: 50 Fehlversuche
// in 15 Minuten aus EINER Quelle erreicht kein normaler Standort, ein Skript sofort.
const maxAnmeldeFehlversucheJeAdresse = 50

var (
	fehlversucheJeAdresse = make(map[string]*fehlversuche)
	fehlversucheMu        sync.Mutex
)

// raeumeAbgelaufeneAuf entfernt abgelaufene Login-Versuche. Aufrufer muss fehlversucheMu halten.
// Wird beim Anlegen neuer Einträge ab einer Schwelle aufgerufen — das hält die Map ohne eine
// dauerhaft laufende Hintergrund-Goroutine beschränkt (verhindert Goroutine-Leaks in Tests).
func raeumeAbgelaufeneAuf(jetzt time.Time) {
	for adresse, versuche := range fehlversucheJeAdresse {
		if jetzt.Sub(versuche.erster) > 15*time.Minute {
			delete(fehlversucheJeAdresse, adresse)
		}
	}
}

// reserviereAnmeldeVersuch prüft/aktualisiert den Fehlversuchszähler einer IP (unter Lock).
// blocked=true bedeutet: die IP ist derzeit gesperrt (siehe maxAnmeldeFehlversucheJeAdresse,
// gezählt im 15-Minuten-Fenster).
func reserviereAnmeldeVersuch(adresse string, jetzt time.Time) (versuche *fehlversuche, gesperrt bool) {
	fehlversucheMu.Lock()
	defer fehlversucheMu.Unlock()

	versuche, vorhanden := fehlversucheJeAdresse[adresse]
	if vorhanden {
		// If the penalty window of 15 minutes has expired, reset the attempt record.
		if jetzt.Sub(versuche.erster) > 15*time.Minute {
			versuche.anzahl = 0
			versuche.erster = jetzt
		} else if versuche.anzahl >= maxAnmeldeFehlversucheJeAdresse {
			return versuche, true
		}
		return versuche, false
	}

	// Vor dem Einfügen ab einer Schwelle abgelaufene Einträge entfernen, um die Map
	// ohne Dauer-Goroutine beschränkt zu halten (analog zum IP-Rate-Limiter).
	if len(fehlversucheJeAdresse) > 5000 {
		raeumeAbgelaufeneAuf(jetzt)
	}
	versuche = &fehlversuche{
		anzahl: 0,
		erster: jetzt,
	}
	fehlversucheJeAdresse[adresse] = versuche
	return versuche, false
}

// verzeichneAnmeldeErgebnis aktualisiert den Fehlversuchszähler anhand des Antwortstatus
// (NUR 401 erhöht, 200 setzt bei Erfolg zurück) — unter Lock.
//
// 403 zählt seit dem 31.08.2026 NICHT mehr: Bei „Zugang beantragt" (Selbstanmeldung)
// und „Konto deaktiviert" war das Passwort richtig — der Handler zählt deshalb bewusst
// nicht (auth/handlers.go), und diese Schicht darf die Entscheidung nicht einsammeln.
// Zum Schuljahresbeginn hätten sonst zwanzig Selbstanmelde-403 hinter der EINEN
// NAT-Adresse der Schule den Login für alle gesperrt — auch für die Bibliothekskraft,
// die die Freischaltungen vornimmt. Für Credential-Stuffing (der Zweck dieser Bremse)
// ist 401 das Signal, nicht 403.
func verzeichneAnmeldeErgebnis(versuche *fehlversuche, status int, jetzt time.Time) {
	switch status {
	case http.StatusUnauthorized:
		fehlversucheMu.Lock()
		versuche.anzahl++
		fehlversucheMu.Unlock()
	case http.StatusOK:
		// Optional: reset counter on successful login
		fehlversucheMu.Lock()
		if versuche.anzahl > 0 {
			versuche.anzahl = 0
			versuche.erster = jetzt
		}
		fehlversucheMu.Unlock()
	}
}

// anmeldeBremse begrenzt fehlgeschlagene Anmeldungen pro IP auf
// maxAnmeldeFehlversucheJeAdresse innerhalb von 15 Minuten; weitere Versuche im Fenster
// beantwortet sie mit 429.
//
// Sie ist die VOLUMETRISCHE Schranke. Der Schutz eines einzelnen Kontos gegen
// gezieltes Raten sitzt im Login-Handler und zählt pro (E-Mail|IP) — siehe
// maxAnmeldeFehlversucheJeAdresse für den Grund, warum beide Schichten unterschiedliche
// Schwellen haben müssen.
//
// The IP is resolved via clientip so that requests behind the Caddy reverse
// proxy are keyed on the real client — not on the single proxy address, which
// would let a handful of failed logins lock out every user (global denial of service).
func anmeldeBremse(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		adresse := clientip.AusAnfrage(r)

		jetzt := time.Now()
		versuche, gesperrt := reserviereAnmeldeVersuch(adresse, jetzt)
		if gesperrt {
			apierrors.SendHTTPError(w, http.StatusTooManyRequests, errors.New("zu viele fehlerhafte Login-Versuche. Bitte warten Sie 15 Minuten"))
			return
		}

		// Intercept the response status
		merker := &statusMerker{
			ResponseWriter: w,
			status:         http.StatusOK, // Default to OK in case the handler doesn't call WriteHeader
		}

		next.ServeHTTP(merker, r)

		// Increment/reset failure count based on the auth result
		verzeichneAnmeldeErgebnis(versuche, merker.status, jetzt)
	})
}
