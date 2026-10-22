import { apiFetch, FRIST_SAMMELVERSAND_MS } from '../apiFetch.js';
import { erzeugeSammelversand } from './sammelversand.svelte.js';

/**
 * Ersatz, falls der Server keine Meldung schickt; dieselben Worte wie mahnlistenMeldung im Server.
 * @param {number} n
 */
function mahnlistenVersendet(n) {
	return `Mahnlisten von ${n} ${n === 1 ? 'Klasse' : 'Klassen'} versendet`;
}

/**
 * Der Aufruf des Mahnlaufs. Steht außerhalb der Factory, weil er keinen Zustand berührt
 * (SonarQube javascript:S7721).
 * @param {import('./sammelversand.svelte.js').VersandRumpf} rumpf
 */
function schickeMahnlisten(rumpf) {
	return apiFetch('/api/mail/send-bulk-overdue', {
		method: 'POST',
		// Je Klasse eine Mail: Der Lauf dauert länger als die Vorgabe für Anfragen.
		timeoutMs: FRIST_SAMMELVERSAND_MS,
		headers: { 'Content-Type': 'application/json' },
		body: JSON.stringify(rumpf)
	});
}

/**
 * Der Mail-Teil des Mahnwesens: nur der Mahnlauf, je gewählte überfällige Klasse die Mahnliste
 * an die Klassenleitung oder, mit overrideEmail, an genau diese eine Adresse. Ein Einzelversand
 * je Klasse wäre ein zweiter Weg zum selben Versand. Sperre, Meldung und Laufzustand stehen in
 * sammelversand.svelte.js.
 */
export function useMahnwesenMail() {
	return erzeugeSammelversand(schickeMahnlisten, (antwort) =>
		mahnlistenVersendet(antwort.sent_count ?? 0)
	);
}
