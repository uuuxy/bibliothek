/**
 * Exemplare löschen, in der Buchakte und in der Buchmaske: Der Dialog fragt nach dem Grund,
 * vorgewählt „Alter Bestand"; das Exemplar wird ausgesondert und steht mit diesem Grund im
 * Abgangsbuch (DELETE /api/buecher/exemplare/{id}). Ein verliehenes lehnt der Server ab und nennt
 * den Weg über die Leserakte.
 */
import { apiFetch, leseApiFehler } from '../../apiFetch.js';
import { loeschenMitGrund } from '../../stores/bestaetigung.svelte.js';

/**
 * Fragt nach dem Grund für die genannten Exemplare.
 * @param {string[]} barcodes
 * @returns {Promise<string | null>} der Grund, null: abgebrochen
 */
export async function frageGrundFuerExemplare(barcodes) {
	const eines = barcodes.length === 1;
	const wahl = await loeschenMitGrund({
		titel: eines ? `Exemplar ${barcodes[0]} löschen?` : `${barcodes.length} Exemplare löschen?`,
		text: eines
			? 'Es kommt mit diesem Grund ins Abgangsbuch.'
			: 'Sie kommen mit diesem Grund ins Abgangsbuch.'
	});
	return wahl?.grund ?? null;
}

/**
 * Löscht ein Exemplar mit dem Grund.
 * @param {string} id
 * @param {string} grund
 * @throws {Error} mit der Meldung des Servers
 */
export async function loescheExemplarMitGrund(id, grund) {
	const res = await apiFetch(`/api/buecher/exemplare/${id}`, {
		method: 'DELETE',
		credentials: 'include',
		headers: { 'Content-Type': 'application/json' },
		body: JSON.stringify({ grund })
	});
	if (!res.ok) throw new Error(await leseApiFehler(res, 'Fehler beim Löschen des Exemplars.'));
}
