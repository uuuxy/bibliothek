import { apiFetch, extractApiError } from '../../apiFetch.js';

/**
 * Die Klassensatz-Reservierungen als Liste — eigene Datei, weil die Ansicht an der
 * Größen-Ratsche steht (200 Zeilen) und das Laden mit den Dialogen daneben nichts zu tun
 * hat. Dieselbe Auslagerung wie schuelerSuche, lesergruppen und geraeteListe.
 *
 * „Keine offenen Klassensatz-Reservierungen" heißt: niemand wartet. Bis zum 12.09.2026
 * stand derselbe Satz da, wenn der Abruf gescheitert war (`reservierungen = res.ok ?
 * await res.json() : []`) — und wer auf Bücher für eine Unterrichtsstunde mit Termin
 * wartet, wartete weiter (Register, Bestands-Durchgang 10.09.2026).
 */
export function erzeugeKlassensatzListe() {
	/** @type {any[]} */
	let liste = $state([]);
	let laedt = $state(true);
	let ladefehler = $state('');

	async function laden() {
		laedt = true;
		try {
			const res = await apiFetch('/api/reservierungen/klassensatz');
			if (!res.ok) throw new Error(await extractApiError(res));
			liste = (await res.json()) || [];
			ladefehler = '';
		} catch (err) {
			// Leer heißt leer — ein Ladefehler heißt Ladefehler.
			liste = [];
			ladefehler =
				err instanceof Error && err.message
					? err.message
					: 'Die Reservierungen konnten nicht geladen werden.';
		} finally {
			laedt = false;
		}
	}

	return {
		get liste() {
			return liste;
		},
		/** GET liefert die ganze Historie (erledigt + offen); die Arbeitsliste zeigt die offenen. */
		get offene() {
			return liste.filter((/** @type {any} */ r) => !r.erledigt);
		},
		get laedt() {
			return laedt;
		},
		get ladefehler() {
			return ladefehler;
		},
		laden,
		/** @param {string} id */
		entferne(id) {
			liste = liste.filter((/** @type {any} */ r) => r.id !== id);
		},
		/** @param {string} id @param {Record<string, any>} felder */
		aendere(id, felder) {
			liste = liste.map((/** @type {any} */ r) => (r.id === id ? { ...r, ...felder } : r));
		}
	};
}
