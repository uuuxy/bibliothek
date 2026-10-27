// stores/bescheide.svelte.js
// Der Reiter „Schadensersatz": Forderungen ohne Brief, geschriebene Briefe, Übergabe.
//
// Eigener Store und keine Erweiterung des Mahnwesen-Stores: Dort geht es um überfällige
// Ausleihen, hier um Geld. Die beiden teilen nur den Bildschirm.
//
// Die Zahl am Reiter zählt, was bei der Schule liegt (liegtBeiDerSchule): Forderungen ohne
// Brief, laufende und abgelaufene Fristen, Rückgaben nach der Übergabe. Mit den
// abgelaufenen Fristen allein stünde ein frischer Bescheid mit „0" am Reiter, obwohl eine
// Zeile darin wartet.

import { apiGet, apiPost } from '../apiFetch.js';
import { snackbars } from './snackbars.svelte.js';
import { fehlertext } from '../utils/fehlertext.js';
import { liegtBeiDerSchule } from '../components/mahnwesen/bescheidStatus.js';

class BescheideStore {
	/** @type {any[]} Die geschriebenen Briefe. */
	liste = $state([]);
	/** @type {any[]} Kinder mit offenen Forderungen, die noch auf keinem Brief stehen. */
	ausstehend = $state([]);
	laedt = $state(false);
	/** Wurde schon einmal geladen? false = die Zahl am Reiter ist noch unbekannt. */
	geladen = $state(false);
	/** Der Grund, wenn der letzte Abruf scheiterte; die Tabelle zeigt ihn statt einer leeren. */
	ladefehler = $state('');
	/** Der Bescheid, dessen Übergabe gerade läuft; leer = keiner. */
	uebergibt = $state('');

	/** Die Briefe, die der Reiter zeigt: erledigte (bezahlt, storniert) bleiben in der Akte. */
	zeilen = $derived(this.liste.filter((b) => b.status !== 'erledigt'));
	/** Die Zahl am Reiter: was bei der Schule liegt. */
	beiDerSchule = $derived(this.ausstehend.length + this.liste.filter(liegtBeiDerSchule).length);
	/** Rückgabe nach der Übergabe: die Aufsicht ist zu informieren. */
	zuInformieren = $derived(this.liste.filter((b) => b.rueckgabe_nach_uebergabe).length);

	async lade() {
		this.laedt = true;
		this.ladefehler = '';
		try {
			const [liste, ausstehend] = await Promise.all([
				apiGet('/api/bescheide'),
				apiGet('/api/bescheide/ausstehend')
			]);
			this.liste = liste || [];
			this.ausstehend = ausstehend || [];
			this.geladen = true;
		} catch (e) {
			// Die Listen bleiben stehen, statt leer zu behaupten, es gäbe nichts.
			this.ladefehler = fehlertext(e);
		} finally {
			this.laedt = false;
		}
	}

	/** @param {string} id */
	async uebergebe(id) {
		// Ein zweiter Klick vor der Antwort schickte die Übergabe noch einmal: Der Server weist sie
		// ab, und neben „gebucht" stünde „Übergabe nicht möglich".
		if (this.uebergibt) return;
		this.uebergibt = id;
		try {
			await apiPost(`/api/bescheide/${id}/uebergeben`, {});
			snackbars.addToast('Bescheid als übergeben gebucht.', 'success');
			await this.lade();
		} catch {
			/* apiPost meldet jeden Fehler selbst */
		} finally {
			this.uebergibt = '';
		}
	}
}

export const bescheideStore = new BescheideStore();
