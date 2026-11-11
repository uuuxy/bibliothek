// Package sse verteilt Ereignisse des Servers an offene Browser-Fenster (Server-Sent Events):
// ein Broker, bei dem sich die Fenster anmelden, und die Tür dazu.
package sse

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"sync"
	"time"
)

// pufferJeFenster ist die Zahl der Nachrichten, die ein langsamer Arbeitsplatz zurückhängen
// darf, bevor Verteile ihn überspringt. Zehn Ereignisse entsprechen mehreren Sekunden
// Bedienung an der Theke.
const pufferJeFenster = 10

// lebenszeichenTakt ist der Abstand der Lebenszeichen. Die Oberfläche meldet eine Störung nach 25
// Sekunden ohne Lebenszeichen (App.svelte).
const lebenszeichenTakt = 15 * time.Second

// hoechstensJeKonto ist die Zahl der Ströme, die ein Konto zugleich offen halten darf: einer je
// Fenster, auch über mehrere Rechner. Jeder Strom hält einen Faden und fragt alle 15 Sekunden die
// Datenbank, ob die Anmeldung noch gilt; ohne Grenze bremste ein einziges Konto mit vielen Strömen
// Server und Datenbank aus.
const hoechstensJeKonto = 20

// hoechstensGesamt begrenzt die Ströme aller Konten zusammen.
const hoechstensGesamt = 1000

// Die Gründe, aus denen meldeAn einen Strom abweist.
var (
	errHerunterfahren = errors.New("broker fährt herunter")
	errZuVieleJeKonto = errors.New("höchstzahl der ströme je konto erreicht")
	errZuVieleGesamt  = errors.New("höchstzahl aller ströme erreicht")
)

// Broker verteilt Ereignisse an alle offenen Ströme.
//
// Der Zustand liegt hinter einem RWMutex statt hinter einer Ereignisschleife mit Kanälen zum
// An- und Abmelden: Kehrt eine solche Schleife beim Herunterfahren zurück, liest niemand mehr
// aus den Kanälen, jede Tür bleibt beim Abmelden stehen, und httpServer.Shutdown wartet bis
// zu seiner Frist.
type Broker struct {
	mu sync.RWMutex
	// fenster ordnet jedem offenen Strom das Konto zu, das ihn geöffnet hat.
	fenster map[chan string]string
	jeKonto map[string]int
	// vertraut sind die Ströme, deren Konto die Fassung eines Ereignisses mit Kennungen sehen
	// darf (VerteileNachVertrauen). Ein Strom ohne Eintrag bekommt die knappe Fassung.
	vertraut map[chan string]bool
	// gemeldet sind die Konten, deren Abweisung schon im Log steht, solange sie an der Grenze
	// bleiben; gesamtGemeldet dasselbe für die Grenze aller Ströme. Ein Konto, das immer wieder
	// anklopft, schreibt so eine Zeile und nicht eine je Versuch.
	gemeldet       map[string]bool
	gesamtGemeldet bool
	geschlossen    bool
	// takt ist der Abstand der Lebenszeichen; ein Test verkürzt ihn, statt 15 Sekunden zu warten.
	takt time.Duration
	// grenzeJeKonto und grenzeGesamt sind hoechstensJeKonto und hoechstensGesamt; ein Test setzt
	// sie klein.
	grenzeJeKonto, grenzeGesamt int
}

// NewBroker baut einen Broker ohne angemeldete Fenster.
func NewBroker() *Broker {
	return &Broker{
		fenster: make(map[chan string]string), jeKonto: make(map[string]int), gemeldet: make(map[string]bool),
		vertraut: make(map[chan string]bool),
		takt:     lebenszeichenTakt, grenzeJeKonto: hoechstensJeKonto, grenzeGesamt: hoechstensGesamt,
	}
}

// FuehreAus hält den Broker am Leben, bis ctx abgebrochen wird, und fährt ihn dann herunter;
// gedacht für `go broker.FuehreAus(ctx)`.
//
// Das Herunterfahren schließt die Kanäle aller Fenster, und das beendet die laufenden Ströme:
// Ohne dieses Signal wartete jede Tür bis zu 15 Sekunden auf ihr nächstes Lebenszeichen,
// länger als die zehn Sekunden, die main.go dem Herunterfahren gibt.
func (b *Broker) FuehreAus(ctx context.Context) {
	<-ctx.Done()
	b.fahreHerunter()
}

// fahreHerunter schließt alle offenen Ströme und weist spätere Anmeldungen ab.
func (b *Broker) fahreHerunter() {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.geschlossen {
		return
	}
	b.geschlossen = true
	for kanal := range b.fenster {
		close(kanal)
		delete(b.fenster, kanal)
	}
	clear(b.jeKonto)
	clear(b.vertraut)
	slog.Info("SSE: Broker heruntergefahren, alle Ströme geschlossen")
}

