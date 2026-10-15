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

// Sitzungen führt je Anmeldung eine Zeile in der Tabelle sitzungen. An ihr hängt die Sperre
// nach Inaktivität: Sie gilt am Server und für jedes Token der Anmeldung, auch für ein
// erneuertes. Ein Token ohne Kennung oder ohne Zeile lässt sich nicht sperren.
type Sitzungen struct {
	pool       Datenbank
	schluessel []byte
	ctx        context.Context
	cancel     context.CancelFunc
}

// newSitzungen legt den Speicher an und startet das Abräumen abgelaufener Zeilen.
func newSitzungen(pool Datenbank, geheimnis []byte) *Sitzungen {
	ctx, cancel := context.WithCancel(context.Background())
	s := &Sitzungen{pool: pool, schluessel: pruefwertSchluessel(geheimnis), ctx: ctx, cancel: cancel}
	go s.raeumSchleife()
	return s
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

// istGesperrt meldet, ob die Anmeldung gesperrt ist. Ohne Kennung und ohne Zeile ist sie es
// nicht: Sperren lässt sich nur, was sich auch wieder aufschließen lässt.
func (s *Sitzungen) istGesperrt(sitzungID string) (bool, error) {
	if sitzungID == "" {
		return false, nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	gesperrt, err := repository.IstSitzungGesperrt(ctx, s.pool, sitzungID)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return gesperrt, nil
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

// entsperre hebt die Sperre auf. Keine Zeile ist kein Fehler: Dann war nichts gesperrt.
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

// Stoppe beendet das Abräumen im Hintergrund.
func (s *Sitzungen) Stoppe() {
	s.cancel()
}

// raeumSchleife räumt alle 15 Minuten die abgelaufenen Zeilen ab.
func (s *Sitzungen) raeumSchleife() {
	takt := time.NewTicker(15 * time.Minute)
	defer takt.Stop()
	for {
		select {
		case <-s.ctx.Done():
			return
		case <-takt.C:
			s.raeumeAb()
		}
	}
}

// raeumeAb löscht Zeilen, deren Token abgelaufen ist.
func (s *Sitzungen) raeumeAb() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := repository.LoescheAbgelaufeneSitzungen(ctx, s.pool); err != nil {
		log.Printf("sitzungen: Abräumen abgelaufener Zeilen fehlgeschlagen: %v", err)
	}
}
