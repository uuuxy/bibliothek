import { apiFetch, extractApiError } from '../apiFetch.js';

/**
 * Die Geräteliste (Laptops, Tablets) — eigene Datei, weil GeraeteVerwaltung an der
 * Größen-Ratsche steht (200 Zeilen) und das Laden mit dem Formular daneben nichts zu tun
 * hat. Dieselbe Auslagerung wie schuelerSuche, lesergruppen und eigeneAnliegen.
 *
 * „Noch keine Geräte erfasst" ist eine Aussage über den Schrank. Bis zum 12.09.2026 stand
 * derselbe Satz da, wenn der Abruf gescheitert war (`const data = res.ok ? await
 * res.json() : null`) — mit der Einladung, Geräte ein zweites Mal anzulegen, die längst
 * im Bestand stehen (Register, Bestands-Durchgang 10.09.2026).
 */
export function erzeugeGeraeteListe() {
	/** @type {any[]} */
	let liste = $state([]);
	let laedt = $state(true);
	let ladefehler = $state('');

	async function lade() {
		laedt = true;
		try {
			const res = await apiFetch('/api/geraete');
			if (!res.ok) throw new Error(await extractApiError(res));
			const daten = await res.json();
			liste = Array.isArray(daten?.data) ? daten.data : [];
			ladefehler = '';
		} catch (err) {
			// Leer heißt leer — ein Ladefehler heißt Ladefehler.
			liste = [];
			ladefehler =
				err instanceof Error && err.message
					? err.message
					: 'Die Geräte konnten nicht geladen werden.';
		} finally {
			laedt = false;
		}
	}

	return {
		get liste() {
			return liste;
		},
		get laedt() {
			return laedt;
		},
		get ladefehler() {
			return ladefehler;
		},
		lade
	};
}
