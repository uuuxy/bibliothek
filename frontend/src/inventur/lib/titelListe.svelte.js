import { holeBuecherListe } from './admin_api.js';
import { appState } from './store.svelte.js';
import { keineAenderungen, merkeAenderungen, mitAenderungen } from './titelListeAenderungen.js';
import { fehlertext } from '../../lib/utils/fehlertext.js';

/**
 * Die Titelliste der Titel-Verwaltung: laden und die Änderungen der Maske übernehmen.
 *
 * Nur die jüngste Abfrage gilt: Wechselt die Sicht, während die vorige Liste noch lädt, stünde
 * deren späte Antwort sonst unter der neuen Sicht.
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
	/** Die Sicht, aus der die Liste stammt; leer, bis die erste Antwort da ist. */
	let sichtDerListe = $state('');
	let lauf = 0;
	let unterwegs = keineAenderungen();

	async function lade() {
		const meiner = ++lauf;
		const sicht = appState.bestandsAnsicht;
		wirdGeladen = true;
		ladefehler = '';
		unterwegs = keineAenderungen();
		try {
			const geladene = await holeBuecherListe();
			if (meiner !== lauf) return;
			buecher = mitAenderungen(geladene, unterwegs);
			sichtDerListe = sicht;
			appState.adminAuthenticated = true;
		} catch (e) {
			// Ein gescheitertes Laden ist keine Abmeldung (die meldet apiFetch bei 401): Die Suche
			// lädt weiter, und die Seite nennt den Grund statt „Keine Titel gefunden".
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
		/**
		 * Die Liste stammt aus einer anderen Sicht als der gewählten oder ist noch gar nicht da.
		 * Dann steht der Ladekreis statt ihrer: Unter der neuen Sicht sähe die alte Liste aus
		 * wie deren Ergebnis.
		 */
		get sichtLaedt() {
			return sichtDerListe !== appState.bestandsAnsicht;
		},
		/** Der Grund, aus dem der letzte Abruf scheiterte; leer, solange er gelingt. */
		get ladefehler() {
			return ladefehler;
		},
		lade
	});
}

/**
 * Lädt die Liste nach jedem Wechsel der Sicht sofort neu. Die Suche fragt den Server nicht, sie
 * filtert die geladene Liste. Den Aufbau lädt die Seite selbst; der erste Lauf des Effekts ist
 * deshalb kein Wechsel, mit ihm lüde die ganze Liste zweimal.
 * @param {() => unknown} lade
 */
export function ladeNachWechsel(lade) {
	let ersterLauf = true;
	$effect(() => {
		const sicht = appState.bestandsAnsicht;
		if (ersterLauf) {
			ersterLauf = false;
			return;
		}
		const warten = setTimeout(() => {
			if (appState.adminAuthenticated && sicht) lade();
		}, 0);
		return () => clearTimeout(warten);
	});
}
