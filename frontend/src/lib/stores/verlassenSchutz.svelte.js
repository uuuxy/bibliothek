// Der Verlassen-Schutz einer Maske mit ungespeicherten Eingaben, über die Tür im uiStore
// (verlassenSperre): Der Wechsel der Seite hält an und fragt, Neuladen und Schließen des
// Fensters fragt der Browser. Aufzurufen beim Aufbau des Bauteils, das die Eingaben hält.

import { untrack } from 'svelte';
import { uiStore } from './uiStore.svelte.js';
import { verwerfenBestaetigen } from './bestaetigung.svelte.js';

/**
 * Für einen Wechsel, der nicht über uiStore.activeTab läuft (Reiter, Theke): Hält der Bildschirm
 * Ungespeichertes, fragt er vor dem Verwerfen.
 * @returns {Promise<boolean>} true: weiter, auch wenn dabei verworfen wird
 */
export async function darfVerwerfen() {
	return !uiStore.verlassenSperre?.() || verwerfenBestaetigen('Verwerfen und weiter');
}

/** @param {() => boolean} ungespeichert */
export function schuetzeUngespeichertes(ungespeichert) {
	// Liest keinen Zustand, läuft also einmal nach dem Aufbau und räumt beim Abbau ab.
	$effect(() => {
		uiStore.verlassenSperre = ungespeichert;
		/** @param {BeforeUnloadEvent} e */
		const warnung = (e) => {
			if (ungespeichert()) e.preventDefault();
		};
		window.addEventListener('beforeunload', warnung);
		return () => {
			uiStore.verlassenSperre = null;
			uiStore.blockierterWechsel = null;
			window.removeEventListener('beforeunload', warnung);
		};
	});

	// Ein angehaltener Wechsel fragt. Hat inzwischen ein neuerer angehalten (Zurück-Taste bei
	// offener Frage), gilt dessen Antwort, nicht die der beantworteten älteren Frage.
	$effect(() => {
		const ziel = uiStore.blockierterWechsel;
		if (ziel === null) return;
		untrack(async () => {
			const verwerfen = await verwerfenBestaetigen('Verwerfen und weiter');
			if (uiStore.blockierterWechsel !== ziel) return;
			if (verwerfen) uiStore.erzwingeWechsel();
			else uiStore.bleibe();
		});
	});
}
