import { apiFetch } from '../apiFetch.js';

/**
 * @typedef {{ standort: string, anzahl: number }} StandortZahl
 */

/**
 * Die Standorte, die an Exemplaren im Bestand vorkommen, der häufigste zuerst — die
 * Vorschläge des Dialogs „Standort ändern". Antwortet der Server nicht, bleibt die Liste
 * leer: Das Feld nimmt weiter freien Text an, es fehlen nur die Vorschläge.
 *
 * @returns {Promise<StandortZahl[]>}
 */
export async function ladeStandorte() {
	try {
		const res = await apiFetch('/api/exemplare/standorte');
		if (!res.ok) {
			return [];
		}
		return (await res.json()) ?? [];
	} catch {
		return [];
	}
}

/**
 * Zählt die Standorte der Exemplare eines Titels für den Kopf der Buchakte. Gezählt wird wie
 * in der Titel-Verwaltung (repository.StandorteDerTitel): nur Exemplare im Bestand, die der
 * Server mit `im_bestand` kennzeichnet.
 *
 * @param {{ standort?: string, im_bestand?: boolean }[]} exemplare
 * @returns {StandortZahl[]}
 */
export function standorteAusExemplaren(exemplare) {
	/** @type {Map<string, number>} */
	const zahl = new Map();
	for (const ex of exemplare ?? []) {
		if (!ex.im_bestand || !ex.standort) continue;
		zahl.set(ex.standort, (zahl.get(ex.standort) ?? 0) + 1);
	}
	return [...zahl].map(([standort, anzahl]) => ({ standort, anzahl }));
}

/**
 * Die Standorte eines Titels in einer Zeile, der häufigste zuerst: „Bibliothek, Regal 3B (2)
 * · Lehrerschrank (1)". Getrennt wird mit dem Punkt, weil ein Standort selbst Kommas trägt.
 * Sortiert wird hier, damit Kopf der Akte und Titel-Verwaltung dieselbe Reihenfolge zeigen.
 *
 * @param {StandortZahl[] | null | undefined} standorte
 * @returns {string} leer, wenn kein Exemplar einen Standort trägt
 */
export function standortZeile(standorte) {
	return [...(standorte ?? [])]
		.sort((a, b) => b.anzahl - a.anzahl || a.standort.localeCompare(b.standort, 'de'))
		.map((s) => `${s.standort} (${s.anzahl})`)
		.join(' · ');
}
