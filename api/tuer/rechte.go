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

var (
	rechteCache   = make(map[string]cacheEintrag)
	rechteCacheMu sync.RWMutex
	// rechteCacheEpoche zählt die Invalidierungen. Eine DB-Entscheidung darf nur dann in
	// den Cache, wenn seit ihrem START keine Invalidierung dazwischenkam: Sonst liest
	// ein Request den ALTEN Stand kurz vor der Rechteänderung, die Invalidierung leert
	// den Cache — und der überholte Leser schreibt den alten Stand danach wieder
	// hinein, wo er bis zu 60 s weiterwirkt (Nebenläufigkeits-Audit 19.08.2026).
	rechteCacheEpoche uint64
)

type cacheEintrag struct {
	erlaubt bool
	giltBis time.Time
}

// LeereRechteCache clears all cached permission entries.
// Call this whenever roles or permissions are changed (e.g. UpdateUser, UpdatePermissions)
// to prevent stale cache entries from granting/denying access incorrectly.
func LeereRechteCache() {
	rechteCacheMu.Lock()
	rechteCache = make(map[string]cacheEintrag)
	rechteCacheEpoche++
	rechteCacheMu.Unlock()
}

// MerkeRechtFuerTest legt die Entscheidung für eine Rolle und ein Recht eine Minute lang in den
// Zwischenspeicher, als hätte die Datenbank sie geliefert: Ein Test ohne Tabelle der Rechte
// stellt so ein, was eine Rolle darf. InvalidatePermissionCache nimmt sie zurück.
func MerkeRechtFuerTest(rolle, recht string, erlaubt bool) {
	rechteCacheMu.Lock()
	rechteCache[rolle+":"+recht] = cacheEintrag{erlaubt: erlaubt, giltBis: time.Now().Add(time.Minute)}
	rechteCacheMu.Unlock()
}

// cacheEpoche liefert den aktuellen Invalidierungs-Stand — VOR dem DB-Lesen
// abfragen, damit der Cache-Schreiber erkennt, ob seine Entscheidung überholt wurde.
func cacheEpoche() uint64 {
	rechteCacheMu.RLock()
	defer rechteCacheMu.RUnlock()
	return rechteCacheEpoche
}

// rechtImCache liefert eine noch gültige, gecachte Entscheidung.
func rechtImCache(schluessel string) (erlaubt bool, gefunden bool) {
	rechteCacheMu.RLock()
	eintrag, ok := rechteCache[schluessel]
	rechteCacheMu.RUnlock()
	if ok && time.Now().Before(eintrag.giltBis) {
		return eintrag.erlaubt, true
	}
	return false, false
}

// ladeUndMerkeRecht prüft das Recht in der DB und cacht ausschließlich
// stabile Entscheidungen (gewährt / explizit nicht gewährt). Transiente DB-Fehler
// (Timeout, Pool erschöpft, Verbindungsabriss) dürfen NICHT gecacht werden — sonst
// zementierte ein kurzer Aussetzer 60 s lang ein 403 für berechtigte Nutzer — und
// sind ein Server- (500), kein Berechtigungsproblem. pgx.ErrNoRows dagegen heißt
// "Recht nicht gewährt" und ist stabil cachebar.
func (b *Basis) ladeUndMerkeRecht(ctx context.Context, rolle, recht, schluessel string) (bool, error) {
	// Epoche VOR dem DB-Lesen festhalten: Nur wenn bis zum Cache-Schreiben keine
	// Invalidierung dazwischenkam, ist die Entscheidung noch aktuell. Kam eine, sah
	// dieses Lesen womöglich den Stand VOR der Rechteänderung — der Request selbst
	// darf damit noch antworten, aber der Cache darf ihn nicht festhalten.
	epocheBeimStart := cacheEpoche()

	erlaubt, err := repository.HatRolleRecht(ctx, b.DB.Pool, rolle, recht)
	if err != nil {
		return false, err
	}

	rechteCacheMu.Lock()
	if rechteCacheEpoche == epocheBeimStart {
		// Abgelaufene Einträge beim Miss-Write opportunistisch entfernen, damit der Cache
		// nicht unbegrenzt wächst. Der Keyspace (Rolle × Permission) ist klein, daher ist
		// ein vollständiger Sweep hier günstig und kommt ohne Hintergrund-Goroutine aus.
		jetzt := time.Now()
		for k, v := range rechteCache {
			if jetzt.After(v.giltBis) {
				delete(rechteCache, k)
			}
		}
		rechteCache[schluessel] = cacheEintrag{
			erlaubt: erlaubt,
			giltBis: jetzt.Add(60 * time.Second),
		}
	}
	rechteCacheMu.Unlock()

	return erlaubt, nil
}

