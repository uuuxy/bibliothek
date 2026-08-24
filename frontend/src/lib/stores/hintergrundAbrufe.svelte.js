// Hintergrund-Abrufe einer angemeldeten Sitzung: Badges (Reservierungen, offene
// Etiketten) und die Offline-Warteschlange des Kiosks.
//
// Jeder Abruf hängt am Recht SEINER Route — nicht an der Rolle. Bis zum 24.08.2026
// liefen alle drei nur für admin/mitarbeiter: Ein Helfer mit view_orders bekam kein
// Reservierungs-Badge, und ein Helfer ohne das Recht hätte alle 30 s einen 403 gesammelt.
import { hatRecht } from '../menu.js';
import { uiStore } from './uiStore.svelte.js';
import { offlineSync } from './offlineSync.svelte.js';

/**
 * @param {any} user  authStore.currentUser
 * @returns {() => void} räumt die Timer wieder ab
 */
export function starteHintergrundAbrufe(user) {
	/** @type {ReturnType<typeof setInterval>[]} */
	const timer = [];
	if (hatRecht(user, 'perform_actions')) offlineSync.init(); // POST /api/action/batch
	if (hatRecht(user, 'view_orders')) {
		// GET /api/reservierungen/klassensatz/anzahl
		uiStore.fetchPendingReservierungen();
		timer.push(setInterval(() => uiStore.fetchPendingReservierungen(), 30_000));
	}
	if (hatRecht(user, 'edit_books')) {
		// GET /api/exemplare/etiketten-offen/anzahl — speist das Badge an „Druck-Center".
		// Seltener als die Reservierungen: Offene Etiketten ändern sich nur beim
		// Einbuchen und beim Drucken, nicht im Minutentakt.
		uiStore.fetchOffeneEtiketten();
		timer.push(setInterval(() => uiStore.fetchOffeneEtiketten(), 120_000));
	}
	return () => {
		for (const t of timer) clearInterval(t);
	};
}
