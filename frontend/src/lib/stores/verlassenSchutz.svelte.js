// Der Verlassen-Schutz einer Maske mit ungespeicherten Eingaben, über die Tür im uiStore
// (verlassenSperre): Der Wechsel der Seite hält an und fragt, Neuladen und Schließen des
// Fensters fragt der Browser. Aufzurufen beim Aufbau des Bauteils, das die Eingaben hält.

import { untrack, getContext, setContext } from 'svelte';
import { uiStore } from './uiStore.svelte.js';
import { verwerfenBestaetigen } from './bestaetigung.svelte.js';

const FORMULARE = Symbol('formulare');

/**
 * Für einen Wechsel, der nicht über uiStore.activeTab läuft (Reiter, Theke, Abmelden): Hält der
 * Bildschirm Ungespeichertes, fragt er vor dem Verwerfen. Wer verwirft, gibt den Wächter frei
 * wie erzwingeWechsel: Abmelden führt danach an die Theke, und das hielte sonst noch an.
 * @param {string} [aktion] der bestätigende Knopf
 * @returns {Promise<boolean>} true: weiter, auch wenn dabei verworfen wird
 */
export async function darfVerwerfen(aktion = 'Verwerfen und weiter') {
	if (!uiStore.verlassenSperre?.()) return true;
	const verwerfen = await verwerfenBestaetigen(aktion);
	if (verwerfen) uiStore.verlassenSperre = null;
	return verwerfen;
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

/**
 * Der Schutz einer Fläche, in der mehrere Formulare stehen (eine Kategorie der Einstellungen):
 * Er fragt, wenn das eigene Formular oder eines darin Ungespeichertes hält. Der Wächter im
 * uiStore hat nur einen Platz; zwei Formulare, die sich je selbst anmeldeten, überschrieben sich.
 * @param {() => boolean} eigenes
 */
export function schuetzeFormulare(eigenes) {
	// Eine Liste statt eines Set: In einer `.svelte.js` ist ein gewöhnliches Set verboten
	// (svelte/prefer-svelte-reactivity), und beobachtet wird die Liste nie.
	/** @type {Array<() => boolean>} */
	const formulare = [];
	setContext(FORMULARE, formulare);
	schuetzeUngespeichertes(() => eigenes() || formulare.some((f) => f()));
}

/**
 * Ein Formular in einer Fläche mit schuetzeFormulare meldet dort, ob es Ungespeichertes hält.
 * Außerhalb einer solchen Fläche tut es nichts.
 * @param {() => boolean} ungespeichert
 */
export function meldeFormular(ungespeichert) {
	/** @type {Array<() => boolean> | undefined} */
	const formulare = getContext(FORMULARE);
	if (!formulare) return;
	$effect(() => {
		formulare.push(ungespeichert);
		return () => {
			const stelle = formulare.indexOf(ungespeichert);
			if (stelle !== -1) formulare.splice(stelle, 1);
		};
	});
}
