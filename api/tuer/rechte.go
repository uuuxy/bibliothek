package tuer

import (
	"context"
	"errors"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"bibliothek/apierrors"
	"bibliothek/auth"
	"bibliothek/repository"
)

// RechteCache hält die Entscheidung über ein Recht je Rolle 60 s lang, damit nicht jede Anfrage
// die Tabelle der Rechte liest. Sein Nullwert ist leer und benutzbar; der Server trägt einen in
// seiner Basis, und alle Themen teilen ihn über die Basis.
type RechteCache struct {
	mu        sync.RWMutex
	eintraege map[string]cacheEintrag
	// epoche zählt, wie oft der Cache geleert wurde. Eine Entscheidung aus der Datenbank kommt
	// nur in den Cache, wenn ihn seit dem Beginn ihres Lesens niemand geleert hat: Sonst schriebe
	// eine Anfrage, die den Stand kurz vor einer Rechteänderung gelesen hat, ihn nach dem Leeren
	// zurück, und er wirkte bis zu 60 s weiter.
	epoche uint64
}

type cacheEintrag struct {
	erlaubt bool
	giltBis time.Time
}

// Leere leert den Cache. Wer Rollen oder Rechte ändert, ruft es, damit keine alte Entscheidung
// bis zu 60 s weiter gilt.
func (c *RechteCache) Leere() {
	c.mu.Lock()
	c.eintraege = nil
	c.epoche++
	c.mu.Unlock()
}

// MerkeFuerTest legt die Entscheidung für eine Rolle und ein Recht eine Minute lang in den
// Cache, als hätte die Datenbank sie geliefert: Ein Test ohne Tabelle der Rechte stellt so ein,
// was eine Rolle darf.
func (c *RechteCache) MerkeFuerTest(rolle, recht string, erlaubt bool) {
	c.mu.Lock()
	c.schreibe(rolle+":"+recht, cacheEintrag{erlaubt: erlaubt, giltBis: time.Now().Add(time.Minute)})
	c.mu.Unlock()
}

// schreibe legt einen Eintrag ab; der Aufrufer hält die Sperre.
func (c *RechteCache) schreibe(schluessel string, eintrag cacheEintrag) {
	if c.eintraege == nil {
		c.eintraege = make(map[string]cacheEintrag)
	}
	c.eintraege[schluessel] = eintrag
}

// aktuelleEpoche liefert, wie oft der Cache bisher geleert wurde. Wer aus der Datenbank liest,
// fragt sie vorher ab, damit er beim Schreiben in den Cache erkennt, ob seine Entscheidung
// überholt ist.
func (c *RechteCache) aktuelleEpoche() uint64 {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.epoche
}

// recht liefert die Entscheidung aus dem Cache, solange sie gilt.
func (c *RechteCache) recht(schluessel string) (erlaubt bool, gefunden bool) {
	c.mu.RLock()
	eintrag, ok := c.eintraege[schluessel]
	c.mu.RUnlock()
	if ok && time.Now().Before(eintrag.giltBis) {
		return eintrag.erlaubt, true
	}
	return false, false
}

// ladeUndMerkeRecht lädt das Recht aus der Datenbank und merkt sich nur eine stabile
// Entscheidung (gewährt oder nicht gewährt; pgx.ErrNoRows heißt nicht gewährt). Ein Fehler der
// Datenbank (Zeitüberschreitung, Pool erschöpft, Verbindung abgerissen) kommt nicht in den
// Cache: Ein kurzer Aussetzer hielte sonst 60 s lang ein 403 fest, und er ist ein Fehler des
// Servers (500), keine Frage des Rechts.
func (b *Basis) ladeUndMerkeRecht(ctx context.Context, rolle, recht, schluessel string) (bool, error) {
	// Die Epoche vor dem Lesen festhalten: Wurde der Cache bis zum Schreiben geleert, hat
	// dieses Lesen womöglich den Stand vor der Rechteänderung gesehen. Die Anfrage darf damit
	// noch antworten, der Cache hält die Entscheidung aber nicht fest.
	epocheBeimStart := b.Rechte.aktuelleEpoche()

	erlaubt, err := repository.HatRolleRecht(ctx, b.DB.Pool, rolle, recht)
	if err != nil {
		return false, err
	}

	b.Rechte.mu.Lock()
	if b.Rechte.epoche == epocheBeimStart {
		// Abgelaufene Einträge fallen beim Schreiben weg, damit der Cache nicht unbegrenzt
		// wächst. Es gibt nur wenige Paare aus Rolle und Recht; ein Durchgang über alle ist
		// billig und braucht keine Goroutine im Hintergrund.
		jetzt := time.Now()
		for k, v := range b.Rechte.eintraege {
			if jetzt.After(v.giltBis) {
				delete(b.Rechte.eintraege, k)
			}
		}
		b.Rechte.schreibe(schluessel, cacheEintrag{
			erlaubt: erlaubt,
			giltBis: jetzt.Add(60 * time.Second),
		})
	}
	b.Rechte.mu.Unlock()

	return erlaubt, nil
}

