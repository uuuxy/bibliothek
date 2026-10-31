// Die Akte eines Lesers: Stammdaten und drei Listen, nebeneinander geholt. Eigene Datei, weil
// useLeserakte an der Größenregel steht.
import { apiFetch } from '../../apiFetch.js';
import { leseTeil } from '../../utils/aktenTeile.js';
import { authStore } from '../../stores/authStore.svelte.js';
import { hatRecht } from '../../menue.js';

/** Die Listen der Akte, in der Reihenfolge ihrer Abrufe. */
const LISTEN = ['Vormerkungen', 'Gebühren', 'Bescheide'];

/**
 * Holt Stammdaten, Vormerkungen, Gebühren und Bescheide eines Lesers. Jeder Teil steht für sich:
 * Fehlt eine Liste, steht die Akte trotzdem da, und die Liste ist mit dem Grund vermerkt
 * (utils/aktenTeile.js). Bei einer Liste heißt 403 nur, sie nicht zu zeigen: Ein Recht kann
 * entzogen sein, seit die Anmeldung läuft. Das Ergebnis kommt erst, wenn alle vier Körper gelesen
 * sind.
 * @param {string} id
 */
export async function ladeLeserakte(id) {
	const [profil, ...listen] = await Promise.allSettled([
		apiFetch(`/api/schueler/${id}`),
		// Die Warteliste hängt am engen Theken-Recht, nicht an view_students.
		hatRecht(authStore.currentUser, 'manage_vormerkungen')
			? apiFetch(`/api/vormerkungen?schueler_id=${id}`)
			: new Response('[]'),
		apiFetch(`/api/schueler/${id}/schadensfaelle`),
		apiFetch(`/api/schueler/${id}/bescheide`)
	]);
	const [akte, ...teile] = await Promise.all([
		leseTeil(profil),
		...listen.map((e) =>
			e.status === 'fulfilled' && e.value.status === 403 ? { wert: [], grund: '' } : leseTeil(e)
		)
	]);
	return {
		akte: akte.wert,
		ladefehler: akte.grund,
		/** @type {any[][]} */
		listen: teile.map((t) => t.wert || []),
		fehlendeListen: LISTEN.flatMap((name, i) =>
			teile[i].grund ? [{ name, grund: teile[i].grund }] : []
		)
	};
}
