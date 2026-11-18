import { describe, it, expect } from 'vitest';
import { readFileSync, readdirSync } from 'node:fs';
import { join } from 'node:path';
import { srcRoot, sammleQuelldateien, relPfad, ohneKommentare } from './hygiene-quellen.js';

// Ratsche: Jede Bewegung läuft auf einer Kurve von Material 3 (M3 Motion, Easing and duration,
// Tokens & specs, Zeile CSS). Tailwinds ease-in, ease-out und ease-in-out sind die Kurven von
// Material 2; die CSS-Wörter ease, ease-in und ease-out und eigene cubic-bezier()-Werte sind
// gar keine Kurve von Google. linear bleibt erlaubt: Ein Zeitbalken läuft gleichmäßig.
// Blind für Kurven aus CSS-Variablen und zusammengesetzten Strings; bei Svelte-Übergängen sieht
// sie nur, dass `easing` dasteht, nicht welche Funktion.
const M3_KURVEN = new Set([
	'0.2,0,0,1', // standard
	'0,0,0,1', // standard decelerate
	'0.3,0,1,1', // standard accelerate
	'0.05,0.7,0.1,1', // emphasized decelerate
	'0.3,0,0.8,0.15' // emphasized accelerate
]);
const FREMDE_KURVE = /(?<![-\w])ease(?:-in-out|-in|-out)?(?![-\w])/g;
const BEZIER = /cubic-bezier\(([^)]*)\)/g;
// Svelte-Übergänge laufen ohne Angabe linear (fade) oder auf cubicOut (slide, fly, scale);
// die M3-Kurve kommt als `easing` aus utils/bewegung.js, nie aus svelte/easing.
const SVELTE_UEBERGANG =
	/\b(?:transition|in|out):(?:fade|fly|slide|scale|blur)\b(=\{\{[^}]*\}\})?/g;
const SVELTE_KURVEN = /from ['"]svelte\/easing['"]/g;

/** @param {string} werte */
function normiert(werte) {
	return werte
		.split(',')
		.map((w) => String(Number(w.trim())))
		.join(',');
}

function cssDateien() {
	const ordner = join(srcRoot, 'styles');
	return readdirSync(ordner)
		.filter((d) => d.endsWith('.css'))
		.map((d) => join(ordner, d));
}

describe('Bewegung läuft auf Kurven von Material 3', () => {
	it('nimmt keine Kurve außerhalb der M3-Tokens', () => {
		const treffer = [];
		for (const datei of [...sammleQuelldateien(srcRoot), ...cssDateien()]) {
			const code = ohneKommentare(readFileSync(datei, 'utf8'));
			const fremd = [...code.matchAll(FREMDE_KURVE)].map((m) => m[0]);
			for (const m of code.matchAll(BEZIER)) {
				if (!M3_KURVEN.has(normiert(m[1]))) fremd.push(m[0]);
			}
			for (const m of code.matchAll(SVELTE_UEBERGANG)) {
				if (!m[1]?.includes('easing')) fremd.push(m[0]);
			}
			fremd.push(...[...code.matchAll(SVELTE_KURVEN)].map((m) => m[0]));
			if (fremd.length) treffer.push(`${relPfad(datei)}: ${fremd.join(', ')}`);
		}
		expect(treffer, 'Kurve von Material 3 nehmen (siehe M3_KURVEN)').toEqual([]);
	});
});