// PruefeRecht lässt eine Anfrage nur mit dem Recht an next durch. Ein Admin darf immer, sonst
// gilt die Tabelle der Rechte je Rolle, gelesen über den Cache. Fehlt das Recht, antwortet sie
// 403, lässt es sich nicht prüfen, 500.
func (b *Basis) PruefeRecht(w http.ResponseWriter, r *http.Request, next http.Handler, recht string) {
	claims, status, err := b.PruefeSitzung(r)
	if err != nil {
		sendeSitzungsfehler(w, status, err)
		return
	}

	// Ein Admin darf alles.
	if strings.EqualFold(string(claims.Rolle), string(auth.RolleAdmin)) {
		erlaubeZugriff(w, r, next, claims)
		return
	}

	schluessel := string(claims.Rolle) + ":" + recht
	if erlaubt, gefunden := b.Rechte.recht(schluessel); gefunden {
		if !erlaubt {
			log.Printf("Permission denied for role '%s' permission '%s' (FROM CACHE).", claims.Rolle, recht)
			apierrors.AntworteFehlerMitMeldung(w, http.StatusForbidden, "Keine Berechtigung für diese Aktion", nil)
			return
		}
		erlaubeZugriff(w, r, next, claims)
		return
	}

	erlaubt, err := b.ladeUndMerkeRecht(r.Context(), string(claims.Rolle), recht, schluessel)
	if err != nil {
		log.Printf("Permission check DB error for role '%s' permission '%s': %v", claims.Rolle, recht, err)
		apierrors.AntworteFehler(w, http.StatusInternalServerError, errors.New("berechtigung konnte nicht geprüft werden"))
		return
	}

	if !erlaubt {
		log.Printf("Permission denied for role '%s' permission '%s'. allowed: %v", claims.Rolle, recht, erlaubt)
		apierrors.AntworteFehlerMitMeldung(w, http.StatusForbidden, "Keine Berechtigung für diese Aktion", nil)
		return
	}

	erlaubeZugriff(w, r, next, claims)
}

// HatRecht prüft in einer Tür ein zusätzliches Recht, wenn ihre Route schon hinter einem
// schwächeren liegt: für Antworten, die je nach Aufrufer mehr oder weniger zeigen. Die Liste
// der Geräte zeigt die Verfügbarkeit jedem mit view_books, den Namen des Ausleihers nur mit
// view_students (bewertung/sicherheitsbefund-kiosk-suche.md). Ein Fehler zählt als nein: Im
// Zweifel zeigt die Tür weniger, nie mehr.
func (b *Basis) HatRecht(r *http.Request, recht string) bool {
	claims, ok := auth.ClaimsAusKontext(r.Context())
	if !ok {
		return false
	}
	if strings.EqualFold(string(claims.Rolle), string(auth.RolleAdmin)) {
		return true
	}
	schluessel := string(claims.Rolle) + ":" + recht
	if erlaubt, gefunden := b.Rechte.recht(schluessel); gefunden {
		return erlaubt
	}
	erlaubt, err := b.ladeUndMerkeRecht(r.Context(), string(claims.Rolle), recht, schluessel)
	if err != nil {
		return false
	}
	return erlaubt
}
