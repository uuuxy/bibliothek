// Escape an der Theke schließt die Akte, leert das Scanfeld und beendet Schnellrückgabe und
// Kamera. Wie überall gehört die Taste zuerst dem, der sie schon verarbeitet hat, und einem
// offenen Dialog; in einem Feld heißt sie „Eingabe verwerfen" (escapeRegel.js). Das Scanfeld
// ist das Feld der Theke selbst: Dort schließt Escape die Akte.

import { omniboxStore } from './stores/omnibox.svelte.js';
import { darfVerwerfen } from './stores/verlassenSchutz.svelte.js';
import { escapeIstBelegt } from './components/ui/escapeSchliesst.js';
import { escapeGehoertJemandAnderem } from './escapeRegel.js';

/**
 * @param {KeyboardEvent} e
 * @param {() => void} stoppeKamera schaltet die Kamera ab; den Strom hält ThekenKamera
 */
export async function escapeAnDerTheke(e, stoppeKamera) {
	if (e.key !== 'Escape' || e.defaultPrevented || escapeIstBelegt()) return;
	const imScanfeld = /** @type {HTMLElement | null} */ (e.target)?.id === 'omnibox-input';
	if (!imScanfeld && escapeGehoertJemandAnderem(e)) return;
	// Die Theke nimmt die Taste (escapeSchliesst.test.js, Escape-Hygiene).
	e.preventDefault();
	// Ungespeichertes in der Akte („Stammdaten bearbeiten") geht nur nach Rückfrage verloren.
	if (!(await darfVerwerfen())) return;
	omniboxStore.escapeGedrueckt();
	omniboxStore.eingabe = '';
	omniboxStore.aktiverLeser = null;
	omniboxStore.letzteFremdrueckgabe = null;
	omniboxStore.suche.trefferlisteOffen = false;
	if (omniboxStore.kameraOffen) stoppeKamera();
}