// meldeAn meldet einen neuen Strom des Kontos an. Fährt der Broker gerade herunter oder ist eine
// Grenze erreicht, weist er ihn mit dem Grund ab, und die Tür antwortet sofort, statt zu warten.
func (b *Broker) meldeAn(konto string) (chan string, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.geschlossen {
		return nil, errHerunterfahren
	}
	if b.jeKonto[konto] >= b.grenzeJeKonto {
		if !b.gemeldet[konto] {
			b.gemeldet[konto] = true
			slog.Warn("SSE: Höchstzahl der Ströme je Konto erreicht, weitere werden abgewiesen", "konto", konto, "grenze", b.grenzeJeKonto)
		}
		return nil, errZuVieleJeKonto
	}
	if len(b.fenster) >= b.grenzeGesamt {
		if !b.gesamtGemeldet {
			b.gesamtGemeldet = true
			slog.Error("SSE: Höchstzahl aller Ströme erreicht, weitere werden abgewiesen", "grenze", b.grenzeGesamt)
		}
		return nil, errZuVieleGesamt
	}
	kanal := make(chan string, pufferJeFenster)
	b.fenster[kanal] = konto
	b.jeKonto[konto]++
	slog.Info("SSE: New client registered")
	return kanal, nil
}

// meldeAb meldet einen Strom ab und schließt seinen Kanal. Ein zweiter Aufruf tut nichts: Nach
// dem Herunterfahren ist der Kanal schon geschlossen, und ein zweites close bräche das
// Programm ab.
func (b *Broker) meldeAb(kanal chan string) {
	b.mu.Lock()
	defer b.mu.Unlock()

	konto, offen := b.fenster[kanal]
	if !offen {
		return
	}
	delete(b.fenster, kanal)
	delete(b.vertraut, kanal)
	close(kanal)
	if b.jeKonto[konto]--; b.jeKonto[konto] <= 0 {
		delete(b.jeKonto, konto)
	}
	if b.jeKonto[konto] < b.grenzeJeKonto {
		delete(b.gemeldet, konto)
	}
	if len(b.fenster) < b.grenzeGesamt {
		b.gesamtGemeldet = false
	}
	slog.Info("SSE: Client disconnected")
}

// Verteile schickt ein Ereignis an alle angemeldeten Fenster.
//
// Die Lesesperre deckt auch das Senden ab: meldeAb und fahreHerunter schließen Kanäle unter
// der Schreibsperre, und ein Senden auf einen geschlossenen Kanal bricht das Programm ab.
// Beides darf sich deshalb nie überschneiden.
func (b *Broker) Verteile(ereignis, daten string) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	nachricht := fmt.Sprintf("event: %s\ndata: %s\n\n", ereignis, daten)
	for kanal := range b.fenster {
		select {
		case kanal <- nachricht:
		default:
			// Ein Fenster mit vollem Puffer verpasst das Ereignis, statt die anderen aufzuhalten.
		}
	}
}

// merkeVertrauen hält fest, ob der Strom die Fassung mit Kennungen bekommt. Für einen Strom, der
// schon abgemeldet ist, tut es nichts.
func (b *Broker) merkeVertrauen(kanal chan string, ja bool) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if _, offen := b.fenster[kanal]; offen {
		b.vertraut[kanal] = ja
	}
}

// VerteileNachVertrauen schickt ein Ereignis in zwei Fassungen: datenVertraut an die Ströme, deren
// Konto Kennungen sehen darf (Tuer, vertraut), datenUebrige an alle anderen. So erfährt jede
// Oberfläche, dass etwas geschehen ist, aber nur die Theke, bei wem.
func (b *Broker) VerteileNachVertrauen(ereignis, datenVertraut, datenUebrige string) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	voll := fmt.Sprintf("event: %s\ndata: %s\n\n", ereignis, datenVertraut)
	knapp := fmt.Sprintf("event: %s\ndata: %s\n\n", ereignis, datenUebrige)
	for kanal := range b.fenster {
		nachricht := knapp
		if b.vertraut[kanal] {
			nachricht = voll
		}
		select {
		case kanal <- nachricht:
		default:
			// Wie in Verteile: Ein Fenster mit vollem Puffer verpasst das Ereignis.
		}
	}
}

