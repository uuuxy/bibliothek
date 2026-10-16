// Package auth meldet eine Person am Postfach der Schule an (IMAP), stellt ihre Sitzung als
// JWT aus und prüft sie bei jeder Anfrage. Dazu gehören die Sperrliste widerrufener Token, die
// Sperre nach Inaktivität und die Selbstanmeldung des Kollegiums.
package auth

import (
	"context"
	"crypto/rand"
	"crypto/tls"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"bibliothek/repository"

	"github.com/golang-jwt/jwt/v5"
	"github.com/jackc/pgx/v5"
)

// Rolle ist die Rolle eines Kontos; was sie darf, steht in role_permissions.
type Rolle string

const (
	// RolleAdmin darf alles, auch die Konfiguration und die Rechte.
	RolleAdmin Rolle = "ADMIN"
	// RolleKollegium ist der Grundzustand jeder Lehrkraft, keine vergebene Rolle: Wer sich über
	// „Mein Portal" mit seiner Schuladresse selbst anmeldet, entsteht als Kollegium
	// (auth/selbstanmeldung.go, inaktiv bis zur Freischaltung) und ist damit niemand
	// Besonderes: Bestand sehen, vormerken, Fehler melden, an der Theke Bücher auf den eigenen
	// Namen. Das sind fast alle.
	//
	// Der Wert heißt nicht „lehrer", damit er nicht mit dem gesperrten Klassennamen 'lehrer'
	// verwechselt wird (api/leser/leser_klasse_regel.go).
	RolleKollegium Rolle = "KOLLEGIUM"
	// RolleLeitung ist die Leitung der Bibliothek: alles außer den zwei Türen der Systempflege,
	// manage_users (Benutzer & Rechte) und manage_settings (Einstellungen). Der Admin erhebt sie
	// an der E-Mail-Adresse, und sie ist kein Administrator: Ein Admin-Konto bleibt ihr
	// verschlossen (api/benutzer/admin_schutz.go).
	RolleLeitung Rolle = "LEITUNG"
	// RolleMitarbeiter führt das tägliche Geschäft der Ausleihe.
	RolleMitarbeiter Rolle = "MITARBEITER"
	// RolleHelfer leiht an der Theke aus und nimmt zurück.
	RolleHelfer Rolle = "HELFER"
)

// MarshalJSON schreibt die Rolle klein, wie die Oberfläche sie vergleicht.
func (r Rolle) MarshalJSON() ([]byte, error) {
	return json.Marshal(strings.ToLower(string(r)))
}

// UnmarshalJSON liest die Rolle in Klein- oder Großschreibung und hält sie groß wie die
// Konstanten.
func (r *Rolle) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}
	*r = Rolle(strings.ToUpper(s))
	return nil
}

// Claims ist der Inhalt des Tokens einer Sitzung.
type Claims struct {
	KontoID       string `json:"user_id"`
	Ausweisnummer string `json:"barcode_id"`
	Rolle         Rolle  `json:"rolle"`
	// SitzungID ist die Kennung der Anmeldung in der Tabelle sitzungen. Sie bleibt über jede
	// Erneuerung des Tokens dieselbe; an ihr hängt die Sperre nach Inaktivität.
	SitzungID string `json:"sid,omitempty"`
	jwt.RegisteredClaims
}

// Anmeldung stellt die Tokens der Sitzungen aus und prüft sie.
type Anmeldung struct {
	geheimnis  []byte
	laufzeit   time.Duration
	Sperrliste *Sperrliste
	Sitzungen  *Sitzungen
	pool       Datenbank
	// imapTLS ändert die Einstellungen für TLS zum Mailserver; im Betrieb nil. Ein Test lässt so
	// seinen Mailserver mit selbst signiertem Zertifikat zu.
	imapTLS func(*tls.Config)
	// bremse zählt die Fehlversuche je E-Mail und Adresse für Anmelden und Entsperren: fünf in
	// 15 Minuten.
	bremse *fehlversuchsBremse
}

// NewAnmeldung baut die Anmeldung mit dem Geheimnis für die Signatur und der Laufzeit der
// Tokens.
func NewAnmeldung(geheimnis string, pool Datenbank, laufzeit time.Duration) (*Anmeldung, error) {
	if len(geheimnis) < 32 {
		return nil, errors.New("JWT secret must be at least 32 bytes for security")
	}
	return &Anmeldung{
		geheimnis:  []byte(geheimnis),
		laufzeit:   laufzeit,
		Sperrliste: newSperrliste(pool),
		Sitzungen:  newSitzungen(pool, []byte(geheimnis)),
		pool:       pool,
		bremse:     newFehlversuchsBremse(5, 15*time.Minute),
	}, nil
}

