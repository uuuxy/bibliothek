// klassensatzBuecher.svelte.js — die Bücherliste des Zuweisungs-Dialogs.
//
// Eigene Datei seit dem Sweep „verschluckte Fehlantwort" (06.09.2026): Der Abruf stand
// mit `if (res.ok) { … }` ohne else im Dialog. Scheiterte er, blieb das Gitter leer —
// nicht zu unterscheiden von „kein Buch im Bestand", und der Dialog war unbenutzbar,
// ohne zu sagen, warum. Der Fehlerzustand hätte den Dialog über seine eingefrorene
// Größe gehoben; die Ratsche lockert man dafür nicht. Jetzt ist er KLEINER als vorher.

import { apiFetch } from '../../../../lib/apiFetch.js';

export function erzeugeBuecherListe() {
	/** @type {any[]} */
	let liste = $state([]);
	let fehler = $state(false);

	return {
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
	};
}
