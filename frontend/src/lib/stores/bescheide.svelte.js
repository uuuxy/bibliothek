// stores/bescheide.svelte.js
// Der Reiter „Schadensersatz": Forderungen ohne Brief, geschriebene Briefe, Übergabe.
//
// Eigener Store und keine Erweiterung des Mahnwesen-Stores: Dort geht es um überfällige
// AUSLEIHEN, hier um GELD. Die beiden teilen nur den Bildschirm.
//
// Die Zahl am Reiter zählt, was bei der Schule liegt (liegtBeiDerSchule): Forderungen
// ohne Brief, laufende und abgelaufene Fristen, Rückgaben nach der Übergabe. Bis zum
// 15.09.2026 zählte sie nur die abgelaufenen Fristen — ein frischer Bescheid stand dann
// mit „0" am Reiter, obwohl eine Zeile darin wartete.

import { apiGet, apiPost } from '../apiFetch.js';
import { toastStore } from './toastStore.svelte.js';
import { liegtBeiDerSchule } from '../bescheidStatus.js';

class BescheideStore {
	/** @type {any[]} Die geschriebenen Briefe. */
	liste = $state([]);
	/** @type {any[]} Kinder mit offenen Forderungen, die noch auf keinem Brief stehen. */
	ausstehend = $state([]);
	laedt = $state(false);
	/** Wurde schon einmal geladen? false = die Zahl am Reiter ist noch unbekannt. */
	geladen = $state(false);

	/** Die Briefe, die der Reiter zeigt: erledigte (bezahlt, storniert) bleiben in der Akte. */
	zeilen = $derived(this.liste.filter((b) => b.status !== 'erledigt'));
	/** Die Zahl am Reiter: was bei der Schule liegt. */
	beiDerSchule = $derived(this.ausstehend.length + this.liste.filter(liegtBeiDerSchule).length);
	/** Rückgabe nach der Übergabe: die Aufsicht ist zu informieren. */
	zuInformieren = $derived(this.liste.filter((b) => b.rueckgabe_nach_uebergabe).length);

	async lade() {
		this.laedt = true;
		try {
			const [liste, ausstehend] = await Promise.all([
				apiGet('/api/bescheide'),
				apiGet('/api/bescheide/ausstehend')
			]);
			this.liste = liste || [];
			this.ausstehend = ausstehend || [];
			this.geladen = true;
		} catch {
			// apiFetch zeigt den Fehler; die Listen bleiben stehen, statt leer zu behaupten,
			// es gäbe nichts.
		} finally {
			this.laedt = false;
		}
	}

	/** @param {string} id */
	async uebergebe(id) {
		try {
			await apiPost(`/api/bescheide/${id}/uebergeben`, {});
			toastStore.addToast('Bescheid als übergeben gebucht.', 'success');
			await this.lade();
		} catch {
			/* apiFetch zeigt den Fehler-Toast */
		}
	}
}

export const bescheideStore = new BescheideStore();
