import { describe, it, expect, vi, afterEach } from 'vitest';
import { readFileSync } from 'node:fs';
import { join } from 'node:path';
import { repoFrontend } from '../hygiene-quellen.js';
import { SCHLUESSEL, darstellung, istDunkel, leseWahl } from './darstellung.svelte.js';

// public/darstellung.js setzt das Schema vor dem ersten Bild, dieses Modul hält es danach
// aktuell. Rechneten beide verschieden, kippte die Seite kurz nach dem Laden.
const vorab = readFileSync(join(repoFrontend, 'public', 'darstellung.js'), 'utf8');

/**
 * Führt public/darstellung.js mit nachgebautem Speicher und System aus.
 * @param {string | null | Error} gespeichert ein Error steht für gesperrten Speicher
 * @param {boolean} systemDunkel
 * @returns {boolean | undefined} ob die Seite dunkel steht; undefined, wenn nichts gesetzt wurde
 */
function vorabDunkel(gespeichert, systemDunkel) {
	/** @type {boolean | undefined} */
	let dunkel;
	const speicher = {
		getItem: (/** @type {string} */ schluessel) => {
			if (gespeichert instanceof Error) throw gespeichert;
			return schluessel === SCHLUESSEL ? gespeichert : null;
		}
	};
	const matchMedia = (/** @type {string} */ abfrage) => ({
		matches: abfrage === '(prefers-color-scheme: dark)' && systemDunkel
	});
	const klassen = {
		toggle: (/** @type {string} */ klasse, /** @type {boolean} */ an) => {
			if (klasse === 'schema-dunkel') dunkel = an;
		}
	};
	new Function('localStorage', 'matchMedia', 'document', vorab)(speicher, matchMedia, {
		documentElement: { classList: klassen }
	});
	return dunkel;
}

describe('Darstellung: Vorab-Skript und Modul rechnen gleich', () => {
	for (const gespeichert of [null, 'hell', 'dunkel', 'system', 'unbekannt']) {
		for (const systemDunkel of [false, true]) {
			it(`gespeichert ${gespeichert}, System ${systemDunkel ? 'dunkel' : 'hell'}`, () => {
				expect(vorabDunkel(gespeichert, systemDunkel)).toBe(
					istDunkel(leseWahl(gespeichert), systemDunkel)
				);
			});
		}
	}

	it('folgt ohne lesbaren Speicher dem System', () => {
		const gesperrt = new Error('SecurityError');
		expect(vorabDunkel(gesperrt, true)).toBe(true);
		expect(vorabDunkel(gesperrt, false)).toBe(false);
	});
});

describe('Darstellung: der Schalter', () => {
	afterEach(() => vi.unstubAllGlobals());

	it('merkt sich an und aus für diesen Browser', () => {
		/** @type {Map<string, string>} */
		const speicher = new Map();
		vi.stubGlobal('localStorage', {
			setItem: (/** @type {string} */ k, /** @type {string} */ v) => speicher.set(k, v)
		});
		darstellung.schalte(true);
		expect(speicher.get(SCHLUESSEL)).toBe('dunkel');
		expect(darstellung.dunkel).toBe(true);
		darstellung.schalte(false);
		expect(speicher.get(SCHLUESSEL)).toBe('hell');
		expect(darstellung.dunkel).toBe(false);
	});

	it('gilt ohne Speicher trotzdem für dieses Fenster', () => {
		vi.stubGlobal('localStorage', {
			setItem: () => {
				throw new Error('QuotaExceededError');
			}
		});
		expect(() => darstellung.schalte(true)).not.toThrow();
		expect(darstellung.dunkel).toBe(true);
	});
});
