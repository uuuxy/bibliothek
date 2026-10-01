package auth

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/jackc/pgx/v5"
)

// Sitzungen führt je Anmeldung eine Zeile in der Tabelle sitzungen. An ihr hängt die Sperre
// nach Inaktivität: Sie gilt am Server und für jedes Token der Anmeldung, auch für ein
// erneuertes. Ein Token ohne Kennung oder ohne Zeile lässt sich nicht sperren.
type Sitzungen struct {
	pool       DatabasePool
	schluessel []byte
	ctx        context.Context
	cancel     context.CancelFunc
}

// NewSitzungen legt den Speicher an und startet das Abräumen abgelaufener Zeilen.
func NewSitzungen(pool DatabasePool, geheimnis []byte) *Sitzungen {
	ctx, cancel := context.WithCancel(context.Background())
	s := &Sitzungen{pool: pool, schluessel: pruefwertSchluessel(geheimnis), ctx: ctx, cancel: cancel}
	go s.raeumSchleife()
	return s
}

// Beginne legt die Zeile einer neuen Anmeldung an und liefert ihre Kennung für das Token.
func (s *Sitzungen) Beginne(ctx context.Context, benutzerID, passwort string, laeuftAb time.Time) (string, error) {
	pruefwert, err := bildePruefwert(s.schluessel, passwort)
	if err != nil {
		return "", err
	}
	var id string
	if err := s.pool.QueryRow(ctx, `
		INSERT INTO sitzungen (benutzer_id, passwort_pruefwert, laeuft_ab)
		VALUES ($1, $2, $3)
		RETURNING id::text
	`, benutzerID, pruefwert, laeuftAb).Scan(&id); err != nil {
		return "", fmt.Errorf("sitzung anlegen: %w", err)
	}
	return id, nil
}

// IstGesperrt meldet, ob die Anmeldung gesperrt ist. Ohne Kennung und ohne Zeile ist sie es
// nicht: Sperren lässt sich nur, was sich auch wieder aufschließen lässt.
func (s *Sitzungen) IstGesperrt(sitzungID string) (bool, error) {
	if sitzungID == "" {
		return false, nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	var gesperrt bool
	err := s.pool.QueryRow(ctx, `
		SELECT gesperrt_seit IS NOT NULL FROM sitzungen WHERE id = $1
	`, sitzungID).Scan(&gesperrt)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return gesperrt, nil
}

// Sperre sperrt die Anmeldung und meldet, ob es eine Zeile dafür gab. Eine schon gesperrte
// Anmeldung behält ihren Zeitpunkt.
func (s *Sitzungen) Sperre(ctx context.Context, sitzungID string) (bool, error) {
	if sitzungID == "" {
		return false, nil
	}
	tag, err := s.pool.Exec(ctx, `
		UPDATE sitzungen SET gesperrt_seit = COALESCE(gesperrt_seit, NOW()) WHERE id = $1
	`, sitzungID)
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() == 1, nil
}

// Entsperre hebt die Sperre auf. Keine Zeile ist kein Fehler: Dann war nichts gesperrt.
func (s *Sitzungen) Entsperre(ctx context.Context, sitzungID string) error {
	if sitzungID == "" {
		return nil
	}
	_, err := s.pool.Exec(ctx, `UPDATE sitzungen SET gesperrt_seit = NULL WHERE id = $1`, sitzungID)
	return err
}

// PasswortPasst prüft ein Passwort gegen den Prüfwert der Anmeldung. vorhanden=false heißt:
// Es gibt keinen Prüfwert, gegen den sich prüfen ließe.
func (s *Sitzungen) PasswortPasst(ctx context.Context, sitzungID, passwort string) (passt, vorhanden bool, err error) {
	if sitzungID == "" {
		return false, false, nil
	}
	var pruefwert string
	err = s.pool.QueryRow(ctx, `
		SELECT passwort_pruefwert FROM sitzungen WHERE id = $1
	`, sitzungID).Scan(&pruefwert)
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

// MerkePasswort schreibt den Prüfwert neu. Der Mailserver hat ein Passwort angenommen, das
// nicht zum Prüfwert passt: Es wurde seit der Anmeldung geändert.
func (s *Sitzungen) MerkePasswort(ctx context.Context, sitzungID, passwort string) error {
	pruefwert, err := bildePruefwert(s.schluessel, passwort)
	if err != nil {
		return err
	}
	_, err = s.pool.Exec(ctx, `
		UPDATE sitzungen SET passwort_pruefwert = $2 WHERE id = $1
	`, sitzungID, pruefwert)
	return err
}

// Verlaengere schiebt das Ende der Zeile mit dem erneuerten Token hinaus und meldet, ob es
// eine Zeile gab. Ohne Zeile wird das Token nicht erneuert (RefreshTokenHandler).
func (s *Sitzungen) Verlaengere(ctx context.Context, sitzungID string, laeuftAb time.Time) (bool, error) {
	if sitzungID == "" {
		return false, nil
	}
	tag, err := s.pool.Exec(ctx, `
		UPDATE sitzungen SET laeuft_ab = $2 WHERE id = $1
	`, sitzungID, laeuftAb)
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() == 1, nil
}

// Beende löscht die Zeile der Anmeldung und mit ihr den Prüfwert (Abmelden).
func (s *Sitzungen) Beende(ctx context.Context, sitzungID string) error {
	if sitzungID == "" {
		return nil
	}
	_, err := s.pool.Exec(ctx, `DELETE FROM sitzungen WHERE id = $1`, sitzungID)
	return err
}

// Stop beendet das Abräumen.
func (s *Sitzungen) Stop() {
	s.cancel()
}

func (s *Sitzungen) raeumSchleife() {
	ticker := time.NewTicker(15 * time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-s.ctx.Done():
			return
		case <-ticker.C:
			s.raeumeAb()
		}
	}
}

// raeumeAb löscht Zeilen, deren Token abgelaufen ist. Eine gesperrte Zeile wird dabei nicht
// geschont: Ihr Token gilt nicht mehr, aufzuschließen gibt es nichts.
func (s *Sitzungen) raeumeAb() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if _, err := s.pool.Exec(ctx, `DELETE FROM sitzungen WHERE laeuft_ab < NOW()`); err != nil {
		log.Printf("sitzungen: Abräumen abgelaufener Zeilen fehlgeschlagen: %v", err)
	}
}
