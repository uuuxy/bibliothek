import { apiFetch } from '../../apiFetch.js';

/**
 * Die Lesergruppen der Schülerdatei — eigene Datei aus demselben Grund wie
 * schuelerSuche.svelte.js und ausweisdruck.svelte.js: StudentDirectory steht an der
 * Größen-Ratsche (200 Zeilen), und das Nachschlagen der Gruppen hat mit dem Führen der
 * Liste nichts zu tun.
 *
 * Die Gruppen füllen die Auswahl im Anlegen-Dialog. Scheitert der Abruf, bleibt die
 * Auswahl leer: Der Wert lässt sich dann von Hand eintragen, es geht nichts verloren —
 * derselbe geduldete Fall wie die übrigen Vorschlagslisten (fehlerausgang.test.js).
 */
export function erzeugeLesergruppen() {
	/** @type {any[]} */
	let liste = $state.raw([]);

	async function lade() {
		try {
			const res = await apiFetch('/api/readergroups');
			if (res.ok) {
				liste = (await res.json()) || [];
			}
		} catch (err) {
			console.error('Fehler beim Laden der Lesergruppen:', err);
		}
	}

	return {
		get liste() {
			return liste;
		},
		lade
	};
}
