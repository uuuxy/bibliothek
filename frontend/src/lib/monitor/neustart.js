// monitor/neustart.js — wann sich der Flur-Monitor selbst neu lädt.
//
// Eigene Datei ohne Runes: Die Rechnung mit Date ist kein reaktiver Zustand, und in einer
// .svelte.js-Datei verlangt eslint (svelte/prefer-svelte-reactivity) dort SvelteDate — das
// wäre hier falsch, nicht nur unnötig.

/** Uhrzeit (Ortszeit) des täglichen Neustarts — nachts stört er niemanden. */
export const NEUSTART_STUNDE = 3;

/**
 * Millisekunden bis zur nächsten vollen NEUSTART_STUNDE in Ortszeit — genau um drei ist es
 * der morgige Termin. Ortszeit, nicht UTC: Die Uhr an der Wand soll drei zeigen, auch
 * wenn die Sommerzeit dazwischen wechselt.
 * @param {Date} [jetzt]
 */
export function msBisNeustart(jetzt = new Date()) {
	const termin = new Date(
		jetzt.getFullYear(),
		jetzt.getMonth(),
		jetzt.getDate(),
		NEUSTART_STUNDE,
		0,
		0,
		0
	);
	if (termin <= jetzt) termin.setDate(termin.getDate() + 1);
	return termin.getTime() - jetzt.getTime();
}
