/**
 * Formular eines Geräts (Laptop, Tablet): eine Stelle für das leere Formular, das Formular
 * eines vorhandenen Geräts und die Nutzlast, die an den Server geht.
 */

import { nurGeaendertes } from './utils/geaendert.js';

/** Die Felder, die sich an einem vorhandenen Gerät ändern lassen. Der Barcode klebt. */
const AENDERBAR = ['modellname', 'seriennummer', 'zubehoer', 'zustand_notiz'];

export function leeresGeraetFormular() {
	return { modellname: '', barcode_id: 'G-', seriennummer: '', zubehoer: '', zustand_notiz: '' };
}

/**
 * Die Maske eines vorhandenen Geräts. `geladen` hält den Stand vom Öffnen fest; daran erkennt
 * das Speichern, was geändert wurde.
 * @param {any} geraet
 */
export function geraetFormularAus(geraet) {
	const felder = {
		modellname: geraet.modellname,
		seriennummer: geraet.seriennummer ?? '',
		zubehoer: geraet.zubehoer ?? '',
		zustand_notiz: geraet.zustand_notiz ?? ''
	};
	return { barcode_id: geraet.barcode_id, ...felder, geladen: { ...felder } };
}

/**
 * Was an den Server geht: beim neuen Gerät alle Felder, beim vorhandenen nur die seit dem
 * Öffnen geänderten. Der Server schreibt nur, was der Rumpf nennt; was ein anderer Platz
 * inzwischen an den übrigen Feldern gespeichert hat, bleibt stehen.
 * @param {any} form
 * @param {boolean} vorhanden das Formular bearbeitet ein Gerät, das es schon gibt
 * @returns {Record<string, any>}
 * @throws {Error} wenn einem vorhandenen Gerät der Stand vom Öffnen fehlt
 */
export function geraetNutzlast(form, vorhanden) {
	if (!vorhanden) {
		return {
			modellname: form.modellname,
			barcode_id: form.barcode_id,
			seriennummer: form.seriennummer,
			zubehoer: form.zubehoer,
			zustand_notiz: form.zustand_notiz
		};
	}
	if (!form.geladen) {
		throw new Error(
			'Der Stand vom Öffnen des Geräts fehlt. Nichts gespeichert: bitte das Gerät neu öffnen.'
		);
	}
	return nurGeaendertes(
		form.geladen,
		Object.fromEntries(AENDERBAR.map((name) => [name, form[name]]))
	);
}
