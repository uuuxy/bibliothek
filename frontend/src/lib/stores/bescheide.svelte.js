// stores/bescheide.svelte.js
// Die Schadensersatz-Bescheide: Arbeitsliste, Zahl der abgelaufenen Fristen, Übergabe.
//
// Eigener Store und keine Erweiterung des Mahnwesen-Stores: Dort geht es um überfällige
// AUSLEIHEN, hier um geschriebene BRIEFE. Die beiden teilen nur den Bildschirm.
//
// Die Zahl am Reiter zählt bewusst NUR die abgelaufenen Fristen — sie ist die Menge
// Arbeit, die wartet. Stünde dort die Gesamtzahl aller Briefe, wäre der Reiter dauerhaft
// zweistellig und niemand sähe mehr, wann etwas zu tun ist.

import { apiGet, apiPost } from '../apiFetch.js';
import { toastStore } from './toastStore.svelte.js';

class BescheideStore {
	/** @type {any[]} */
	liste = $state([]);
	laedt = $state(false);
	/** Wurde schon einmal geladen? false = die Zahl am Reiter ist noch unbekannt. */
	geladen = $state(false);

	/** Bescheide mit abgelaufener Frist — die Arbeit, die wartet. */
	faellig = $derived(this.liste.filter((b) => b.frist_abgelaufen).length);
	/** Rückgabe nach der Übergabe: die Aufsicht ist zu informieren. */
	zuInformieren = $derived(this.liste.filter((b) => b.rueckgabe_nach_uebergabe).length);

	async lade() {
		this.laedt = true;
		try {
			this.liste = (await apiGet('/api/bescheide')) || [];
			this.geladen = true;
		} catch {
			// apiFetch zeigt den Fehler; die Liste bleibt stehen, statt leer zu behaupten,
			// es gäbe keine Bescheide.
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