// Tuer liefert die Tür, an der ein Fenster seinen Strom öffnet: Kopfzeilen für den Strom, ein
// Lebenszeichen alle 15 Sekunden, dann die Ereignisse.
//
// Vor jedem Lebenszeichen fragt die Tür gilt, ob die Anmeldung noch gilt, mit der der Strom
// aufgebaut wurde, und beendet ihn sonst. Der Strom hat keine Frist: Wer ihn mit einem kopierten
// Token geöffnet hat, läse nach Abmelden oder Sperre sonst bis zum nächsten Neustart mit. konto
// nennt das Konto der Anmeldung; je Konto und insgesamt ist die Zahl der Ströme begrenzt.
// vertraut sagt, ob das Konto die Fassung mit Kennungen bekommt (VerteileNachVertrauen); gefragt
// wird beim Aufbau und nach jedem Stück des Stroms neu, damit ein entzogenes Recht spätestens
// mit dem nächsten Lebenszeichen wirkt.
func (b *Broker) Tuer(gilt func(*http.Request) bool, konto func(*http.Request) string, vertraut func(*http.Request) bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		kanal, err := b.meldeAn(konto(r))
		if errors.Is(err, errHerunterfahren) {
			// 503, damit die Tür das Herunterfahren nicht aufhält; der Browser verbindet sich
			// später neu.
			http.Error(w, "Server fährt herunter", http.StatusServiceUnavailable)
			return
		}
		if err != nil {
			// Die Oberfläche versucht es mit wachsender Wartezeit wieder (liveEvents.js).
			http.Error(w, "Zu viele offene Live-Verbindungen", http.StatusTooManyRequests)
			return
		}
		defer b.meldeAb(kanal)
		b.merkeVertrauen(kanal, vertraut(r))

		rc := http.NewResponseController(w)
		hebeFristenAuf(rc)

		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		// Kein „Connection: keep-alive": HTTP/2 verbietet das Kopffeld (RFC 9113 §8.2.2), der
		// Browser bricht mit ERR_HTTP2_PROTOCOL_ERROR ab. Der Strom läuft über HTTP/1.1 und
		// HTTP/2 auch ohne; CORS regelt corsMiddleware für alle Wege.

		b.schreibeEreignisse(w, r, rc, kanal, gilt, vertraut)
	}
}

// hebeFristenAuf hebt die Fristen zum Lesen und Schreiben für den langen Strom auf, damit ihn
// die Fristen des Servers nicht beenden. Nicht jede Verbindung kennt Fristen; ein Fehler steht
// deshalb nur im Log.
func hebeFristenAuf(rc *http.ResponseController) {
	if err := rc.SetReadDeadline(time.Time{}); err != nil {
		slog.Warn("SSE: could not clear read deadline", "fehler", err)
	}
	if err := rc.SetWriteDeadline(time.Time{}); err != nil {
		slog.Warn("SSE: could not clear write deadline", "fehler", err)
	}
}

// schreibeEreignisse schreibt die Begrüßung, alle 15 Sekunden ein Lebenszeichen und die
// verteilten Ereignisse, bis der Browser die Verbindung schließt, der Broker herunterfährt,
// die Anmeldung nicht mehr gilt oder das Schreiben scheitert.
func (b *Broker) schreibeEreignisse(w http.ResponseWriter, r *http.Request, rc *http.ResponseController, kanal chan string, gilt, vertraut func(*http.Request) bool) {
	// schreibe schickt ein Stück des Stroms sofort hinaus. Ein Fehler heißt, dass der Browser
	// weg ist; die Tür endet dann.
	schreibe := func(stueck string) error {
		if _, err := io.WriteString(w, stueck); err != nil {
			return err
		}
		return rc.Flush()
	}

	if err := schreibe("event: connected\ndata: {\"status\":\"ok\"}\n\n"); err != nil {
		return
	}

	// Am Lebenszeichen erkennt die Oberfläche, dass die Verbindung lebt; bleibt es aus, meldet
	// sie die Störung. Der Server merkt am Schreiben, dass ein Fenster weg ist.
	takt := time.NewTicker(b.takt)
	defer takt.Stop()

	for {
		stueck, weiter := naechstesStueck(r, takt.C, kanal, gilt)
		if !weiter || schreibe(stueck) != nil {
			return
		}
		b.merkeVertrauen(kanal, vertraut(r))
	}
}

// naechstesStueck wartet auf das nächste Stück des Stroms: ein Lebenszeichen, wenn die Anmeldung
// noch gilt, oder ein verteiltes Ereignis. false heißt, dass der Strom endet: Der Browser ist weg,
// die Anmeldung gilt nicht mehr, oder der Broker ist heruntergefahren oder hat das Fenster
// abgemeldet.
func naechstesStueck(r *http.Request, takt <-chan time.Time, kanal <-chan string, gilt func(*http.Request) bool) (string, bool) {
	select {
	case <-r.Context().Done():
		return "", false
	case <-takt:
		if !gilt(r) {
			return "", false
		}
		return fmt.Sprintf("event: ping\ndata: {\"timestamp\":%d}\n\n", time.Now().Unix()), true
	case nachricht, ok := <-kanal:
		return nachricht, ok
	}
}
