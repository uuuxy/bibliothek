// Die Rechnung hinter titelListe.svelte.js: was sich an der Titelliste geändert hat, solange
// ein Abruf lief, und wie es auf dessen Antwort gelegt wird. Ohne Zustand, deshalb hier.

/** @typedef {{ gesetzt: Map<string, any>, weg: Set<string> }} Aenderungen */

/** @returns {Aenderungen} */
export function keineAenderungen() {
	return { gesetzt: new Map(), weg: new Set() };
}

/**
 * Hält fest, worin sich die Liste nach einer Änderung von der davor unterscheidet: neue und
 * ersetzte Zeilen, entfernte Zeilen.
 * @param {any[]} vorher
 * @param {any[]} nachher
 * @param {Aenderungen} aenderungen
 */
export function merkeAenderungen(vorher, nachher, aenderungen) {
	const alt = new Map(vorher.map((b) => [b.id, b]));
	const bleibt = new Set(nachher.map((b) => b.id));
	for (const b of nachher) {
		if (alt.get(b.id) === b) continue;
		aenderungen.gesetzt.set(b.id, b);
		aenderungen.weg.delete(b.id);
	}
	for (const id of alt.keys()) {
		if (bleibt.has(id)) continue;
		aenderungen.weg.add(id);
		aenderungen.gesetzt.delete(id);
	}
}

/**
 * Die geladene Liste mit den Änderungen, die während des Ladens dazukamen: Entferntes fehlt,
 * Ersetztes steht an seinem Platz, Neues oben, das zuletzt Gespeicherte zuerst.
 * @param {any[]} geladene
 * @param {Aenderungen} aenderungen
 * @returns {any[]}
 */
export function mitAenderungen(geladene, aenderungen) {
	if (aenderungen.gesetzt.size === 0 && aenderungen.weg.size === 0) return geladene;
	const vorhanden = new Set(geladene.map((b) => b.id));
	const neue = [...aenderungen.gesetzt.values()].filter((b) => !vorhanden.has(b.id)).reverse();
	const uebrige = geladene
		.filter((b) => !aenderungen.weg.has(b.id))
		.map((b) => aenderungen.gesetzt.get(b.id) ?? b);
	return [...neue, ...uebrige];
}
