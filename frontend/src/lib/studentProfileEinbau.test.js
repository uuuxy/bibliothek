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
 * Alle Start-Tags einer Komponente in einer .svelte-Datei, jeweils bis zum schließenden `>`
 * AUSSERHALB geschweifter Klammern — ein `=>` in einem Attributausdruck beendet das Tag nicht.
 * @param {string} quelle @param {string} name
 */
function einbauorte(quelle, name) {
	/** @type {string[]} */
	const tags = [];
	const start = new RegExp(`<${name}(?=[\\s/>])`, 'g');
	for (const m of quelle.matchAll(start)) {
		let tiefe = 0;
		for (let i = m.index; i < quelle.length; i++) {
			const z = quelle[i];
			if (z === '{') tiefe++;
			else if (z === '}') tiefe--;
			else if (z === '>' && tiefe === 0) {
				tags.push(quelle.slice(m.index, i + 1));
				break;
			}
		}
	}
	return tags;
}

const PFLICHT = [
	['StudentProfile', 'onMerged'],
	['StudentProfileAusleihen', 'fehlendeListen']
];

describe('Einbauorte der Schülerakte', () => {
	const dateien = sammleQuelldateien(srcRoot).filter((p) => p.endsWith('.svelte'));

	// „Buch zurückgeben" an der Theke geht über gibZurueck, das die Absicht kennt — nicht
	// über queryVal + submitAction, das offline aus dem geladenen Schüler eine Ausleihe
	// machte (OFFEN.md 2.2, Commit 3, 15.09.2026).
	it('die Theke gibt aus der Akte über gibZurueck zurück', () => {
		const omnibox = dateien.find((p) => p.endsWith('/lib/Omnibox.svelte'));
		expect(omnibox, 'Omnibox.svelte nicht gefunden').toBeTruthy();
		const tags = einbauorte(
			readFileSync(/** @type {string} */ (omnibox), 'utf8'),
			'StudentProfile'
		);
		expect(tags).toHaveLength(1);
		expect(tags[0]).toMatch(/onReturnClick=\{[^}]*gibZurueck\(/);
	});

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
