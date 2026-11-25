import { describe, it, expect } from 'vitest';
import { readFileSync } from 'node:fs';
import { relative } from 'node:path';
import { sammleQuelldateien, srcRoot, repoFrontend } from './hygiene-quellen.js';

// Ratsche auf die Ordnung angezeigter Listen: deutsch und Zahlen nach Wert, wie die Kollation
// deutsch am Server (docs/sweeps.md, „Ordnung nach Codepunkt“). Ohne sie stand „Mathe 10“ vor
// „Mathe 5“ und Klasse „10R1“ vor „5F1“. Geprüft wird jeder Aufruf von localeCompare und jedes
// .sort() ohne Vergleich; ein .sort() ohne Vergleich steht nur mit Grund in der Liste.
const SORT_OHNE_VERGLEICH = {
	'src/lib/components/bestellungen/TrefferPruefungSchlagworte.svelte':
		'vergleicht zwei Mengen, zeigt nichts in dieser Ordnung'
};

const dateien = sammleQuelldateien(srcRoot).map((absolut) => ({
	pfad: relative(repoFrontend, absolut),
	text: readFileSync(absolut, 'utf8')
}));

describe('Ordnung angezeigter Listen: deutsch und Zahlen nach Wert', () => {
	it('findet überhaupt Aufrufe', () => {
		const aufrufe = dateien.filter((d) => d.text.includes('localeCompare('));
		expect(aufrufe.length).toBeGreaterThan(5);
	});

	it('jeder localeCompare ordnet mit de und numeric: true', () => {
		// Ein Aufruf samt einer Ebene innerer Klammern, etwa localeCompare(String(b.titel), 'de').
		const aufruf = /localeCompare\((?:[^()]|\([^()]*\))*\)/g;
		const funde = dateien.flatMap((d) =>
			[...d.text.matchAll(aufruf)]
				.map((m) => m[0])
				.filter((a) => !a.includes("'de'") || !a.includes('numeric: true'))
				.map((a) => `${d.pfad}: ${a.replace(/\s+/g, ' ')}`)
		);
		expect(funde).toEqual([]);
	});

	it('ein .sort() ohne Vergleich steht nur mit Grund in der Liste', () => {
		const funde = dateien
			.filter((d) => d.text.includes('.sort()') && !(d.pfad in SORT_OHNE_VERGLEICH))
			.map((d) => d.pfad);
		expect(funde).toEqual([]);
		for (const pfad of Object.keys(SORT_OHNE_VERGLEICH)) {
			expect(
				dateien.find((d) => d.pfad === pfad)?.text,
				`${pfad} steht in der Liste, sortiert aber nicht mehr ohne Vergleich`
			).toContain('.sort()');
		}
	});
});