// PruefeRecht setzt die RBAC-Prüfung für einen Request um: Admins dürfen immer,
// sonst wird die konfigurierte Rolle→Permission-Zuordnung (mit Cache) geprüft. Bei
// Erlaubnis wird an next delegiert, andernfalls 403 bzw. 500 gesendet.
func (b *Basis) PruefeRecht(w http.ResponseWriter, r *http.Request, next http.Handler, recht string) {
	claims, status, err := b.PruefeSitzung(r)
	if err != nil {
		sendeSitzungsfehler(w, status, err)
		return
	}

	// Admin role always has all permissions allowed
	if strings.EqualFold(string(claims.Rolle), string(auth.RoleAdmin)) {
		erlaubeZugriff(w, r, next, claims)
		return
	}

	schluessel := string(claims.Rolle) + ":" + recht
	if erlaubt, gefunden := rechtImCache(schluessel); gefunden {
		if !erlaubt {
			log.Printf("Permission denied for role '%s' permission '%s' (FROM CACHE).", claims.Rolle, recht)
			apierrors.SendHTTPError(w, http.StatusForbidden, errors.New("keine Berechtigung für diese Aktion"))
			return
		}
		erlaubeZugriff(w, r, next, claims)
		return
	}

	erlaubt, err := b.ladeUndMerkeRecht(r.Context(), string(claims.Rolle), recht, schluessel)
	if err != nil {
		log.Printf("Permission check DB error for role '%s' permission '%s': %v", claims.Rolle, recht, err)
		apierrors.SendHTTPError(w, http.StatusInternalServerError, errors.New("berechtigung konnte nicht geprüft werden"))
		return
	}

	if !erlaubt {
		log.Printf("Permission denied for role '%s' permission '%s'. allowed: %v", claims.Rolle, recht, erlaubt)
		apierrors.SendHTTPError(w, http.StatusForbidden, errors.New("keine Berechtigung für diese Aktion"))
		return
	}

	erlaubeZugriff(w, r, next, claims)
}

// HatRecht prüft ein Zusatzrecht INNERHALB eines Handlers, dessen Route
// bereits hinter einem schwächeren Recht liegt — für Antworten, die je nach
// Aufrufer mehr oder weniger zeigen (z. B. Geräteliste: Verfügbarkeit für alle
// mit view_books, der Name des Ausleihers nur mit view_students; siehe
// bewertung/sicherheitsbefund-kiosk-suche.md). Fehler werten als "nein":
// Im Zweifel wird weniger gezeigt, nie mehr.
func (b *Basis) HatRecht(r *http.Request, recht string) bool {
	claims, ok := auth.GetClaims(r.Context())
	if !ok {
		return false
	}
	if strings.EqualFold(string(claims.Rolle), string(auth.RoleAdmin)) {
		return true
	}
	schluessel := string(claims.Rolle) + ":" + recht
	if erlaubt, gefunden := rechtImCache(schluessel); gefunden {
		return erlaubt
	}
	erlaubt, err := b.ladeUndMerkeRecht(r.Context(), string(claims.Rolle), recht, schluessel)
	if err != nil {
		return false
	}
	return erlaubt
}
