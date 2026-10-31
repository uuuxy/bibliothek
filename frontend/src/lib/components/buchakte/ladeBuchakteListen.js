// Die Listen der Akte eines Titels, nebeneinander geholt. Eigene Datei, weil useBuchakte an der
// Größenregel steht.
import { apiFetch } from '../../apiFetch.js';
import { leseTeil } from '../../utils/aktenTeile.js';

/**
 * Holt die Listen, die das Konto sehen darf; nach den anderen fragt es nicht. Jede Liste steht für
 * sich: Scheitert eine, kommen die übrigen, und die eine ist mit dem Grund vermerkt
 * (utils/aktenTeile.js). Das Ergebnis kommt erst, wenn alle Körper gelesen sind.
 * @param {string} id
 * @param {{ darfPersonen: boolean, darfVormerken: boolean }} rechte
 * @returns {Promise<{ werte: Record<string, any[]>, fehlend: { name: string, grund: string }[] }>}
 */
export async function ladeBuchakteListen(id, { darfPersonen, darfVormerken }) {
	/** @type {[string, boolean, string][]} */
	const alle = [
		['Ausleiher', darfPersonen, `/api/buecher/titel/${id}/ausleiher`],
		['Exemplare', true, `/api/buecher/titel/${id}/exemplare`],
		['Historie', darfPersonen, `/api/buecher/titel/${id}/historie`],
		['Vormerkungen', darfVormerken, `/api/vormerkungen?titel_id=${id}`]
	];
	const listen = alle.filter(([, darf]) => darf);
	const antworten = await Promise.allSettled(
		listen.map(([, , adresse]) => apiFetch(adresse, { credentials: 'include' }))
	);
	const daten = await Promise.all(antworten.map(leseTeil));
	return {
		werte: Object.fromEntries(listen.map(([name], i) => [name, daten[i].wert ?? []])),
		fehlend: listen.flatMap(([name], i) =>
			daten[i].grund ? [{ name, grund: daten[i].grund }] : []
		)
	};
}
