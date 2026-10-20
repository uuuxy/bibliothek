// klassensatzBuecher.svelte.js — die Bücherliste des Zuweisungs-Dialogs.
//
// Ein gescheiterter Abruf setzt `fehler`: Ein leeres Gitter sähe sonst aus wie „kein Buch im
// Bestand", und der Dialog wäre unbenutzbar, ohne zu sagen, warum.

import { apiFetch } from '../../../../lib/apiFetch.js';

export function erzeugeBuecherListe() {
	/** @type {any[]} */
	let liste = $state([]);
	let fehler = $state(false);

	return /** @satisfies {object} */ ({
		get liste() {
			return liste;
		},
		/** Der Abruf ist gescheitert — die Leere im Gitter ist dann keine Aussage über den Bestand. */
		get fehler() {
			return fehler;
		},
		async laden() {
			try {
				const res = await apiFetch('/api/books');
				if (!res.ok) {
					fehler = true;
					return;
				}
				const json = await res.json();
				if (json.data) liste = json.data;
				fehler = false;
			} catch (e) {
				fehler = true;
				console.error('Fehler beim Laden der Bücher:', e);
			}
		}
	});
}
