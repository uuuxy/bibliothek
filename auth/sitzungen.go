package auth

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"bibliothek/repository"

	"github.com/jackc/pgx/v5"
)

// hoechstdauerNachPasswort: So lange gilt eine Anmeldung nach der letzten Passworteingabe, danach
// ist sie gesperrt. Zehn Stunden decken den längsten Schultag mit einer Stunde Puffer, ein
// kopiertes Token gilt so höchstens bis zum Abend; MeldungHoechstdauerErreicht und der
// Sperrbildschirm nennen dieselbe Zahl.
const hoechstdauerNachPasswort = 10 * time.Hour

// sperrGrund sagt, warum eine Anmeldung gesperrt ist; die Oberfläche nennt ihn am
// Sperrbildschirm. Der leere Grund heißt: nicht gesperrt.
type sperrGrund string

const (
	sperrGrundKeiner       sperrGrund = ""
	sperrGrundInaktivitaet sperrGrund = "inaktivitaet"
	sperrGrundHoechstdauer sperrGrund = "hoechstdauer"
)

// Sitzungen führt je Anmeldung eine Zeile in der Tabelle sitzungen. An ihr hängen die Sperre
// nach Inaktivität und die Höchstdauer nach der Passworteingabe: Beide gelten am Server und für
// jedes Token der Anmeldung, auch für ein erneuertes. Ein Token ohne Kennung oder ohne Zeile
// lässt sich nicht sperren.
type Sitzungen struct {
	pool       Datenbank
	schluessel []byte
}

// newSitzungen legt den Speicher an. Abgelaufene Zeilen räumt Anmeldung.RaeumeAuf ab.
func newSitzungen(pool Datenbank, geheimnis []byte) *Sitzungen {
	return &Sitzungen{pool: pool, schluessel: pruefwertSchluessel(geheimnis)}
}

// beginne legt die Zeile einer neuen Anmeldung an und liefert ihre Kennung für das Token.
func (s *Sitzungen) beginne(ctx context.Context, kontoID, passwort string, laeuftAb time.Time) (string, error) {
	pruefwert, err := bildePruefwert(s.schluessel, passwort)
	if err != nil {
		return "", err
	}
	id, err := repository.LegeSitzungAn(ctx, s.pool, kontoID, pruefwert, laeuftAb)
	if err != nil {
		return "", fmt.Errorf("sitzung anlegen: %w", err)
	}
	return id, nil
}

// grundDerSperre nennt, warum die Anmeldung gesperrt ist, leer für offen. Ohne Kennung und ohne
// Zeile ist sie es nicht: Sperren lässt sich nur, was sich auch wieder aufschließen lässt.
func (s *Sitzungen) grundDerSperre(ctx context.Context, sitzungID string) (sperrGrund, error) {
	grund, _, err := s.zustand(ctx, sitzungID)
	return grund, err
}

// zustand meldet, warum die Anmeldung gesperrt ist, und ob ihre Zeile noch besteht. Ein Token
// ohne Kennung hat keine Zeile und keine Sperre; es läuft aus, weil die Erneuerung es nicht
// verlängert.
func (s *Sitzungen) zustand(ctx context.Context, sitzungID string) (grund sperrGrund, besteht bool, err error) {
	if sitzungID == "" {
		return sperrGrundKeiner, true, nil
	}
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	gesperrt, passwortGeprueftAm, err := repository.LadeSitzungsZustand(ctx, s.pool, sitzungID)
	if errors.Is(err, pgx.ErrNoRows) {
		return sperrGrundKeiner, false, nil
	}
	if err != nil {
		return sperrGrundKeiner, false, err
	}
	switch {
	case gesperrt:
		return sperrGrundInaktivitaet, true, nil
	case time.Since(passwortGeprueftAm) >= hoechstdauerNachPasswort:
		return sperrGrundHoechstdauer, true, nil
	}
	return sperrGrundKeiner, true, nil
}

// sperre sperrt die Anmeldung und meldet, ob es eine Zeile dafür gab. Eine schon gesperrte
// Anmeldung behält ihren Zeitpunkt.
func (s *Sitzungen) sperre(ctx context.Context, sitzungID string) (bool, error) {
	if sitzungID == "" {
		return false, nil
	}
	zeilen, err := repository.SperreSitzung(ctx, s.pool, sitzungID)
	if err != nil {
		return false, err
	}
	return zeilen == 1, nil
}

// entsperre hebt die Sperre auf, nachdem das Passwort geprüft ist; die Höchstdauer beginnt damit
// neu. Keine Zeile ist kein Fehler: Dann war nichts gesperrt.
func (s *Sitzungen) entsperre(ctx context.Context, sitzungID string) error {
	if sitzungID == "" {
		return nil
	}
	return repository.EntsperreSitzung(ctx, s.pool, sitzungID)
}

// passwortPasst prüft ein Passwort gegen den Prüfwert der Anmeldung. vorhanden=false heißt:
// Es gibt keinen Prüfwert, gegen den sich prüfen ließe.
func (s *Sitzungen) passwortPasst(ctx context.Context, sitzungID, passwort string) (passt, vorhanden bool, err error) {
	if sitzungID == "" {
		return false, false, nil
	}
	pruefwert, err := repository.LadeSitzungsPruefwert(ctx, s.pool, sitzungID)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, false, nil
	}
	if err != nil {
		return false, false, err
	}
	passt, err = pruefwertPasst(s.schluessel, pruefwert, passwort)
	if err != nil {
		return false, false, err
	}
	return passt, true, nil
}

// merkePasswort schreibt den Prüfwert neu. Der Mailserver hat ein Passwort angenommen, das
// nicht zum Prüfwert passt: Es wurde seit der Anmeldung geändert.
func (s *Sitzungen) merkePasswort(ctx context.Context, sitzungID, passwort string) error {
	pruefwert, err := bildePruefwert(s.schluessel, passwort)
	if err != nil {
		return err
	}
	return repository.SetzeSitzungsPruefwert(ctx, s.pool, sitzungID, pruefwert)
}

// verlaengere schiebt das Ende der Zeile mit dem erneuerten Token hinaus und meldet, ob es
// eine Zeile gab. Ohne Zeile wird das Token nicht erneuert (SitzungVerlaengern).
func (s *Sitzungen) verlaengere(ctx context.Context, sitzungID string, laeuftAb time.Time) (bool, error) {
	if sitzungID == "" {
		return false, nil
	}
	zeilen, err := repository.VerlaengereSitzung(ctx, s.pool, sitzungID, laeuftAb)
	if err != nil {
		return false, err
	}
	return zeilen == 1, nil
}

// Beende löscht die Zeile der Anmeldung und mit ihr den Prüfwert (Abmelden).
func (s *Sitzungen) Beende(ctx context.Context, sitzungID string) error {
	if sitzungID == "" {
		return nil
	}
	return repository.LoescheSitzung(ctx, s.pool, sitzungID)
}

// raeumeAb löscht Zeilen, deren Token abgelaufen ist.
func (s *Sitzungen) raeumeAb(ctx context.Context) {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	if err := repository.LoescheAbgelaufeneSitzungen(ctx, s.pool); err != nil {
		log.Printf("sitzungen: Abräumen abgelaufener Zeilen fehlgeschlagen: %v", err)
	}
}
