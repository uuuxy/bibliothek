import { holeBuecherListe } from './admin_api.js';
import { appState } from './store.svelte.js';
import { keineAenderungen, merkeAenderungen, mitAenderungen } from './titelListeAenderungen.js';
import { fehlertext } from '../../lib/utils/fehlertext.js';

/**
 * Die Titelliste der Titel-Verwaltung: laden und die Änderungen der Maske übernehmen.
 *
 * Nur die jüngste Abfrage gilt: Die ganze Liste lädt länger als ein Suchergebnis und stünde
 * sonst, wenn sie danach ankommt, unter dem Suchwort.
 *
 * Was die Maske ändert, solange ein Abruf läuft (gespeichert, gelöscht, neues Cover), kennt
 * dessen Antwort nicht. Sie ersetzte die Liste samt der Änderung: Der eben angelegte Titel
 * verschwand, bis die Seite neu lud. Die Änderungen werden deshalb gemerkt und auf die
 * Antwort gelegt.
 */
export function erstelleTitelListe() {
	/** @type {any[]} */
	let buecher = $state.raw([]);
	let wirdGeladen = $state(false);
	let ladefehler = $state('');
	let lauf = 0;
	let unterwegs = keineAenderungen();

	async function lade() {
		const meiner = ++lauf;
		wirdGeladen = true;
		ladefehler = '';
		unterwegs = keineAenderungen();
		try {
			const geladene = await holeBuecherListe();
			if (meiner !== lauf) return;
			buecher = mitAenderungen(geladene, unterwegs);
			appState.adminAuthenticated = true;
		} catch (e) {
			// Ein gescheitertes Laden ist keine Abmeldung (die meldet apiFetch bei 401): Die Suche
			// lädt weiter, und die Seite nennt den Grund statt „Keine Bücher gefunden".
			if (meiner === lauf) ladefehler = fehlertext(e);
		} finally {
			if (meiner === lauf) wirdGeladen = false;
		}
	}

	return /** @satisfies {object} */ ({
		get buecher() {
			return buecher;
		},
		/** Die Maske oder eine Massenaktion hat die Liste geändert. */
		set buecher(neu) {
			if (wirdGeladen) merkeAenderungen(buecher, neu, unterwegs);
			buecher = neu;
		},
		get wirdGeladen() {
			return wirdGeladen;
		},
		/** Der Grund, aus dem der letzte Abruf scheiterte; leer, solange er gelingt. */
		get ladefehler() {
			return ladefehler;
		},
		lade
	});
}
