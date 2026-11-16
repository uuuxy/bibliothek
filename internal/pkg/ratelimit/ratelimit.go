// Package ratelimit bremst Anfragen je Adresse: ein Vorrat je Adresse, der sich mit der Zeit
// wieder füllt. Die Middleware des Servers nimmt es für jede Anfrage, der Abruf der Cover für
// die Fehltreffer, die er bei fremden Diensten nachfragt.
package ratelimit

import (
	"sync"
	"time"
)

type besucher struct {
	vorrat  float64
	zuletzt time.Time
}

// Bremse führt je Adresse einen Vorrat von jeSekunde Anfragen, der sich in einer Sekunde
// wieder füllt.
type Bremse struct {
	mu        sync.Mutex
	jeAdresse map[string]*besucher
	jeSekunde int
}

// New liefert eine Bremse mit limit Anfragen je Sekunde und Adresse.
func New(limit int) *Bremse {
	return &Bremse{
		jeAdresse: make(map[string]*besucher),
		jeSekunde: limit,
	}
}

// raeumeAlteAuf entfernt Adressen, die fünf Minuten nichts geschickt haben, sobald mehr als
// 5000 gemerkt sind; sonst wüchse die Karte mit jeder neuen Adresse.
func (b *Bremse) raeumeAlteAuf(jetzt time.Time) {
	if len(b.jeAdresse) <= 5000 {
		return
	}
	for adresse, alt := range b.jeAdresse {
		if jetzt.Sub(alt.zuletzt) > 5*time.Minute {
			delete(b.jeAdresse, adresse)
		}
	}
}

// LaesstDurch meldet, ob die Adresse noch eine Anfrage frei hat, und nimmt sie dann aus
// ihrem Vorrat.
func (b *Bremse) LaesstDurch(adresse string) bool {
	b.mu.Lock()
	defer b.mu.Unlock()

	jetzt := time.Now()
	besuch, bekannt := b.jeAdresse[adresse]
	if !bekannt {
		besuch = &besucher{
			vorrat:  float64(b.jeSekunde),
			zuletzt: jetzt,
		}
		b.jeAdresse[adresse] = besuch

		b.raeumeAlteAuf(jetzt)

		besuch.vorrat -= 1.0
		return true
	}

	vergangen := jetzt.Sub(besuch.zuletzt)
	besuch.zuletzt = jetzt

	besuch.vorrat += vergangen.Seconds() * float64(b.jeSekunde)
	if besuch.vorrat > float64(b.jeSekunde) {
		besuch.vorrat = float64(b.jeSekunde)
	}

	if besuch.vorrat >= 1.0 {
		besuch.vorrat -= 1.0
		return true
	}

	return false
}
