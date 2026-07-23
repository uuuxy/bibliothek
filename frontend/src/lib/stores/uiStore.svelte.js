import { apiFetch } from '../apiFetch.js';

class UIStore {
	activeTab = $state('kiosk');
	selectedBook = $state(/** @type {any} */ (null));
	isSidebarCollapsed = $state(false);
	pendingReservierungen = $state(0);
	isInitialRouteMatched = $state(false);
	/** Welche Statistik-Detailliste die stats_detail-Seite zeigt (deep-linkbar via URL). */
	statsDetailKind = $state(/** @type {'renner' | 'ladenhueter'} */ ('renner'));

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