// ErzeugeToken stellt ein signiertes Token mit Konto und Rolle aus. sitzungID ist die Kennung
// aus Sitzungen.Beginne; ein Token ohne sie lässt sich nicht sperren.
func (a *Anmeldung) ErzeugeToken(kontoID, ausweisnummer string, rolle Rolle, sitzungID string) (string, error) {
	// jti macht jedes Token einmalig. Ohne sie wären zwei Anmeldungen desselben Kontos in
	// derselben Sekunde Byte für Byte gleich (iat und exp zählen Sekunden), und das Abmelden der
	// einen widerriefe über den Hash in der Sperrliste auch die andere.
	kennung := make([]byte, 16)
	if _, err := rand.Read(kennung); err != nil {
		return "", fmt.Errorf("failed to generate token id: %w", err)
	}

	claims := Claims{
		KontoID:       kontoID,
		Ausweisnummer: ausweisnummer,
		Rolle:         rolle,
		SitzungID:     sitzungID,
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        hex.EncodeToString(kennung),
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(a.laufzeit)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
			Issuer:    "bibliothek-system",
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signiert, err := token.SignedString(a.geheimnis)
	if err != nil {
		return "", fmt.Errorf("failed to sign token: %w", err)
	}

	return signiert, nil
}

// PruefeToken prüft das Token wie PruefeTokenTrotzSperre und lehnt zusätzlich ab, wenn die
// Anmeldung gesperrt ist (ErrSitzungGesperrt). Das ist die Prüfung für jede Anfrage, die Daten
// liefert oder ändert.
func (a *Anmeldung) PruefeToken(token string) (*Claims, error) {
	claims, err := a.PruefeTokenTrotzSperre(token)
	if err != nil {
		return nil, err
	}
	gesperrt, err := a.Sitzungen.istGesperrt(claims.SitzungID)
	if err != nil {
		return nil, fmt.Errorf("%w: sitzung: %v", ErrPruefungGestoert, err)
	}
	if gesperrt {
		return nil, ErrSitzungGesperrt
	}
	return claims, nil
}

// PruefeTokenTrotzSperre prüft Signatur, Widerruf in der Sperrliste und den aktuellen Stand des
// Kontos und liefert die Claims. Die Sperre nach Inaktivität prüft sie nicht; sie ist für die
// Wege, die eine gesperrte Anmeldung braucht: aufschließen, abmelden, den eigenen Zustand lesen,
// das Token erneuern.
func (a *Anmeldung) PruefeTokenTrotzSperre(token string) (*Claims, error) {
	widerrufen, err := a.Sperrliste.istWiderrufen(token)
	if err != nil {
		return nil, fmt.Errorf("%w: sperrliste: %v", ErrPruefungGestoert, err)
	}
	if widerrufen {
		return nil, errors.New("token has been revoked (logged out)")
	}

	gelesen, err := jwt.ParseWithClaims(token, &Claims{}, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return a.geheimnis, nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to parse token: %w", err)
	}

	claims, ok := gelesen.Claims.(*Claims)
	if !ok || !gelesen.Valid {
		return nil, errors.New("invalid token claims")
	}

	// Signatur und Sperrliste allein reichen nicht: Ein deaktiviertes oder gelöschtes Konto hat
	// sich nie abgemeldet, die Sperrliste kennt sein Token nicht, und es behielte bis zum Ablauf
	// (12 Stunden) vollen Zugriff. Deshalb gilt bei jeder Prüfung der Stand in der Datenbank.
	rolle, err := a.ladeKontoStatus(claims.KontoID)
	if err != nil {
		return nil, err
	}

	// Die Rolle aus der Datenbank ersetzt die im Token signierte. Sonst wirkte eine
	// Herabstufung erst mit dem Ablauf des Tokens: api.verlangeRecht wertet claims.Rolle aus, und
	// jedes erneuerte Token schriebe die alte Rolle fort. Die Abfrage läuft für den Status ohnehin.
	claims.Rolle = rolle

	return claims, nil
}

// ladeKontoStatus stellt sicher, dass das Konto noch besteht und aktiv ist, und liefert seine
// aktuelle Rolle. Ein Fehler der Datenbank verweigert den Zugriff wie bei der Sperrliste,
// gemeldet als ErrPruefungGestoert (503) und nicht als ungültige Sitzung (401).
func (a *Anmeldung) ladeKontoStatus(kontoID string) (Rolle, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	aktiv, rolle, err := repository.LadeKontoAktivUndRolle(ctx, a.pool, kontoID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", errors.New("user account no longer exists")
		}
		return "", fmt.Errorf("%w: kontostatus: %v", ErrPruefungGestoert, err)
	}
	if !aktiv {
		return "", errors.New("user account is deactivated")
	}
	// Groß wie die Konstanten: role_permissions vergleicht zwar mit UPPER(), die Ausnahme für den
	// Admin in verlangeRecht aber mit EqualFold auf diesen Wert.
	return Rolle(strings.ToUpper(strings.TrimSpace(rolle))), nil
}
