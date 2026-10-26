// Ein Sammelversand an die Klassenleitungen: der Mahnlauf und die Kontoauszüge der Abgänger.
// Beide schicken denselben Rumpf an ihre Tür (klassenVersandAnfrage im Server) und melden auf
// dieselbe Weise; die Regeln darunter gibt es deshalb einmal. Den Aufruf schreibt jeder
// Aufrufer selbst, mit Adresse und Frist (api/mail_routen_frist_test.go liest ihn dort).
//
// Ein Lauf zur Zeit: Er dauert je Klasse eine Mail lang, bis zu FRIST_SAMMELVERSAND_MS, und ein
// zweiter Start in der Zeit verschickte jede Mail noch einmal. Hier hält die Sperre den zweiten
// Klick auf, bevor seine Anfrage hinausgeht; einen anderen Platz weist der Server ab (409).

import { melde } from '../../inventur/lib/store.svelte.js';
import { leseApiFehler } from '../apiFetch.js';
import { fehlertext } from '../utils/fehlertext.js';

/** @typedef {{ klassen: string[], override_email: string }} VersandRumpf */

/**
 * @param {(rumpf: VersandRumpf) => Promise<Response>} schicke der Aufruf der Tür
 * @param {(antwort: { sent_count?: number }) => string} ersatzMeldung falls der Server keine
 *   Meldung schickt
 */
export function erzeugeSammelversand(schicke, ersatzMeldung) {
	let laeuft = $state(false);

	/** @param {{ klassen: string[], overrideEmail?: string }} auswahl */
	async function sende(auswahl) {
		if (laeuft) return;
		// Ohne Auswahl gar nicht erst losschicken: Der Server wiese den Rumpf mit 400 ab
		// (klassenVersandAnfrage), hier kommt dieselbe Meldung ohne den Umweg.
		if (!auswahl?.klassen?.length) {
			melde('Keine Klasse ausgewählt.', 'error');
			return;
		}
		laeuft = true;
		try {
			const res = await schicke({
				klassen: auswahl.klassen,
				override_email: auswahl.overrideEmail ?? ''
			});
			if (res.ok) {
				const json = await res.json();
				// Die Meldung des Servers nennt, an wen die Mails gingen. failed_count > 0: Ein Teil
				// des Laufs ist nicht zugestellt, als Warnung, nicht als Erfolg.
				melde(
					json.message ?? ersatzMeldung(json),
					(json.failed_count ?? 0) > 0 ? 'warning' : 'success'
				);
			} else {
				melde(await leseApiFehler(res, 'Versand fehlgeschlagen.'), 'error');
			}
		} catch (e) {
			melde(`Netzwerkfehler beim Versand: ${fehlertext(e)}`, 'error');
		} finally {
			laeuft = false;
		}
	}

	return /** @satisfies {object} */ ({
		get laeuft() {
			return laeuft;
		},
		sende
	});
}
