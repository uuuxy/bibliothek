import { describe, it, expect } from 'vitest';
import { readFileSync } from 'node:fs';
import { join } from 'node:path';
import { srcRoot, ohneKommentare } from './hygiene-quellen.js';

// Ratsche: Jedes Fenster, das die Theke selbst einhängt, steht in rueckfrageOffen
// (stores/omnibox.svelte.js). Daran hängen drei Zusagen: Ein Scan bei offenem Fenster drückt
// keinen Knopf darin (scanOhneFokus.js), seine Zeichen fallen nicht ins Scanfeld dahinter, und
// die Scan-Reihe hält an. Ein Fenster, das dort fehlt, schlösse der nächste Scan.
//
// Blind: für ein Fenster, das tiefer hängt (in der Akte) oder nicht aus components/Omnibox…
// kommt, und für eines, das die Theke anders ein- und ausblendet als mit
// {#if omniboxStore.<Zustand>} in der eigenen Datei. Ob der Scan wirklich abgefangen wird,
// prüft e2e/theke-scan-reihe.spec.js.

/** @param {string} rel Pfad unter src/lib */
const lies = (rel) => ohneKommentare(readFileSync(join(srcRoot, 'lib', rel), 'utf8'));

describe('Fenster der Theke stehen in rueckfrageOffen', () => {
	const zustaende =
		/const rueckfrageOffen = \(\) => !!\(([^)]*)\)/
			.exec(lies('stores/omnibox.svelte.js'))?.[1]
			.split('||')
			.map((z) => z.trim()) ?? [];
	const fenster = [...lies('Omnibox.svelte').matchAll(/'\.\/(components\/Omnibox\w+\.svelte)'/g)]
		.map((m) => m[1])
		.filter((rel) => /aria-modal="true"|<Modal\b/.test(lies(rel)));

	it('findet Fenster und Zustände — sonst wäre das Gate wertlos grün', () => {
		expect(zustaende.length).toBeGreaterThanOrEqual(3);
		expect(fenster.length).toBeGreaterThanOrEqual(3);
	});

	it('jedes Fenster hängt an einem Zustand, den rueckfrageOffen kennt', () => {
		const ohne = fenster.filter(
			(rel) => !zustaende.some((z) => lies(rel).includes(`{#if omniboxStore.${z}}`))
		);
		expect(ohne, 'Fenster der Theke, das in rueckfrageOffen fehlt').toEqual([]);
	});
});
