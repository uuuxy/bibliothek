import { describe, it, expect } from 'vitest';
import { readFileSync } from 'node:fs';
import { srcRoot, sammleQuelldateien, relPfad } from './hygiene-quellen.js';

// Was jeder Einbauort der Schülerakte durchreichen muss — geprüft am Draht, nicht am
// Bauteil (15.09.2026).
//
// - `onMerged` an StudentProfile: Nach dem Zusammenführen zweier Datensätze ist die
//   Kennung des Quell-Datensatzes gelöscht. Die Akte lädt das Ziel selbst nach; der
//   Aufrufer muss SEINEN aktiven Schüler umhängen, sonst bucht die nächste Aktion auf die
//   gelöschte Kennung. Die Schülerdatei tat das seit dem 03.09.2026, die Theke nicht
//   (OFFEN.md 3.2) — und dort landet die Kennung auch in der Offline-Warteschlange.
// - `fehlendeListen` an StudentProfileAusleihen: Ohne das Prop schweigt der Reiter über
//   einen gescheiterten Abruf, und eine leere Karte liest sich als „nichts offen"
//   (OFFEN.md 1.5).

/**
 * Alle Start-Tags einer Komponente in einer .svelte-Datei, jeweils bis zum schließenden `>`.
 * @param {string} quelle @param {string} name
 */
function einbauorte(quelle, name) {
	const re = new RegExp(`<${name}(?=[\\s/>])[^>]*>`, 'g');
	return [...quelle.matchAll(re)].map((m) => m[0]);
}

const PFLICHT = [['StudentProfile', 'onMerged']];

describe('Einbauorte der Schülerakte', () => {
	const dateien = sammleQuelldateien(srcRoot).filter((p) => p.endsWith('.svelte'));

	for (const [komponente, prop] of PFLICHT) {
		it(`reichen ${prop} an ${komponente} durch`, () => {
			/** @type {string[]} */
			const gesehen = [];
			/** @type {string[]} */
			const ohne = [];
			for (const datei of dateien) {
				for (const tag of einbauorte(readFileSync(datei, 'utf8'), komponente)) {
					gesehen.push(relPfad(datei));
					if (!new RegExp(`\\b${prop}=`).test(tag)) ohne.push(relPfad(datei));
				}
			}
			expect(gesehen.length, 'kein Einbauort gefunden — das Gate wäre still grün').toBeGreaterThan(
				0
			);
			expect(ohne, `${komponente} ohne ${prop}`).toEqual([]);
		});
	}
});
