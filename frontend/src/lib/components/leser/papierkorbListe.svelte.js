// papierkorbListe.svelte.js — der Zustand des Papierkorbs: laden, wiederherstellen,
// endgültig löschen. Jeder dieser Wege meldet seinen Fehlausgang:
//
//  1. Wiederherstellen: Nach 180 Tagen im Papierkorb anonymisiert der nächtliche DSGVO-Lauf
//     die Zeile, und LeserWiederherstellen weist die Wiederherstellung mit 409 ab, denn ein
//     Pseudonym ist kein Schüler mehr. Ohne Meldung täte ein Klick nichts: eine Tür, die
//     sich nicht öffnet und nicht sagt, warum.
//  2. Laden: Ein gescheiterter Abruf ließe die Liste leer, und die Ansicht schriebe „Der
//     Papierkorb ist leer.", eine falsche Auskunft über gelöschte Schülerdaten.

// Gezeigt wird die Meldung des Servers, nicht unsere Vermutung: Er begründet 409 und 400 im
// Klartext („offene Ausleihen", „bereits anonymisiert").
import { SvelteSet } from 'svelte/reactivity';
import { apiFetch, leseApiFehler } from '../../apiFetch.js';
import { snackbars } from '../../stores/snackbars.svelte.js';

/**
 * Anonymisiert? Dann ist die Zeile kein Schüler mehr, sondern ein Pseudonym: Name,
 * Adresse, Geburtsdatum und Barcode sind getilgt (DSGVO), zurückholen lässt sich das
 * nicht. Das Feld kommt aus GET /api/schueler/deleted mit.
 * @param {{ anonymized_at?: string | null }} s
 */
export function istAnonymisiert(s) {
	return Boolean(s?.anonymized_at);
}

/** @param {() => void} onRestoreSuccess */
export function erzeugePapierkorb(onRestoreSuccess = () => {}) {
	/** @type {any[]} */
	let liste = $state.raw([]);
	let laedt = $state(false);
	let ladefehler = $state('');
	let loeschtGerade = $state(false);
	// Leser, deren Wiederherstellung läuft. Ein zweiter Klick vor der Antwort schickte eine zweite
	// Anfrage, und deren „nicht im Papierkorb" stand nach dem Erfolg der ersten da.
	const stelltHer = new SvelteSet();

	return /** @satisfies {object} */ ({
		get liste() {
			return liste;
		},
		get laedt() {
			return laedt;
		},
		/** Leer heißt leer — ein Ladefehler heißt Ladefehler. */
		get ladefehler() {
			return ladefehler;
		},
		get loeschtGerade() {
			return loeschtGerade;
		},
		/** @param {string} id */
		stelltHer(id) {
			return stelltHer.has(id);
		},

		async laden() {
			laedt = true;
			try {
				const res = await apiFetch('/api/schueler/deleted');
				if (res.ok) {
					liste = (await res.json()) || [];
					ladefehler = '';
				} else {
					liste = [];
					ladefehler = await leseApiFehler(res, 'Der Papierkorb konnte nicht geladen werden.');
				}
			} catch (err) {
				liste = [];
				ladefehler = 'Der Papierkorb konnte nicht geladen werden (Netzwerkfehler).';
				console.error('Fehler beim Laden des Papierkorbs:', err);
			} finally {
				laedt = false;
			}
		},

		/** @param {string} id */
		async wiederherstellen(id) {
			if (stelltHer.has(id)) return;
			stelltHer.add(id);
			try {
				const res = await apiFetch(`/api/schueler/${id}/restore`, { method: 'POST' });
				if (res.ok) {
					await this.laden();
					onRestoreSuccess();
					return;
				}
				snackbars.addToast(await leseApiFehler(res, 'Wiederherstellen fehlgeschlagen.'), 'error');
				// Der Grund kann in der Zwischenzeit entstanden sein (der nächtliche Lauf
				// hat anonymisiert) — nachladen, damit die Liste die Wahrheit zeigt.
				await this.laden();
			} catch (err) {
				snackbars.addToast('Netzwerkfehler bei der Wiederherstellung.', 'error');
				console.error('Fehler bei Wiederherstellung:', err);
			} finally {
				stelltHer.delete(id);
			}
		},

		/** @param {string} id */
		async endgueltigLoeschen(id) {
			loeschtGerade = true;
			try {
				const res = await apiFetch(`/api/schueler/deleted/${id}`, { method: 'DELETE' });
				if (res.ok) {
					snackbars.addToast('Schüler endgültig gelöscht.', 'success');
					await this.laden();
				} else {
					snackbars.addToast(
						await leseApiFehler(res, 'Endgültiges Löschen fehlgeschlagen.'),
						'error'
					);
				}
			} catch (err) {
				snackbars.addToast('Netzwerkfehler beim endgültigen Löschen.', 'error');
				console.error(err);
			} finally {
				loeschtGerade = false;
			}
		}
	});
}
