// papierkorbListe.svelte.js — der Zustand des Papierkorbs: laden, wiederherstellen,
// endgültig löschen.
//
// Eigene Datei seit dem Rasterdurchgang am 06.09.2026 (Frage 5, stille Fehler): Die
// Ansicht verschluckte JEDEN Fehlausgang dieser drei Wege, und der Platz für die
// Behandlung hätte die Komponente über die 200-Zeilen-Regel gehoben. Die Ratsche
// lockert man dafür nicht.
//
// Die zwei Funde, die hier zusammenkommen:
//
//  1. `restoreStudent` hatte kein `else`. Nach 180 Tagen im Papierkorb anonymisiert der
//     nächtliche DSGVO-Lauf die Zeile; RestoreStudentHandler weist die Wiederherstellung
//     dann mit 409 ab — richtig so, ein Pseudonym ist kein Schüler mehr. Die Oberfläche
//     zeigte den Knopf trotzdem an jeder Zeile, und ein Klick tat NICHTS: keine Meldung,
//     keine Änderung, kein Nachladen. Eine Tür, die sich nicht öffnet und nicht sagt,
//     warum.
//  2. `loadDeletedStudents` hatte ebenfalls kein `else` — ein gescheiterter Abruf ließ
//     die Liste leer und die Ansicht schrieb „Der Papierkorb ist leer." Das ist keine
//     fehlende Meldung, das ist eine falsche Auskunft über gelöschte Schülerdaten.

import { apiFetch } from '../../apiFetch.js';
import { toastStore } from '../../stores/toastStore.svelte.js';

/**
 * Anonymisiert? Dann ist die Zeile kein Schüler mehr, sondern ein Pseudonym: Name,
 * Adresse, Geburtsdatum und Barcode sind getilgt (DSGVO), zurückholen lässt sich das
 * nicht. Das Feld kommt seit dem 06.09.2026 aus GET /api/schueler/deleted mit.
 * @param {{ anonymized_at?: string | null }} s
 */
export function istAnonymisiert(s) {
	return Boolean(s?.anonymized_at);
}

/**
 * Die Meldung des Servers, nicht unsere Vermutung: Das Backend begründet 409 und 400
 * im Klartext („offene Ausleihen", „bereits anonymisiert") — genau das gehört dem
 * Menschen vor dem Bildschirm gesagt.
 * @param {Response} res
 * @param {string} vorgabe
 */
async function meldungAus(res, vorgabe) {
	try {
		const text = await res.text();
		if (!text) return vorgabe;
		try {
			return JSON.parse(text).error || vorgabe;
		} catch {
			return text;
		}
	} catch {
		return vorgabe;
	}
}

/** @param {() => void} onRestoreSuccess */
export function erzeugePapierkorb(onRestoreSuccess = () => {}) {
	/** @type {any[]} */
	let liste = $state.raw([]);
	let laedt = $state(false);
	let ladefehler = $state('');
	let loeschtGerade = $state(false);

	return {
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

		async laden() {
			laedt = true;
			try {
				const res = await apiFetch('/api/schueler/deleted');
				if (res.ok) {
					liste = (await res.json()) || [];
					ladefehler = '';
				} else {
					liste = [];
					ladefehler = await meldungAus(res, 'Der Papierkorb konnte nicht geladen werden.');
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
			try {
				const res = await apiFetch(`/api/schueler/${id}/restore`, { method: 'POST' });
				if (res.ok) {
					await this.laden();
					onRestoreSuccess();
					return;
				}
				toastStore.addToast(await meldungAus(res, 'Wiederherstellen fehlgeschlagen.'), 'error');
				// Der Grund kann in der Zwischenzeit entstanden sein (der nächtliche Lauf
				// hat anonymisiert) — nachladen, damit die Liste die Wahrheit zeigt.
				await this.laden();
			} catch (err) {
				toastStore.addToast('Netzwerkfehler bei der Wiederherstellung.', 'error');
				console.error('Fehler bei Wiederherstellung:', err);
			}
		},

		/** @param {string} id */
		async endgueltigLoeschen(id) {
			loeschtGerade = true;
			try {
				const res = await apiFetch(`/api/schueler/deleted/${id}`, { method: 'DELETE' });
				if (res.ok) {
					toastStore.addToast('Schüler endgültig gelöscht.', 'success');
					await this.laden();
				} else {
					toastStore.addToast(
						await meldungAus(res, 'Endgültiges Löschen fehlgeschlagen.'),
						'error'
					);
				}
			} catch (err) {
				toastStore.addToast('Netzwerkfehler beim endgültigen Löschen.', 'error');
				console.error(err);
			} finally {
				loeschtGerade = false;
			}
		}
	};
}
