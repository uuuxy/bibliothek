// Package tuer trägt, was jede Tür braucht: Eingabe lesen und Antwort schreiben, die Sitzung
// und die Rechte einer Anfrage prüfen, einen Vorgang der Verwaltung ins Protokoll schreiben,
// die Uhr der Schule, den Kopf ihrer Briefe und den Versand von Mail. Es steht unter den Türen,
// damit ein Paket mit den Türen eines Themas es einbinden kann; api/ selbst ließe sich von dort
// nicht einbinden.
package tuer

import (
	"context"
	"fmt"
	"time"

	"bibliothek/auth"
	"bibliothek/db"
	"bibliothek/mailservice"
	"bibliothek/pkg/schulzeit"
	"bibliothek/sse"
)

// Basis hält, worauf jede Tür steht: Datenbank, Anmeldung, Live-Leitung, Uhr, Rechte und
// Mailversand. Der Server bettet sie ein, damit die Türen ihre Felder und Methoden unter
// demselben Namen erreichen.
type Basis struct {
	DB     *db.Verbindung
	Auth   *auth.Anmeldung
	Broker *sse.Broker
	// Uhr liefert „jetzt" aus Sicht der Schule; nil = pkg/schulzeit. Nur Tests setzen
	// sie — für das, was am Kalender hängt (Saison der Abgängerliste): Sonst bewiese
	// derselbe Test im Oktober das Gegenteil von dem im Juni.
	Uhr func() time.Time
	// Rechte hält die Entscheidungen der Rechteprüfung; der Nullwert ist ein leerer Cache.
	Rechte RechteCache
	// Mail verschickt Mails; nil heißt, es ist kein Versand eingerichtet (siehe Post).
	Mail Mailversand
}

// Mailversand verschickt Mails für die Türen: im Betrieb ein *mailservice.Postausgang, in einem
// Test einer gegen seinen Testserver oder eine Attrappe.
type Mailversand interface {
	Sende(context.Context, mailservice.Nachricht) error
	IstKonfiguriert(context.Context) bool
}

// Post liefert, worüber die Tür Mails verschickt. Ohne eingerichteten Versand lehnt er jede Mail
// mit mailservice.ErrMailNichtKonfiguriert ab, wie ein Server ohne SMTP-Konfiguration.
func (b *Basis) Post() Mailversand { //nolint:ireturn // liefert eine von mehreren Umsetzungen: die eingesetzte oder den Ersatz ohne Versand
	if b.Mail != nil {
		return b.Mail
	}
	return ohneMailversand{}
}

type ohneMailversand struct{}

func (ohneMailversand) Sende(context.Context, mailservice.Nachricht) error {
	return fmt.Errorf("%w: kein Versand eingerichtet", mailservice.ErrMailNichtKonfiguriert)
}

func (ohneMailversand) IstKonfiguriert(context.Context) bool { return false }

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
