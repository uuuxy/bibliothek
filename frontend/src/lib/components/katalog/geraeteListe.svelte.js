import { apiFetch, leseApiFehler } from '../../apiFetch.js';

/**
 * Die Geräteliste (Laptops, Tablets) — eigene Datei, weil GeraeteVerwaltung an der
 * Größen-Ratsche steht (200 Zeilen) und das Laden mit dem Formular daneben nichts zu tun
 * hat. Dieselbe Auslagerung wie schuelerSuche, klassenVorschlaege und eigeneAnliegen.
 *
 * „Noch keine Geräte erfasst" ist eine Aussage über den Schrank. Nach einem gescheiterten
 * Abruf stünde derselbe Satz da, mit der Einladung, Geräte ein zweites Mal anzulegen, die
 * längst im Bestand stehen; deshalb hat die Liste einen eigenen Fehlerzustand.
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
			if (!res.ok) throw new Error(await leseApiFehler(res));
			const daten = await res.json();
			liste = Array.isArray(daten) ? daten : [];
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

	return /** @satisfies {object} */ ({
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
	});
}
