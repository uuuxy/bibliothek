package api

import (
	"net/http"
	"sync"
	"time"

	"bibliothek/internal/apierrors"
	"bibliothek/internal/auth"
	"bibliothek/internal/pkg/clientip"
)

// fehlversuche zählt die gescheiterten Anmeldungen einer Adresse und merkt sich den ersten
// Fehlversuch im Fenster.
type fehlversuche struct {
	anzahl int
	erster time.Time
	// laufend zählt die Anmeldungen der Adresse, deren Antwort noch aussteht.
	laufend int
}

// maxAnmeldeFehlversucheJeAdresse ist die Obergrenze dieser Middleware für die Menge, bewusst
// hoch und nicht als Schutz eines Kontos gedacht.
//
// Die Tür (internal/auth/anmelden.go) zählt je E-Mail und Adresse, diese Schicht nur je Adresse.
// Läge die Grenze hier so niedrig wie dort (fünf), genügten in einer Schule, deren Geräte über
// eine NAT-Adresse herauskommen, fünf Vertipper irgendeines Nutzers, um die Anmeldung für alle
// anderen 15 Minuten lang zu sperren.
//
// Den Schutz gegen gezieltes Raten trägt allein der Zähler im Handler. Was hier bleibt, ist die
// Bremse gegen Credential-Stuffing, das sich über viele Konten verteilt und den IMAP-Server der
// Schule flutet: 50 Fehlversuche in 15 Minuten aus einer Quelle erreicht kein normaler
// Standort, ein Skript sofort.
const maxAnmeldeFehlversucheJeAdresse = 50

// anmeldeBremse zählt je Adresse die gescheiterten Anmeldungen der letzten 15 Minuten. Ihr
// Nullwert ist leer und benutzbar; der Server trägt eine, die Anmelden und Entsperren teilen.
type anmeldeBremse struct {
	mu        sync.Mutex
	jeAdresse map[string]*fehlversuche
}

// raeumeAbgelaufeneAuf entfernt abgelaufene Fehlversuche; der Aufrufer hält b.mu. Es läuft
// beim Anlegen neuer Einträge ab einer Schwelle. So bleibt die Map ohne eine dauernd laufende
// Goroutine begrenzt, die in Tests liegen bliebe.
func (b *anmeldeBremse) raeumeAbgelaufeneAuf(jetzt time.Time) {
	for adresse, versuche := range b.jeAdresse {
		if versuche.laufend == 0 && jetzt.Sub(versuche.erster) > 15*time.Minute {
			delete(b.jeAdresse, adresse)
		}
	}
}

// reserviere prüft und erneuert den Zähler einer Adresse, unter der Sperre, und meldet die
// Anmeldung als laufend an. gesperrt heißt: Gezählte Fehlversuche im Fenster von 15 Minuten und
// laufende Anmeldungen erreichen zusammen maxAnmeldeFehlversucheJeAdresse. Die laufenden zählen
// mit, weil ein Fehlversuch erst nach der Antwort feststeht; sonst gingen beliebig viele
// gleichzeitige Versuche durch.
func (b *anmeldeBremse) reserviere(adresse string, jetzt time.Time) (versuche *fehlversuche, gesperrt bool) {
	b.mu.Lock()
	defer b.mu.Unlock()

	versuche, vorhanden := b.jeAdresse[adresse]
	if vorhanden {
		// Ist das Fenster von 15 Minuten abgelaufen, beginnt die Zählung neu.
		if jetzt.Sub(versuche.erster) > 15*time.Minute {
			versuche.anzahl = 0
			versuche.erster = jetzt
		}
		if versuche.anzahl+versuche.laufend >= maxAnmeldeFehlversucheJeAdresse {
			return versuche, true
		}
		versuche.laufend++
		return versuche, false
	}

	// Vor dem Einfügen ab einer Schwelle abgelaufene Einträge entfernen, um die Map
	// ohne Dauer-Goroutine beschränkt zu halten (wie ratelimit.Bremse).
	if len(b.jeAdresse) > 5000 {
		b.raeumeAbgelaufeneAuf(jetzt)
	}
	if b.jeAdresse == nil {
		b.jeAdresse = make(map[string]*fehlversuche)
	}
	versuche = &fehlversuche{
		anzahl:  0,
		erster:  jetzt,
		laufend: 1,
	}
	b.jeAdresse[adresse] = versuche
	return versuche, false
}

// verzeichne meldet die Anmeldung ab und zählt nach dem Status der Antwort, unter der Sperre: Nur
// 401 zählt als Fehlversuch. Eine gelungene Anmeldung leert den Zähler nicht, er läuft mit seinem Fenster ab:
// Sonst riete, wer selbst ein Konto hat, knapp unter der Grenze die Passwörter anderer, meldete
// sich an und riete weiter.
//
// 403 zählt nicht: Bei „Zugang beantragt" (Selbstanmeldung) und „Konto deaktiviert" war das
// Passwort richtig. Die Tür zählt deshalb nicht (internal/auth/anmelden.go), und diese Schicht darf
// die Entscheidung nicht wieder einsammeln. Zum Schuljahresbeginn sperrten sonst zwanzig 403
// der Selbstanmeldung hinter der einen NAT-Adresse der Schule die Anmeldung für alle, auch für
// die Bibliothekskraft, die die Freischaltungen vornimmt. Für Credential-Stuffing, den Zweck
// dieser Bremse, ist 401 das Signal, nicht 403.
func (b *anmeldeBremse) verzeichne(versuche *fehlversuche, status int) {
	b.mu.Lock()
	defer b.mu.Unlock()
	versuche.laufend--
	if status == http.StatusUnauthorized {
		versuche.anzahl++
	}
}

// schuetze lässt je Adresse höchstens maxAnmeldeFehlversucheJeAdresse gescheiterte Anmeldungen
// in 15 Minuten zu; weitere Versuche im Fenster beantwortet sie mit 429.
//
// Sie ist die Schranke für die Menge. Den Schutz eines einzelnen Kontos gegen gezieltes Raten
// trägt der Handler der Anmeldung, der je E-Mail und Adresse zählt; warum beide Schichten
// verschiedene Schwellen brauchen, steht an maxAnmeldeFehlversucheJeAdresse.
//
// Die Adresse kommt aus clientip, damit Anfragen hinter Caddy als Reverse-Proxy nach dem
// echten Aufrufer zählen und nicht nach der einen Adresse des Proxys; sonst sperrten wenige
// Fehlversuche alle Nutzer aus.
func (b *anmeldeBremse) schuetze(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		adresse := clientip.AusAnfrage(r)

		jetzt := time.Now()
		versuche, gesperrt := b.reserviere(adresse, jetzt)
		if gesperrt {
			apierrors.AntworteFehlerMitMeldung(w, http.StatusTooManyRequests, auth.MeldungZuVieleFehlversuche, nil)
			return
		}

		// Den Status der Antwort mitlesen.
		merker := &statusMerker{
			ResponseWriter: w,
			status:         http.StatusOK, // ohne WriteHeader gilt 200
		}

		// Abgemeldet wird auch nach einer Panik in der Anmeldung, sonst bliebe ihr Platz belegt.
		defer func() { b.verzeichne(versuche, merker.status) }()
		next.ServeHTTP(merker, r)
	})
}
