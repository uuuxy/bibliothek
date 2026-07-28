import { apiFetch } from '../apiFetch.js';

class UIStore {
	activeTab = $state('kiosk');
	selectedBook = $state(/** @type {any} */ (null));
	isSidebarCollapsed = $state(false);
	pendingReservierungen = $state(0);
	isInitialRouteMatched = $state(false);
	/** Welche Statistik-Detailliste die stats_detail-Seite zeigt (deep-linkbar via URL). */
	statsDetailKind = $state(/** @type {'renner' | 'ladenhueter'} */ ('renner'));
	/**
	 * Aus einer anderen Ansicht (Mahnwesen/Abgänger) angefordertes Schülerprofil.
	 * Zentral im Store, bewusst NICHT localStorage: welches Profil DU gerade ansiehst,
	 * ist Session-lokal und nicht PC-übergreifend zu teilen (Multi-PC). StudentDirectory
	 * greift die ID auf, öffnet das Profil und setzt sie zurück.
	 */
	requestedStudentId = $state(/** @type {string | null} */ (null));
	/**
	 * Aus einem System-Alert angeforderter Reiter der Einstellungen. Gleiche Mechanik
	 * wie requestedStudentId: SystemSettings greift den Wert auf und setzt ihn zurück.
	 * Damit kann ein Alert direkt dorthin verweisen, wo sich das Problem beheben lässt,
	 * statt den Nutzer auf „Allgemein" abzusetzen.
	 */
	requestedSettingsTab = $state(/** @type {string | null} */ (null));

	async fetchPendingReservierungen() {
		try {
			const res = await apiFetch('/api/reservierungen/klassensatz/anzahl');
			if (res.ok) {
				const data = await res.json();
				this.pendingReservierungen = data.anzahl ?? 0;
			}
		} catch {
			/* ignore */
		}
	}
}

export const uiStore = new UIStore();
