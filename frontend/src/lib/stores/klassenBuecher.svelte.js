import { apiFetch, leseApiFehler } from '../apiFetch.js';
import { fehlertext } from '../utils/fehlertext.js';

/**
 * Die Klassen mit ihren Büchern für „Aus Klasse laden" im Druck-Center. Wie die Klassenauswahl
 * der Leserakte (klassenVorschlaege, dort nur die Namen aus /api/klassen) merkt sie sich den
 * Grund, wenn der Abruf scheitert: Ohne ihn stünde eine leere Auswahl da, als hätte die Schule
 * keine Klassen.
 */
export function erzeugeKlassenBuecher() {
	/** @type {any[]} */
	let liste = $state.raw([]);
	/** Der Grund, wenn der Abruf scheiterte; leer heißt geladen. */
	let ladefehler = $state('');

	async function lade() {
		try {
			const res = await apiFetch('/api/class-books');
			if (!res.ok) throw new Error(await leseApiFehler(res));
			liste = await res.json();
			ladefehler = '';
		} catch (err) {
			// Eine Liste von vorher bleibt stehen: Sie ist älter, aber nicht falsch.
			ladefehler = fehlertext(err);
			console.error('Fehler beim Laden der Klassengruppen:', err);
		}
	}

	return /** @satisfies {object} */ ({
		get liste() {
			return liste;
		},
		get ladefehler() {
			return ladefehler;
		},
		lade
	});
}
