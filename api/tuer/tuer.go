// Package tuer trägt, was jede Tür braucht: Eingabe lesen und Antwort schreiben, die Sitzung
// und die Rechte einer Anfrage prüfen, einen Vorgang der Verwaltung ins Protokoll schreiben,
// die Uhr der Schule. Es steht unter den Türen, damit ein Paket mit den Türen eines Themas es
// einbinden kann; api/ selbst ließe sich von dort nicht einbinden.
package tuer

import (
	"time"

	"bibliothek/auth"
	"bibliothek/db"
	"bibliothek/pkg/schulzeit"
	"bibliothek/sse"
)

// Basis hält, worauf jede Tür steht: Datenbank, Anmeldung, Live-Leitung und Uhr. Der Server
// bettet sie ein, damit die Türen ihre Felder und Methoden unter demselben Namen erreichen.
type Basis struct {
	DB     *db.Verbindung
	Auth   *auth.Authenticator
	Broker *sse.Broker
	// Uhr liefert „jetzt" aus Sicht der Schule; nil = pkg/schulzeit. Nur Tests setzen
	// sie — für das, was am Kalender hängt (Saison der Abgängerliste): Sonst bewiese
	// derselbe Test im Oktober das Gegenteil von dem im Juni.
	Uhr func() time.Time
}

// Jetzt ist die Uhr des Servers: in Tests setzbar (Basis.Uhr), sonst die Schulzeit.
func (b *Basis) Jetzt() time.Time {
	if b.Uhr != nil {
		return b.Uhr()
	}
	return schulzeit.Jetzt()
}

// Datumsformate (Go-Referenzzeit "Mon Jan 2 15:04:05 MST 2006").
const (
	DatumsformatDE  = "02.01.2006" // TT.MM.JJJJ
	DatumsformatISO = "2006-01-02" // JJJJ-MM-TT
)
