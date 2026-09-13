import { apiFetch } from '../../apiFetch.js';

/**
 * Die eigenen Anliegen einer Lehrkraft im Kollegiums-Portal — eigene Datei, damit der
 * Zustand EINMAL da ist: Der Zähler am Reiter, die Startfläche und der Anliegen-Reiter
 * lesen alle daraus (KollegiumPortal.svelte). Drei eigene Abrufe hätten drei Wahrheiten
 * ergeben, und nach dem Absenden zeigte der Zähler noch den alten Stand.
 *
 * Scheitert ein Abruf, bleibt der ALTE Stand stehen. Bis zum 12.09.2026 stand hier
 * `const daten = res.ok ? await res.json() : []` — die Liste wurde dann geleert, und wer
 * gerade einen Wunsch abgeschickt hatte, sah ihn beim Nachladen wieder verschwinden
 * (Register, Bestands-Durchgang 10.09.2026). Ein gescheiterter Abruf weiß nichts über
 * die Anliegen; er darf also auch nichts über sie behaupten.
 */
export function erzeugeEigeneAnliegen() {
	/** @type {any[]} */
	let liste = $state([]);

	async function lade() {
		try {
			const res = await apiFetch('/api/anliegen/eigene');
			if (!res.ok) return;
			const daten = await res.json();
			if (Array.isArray(daten)) liste = daten;
		} catch {
			/* Netz weg: Der alte Stand bleibt stehen, der nächste Aufruf holt ihn nach. */
		}
	}

	return {
		get liste() {
			return liste;
		},
		get offene() {
			return liste.filter((/** @type {any} */ a) => !a.erledigt_am).length;
		},
		lade
	};
}
